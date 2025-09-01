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
	mainBaseURL, mainVersion, err := GetTemplateURLs(config, "hourglass", "go")
	if err != nil {
		t.Fatalf("Failed to get template URLs: %v", err)
	}

	expectedBaseURL := "https://github.com/Layr-Labs/hourglass-avs-template"
	expectedVersion := "v0.0.24"

	if mainBaseURL != expectedBaseURL {
		t.Errorf("Unexpected main template base URL: got %s, want %s", mainBaseURL, expectedBaseURL)
	}

	if mainVersion != expectedVersion {
		t.Errorf("Unexpected main template version: got %s, want %s", mainVersion, expectedVersion)
	}

	// Test non-existent architecture
	mainBaseURL, mainVersion, err = GetTemplateURLs(config, "nonexistent", "go")
	if err == nil {
		t.Fatalf("Expected to fail to get template URLs: %v", err)
	}
	if mainBaseURL != "" {
		t.Errorf("Expected empty URL for nonexistent architecture, got %s", mainBaseURL)
	}
	if mainVersion != "" {
		t.Errorf("Expected empty version for nonexistent architecture, got %s", mainVersion)
	}

	// Test non-existent language
	mainBaseURL, mainVersion, err = GetTemplateURLs(config, "hourglass", "nonexistent")
	if err == nil {
		t.Fatalf("Expected to fail to get template URLs: %v", err)
	}
	if mainBaseURL != "" {
		t.Errorf("Expected empty URL for nonexistent language, got %s", mainBaseURL)
	}
	if mainVersion != "" {
		t.Errorf("Expected empty version for nonexistent language, got %s", mainVersion)
	}
}
