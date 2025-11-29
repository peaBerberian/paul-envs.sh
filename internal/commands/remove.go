package commands

import (
	"fmt"

	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
	"github.com/peaberberian/paul-envs/internal/utils"
)

func Remove(args []string, filestore *files.FileStore, console *console.Console) error {
	var name string
	if len(args) == 0 {
		var err error
		err = List(filestore, console)
		if err != nil {
			return fmt.Errorf("no project name given, and failed to list other projects: %w", err)
		}
		console.WriteLn("No project name given, listing projects...")
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

	if !filestore.DoesProjectExist(name) {
		return fmt.Errorf("Project '%s' not found\nHint: Use 'paul-envs list' to see available projects", name)
	}

	choice, err := console.AskYesNo(fmt.Sprintf("Remove project '%s'?", name), false)
	if err != nil {
		return err
	}
	if !choice {
		return nil
	}

	if err := filestore.DeleteProjectDirectory(name); err != nil {
		return fmt.Errorf("Failed to remove project: %w", err)
	}
	console.Success("Removed project '%s'", name)
	console.WriteLn("Note: Docker volumes are preserved. To remove them, run:")
	console.WriteLn("  docker volume rm paulenv-%s-local", name)
	return nil
}
