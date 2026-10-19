package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func awaitDone(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(4 * time.Second):
		t.Fatal("session did not finish")
	}
}
func TestSimultaneousDuplicateStart(t *testing.T) {
	ts := NewTerminalService()
	t.Cleanup(ts.cleanupAllSessions)
	var successes atomic.Int32
	var wg sync.WaitGroup
	gate := make(chan struct{})
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-gate
			if ts.StartSession("same", 80, 24) == nil {
				successes.Add(1)
			}
		}()
	}
	close(gate)
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("created %d sessions for one ID", successes.Load())
	}
}
func TestNaturalExitClosesDescriptorAndRegistry(t *testing.T) {
	ts := NewTerminalService()
	t.Cleanup(ts.cleanupAllSessions)
	if err := ts.StartSession("natural", 80, 24); err != nil {
		t.Fatal(err)
	}
	s := ts.session("natural")
	if err := ts.WriteToTerminal(s.ID, "exit 7\n"); err != nil {
		t.Fatal(err)
	}
	awaitDone(t, s.done)
	if s.exitCode != 7 {
		t.Fatalf("exit code = %d", s.exitCode)
	}
	if ts.session(s.ID) != nil {
		t.Fatal("exited session remains registered")
	}
	if _, err := s.ptyFile.Stat(); err == nil {
		t.Fatal("PTY remains open")
	}
}
func TestOutputBytesAndExitOrdering(t *testing.T) {
	ts := NewTerminalService()
	t.Cleanup(ts.cleanupAllSessions)
	var output bytes.Buffer
	var mu sync.Mutex
	exited := make(chan struct{})
	var last uint64
	ts.emit = func(name string, value interface{}) {
		if strings.HasPrefix(name, "terminal:data:") {
			event := value.(TerminalOutput)
			data, err := base64.StdEncoding.DecodeString(event.Data)
			if err != nil {
				t.Error(err)
			}
			mu.Lock()
			if event.Sequence != last+1 {
				t.Errorf("out of order: %d after %d", event.Sequence, last)
			}
			last = event.Sequence
			output.Write(data)
			mu.Unlock()
			ts.AcknowledgeOutput("bytes", event.Sequence)
		}
		if name == "terminal:exit:bytes" {
			mu.Lock()
			text := output.String()
			mu.Unlock()
			if !strings.Contains(text, "\xf0\x9f\x98\x80FINAL") {
				t.Errorf("missing final Unicode bytes: %q", text)
			}
			if value.(TerminalExit).Code != 3 {
				t.Errorf("unexpected exit: %v", value)
			}
			close(exited)
		}
	}
	if err := ts.StartSession("bytes", 80, 24); err != nil {
		t.Fatal(err)
	}
	s := ts.session("bytes")
	// Force a Unicode character across separate writes/transport chunks.
	if err := ts.WriteToTerminal("bytes", "stty -echo; printf '\\360\\237'; sleep 0.05; printf '\\230\\200FINAL'; exit 3\n"); err != nil {
		t.Fatal(err)
	}
	awaitDone(t, exited)
	awaitDone(t, s.done)
}
func TestSlowConsumerIsBoundedAndCancellationWorks(t *testing.T) {
	ts := NewTerminalService()
	t.Cleanup(ts.cleanupAllSessions)
	output := make(chan TerminalOutput, 10)
	ts.emit = func(name string, value interface{}) {
		if name == "terminal:data:slow" {
			output <- value.(TerminalOutput)
		}
	}
	if err := ts.StartSession("slow", 80, 24); err != nil {
		t.Fatal(err)
	}
	s := ts.session("slow")
	if err := ts.WriteToTerminal("slow", "yes output\n"); err != nil {
		t.Fatal(err)
	}
	var first TerminalOutput
	select {
	case first = <-output:
	case <-time.After(2 * time.Second):
		t.Fatal("no output")
	}
	if len(first.Data) > base64.StdEncoding.EncodedLen(outputChunkSize) {
		t.Fatal("output chunk exceeded bound")
	}
	ts.AcknowledgeOutput("slow", first.Sequence+1) // Future acknowledgements cannot release a chunk.
	select {
	case <-output:
		t.Fatal("producer advanced without consumption")
	case <-time.After(100 * time.Millisecond):
	}
	ts.KillSession("slow")
	awaitDone(t, s.done)
}
func TestBlockedWriteDoesNotBlockShutdown(t *testing.T) {
	ts := NewTerminalService()
	t.Cleanup(ts.cleanupAllSessions)
	if err := ts.StartSession("blocked", 80, 24); err != nil {
		t.Fatal(err)
	}
	s := ts.session("blocked")
	if err := ts.ResizeTerminal("blocked", 100, 30); err != nil {
		t.Fatal(err)
	}
	if err := ts.WriteToTerminal("blocked", "stty -echo; sleep 60\n"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(80 * time.Millisecond)
	written := make(chan struct{})
	go func() { _ = ts.WriteToTerminal("blocked", strings.Repeat("x", 1024*1024)); close(written) }()
	time.Sleep(30 * time.Millisecond)
	shutdown := make(chan struct{})
	go func() { ts.cleanupAllSessions(); close(shutdown) }()
	awaitDone(t, shutdown)
	awaitDone(t, written)
	awaitDone(t, s.done)
}
func TestConcurrentStartShutdown(t *testing.T) {
	ts := NewTerminalService()
	t.Cleanup(ts.cleanupAllSessions)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) { defer wg.Done(); _ = ts.StartSession(fmt.Sprint(i), 80, 24) }(i)
	}
	wg.Add(1)
	go func() { defer wg.Done(); ts.cleanupAllSessions() }()
	wg.Wait()
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if len(ts.sessions) != 0 {
		t.Fatalf("%d sessions survived shutdown", len(ts.sessions))
	}
}
func TestShellValidationAndLogging(t *testing.T) {
	ts := NewTerminalService()
	t.Cleanup(ts.cleanupAllSessions)
	path := t.TempDir() + "/shell"
	if err := os.WriteFile(path, []byte("not executable"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SHELL", path)
	if shell, _ := detectShell(); shell == path {
		t.Fatal("selected non-executable shell")
	}
	if err := ts.StartSessionWithOptions("bad", 80, 24, SessionOptions{Shell: path}); err == nil {
		t.Fatal("accepted non-executable shell")
	}
	if err := ts.StartSession("log", 80, 24); err != nil {
		t.Fatal(err)
	}
	s := ts.session("log")
	logPath := t.TempDir() + "/output.log"
	if err := ts.LogSession("log", logPath); err != nil {
		t.Fatal(err)
	}
	if err := ts.WriteToTerminal("log", "echo LOG_MARKER; exit\n"); err != nil {
		t.Fatal(err)
	}
	awaitDone(t, s.done)
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("LOG_MARKER")) {
		t.Fatal("log missing output")
	}
	info, _ := os.Stat(logPath)
	if info.Mode().Perm() != 0600 {
		t.Fatalf("permissions = %o", info.Mode().Perm())
	}
}

func TestSustainedOutputHasBoundedChunks(t *testing.T) {
	ts := NewTerminalService()
	t.Cleanup(ts.cleanupAllSessions)
	var bytesSeen atomic.Int64
	exited := make(chan struct{})
	ts.emit = func(name string, value interface{}) {
		if name == "terminal:data:load" {
			packet := value.(TerminalOutput)
			data, err := base64.StdEncoding.DecodeString(packet.Data)
			if err != nil {
				t.Error(err)
			}
			if len(data) > outputChunkSize {
				t.Errorf("oversized chunk: %d", len(data))
			}
			bytesSeen.Add(int64(len(data)))
			// Model a consumer that occasionally stalls, without growing an output queue.
			if packet.Sequence%16 == 0 {
				time.Sleep(time.Millisecond)
			}
			ts.AcknowledgeOutput("load", packet.Sequence)
			ts.AcknowledgeOutput("load", packet.Sequence) // Duplicate delivery must not clog the next packet.
		}
		if name == "terminal:exit:load" {
			close(exited)
		}
	}
	if err := ts.StartSession("load", 80, 24); err != nil {
		t.Fatal(err)
	}
	s := ts.session("load")
	if err := ts.WriteToTerminal("load", "stty -echo; head -c 8388608 /dev/zero; exit\n"); err != nil {
		t.Fatal(err)
	}
	awaitDone(t, exited)
	awaitDone(t, s.done)
	if bytesSeen.Load() < 8388608 {
		t.Fatalf("lost output: %d bytes", bytesSeen.Load())
	}
}
