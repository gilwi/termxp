package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func executable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0 && filepath.IsAbs(path)
}
func shellArgs(path string) []string {
	switch filepath.Base(path) {
	case "bash", "zsh", "fish", "ksh", "dash", "sh":
		return []string{"-l"}
	default:
		return nil
	}
}
func detectShell() (string, []string) {
	for _, path := range []string{os.Getenv("SHELL"), "/bin/bash", "/bin/sh"} {
		if executable(path) {
			return path, shellArgs(path)
		}
	}
	return "/bin/sh", nil
}
func (ts *TerminalService) ListShells() []string {
	preferred, _ := detectShell()
	shells := []string{preferred}
	if data, err := os.ReadFile("/etc/shells"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			path := strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
			if !executable(path) {
				continue
			}
			found := false
			for _, existing := range shells {
				if existing == path {
					found = true
				}
			}
			if !found {
				shells = append(shells, path)
			}
		}
	}
	return shells
}
func validateSSH(p SSHProfile) error {
	if strings.TrimSpace(p.Name) == "" || len(p.Name) > 128 {
		return fmt.Errorf("SSH profile requires a name (up to 128 characters)")
	}
	valid := func(s string) bool {
		return s != "" && !strings.HasPrefix(s, "-") && !strings.ContainsAny(s, " \t\r\n\x00")
	}
	if !valid(p.Host) || !regexp.MustCompile(`^[a-zA-Z0-9_.:\[\]-]+$`).MatchString(p.Host) {
		return fmt.Errorf("invalid SSH host")
	}
	if p.User != "" && (!valid(p.User) || !regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`).MatchString(p.User)) {
		return fmt.Errorf("invalid SSH user")
	}
	if p.Port < 0 || p.Port > 65535 {
		return fmt.Errorf("invalid SSH port")
	}
	if p.KeyPath != "" && !filepath.IsAbs(p.KeyPath) {
		return fmt.Errorf("SSH key path must be absolute")
	}
	if p.JumpHost != "" && (!valid(p.JumpHost) || !regexp.MustCompile(`^[a-zA-Z0-9_.:@,\[\]-]+$`).MatchString(p.JumpHost)) {
		return fmt.Errorf("invalid SSH jump host")
	}
	return nil
}
func sshArgs(p SSHProfile) []string {
	// No shell interpolation. Separate option values prevent host/user option injection.
	args := []string{}
	if p.Port != 0 {
		args = append(args, "-p", strconv.Itoa(p.Port))
	}
	if p.User != "" {
		args = append(args, "-l", p.User)
	}
	if p.KeyPath != "" {
		args = append(args, "-i", p.KeyPath)
	}
	if p.JumpHost != "" {
		args = append(args, "-J", p.JumpHost)
	}
	return append(args, "--", p.Host)
}
