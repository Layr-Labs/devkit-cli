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
	mainBaseURL, mainVersion, contractsBaseURL, contractsVersion, err := GetTemplateURLs(config, "task", "go")
	if err != nil {
		t.Fatalf("Failed to get template URLs: %v", err)
	}

	expectedBaseURL := "https://github.com/Layr-Labs/hourglass-avs-template"
	expectedVersion := "v0.0.4"

	if mainBaseURL != expectedBaseURL {
		t.Errorf("Unexpected main template base URL: got %s, want %s", mainBaseURL, expectedBaseURL)
	}

	if mainVersion != expectedVersion {
		t.Errorf("Unexpected main template version: got %s, want %s", mainVersion, expectedVersion)
	}

	expectedContractsBaseURL := "https://github.com/Layr-Labs/hourglass-contracts-template"
	expectedContractsVersion := "main"

	if contractsBaseURL != expectedContractsBaseURL {
		t.Errorf("Unexpected contracts template base URL: got %s, want %s", contractsBaseURL, expectedContractsBaseURL)
	}

	if contractsVersion != expectedContractsVersion {
		t.Errorf("Unexpected contracts template version: got %s, want %s", contractsVersion, expectedContractsVersion)
	}

	// Test non-existent architecture
	mainBaseURL, mainVersion, _, _, err = GetTemplateURLs(config, "nonexistent", "go")
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
