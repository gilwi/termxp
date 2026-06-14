package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/creack/pty"
	"github.com/google/uuid"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
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

// configPath returns the path to the termxp config file
func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "termxp", "wsl_distro")
}

// GetWSLDistro returns the saved WSL distro name, or "" if none is set
func (ts *TerminalService) GetWSLDistro() string {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// SetWSLDistro saves the chosen WSL distro name to disk
func (ts *TerminalService) SetWSLDistro(distro string) error {
	p := configPath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(strings.TrimSpace(distro)), 0644)
}

// ListWSLDistros returns the list of installed WSL distro names.
// Returns an empty slice on non-Windows or when WSL is not installed.
func (ts *TerminalService) ListWSLDistros() ([]string, error) {
	if runtime.GOOS != "windows" {
		return []string{}, nil
	}
	wslPath, err := exec.LookPath("wsl.exe")
	if err != nil {
		return []string{}, nil
	}

	// wsl.exe --list --quiet outputs UTF-16LE on Windows; strip BOM / null bytes
	out, err := exec.Command(wslPath, "--list", "--quiet").Output()
	if err != nil {
		return []string{}, fmt.Errorf("wsl --list failed: %w", err)
	}

	// Strip UTF-16 null bytes (every other byte is 0x00 in UTF-16LE ASCII range)
	cleaned := bytes.ReplaceAll(out, []byte{0x00}, []byte{})
	// Strip BOM if present
	cleaned = bytes.TrimPrefix(cleaned, []byte{0xFF, 0xFE})

	var distros []string
	scanner := bufio.NewScanner(bytes.NewReader(cleaned))
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name != "" {
			distros = append(distros, name)
		}
	}
	return distros, nil
}

// resolveShell returns the command and arguments to launch a shell.
// On Windows it uses the saved WSL distro, or falls back to cmd.exe.
// On Unix it picks bash or sh.
func resolveShell() (string, []string) {
	if runtime.GOOS == "windows" {
		if wslPath, err := exec.LookPath("wsl.exe"); err == nil {
			ts := &TerminalService{}
			distro := ts.GetWSLDistro()
			if distro != "" {
				return wslPath, []string{"-d", distro}
			}
			// WSL present but no distro configured — fall through to cmd.exe
		}
		return "cmd.exe", []string{}
	}

	for _, sh := range []string{"/bin/bash", "/bin/sh"} {
		if _, err := os.Stat(sh); err == nil {
			return sh, []string{"--login"}
		}
	}
	return "/bin/sh", []string{"--login"}
}

// StartSession spawns a new shell inside a PTY and returns the session ID
func (ts *TerminalService) StartSession(cols, rows int) (string, error) {
	sessionID := uuid.New().String()

	shellPath, shellArgs := resolveShell()

	cmd := exec.Command(shellPath, shellArgs...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to start command in PTY: %w", err)
	}

	err = pty.Setsize(ptyFile, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
	if err != nil {
		ptyFile.Close()
		return "", fmt.Errorf("failed to set size: %w", err)
	}

	session := &TerminalSession{
		ID:      sessionID,
		cmd:     cmd,
		ptyFile: ptyFile,
	}

	ts.mu.Lock()
	ts.sessions[sessionID] = session
	ts.mu.Unlock()

	go ts.readLoop(session)

	return sessionID, nil
}

// readLoop reads output from the PTY and emits events to the frontend
func (ts *TerminalService) readLoop(s *TerminalSession) {
	buf := make([]byte, 32*1024)
	for {
		n, err := s.ptyFile.Read(buf)
		if err != nil {
			if err == io.EOF {
				// Normal termination
			}
			break
		}
		if n > 0 {
			wailsRuntime.EventsEmit(ts.ctx, "terminal:data:"+s.ID, string(buf[:n]))
		}
	}

	ts.mu.Lock()
	isClosed := s.closed
	ts.mu.Unlock()

	if !isClosed {
		wailsRuntime.EventsEmit(ts.ctx, "terminal:exit:"+s.ID, nil)
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
