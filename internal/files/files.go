package files

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"text/template"

	"github.com/peaberberian/paul-envs/internal/console"
)

//go:embed assets/*
var assets embed.FS

const (
	BaseComposeFilename       = "compose.yaml"
	ProjectComposeFilename    = "compose.yaml"
	ProjectDockerfileFilename = "Dockerfile"
	ProjectEnvFilename        = ".env"
)

type FileStore struct {
	baseDataDir string
	dotfilesDir string
	projectsDir string
}

func NewFileStore() (*FileStore, error) {
	userDataDir, err := getUserDataDir()
	if err != nil {
		return nil, err
	}
	userConfigDir, err := getUserConfigDir()
	if err != nil {
		return nil, err
	}
	return &FileStore{
		baseDataDir: userDataDir,
		dotfilesDir: filepath.Join(userConfigDir, "dotfiles"),
		projectsDir: filepath.Join(userDataDir, "projects"),
	}, nil
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

// File path helpers
func (f *FileStore) GetBaseComposeFile() string {
	return filepath.Join(f.baseDataDir, BaseComposeFilename)
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

func resolveRealUserHome() (string, error) {
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser != "" && sudoUser != "root" {
		u, err := user.Lookup(sudoUser)
		if err != nil {
			return "", fmt.Errorf("failed to lookup sudo user %s: %w", sudoUser, err)
		}
		return u.HomeDir, nil
	}

	// Not running under sudo or actual root user
	home := os.Getenv("HOME")
	if home == "" {
		return "", errors.New("cannot determine HOME: HOME not set")
	}
	return home, nil
}

func getUserDataDir() (string, error) {
	var baseDataDir string

	switch runtime.GOOS {
	case "windows":
		baseDataDir = os.Getenv("APPDATA")
		if baseDataDir == "" {
			baseDataDir = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
		}
	case "darwin":
		baseDataDir = filepath.Join(os.Getenv("HOME"), "Library", "Application Support")
	default: // linux / unix
		homeDir, err := resolveRealUserHome()
		if err != nil {
			return "", err
		}
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" && os.Getenv("SUDO_USER") == "" {
			baseDataDir = xdg
		} else {
			// sudo don't preserve the original XDG_DATA_HOME
			// So we just use the default XDG location
			baseDataDir = filepath.Join(homeDir, ".local", "share")
		}
	}
	return filepath.Join(baseDataDir, "paul-envs"), nil
}

func getUserConfigDir() (string, error) {
	var baseDataDir string

	switch runtime.GOOS {
	case "windows":
		// Windows: %APPDATA%
		baseDataDir = os.Getenv("APPDATA")
		if baseDataDir == "" {
			baseDataDir = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}

	case "darwin":
		home := os.Getenv("HOME")
		if home == "" {
			return "", errors.New("cannot determine HOME on macOS")
		}
		baseDataDir = filepath.Join(home, "Library", "Preferences")

	default: // Linux / Unix
		homeDir, err := resolveRealUserHome()
		if err != nil {
			return "", err
		}
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" && os.Getenv("SUDO_USER") == "" {
			baseDataDir = xdg
		} else {
			// sudo don't preserve the original XDG_CONFIG_HOME
			// So we just use the default XDG location
			baseDataDir = filepath.Join(homeDir, ".config")
		}
	}
	return filepath.Join(baseDataDir, "paul-envs"), nil
}
