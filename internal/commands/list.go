package commands

import (
	"context"
	"flag"
	"fmt"

	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/engine"
	"github.com/peaberberian/paul-envs/internal/files"
)

func List(ctx context.Context, args []string, filestore *files.FileStore, console *console.Console) error {
	nameOnly := false
	flagset := flag.NewFlagSet("list", flag.ContinueOnError)
	flagset.BoolVar(&nameOnly, "names", false, "Only display names")
	if err := flagset.Parse(args); err != nil {
		return err
	}

	entries, err := filestore.GetAllProjects()
	if err != nil {
		return fmt.Errorf("could not list all projects: %w", err)
	}

	containerEngine, err := engine.New(ctx)
	if err != nil {
		console.Warn("Could not instantiate container engine: %w", err)
	}

	var lastImageInfoWarning error = nil
	if len(entries) == 0 {
		console.WriteLn("  (no project found)")
		console.WriteLn("Hint: Create one with 'paul-envs create <path>'")
	} else if nameOnly {
		for _, entry := range entries {
			console.WriteLn(entry.ProjectName)
		}
	} else {
		for _, entry := range entries {
			var imageInfo *engine.ImageInfo
			if containerEngine == nil {
				imageInfo = nil
			} else {
				imageInfo, err = containerEngine.GetImageInfo(ctx, entry.ProjectName)
				if err != nil {
					lastImageInfoWarning = err
				}
			}
			printProjectInfo(entry, imageInfo, console)
		}
		if len(entries) <= 1 {
			console.WriteLn("Total: %d project", len(entries))
		} else {
			console.WriteLn("Total: %d projects", len(entries))
		}
		if lastImageInfoWarning != nil {
			console.Warn("Could not obtain image info for some project(s): %s", err)
		}
	}
	return nil
}

func printProjectInfo(projectEntry files.ProjectEntry, imageInfo *engine.ImageInfo, console *console.Console) bool {
	console.Info("%s", projectEntry.ProjectName)
	console.WriteLn("  Mounted project   : %s", projectEntry.ProjectPath)
	console.WriteLn("  .env file         : %s", projectEntry.EnvFilePath)
	console.WriteLn("  compose.yaml file : %s", projectEntry.ComposeFilePath)
	if imageInfo != nil {
		console.WriteLn("  Container image   : %s", imageInfo.ImageName)
		if imageInfo.BuiltAt == nil {
			console.WriteLn("  Last built at     : Never")
		} else {
			console.WriteLn("  Last built at     : %s", imageInfo.BuiltAt)
		}
	}
	console.WriteLn("")
	return true
}
