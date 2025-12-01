package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/engine"
	"github.com/peaberberian/paul-envs/internal/files"
	"github.com/peaberberian/paul-envs/internal/utils"
)

func Build(ctx context.Context, args []string, filestore *files.FileStore, console *console.Console) error {
	containerEngine, err := engine.New(ctx)
	if err != nil {
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
		filestore.RemoveProjectDotfilesDir(name)
		return fmt.Errorf("failed to prepare dotfiles for the container: %w", err)
	}
	defer filestore.RemoveProjectDotfilesDir(name)

	console.Info("Ensuring that the shared cache volume is created...")
	if err := containerEngine.CreateVolume(ctx, "paulenv-shared-cache"); err != nil {
		return fmt.Errorf("Failed to create shared volume: %w.", err)
	}

	console.Info("Building project '%s'...", name)
	project, err := filestore.GetProject(name)
	if err != nil {
		return fmt.Errorf("failed to obtain information on project '%s': %w", name, err)
	}
	if err := containerEngine.BuildImage(ctx, project, tmpDotfilesDir); err != nil {
		return err
	}
	engineInfo, err := containerEngine.Info(ctx)
	if err != nil {
		console.Warn("Could not refresh 'project.buildinfo' file for this project: impossible to get container engine version: %s", err)
	} else {
		err = filestore.RefreshBuildInfoFile(name, engineInfo.Name, engineInfo.Version)
		if err != nil {
			console.Warn("Could not refresh 'project.buildinfo' file for this project: %s", err)
		}
	}
	console.Success("Built project '%s'", name)
	return nil
}

func getProjectName(args []string, filestore *files.FileStore, console *console.Console, action string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}
	entries, err := filestore.GetAllProjects()
	if err != nil {
		return "", fmt.Errorf("could not list all projects: %w", err)
	}
	if len(entries) == 0 {
		console.WriteLn("  (no project found)")
		console.WriteLn("Hint: Create one with 'paul-envs create <path>'")
		return "", errors.New("no existing project")
	}
	for _, entry := range entries {
		console.WriteLn("")
		console.WriteLn(entry.ProjectName)
		console.WriteLn("  Project path: %s", entry.ProjectPath)
	}
	console.WriteLn("")
	name, err := console.AskString(fmt.Sprintf("Enter project name to %s", action), "")
	if err != nil {
		return "", err
	}
	if name == "" {
		return "", errors.New("no project name provided")
	}
	return name, nil
}
