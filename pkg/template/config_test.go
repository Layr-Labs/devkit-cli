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
	url, _, err := GetTemplateURLs(config, "task", "go")
	if err != nil {
		t.Fatalf("Failed to get template URL: %v", err)
	}
	expected := "https://github.com/Layr-Labs/hourglass-avs-template/tree/pinned-may-16"
	if url != expected {
		t.Errorf("Unexpected template URL: got %s, want %s", url, expected)
	}

	// Test non-existent architecture
	url, _, err = GetTemplateURLs(config, "nonexistent", "go")
	if err != nil {
		t.Fatalf("Failed to get template URL: %v", err)
	}
	if url != "" {
		t.Errorf("Expected empty URL for nonexistent architecture, got %s", url)
	}
}
