package main

import (
	"regexp"
	"strings"
)

func defaultSnippets() []Snippet {
	return []Snippet{
		{ID: "list", Label: "List Files", Command: "ls -lah"}, {ID: "system", Label: "System Info", Command: "uname -a"},
		{ID: "disk", Label: "Disk Space", Command: "df -h"}, {ID: "network", Label: "Active Network", Command: "ss -tulpn"},
		{ID: "cpu", Label: "CPU Load Check", Command: "cat /proc/loadavg"}, {ID: "who", Label: "Who Am I", Command: "whoami && pwd"},
	}
}
func defaultShortcuts() map[string]string {
	return map[string]string{
		"new": "Ctrl+Shift+T", "close": "Ctrl+Shift+W", "right": "Ctrl+Shift+E", "below": "Ctrl+Shift+O", "maximize": "Ctrl+Shift+X", "search": "Ctrl+F", "palette": "Ctrl+P",
	}
}
func validShortcut(key string) bool {
	if key == "" {
		return true
	}
	if !regexp.MustCompile(`^(Ctrl\+)?(Alt\+)?(Shift\+)?[A-Z0-9]$`).MatchString(key) {
		return false
	}
	if !strings.Contains(key, "Ctrl+") && !strings.Contains(key, "Alt+") {
		return false
	}
	// Clipboard and desktop close bindings must retain their established meaning.
	return key != "Ctrl+Shift+C" && key != "Ctrl+Shift+V" && key != "Alt+F4"
}
