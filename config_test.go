package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfigRoundTripAndPermissions(t *testing.T) {
	service := &ConfigService{path: filepath.Join(t.TempDir(), "private", "config.json")}
	c := defaultConfig()
	c.Theme = "dracula"
	c.FontSize = 18
	c.Workspace = []WorkspaceTab{{Name: "Build", Root: WorkspaceNode{Type: "terminal", Shell: "/bin/sh", Cwd: "/tmp"}}}
	if err := service.SaveConfig(c); err != nil {
		t.Fatal(err)
	}
	result := service.LoadConfig()
	if result.Warning != "" || !reflect.DeepEqual(c, result.Config) {
		t.Fatalf("round trip: %+v", result)
	}
	info, _ := os.Stat(service.path)
	if info.Mode().Perm() != 0600 {
		t.Fatalf("file permissions = %o", info.Mode().Perm())
	}
	info, _ = os.Stat(filepath.Dir(service.path))
	if info.Mode().Perm() != 0700 {
		t.Fatalf("directory permissions = %o", info.Mode().Perm())
	}
	c.FontSize = 20
	if err := service.SaveConfig(c); err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob(filepath.Join(filepath.Dir(service.path), ".config-*"))
	if len(files) != 0 {
		t.Fatal("temporary files remain")
	}
}
func TestConfigFallbackAndInvalidSavePreservesFile(t *testing.T) {
	service := &ConfigService{path: filepath.Join(t.TempDir(), "config.json")}
	if result := service.LoadConfig(); result.Warning != "" || result.Config.FontSize != 14 {
		t.Fatal(result)
	}
	for _, raw := range []string{"{", "{}", `{"version":99}`, `{} {}`, `{"version":1,"theme":"dracula","fontSize":999}`} {
		if err := os.WriteFile(service.path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		result := service.LoadConfig()
		if result.Warning == "" || result.Config.FontSize != 14 {
			t.Fatalf("no safe fallback for %s", raw)
		}
	}
	valid := defaultConfig()
	if _, err := service.RecoverConfig(valid); err != nil {
		t.Fatal(err)
	}
	bad := valid
	bad.Workspace = []WorkspaceTab{{Root: WorkspaceNode{Type: "split", Children: []WorkspaceNode{}}}}
	if err := service.SaveConfig(bad); err == nil {
		t.Fatal("accepted invalid workspace")
	}
	if result := service.LoadConfig(); result.Warning != "" {
		t.Fatal("invalid save damaged configuration")
	}
}
func TestSSHArgumentValidation(t *testing.T) {
	p := SSHProfile{Name: "dev", Host: "example.org", User: "alice", Port: 2222, KeyPath: "/tmp/key with spaces", JumpHost: "jump@bastion:22"}
	if err := validateSSH(p); err != nil {
		t.Fatal(err)
	}
	want := []string{"-p", "2222", "-l", "alice", "-i", "/tmp/key with spaces", "-J", "jump@bastion:22", "--", "example.org"}
	if got := sshArgs(p); !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %q", got)
	}
	for _, host := range []string{"-oProxyCommand=id", "bad host", "user@host", "host\ncommand"} {
		p.Host = host
		if validateSSH(p) == nil {
			t.Errorf("accepted %q", host)
		}
	}
}

func TestInvalidConfigurationRequiresExplicitRecovery(t *testing.T) {
	for _, raw := range []string{"{ broken", `{"version":99,"future":"preserve me"}`} {
		service := &ConfigService{path: filepath.Join(t.TempDir(), "config.json")}
		if err := os.WriteFile(service.path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if result := service.LoadConfig(); !result.RecoveryRequired {
			t.Fatal("invalid configuration did not require recovery")
		}
		if err := service.SaveConfig(defaultConfig()); err == nil {
			t.Fatal("autosave overwrote original")
		}
		original, _ := os.ReadFile(service.path)
		if string(original) != raw {
			t.Fatal("original was changed")
		}
		backup, err := service.RecoverConfig(defaultConfig())
		if err != nil {
			t.Fatal(err)
		}
		saved, _ := os.ReadFile(backup)
		if string(saved) != raw {
			t.Fatal("backup did not preserve original bytes")
		}
		if result := service.LoadConfig(); result.RecoveryRequired || result.Warning != "" {
			t.Fatal(result)
		}
	}
}
func TestLegacyProfileMigrationAndPreferences(t *testing.T) {
	service := &ConfigService{path: filepath.Join(t.TempDir(), "config.json")}
	legacy := `{"version":1,"theme":"dracula","fontSize":14,"sshProfiles":[{"name":"build","host":"example.org","port":22}],"workspace":[{"name":"SSH","root":{"type":"terminal","ssh":{"name":"build","host":"example.org","port":22}}}]}`
	if err := os.WriteFile(service.path, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	result := service.LoadConfig()
	if result.Warning != "" {
		t.Fatal(result.Warning)
	}
	id := result.Config.SSHProfiles[0].ID
	if id == "" || result.Config.Workspace[0].Root.SSHProfileID != id {
		t.Fatal("legacy profile was not linked")
	}
	result.Config.SSHProfiles[0].Name = "renamed"
	result.Config.Shortcuts["new"] = "Ctrl+Alt+N"
	if err := service.SaveConfig(result.Config); err != nil {
		t.Fatal(err)
	}
	again := service.LoadConfig()
	if again.Config.SSHProfiles[0].ID != id || again.Config.Shortcuts["new"] != "Ctrl+Alt+N" {
		t.Fatal("identity or shortcut lost")
	}
}

func TestActionPreferenceValidation(t *testing.T) {
	config := defaultConfig()
	config.Shortcuts["new"] = "Ctrl+Alt+N"
	config.Shortcuts["close"] = ""
	config.Shortcuts["search"] = ""
	if err := validateConfig(config); err != nil {
		t.Fatal(err)
	}
	config.Shortcuts["new"] = "Ctrl+Shift+C"
	if validateConfig(config) == nil {
		t.Fatal("clipboard binding was overridden")
	}
	config = defaultConfig()
	config.Shortcuts["new"] = config.Shortcuts["close"]
	if validateConfig(config) == nil {
		t.Fatal("duplicate binding accepted")
	}
	config = defaultConfig()
	config.Snippets = append(config.Snippets, config.Snippets[0])
	if validateConfig(config) == nil {
		t.Fatal("duplicate snippet identity accepted")
	}
}
