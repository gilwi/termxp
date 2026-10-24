package main

import (
	"errors"
	"sync"
	"testing"
	"time"
)

type blockedLogWriter struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (w *blockedLogWriter) Write(data []byte) (int, error) {
	w.once.Do(func() { close(w.entered) })
	<-w.release
	return len(data), nil
}
func (w *blockedLogWriter) Close() error { return nil } // Model a filesystem whose write cannot be interrupted.
func TestLoggingCannotBlockSessionCancellation(t *testing.T) {
	ts := NewTerminalService()
	t.Cleanup(ts.cleanupAllSessions)
	if err := ts.StartSession("disk", 80, 24); err != nil {
		t.Fatal(err)
	}
	s := ts.session("disk")
	writer := &blockedLogWriter{entered: make(chan struct{}), release: make(chan struct{})}
	defer close(writer.release)
	logger := newSessionLog(writer, nil)
	s.mu.Lock()
	s.logger = logger
	s.mu.Unlock()
	logger.enqueue([]byte("blocked"))
	awaitDone(t, writer.entered)
	stopped := make(chan struct{})
	go func() { ts.cleanupAllSessions(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("logging blocked terminal shutdown")
	}
	awaitDone(t, s.done)
}
func TestLoggingQueueOverflowStopsLoggingInsteadOfTerminal(t *testing.T) {
	writer := &blockedLogWriter{entered: make(chan struct{}), release: make(chan struct{})}
	defer close(writer.release)
	failures := make(chan error, 1)
	logger := newSessionLog(writer, func(err error) { failures <- err })
	logger.enqueue(make([]byte, outputChunkSize))
	awaitDone(t, writer.entered)
	for i := 0; i < 33; i++ {
		logger.enqueue(make([]byte, outputChunkSize))
	}
	select {
	case err := <-failures:
		if err == nil {
			t.Fatal("missing overflow error")
		}
	case <-time.After(time.Second):
		t.Fatal("overflow did not stop logging")
	}
	logger.mu.Lock()
	active := logger.accepting
	logger.mu.Unlock()
	if active {
		t.Fatal("overflowed logger remains active")
	}
}

type failedLogWriter struct{}

func (failedLogWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }
func (failedLogWriter) Close() error              { return nil }
func TestLoggingDiskFailureIsReported(t *testing.T) {
	failures := make(chan error, 1)
	logger := newSessionLog(failedLogWriter{}, func(err error) { failures <- err })
	logger.enqueue([]byte("output"))
	awaitDone(t, logger.done)
	select {
	case <-failures:
	default:
		t.Fatal("disk failure was hidden")
	}
}
