package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
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

func (c *DockerEngine) BuildImage(ctx context.Context, project files.ProjectEntry, relativeDotfilesDir string) error {
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
		return EngineInfo{}, fmt.Errorf("failed to obtain docker version: %w", err)
	}
	parsed := strings.TrimSpace(fmt.Sprintf("%s", output))
	re := regexp.MustCompile(`Docker version ([0-9]+\.[0-9]+\.[0-9]+)`)
	matches := re.FindStringSubmatch(string(parsed))
	if len(matches) > 1 {
		version := matches[1] // "24.0.7"
		return EngineInfo{Version: version, Name: "docker"}, nil
	}
	return EngineInfo{}, fmt.Errorf("failed to obtain docker version, unknown version format: %s", parsed)
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

func (c *DockerEngine) ListContainers(ctx context.Context) ([]ContainerInfo, error) {
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "name=paulenv-", "--format", "{{.ID}} {{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return []ContainerInfo{}, pErr
		}
		return []ContainerInfo{}, fmt.Errorf("failed to list containers: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	result := make([]ContainerInfo, 0, len(lines))
	for _, s := range lines {
		if s != "" {
			parts := strings.SplitN(s, " ", 2)
			id := parts[0]
			name := ""
			if len(parts) > 1 {
				name = parts[1]
			}

			result = append(result, ContainerInfo{
				ContainerName: name,
				ContainerId:   id,
			})
		}
	}
	return result, nil
}

func (c *DockerEngine) RemoveContainer(ctx context.Context, container ContainerInfo) error {
	cmd := exec.CommandContext(ctx, "docker", "rm", "-f", container.ContainerId)
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return err
	}
	return nil
}

func (c *DockerEngine) checkPermissions(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "ps")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if strings.Contains(stderrStr, "permission denied") ||
			strings.Contains(stderrStr, "access denied") ||
			strings.Contains(stderrStr, "dial unix") && strings.Contains(stderrStr, "connect: permission denied") {
			return errors.New("permission denied. Please run with elevated privileges")
		}
		return fmt.Errorf("failed to connect to Docker: %w\n%s", err, stderrStr)
	}
	return nil
}
