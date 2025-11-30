package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

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

func (c *DockerEngine) BuildContainer(ctx context.Context, project files.ProjectEntry, relativeDotfilesDir string) error {
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", project.ComposeFilePath, "--env-file", project.EnvFilePath, "build")
	envVars := append(os.Environ(),
		"COMPOSE_PROJECT_NAME=paulenv-"+project.ProjectName,
		"DOTFILES_DIR="+relativeDotfilesDir,
	)
	cmd.Env = envVars
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("Build failed: %w", err)
	}
	return nil
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
	}
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("Run failed: %w", err)
	}
	return nil
}

func (c *DockerEngine) HasBeenBuilt(ctx context.Context, projectName string) (bool, error) {
	imageName := fmt.Sprintf("paulenv:%s", projectName)
	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", imageName)
	err := cmd.Run()

	if err != nil {
		if err := cmd.Run(); err != nil {
			if pErr := c.checkPermissions(ctx); pErr != nil {
				return false, pErr
			}
		}
		// Check if it's a "not found" error vs other error
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				// Image doesn't exist
				return false, nil
			}
		}
		// Other error
		return false, err
	}

	return true, nil
}

func (c *DockerEngine) Info(ctx context.Context) (EngineInfo, error) {
	cmd := exec.CommandContext(ctx, "docker", "--version")
	output, err := cmd.Output()
	if err != nil {
		return EngineInfo{}, fmt.Errorf("Failed to obtain docker version: %w", err)
	}
	version := strings.TrimSpace(fmt.Sprintf("%s", output))
	return EngineInfo{Version: version, Name: "docker"}, nil
}

func (c *DockerEngine) CreateVolume(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "docker", "volume", "create", name)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("Failed to create shared volume: %w.", err)
	}
	return nil
}

func (c *DockerEngine) GetImageInfo(ctx context.Context, projectName string) (*ImageInfo, error) {
	imageName := fmt.Sprintf("paulenv:%s", projectName)
	info := &ImageInfo{ImageName: imageName}

	// Check if image exists and get build time
	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", imageName, "--format", "{{.Created}}")
	output, err := cmd.Output()
	if err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return nil, pErr
		} else if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return info, nil
		}
		return nil, err
	}
	if buildTime, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(output))); err == nil {
		info.BuiltAt = &buildTime
	}
	return info, nil
}

func (c *DockerEngine) checkPermissions(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "ps")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()

		// TODO: better placed management
		if strings.Contains(stderrStr, "permission denied") ||
			strings.Contains(stderrStr, "access denied") ||
			strings.Contains(stderrStr, "dial unix") && strings.Contains(stderrStr, "connect: permission denied") {
			return errors.New("permission denied. Please run with elevated privileges")
		}
		return fmt.Errorf("failed to connect to Docker: %w\n%s", err, stderrStr)
	}
	return nil
}
