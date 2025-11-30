// # engine.go
// This file draft a base `ContainerEngine` interface to be able to build
// containers through various implementations of OCI and the compose spec.
// It was initially thought for docker compose and might necessitate heavy
// updates to integrate other "engines"

package engine

import (
	"context"
	"fmt"

	"github.com/peaberberian/paul-envs/internal/files"
)

type EngineType int

const (
	Docker EngineType = iota
)

type ContainerEngine interface {
	CheckPermissions(ctx context.Context) error
	BuildContainer(ctx context.Context, baseCompose string, project files.ProjectEntry, dotfilesDir string) error
	RunContainer(ctx context.Context, baseCompose string, project files.ProjectEntry, args []string) error
	Info(ctx context.Context) (ContainerInfo, error)
	CreateVolume(ctx context.Context, name string) error
}

type ContainerInfo struct {
	Name    string
	Version string
}

func New(ctx context.Context) (ContainerEngine, error) {
	// For now only docker is handled
	// TODO: other engines
	if docker, err := newDocker(ctx); err == nil {
		return docker, nil
	}
	return nil, fmt.Errorf("no supported container engine found, please install docker first")
}
