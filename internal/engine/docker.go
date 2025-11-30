package engine

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/peaberberian/paul-envs/internal/files"
)

// Implements `ContainerEngine` for docker compose
type DockerEngine struct{}

func newDocker(ctx context.Context) (*DockerEngine, error) {
	cmd := exec.CommandContext(ctx, "docker", "compose", "version")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker compose command not found: %w", err)
	}
	return &DockerEngine{}, nil
}

// TODO: integrate into corresponding command?
func (c *DockerEngine) CheckPermissions(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "ps")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()

		// TODO: better placed management
		if strings.Contains(stderrStr, "permission denied") ||
			strings.Contains(stderrStr, "access denied") ||
			strings.Contains(stderrStr, "dial unix") && strings.Contains(stderrStr, "connect: permission denied") {
			return fmt.Errorf("permission denied. Please run with elevated privileges:\n\n%s", getSudoCommand())
		}
		return fmt.Errorf("failed to connect to Docker: %w\n%s", err, stderrStr)
	}
	return nil
}

func (c *DockerEngine) BuildContainer(ctx context.Context, project files.ProjectEntry, relativeDotfilesDir string) error {
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", project.ComposeFilePath, "--env-file", project.EnvFilePath, "build")
	envVars := append(os.Environ(),
		"COMPOSE_PROJECT_NAME=paulenv-"+project.ProjectName,
		"DOTFILES_DIR="+relativeDotfilesDir,
	)
	cmd.Env = envVars
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c *DockerEngine) RunContainer(ctx context.Context, project files.ProjectEntry, args []string) error {
	cmdArgs := []string{"compose", "-f", project.ComposeFilePath, "--env-file", project.EnvFilePath, "run", "--rm", "paulenv"}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	cmd.Env = append(os.Environ(), "COMPOSE_PROJECT_NAME=paulenv-"+project.ProjectName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Run failed: %w", err)
	}

	return nil
}

func (c *DockerEngine) Info(ctx context.Context) (ContainerInfo, error) {
	cmd := exec.CommandContext(ctx, "docker", "--version")
	output, err := cmd.Output()
	if err != nil {
		return ContainerInfo{}, fmt.Errorf("Failed to obtain docker version: %w", err)
	}
	version := strings.TrimSpace(fmt.Sprintf("%s", output))
	return ContainerInfo{Version: version, Name: "docker"}, nil
}

func (c *DockerEngine) CreateVolume(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "docker", "volume", "create", name)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to create shared volume: %w.", err)
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
