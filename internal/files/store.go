package files

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/peaberberian/paul-envs/internal/console"
)

const (
	BaseComposeFilename       = "compose.yaml"
	ProjectComposeFilename    = "compose.yaml"
	ProjectDockerfileFilename = "Dockerfile"
	ProjectEnvFilename        = ".env"
)

type FileStore struct {
	userFS        *UserFS
	baseDataDir   string
	baseConfigDir string
	dotfilesDir   string
	projectsDir   string
}

func NewFileStore() (*FileStore, error) {
	userFS, err := NewUserFS()
	if err != nil {
		return nil, err
	}

	paulEnvsDataDir := filepath.Join(userFS.GetUserDataDir(), "paul-envs")
	paulEnvsConfigDir := filepath.Join(userFS.GetUserConfigDir(), "paul-envs")
	dotfilesDir := filepath.Join(paulEnvsConfigDir, "dotfiles")

	if err := userFS.MkdirAsUser(dotfilesDir, 0755); err != nil {
		return nil, fmt.Errorf("create base config directory: %w", err)
	}

	if err := userFS.MkdirAsUser(filepath.Join(paulEnvsDataDir, "placeholder"), 0755); err != nil {
		return nil, fmt.Errorf("create empty dotfiles placeholder: %w", err)
	}

	return &FileStore{
		userFS:        userFS,
		baseDataDir:   paulEnvsDataDir,
		baseConfigDir: paulEnvsConfigDir,
		dotfilesDir:   filepath.Join(paulEnvsConfigDir, "dotfiles"),
		projectsDir:   filepath.Join(paulEnvsDataDir, "projects"),
	}, nil
}

// File path helpers
func (f *FileStore) RemoveBaseDataDirectory() error {
	return os.RemoveAll(f.baseDataDir)
}

func (f *FileStore) RemoveConfigDirectory() error {
	return os.RemoveAll(f.baseConfigDir)
}

func (f *FileStore) GetBaseComposeFile() string {
	return filepath.Join(f.baseDataDir, BaseComposeFilename)
}

func (f *FileStore) GetDotfileDirBase() string {
	return f.dotfilesDir
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
