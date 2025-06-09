package common

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSaveUserIdAndLoadGlobalSettings(t *testing.T) {
	// Set HOME to a temp directory
	tmp := t.TempDir()
	os.Setenv("HOME", tmp)

	const id1 = "uuid-1234"
	// Save first UUID
	if err := SaveUserId(id1); err != nil {
		t.Fatalf("SaveUserId failed: %v", err)
	}

	// Path where config should be
	cfg := filepath.Join(tmp, ".devkit", GlobalConfigFile)

	// Check file exists
	if _, err := os.Stat(cfg); err != nil {
		t.Fatalf("config file not found: %v", err)
	}

	// Load and verify content
	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	var s struct {
		UserUUID string `yaml:"user_uuid"`
	}
	if err := yaml.Unmarshal(data, &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if s.UserUUID != id1 {
		t.Errorf("expected %s, got %s", id1, s.UserUUID)
	}

	// Save a new UUID: since existing settings loads fine, code preserves the old UUID
	const id2 = "uuid-5678"
	if err := SaveUserId(id2); err != nil {
		t.Fatalf("SaveUserId overwrite failed: %v", err)
	}
	// Reload
	out, err := LoadGlobalSettings()
	if err != nil {
		t.Fatalf("LoadGlobalSettings failed: %v", err)
	}
	if out.UserUUID != id1 {
		t.Errorf("expected preserved %s after overwrite attempt, got %s", id1, out.UserUUID)
	}
}

func TestGetUserUUIDFromGlobalSettings_Empty(t *testing.T) {
	// Unset HOME
	os.Unsetenv("HOME")

	// Ensure no config and HOME unset
	uuid := getUserUUIDFromGlobalSettings()
	if uuid != "" {
		t.Errorf("expected empty UUID when HOME unset, got %q", uuid)
	}
}

func TestLoadGlobalSettings_MalformedYAML(t *testing.T) {
	// Set HOME to temp
	tmp := t.TempDir()
	os.Setenv("HOME", tmp)

	// Create config dir and invalid YAML
	d := filepath.Join(tmp, ".devkit")
	os.MkdirAll(d, 0755)
	cfg := filepath.Join(d, GlobalConfigFile)
	if err := os.WriteFile(cfg, []byte("not: [valid: yaml"), 0644); err != nil {
		t.Fatalf("write malformed YAML: %v", err)
	}

	// Load should error
	if _, err := LoadGlobalSettings(); err == nil {
		t.Error("expected error loading malformed YAML, got nil")
	}

	// SaveUserId should overwrite malformed and succeed
	const id = "uuid-0000"
	if err := SaveUserId(id); err != nil {
		t.Fatalf("SaveUserId did not overwrite malformed YAML: %v", err)
	}
	// Verify valid YAML now
	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("read config after overwrite: %v", err)
	}
	var s struct {
		UserUUID string `yaml:"user_uuid"`
	}
	if err := yaml.Unmarshal(data, &s); err != nil {
		t.Fatalf("unmarshal after overwrite: %v", err)
	}
	if s.UserUUID != id {
		t.Errorf("expected %s after overwrite, got %s", id, s.UserUUID)
	}
}

func TestSaveUserId_PermissionsError(t *testing.T) {
	// Set HOME to temp
	tmp := t.TempDir()
	os.Setenv("HOME", tmp)

	// Create file where directory should be to block MkdirAll
	block := filepath.Join(tmp, ".devkit")
	if err := os.WriteFile(block, []byte(""), 0644); err != nil {
		t.Fatalf("setup block file: %v", err)
	}

	// Now SaveUserId should fail on MkdirAll
	if err := SaveUserId("any"); err == nil {
		t.Error("expected error when MkdirAll fails, got nil")
	}
}

func TestSaveUserId_WriteFileError(t *testing.T) {
	// Set HOME to temp
	tmp := t.TempDir()
	os.Setenv("HOME", tmp)

	// Create directory and make it read-only
	d := filepath.Join(tmp, ".devkit")
	if err := os.MkdirAll(d, 0555); err != nil {
		t.Fatalf("setup readonly dir: %v", err)
	}

	// Attempt write should fail
	if err := SaveUserId("uuid-error"); err == nil {
		t.Error("expected write error due to permissions, got nil")
	}
}
