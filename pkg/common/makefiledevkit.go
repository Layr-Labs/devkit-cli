package common

import (
	"os"
	"os/exec"
)

func MakefileDevkitRun() error {

	// Execute make run with Makefile.Devkit
	cmd := exec.Command("make", "-f", DevkitMakefile, "run")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func MakefileDevkitDeploy() error {

	// Execute make deploy with Makefile.Devkit
	cmd := exec.Command("make", "-f", DevkitMakefile, "deploy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
