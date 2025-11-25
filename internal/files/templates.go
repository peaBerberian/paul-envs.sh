package files

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed assets/*
var assets embed.FS

type EnvTemplateData struct {
	ProjectComposeFilename string
	ProjectID              string
	ProjectDestPath        string
	ProjectHostPath        string
	HostUID                string
	HostGID                string
	Username               string
	Shell                  string
	InstallNode            string
	InstallRust            string
	InstallPython          string
	InstallGo              string
	EnableWasm             string
	EnableSSH              string
	EnableSudo             string
	Packages               string
	InstallNeovim          string
	InstallStarship        string
	InstallAtuin           string
	InstallMise            string
	InstallZellij          string
	InstallJujutsu         string
	GitName                string
	GitEmail               string
}

type ComposeTemplateData struct {
	ProjectName string
	Ports       []uint16
	EnableSSH   bool
	SSHKeyPath  string
	Volumes     []string
}

func (f *FileStore) CreateProjectEnvFile(projectName string, tplData EnvTemplateData) error {
	fileLoc := f.GetEnvFilePathFor(projectName)
	if err := f.ensureProjectDir(fileLoc); err != nil {
		return err
	}

	tmplContent, err := assets.ReadFile("assets/env.tmpl")
	if err != nil {
		return fmt.Errorf("read env template: %w", err)
	}

	envTpl, err := template.New("env").Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("parse env template: %w", err)
	}

	var buf bytes.Buffer
	if err := envTpl.Execute(&buf, tplData); err != nil {
		return fmt.Errorf("execute env template: %w", err)
	}

	if err := f.ensureCreatedBaseFiles(); err != nil {
		return fmt.Errorf("write base project files: %w", err)
	}
	if err := os.WriteFile(fileLoc, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write env file: %w", err)
	}
	return nil
}

func (f *FileStore) CreateProjectComposeFile(projectName string, tplData ComposeTemplateData) error {
	fileLoc := f.GetComposeFilePathFor(projectName)
	if err := f.ensureProjectDir(fileLoc); err != nil {
		return err
	}

	tmplContent, err := assets.ReadFile("assets/compose.tmpl")
	if err != nil {
		return fmt.Errorf("read compose template: %w", err)
	}

	composeTpl, err := template.New("compose").Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("parse compose template: %w", err)
	}

	var buf bytes.Buffer
	if err := composeTpl.Execute(&buf, tplData); err != nil {
		return fmt.Errorf("execute compose template: %w", err)
	}

	if err := f.ensureCreatedBaseFiles(); err != nil {
		return fmt.Errorf("write base project files: %w", err)
	}
	if err := os.WriteFile(fileLoc, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write compose file: %w", err)
	}
	return nil
}

func (f *FileStore) ensureProjectDir(fileLoc string) error {
	if err := os.MkdirAll(f.GetProjectDirBase(), 0755); err != nil {
		return fmt.Errorf("create base project directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(fileLoc), 0755); err != nil {
		return fmt.Errorf("create project directory: %w", err)
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
		dockerfileData, err := assets.ReadFile("assets/Dockerfile")
		if err != nil {
			return err
		}

		err = os.WriteFile(
			filepath.Join(f.baseDataDir, "Dockerfile"),
			dockerfileData, 0644)
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
		composeData, err := assets.ReadFile("assets/compose.yaml")
		if err != nil {
			return err
		}

		err = os.WriteFile(
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
