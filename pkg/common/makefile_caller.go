package common

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// CallDevkitMakeTarget runs a `make <target>` using Makefile.Devkit with context.
func CallDevkitMakeTarget(ctx context.Context, target string, args ...string) error {
	cmdArgs := append([]string{"-f", Makefile, target}, args...)
	cmd := exec.CommandContext(ctx, "make", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("make %s failed: %w", target, err)
	}
	return nil
}
