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

// Abstraction allowing to create images and run containers regardless of the softwared
// used (docker, podman...)
type ContainerEngine interface {
	// Return information on the current chosen "container engine" (its name, its version...)
	Info(ctx context.Context) (EngineInfo, error)
	// Build the image associated to the given project, also copying the given
	// `relDotfilesDir` in the container's $HOME. `relDotfilesDir` must be a relative path
	// from paul-envs' Dockerfile and reachable from its context.
	BuildImage(ctx context.Context, project files.ProjectEntry, relDotfilesDir string) error
	// Run the container whose image has previously been built with `BuildImage`.
	//
	// If `args` is empty, will start an interactive tty session with the project's shell of
	// choice.
	//
	// If `args` is not empty, the container will just execute the given commands and then
	// exit.
	RunContainer(ctx context.Context, project files.ProjectEntry, args []string) error
	JoinContainer(ctx context.Context, containerInfo ContainerInfo, args []string) error
	// Create the persistent volume whose name is given as argument.
	CreateVolume(ctx context.Context, name string) error
	// Check if the project in argument has been built succesfully before and return
	// `true` if that's the case.
	//
	// Return an `error` if we could not do the check, in which case we don't know if the
	// project has been built.
	HasBeenBuilt(ctx context.Context, projectName string) (bool, error)
	// Returns information on the given project from the point of view of the container
	// engine.
	GetImageInfo(ctx context.Context, projectName string) (*ImageInfo, error)
	// List containers currently known by this container engine
	ListContainers(ctx context.Context) ([]ContainerInfo, error)
	// Remove container listed from this container engine
	RemoveContainer(ctx context.Context, container ContainerInfo) error
	// TODO:
	// // List images currently known by this container engine
	// ListImages(ctx context.Context) ([]ImageInfo, error)
	// // Remove image listed from this container engine
	// RemoveImage(ctx context.Context, image ImageInfo) error
	// // List volumes currently known by this container engine
	// ListVolumes(ctx context.Context) ([]VolumeInfo, error)
	// // Remove volume listed from this container engine
	// RemoveVolume(ctx context.Context, volume VolumeInfo) error
	// // List networks currently known by this container engine
	// ListNetworks(ctx context.Context) ([]NetworkInfo, error)
	// // Remove network listed from this container engine
	// RemoveNetwork(ctx context.Context, network NetworkInfo) error
	// // Remove the `ContainerEngine`'s build cache from metadata linked to this
	// // executable
	// PruneBuildCache(ctx context.Context) error
}

// Returns information on a specific "engine" able to create images and run containers
type EngineInfo struct {
	// The name to which it is refered to, e.g. "docker"
	Name string
	// The version of that software that is currently used.
	// /!\ Should fit on a single line
	Version string
}

// Information on a particular built image
type ImageInfo struct {
	// The name of the corresponding paulenv project, if one
	ProjectName *string
	// The name it is actually refered to by the container engine.
	ImageName string
	// The timestamp at which it has last been built.
	// `nil` if it never has been built.
	BuiltAt *time.Time
}

// Information on a particular container as stored by the container engine
type ContainerInfo struct {
	// The name of the corresponding paulenv project, if one
	ProjectName *string
	// The name it is actually refered to by the container engine.
	ContainerName *string
	// The name of the corresponding image
	ImageName *string
	// Its Id with which it can be refered to
	ContainerId string
}

// Create a new `ContainerEngine`, based on what's available right now.
func New(ctx context.Context) (ContainerEngine, error) {
	// For now only docker is handled
	// TODO: other engines
	if docker, err := newDocker(ctx); err == nil {
		return docker, nil
	}
	return nil, fmt.Errorf("no supported container engine found, please install docker first")
}
