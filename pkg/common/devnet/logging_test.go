package devnet

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func mockCommand(expectedOutput string, capture *[]string) func(string, ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		if capture != nil {
			*capture = append([]string{name}, args...)
		}
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = append(os.Environ(),
			"GO_WANT_HELPER_PROCESS=1",
			fmt.Sprintf("MOCK_OUTPUT=%s", expectedOutput),
		)
		return cmd
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	fmt.Fprint(os.Stdout, os.Getenv("MOCK_OUTPUT"))
	os.Exit(0)
}

func TestStreamLogsWithLabel_FindsContainer(t *testing.T) {
	var capturedArgs []string

	original := ExecCommand
	defer func() { ExecCommand = original }()

	step := 0
	ExecCommand = func(name string, args ...string) *exec.Cmd {
		if step == 0 {
			// Mock docker ps
			step++
			capturedArgs = append([]string{name}, args...)
			cs := []string{"-test.run=TestHelperProcess", "--", name}
			cs = append(cs, args...)
			cmd := exec.Command(os.Args[0], cs...)
			cmd.Env = append(os.Environ(),
				"GO_WANT_HELPER_PROCESS=1",
				"MOCK_OUTPUT=devkit-devnet",
			)
			return cmd
		} else {
			// Mock docker logs -f
			cs := []string{"-test.run=TestHelperProcess", "--", name}
			cs = append(cs, args...)
			cmd := exec.Command(os.Args[0], cs...)
			cmd.Env = append(os.Environ(),
				"GO_WANT_HELPER_PROCESS=1",
				"MOCK_OUTPUT=logs streaming...",
			)
			return cmd
		}
	}

	err := StreamLogsWithLabel("anvil")
	assert.NoError(t, err)

	fullCmd := strings.Join(capturedArgs, " ")
	assert.Contains(t, fullCmd, "docker ps")
	assert.Contains(t, fullCmd, "label=devkit.role=anvil")
}

func TestStreamLogsWithLabel_NotFound(t *testing.T) {
	original := ExecCommand
	defer func() { ExecCommand = original }()

	ExecCommand = mockCommand("", &[]string{})

	err := StreamLogsWithLabel("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no running container found")
}
