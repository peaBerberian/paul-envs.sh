package files

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/peaberberian/paul-envs/internal/console"
)

const (
	baseComposeFilename    = "compose.yaml"
	projectComposeFilename = "compose.yaml"
	projectEnvFilename     = ".env"
	projectInfoFilename    = "project.info"
)

type FileStore struct {
	userFS        *UserFS
	baseDataDir   string
	baseConfigDir string
	dotfilesDir   string
	projectsDir   string
}

func NewFileStore() (*FileStore, error) {
	// TODO: Set lazily?
	userFS, err := NewUserFS()
	if err != nil {
		return nil, err
	}

	paulEnvsDataDir := filepath.Join(userFS.GetUserDataDir(), "paul-envs")
	paulEnvsConfigDir := filepath.Join(userFS.GetUserConfigDir(), "paul-envs")

	// TODO: lazily would be best
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

// Completely delete the "data" directory which stores all current project
// configurations.
//
// Doing that will remove all project configuration files.
func (f *FileStore) DeleteBaseDataDirectory() error {
	return os.RemoveAll(f.baseDataDir)
}

// Completely delete the "config" directory which stores all current project
// configurations.
//
// Doing that will remove the `paul-envs` configuration
func (f *FileStore) DeleteConfigDirectory() error {
	return os.RemoveAll(f.baseConfigDir)
}

// Get the path to the "base" compose.yaml file on which all projects depend on.
//
// /!\ The file might not be yet created. It will be created automatically by
// the FileStore once the first project is created.
func (f *FileStore) GetBaseComposeFilePath() string {
	return filepath.Join(f.baseDataDir, baseComposeFilename)
}

// Get the path to where all projects config will be put.
//
// This is only for information matters, the FileStore should take care of
// creating all files inside.
func (f *FileStore) GetProjectDirBase() string {
	return f.projectsDir
}

// Create the "dotfiles" directory in paul-envs' config directory.
//
// You can advertise this directory to the user, reading it through
// `GetDotfilesDirBase`.
func (f *FileStore) CreateDotfilesDirBase() error {
	if err := f.userFS.MkdirAsUser(f.dotfilesDir, 0755); err != nil {
		return fmt.Errorf("create base config directory: %w", err)
	}
	return nil
}

// Return the path to the "dotfiles" directory of paul-envs, where the dotfiles
// to port to all containers should be put.
//
// /!\ The directory might not be yet created
func (f *FileStore) GetDotfilesDirBase() string {
	return f.dotfilesDir
}

// Copy the content of the "dotfiles" directory of paul-envs to the given
// `destDir` path.
func (f *FileStore) CopyDotfilesTo(ctx context.Context, destDir string) error {
	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("cannot copy dotfiles because %s cannot be removed: %w", destDir, err)
	}
	return f.userFS.CopyDirAsUser(ctx, f.dotfilesDir, destDir)
}

// Get directory where a specific project's files will be put.
//
// TODO: make private
func (f *FileStore) GetProjectDir(name string) string {
	return filepath.Join(f.projectsDir, name)
}

// Get path to the given project's compose file.
func (f *FileStore) GetComposeFilePathFor(name string) string {
	return filepath.Join(f.projectsDir, name, projectComposeFilename)
}

// Get path to the given project's .env file.
func (f *FileStore) GetEnvFilePathFor(name string) string {
	return filepath.Join(f.projectsDir, name, projectEnvFilename)
}

// Check that the given project name is not taken.
// If it is, warn the user through the given console and return an error.
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

// Get path to the 'project.info' file associated to a project.
func (f *FileStore) getProjectInfoFilePathFor(projectName string) string {
	return filepath.Join(f.projectsDir, projectName, projectInfoFilename)
}
