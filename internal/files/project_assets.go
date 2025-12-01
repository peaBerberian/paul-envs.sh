// # project_assets.go
// This file creates all files associated to a given project:
// -  Its `compose.yaml` file
// -  Its `.env` file
// -  Its `project.lock` lockfile

package files

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	constants "github.com/peaberberian/paul-envs/internal"
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
	// The version of the `build.info` file
	version utils.Version
	// A unique identifier for the machine that perform the build
	builtBy string
	// The hash of the `.env` file the last time the project has been built
	buildEnvHash string
	// The hash of the `compose.yaml` file the last time the project has been built
	buildComposeHash string
	// The last time it was built according to this tool
	builtAt time.Time
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
	// TODO: On build only
	// envHash := utils.BufferHash(envBytes)
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
	// TODO: On build only
	// composeHash := utils.BufferHash(composeBytes)
	if err := f.userFS.WriteFileAsUser(f.GetProjectComposeFilePath(projectName), composeBytes, 0644); err != nil {
		return fmt.Errorf("write compose file: %w", err)
	}

	// Now create project.lock file

	projectInfoVersion, err := utils.ParseVersion(constants.ProjectLockVersion)
	if err != nil {
		return fmt.Errorf("impossibility to parse embedded file version: %w", err)
	}

	dockerfileVersion, err := utils.ParseVersion(constants.DockerfileVersion)
	if err != nil {
		return fmt.Errorf("impossibility to parse embedded file version: %w", err)
	}

	pInfo := projectLockInfo{
		version:           projectInfoVersion,
		dockerfileVersion: dockerfileVersion,
	}

	if err := f.writeProjectInfo(projectName, pInfo); err != nil {
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
func (f *FileStore) writeProjectInfo(projectName string, pInfo projectLockInfo) error {
	bytes, err := formatProjectInfo(pInfo)
	if err != nil {
		return fmt.Errorf("could not format 'project.lock' file: %v", err)
	}
	return f.userFS.WriteFileAsUser(f.getProjectInfoFilePathFor(projectName), bytes, 0644)
}

func formatProjectInfo(pInfo projectLockInfo) ([]byte, error) {
	var buf bytes.Buffer
	_, err := fmt.Fprintf(&buf,
		"VERSION=%s\n"+
			"DOCKERFILE_VERSION=%s\n",
		pInfo.version.ToString(),
		pInfo.dockerfileVersion.ToString(),
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

// TODO:
func (filestore *FileStore) CheckProjectLock(projectName string) error {
	file, err := os.Open(filestore.getProjectInfoFilePathFor(projectName))
	if err != nil {
		return fmt.Errorf("could not open project.lock: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	var fileVersion *utils.Version = nil
	var dockerfileVersion *utils.Version = nil
	var buildEnvHash = ""
	var buildComposeHash = ""
	for scanner.Scan() {
		line := scanner.Text()

		if vStr, ok := strings.CutPrefix(line, "VERSION="); ok {
			v, err := utils.ParseVersion(vStr)
			if err != nil {
				return fmt.Errorf("invalid 'project.lock' version '%s': %w", vStr, err)
			}

			piVersion, err := utils.ParseVersion(constants.ProjectLockVersion)
			if err != nil {
				return fmt.Errorf("embedded project.lock version is wrong '%s': %w", constants.ProjectLockVersion, err)
			}
			if !v.IsCompatibleWithBase(piVersion) {
				return fmt.Errorf("this project is incompatible with the current version of paul-envs")
			}
			fileVersion = &v
			continue
		}
		if vStr, ok := strings.CutPrefix(line, "DOCKERFILE_VERSION="); ok {
			v, err := utils.ParseVersion(vStr)
			if err != nil {
				return fmt.Errorf("invalid 'project.lock' Dockerfile version '%s': %w", vStr, err)
			}

			currBaseVersion, err := utils.ParseVersion(constants.DockerfileVersion)
			if err != nil {
				return fmt.Errorf("embedded file version is wrong '%s': %w", constants.DockerfileVersion, err)
			}
			if !v.IsCompatibleWithBase(currBaseVersion) {
				return fmt.Errorf("this project is incompatible with our Dockerfile")
			}

			dockerfileVersion = &v
			continue
		}
		if v, ok := strings.CutPrefix(line, "BUILD_ENV="); ok {
			buildEnvHash = v
			hash, err := utils.FileHash(filestore.GetProjectEnvFilePath(projectName))
			if err != nil {
				// TODO:: Caller should then propose to re-build the project
				return fmt.Errorf("error hashing project's env file: %w", err)
			}
			if hash != buildEnvHash {
				// TODO:: Caller should then propose to re-build the project
				return fmt.Errorf(".env file hash does not match its last build")
			}
			continue
		}
		if v, ok := strings.CutPrefix(line, "BUILD_COMPOSE="); ok {
			buildComposeHash = v
			hash, err := utils.FileHash(filestore.GetProjectComposeFilePath(projectName))
			if err != nil {
				// TODO:: Caller should then propose to re-build the project
				return fmt.Errorf("error hashing project's compose file: %w", err)
			}
			if hash != buildEnvHash {
				// TODO:: Caller should then propose to re-build the project
				return fmt.Errorf("compose.yaml file hash does not match its last build")
			}
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading project.lock: %w", err)
	}

	if fileVersion == nil {
		return fmt.Errorf("error reading project.lock: no VERSION key")
	}

	if dockerfileVersion == nil {
		return fmt.Errorf("error reading project.lock: no DOCKERFILE_VERSION key")
	}

	if buildEnvHash == "" {
		return fmt.Errorf("error reading project.lock: no BUILD_ENV key")
	}

	if buildComposeHash == "" {
		return fmt.Errorf("error reading project.lock: no BUILD_COMPOSE key")
	}

	return nil
}

// Update the file which stores information on the last performed build, mainly
// to detect if we should re-build an image.
//
// Should be called after each build.
func (f *FileStore) RefreshBuildInfoFile(projectName string) error {
	machineId, err := f.getMachineID()
	if err != nil {
		return fmt.Errorf("failed to create 'build.info' file: %w", err)
	}
	version, err := utils.ParseVersion(constants.BuildInfoVersion)
	if err != nil {
		return fmt.Errorf("failed to create 'build.info' file due to invalid embedded version: %w", err)
	}
	envFilePath := f.GetProjectEnvFilePath(projectName)
	envBytes, err := os.ReadFile(envFilePath)
	if err != nil {
		return fmt.Errorf("failed to create 'build.info' file due to impossibility to read file '%s': %w", envFilePath, err)
	}
	envHash := utils.BufferHash(envBytes)
	composeFilePath := f.GetProjectComposeFilePath(projectName)
	composeBytes, err := os.ReadFile(composeFilePath)
	if err != nil {
		return fmt.Errorf("failed to create 'build.info' file due to impossibility to read file '%s': %w", composeFilePath, err)
	}
	composeHash := utils.BufferHash(composeBytes)
	now := time.Now()
	buildInfoBytes, err := formatBuildInfo(buildState{
		version:          version,
		builtBy:          machineId,
		buildEnvHash:     envHash,
		buildComposeHash: composeHash,
		builtAt:          now,
	})
	if err != nil {
		return fmt.Errorf("failed to create 'build.info' due to impossibility to format it: %w", err)
	}
	buildInfoPath := filepath.Join(f.getProjectDir(projectName), "build.info")
	err = f.userFS.WriteFileAsUser(buildInfoPath, buildInfoBytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to create 'build.info' due to impossibility to write '%s': %w", buildInfoPath, err)
	}
	return nil
}

// Returns the format of the "build.info" file which contains information on
// the last build of a project.
func formatBuildInfo(bInfo buildState) ([]byte, error) {
	var buf bytes.Buffer
	_, err := fmt.Fprintf(&buf,
		"VERSION=%s\n"+
			"BUILT_BY=%s\n"+
			"BUILD_ENV=%s\n"+
			"BUILD_COMPOSE=%s\n"+
			"LAST_BUILT_AT=%s\n",
		bInfo.version.ToString(),
		bInfo.builtBy,
		bInfo.buildEnvHash,
		bInfo.buildComposeHash,
		bInfo.builtAt.Format(time.RFC3339),
	)

	if err != nil {
		return nil, fmt.Errorf("error formatting build.info content: %w", err)
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
