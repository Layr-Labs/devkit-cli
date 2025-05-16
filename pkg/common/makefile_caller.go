package common

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// CallMakeTarget runs a `make <target>` using Makefile with context.
func CallMakefileTarget(ctx context.Context, dir string, makefile string, target string, args ...string) error {
	cmdArgs := append([]string{"-f", makefile, target}, args...)
	cmd := exec.CommandContext(ctx, "make", cmdArgs...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("make %s failed: %w", target, err)
	}
	return nil
}
