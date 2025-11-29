package commands

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func checkDockerPermissions(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "ps")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if strings.Contains(stderrStr, "permission denied") ||
			strings.Contains(stderrStr, "access denied") ||
			strings.Contains(stderrStr, "dial unix") && strings.Contains(stderrStr, "connect: permission denied") {
			return fmt.Errorf("Permission denied. Please run with elevated privileges:\n\n%s", getSudoCommand())
		}
		return fmt.Errorf("failed to connect to Docker: %w\n%s", err, stderrStr)
	}
	return nil
}

func getSudoCommand() string {
	executable, err := os.Executable()
	if err != nil {
		executable = os.Args[0]
	}
	args := strings.Join(os.Args[1:], " ")
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("  Run PowerShell or Command Prompt as Administrator, then:\n  %s %s", executable, args)
	}
	return fmt.Sprintf("  sudo %s %s", executable, args)
}

func checkDockerComposeInstallation(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "compose", "version")
	return cmd.Run()
}
