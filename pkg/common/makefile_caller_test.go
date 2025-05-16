package common

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCallMakefileTarget(t *testing.T) {
	tmpDir := t.TempDir()
	makefilePath := filepath.Join(tmpDir, Makefile)

	makefile := `
print:
	echo "Hello, test"
fail:
	exit 1

` // Write file to tmpDir
	if err := os.WriteFile(makefilePath, []byte(makefile), 0644); err != nil {
		t.Fatalf("failed to write Makefile: %v", err)
	}

	// Test success
	if err := CallMakefileTarget(context.Background(), tmpDir, Makefile, "print"); err != nil {
		t.Errorf("expected success, got error: %v", err)
	}

	// Test failure
	err := CallMakefileTarget(context.Background(), tmpDir, Makefile, "fail")
	if err == nil {
		t.Error("expected error, got nil")
	}
}
