package files

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/peaberberian/paul-envs/internal/console"
)

func TestNewFileStore(t *testing.T) {
	store, err := NewFileStore()
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	if store == nil {
		t.Fatal("NewFileStore() returned nil")
	}
	if store.baseDataDir == "" {
		t.Error("baseDataDir should not be empty")
	}
	if store.projectsDir == "" {
		t.Error("projectsDir should not be empty")
	}
}

func TestFileStore_PathHelpers(t *testing.T) {
	store := &FileStore{
		baseDataDir: "/test/base",
		dotfilesDir: "/test/config",
		projectsDir: "/test/base/projects",
	}

	tests := []struct {
		name     string
		fn       func() string
		expected string
	}{
		{
			name:     "GetBaseComposeFile",
			fn:       store.GetBaseComposeFile,
			expected: "/test/base/compose.yaml",
		},
		{
			name:     "GetProjectDirBase",
			fn:       store.GetProjectDirBase,
			expected: "/test/base/projects",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn()
			if got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

func TestFileStore_GetProjectDir(t *testing.T) {
	store := &FileStore{
		baseDataDir: "/test/base",
		dotfilesDir: "/test/config",
		projectsDir: "/test/base/projects",
	}

	got := store.GetProjectDir("myproject")
	expected := "/test/base/projects/myproject"
	if got != expected {
		t.Errorf("GetProjectDir() = %v, want %v", got, expected)
	}
}

func TestFileStore_GetComposeFilePathFor(t *testing.T) {
	store := &FileStore{
		baseDataDir: "/test/base",
		dotfilesDir: "/test/config",
		projectsDir: "/test/base/projects",
	}

	got := store.GetComposeFilePathFor("myproject")
	expected := "/test/base/projects/myproject/compose.yaml"
	if got != expected {
		t.Errorf("GetComposeFilePathFor() = %v, want %v", got, expected)
	}
}

func TestFileStore_GetEnvFilePathFor(t *testing.T) {
	store := &FileStore{
		baseDataDir: "/test/base",
		dotfilesDir: "/test/config",
		projectsDir: "/test/base/projects",
	}

	got := store.GetEnvFilePathFor("myproject")
	expected := "/test/base/projects/myproject/.env"
	if got != expected {
		t.Errorf("GetEnvFilePathFor() = %v, want %v", got, expected)
	}
}

func TestFileStore_CheckProjectNameAvailable(t *testing.T) {
	baseDataDir := t.TempDir()
	dotfilesDir := t.TempDir()
	store := &FileStore{
		baseDataDir: baseDataDir,
		dotfilesDir: dotfilesDir,
		projectsDir: filepath.Join(baseDataDir, "projects"),
	}
	ctx := t.Context()
	cons := console.New(ctx, os.Stdin, io.Discard, io.Discard)

	t.Run("available project name", func(t *testing.T) {
		err := store.CheckProjectNameAvailable("newproject", cons)
		if err != nil {
			t.Errorf("CheckProjectNameAvailable() error = %v, want nil", err)
		}
	})

	t.Run("unavailable - compose file exists", func(t *testing.T) {
		projectName := "existing1"
		composeFile := store.GetComposeFilePathFor(projectName)
		if err := os.MkdirAll(filepath.Dir(composeFile), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(composeFile, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		err := store.CheckProjectNameAvailable(projectName, cons)
		if err == nil {
			t.Error("CheckProjectNameAvailable() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("error message should contain 'already exists', got: %v", err)
		}
	})

	t.Run("unavailable - env file exists", func(t *testing.T) {
		projectName := "existing2"
		envFile := store.GetEnvFilePathFor(projectName)
		if err := os.MkdirAll(filepath.Dir(envFile), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(envFile, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		err := store.CheckProjectNameAvailable(projectName, cons)
		if err == nil {
			t.Error("CheckProjectNameAvailable() error = nil, want error")
		}
	})
}
