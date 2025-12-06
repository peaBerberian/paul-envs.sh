// # project_assets.go
// This file creates all files associated to a given project:
// -  Its `compose.yaml` file
// -  Its `.env` file
// -  Its `project.lock` lockfile
// -  Its `project.buildinfo` build state

package files

import (
	"bufio"
	"bytes"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	versions "github.com/peaberberian/paul-envs/internal"
	"github.com/peaberberian/paul-envs/internal/utils"
)

//go:embed embeds/*
var assets embed.FS

// Data needed to construct a project's `.env` file, which will list environment
// variables
type EnvTemplateData struct {
	ProjectID       string
	ProjectDestPath string
	ProjectHostPath string
	HostUID         string
	HostGID         string
	Username        string
	Shell           string
	InstallNode     string
	InstallRust     string
	InstallPython   string
	InstallGo       string
	EnableWasm      string
	EnableSSH       string
	EnableSudo      string
	Packages        string
	InstallNeovim   string
	InstallStarship string
	InstallAtuin    string
	InstallMise     string
	InstallZellij   string
	InstallJujutsu  string
	GitName         string
	GitEmail        string
}

// Data needed to construct a project's `compose.yaml` file, listing mounted
// ports, volumes...
type ComposeTemplateData struct {
	ProjectName string
	Ports       []uint16
	EnableSSH   bool
	SSHKeyPath  string
	Volumes     []string
}

// Holds the parsed values from the `project.lock` file associated to each project
type projectLockInfo struct {
	// The version of the `project.lock` file
	version utils.Version
	// The Dockerfile version it has been created for.
	dockerfileVersion utils.Version
}

// Hols
type buildState struct {
	// The version of the `project.buildinfo` file
	version utils.Version
	// A unique identifier for the machine that perform the build
	builtBy string
	// The hash of the `.env` file the last time the project has been built
	buildEnvHash string
	// The hash of the `compose.yaml` file the last time the project has been built
	buildComposeHash string
	// The last time it was built according to this tool
	builtAt time.Time
	// The name of the container engine which produced the last build (e.g. "docker")
	containerEngine string
	// The version of the container engine which produced the last build
	containerEngineVersion string
}

// RebuildReason indicates why a project needs to be rebuilt
type RebuildReason int

const (
	RebuildNotNeeded RebuildReason = iota
	RebuildDifferentMachine
	RebuildComposeChanged
	RebuildEnvChanged
	RebuildDifferentEngine
)

func (r RebuildReason) String() string {
	switch r {
	case RebuildNotNeeded:
		return "no rebuild needed"
	case RebuildDifferentMachine:
		return "last built on a different machine"
	case RebuildComposeChanged:
		return "compose.yaml file has changed since last build"
	case RebuildEnvChanged:
		return ".env file has changed since last build"
	case RebuildDifferentEngine:
		return "built on a different container engine"
	default:
		return "unknown reason"
	}
}

// ProjectLockStatus indicates the validity status of a project.lock file
type ProjectLockStatus int

const (
	ProjectLockValid ProjectLockStatus = iota
	ProjectLockMissing
	ProjectLockInvalidVersion
	ProjectLockIncompatibleDockerfile
	ProjectLockCorrupted
)

func (s ProjectLockStatus) String() string {
	switch s {
	case ProjectLockValid:
		return "valid"
	case ProjectLockMissing:
		return "project.lock file not found"
	case ProjectLockInvalidVersion:
		return "incompatible project.lock version"
	case ProjectLockIncompatibleDockerfile:
		return "incompatible Dockerfile version"
	case ProjectLockCorrupted:
		return "corrupted or malformed project.lock"
	default:
		return "unknown status"
	}
}

func (s ProjectLockStatus) IsValid() bool {
	return s == ProjectLockValid
}

// Create the directory and all files needed for the given project name, with
// the configuration given.
func (f *FileStore) CreateProjectFiles(
	projectName string,
	envTplData EnvTemplateData,
	composeTplData ComposeTemplateData,
) error {
	if err := f.ensureCreatedBaseFiles(); err != nil {
		return fmt.Errorf("create base files: %w", err)
	}

	// For env

	envTplCtnt, err := assets.ReadFile("embeds/env.tmpl")
	if err != nil {
		return fmt.Errorf("read env template: %w", err)
	}

	envTpl, err := template.New("env").Parse(string(envTplCtnt))
	if err != nil {
		return fmt.Errorf("parse env template: %w", err)
	}

	var buf bytes.Buffer
	if err := envTpl.Execute(&buf, envTplData); err != nil {
		return fmt.Errorf("execute env template: %w", err)
	}

	if err := f.userFS.MkdirAsUser(f.getProjectDir(projectName), 0755); err != nil {
		return fmt.Errorf("create project directory: %w", err)
	}

	envBytes := buf.Bytes()
	if err := f.userFS.WriteFileAsUser(f.GetProjectEnvFilePath(projectName), envBytes, 0644); err != nil {
		return fmt.Errorf("write env file: %w", err)
	}

	// Now for compose

	composeTplCtnt, err := assets.ReadFile("embeds/compose.tmpl")
	if err != nil {
		return fmt.Errorf("read compose template: %w", err)
	}

	composeTpl, err := template.New("compose").Parse(string(composeTplCtnt))
	if err != nil {
		return fmt.Errorf("parse compose template: %w", err)
	}

	buf.Reset()
	if err := composeTpl.Execute(&buf, composeTplData); err != nil {
		return fmt.Errorf("execute compose template: %w", err)
	}

	composeBytes := buf.Bytes()
	if err := f.userFS.WriteFileAsUser(f.GetProjectComposeFilePath(projectName), composeBytes, 0644); err != nil {
		return fmt.Errorf("write compose file: %w", err)
	}

	if err := f.writeProjectInfo(projectName); err != nil {
		return fmt.Errorf("impossibility to write 'project.lock' file: %w", err)
	}
	return nil
}

// Write the base Dockerfile file in the base directory if not already done
func (f *FileStore) ensureCreatedBaseFiles() error {
	// Write Dockerfile if needed
	baseDockerfilePath := filepath.Join(f.baseDataDir, "Dockerfile")
	_, err := os.Stat(baseDockerfilePath)
	if os.IsNotExist(err) {
		dockerfileData, err := assets.ReadFile("embeds/Dockerfile")
		if err != nil {
			return err
		}

		if err = f.userFS.MkdirAsUser(f.baseDataDir, 0755); err != nil {
			return err
		}
		if err = f.userFS.WriteFileAsUser(
			filepath.Join(f.baseDataDir, "Dockerfile"),
			dockerfileData, 0644); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	baseEntryPointPath := filepath.Join(f.baseDataDir, "entrypoint.sh")
	_, err = os.Stat(baseEntryPointPath)
	if os.IsNotExist(err) {
		entrypointData, err := assets.ReadFile("embeds/entrypoint.sh")
		if err != nil {
			return err
		}

		err = f.userFS.WriteFileAsUser(
			filepath.Join(f.baseDataDir, "entrypoint.sh"),
			entrypointData, 0644)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// Now, the placeholder directory
	if err := f.userFS.MkdirAsUser(filepath.Join(f.baseDataDir, "placeholder"), 0755); err != nil {
		return fmt.Errorf("create empty dotfiles placeholder: %w", err)
	}
	return nil
}

// writeProjectInfo writes the given projectLockInfo to a project.lock file
func (f *FileStore) writeProjectInfo(projectName string) error {
	bytes, err := formatProjectInfo()
	if err != nil {
		return fmt.Errorf("could not format 'project.lock' file: %v", err)
	}
	return f.userFS.WriteFileAsUser(f.getProjectInfoFilePathFor(projectName), bytes, 0644)
}

func formatProjectInfo() ([]byte, error) {
	var buf bytes.Buffer
	_, err := fmt.Fprintf(&buf,
		"VERSION=%s\n"+
			"DOCKERFILE_VERSION=%s\n",
		versions.ProjectLockVersion.ToString(),
		versions.DockerfileVersion.ToString(),
	)

	if err != nil {
		return nil, fmt.Errorf("error formatting project.lock content: %w", err)
	}

	return buf.Bytes(), nil
}

// ReadProjectInfo reads the project.lock file and returns a populated projectLockInfo struct.
func (filestore *FileStore) ReadProjectInfo(projectName string) (projectLockInfo, error) {
	file, err := os.Open(filestore.getProjectInfoFilePathFor(projectName))
	if err != nil {
		return projectLockInfo{}, fmt.Errorf("could not open project.lock: %w", err)
	}
	defer file.Close()

	var pInfo projectLockInfo
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		if vStr, ok := strings.CutPrefix(line, "VERSION="); ok {
			v, err := utils.ParseVersion(vStr)
			if err != nil {
				return projectLockInfo{}, fmt.Errorf("invalid 'project.lock' version '%s': %w", vStr, err)
			}

			pInfo.version = v
			continue
		}
		if vStr, ok := strings.CutPrefix(line, "DOCKERFILE_VERSION="); ok {
			v, err := utils.ParseVersion(vStr)
			if err != nil {
				return projectLockInfo{}, fmt.Errorf("invalid 'project.lock' Dockerfile version '%s': %w", vStr, err)
			}

			pInfo.dockerfileVersion = v
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return projectLockInfo{}, fmt.Errorf("error reading project.lock: %w", err)
	}

	return pInfo, nil
}

// Update the file which stores information on the last performed build, mainly
// to detect if we should re-build an image.
//
// Should be called after each build.
func (f *FileStore) RefreshBuildInfoFile(projectName string, engineName string, engineVersion string) error {
	machineId, err := f.getMachineID()
	if err != nil {
		return fmt.Errorf("failed to create 'project.buildinfo' file: %w", err)
	}
	envFilePath := f.GetProjectEnvFilePath(projectName)
	envBytes, err := os.ReadFile(envFilePath)
	if err != nil {
		return fmt.Errorf("failed to create 'project.buildinfo' file due to impossibility to read file '%s': %w", envFilePath, err)
	}
	envHash := utils.BufferHash(envBytes)
	composeFilePath := f.GetProjectComposeFilePath(projectName)
	composeBytes, err := os.ReadFile(composeFilePath)
	if err != nil {
		return fmt.Errorf("failed to create 'project.buildinfo' file due to impossibility to read file '%s': %w", composeFilePath, err)
	}
	composeHash := utils.BufferHash(composeBytes)
	now := time.Now()
	buildInfoBytes, err := formatBuildInfo(buildState{
		version:                versions.BuildInfoVersion,
		builtBy:                machineId,
		buildEnvHash:           envHash,
		buildComposeHash:       composeHash,
		builtAt:                now,
		containerEngine:        engineName,
		containerEngineVersion: engineVersion,
	})
	if err != nil {
		return fmt.Errorf("failed to create 'project.buildinfo' due to impossibility to format it: %w", err)
	}
	buildInfoPath := filepath.Join(f.getProjectDir(projectName), "project.buildinfo")
	err = f.userFS.WriteFileAsUser(buildInfoPath, buildInfoBytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to create 'project.buildinfo' due to impossibility to write '%s': %w", buildInfoPath, err)
	}
	return nil
}

// ReadBuildInfo reads the "project.buildinfo" file and returns a populated buildState struct.
func (filestore *FileStore) ReadBuildInfo(projectName string) (*buildState, error) {
	file, err := os.Open(filestore.getBuildInfoFilePathFor(projectName))
	if err != nil {
		return nil, fmt.Errorf("could not open 'project.buildinfo': %w", err)
	}
	defer file.Close()

	var bState buildState
	scanner := bufio.NewScanner(file)

	var parsedBuiltAt *time.Time = nil
	var parsedVersion *utils.Version = nil
	for scanner.Scan() {
		line := scanner.Text()

		if vStr, ok := strings.CutPrefix(line, "VERSION="); ok {
			v, err := utils.ParseVersion(vStr)
			if err != nil {
				return nil, fmt.Errorf("invalid 'project.buildinfo' version '%s': %w", vStr, err)
			}
			if !v.IsCompatibleWithBase(versions.BuildInfoVersion) {
				return nil, fmt.Errorf("unknown 'project.buildinfo' version '%s'", vStr)
			}
			parsedVersion = &v
			continue
		}
		if v, ok := strings.CutPrefix(line, "BUILT_BY="); ok {
			bState.builtBy = v
			continue
		}
		if v, ok := strings.CutPrefix(line, "BUILD_ENV="); ok {
			bState.buildEnvHash = v
			continue
		}
		if v, ok := strings.CutPrefix(line, "BUILD_COMPOSE="); ok {
			bState.buildComposeHash = v
			continue
		}
		if v, ok := strings.CutPrefix(line, "CONTAINER_ENGINE="); ok {
			bState.containerEngine = v
			continue
		}
		if v, ok := strings.CutPrefix(line, "CONTAINER_ENGINE_VERSION="); ok {
			bState.containerEngineVersion = v
			continue
		}
		if v, ok := strings.CutPrefix(line, "LAST_BUILT_AT="); ok {
			parsed, err := time.Parse(time.RFC3339, v)
			if err != nil {
				return nil, fmt.Errorf("invalid 'project.buildinfo' LAST_BUILT_AT value '%s': %w", v, err)
			}

			parsedBuiltAt = &parsed
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading 'project.buildinfo': %w", err)
	}

	if parsedBuiltAt == nil {
		return nil, errors.New("invalid 'project.buildinfo': no LAST_BUILT_AT value")
	}
	bState.builtAt = *parsedBuiltAt
	if parsedVersion == nil {
		return nil, errors.New("invalid 'project.buildinfo': no VERSION")
	}
	bState.version = *parsedVersion

	if bState.containerEngineVersion == "" {
		return nil, errors.New("invalid 'project.buildinfo': no CONTAINER_ENGINE_VERSION")
	}
	if bState.containerEngine == "" {
		return nil, errors.New("invalid 'project.buildinfo': no CONTAINER_ENGINE")
	}
	if bState.builtBy == "" {
		return nil, errors.New("invalid 'project.buildinfo': no BUILT_BY")
	}
	if bState.buildEnvHash == "" {
		return nil, errors.New("invalid 'project.buildinfo': no BUILD_ENV")
	}
	if bState.buildComposeHash == "" {
		return nil, errors.New("invalid 'project.buildinfo': no BUILD_COMPOSE")
	}
	return &bState, nil
}

// Updated NeedsRebuild function with reason
func (filestore *FileStore) NeedsRebuild(projectName string, bState *buildState) (bool, RebuildReason, error) {
	if bState == nil {
		return false, RebuildNotNeeded, errors.New("cannot determine if rebuild is needed, no build state")
	}
	machineId, err := filestore.getMachineID()
	if err != nil {
		return false, RebuildNotNeeded, fmt.Errorf("cannot get current 'machineid': %w", err)
	}
	if bState.builtBy != machineId {
		return true, RebuildDifferentMachine, nil
	}

	composeHash, err := utils.FileHash(filestore.GetProjectComposeFilePath(projectName))
	if err != nil {
		return false, RebuildNotNeeded, fmt.Errorf("cannot hash current compose file: %w", err)
	}
	if bState.buildComposeHash != composeHash {
		return true, RebuildComposeChanged, nil
	}

	envHash, err := utils.FileHash(filestore.GetProjectEnvFilePath(projectName))
	if err != nil {
		return false, RebuildNotNeeded, fmt.Errorf("cannot hash current env file: %w", err)
	}
	if bState.buildEnvHash != envHash {
		return true, RebuildEnvChanged, nil
	}

	if bState.containerEngine != "docker" {
		return true, RebuildDifferentEngine, nil
	}

	return false, RebuildNotNeeded, nil
}

// New function to validate project.lock status
func (filestore *FileStore) ValidateProjectLock(projectName string) (ProjectLockStatus, error) {
	pInfo, err := filestore.ReadProjectInfo(projectName)
	if err != nil {
		if os.IsNotExist(err) {
			return ProjectLockMissing, nil
		}
		// If it's a parsing error, it's likely corrupted
		if strings.Contains(err.Error(), "invalid") {
			return ProjectLockCorrupted, err
		}
		return ProjectLockCorrupted, err
	}

	// Check if the project.lock version is compatible
	if !pInfo.version.IsCompatibleWithBase(versions.ProjectLockVersion) {
		return ProjectLockInvalidVersion, fmt.Errorf(
			"project.lock version %s is incompatible with current version %s",
			pInfo.version.ToString(),
			versions.ProjectLockVersion.ToString(),
		)
	}

	// Check if the Dockerfile version is compatible
	if !pInfo.dockerfileVersion.IsCompatibleWithBase(versions.DockerfileVersion) {
		return ProjectLockIncompatibleDockerfile, fmt.Errorf(
			"Dockerfile version %s is incompatible with current version %s",
			pInfo.dockerfileVersion.ToString(),
			versions.DockerfileVersion.ToString(),
		)
	}

	return ProjectLockValid, nil
}

// Returns the format of the "project.buildinfo" file which contains information on
// the last build of a project.
func formatBuildInfo(bInfo buildState) ([]byte, error) {
	var buf bytes.Buffer
	_, err := fmt.Fprintf(&buf,
		"VERSION=%s\n"+
			"BUILT_BY=%s\n"+
			"BUILD_ENV=%s\n"+
			"BUILD_COMPOSE=%s\n"+
			"LAST_BUILT_AT=%s\n"+
			"CONTAINER_ENGINE=%s\n"+
			"CONTAINER_ENGINE_VERSION=%s\n",
		bInfo.version.ToString(),
		bInfo.builtBy,
		bInfo.buildEnvHash,
		bInfo.buildComposeHash,
		bInfo.builtAt.Format(time.RFC3339),
		bInfo.containerEngine,
		bInfo.containerEngineVersion,
	)

	if err != nil {
		return nil, fmt.Errorf("error formatting 'project.buildinfo' content: %w", err)
	}

	return buf.Bytes(), nil
}

// GetMachineID returns a persistent per-machine UUID stored in DATA_DIR/machine-id.
// If the file doesn't exist, it creates one.
func (f *FileStore) getMachineID() (string, error) {
	machineIdPath := filepath.Join(f.baseDataDir, "machine-id")
	if err := os.MkdirAll(filepath.Dir(machineIdPath), 0o700); err != nil {
		return "", fmt.Errorf("cannot create directory: %w", err)
	}
	if data, err := os.ReadFile(machineIdPath); err == nil {
		return string(data), nil
	}

	// Generate a new one
	id, err := utils.GenerateUUIDv4()
	if err != nil {
		return "", fmt.Errorf("cannot generate machine-id: %w", err)
	}

	if err := f.userFS.WriteFileAsUser(machineIdPath, []byte(id), 0o600); err != nil {
		return "", fmt.Errorf("cannot write machine-id: %w", err)
	}

	return id, nil
}
