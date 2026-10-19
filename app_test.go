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
		{1024*1024*3 + 512*1024, "3.5 GB"},
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

	if stats.CPU != nil || stats.Memory != nil || stats.MemoryRaw != "Unavailable" {
		t.Fatalf("fallback metrics must report unavailable: %+v", stats)
	}
	if !strings.Contains(stats.Uptime, "m") {
		t.Fatalf("invalid uptime: %s", stats.Uptime)
	}
}

func TestProcStatParsingWithSpacesInComm(t *testing.T) {
	// Sample stat line where binary comm contains parentheses and spaces
	raw := "12345 (termxp pro (dev)) S 100 12345 12345 0 -1 4194304 100 0 0 0 450 150 0 0 20 0 1 0"

	proc, total, err := parseProcessCPUStats(raw, 1000)
	if err != nil || proc != 600 || total != 1000 {
		t.Fatalf("parse = %d %d %v", proc, total, err)
	}
	for _, invalid := range []string{"", "123 (broken", "123 (x) S", strings.Replace(raw, "450", "invalid", 1)} {
		if _, _, err := parseProcessCPUStats(invalid, 1000); err == nil {
			t.Errorf("accepted invalid stat: %q", invalid)
		}
	}
}
