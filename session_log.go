package main

import (
	"fmt"
	"io"
	"sync"
)

// sessionLog owns disk I/O. Neither its writer nor Close runs under session locks.
// At most one active or still-finishing sink is allowed per session.
type sessionLog struct {
	mu        sync.Mutex
	writer    io.WriteCloser
	queue     chan []byte
	stop      chan struct{}
	done      chan struct{}
	accepting bool
	abortOnce sync.Once
	report    func(error)
}

func newSessionLog(writer io.WriteCloser, report func(error)) *sessionLog {
	log := &sessionLog{writer: writer, queue: make(chan []byte, 32), stop: make(chan struct{}), done: make(chan struct{}), accepting: true, report: report}
	go log.run()
	return log
}
func (l *sessionLog) enqueue(data []byte) {
	l.mu.Lock()
	if !l.accepting {
		l.mu.Unlock()
		return
	}
	select {
	case l.queue <- append([]byte(nil), data...):
		l.mu.Unlock()
	default:
		l.mu.Unlock()
		l.abort(fmt.Errorf("logging stopped: disk queue exceeded 1 MiB"))
	}
}
func (l *sessionLog) finish() <-chan struct{} {
	l.mu.Lock()
	if l.accepting {
		l.accepting = false
		close(l.queue)
	}
	l.mu.Unlock()
	return l.done
}
func (l *sessionLog) abort(err error) {
	l.abortOnce.Do(func() {
		l.mu.Lock()
		l.accepting = false
		l.mu.Unlock()
		close(l.stop)
		// Some filesystems cannot cancel a write; never wait for it in terminal cleanup.
		go l.writer.Close()
		if err != nil && l.report != nil {
			l.report(err)
		}
	})
}
func (l *sessionLog) run() {
	defer close(l.done)
	defer l.writer.Close()
	for {
		select {
		case <-l.stop:
			return
		default:
		}
		select {
		case <-l.stop:
			return
		case data, ok := <-l.queue:
			if !ok {
				return
			}
			if _, err := l.writer.Write(data); err != nil {
				l.abort(fmt.Errorf("logging stopped: %w", err))
				return
			}
		}
	}
}
