package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestTerminalServiceValidation(t *testing.T) {
	ts := NewTerminalService()

	// Empty session ID
	err := ts.StartSession("", 80, 24)
	if err == nil {
		t.Errorf("expected error for empty session id, got nil")
	}

	// Very long session ID
	longID := string(make([]byte, 200))
	err = ts.StartSession(longID, 80, 24)
	if err == nil {
		t.Errorf("expected error for oversized session id, got nil")
	}

	// Non-existent session operations
	err = ts.WriteToTerminal("non-existent", "hello")
	if err == nil {
		t.Errorf("expected error writing to non-existent session, got nil")
	}

	err = ts.ResizeTerminal("non-existent", 80, 24)
	if err == nil {
		t.Errorf("expected error resizing non-existent session, got nil")
	}

	// Payload size limit check
	hugePayload := string(make([]byte, 2*1024*1024))
	err = ts.WriteToTerminal("any", hugePayload)
	if err == nil {
		t.Errorf("expected error for payload > 1MB, got nil")
	}
}

func TestTerminalSessionLifecycle(t *testing.T) {
	ts := NewTerminalService()
	sessionID := "test-session-1"

	err := ts.StartSession(sessionID, 80, 24)
	if err != nil {
		t.Fatalf("failed to start session: %v", err)
	}

	// Duplicate session ID should be rejected
	err = ts.StartSession(sessionID, 80, 24)
	if err == nil {
		t.Errorf("expected error starting duplicate session ID, got nil")
	}

	// Write command to shell
	err = ts.WriteToTerminal(sessionID, "echo hello\n")
	if err != nil {
		t.Errorf("failed to write to terminal: %v", err)
	}

	// Resize terminal
	err = ts.ResizeTerminal(sessionID, 100, 30)
	if err != nil {
		t.Errorf("failed to resize terminal: %v", err)
	}

	// Kill session
	ts.KillSession(sessionID)

	// Writing to killed session should return error
	err = ts.WriteToTerminal(sessionID, "after kill\n")
	if err == nil {
		t.Errorf("expected error writing to killed session, got nil")
	}

	// Multiple kills should not panic or error
	ts.KillSession(sessionID)
}

func TestConcurrentTerminalOperations(t *testing.T) {
	ts := NewTerminalService()
	const sessionCount = 5

	var wg sync.WaitGroup
	for i := 0; i < sessionCount; i++ {
		wg.Add(1)
		sID := fmt.Sprintf("concurrent-%d", i)
		go func(id string) {
			defer wg.Done()
			err := ts.StartSession(id, 80, 24)
			if err != nil {
				t.Errorf("concurrent start failed: %v", err)
				return
			}

			// Perform rapid concurrent writes and resizes
			for j := 0; j < 10; j++ {
				_ = ts.WriteToTerminal(id, "echo 1\n")
				_ = ts.ResizeTerminal(id, 80+j, 24+j)
				time.Sleep(2 * time.Millisecond)
			}

			ts.KillSession(id)
		}(sID)
	}

	wg.Wait()
}

func TestCleanupAllSessions(t *testing.T) {
	ts := NewTerminalService()

	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("cleanup-%d", i)
		err := ts.StartSession(id, 80, 24)
		if err != nil {
			t.Fatalf("failed to start session %s: %v", id, err)
		}
	}

	ts.cleanupAllSessions()

	// Starting new session after cleanup should fail because shutting down
	err := ts.StartSession("new-after-shutdown", 80, 24)
	if err == nil {
		t.Errorf("expected error starting session after shutdown, got nil")
	}
}

func TestProcessReaping(t *testing.T) {
	ts := NewTerminalService()
	sessionID := "reap-test"

	err := ts.StartSession(sessionID, 80, 24)
	if err != nil {
		t.Fatalf("failed to start session: %v", err)
	}

	ts.mu.Lock()
	session := ts.sessions[sessionID]
	ts.mu.Unlock()

	if session == nil {
		t.Fatalf("expected active session")
	}

	// Exit the shell cleanly
	_ = ts.WriteToTerminal(sessionID, "exit 0\n")

	// Wait for waitDone channel to close, confirming process was reaped
	select {
	case <-session.waitDone:
		if session.exitCode != 0 {
			t.Errorf("expected exit code 0, got %d", session.exitCode)
		}
	case <-time.After(2 * time.Second):
		t.Errorf("timed out waiting for process to be reaped")
	}
}

