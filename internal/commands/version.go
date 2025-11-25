package commands

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	constants "github.com/peaberberian/paul-envs/internal"
	"github.com/peaberberian/paul-envs/internal/console"
)

func Version(ctx context.Context, console *console.Console) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	console.WriteLn("paul-envs version %s", constants.Version)
	console.WriteLn("Go version: %s", runtime.Version())
	cmd := exec.CommandContext(ctx, "docker", "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("Failed to obtain docker version: %w", err)
	}
	console.WriteLn("Docker version: %s", output)
	return nil
}
