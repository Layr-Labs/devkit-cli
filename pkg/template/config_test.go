package template

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test template URL lookup
	mainBaseURL, mainVersion, err := GetTemplateURLs(config, "task", "go")
	if err != nil {
		t.Fatalf("Failed to get template URLs: %v", err)
	}

	expectedBaseURL := "https://github.com/Layr-Labs/hourglass-avs-template"
	expectedVersion := "95f9067bd35b770e989d7d6442003e405d7639ae"

	if mainBaseURL != expectedBaseURL {
		t.Errorf("Unexpected main template base URL: got %s, want %s", mainBaseURL, expectedBaseURL)
	}

	if mainVersion != expectedVersion {
		t.Errorf("Unexpected main template version: got %s, want %s", mainVersion, expectedVersion)
	}

	// Test non-existent architecture
	mainBaseURL, mainVersion, err = GetTemplateURLs(config, "nonexistent", "go")
	if err != nil {
		t.Fatalf("Failed to get template URLs: %v", err)
	}
	if mainBaseURL != "" {
		t.Errorf("Expected empty URL for nonexistent architecture, got %s", mainBaseURL)
	}
	if mainVersion != "" {
		t.Errorf("Expected empty version for nonexistent architecture, got %s", mainVersion)
	}
}
