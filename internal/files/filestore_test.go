package files

import (
	"testing"
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
	if store.baseConfigDir == "" {
		t.Error("baseConfigDir should not be empty")
	}
	if store.dotfilesDir == "" {
		t.Error("dotfilesDir should not be empty")
	}
	if store.projectsDir == "" {
		t.Error("projectsDir should not be empty")
	}
}

func TestFileStore_PathHelpers(t *testing.T) {
	store := &FileStore{
		baseDataDir:   "/test/base",
		baseConfigDir: "/test/config",
		dotfilesDir:   "/test/config/dotfiles",
		projectsDir:   "/test/base/projects",
	}

	tests := []struct {
		name     string
		fn       func() string
		expected string
	}{
		{
			name:     "GetBaseComposeFile",
			fn:       store.GetBaseComposeFilePath,
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
		baseDataDir:   "/test/base",
		baseConfigDir: "/test/config",
		dotfilesDir:   "/test/config/dotfiles",
		projectsDir:   "/test/base/projects",
	}

	got := store.getProjectDir("myproject")
	expected := "/test/base/projects/myproject"
	if got != expected {
		t.Errorf("GetProjectDir() = %v, want %v", got, expected)
	}
}

func TestFileStore_GetComposeFilePathFor(t *testing.T) {
	store := &FileStore{
		baseDataDir:   "/test/base",
		baseConfigDir: "/test/config",
		dotfilesDir:   "/test/config/dotfiles",
		projectsDir:   "/test/base/projects",
	}

	got := store.GetComposeFilePathFor("myproject")
	expected := "/test/base/projects/myproject/compose.yaml"
	if got != expected {
		t.Errorf("GetComposeFilePathFor() = %v, want %v", got, expected)
	}
}

func TestFileStore_GetEnvFilePathFor(t *testing.T) {
	store := &FileStore{
		baseDataDir:   "/test/base",
		baseConfigDir: "/test/config",
		dotfilesDir:   "/test/config/dotfiles",
		projectsDir:   "/test/base/projects",
	}

	got := store.GetEnvFilePathFor("myproject")
	expected := "/test/base/projects/myproject/.env"
	if got != expected {
		t.Errorf("GetEnvFilePathFor() = %v, want %v", got, expected)
	}
}
