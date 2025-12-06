package commands

import (
	"context"
	"fmt"

	versions "github.com/peaberberian/paul-envs/internal"
	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/engine"
)

func Version(ctx context.Context, console *console.Console) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	console.WriteLn("paul-envs version %d.%d.%d",
		versions.Version.Major, versions.Version.Minor, versions.Version.Patch)
	containerEngine, err := engine.New(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch information on container engine: %w", err)
	}
	info, err := containerEngine.Info(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch information on container engine: %w", err)
	}
	console.WriteLn("Container engine: %s", info.Name)
	console.WriteLn("Container engine version: %s", info.Version)
	return nil
}
