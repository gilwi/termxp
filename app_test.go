package main

import (
	"os"
	"testing"
)

func TestConfigLoadSave(t *testing.T) {
	// Create a new App instance
	app := NewApp()

	// Backup existing config if any
	configPath, err := app.GetConfigPath()
	if err != nil {
		t.Fatalf("Failed to get config path: %v", err)
	}

	backupPath := configPath + ".bak"
	if _, err := os.Stat(configPath); err == nil {
		if err := os.Rename(configPath, backupPath); err != nil {
			t.Fatalf("Failed to backup existing config: %v", err)
		}
		defer func() {
			// Restore backup
			_ = os.Remove(configPath)
			_ = os.Rename(backupPath, configPath)
		}()
	} else {
		defer func() {
			// Clean up config file created by test
			_ = os.Remove(configPath)
		}()
	}

	// 1. Load config when file doesn't exist (should return defaults)
	config, err := app.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if config["theme"] != "glassmorphic" {
		t.Errorf("Expected theme to be 'glassmorphic', got %v", config["theme"])
	}
	// In the default map, it should be int 14
	if val, ok := config["fontSize"].(int); !ok || val != 14 {
		t.Errorf("Expected fontSize to be 14 (int), got %v", config["fontSize"])
	}

	// 2. Save some settings
	newSettings := map[string]interface{}{
		"theme":    "cyberpunk",
		"fontSize": 16,
	}
	err = app.SaveConfig(newSettings)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// 3. Load settings back and check
	loaded, err := app.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if loaded["theme"] != "cyberpunk" {
		t.Errorf("Expected theme to be 'cyberpunk', got %v", loaded["theme"])
	}
	// Values from JSON unmarshal are float64 by default for numbers
	fontSizeVal, ok := loaded["fontSize"].(float64)
	if !ok || fontSizeVal != 16 {
		t.Errorf("Expected fontSize to be 16, got %v", loaded["fontSize"])
	}

	// 4. Save another setting to verify merging behavior
	err = app.SaveConfig(map[string]interface{}{
		"fontSize": 18,
		"someOther": "value",
	})
	if err != nil {
		t.Fatalf("SaveConfig merge failed: %v", err)
	}

	// 5. Load and verify merge
	merged, err := app.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig merge failed: %v", err)
	}
	if merged["theme"] != "cyberpunk" {
		t.Errorf("Expected theme to remain 'cyberpunk', got %v", merged["theme"])
	}
	fontSizeVal, ok = merged["fontSize"].(float64)
	if !ok || fontSizeVal != 18 {
		t.Errorf("Expected fontSize to be updated to 18, got %v", merged["fontSize"])
	}
	if merged["someOther"] != "value" {
		t.Errorf("Expected someOther to be 'value', got %v", merged["someOther"])
	}
}
