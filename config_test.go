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
	if err := service.SaveConfig(valid); err != nil {
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
