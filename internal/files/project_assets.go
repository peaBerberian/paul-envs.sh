// # project_assets.go
// This file creates all files associated to a given project:
// -  Its `compose.yaml` file (which is on top of the base compose.yaml file)
// -  Its `.env` file
// -  Its `project.info` lockfile

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

	constants "github.com/peaberberian/paul-envs/internal"
	"github.com/peaberberian/paul-envs/internal/utils"
)

//go:embed embeds/*
var assets embed.FS

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

type ComposeTemplateData struct {
	ProjectName string
	Ports       []uint16
	EnableSSH   bool
	SSHKeyPath  string
	Volumes     []string
}

// projectInfo holds the parsed values from the project.info file
type projectInfo struct {
	version           utils.Version
	dockerfileVersion utils.Version
	buildEnvHash      string
	buildComposeHash  string
}

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

	if err := f.userFS.MkdirAsUser(f.GetProjectDir(projectName), 0755); err != nil {
		return fmt.Errorf("create project directory: %w", err)
	}

	envBytes := buf.Bytes()
	envHash := utils.BufferHash(envBytes)
	if err := f.userFS.WriteFileAsUser(f.GetEnvFilePathFor(projectName), envBytes, 0644); err != nil {
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
	composeHash := utils.BufferHash(composeBytes)
	if err := f.userFS.WriteFileAsUser(f.GetComposeFilePathFor(projectName), composeBytes, 0644); err != nil {
		return fmt.Errorf("write compose file: %w", err)
	}

	// Now create project.info file

	projectInfoVersion, err := utils.ParseVersion(constants.ProjectInfoVersion)
	if err != nil {
		return fmt.Errorf("impossibility to parse embedded file version: %w", err)
	}

	dockerfileVersion, err := utils.ParseVersion(constants.FileVersion)
	if err != nil {
		return fmt.Errorf("impossibility to parse embedded file version: %w", err)
	}

	pInfo := projectInfo{
		version:           projectInfoVersion,
		dockerfileVersion: dockerfileVersion,
		buildEnvHash:      envHash,
		buildComposeHash:  composeHash,
	}

	if err := f.writeProjectInfo(projectName, pInfo); err != nil {
		return fmt.Errorf("impossibility to write 'project.info' file: %w", err)
	}
	return nil
}

// ReadProjectInfo reads the project.info file and returns a populated projectInfo struct.
// TODO: call it
func (filestore *FileStore) CheckProject(projectName string) error {
	file, err := os.Open(filestore.getProjectInfoFilePathFor(projectName))
	if err != nil {
		return fmt.Errorf("could not open project.info: %w", err)
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
				return fmt.Errorf("invalid 'project.info' version '%s': %w", vStr, err)
			}

			piVersion, err := utils.ParseVersion(constants.ProjectInfoVersion)
			if err != nil {
				return fmt.Errorf("embedded project.info version is wrong '%s': %w", constants.ProjectInfoVersion, err)
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
				return fmt.Errorf("invalid 'project.info' Dockerfile version '%s': %w", vStr, err)
			}

			currBaseVersion, err := utils.ParseVersion(constants.FileVersion)
			if err != nil {
				return fmt.Errorf("embedded file version is wrong '%s': %w", constants.FileVersion, err)
			}
			if !v.IsCompatibleWithBase(currBaseVersion) {
				return fmt.Errorf("this project is incompatible with our Dockerfile")
			}

			dockerfileVersion = &v
			continue
		}
		if v, ok := strings.CutPrefix(line, "BUILD_ENV="); ok {
			buildEnvHash = v
			hash, err := utils.FileHash(filestore.GetEnvFilePathFor(projectName))
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
			hash, err := utils.FileHash(filestore.GetComposeFilePathFor(projectName))
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
		return fmt.Errorf("error reading project.info: %w", err)
	}

	if fileVersion == nil {
		return fmt.Errorf("error reading project.info: no VERSION key")
	}

	if dockerfileVersion == nil {
		return fmt.Errorf("error reading project.info: no DOCKERFILE_VERSION key")
	}

	if buildEnvHash == "" {
		return fmt.Errorf("error reading project.info: no BUILD_ENV key")
	}

	if buildComposeHash == "" {
		return fmt.Errorf("error reading project.info: no BUILD_COMPOSE key")
	}

	return nil
}

// Write the base Dockerfile and compose.yaml file in the base directory if not
// already done
func (f *FileStore) ensureCreatedBaseFiles() error {
	// Write docker file if needed
	baseDockerfilePath := filepath.Join(f.baseDataDir, "Dockerfile")
	_, err := os.Stat(baseDockerfilePath)
	if os.IsNotExist(err) {
		dockerfileData, err := assets.ReadFile("embeds/Dockerfile")
		if err != nil {
			return err
		}

		err = f.userFS.WriteFileAsUser(
			filepath.Join(f.baseDataDir, "Dockerfile"),
			dockerfileData, 0644)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	baseEntryPointPath := filepath.Join(f.baseDataDir, "docker-entrypoint.sh")
	_, err = os.Stat(baseEntryPointPath)
	if os.IsNotExist(err) {
		entrypointData, err := assets.ReadFile("embeds/docker-entrypoint.sh")
		if err != nil {
			return err
		}

		err = f.userFS.WriteFileAsUser(
			filepath.Join(f.baseDataDir, "docker-entrypoint.sh"),
			entrypointData, 0644)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// Then write base compose if needed
	baseComposePath := filepath.Join(f.baseDataDir, "Compose")
	_, err = os.Stat(baseComposePath)
	if os.IsNotExist(err) {
		composeData, err := assets.ReadFile("embeds/compose.yaml")
		if err != nil {
			return err
		}

		err = f.userFS.WriteFileAsUser(
			filepath.Join(f.baseDataDir, "compose.yaml"),
			composeData, 0644)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

// writeProjectInfo writes the given projectInfo to a project.info file
func (f *FileStore) writeProjectInfo(projectName string, pInfo projectInfo) error {
	bytes, err := formatProjectInfo(pInfo)
	if err != nil {
		return fmt.Errorf("could not format 'project.info' file: %v", err)
	}
	return f.userFS.WriteFileAsUser(f.getProjectInfoFilePathFor(projectName), bytes, 0644)
}

func formatProjectInfo(pInfo projectInfo) ([]byte, error) {
	var buf bytes.Buffer
	_, err := fmt.Fprintf(&buf,
		"VERSION=%s\n"+
			"DOCKERFILE_VERSION=%s\n"+
			"BUILD_ENV=%s\n"+
			"BUILD_COMPOSE=%s\n",
		pInfo.version.ToString(),
		pInfo.dockerfileVersion.ToString(),
		pInfo.buildEnvHash,
		pInfo.buildComposeHash,
	)

	if err != nil {
		return nil, fmt.Errorf("error formatting project.info content: %w", err)
	}

	return buf.Bytes(), nil
}
