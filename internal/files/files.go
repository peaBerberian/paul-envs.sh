package files

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/peaberberian/paul-envs/internal/console"
)

//go:embed templates/*
var templates embed.FS

const (
	BaseComposeFilename    = "compose.yaml"
	ProjectComposeFilename = "compose.yaml"
	ProjectEnvFilename     = ".env"
)

type FileStore struct {
	baseDir     string
	projectsDir string
}

func NewFileStore() (*FileStore, error) {
	// Get binary directory
	ex, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("get executable path: %w", err)
	}

	// TODO: Rely on XDG_DATA_HOME and equivalents (Linux, MacOS, Windows)
	return &FileStore{
		baseDir:     filepath.Dir(ex),
		projectsDir: filepath.Join(filepath.Dir(ex), "projects"),
	}, nil
}

// File path helpers
func (f *FileStore) GetBaseComposeFile() string {
	return filepath.Join(f.baseDir, BaseComposeFilename)
}

func (f *FileStore) GetProjectDirBase() string {
	return f.projectsDir
}

func (f *FileStore) GetProjectDir(name string) string {
	return filepath.Join(f.projectsDir, name)
}

func (f *FileStore) GetComposeFilePathFor(name string) string {
	return filepath.Join(f.projectsDir, name, ProjectComposeFilename)
}

func (f *FileStore) GetEnvFilePathFor(name string) string {
	return filepath.Join(f.projectsDir, name, ProjectEnvFilename)
}

func (f *FileStore) CheckProjectNameAvailable(name string, console *console.Console) error {
	composeFile := f.GetComposeFilePathFor(name)
	envFile := f.GetEnvFilePathFor(name)

	if _, err := os.Stat(composeFile); err == nil {
		console.Warn("Project '%s' already exists. You can have multiple configurations for the same project by calling 'create' with the '--name' flag. Hint: Use 'paul-envs list' to see all projects or 'paul-envs remove %s' to delete it", name, name)
		return fmt.Errorf("project %s already exists", name)
	}
	if _, err := os.Stat(envFile); err == nil {
		console.Warn("Project '%s' already exists. You can have multiple configurations for the same project by calling 'create' with the '--name' flag. Hint: Use 'paul-envs list' to see all projects or 'paul-envs remove %s' to delete it", name, name)
		return fmt.Errorf("project %s already exists", name)
	}
	return nil
}

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

	tmplContent, err := templates.ReadFile("templates/env.tmpl")
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

	tmplContent, err := templates.ReadFile("templates/compose.tmpl")
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
