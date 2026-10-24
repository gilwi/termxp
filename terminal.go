package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/unix"
)

const outputChunkSize = 32 * 1024

type TerminalOutput struct {
	Sequence uint64 `json:"sequence"`
	Data     string `json:"data"` // Base64 bytes, never a partial UTF-8 string.
}
type TerminalExit struct {
	Code int `json:"code"`
}
type SessionOptions struct {
	Shell string      `json:"shell"`
	Cwd   string      `json:"cwd"`
	SSH   *SSHProfile `json:"ssh,omitempty"`
}

type TerminalSession struct {
	ID            string
	cmd           *exec.Cmd
	ptyFile       *os.File
	mu            sync.Mutex
	closed        bool
	waitDone      chan struct{}
	done          chan struct{}
	stop          chan struct{}
	stopOnce      sync.Once
	fileOnce      sync.Once
	ack           chan uint64
	acknowledged  uint64
	sequence      uint64
	exitCode      int // Published by closing waitDone.
	logger        *sessionLog
	logOpening    bool
	logGeneration uint64
}

type TerminalService struct {
	ctx          context.Context
	sessions     map[string]*TerminalSession
	mu           sync.Mutex
	shuttingDown bool
	// Injectable event sink keeps transport/lifecycle tests independent of Wails.
	emit func(string, interface{})
}

func NewTerminalService() *TerminalService {
	return &TerminalService{sessions: make(map[string]*TerminalSession)}
}
func (ts *TerminalService) setContext(ctx context.Context) { ts.ctx = ctx }
func (ts *TerminalService) event(name string, value interface{}) {
	if ts.emit != nil {
		ts.emit(name, value)
	} else if ts.ctx != nil {
		runtime.EventsEmit(ts.ctx, name, value)
	}
}
func dimensions(cols, rows int) (int, int) {
	if cols < 10 {
		cols = 80
	}
	if cols > 1000 {
		cols = 1000
	}
	if rows < 2 {
		rows = 24
	}
	if rows > 500 {
		rows = 500
	}
	return cols, rows
}
func (ts *TerminalService) StartSession(id string, cols, rows int) error {
	return ts.StartSessionWithOptions(id, cols, rows, SessionOptions{})
}
func (ts *TerminalService) StartSessionWithOptions(id string, cols, rows int, options SessionOptions) error {
	if id == "" || len(id) > 128 {
		return fmt.Errorf("invalid session id")
	}
	cols, rows = dimensions(cols, rows)
	shell, args := detectShell()
	if options.Shell != "" && options.SSH == nil {
		if !executable(options.Shell) {
			return fmt.Errorf("shell must be an absolute executable path")
		}
		shell, args = options.Shell, shellArgs(options.Shell)
	}
	if options.SSH != nil {
		if err := validateSSH(*options.SSH); err != nil {
			return err
		}
		var err error
		shell, err = exec.LookPath("ssh")
		if err != nil {
			return fmt.Errorf("OpenSSH is unavailable: %w", err)
		}
		args = sshArgs(*options.SSH)
	}
	if options.Cwd != "" && !filepath.IsAbs(options.Cwd) {
		return fmt.Errorf("working directory must be absolute")
	}
	cmd := exec.Command(shell, args...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	cmd.Dir = options.Cwd
	// Hold the registry lock through spawn: duplicate IDs and shutdown cannot cross
	// the reservation/registration boundary. No blocking PTY I/O occurs here.
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.shuttingDown {
		return fmt.Errorf("terminal service is shutting down")
	}
	if _, exists := ts.sessions[id]; exists {
		return fmt.Errorf("session already exists: %s", id)
	}
	if len(ts.sessions) >= 64 {
		return fmt.Errorf("session limit reached (64)")
	}
	file, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return fmt.Errorf("start terminal: %w", err)
	}
	// creack/pty's ioctls call File.Fd(), which switches Go files to blocking
	// mode. Rewrap a nonblocking duplicate so Close can cancel pending I/O;
	// future resize ioctls use SyscallConn rather than File.Fd().
	file, err = pollablePTY(file)
	if err != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		_ = cmd.Wait()
		return fmt.Errorf("configure cancellable PTY: %w", err)
	}
	s := &TerminalSession{ID: id, cmd: cmd, ptyFile: file, waitDone: make(chan struct{}), done: make(chan struct{}), stop: make(chan struct{}), ack: make(chan uint64, 1)}
	ts.sessions[id] = s
	go func() {
		err := cmd.Wait()
		if err != nil {
			s.exitCode = 1
			var exit *exec.ExitError
			if errors.As(err, &exit) {
				s.exitCode = exit.ExitCode()
			}
		}
		close(s.waitDone)
	}()
	go ts.readLoop(s)
	go ts.watchCwd(s)
	return nil
}
func pollablePTY(original *os.File) (*os.File, error) {
	defer original.Close()
	raw, err := original.SyscallConn()
	if err != nil {
		return nil, err
	}
	duplicate := -1
	var fdErr error
	err = raw.Control(func(fd uintptr) {
		duplicate, fdErr = unix.FcntlInt(fd, unix.F_DUPFD_CLOEXEC, 0)
		if fdErr == nil {
			fdErr = unix.SetNonblock(duplicate, true)
		}
	})
	if err != nil || fdErr != nil {
		if duplicate >= 0 {
			_ = unix.Close(duplicate)
		}
		if err != nil {
			return nil, err
		}
		return nil, fdErr
	}
	return os.NewFile(uintptr(duplicate), original.Name()), nil
}

func (ts *TerminalService) session(id string) *TerminalSession {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.sessions[id]
}

func (ts *TerminalService) WriteToTerminal(id, data string) error {
	if len(data) > 1024*1024 {
		return fmt.Errorf("write payload exceeds 1MB limit")
	}
	s := ts.session(id)
	if s == nil {
		return fmt.Errorf("session not found: %s", id)
	}
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return fmt.Errorf("session closed")
	}
	_, err := s.ptyFile.Write([]byte(data))
	return err
}
func (ts *TerminalService) ResizeTerminal(id string, cols, rows int) error {
	s := ts.session(id)
	if s == nil {
		return fmt.Errorf("session not found: %s", id)
	}
	cols, rows = dimensions(cols, rows)
	raw, err := s.ptyFile.SyscallConn()
	if err != nil {
		return err
	}
	var resizeErr error
	err = raw.Control(func(fd uintptr) {
		resizeErr = unix.IoctlSetWinsize(int(fd), unix.TIOCSWINSZ, &unix.Winsize{Col: uint16(cols), Row: uint16(rows)})
	})
	if err != nil {
		return err
	}
	return resizeErr
}
func (ts *TerminalService) KillSession(id string) {
	if s := ts.session(id); s != nil {
		ts.killSessionInternal(s)
	}
}
func (ts *TerminalService) killSessionInternal(s *TerminalSession) {
	s.stopOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()
		close(s.stop)
		s.fileOnce.Do(func() { _ = s.ptyFile.Close() })
		select {
		case <-s.waitDone:
			return
		default:
		}
		_ = syscall.Kill(-s.cmd.Process.Pid, syscall.SIGTERM)
		go func() {
			select {
			case <-s.waitDone:
			case <-time.After(200 * time.Millisecond):
				_ = syscall.Kill(-s.cmd.Process.Pid, syscall.SIGKILL)
			}
		}()
	})
}
func (ts *TerminalService) cleanupAllSessions() {
	ts.mu.Lock()
	ts.shuttingDown = true
	sessions := make([]*TerminalSession, 0, len(ts.sessions))
	for _, s := range ts.sessions {
		sessions = append(sessions, s)
	}
	ts.mu.Unlock()
	for _, s := range sessions {
		ts.killSessionInternal(s)
	}
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	for _, s := range sessions {
		select {
		case <-s.done:
		case <-deadline.C:
			return
		}
	}
}
func (ts *TerminalService) watchCwd(s *TerminalSession) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	previous := ""
	for {
		if cwd, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", s.cmd.Process.Pid)); err == nil && cwd != previous {
			previous = cwd
			ts.event("terminal:cwd:"+s.ID, cwd)
		}
		select {
		case <-s.done:
			return
		case <-ticker.C:
		}
	}
}

// Logging is opt-in and creates a new private file; existing files are never overwritten.
func (ts *TerminalService) LogSession(id, path string) error {
	s := ts.session(id)
	if s == nil {
		return fmt.Errorf("session not found")
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return fmt.Errorf("session closed")
	}
	if path == "" {
		s.logGeneration++
		logger := s.logger
		s.mu.Unlock()
		if logger != nil {
			logger.finish()
		}
		return nil
	}
	if !filepath.IsAbs(path) {
		s.mu.Unlock()
		return fmt.Errorf("choose an absolute log path")
	}
	if s.logOpening {
		s.mu.Unlock()
		return fmt.Errorf("a log file is already opening")
	}
	if s.logger != nil {
		select {
		case <-s.logger.done:
		default:
			s.mu.Unlock()
			return fmt.Errorf("previous log is still active or finishing")
		}
	}
	s.logOpening = true
	generation := s.logGeneration
	s.mu.Unlock()
	// Opening a file may itself block on remote storage. Keep it outside the
	// lifecycle mutex and discard a late result after stop/close.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	s.mu.Lock()
	s.logOpening = false
	if err != nil {
		s.mu.Unlock()
		return err
	}
	if s.closed || generation != s.logGeneration {
		s.mu.Unlock()
		go file.Close()
		return fmt.Errorf("log start cancelled")
	}
	s.logger = newSessionLog(file, func(err error) { ts.event("terminal:log-error:"+id, err.Error()) })
	s.mu.Unlock()
	return nil
}

func (ts *TerminalService) finalizeSession(s *TerminalSession) {
	defer close(s.done)
	ts.killSessionInternal(s)
	<-s.waitDone
	s.mu.Lock()
	logger := s.logger
	s.mu.Unlock()
	if logger != nil {
		select {
		case <-logger.finish():
		case <-time.After(200 * time.Millisecond):
			logger.abort(fmt.Errorf("logging stopped before pending writes completed"))
		}
	}
	ts.mu.Lock()
	if ts.sessions[s.ID] == s {
		delete(ts.sessions, s.ID)
	}
	ts.mu.Unlock()
}
