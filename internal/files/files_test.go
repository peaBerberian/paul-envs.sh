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
	if store.baseDir == "" {
		t.Error("baseDir should not be empty")
	}
	if store.projectsDir == "" {
		t.Error("projectsDir should not be empty")
	}
}

func TestFileStore_PathHelpers(t *testing.T) {
	store := &FileStore{
		baseDir:     "/test/base",
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
		baseDir:     "/test/base",
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
		baseDir:     "/test/base",
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
		baseDir:     "/test/base",
		projectsDir: "/test/base/projects",
	}

	got := store.GetEnvFilePathFor("myproject")
	expected := "/test/base/projects/myproject/.env"
	if got != expected {
		t.Errorf("GetEnvFilePathFor() = %v, want %v", got, expected)
	}
}

func TestFileStore_CheckProjectNameAvailable(t *testing.T) {
	tempDir := t.TempDir()
	store := &FileStore{
		baseDir:     tempDir,
		projectsDir: filepath.Join(tempDir, "projects"),
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

func TestFileStore_CreateProjectEnvFile(t *testing.T) {
	tempDir := t.TempDir()
	store := &FileStore{
		baseDir:     tempDir,
		projectsDir: filepath.Join(tempDir, "projects"),
	}

	tplData := EnvTemplateData{
		ProjectComposeFilename: "compose.yaml",
		ProjectID:              "test-id",
		ProjectDestPath:        "myproject",
		ProjectHostPath:        "/host/path",
		HostUID:                "1000",
		HostGID:                "1000",
		Username:               "testuser",
		Shell:                  "bash",
		InstallNode:            "latest",
		InstallRust:            "none",
		InstallPython:          "3.12.0",
		InstallGo:              "none",
		EnableWasm:             "false",
		EnableSSH:              "true",
		EnableSudo:             "true",
		Packages:               "git vim",
		InstallNeovim:          "true",
		InstallStarship:        "true",
		InstallAtuin:           "false",
		InstallMise:            "true",
		InstallZellij:          "false",
		InstallJujutsu:         "false",
		GitName:                "Test User",
		GitEmail:               "test@example.com",
	}

	err := store.CreateProjectEnvFile("testproject", tplData)
	if err != nil {
		t.Fatalf("CreateProjectEnvFile() error = %v", err)
	}

	// Verify file exists
	envFile := store.GetEnvFilePathFor("testproject")
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		t.Fatal("env file was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)
	checks := []string{
		`PROJECT_ID="test-id"`,
		`PROJECT_DIRNAME="myproject"`,
		`PROJECT_PATH="/host/path"`,
		`HOST_UID="1000"`,
		`USERNAME="testuser"`,
		`USER_SHELL="bash"`,
		`INSTALL_NODE="latest"`,
		`GIT_AUTHOR_NAME="Test User"`,
		`GIT_AUTHOR_EMAIL="test@example.com"`,
	}

	for _, check := range checks {
		if !strings.Contains(contentStr, check) {
			t.Errorf("env file missing expected content: %s", check)
		}
	}
}

func TestFileStore_CreateProjectComposeFile(t *testing.T) {
	tempDir := t.TempDir()
	store := &FileStore{
		baseDir:     tempDir,
		projectsDir: filepath.Join(tempDir, "projects"),
	}

	tplData := ComposeTemplateData{
		ProjectName: "testproject",
		Ports:       []uint16{3000, 8080},
		EnableSSH:   true,
		SSHKeyPath:  "/home/user/.ssh/id_ed25519.pub",
		Volumes:     []string{"./data:/app/data", "./config:/app/config"},
	}

	err := store.CreateProjectComposeFile("testproject", tplData)
	if err != nil {
		t.Fatalf("CreateProjectComposeFile() error = %v", err)
	}

	// Verify file exists
	composeFile := store.GetComposeFilePathFor("testproject")
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		t.Fatal("compose file was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(composeFile)
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)
	checks := []string{
		`image: paulenv:testproject`,
		`"3000:3000"`,
		`"8080:8080"`,
		`"22:22"`,
		`./data:/app/data`,
		`./config:/app/config`,
		`/home/user/.ssh/id_ed25519.pub:/etc/ssh/authorized_keys/${USERNAME}:ro`,
	}

	for _, check := range checks {
		if !strings.Contains(contentStr, check) {
			t.Errorf("compose file missing expected content: %s", check)
		}
	}
}

func TestFileStore_CreateProjectComposeFile_NoSSH(t *testing.T) {
	tempDir := t.TempDir()
	store := &FileStore{
		baseDir:     tempDir,
		projectsDir: filepath.Join(tempDir, "projects"),
	}

	tplData := ComposeTemplateData{
		ProjectName: "testproject",
		Ports:       []uint16{3000},
		EnableSSH:   false,
		Volumes:     []string{"./data:/app/data"},
	}

	err := store.CreateProjectComposeFile("testproject", tplData)
	if err != nil {
		t.Fatalf("CreateProjectComposeFile() error = %v", err)
	}

	content, err := os.ReadFile(store.GetComposeFilePathFor("testproject"))
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)
	if strings.Contains(contentStr, `"22:22"`) {
		t.Error("compose file should not contain SSH port when disabled")
	}
	if strings.Contains(contentStr, "authorized_keys") {
		t.Error("compose file should not contain SSH key mount when disabled")
	}
}

func TestFileStore_ensureProjectDir(t *testing.T) {
	tempDir := t.TempDir()
	store := &FileStore{
		baseDir:     tempDir,
		projectsDir: filepath.Join(tempDir, "projects"),
	}

	testFile := filepath.Join(store.projectsDir, "newproject", "test.txt")
	err := store.ensureProjectDir(testFile)
	if err != nil {
		t.Fatalf("ensureProjectDir() error = %v", err)
	}

	// Verify directories were created
	if _, err := os.Stat(store.projectsDir); os.IsNotExist(err) {
		t.Error("base projects directory was not created")
	}
	if _, err := os.Stat(filepath.Dir(testFile)); os.IsNotExist(err) {
		t.Error("project directory was not created")
	}
}
