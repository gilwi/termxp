package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// HistoryEntry represents a command executed in a session
type HistoryEntry struct {
	ID        string `json:"id"`
	Command   string `json:"command"`
	Host      string `json:"host"`
	Timestamp int64  `json:"timestamp"`
}

// HistoryService manages persistence of command history
type HistoryService struct {
	entries  []HistoryEntry
	filePath string
	mu       sync.Mutex
}

// NewHistoryService creates a new HistoryService instance
func NewHistoryService() *HistoryService {
	homeDir, err := os.UserHomeDir()
	var filePath string
	if err != nil {
		filePath = ".termxp_history.json"
	} else {
		filePath = filepath.Join(homeDir, ".termxp_history.json")
	}
	return &HistoryService{
		entries:  []HistoryEntry{},
		filePath: filePath,
	}
}

// LoadHistory loads and returns all history entries
func (hs *HistoryService) LoadHistory() ([]HistoryEntry, error) {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	file, err := os.ReadFile(hs.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []HistoryEntry{}, nil
		}
		return nil, err
	}

	var entries []HistoryEntry
	err = json.Unmarshal(file, &entries)
	if err != nil {
		return nil, err
	}
	hs.entries = entries
	return entries, nil
}

// AddHistoryEntry appends an entry to the history file
func (hs *HistoryService) AddHistoryEntry(command, host string) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	// Load current entries first
	var entries []HistoryEntry
	file, err := os.ReadFile(hs.filePath)
	if err == nil {
		_ = json.Unmarshal(file, &entries)
	}

	// Create new entry
	entry := HistoryEntry{
		ID:        uuid.New().String(),
		Command:   command,
		Host:      host,
		Timestamp: time.Now().Unix(),
	}

	// Prepend to entries
	entries = append([]HistoryEntry{entry}, entries...)

	// Cap at 1000 entries
	if len(entries) > 1000 {
		entries = entries[:1000]
	}

	// Marshal and save
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(hs.filePath, data, 0644)
	if err != nil {
		return err
	}

	hs.entries = entries
	return nil
}
