package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
	"github.com/peaberberian/paul-envs/internal/utils"
)

func Run(ctx context.Context, args []string, filestore *files.FileStore, console *console.Console) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := checkDockerComposeInstallation(ctx); err != nil {
		return fmt.Errorf("docker compose not found, is it installed: %w", err)
	}
	if err := checkDockerPermissions(ctx); err != nil {
		return err
	}

	baseCompose := filestore.GetBaseComposeFile()
	if _, err := os.Stat(baseCompose); os.IsNotExist(err) {
		return fmt.Errorf("Base compose.yaml not found at %s", baseCompose)
	}

	var name string
	var cmdArgs []string

	if len(args) == 0 {
		var err error
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

	composeFile := filestore.GetComposeFilePathFor(name)
	envFile := filestore.GetEnvFilePathFor(name)
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("Project '%s' not found\nHint: Use 'paul-envs list' to see available projects", name)
	}
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		return fmt.Errorf("Project '%s' not found\nHint: Use 'paul-envs list' to see available projects", name)
	}

	args = []string{"compose", "-f", baseCompose, "-f", composeFile, "--env-file", envFile, "run", "--rm", "paulenv"}
	args = append(args, cmdArgs...)

	if err := ctx.Err(); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Env = append(os.Environ(), "COMPOSE_PROJECT_NAME=paulenv-"+name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Run failed: %w", err)
	}

	return nil
}
