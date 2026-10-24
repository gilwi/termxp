package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type SSHProfile struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name"`
	Host     string `json:"host"`
	User     string `json:"user"`
	Port     int    `json:"port"`
	KeyPath  string `json:"keyPath"`
	JumpHost string `json:"jumpHost"`
}

// Workspace descriptors deliberately contain no runtime IDs or terminal output.
type WorkspaceNode struct {
	Type         string          `json:"type"`
	Orientation  string          `json:"orientation,omitempty"`
	Children     []WorkspaceNode `json:"children,omitempty"`
	Sizes        []float64       `json:"sizes,omitempty"`
	Shell        string          `json:"shell,omitempty"`
	Cwd          string          `json:"cwd,omitempty"`
	SSHProfileID string          `json:"sshProfileId,omitempty"`
	SSH          *SSHProfile     `json:"ssh,omitempty"`
}
type WorkspaceTab struct {
	Name string        `json:"name"`
	Root WorkspaceNode `json:"root"`
}
type Snippet struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Command string `json:"command"`
}
type Config struct {
	Snippets         []Snippet         `json:"snippets"`
	Shortcuts        map[string]string `json:"shortcuts"`
	Version          int               `json:"version"`
	Theme            string            `json:"theme"`
	FontSize         int               `json:"fontSize"`
	DefaultShell     string            `json:"defaultShell"`
	CopyOnSelect     bool              `json:"copyOnSelect"`
	RestoreWorkspace bool              `json:"restoreWorkspace"`
	Workspace        []WorkspaceTab    `json:"workspace"`
	SSHProfiles      []SSHProfile      `json:"sshProfiles"`
}
type ConfigResult struct {
	Config           Config `json:"config"`
	Warning          string `json:"warning"`
	RecoveryRequired bool   `json:"recoveryRequired"`
}
type ConfigService struct {
	mu   sync.Mutex
	path string
}

func NewConfigService() *ConfigService { return &ConfigService{} }
func defaultConfig() Config {
	return Config{Snippets: defaultSnippets(), Shortcuts: defaultShortcuts(), Version: 1, Theme: "glassmorphic", FontSize: 14, RestoreWorkspace: true, Workspace: []WorkspaceTab{}, SSHProfiles: []SSHProfile{}}
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
	ids := map[string]bool{}
	if len(c.Snippets) > 100 || len(c.Shortcuts) > 32 {
		return fmt.Errorf("too many snippets or shortcuts")
	}
	for _, snippet := range c.Snippets {
		if snippet.ID == "" || ids[snippet.ID] || strings.TrimSpace(snippet.Label) == "" || len(snippet.Label) > 128 || len(snippet.Command) > 65536 {
			return fmt.Errorf("invalid or duplicate snippet")
		}
		ids[snippet.ID] = true
	}
	seenKeys := map[string]bool{}
	for action, key := range c.Shortcuts {
		if _, ok := defaultShortcuts()[action]; !ok {
			return fmt.Errorf("unknown shortcut action: %s", action)
		}
		if !validShortcut(key) || (key != "" && seenKeys[key]) {
			return fmt.Errorf("invalid or duplicate shortcut: %s", key)
		}
		seenKeys[key] = true
	}
	ids = map[string]bool{}
	for _, p := range c.SSHProfiles {
		if err := validateSSH(p); err != nil {
			return err
		}
		if names[p.Name] {
			return fmt.Errorf("duplicate SSH profile name")
		}
		names[p.Name] = true
		if p.ID == "" || len(p.ID) > 128 || ids[p.ID] {
			return fmt.Errorf("invalid or duplicate SSH profile ID")
		}
		ids[p.ID] = true
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
func normalizeConfig(c *Config) {
	if c.Snippets == nil {
		c.Snippets = defaultSnippets()
	}
	if c.Shortcuts == nil {
		c.Shortcuts = defaultShortcuts()
	}
	if c.Workspace == nil {
		c.Workspace = []WorkspaceTab{}
	}
	if c.SSHProfiles == nil {
		c.SSHProfiles = []SSHProfile{}
	}
	legacyProfiles := false
	for i := range c.SSHProfiles {
		p := &c.SSHProfiles[i]
		if p.ID == "" {
			legacyProfiles = true
			p.ID = fmt.Sprintf("legacy-%x", sha256.Sum256([]byte(p.Name)))
		}
	}
	var link func(*WorkspaceNode)
	link = func(n *WorkspaceNode) {
		if legacyProfiles && n.SSH != nil && n.SSHProfileID == "" {
			for _, p := range c.SSHProfiles {
				if p.Name == n.SSH.Name && p.Host == n.SSH.Host && p.User == n.SSH.User && p.Port == n.SSH.Port && p.KeyPath == n.SSH.KeyPath && p.JumpHost == n.SSH.JumpHost {
					n.SSHProfileID = p.ID
					n.SSH.ID = p.ID
					break
				}
			}
		}
		for i := range n.Children {
			link(&n.Children[i])
		}
	}
	for i := range c.Workspace {
		link(&c.Workspace[i].Root)
	}
}
func (s *ConfigService) readConfigLocked() (Config, error) {
	path, err := s.configPath()
	if err != nil {
		return Config{}, err
	}
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return defaultConfig(), nil
	}
	if err != nil {
		return Config{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return Config{}, err
	}
	if info.Size() > 1024*1024 {
		return Config{}, fmt.Errorf("configuration exceeds 1MB")
	}
	var config Config
	decoder := json.NewDecoder(io.LimitReader(file, 1024*1024+1))
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("configuration could not be read: %w", err)
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		return Config{}, fmt.Errorf("configuration contains trailing data")
	}
	normalizeConfig(&config)
	if err := validateConfig(config); err != nil {
		return Config{}, err
	}
	return config, nil
}
func (s *ConfigService) LoadConfig() ConfigResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	config, err := s.readConfigLocked()
	if err != nil {
		return ConfigResult{Config: defaultConfig(), Warning: err.Error(), RecoveryRequired: true}
	}
	return ConfigResult{Config: config}
}

// RecoverConfig explicitly preserves the original bytes before saving defaults
// or recovered preferences. Autosave cannot call this operation accidentally.
func (s *ConfigService) RecoverConfig(config Config) (string, error) {
	normalizeConfig(&config)
	if err := validateConfig(config); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path, err := s.configPath()
	if err != nil {
		return "", err
	}
	backup := ""
	if _, err := os.Stat(path); err == nil {
		original, err := os.Open(path)
		if err != nil {
			return "", err
		}
		defer original.Close()
		file, err := os.CreateTemp(filepath.Dir(path), "config-recovery-"+time.Now().Format("20060102-150405")+"-*.json")
		if err != nil {
			return "", err
		}
		backup = file.Name()
		if _, err := io.Copy(file, original); err != nil {
			file.Close()
			return "", err
		}
		if err := file.Sync(); err != nil {
			file.Close()
			return "", err
		}
		if err := file.Close(); err != nil {
			return "", err
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := s.saveConfigLocked(config); err != nil {
		return backup, err
	}
	return backup, nil
}
func (s *ConfigService) SaveConfig(config Config) error {
	normalizeConfig(&config)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.readConfigLocked(); err != nil {
		return fmt.Errorf("configuration recovery is required before saving: %w", err)
	}
	return s.saveConfigLocked(config)
}
func (s *ConfigService) saveConfigLocked(config Config) error {
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
