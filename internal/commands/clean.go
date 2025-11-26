package commands

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
)

func Clean(ctx context.Context, filestore *files.FileStore, console *console.Console) error {
	if err := checkDockerPermissions(ctx); err != nil {
		return err
	}

	console.Info("\n1. Projects' configuration")
	console.WriteLn("This will clean-up the container configurations you created with the 'create' command.")
	choice, err := console.AskYesNo("Remove projects configuration files?", true)
	if err != nil {
		return err
	} else if !choice {
		console.WriteLn("\nSkipping container configurations")
	} else {
		console.WriteLn("\nRemoving projects configuration files...")
		if err := filestore.RemoveBaseDataDirectory(); err != nil {
			return err
		}
	}

	console.Info("\n2. paul-envs' configuration")
	console.WriteLn("This will reset the global 'paul-envs' configuration.")
	choice, err = console.AskYesNo("Remove paul-envs configuration?", true)
	if err != nil {
		return err
	} else if !choice {
		console.WriteLn("\nSkipping container configurations")
	} else {
		console.WriteLn("\nRemoving projects configuration files...")
		if err := filestore.RemoveConfigDirectory(); err != nil {
			return err
		}
	}

	console.Info("\n3. Container images removal")
	console.WriteLn("This will clean the container images built through the 'build' command.")
	choice, err = console.AskYesNo("Remove containers?", true)
	if err != nil {
		return err
	} else if !choice {
		console.WriteLn("\nSkipping container removal")
	} else {
		if err := removeContainers(ctx, console); err != nil {
			return err
		}
		if err := removeImages(ctx, console); err != nil {
			return err
		}
		if err := removeVolumes(ctx, console); err != nil {
			return err
		}
		if err := removeNetworks(ctx, console); err != nil {
			return err
		}
	}

	console.Info("\n4. Docker build cache?")
	console.WriteLn("This will free up disk space but slow down future rebuilds.")
	choice, err = console.AskYesNo("Remove cache?", false)
	if err != nil {
		return err
	} else if !choice {
		console.WriteLn("\nSkipping build cache removal")
	} else {
		// TODO: recheck that one
		if err := pruneBuildCache(ctx, console); err != nil {
			return err
		}
	}

	console.Success("\nCleanup complete!")
	return nil
}

func removeContainers(ctx context.Context, console *console.Console) error {
	console.WriteLn("\nStopping and removing containers...")

	// List containers
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "name=paulenv-", "--format", "{{.ID}} {{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		console.WriteLn("  • No containers found")
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

		console.WriteLn("  • Removing container: %s", name)
		cmd := exec.CommandContext(ctx, "docker", "rm", "-f", id)
		if err := cmd.Run(); err != nil {
			console.Warn("    WARNING: failed to remove %s: %v", name, err)
		}
	}

	return nil
}

func removeImages(ctx context.Context, console *console.Console) error {
	console.WriteLn("\nRemoving images...")

	cmd := exec.CommandContext(ctx, "docker", "images", "--filter", "reference=paulenv:*", "--format", "{{.ID}} {{.Repository}}:{{.Tag}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		console.WriteLn("  • No images found")
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

		console.WriteLn("  • Removing image: %s", tag)
		cmd := exec.CommandContext(ctx, "docker", "rmi", "-f", id)
		if err := cmd.Run(); err != nil {
			console.Warn("    WARNING: failed to remove %s: %v", tag, err)
		}
	}

	return nil
}

func removeVolumes(ctx context.Context, console *console.Console) error {
	console.WriteLn("\nRemoving volumes...")

	cmd := exec.CommandContext(ctx, "docker", "volume", "ls", "--filter", "name=paulenv-", "--format", "{{.Name}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list volumes: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		console.WriteLn("  • No volumes found")
		return nil
	}

	for _, line := range lines {
		if line == "" {
			continue
		}
		name := strings.TrimSpace(line)

		console.WriteLn("  • Removing volume: %s", name)
		cmd := exec.CommandContext(ctx, "docker", "volume", "rm", "-f", name)
		if err := cmd.Run(); err != nil {
			console.Warn("    WARNING: failed to remove %s: %v", name, err)
		}
	}

	return nil
}

func removeNetworks(ctx context.Context, console *console.Console) error {
	console.WriteLn("\nRemoving networks...")

	cmd := exec.CommandContext(ctx, "docker", "network", "ls", "--filter", "name=paulenv-", "--format", "{{.ID}} {{.Name}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		console.WriteLn("  • No networks found")
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

		console.WriteLn("  • Removing network: %s", name)
		cmd := exec.CommandContext(ctx, "docker", "network", "rm", id)
		if err := cmd.Run(); err != nil {
			console.Warn("    WARNING: failed to remove %s: %v", name, err)
		}
	}

	return nil
}

func pruneBuildCache(ctx context.Context, console *console.Console) error {
	console.WriteLn("\nPruning build cache...")

	cmd := exec.CommandContext(ctx, "docker", "buildx", "prune", "--filter", "label=paulenv=true", "-f")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Check if buildx is not available
		if strings.Contains(stderr.String(), "buildx") {
			console.Warn("  WARNING: buildx not available, using standard builder prune")
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
		console.WriteLn("  • %s", strings.TrimSpace(output))
	} else {
		console.Success("  • Build cache pruned")
	}

	return nil
}
