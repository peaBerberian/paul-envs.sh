package commands

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
)

// TODO: local data
func Clean(ctx context.Context, filestore *files.FileStore, console *console.Console) error {
	if err := checkDockerPermissions(ctx); err != nil {
		return err
	}
	if err := removeContainers(ctx); err != nil {
		return err
	}
	if err := removeImages(ctx); err != nil {
		return err
	}
	if err := removeVolumes(ctx); err != nil {
		return err
	}
	if err := removeNetworks(ctx); err != nil {
		return err
	}
	// TODO: recheck that one
	if shouldPruneCache() {
		if err := pruneBuildCache(ctx); err != nil {
			return err
		}
	} else {
		console.WriteLn("Skipping build cache removal")
	}

	console.WriteLn("Cleanup complete!")
	return nil
}

func removeContainers(ctx context.Context) error {
	fmt.Println("Stopping and removing containers...")

	// List containers
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "name=paulenv-", "--format", "{{.ID}} {{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		fmt.Println("  • No containers found")
		return nil
	}

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		id := parts[0]
		name := ""
		if len(parts) > 1 {
			name = parts[1]
		}

		fmt.Printf("  • Removing container: %s\n", name)
		cmd := exec.CommandContext(ctx, "docker", "rm", "-f", id)
		if err := cmd.Run(); err != nil {
			fmt.Printf("    ⚠️  Warning: failed to remove %s: %v\n", name, err)
		}
	}

	return nil
}

func removeImages(ctx context.Context) error {
	fmt.Println("Removing images...")

	cmd := exec.CommandContext(ctx, "docker", "images", "--filter", "reference=paulenv:*", "--format", "{{.ID}} {{.Repository}}:{{.Tag}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		fmt.Println("  • No images found")
		return nil
	}

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		id := parts[0]
		tag := ""
		if len(parts) > 1 {
			tag = parts[1]
		}

		fmt.Printf("  • Removing image: %s\n", tag)
		cmd := exec.CommandContext(ctx, "docker", "rmi", "-f", id)
		if err := cmd.Run(); err != nil {
			fmt.Printf("    ⚠️  Warning: failed to remove %s: %v\n", tag, err)
		}
	}

	return nil
}

func removeVolumes(ctx context.Context) error {
	fmt.Println("Removing volumes...")

	cmd := exec.CommandContext(ctx, "docker", "volume", "ls", "--filter", "name=paulenv-", "--format", "{{.Name}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list volumes: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		fmt.Println("  • No volumes found")
		return nil
	}

	for _, line := range lines {
		if line == "" {
			continue
		}
		name := strings.TrimSpace(line)

		fmt.Printf("  • Removing volume: %s\n", name)
		cmd := exec.CommandContext(ctx, "docker", "volume", "rm", "-f", name)
		if err := cmd.Run(); err != nil {
			fmt.Printf("    ⚠️  Warning: failed to remove %s: %v\n", name, err)
		}
	}

	return nil
}

func removeNetworks(ctx context.Context) error {
	fmt.Println("Removing networks...")

	cmd := exec.CommandContext(ctx, "docker", "network", "ls", "--filter", "name=paulenv-", "--format", "{{.ID}} {{.Name}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		fmt.Println("  • No networks found")
		return nil
	}

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		id := parts[0]
		name := ""
		if len(parts) > 1 {
			name = parts[1]
		}

		fmt.Printf("  • Removing network: %s\n", name)
		cmd := exec.CommandContext(ctx, "docker", "network", "rm", id)
		if err := cmd.Run(); err != nil {
			fmt.Printf("    ⚠️  Warning: failed to remove %s: %v\n", name, err)
		}
	}

	return nil
}

func shouldPruneCache() bool {
	fmt.Println("\nRemove Docker build cache?")
	fmt.Println("  This will free up disk space but slow down future rebuilds.")
	fmt.Print("  Remove cache? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

func pruneBuildCache(ctx context.Context) error {
	fmt.Println("Pruning build cache...")

	cmd := exec.CommandContext(ctx, "docker", "buildx", "prune", "--filter", "label=paulenv=true", "-f")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Check if buildx is not available
		if strings.Contains(stderr.String(), "buildx") {
			fmt.Println("  ⚠️  buildx not available, using standard builder prune")
			cmd = exec.CommandContext(ctx, "docker", "builder", "prune", "--filter", "label=paulenv=true", "-f")
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to prune build cache: %w\n%s", err, stderr.String())
			}
		} else {
			return fmt.Errorf("failed to prune build cache: %w\n%s", err, stderr.String())
		}
	}

	// Try to extract space reclaimed from output
	output := stdout.String()
	if strings.Contains(output, "Total:") || strings.Contains(output, "reclaimed") {
		fmt.Printf("  • %s", strings.TrimSpace(output))
	} else {
		fmt.Println("  • Build cache pruned")
	}

	return nil
}
