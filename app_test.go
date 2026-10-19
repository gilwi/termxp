package main

import (
	"strings"
	"testing"
)

func TestFormatMemory(t *testing.T) {
	tests := []struct {
		kb       uint64
		expected string
	}{
		{512, "0 MB"},
		{1024, "1 MB"},
		{1024 * 100, "100 MB"},
		{1024 * 1024, "1.0 GB"},
		{1024 * 1024 * 2, "2.0 GB"},
		{1024 * 1024 * 3 + 512*1024, "3.5 GB"},
	}

	for _, tt := range tests {
		actual := formatMemory(tt.kb)
		if actual != tt.expected {
			t.Errorf("formatMemory(%d) = %s; want %s", tt.kb, actual, tt.expected)
		}
	}
}

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		secs     float64
		expected string
	}{
		{30, "0m"},
		{120, "2m"},
		{3600, "1h 0m"},
		{3665, "1h 1m"},
		{86400, "1d 0h 0m"},
		{90060, "1d 1h 1m"},
	}

	for _, tt := range tests {
		actual := formatUptime(tt.secs)
		if actual != tt.expected {
			t.Errorf("formatUptime(%f) = %s; want %s", tt.secs, actual, tt.expected)
		}
	}
}

func TestFallbackMetricsCollector(t *testing.T) {
	collector := NewFallbackMetricsCollector()
	stats, err := collector.GetStats()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats["cpu"] != 0.0 {
		t.Errorf("expected cpu 0.0, got %v", stats["cpu"])
	}
	if stats["memory"] != 0.0 {
		t.Errorf("expected memory 0.0, got %v", stats["memory"])
	}
	if stats["memoryRaw"] != "0 MB" {
		t.Errorf("expected memoryRaw '0 MB', got %v", stats["memoryRaw"])
	}
	if !strings.Contains(stats["uptime"].(string), "m") {
		t.Errorf("expected uptime string with 'm', got %v", stats["uptime"])
	}
}

func TestProcStatParsingWithSpacesInComm(t *testing.T) {
	// Sample stat line where binary comm contains parentheses and spaces
	raw := "12345 (termxp pro (dev)) S 100 12345 12345 0 -1 4194304 100 0 0 0 450 150 0 0 20 0 1 0"

	lastParen := strings.LastIndex(raw, ")")
	if lastParen == -1 {
		t.Fatalf("expected last paren")
	}

	fields := strings.Fields(raw[lastParen+1:])
	if len(fields) < 13 {
		t.Fatalf("expected at least 13 fields, got %d", len(fields))
	}

	if fields[0] != "S" {
		t.Errorf("expected state 'S', got %s", fields[0])
	}
	if fields[11] != "450" {
		t.Errorf("expected utime '450', got %s", fields[11])
	}
	if fields[12] != "150" {
		t.Errorf("expected stime '150', got %s", fields[12])
	}
}
