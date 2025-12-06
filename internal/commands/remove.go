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

func Remove(ctx context.Context, args []string, filestore *files.FileStore, console *console.Console) error {
	var name string
	if len(args) == 0 {
		console.WriteLn("No project name given, listing projects...")
		entries, err := filestore.GetAllProjects()
		if err != nil {
			return fmt.Errorf("could not list all projects: %w", err)
		}
		if len(entries) == 0 {
			console.WriteLn("  (no project found)")
			console.WriteLn("Hint: Create one with 'paul-envs create <path>'")
			return errors.New("no existing project")
		}
		for _, entry := range entries {
			console.WriteLn("")
			console.WriteLn(entry.ProjectName)
			console.WriteLn("  Project path: %s", entry.ProjectPath)
		}
		console.WriteLn("")
		name, err = console.AskString("Enter project name to remove", "")
		if err != nil {
			return err
		}
		if name == "" {
			return fmt.Errorf("No project name provided")
		}
	} else {
		name = args[0]
	}

	if err := utils.ValidateProjectName(name); err != nil {
		return err
	}

	// if !filestore.DoesProjectExist(name) {
	// 	return fmt.Errorf("Project '%s' not found\nHint: Use 'paul-envs list' to see available projects", name)
	// }

	choice, err := console.AskYesNo(fmt.Sprintf("Remove project '%s'?", name), false)
	if err != nil {
		return err
	}
	if !choice {
		return nil
	}

	containerEngine, err := engine.New(ctx)
	if err != nil {
		return err
	}
	err = removeContainer(ctx, name, containerEngine, console)
	if err != nil {
		return err
	}
	err = removeImage(ctx, name, containerEngine, console)
	if err != nil {
		return err
	}
	err = removeVolume(ctx, name, containerEngine, console)
	if err != nil {
		return err
	}
	err = removeNetwork(ctx, name, containerEngine, console)
	if err != nil {
		return err
	}
	console.WriteLn("Removing '%s' project directory...", name)
	if err := filestore.DeleteProjectDirectory(name); err != nil {
		return fmt.Errorf("Failed to remove project directory: %w", err)
	}
	console.Success("Removed project directory with success!")
	console.Success("The project '%s' has been succesfully removed from your system!", name)
	return nil
}

func removeContainer(ctx context.Context, projectName string, containerEngine engine.ContainerEngine, console *console.Console) error {
	console.WriteLn("Stopping and removing '%s' container...", projectName)

	containers, err := containerEngine.ListContainers(ctx)
	if err != nil {
		return fmt.Errorf("cannot list current containers: %w", err)
	}
	for _, container := range containers {
		if container.ProjectName != nil && *container.ProjectName == projectName {
			if err := containerEngine.RemoveContainer(ctx, container); err != nil {
				return err
			}
			console.Success("Removed container with success!")
			return nil
		}
	}
	console.Info("no current running '%s' container found", projectName)
	return nil
}

func removeImage(ctx context.Context, projectName string, containerEngine engine.ContainerEngine, console *console.Console) error {
	console.WriteLn("Stopping and removing '%s' image...", projectName)

	images, err := containerEngine.ListImages(ctx)
	if err != nil {
		return fmt.Errorf("cannot list current images: %w", err)
	}
	for _, image := range images {
		if image.ProjectName != nil && *image.ProjectName == projectName {
			if err := containerEngine.RemoveImage(ctx, image); err != nil {
				return err
			}
			console.Success("Removed '%s' image with success!", image.ImageName)
			return nil
		}
	}
	console.Info("no '%s' image found", projectName)
	return nil
}

func removeVolume(ctx context.Context, projectName string, containerEngine engine.ContainerEngine, console *console.Console) error {
	console.WriteLn("Stopping and removing 'paulenv-%s-local' volume...", projectName)

	volumes, err := containerEngine.ListVolumes(ctx)
	if err != nil {
		return fmt.Errorf("cannot list current volumes: %w", err)
	}
	for _, volume := range volumes {
		if volume.VolumeName == fmt.Sprintf("paulenv-%s-local", projectName) {
			if err := containerEngine.RemoveVolume(ctx, volume); err != nil {
				return err
			}
			console.Success("Removed '%s' volume with success!", volume.VolumeName)
			return nil
		}
	}
	console.Info("no 'paulenv-%s-local' volume found", projectName)
	return nil
}

func removeNetwork(ctx context.Context, projectName string, containerEngine engine.ContainerEngine, console *console.Console) error {
	console.WriteLn("Stopping and removing '%s' network interfaces...", projectName)

	networks, err := containerEngine.ListNetworks(ctx)
	if err != nil {
		return fmt.Errorf("cannot list current networks: %w", err)
	}
	for _, network := range networks {
		if network.ProjectName != nil && *network.ProjectName == projectName {
			if err := containerEngine.RemoveNetwork(ctx, network); err != nil {
				return err
			}
			console.Success("Removed '%s' network with success!", network.NetworkName)
			return nil
		}
	}
	console.Info("no '%s' network found", projectName)
	return nil
}
