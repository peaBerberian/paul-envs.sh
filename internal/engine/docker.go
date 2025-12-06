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
func (c *DockerEngine) JoinContainer(ctx context.Context, containerInfo ContainerInfo, args []string) error {
	cmdArgs := []string{"exec", "-it", containerInfo.ContainerId, "/usr/local/bin/entrypoint.sh"}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("join exited: %w", err)
	}
	return nil
}

func (c *DockerEngine) HasBeenBuilt(ctx context.Context, projectName string) (bool, error) {
	imageName := fmt.Sprintf("paulenv:%s", projectName)
	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", imageName)
	err := cmd.Run()

	if err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return false, pErr
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
	info := &ImageInfo{ImageName: imageName, ProjectName: &projectName}

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
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "name=paulenv-", "--format", "{{.ID}}\t{{.Image}}\t{{.Names}}")
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
			parts := strings.SplitN(s, "\t", 3)
			id := parts[0]
			var image *string
			var name *string
			var projectName *string

			if len(parts) > 1 {
				image = &parts[1]
			}
			if len(parts) > 2 {
				name = &parts[2]
			}

			if image != nil && strings.HasPrefix(*image, "paulenv:") && len(*image) > len("paulenv:") {
				sliced := (*image)[len("paulenv:"):]
				projectName = &sliced
			}

			result = append(result, ContainerInfo{
				ProjectName:   projectName,
				ContainerName: name,
				ContainerId:   id,
				ImageName:     image,
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

// List volumes currently known by this container engine
func (c *DockerEngine) ListVolumes(ctx context.Context) ([]VolumeInfo, error) {
	cmd := exec.CommandContext(ctx, "docker", "volume", "ls", "--filter", "name=paulenv-", "--format", "{{.Name}}")
	output, err := cmd.Output()
	if err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return []VolumeInfo{}, pErr
		}
		return []VolumeInfo{}, fmt.Errorf("failed to list volumes: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	result := make([]VolumeInfo, 0, len(lines))
	for _, volumeName := range lines {
		if volumeName != "" {
			result = append(result, VolumeInfo{
				VolumeId:   volumeName,
				VolumeName: volumeName,
			})
		}
	}
	return result, nil
}

func (c *DockerEngine) RemoveVolume(ctx context.Context, volume VolumeInfo) error {
	cmd := exec.CommandContext(ctx, "docker", "volume", "rm", volume.VolumeName)
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("failed to remove volume %s: %w", volume.VolumeName, err)
	}
	return nil
}

// List networks currently known by this container engine
func (c *DockerEngine) ListNetworks(ctx context.Context) ([]NetworkInfo, error) {
	cmd := exec.CommandContext(ctx, "docker", "network", "ls", "--filter", "name=paulenv-", "--format", "{{.ID}}\t{{.Name}}")
	output, err := cmd.Output()
	if err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return []NetworkInfo{}, pErr
		}
		return []NetworkInfo{}, fmt.Errorf("failed to list networks: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	result := make([]NetworkInfo, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) >= 2 {
				networkId := parts[0]
				networkName := parts[1]
				var projectName *string

				// Extract project name from network name if it follows the pattern "paulenv-{project}_default"
				if strings.HasPrefix(networkName, "paulenv-") {
					// Remove "paulenv-" prefix
					withoutPrefix := networkName[len("paulenv-"):]
					// Remove "_default" suffix if present
					if strings.HasSuffix(withoutPrefix, "_default") {
						sliced := withoutPrefix[:len(withoutPrefix)-len("_default")]
						projectName = &sliced
					} else {
						projectName = &withoutPrefix
					}
				}

				result = append(result, NetworkInfo{
					NetworkId:   networkId,
					NetworkName: networkName,
					ProjectName: projectName,
				})
			}
		}
	}
	return result, nil
}

// Remove network listed from this container engine
func (c *DockerEngine) RemoveNetwork(ctx context.Context, network NetworkInfo) error {
	cmd := exec.CommandContext(ctx, "docker", "network", "rm", network.NetworkId)
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("failed to remove network %s: %w", network.NetworkName, err)
	}
	return nil
}

// Remove the `ContainerEngine`'s build cache from metadata linked to this
// executable
func (c *DockerEngine) PruneBuildCache(ctx context.Context) error {
	// Prune build cache for paulenv images specifically
	cmd := exec.CommandContext(ctx, "docker", "builder", "prune", "-f", "--filter", "label=paulenv=true", "-f")
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("failed to prun build cache: %w", err)
	}
	return nil
}

// List images currently known by this container engine
func (c *DockerEngine) ListImages(ctx context.Context) ([]ImageInfo, error) {
	cmd := exec.CommandContext(ctx, "docker", "images", "--filter", "reference=paulenv:*", "--format", "{{.Repository}}:{{.Tag}}\t{{.CreatedAt}}")
	output, err := cmd.Output()
	if err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return []ImageInfo{}, pErr
		}
		return []ImageInfo{}, fmt.Errorf("failed to list images: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	result := make([]ImageInfo, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			parts := strings.SplitN(line, "\t", 2)
			imageName := parts[0]
			var projectName *string
			var builtAt *time.Time

			// Extract project name from image name if it follows the pattern "paulenv:{project}"
			if strings.HasPrefix(imageName, "paulenv:") && len(imageName) > len("paulenv:") {
				sliced := imageName[len("paulenv:"):]
				projectName = &sliced
			}

			// Parse build time if available
			if len(parts) > 1 {
				timeStr := strings.TrimSpace(parts[1])
				// Docker's CreatedAt format can vary, try common formats
				formats := []string{
					"2006-01-02 15:04:05 -0700 MST",
					time.RFC3339,
					time.RFC3339Nano,
				}
				for _, format := range formats {
					if parsedTime, err := time.Parse(format, timeStr); err == nil {
						builtAt = &parsedTime
						break
					}
				}
			}

			result = append(result, ImageInfo{
				ImageName:   imageName,
				ProjectName: projectName,
				BuiltAt:     builtAt,
			})
		}
	}
	return result, nil
}

// Remove image from this container engine
func (c *DockerEngine) RemoveImage(ctx context.Context, image ImageInfo) error {
	cmd := exec.CommandContext(ctx, "docker", "rmi", "-f", image.ImageName)
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("failed to remove image %s: %w", image.ImageName, err)
	}
	return nil
}
