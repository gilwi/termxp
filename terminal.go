package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// TerminalSession represents an active shell process running in a PTY
type TerminalSession struct {
	ID      string
	cmd     *exec.Cmd
	ptyFile *os.File
	mu      sync.Mutex
	closed  bool
}

// TerminalService manages terminal sessions
type TerminalService struct {
	ctx      context.Context
	sessions map[string]*TerminalSession
	mu       sync.Mutex
}

// NewTerminalService creates a new TerminalService instance
func NewTerminalService() *TerminalService {
	return &TerminalService{
		sessions: make(map[string]*TerminalSession),
	}
}

// SetContext assigns the Wails context to this service
func (ts *TerminalService) SetContext(ctx context.Context) {
	ts.ctx = ctx
}

// StartSession spawns a new shell inside a PTY with the given session ID
func (ts *TerminalService) StartSession(sessionID string, cols, rows int) error {
	// Try to find a suitable shell (bash or sh)
	shellPath := "/bin/bash"
	if _, err := os.Stat(shellPath); os.IsNotExist(err) {
		shellPath = "/bin/sh"
	}

	// Start command inside PTY
	cmd := exec.Command(shellPath, "--login")
	// Set custom environment variables so the terminal behaves like a modern color terminal
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("failed to start command in PTY: %w", err)
	}

	// Set initial win size
	err = pty.Setsize(ptyFile, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
	if err != nil {
		ptyFile.Close()
		return fmt.Errorf("failed to set size: %w", err)
	}

	session := &TerminalSession{
		ID:      sessionID,
		cmd:     cmd,
		ptyFile: ptyFile,
	}

	ts.mu.Lock()
	ts.sessions[sessionID] = session
	ts.mu.Unlock()

	// Start reading PTY output in a goroutine
	go ts.readLoop(session)

	return nil
}

// readLoop reads output from the PTY and emits coalesced events to the frontend.
// Instead of emitting one event per read, it batches bytes and flushes at most
// once per frame (or sooner if the buffer grows large), which dramatically
// reduces IPC/serialization overhead during bursty output.
func (ts *TerminalService) readLoop(s *TerminalSession) {
	buf := make([]byte, 32*1024)

	const (
		flushInterval = 12 * time.Millisecond // ~one frame; keeps latency invisible
		maxPending    = 64 * 1024             // force a flush before buffering grows unbounded
	)

	var (
		mu        sync.Mutex
		pending   []byte
		flushChan = make(chan struct{}, 1) // size-triggered flush nudge
		done      = make(chan struct{})    // signals reader has stopped
	)

	// flush emits whatever is currently buffered as a single event.
	flush := func() {
		mu.Lock()
		if len(pending) == 0 {
			mu.Unlock()
			return
		}
		data := pending
		pending = nil
		mu.Unlock()
		runtime.EventsEmit(ts.ctx, "terminal:data:"+s.ID, string(data))
	}

	// Flusher goroutine: time-based + size-triggered flushing.
	ticker := time.NewTicker(flushInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				flush()
			case <-flushChan:
				flush()
			case <-done:
				flush() // final flush so trailing bytes aren't lost
				return
			}
		}
	}()

	// Reader loop.
	for {
		n, err := s.ptyFile.Read(buf)
		if n > 0 {
			mu.Lock()
			pending = append(pending, buf[:n]...)
			tooBig := len(pending) >= maxPending
			mu.Unlock()

			if tooBig {
				// Non-blocking nudge; if a flush is already queued, skip.
				select {
				case flushChan <- struct{}{}:
				default:
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				// Normal termination
			}
			break
		}
	}

	// Stop the flusher and perform the final flush before emitting exit.
	close(done)

	// Clean up and emit exit event
	ts.mu.Lock()
	isClosed := s.closed
	ts.mu.Unlock()

	if !isClosed {
		runtime.EventsEmit(ts.ctx, "terminal:exit:"+s.ID, nil)
		ts.KillSession(s.ID)
	}
}

// WriteToTerminal writes user input to the PTY
func (ts *TerminalService) WriteToTerminal(sessionID string, data string) error {
	ts.mu.Lock()
	session, exists := ts.sessions[sessionID]
	ts.mu.Unlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.closed {
		return fmt.Errorf("session closed")
	}

	_, err := session.ptyFile.Write([]byte(data))
	return err
}

// ResizeTerminal updates the terminal PTY winsize
func (ts *TerminalService) ResizeTerminal(sessionID string, cols, rows int) error {
	ts.mu.Lock()
	session, exists := ts.sessions[sessionID]
	ts.mu.Unlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.closed {
		return fmt.Errorf("session closed")
	}

	return pty.Setsize(session.ptyFile, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
}

// KillSession terminates the session command and closes file descriptors
func (ts *TerminalService) KillSession(sessionID string) {
	ts.mu.Lock()
	session, exists := ts.sessions[sessionID]
	if exists {
		delete(ts.sessions, sessionID)
	}
	ts.mu.Unlock()

	if !exists {
		return
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.closed {
		return
	}

	session.closed = true
	_ = session.ptyFile.Close()
	if session.cmd.Process != nil {
		_ = session.cmd.Process.Kill()
	}
}

// CleanupAllSessions shuts down all running terminal processes (called on exit)
func (ts *TerminalService) CleanupAllSessions() {
	ts.mu.Lock()
	sessionsCopy := make([]*TerminalSession, 0, len(ts.sessions))
	for _, s := range ts.sessions {
		sessionsCopy = append(sessionsCopy, s)
	}
	ts.sessions = make(map[string]*TerminalSession)
	ts.mu.Unlock()

	for _, s := range sessionsCopy {
		s.mu.Lock()
		if !s.closed {
			s.closed = true
			_ = s.ptyFile.Close()
			if s.cmd.Process != nil {
				_ = s.cmd.Process.Kill()
			}
		}
		s.mu.Unlock()
	}
}
