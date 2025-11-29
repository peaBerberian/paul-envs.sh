package files

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
func (f *FileStore) DeleteDataDirectory() error {
	return os.RemoveAll(f.baseDataDir)
}

// Completely delete the "config" directory which stores all current project
// configurations.
//
// Doing that will remove the `paul-envs` configuration
func (f *FileStore) DeleteConfigDirectory() error {
	return os.RemoveAll(f.baseConfigDir)
}

// Delete files associated to the named project.
func (f *FileStore) DeleteProjectDirectory(name string) error {
	return os.RemoveAll(f.getProjectDir(name))
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
// TODO: make private
func (f *FileStore) GetProjectDirBase() string {
	return f.projectsDir
}

// Ensure the "dotfiles" directory in paul-envs' config directory is created and
// return its path so you can advertise it to the user.
func (f *FileStore) InitGlobalDotfilesDir() (string, error) {
	if err := f.userFS.MkdirAsUser(f.dotfilesDir, 0755); err != nil {
		return "", fmt.Errorf("create base config directory: %w", err)
	}
	return f.dotfilesDir, nil
}

// Copy the content of the "dotfiles" directory of paul-envs to the given
// `destDir` path.
//
// Returns the path of the created project-specific dotfiles dir (should be
// removed) when finished, or an error if it failed.
func (f *FileStore) CreateDotfilesDirFor(ctx context.Context, projectName string) (string, error) {
	destDir := filepath.Join(f.getProjectDir(projectName), "nextdotfiles")
	if err := os.RemoveAll(destDir); err != nil {
		return "", fmt.Errorf("cannot copy dotfiles because %s cannot be removed: %w", destDir, err)
	}
	if err := f.userFS.CopyDirAsUser(ctx, f.dotfilesDir, destDir); err != nil {
		return "", fmt.Errorf("cannot copy dotfiles to '%s': %w", destDir, err)
	}
	return destDir, nil
}

// Get path to the given project's compose file.
func (f *FileStore) GetComposeFilePathFor(name string) string {
	return filepath.Join(f.projectsDir, name, projectComposeFilename)
}

// Get path to the given project's .env file.
func (f *FileStore) GetEnvFilePathFor(name string) string {
	return filepath.Join(f.projectsDir, name, projectEnvFilename)
}

// Returns true if the given project name currently exists on disk.
func (f *FileStore) DoesProjectExist(name string) bool {
	infoFile := f.getProjectDir(name)
	if _, err := os.Stat(infoFile); os.IsNotExist(err) {
		return false
	}
	return true
}

// Get path to the 'project.info' file associated to a project.
func (f *FileStore) getProjectInfoFilePathFor(projectName string) string {
	return filepath.Join(f.projectsDir, projectName, projectInfoFilename)
}

// Get directory where a specific project's files will be put.
func (f *FileStore) getProjectDir(name string) string {
	return filepath.Join(f.projectsDir, name)
}
