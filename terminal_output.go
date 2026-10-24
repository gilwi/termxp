package main

import (
	"encoding/base64"
	"time"
)

// One chunk is in flight at a time. xterm acknowledges only after consuming it,
// so a slow renderer applies backpressure all the way to the child process.
func (ts *TerminalService) readLoop(s *TerminalSession) {
	defer ts.finalizeSession(s)
	buf := make([]byte, outputChunkSize)
	for {
		n, err := s.ptyFile.Read(buf)
		if n > 0 {
			s.mu.Lock()
			logger := s.logger
			s.sequence++
			seq := s.sequence
			s.mu.Unlock()
			if logger != nil {
				logger.enqueue(buf[:n])
			}
			ts.event("terminal:data:"+s.ID, TerminalOutput{Sequence: seq, Data: base64.StdEncoding.EncodeToString(buf[:n])})
			if ts.ctx != nil || ts.emit != nil {
				for {
					select {
					case ack := <-s.ack:
						if ack == seq {
							goto consumed
						}
					case <-s.stop:
						return
					}
				}
			}
		}
	consumed:
		if err != nil {
			break
		}
	}
	// A closed slave can precede process exit. Ensure it cannot keep a session alive.
	select {
	case <-s.waitDone:
	case <-time.After(200 * time.Millisecond):
		ts.killSessionInternal(s)
		<-s.waitDone
	}
	select {
	case <-s.stop:
		return
	default:
	}
	ts.event("terminal:exit:"+s.ID, TerminalExit{Code: s.exitCode})
}
func (ts *TerminalService) AcknowledgeOutput(id string, sequence uint64) {
	s := ts.session(id)
	if s == nil {
		return
	}
	s.mu.Lock()
	valid := sequence == s.sequence && sequence > s.acknowledged
	if valid {
		s.acknowledged = sequence
	}
	s.mu.Unlock()
	if valid {
		select {
		case s.ack <- sequence:
		default:
		}
	}
}
