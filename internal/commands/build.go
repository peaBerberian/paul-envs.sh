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
	if err := checkDockerComposeInstallation(ctx); err != nil {
		return fmt.Errorf("docker compose executable not found: %w", err)
	}
	if err := checkDockerPermissions(ctx); err != nil {
		return err
	}

	if err := ensureBaseComposeExists(filestore); err != nil {
		return err
	}

	name, err := getProjectName(args, filestore, console, "build")
	if err != nil {
		return err
	}

	if err := validateProjectFiles(filestore, name); err != nil {
		return err
	}

	// TODO: nextdotfiles creation should probably be done by the filestore
	tmpDotfilesDir := filepath.Join(filestore.GetProjectDir(name), "nextdotfiles")
	console.Info("Preparing dotfiles...")
	if err := filestore.CopyDotfilesTo(ctx, tmpDotfilesDir); err != nil {
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

func validateProjectFiles(filestore *files.FileStore, name string) error {
	composeFile := filestore.GetComposeFilePathFor(name)
	envFile := filestore.GetEnvFilePathFor(name)
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("project '%s' not found; use 'paul-envs list' to see available projects", name)
	}
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		return fmt.Errorf("project '%s' not found; use 'paul-envs list' to see available projects", name)
	}
	return utils.ValidateProjectName(name)
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
	compose := filestore.GetComposeFilePathFor(name)
	env := filestore.GetEnvFilePathFor(name)

	relativeDotfilesDir, err := filepath.Rel(base, dotfilesDir)
	if err != nil {
		return fmt.Errorf("failed to construct dotfiles relative path: %w", err)
	}

	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", base, "-f", compose, "--env-file", env, "build")
	envVars := append(os.Environ(),
		"COMPOSE_PROJECT_NAME=paulenv-"+name,
		"DOTFILES_DIR="+relativeDotfilesDir,
	)
	cmd.Env = envVars
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
