// # engine.go
// This file draft a base `ContainerEngine` interface to be able to build
// containers through various implementations of OCI and the compose spec.
// It was initially thought for docker compose and might necessitate heavy
// updates to integrate other "engines"

package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/peaberberian/paul-envs/internal/files"
)

type EngineType int

const (
	Docker EngineType = iota
)

type ContainerEngine interface {
	BuildContainer(ctx context.Context, project files.ProjectEntry, dotfilesDir string) error
	RunContainer(ctx context.Context, project files.ProjectEntry, args []string) error
	Info(ctx context.Context) (EngineInfo, error)
	CreateVolume(ctx context.Context, name string) error
	HasBeenBuilt(ctx context.Context, projectName string) (bool, error)
	GetImageInfo(ctx context.Context, projectName string) (*ImageInfo, error)
}

type EngineInfo struct {
	Name    string
	Version string
}

type ImageInfo struct {
	ImageName string
	BuiltAt   *time.Time
}

func New(ctx context.Context) (ContainerEngine, error) {
	// For now only docker is handled
	// TODO: other engines
	if docker, err := newDocker(ctx); err == nil {
		return docker, nil
	}
	return nil, fmt.Errorf("no supported container engine found, please install docker first")
}
