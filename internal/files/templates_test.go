package files

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileStore_CreateProjectEnvFile(t *testing.T) {
	baseDataDir := t.TempDir()
	dotfilesDir := t.TempDir()
	store := &FileStore{
		baseDataDir: baseDataDir,
		dotfilesDir: dotfilesDir,
		projectsDir: filepath.Join(baseDataDir, "projects"),
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
	baseDataDir := t.TempDir()
	dotfilesDir := t.TempDir()
	store := &FileStore{
		baseDataDir: baseDataDir,
		dotfilesDir: dotfilesDir,
		projectsDir: filepath.Join(baseDataDir, "projects"),
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
	baseDataDir := t.TempDir()
	dotfilesDir := t.TempDir()
	store := &FileStore{
		baseDataDir: baseDataDir,
		dotfilesDir: dotfilesDir,
		projectsDir: filepath.Join(baseDataDir, "projects"),
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
	baseDataDir := t.TempDir()
	dotfilesDir := t.TempDir()
	store := &FileStore{
		baseDataDir: baseDataDir,
		dotfilesDir: dotfilesDir,
		projectsDir: filepath.Join(baseDataDir, "projects"),
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
