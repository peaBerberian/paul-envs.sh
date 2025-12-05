package commands

import (
	"context"
	"fmt"

	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/engine"
	"github.com/peaberberian/paul-envs/internal/files"
)

func Clean(ctx context.Context, filestore *files.FileStore, console *console.Console) error {
	console.Info("\n1. Projects' configuration")
	console.WriteLn("This will clean-up the container configurations you created with the 'create' command.")
	choice, err := console.AskYesNo("Remove projects configuration files?", true)
	if err != nil {
		return err
	} else if !choice {
		console.WriteLn("\nSkipping container configurations")
	} else {
		console.WriteLn("\nRemoving projects configuration files...")
		if err := filestore.DeleteDataDirectory(); err != nil {
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
		if err := filestore.DeleteConfigDirectory(); err != nil {
			return err
		}
	}

	containerEngine, err := engine.New(ctx)
	if err != nil {
		return err
	}

	console.Info("\n3. Container images removal")
	console.WriteLn("This will clean the container images built through the 'build' command.")
	choice, err = console.AskYesNo("Remove containers?", true)
	if err != nil {
		return err
	} else if !choice {
		console.WriteLn("\nSkipping container removal")
	} else {
		if err := removeContainers(ctx, containerEngine, console); err != nil {
			return err
		}
		if err := removeImages(ctx, containerEngine, console); err != nil {
			return err
		}
		if err := removeVolumes(ctx, containerEngine, console); err != nil {
			return err
		}
		if err := removeNetworks(ctx, containerEngine, console); err != nil {
			return err
		}
	}

	console.Info("\n4. Image build cache?")
	console.WriteLn("This will free up disk space but slow down future rebuilds.")
	choice, err = console.AskYesNo("Remove cache?", false)
	if err != nil {
		return err
	} else if !choice {
		console.WriteLn("\nSkipping build cache removal")
	} else if err := pruneBuildCache(ctx, containerEngine, console); err != nil {
		return err
	}

	console.Success("\nCleanup complete!")
	return nil
}

func removeContainers(ctx context.Context, containerEngine engine.ContainerEngine, console *console.Console) error {
	console.WriteLn("\nStopping and removing containers...")

	containers, err := containerEngine.ListContainers(ctx)
	if err != nil {
		return fmt.Errorf("cannot list current containers: %w", err)
	}
	for _, container := range containers {
		if container.ContainerName == nil {
			console.WriteLn("  • Removing unknown container")
		} else {
			console.WriteLn("  • Removing container: %s", container.ContainerName)
		}
		if err := containerEngine.RemoveContainer(ctx, container); err != nil {
			console.Warn("    WARNING: failed to remove container: %v", err)
		}
	}
	return nil
}

func removeImages(ctx context.Context, containerEngine engine.ContainerEngine, console *console.Console) error {
	console.WriteLn("\nRemoving images...")

	images, err := containerEngine.ListImages(ctx)
	if err != nil {
		return fmt.Errorf("cannot list current images: %w", err)
	}
	for _, image := range images {
		console.WriteLn("  • Removing image: %s", image.ImageName)
		if err := containerEngine.RemoveImage(ctx, image); err != nil {
			console.Warn("    WARNING: failed to remove image: %v", err)
		}
	}
	return nil
}

func removeVolumes(ctx context.Context, containerEngine engine.ContainerEngine, console *console.Console) error {
	console.WriteLn("\nRemoving volumes...")
	volumes, err := containerEngine.ListVolumes(ctx)
	if err != nil {
		return fmt.Errorf("cannot list current volumes: %w", err)
	}
	for _, volume := range volumes {
		console.WriteLn("  • Removing volume: %s", volume.VolumeName)
		if err := containerEngine.RemoveVolume(ctx, volume); err != nil {
			console.Warn("    WARNING: failed to remove volume: %v", err)
		}
	}
	return nil
}

func removeNetworks(ctx context.Context, containerEngine engine.ContainerEngine, console *console.Console) error {
	console.WriteLn("\nRemoving networks...")
	networks, err := containerEngine.ListNetworks(ctx)
	if err != nil {
		return fmt.Errorf("cannot list current networks: %w", err)
	}
	for _, network := range networks {
		console.WriteLn("  • Removing network: %s", network.NetworkName)
		if err := containerEngine.RemoveNetwork(ctx, network); err != nil {
			console.Warn("    WARNING: failed to remove network: %v", err)
		}
	}
	return nil
}

func pruneBuildCache(ctx context.Context, containerEngine engine.ContainerEngine, console *console.Console) error {
	console.WriteLn("\nPruning build cache...")
	return containerEngine.PruneBuildCache(ctx)
}
