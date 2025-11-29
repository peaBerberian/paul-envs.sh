package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
	"github.com/peaberberian/paul-envs/internal/utils"
)

func Build(ctx context.Context, args []string, filestore *files.FileStore, console *console.Console) error {
	if err := utils.CheckDockerComposeInstallation(ctx); err != nil {
		return fmt.Errorf("docker compose executable not found: %w", err)
	}
	if err := utils.CheckDockerPermissions(ctx); err != nil {
		return err
	}

	// TODO: Remove this check, should not be needed with a proper `getProjectName`
	if err := ensureBaseComposeExists(filestore); err != nil {
		return err
	}

	name, err := getProjectName(args, filestore, console, "build")
	if err != nil {
		return err
	}

	if err := utils.ValidateProjectName(name); err != nil {
		return err
	}
	if !filestore.DoesProjectExist(name) {
		return fmt.Errorf("project '%s' not found\nHint: Use 'paul-envs list' to see available projects", name)
	}
	console.Info("Preparing dotfiles...")
	tmpDotfilesDir, err := filestore.CreateProjectDotfilesDir(ctx, name)
	if err != nil {
		os.RemoveAll(tmpDotfilesDir)
		return fmt.Errorf("failed to prepare dotfiles for the container: %w", err)
	}
	defer os.RemoveAll(tmpDotfilesDir)

	console.Info("Ensuring that the shared cache volume is created...")
	if err := createSharedCacheVolume(ctx); err != nil {
		return err
	}

	console.Info("Building project '%s'...", name)
	if err := dockerComposeBuild(ctx, filestore, name, tmpDotfilesDir); err != nil {
		return err
	}
	console.Success("Built project '%s'", name)
	return nil
}

func ensureBaseComposeExists(filestore *files.FileStore) error {
	if _, err := os.Stat(filestore.GetBaseComposeFilePath()); os.IsNotExist(err) {
		return fmt.Errorf("base compose.yaml not found at %s\nCreate a configuration through the 'create' command first.", filestore.GetBaseComposeFilePath())
	}
	return nil
}

func getProjectName(args []string, filestore *files.FileStore, console *console.Console, action string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}

	// TODO: Re-using list here is bad
	if err := List(filestore, console); err != nil {
		return "", fmt.Errorf("failed to list projects: %w", err)
	}

	name, err := console.AskString(fmt.Sprintf("Enter project name to %s", action), "")
	if err != nil {
		return "", err
	}
	if name == "" {
		return "", errors.New("no project name provided")
	}
	return name, nil
}

func createSharedCacheVolume(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "volume", "create", "paulenv-shared-cache")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to create shared volume: %w.", err)
	}
	return nil
}

func dockerComposeBuild(ctx context.Context, filestore *files.FileStore, name string, dotfilesDir string) error {
	base := filestore.GetBaseComposeFilePath()
	relativeDotfilesDir, err := filepath.Rel(base, dotfilesDir)
	if err != nil {
		return fmt.Errorf("failed to construct dotfiles relative path: %w", err)
	}

	project, err := filestore.GetProject(name)
	if err != nil {
		return fmt.Errorf("failed to obtain information on project '%s': %w", name, err)
	}
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", base, "-f", project.ComposeFilePath, "--env-file", project.EnvFilePath, "build")
	envVars := append(os.Environ(),
		"COMPOSE_PROJECT_NAME=paulenv-"+name,
		"DOTFILES_DIR="+relativeDotfilesDir,
	)
	cmd.Env = envVars
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
