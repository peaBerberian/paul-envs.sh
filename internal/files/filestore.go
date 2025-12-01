package files

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

const (
	projectComposeFilename = "compose.yaml"
	projectEnvFilename     = ".env"
	projectInfoFilename    = "project.lock"
)

// Struct allowing to create, read and obtain the path of all files created by
// this project:
// -  project-specific configuration
// -  base Dockerfile file
// -  the user's dotfiles directory
type FileStore struct {
	userFS        *UserFS
	baseDataDir   string
	baseConfigDir string
	projectsDir   string
}

// Information linked to a given project.
type ProjectEntry struct {
	// The "name" by which it is refered to in `paul-envs`
	ProjectName string
	// The path on the host to the mounted directory which corresponds to the
	// "project" of that container (as indicated in the `.env` file).
	// TODO: should this one be behind a method?
	ProjectPath string
	// `compose.yaml` file associated to this project.
	ComposeFilePath string
	// `.env` file associated to this project.
	EnvFilePath string
	// TODO: Last built / last run?
}

// Create a new `FileStore`.
// Returns an `error` if we didn't succeed to obtain the needed filesystem
// information.
func NewFileStore() (*FileStore, error) {
	userFS, err := NewUserFS()
	if err != nil {
		return nil, err
	}
	paulEnvsDataDir := filepath.Join(userFS.GetUserDataDir(), "paul-envs")
	paulEnvsConfigDir := filepath.Join(userFS.GetUserConfigDir(), "paul-envs")
	return &FileStore{
		userFS:        userFS,
		baseDataDir:   paulEnvsDataDir,
		baseConfigDir: paulEnvsConfigDir,
		projectsDir:   filepath.Join(paulEnvsDataDir, "projects"),
	}, nil
}

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

// Returns true if the given project name currently exists on disk.
func (f *FileStore) DoesProjectExist(name string) bool {
	infoFile := f.getProjectDir(name)
	if _, err := os.Stat(infoFile); os.IsNotExist(err) {
		return false
	}
	return true
}

// Get the project information behind the given project name
func (f *FileStore) GetProject(name string) (ProjectEntry, error) {
	projectPath, err := f.parseProjectPath(name)
	if err != nil {
		if !f.DoesProjectExist(name) {
			return ProjectEntry{}, fmt.Errorf("project '%s' does not exist", name)
		}
		return ProjectEntry{}, err
	}
	return ProjectEntry{
		ProjectName:     name,
		ProjectPath:     projectPath,
		ComposeFilePath: f.GetProjectComposeFilePath(name),
		EnvFilePath:     f.GetProjectEnvFilePath(name),
	}, nil
}

// Get a list of `ProjectEntry` struct, each describing a single project whose
// configuration has been created.
func (f *FileStore) GetAllProjects() ([]ProjectEntry, error) {
	dirBase := f.getProjectDirBase()
	if _, err := os.Stat(dirBase); os.IsNotExist(err) {
		return []ProjectEntry{}, nil
	}

	dirs, err := os.ReadDir(dirBase)
	if err != nil {
		return []ProjectEntry{}, fmt.Errorf("reading project directory failed: %w", err)
	}

	entries := make([]ProjectEntry, 0, len(dirs))
	for _, entry := range dirs {
		if !entry.IsDir() {
			continue
		}
		project, err := f.GetProject(entry.Name())
		if err != nil {
			return []ProjectEntry{}, err
		}
		entries = append(entries, project)
	}
	return entries, nil
}

// Ensure the "dotfiles" directory in paul-envs' config directory is created and
// return its path so you can advertise it to the user.
func (f *FileStore) InitGlobalDotfilesDir() (string, error) {
	dotfilesDir := filepath.Join(f.baseConfigDir, "dotfiles")
	if err := f.userFS.MkdirAsUser(dotfilesDir, 0755); err != nil {
		return "", fmt.Errorf("create base config directory: %w", err)
	}
	return dotfilesDir, nil
}

// Copy the content of the "dotfiles" directory of paul-envs to the given
// `destDir` path.
//
// Returns the relative path (from the Dockerfile base) of the created
// project-specific dotfiles dir (should be removed) when finished, or an error
// if it failed.
func (f *FileStore) CreateProjectDotfilesDir(ctx context.Context, projectName string) (string, error) {
	if !f.DoesProjectExist(projectName) {
		return "", fmt.Errorf("cannot copy dotfiles for project '%s': this project does not exist", projectName)
	}
	destDir := filepath.Join(f.getProjectDir(projectName), "nextdotfiles")
	if err := os.RemoveAll(destDir); err != nil {
		return "", fmt.Errorf("cannot copy dotfiles because %s cannot be removed: %w", destDir, err)
	}
	dotfilesDir, err := f.InitGlobalDotfilesDir()
	if err != nil {
		return "", err
	}
	if err := f.userFS.CopyDirAsUser(ctx, dotfilesDir, destDir); err != nil {
		return "", fmt.Errorf("cannot copy dotfiles to '%s': %w", destDir, err)
	}
	relativeDotfilesDir, err := filepath.Rel(f.baseDataDir, destDir)
	if err != nil {
		return "", fmt.Errorf("failed to construct dotfiles relative path: %w", err)
	}

	return relativeDotfilesDir, nil
}
func (f *FileStore) RemoveProjectDotfilesDir(projectName string) error {
	expectedDir := filepath.Join(f.getProjectDir(projectName), "nextdotfiles")
	return os.RemoveAll(expectedDir)
}

// Get path to the given project's compose file.
// TODO: make private
func (f *FileStore) GetProjectComposeFilePath(name string) string {
	return filepath.Join(f.projectsDir, name, projectComposeFilename)
}

// Get path to the given project's .env file.
// TODO: make private
func (f *FileStore) GetProjectEnvFilePath(name string) string {
	return filepath.Join(f.projectsDir, name, projectEnvFilename)
}

// Get the path to where all projects config will be put.
//
// This is only for information matters, the FileStore should take care of
// creating all files inside.
func (f *FileStore) getProjectDirBase() string {
	return f.projectsDir
}

// Get path to the 'project.lock' file associated to a project.
func (f *FileStore) getProjectInfoFilePathFor(projectName string) string {
	return filepath.Join(f.projectsDir, projectName, projectInfoFilename)
}

// Get directory where a specific project's files will be put.
func (f *FileStore) getProjectDir(name string) string {
	return filepath.Join(f.projectsDir, name)
}

// Returns the mounted host directory associated with the project name given.
// Returns an `error` if that data could not have been obtained.
func (f *FileStore) parseProjectPath(name string) (string, error) {
	envFile := f.GetProjectEnvFilePath(name)
	file, err := os.Open(envFile)
	if err != nil {
		return "", fmt.Errorf("could not open .env file associated to project '%s': %w", name, err)
	}
	defer file.Close()

	re := regexp.MustCompile(`PROJECT_PATH="([^"]*)"`)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if matches := re.FindStringSubmatch(scanner.Text()); len(matches) == 2 {
			return matches[1], nil
		}
	}
	return "", fmt.Errorf("did not found the project path associated to project '%s'", name)
}
