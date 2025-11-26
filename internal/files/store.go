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
	if err := os.MkdirAll(userConfigDir, 0755); err != nil {
		return nil, fmt.Errorf("create base config directory: %w", err)
	}
	return &FileStore{
		baseDataDir: userDataDir,
		dotfilesDir: filepath.Join(userConfigDir, "dotfiles"),
		projectsDir: filepath.Join(userDataDir, "projects"),
	}, nil
}

// File path helpers
func (f *FileStore) RemoveBaseDataDirectory() error {
	return os.RemoveAll(f.baseDataDir)
}

func (f *FileStore) RemoveConfigDirectory() error {
	return os.RemoveAll(f.dotfilesDir)
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
