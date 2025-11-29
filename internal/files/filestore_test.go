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
	if store.projectsDir == "" {
		t.Error("projectsDir should not be empty")
	}
}

func TestFileStore_PathHelpers(t *testing.T) {
	store := &FileStore{
		baseDataDir:   "/test/base",
		baseConfigDir: "/test/config",
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
			name:     "getProjectDirBase",
			fn:       store.getProjectDirBase,
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
		projectsDir:   "/test/base/projects",
	}

	got := store.getProjectDir("myproject")
	expected := "/test/base/projects/myproject"
	if got != expected {
		t.Errorf("GetProjectDir() = %v, want %v", got, expected)
	}
}

func TestFileStore_GetProjectComposeFilePath(t *testing.T) {
	store := &FileStore{
		baseDataDir:   "/test/base",
		baseConfigDir: "/test/config",
		projectsDir:   "/test/base/projects",
	}

	got := store.GetProjectComposeFilePath("myproject")
	expected := "/test/base/projects/myproject/compose.yaml"
	if got != expected {
		t.Errorf("GetProjectComposeFilePath() = %v, want %v", got, expected)
	}
}

func TestFileStore_GetProjectEnvFilePath(t *testing.T) {
	store := &FileStore{
		baseDataDir:   "/test/base",
		baseConfigDir: "/test/config",
		projectsDir:   "/test/base/projects",
	}

	got := store.GetProjectEnvFilePath("myproject")
	expected := "/test/base/projects/myproject/.env"
	if got != expected {
		t.Errorf("GetProjectEnvFilePath() = %v, want %v", got, expected)
	}
}
