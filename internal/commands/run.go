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

func Run(ctx context.Context, args []string, filestore *files.FileStore, console *console.Console) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	containerEngine, err := engine.New(ctx)
	if err != nil {
		return err
	}

	var name string
	var cmdArgs []string

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
		name, err = console.AskString("Enter project name to run", "")
		if err != nil {
			return err
		}
		if name == "" {
			return fmt.Errorf("No project name provided")
		}
	} else {
		name = args[0]
		if len(args) > 1 {
			cmdArgs = args[1:]
		}
	}

	if err := utils.ValidateProjectName(name); err != nil {
		return err
	}

	project, err := filestore.GetProject(name)
	if err != nil {
		if !filestore.DoesProjectExist(name) {
			return fmt.Errorf("project '%s' not found\nHint: Use 'paul-envs list' to see available projects", name)
		}
		return fmt.Errorf("failed to obtain information on project '%s': %w", name, err)
	}

	hasBeenBuilt, err := containerEngine.HasBeenBuilt(ctx, project.ProjectName)
	if err != nil {
		return fmt.Errorf("failed to get the status of the '%s' project: %w", project.ProjectName, err)
	}
	if !hasBeenBuilt {
		console.WriteLn("The '%s' project has not been built yet", project.ProjectName)
		choice, err := console.AskYesNo("Do you want to build it?", true)
		if err != nil || !choice {
			return fmt.Errorf("please run 'paul-envs build %s' first", project.ProjectName)
		}
		if err = Build(ctx, []string{project.ProjectName}, filestore, console); err != nil {
			return fmt.Errorf("did not succeed to build project: %w", err)
		}
	}

	buildInfo, err := filestore.ReadBuildInfo(project.ProjectName)
	if err != nil {
		console.Warn("Could not get the information from a precedent build: %s", err)
	} else if buildInfo == nil {
		console.Warn("NIL BUILD INFO")
	} else {
		needsRebuild, err := filestore.NeedsRebuild(project.ProjectName, buildInfo)
		if err != nil {
			console.Warn("Cannot check previous build metadata: %s", err)
		}
		if needsRebuild {
			console.WriteLn("The '%s' project has changed and needs to be re-built", project.ProjectName)
			choice, err := console.AskYesNo("Do you want to build it?", true)
			if err != nil || choice {
				if err = Build(ctx, []string{project.ProjectName}, filestore, console); err != nil {
					return fmt.Errorf("did not succeed to build project: %w", err)
				}
			}
		}

	}

	containerList, err := containerEngine.ListContainers(ctx)
	if err != nil {
		console.Warn("Could not list already launched containers: %s", err)
	} else {
		for _, container := range containerList {
			if *container.ProjectName == name {
				console.Info("Container already created, joining it.")
				return containerEngine.JoinContainer(ctx, container, cmdArgs)
			}
		}
	}

	console.Info("Creating \"leader\" container for the project '%s', other 'run' calls will join it.", name)
	err = containerEngine.RunContainer(ctx, project, cmdArgs)
	if err != nil {
		return err
	}
	console.Info("Exiting leader container for that project, those that have joined it will also exit.")
	return nil
}
