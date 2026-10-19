package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type SSHProfile struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	User     string `json:"user"`
	Port     int    `json:"port"`
	KeyPath  string `json:"keyPath"`
	JumpHost string `json:"jumpHost"`
}

// Workspace descriptors deliberately contain no runtime IDs or terminal output.
type WorkspaceNode struct {
	Type        string          `json:"type"`
	Orientation string          `json:"orientation,omitempty"`
	Children    []WorkspaceNode `json:"children,omitempty"`
	Sizes       []float64       `json:"sizes,omitempty"`
	Shell       string          `json:"shell,omitempty"`
	Cwd         string          `json:"cwd,omitempty"`
	SSH         *SSHProfile     `json:"ssh,omitempty"`
}
type WorkspaceTab struct {
	Name string        `json:"name"`
	Root WorkspaceNode `json:"root"`
}
type Config struct {
	Version          int            `json:"version"`
	Theme            string         `json:"theme"`
	FontSize         int            `json:"fontSize"`
	DefaultShell     string         `json:"defaultShell"`
	CopyOnSelect     bool           `json:"copyOnSelect"`
	RestoreWorkspace bool           `json:"restoreWorkspace"`
	Workspace        []WorkspaceTab `json:"workspace"`
	SSHProfiles      []SSHProfile   `json:"sshProfiles"`
}
type ConfigResult struct {
	Config  Config `json:"config"`
	Warning string `json:"warning"`
}
type ConfigService struct {
	mu   sync.Mutex
	path string
}

func NewConfigService() *ConfigService { return &ConfigService{} }
func defaultConfig() Config {
	return Config{Version: 1, Theme: "glassmorphic", FontSize: 14, RestoreWorkspace: true, Workspace: []WorkspaceTab{}, SSHProfiles: []SSHProfile{}}
}
func (s *ConfigService) configPath() (string, error) {
	if s.path != "" {
		return s.path, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "termxp", "config.json"), nil
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
func validateConfig(c Config) error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported config version %d", c.Version)
	}
	switch c.Theme {
	case "glassmorphic", "cyberpunk", "dracula", "matrix", "monokai":
	default:
		return fmt.Errorf("invalid theme")
	}
	if c.FontSize < 10 || c.FontSize > 24 {
		return fmt.Errorf("font size must be between 10 and 24")
	}
	if c.DefaultShell != "" && !filepath.IsAbs(c.DefaultShell) {
		return fmt.Errorf("default shell must be absolute")
	}
	if len(c.SSHProfiles) > 100 || len(c.Workspace) > 64 {
		return fmt.Errorf("configuration limit exceeded")
	}
	names := map[string]bool{}
	for _, p := range c.SSHProfiles {
		if err := validateSSH(p); err != nil {
			return err
		}
		if names[p.Name] {
			return fmt.Errorf("duplicate SSH profile name")
		}
		names[p.Name] = true
	}
	count := 0
	var validateNode func(WorkspaceNode, int) error
	validateNode = func(n WorkspaceNode, depth int) error {
		if depth > 16 {
			return fmt.Errorf("workspace nesting exceeds 16")
		}
		switch n.Type {
		case "terminal":
			count++
			if count > 64 {
				return fmt.Errorf("workspace exceeds 64 terminals")
			}
			if n.Shell != "" && !filepath.IsAbs(n.Shell) {
				return fmt.Errorf("workspace shell must be absolute")
			}
			if n.Cwd != "" && !filepath.IsAbs(n.Cwd) {
				return fmt.Errorf("workspace directory must be absolute")
			}
			if n.SSH != nil {
				return validateSSH(*n.SSH)
			}
		case "split":
			if n.Orientation != "horizontal" && n.Orientation != "vertical" {
				return fmt.Errorf("invalid split orientation")
			}
			if len(n.Children) < 2 || len(n.Children) > 64 || len(n.Sizes) != len(n.Children) {
				return fmt.Errorf("invalid split children/sizes")
			}
			total := 0.0
			for _, size := range n.Sizes {
				if size <= 0 || size > 100 {
					return fmt.Errorf("invalid split size")
				}
				total += size
			}
			if math.Abs(total-100) > 0.01 {
				return fmt.Errorf("split sizes must total 100 percent")
			}
			for _, child := range n.Children {
				if err := validateNode(child, depth+1); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("invalid workspace node type")
		}
		return nil
	}
	for _, tab := range c.Workspace {
		if len(tab.Name) > 256 {
			return fmt.Errorf("tab name too long")
		}
		if err := validateNode(tab.Root, 0); err != nil {
			return err
		}
	}
	return nil
}
func (s *ConfigService) LoadConfig() ConfigResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	fallback := func(err error) ConfigResult { return ConfigResult{Config: defaultConfig(), Warning: err.Error()} }
	path, err := s.configPath()
	if err != nil {
		return fallback(err)
	}
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return ConfigResult{Config: defaultConfig()}
	}
	if err != nil {
		return fallback(err)
	}
	defer file.Close()
	var config Config
	decoder := json.NewDecoder(io.LimitReader(file, 1024*1024+1))
	if err := decoder.Decode(&config); err != nil {
		return fallback(fmt.Errorf("configuration could not be read: %w", err))
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		return fallback(fmt.Errorf("configuration contains trailing data"))
	}
	if err := validateConfig(config); err != nil {
		return fallback(err)
	}
	if config.Workspace == nil {
		config.Workspace = []WorkspaceTab{}
	}
	if config.SSHProfiles == nil {
		config.SSHProfiles = []SSHProfile{}
	}
	return ConfigResult{Config: config}
}
func (s *ConfigService) SaveConfig(config Config) error {
	if err := validateConfig(config); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	if len(data) > 1024*1024 {
		return fmt.Errorf("configuration exceeds 1MB")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path, err := s.configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if err := os.Chmod(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".config-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(file.Name(), path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
