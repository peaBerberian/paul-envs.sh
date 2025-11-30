package commands

import (
	"fmt"

	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
)

func List(filestore *files.FileStore, console *console.Console) error {
	entries, err := filestore.GetAllProjects()
	if err != nil {
		return fmt.Errorf("could not list all projects: %w", err)
	}
	if len(entries) == 0 {
		console.WriteLn("  (no project found)")
		console.WriteLn("Hint: Create one with 'paul-envs create <path>'")
	} else {
		for _, entry := range entries {
			printProjectInfo(entry, console)
		}
		if len(entries) <= 1 {
			console.WriteLn("Total: %d project", len(entries))
		} else {
			console.WriteLn("Total: %d projects", len(entries))
		}
	}
	return nil
}

func printProjectInfo(projectEntry files.ProjectEntry, console *console.Console) bool {
	console.Info("%s", projectEntry.ProjectName)
	console.WriteLn("  Mounted project   : %s", projectEntry.ProjectPath)
	console.WriteLn("  .env file         : %s", projectEntry.EnvFilePath)
	console.WriteLn("  compose.yaml file : %s", projectEntry.ComposeFilePath)
	console.WriteLn("")
	return true
}
