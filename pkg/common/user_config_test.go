package common

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSaveUserIdAndLoadGlobalConfig(t *testing.T) {
	// Set XDG_CONFIG_HOME to a temp directory
	tmp := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmp)

	// Get the global config dir so that we can create it
	globalConfigDir, err := GetGlobalConfigDir()
	if err != nil {
		t.Fatalf("could not locate global config: %v", err)
	}

	const id1 = "uuid-1234"
	// Save first UUID
	if err := SaveUserId(id1); err != nil {
		t.Fatalf("SaveUserId failed: %v", err)
	}

	// Path where config should be
	cfg := filepath.Join(globalConfigDir, GlobalConfigFile)

	// Check file exists
	if _, err := os.Stat(cfg); err != nil {
		t.Fatalf("config file not found: %v", err)
	}

	// Load and verify content
	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	fmt.Printf("what %s", string(data))
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
	out, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("LoadGlobalConfig failed: %v", err)
	}
	if out.UserUUID != id1 {
		t.Errorf("expected preserved %s after overwrite attempt, got %s", id1, out.UserUUID)
	}
}

func TestGetUserUUIDFromGlobalConfig_Empty(t *testing.T) {
	// Save original environment
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	originalHOME := os.Getenv("HOME")

	// Clean up after test
	defer func() {
		if originalXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
		if originalHOME != "" {
			os.Setenv("HOME", originalHOME)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	// Unset XDG_CONFIG_HOME and set HOME to a temp directory with no config
	os.Unsetenv("XDG_CONFIG_HOME")
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)

	// Now getUserUUIDFromGlobalConfig should return empty string since no config exists
	uuid := getUserUUIDFromGlobalConfig()
	if uuid != "" {
		t.Errorf("expected empty UUID when no config exists, got %q", uuid)
	}
}

func TestLoadGlobalConfig_MalformedYAML(t *testing.T) {
	// Set XDG_CONFIG_HOME to temp
	tmp := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmp)

	// Get the global config dir so that we can create it
	globalConfigDir, err := GetGlobalConfigDir()
	if err != nil {
		t.Fatalf("could not locate global config: %v", err)
	}

	// Create config dir and invalid YAML
	if err := os.MkdirAll(globalConfigDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	cfg := filepath.Join(globalConfigDir, GlobalConfigFile)
	if err := os.WriteFile(cfg, []byte("not: [valid: yaml"), 0644); err != nil {
		t.Fatalf("write malformed YAML: %v", err)
	}

	// Load should error
	if _, err := LoadGlobalConfig(); err == nil {
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
	// Set XDG_CONFIG_HOME to temp
	tmp := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmp)

	// Get the global config dir so that we can create it
	globalConfigDir, err := GetGlobalConfigDir()
	if err != nil {
		t.Fatalf("could not locate global config: %v", err)
	}

	// Create file where directory should be to block MkdirAll
	if err := os.WriteFile(globalConfigDir, []byte(""), 0644); err != nil {
		t.Fatalf("setup block file: %v", err)
	}

	// Now SaveUserId should fail on MkdirAll
	if err := SaveUserId("any"); err == nil {
		t.Error("expected error when MkdirAll fails, got nil")
	}
}

func TestSaveUserId_WriteFileError(t *testing.T) {
	// Set XDG_CONFIG_HOME to temp
	tmp := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmp)

	// Get the global config dir so that we can create it
	globalConfigDir, err := GetGlobalConfigDir()
	if err != nil {
		t.Fatalf("could not locate global config: %v", err)
	}

	// Create directory and make it read-only
	if err := os.MkdirAll(globalConfigDir, 0555); err != nil {
		t.Fatalf("setup readonly dir: %v", err)
	}

	// Attempt write should fail
	if err := SaveUserId("uuid-error"); err == nil {
		t.Error("expected write error due to permissions, got nil")
	}
}
