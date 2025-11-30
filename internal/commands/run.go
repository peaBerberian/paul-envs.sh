package commands

import (
	"context"
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
	if err := containerEngine.CheckPermissions(ctx); err != nil {
		return err
	}

	var name string
	var cmdArgs []string

	if len(args) == 0 {
		var err error
		// TODO: don't use list here
		err = List(filestore, console)
		if err != nil {
			return fmt.Errorf("no project name given, and failed to list other projects: %w", err)
		}
		console.WriteLn("No project name given, listing projects...")
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

	hasBeenBuilt, _ := containerEngine.HasBeenBuilt(ctx, project.ProjectName)
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

	err = containerEngine.RunContainer(ctx, project, cmdArgs)
	if err != nil {
		return err
	}

	return nil
}
