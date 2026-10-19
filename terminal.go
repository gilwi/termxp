package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// TerminalSession represents an active shell process running in a PTY
type TerminalSession struct {
	ID       string
	cmd      *exec.Cmd
	ptyFile  *os.File
	mu       sync.Mutex
	closed   bool
	waitDone chan struct{}
	exitCode int
}

// TerminalService manages terminal sessions
type TerminalService struct {
	ctx          context.Context
	sessions     map[string]*TerminalSession
	mu           sync.Mutex
	shuttingDown bool
}

// NewTerminalService creates a new TerminalService instance
func NewTerminalService() *TerminalService {
	return &TerminalService{
		sessions: make(map[string]*TerminalSession),
	}
}

// setContext assigns the Wails context internally (unexported from Wails IPC bridge)
func (ts *TerminalService) setContext(ctx context.Context) {
	ts.ctx = ctx
}

// detectShell resolves the default shell path and appropriate login arguments
func detectShell() (string, []string) {
	if shell := os.Getenv("SHELL"); shell != "" {
		if _, err := os.Stat(shell); err == nil {
			return shell, []string{"--login"}
		}
	}
	if _, err := os.Stat("/bin/bash"); err == nil {
		return "/bin/bash", []string{"--login"}
	}
	return "/bin/sh", []string{}
}

// StartSession spawns a new shell inside a PTY with the given session ID
func (ts *TerminalService) StartSession(sessionID string, cols, rows int) error {
	if sessionID == "" || len(sessionID) > 128 {
		return fmt.Errorf("invalid session id")
	}

	// Clamp terminal dimensions to safe bounds
	if cols < 10 {
		cols = 80
	} else if cols > 1000 {
		cols = 1000
	}
	if rows < 2 {
		rows = 24
	} else if rows > 500 {
		rows = 500
	}

	ts.mu.Lock()
	if ts.shuttingDown {
		ts.mu.Unlock()
		return fmt.Errorf("terminal service is shutting down")
	}
	if _, exists := ts.sessions[sessionID]; exists {
		ts.mu.Unlock()
		return fmt.Errorf("session already exists: %s", sessionID)
	}
	ts.mu.Unlock()

	shellPath, shellArgs := detectShell()
	cmd := exec.Command(shellPath, shellArgs...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("failed to start command in PTY: %w", err)
	}

	// Set initial window size
	err = pty.Setsize(ptyFile, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
	if err != nil {
		// Clean up process and descriptor on partial failure
		_ = ptyFile.Close()
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			_ = cmd.Wait()
		}
		return fmt.Errorf("failed to set size: %w", err)
	}

	session := &TerminalSession{
		ID:       sessionID,
		cmd:      cmd,
		ptyFile:  ptyFile,
		waitDone: make(chan struct{}),
	}

	// Reap child process asynchronously so it never remains a zombie
	go func() {
		defer close(session.waitDone)
		if err := cmd.Wait(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				session.exitCode = exitErr.ExitCode()
			} else {
				session.exitCode = 1
			}
		} else {
			session.exitCode = 0
		}
	}()

	ts.mu.Lock()
	if ts.shuttingDown {
		ts.mu.Unlock()
		ts.killSessionInternal(session)
		return fmt.Errorf("terminal service is shutting down")
	}
	ts.sessions[sessionID] = session
	ts.mu.Unlock()

	// Start reading PTY output in a goroutine
	go ts.readLoop(session)

	return nil
}

// readLoop reads output from the PTY and emits coalesced events to the frontend.
// It synchronizes with the flusher so trailing bytes are never lost on exit.
func (ts *TerminalService) readLoop(s *TerminalSession) {
	buf := make([]byte, 32*1024)

	const (
		flushInterval = 12 * time.Millisecond // ~one frame; keeps latency invisible
		maxPending    = 64 * 1024             // force a flush before buffering grows unbounded
	)

	var (
		mu          sync.Mutex
		pending     []byte
		flushChan   = make(chan struct{}, 1) // size-triggered flush nudge
		done        = make(chan struct{})    // signals reader has stopped
		flusherDone = make(chan struct{})    // signals flusher has fully completed
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

		if ts.ctx != nil {
			runtime.EventsEmit(ts.ctx, "terminal:data:"+s.ID, string(data))
		}
	}

	// Flusher goroutine: time-based + size-triggered flushing.
	ticker := time.NewTicker(flushInterval)
	go func() {
		defer close(flusherDone)
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

	// Signal reader stop and wait for final flush to complete
	close(done)
	<-flusherDone

	// Wait briefly for process to be reaped and exit status recorded
	select {
	case <-s.waitDone:
	case <-time.After(1 * time.Second):
	}

	// Safely check and mark closed under session mutex
	s.mu.Lock()
	isClosed := s.closed
	s.closed = true
	s.mu.Unlock()

	if !isClosed {
		if ts.ctx != nil {
			runtime.EventsEmit(ts.ctx, "terminal:exit:"+s.ID, map[string]interface{}{
				"code": s.exitCode,
			})
		}
		ts.removeAndKillSession(s)
	}
}

// WriteToTerminal writes user input to the PTY without holding session mutex across blocking I/O
func (ts *TerminalService) WriteToTerminal(sessionID string, data string) error {
	if len(data) > 1024*1024 {
		return fmt.Errorf("write payload exceeds 1MB limit")
	}

	ts.mu.Lock()
	session, exists := ts.sessions[sessionID]
	ts.mu.Unlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.mu.Lock()
	if session.closed {
		session.mu.Unlock()
		return fmt.Errorf("session closed")
	}
	ptyFile := session.ptyFile
	session.mu.Unlock()

	_, err := ptyFile.Write([]byte(data))
	return err
}

// ResizeTerminal updates the terminal PTY winsize with validated bounds
func (ts *TerminalService) ResizeTerminal(sessionID string, cols, rows int) error {
	if cols < 10 {
		cols = 80
	} else if cols > 1000 {
		cols = 1000
	}
	if rows < 2 {
		rows = 24
	} else if rows > 500 {
		rows = 500
	}

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

	ts.killSessionInternal(session)
}

func (ts *TerminalService) removeAndKillSession(s *TerminalSession) {
	ts.mu.Lock()
	if ts.sessions[s.ID] == s {
		delete(ts.sessions, s.ID)
	}
	ts.mu.Unlock()

	ts.killSessionInternal(s)
}

func (ts *TerminalService) killSessionInternal(s *TerminalSession) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}
	s.closed = true

	if s.ptyFile != nil {
		_ = s.ptyFile.Close()
	}

	if s.cmd != nil && s.cmd.Process != nil {
		// Terminate the process group gracefully with SIGTERM
		_ = syscall.Kill(-s.cmd.Process.Pid, syscall.SIGTERM)

		// If process doesn't terminate quickly, forcefully kill with SIGKILL
		go func(pid int, waitDone <-chan struct{}) {
			select {
			case <-waitDone:
				return
			case <-time.After(200 * time.Millisecond):
				_ = syscall.Kill(-pid, syscall.SIGKILL)
			}
		}(s.cmd.Process.Pid, s.waitDone)
	}
}

// cleanupAllSessions shuts down all running terminal processes on exit (unexported from Wails IPC bridge)
func (ts *TerminalService) cleanupAllSessions() {
	ts.mu.Lock()
	ts.shuttingDown = true
	sessionsCopy := make([]*TerminalSession, 0, len(ts.sessions))
	for _, s := range ts.sessions {
		sessionsCopy = append(sessionsCopy, s)
	}
	ts.sessions = make(map[string]*TerminalSession)
	ts.mu.Unlock()

	for _, s := range sessionsCopy {
		ts.killSessionInternal(s)
	}
}
