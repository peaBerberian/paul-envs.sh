package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/peaberberian/paul-envs/internal/commands"
	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Console: Handle input/output messages
	console := console.New(ctx, os.Stdin, os.Stdout, os.Stderr)

	// FileStore: Handle Compose and Environment files
	filestore, err := files.NewFileStore()
	if err != nil {
		console.Error("Error: %v", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		commands.Help(filestore, console)
		os.Exit(0)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var cmdErr error
	switch cmd {
	case "create", "c", "--create", "-c":
		cmdErr = commands.Create(args, filestore, console)
	case "list", "ls", "l", "--list", "-l":
		cmdErr = commands.List(filestore, console)
	case "build", "b", "--build", "-b":
		cmdErr = commands.Build(ctx, args, filestore, console)
	case "run", "e", "--run", "-e":
		cmdErr = commands.Run(ctx, args, filestore, console)
	case "remove", "rm", "r", "--remove", "-r":
		cmdErr = commands.Remove(args, filestore, console)
	case "version", "v", "--version", "-v":
		cmdErr = commands.Version(ctx, console)
	case "clean", "x", "--clean", "-x":
		cmdErr = commands.Clean(ctx, filestore, console)
	case "interactive", "i", "--interactive", "-i":
		cmdErr = commands.Interactive(ctx, filestore, console)
	case "help", "h", "--help", "-h":
		commands.Help(filestore, console)
	default:
		console.Error("Error: unknown command: %s", cmd)
		console.Error("Run with --help to have a list of authorized commands")
		os.Exit(1)
	}

	if cmdErr != nil {
		if errors.Is(cmdErr, context.Canceled) {
			console.Error("\nOperation cancelled")
			os.Exit(130)
		}
		console.Error("Error: %v", cmdErr)
		os.Exit(1)
	}
}
