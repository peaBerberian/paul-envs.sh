package files

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	constants "github.com/peaberberian/paul-envs/internal"
)

func TestFileStore_CreateProjectFiles(t *testing.T) {
	baseDataDir := t.TempDir()
	configDir := t.TempDir()
	store := &FileStore{
		userFS: &UserFS{
			homeDir:  t.TempDir(),
			sudoUser: nil,
		},
		baseDataDir:   baseDataDir,
		baseConfigDir: configDir,
		projectsDir:   filepath.Join(baseDataDir, "projects"),
	}

	envTplData := EnvTemplateData{
		ProjectID:       "test-id",
		ProjectDestPath: "myproject",
		ProjectHostPath: "/host/path",
		HostUID:         "1000",
		HostGID:         "1000",
		Username:        "testuser",
		Shell:           "bash",
		InstallNode:     "latest",
		InstallRust:     "none",
		InstallPython:   "3.12.0",
		InstallGo:       "none",
		EnableWasm:      "false",
		EnableSSH:       "true",
		EnableSudo:      "true",
		Packages:        "git vim",
		InstallNeovim:   "true",
		InstallStarship: "true",
		InstallAtuin:    "false",
		InstallMise:     "true",
		InstallZellij:   "false",
		InstallJujutsu:  "false",
		GitName:         "Test User",
		GitEmail:        "test@example.com",
	}

	composeTplData := ComposeTemplateData{
		ProjectName: "testproject",
		Ports:       []uint16{3000, 8080},
		EnableSSH:   true,
		SSHKeyPath:  "/home/user/.ssh/id_ed25519.pub",
		Volumes:     []string{"./data:/app/data", "./config:/app/config"},
	}

	err := store.CreateProjectFiles("testproject", envTplData, composeTplData)
	if err != nil {
		t.Fatalf("CreateProjectFiles() error = %v", err)
	}

	// Verify files exists
	envFile := store.GetProjectEnvFilePath("testproject")
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		t.Fatal("env file was not created")
	}
	composeFile := store.GetProjectComposeFilePath("testproject")
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		t.Fatal("compose file was not created")
	}

	projectInfoFile := store.getProjectInfoFilePathFor("testproject")
	if _, err := os.Stat(projectInfoFile); os.IsNotExist(err) {
		t.Fatal("project.info file was not created")
	}

	// Read and verify content

	envCtnt, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatal(err)
	}

	envCtntStr := string(envCtnt)
	envChecks := []string{
		`PROJECT_ID="test-id"`,
		`PROJECT_DIRNAME="myproject"`,
		`PROJECT_PATH="/host/path"`,
		`HOST_UID="1000"`,
		`USERNAME="testuser"`,
		`USER_SHELL="bash"`,
		`INSTALL_NODE="latest"`,
		`GIT_AUTHOR_NAME="Test User"`,
		`GIT_AUTHOR_EMAIL="test@example.com"`,
		`ENABLE_SSH="true"`,
	}

	for _, check := range envChecks {
		if !strings.Contains(envCtntStr, check) {
			t.Errorf("env file missing expected content: %s", check)
		}
	}

	// Read and verify content
	composeCtnt, err := os.ReadFile(composeFile)
	if err != nil {
		t.Fatal(err)
	}

	composeCtntString := string(composeCtnt)
	composeChecks := []string{
		`image: paulenv:testproject`,
		`"3000:3000"`,
		`"8080:8080"`,
		`"22:22"`,
		`./data:/app/data`,
		`./config:/app/config`,
		`/home/user/.ssh/id_ed25519.pub:/etc/ssh/authorized_keys/${USERNAME}:ro`,
	}

	for _, check := range composeChecks {
		if !strings.Contains(composeCtntString, check) {
			t.Errorf("compose file missing expected content: %s", check)
		}
	}

	pInfoCtnt, err := os.ReadFile(projectInfoFile)
	if err != nil {
		t.Fatal(err)
	}

	pInfoCtntStr := string(pInfoCtnt)
	pInfoChecks := []string{
		`VERSION=` + constants.FileVersion,
		`DOCKERFILE_VERSION=` + constants.FileVersion,
		`BUILD_ENV=`,
		`BUILD_COMPOSE`,
	}

	for _, check := range pInfoChecks {
		if !strings.Contains(pInfoCtntStr, check) {
			t.Errorf("project.info file missing expected content: %s", check)
		}
	}
}

func TestFileStore_CreateProjectComposeFiles(t *testing.T) {
	baseDataDir := t.TempDir()
	configDir := t.TempDir()
	store := &FileStore{
		userFS: &UserFS{
			homeDir:  t.TempDir(),
			sudoUser: nil,
		},
		baseDataDir:   baseDataDir,
		baseConfigDir: configDir,
		projectsDir:   filepath.Join(baseDataDir, "projects"),
	}

	envTplData := EnvTemplateData{
		ProjectID:       "test-id",
		ProjectDestPath: "myproject",
		ProjectHostPath: "/host/path",
		HostUID:         "1000",
		HostGID:         "1000",
		Username:        "testuser",
		Shell:           "bash",
		InstallNode:     "latest",
		InstallRust:     "none",
		InstallPython:   "3.12.0",
		InstallGo:       "none",
		EnableWasm:      "false",
		EnableSSH:       "false",
		EnableSudo:      "true",
		Packages:        "git vim",
		InstallNeovim:   "true",
		InstallStarship: "true",
		InstallAtuin:    "false",
		InstallMise:     "true",
		InstallZellij:   "false",
		InstallJujutsu:  "false",
		GitName:         "Test User",
		GitEmail:        "test@example.com",
	}

	composeTplData := ComposeTemplateData{
		ProjectName: "testproject",
		Ports:       []uint16{3000},
		EnableSSH:   false,
		Volumes:     []string{"./data:/app/data"},
	}

	err := store.CreateProjectFiles("testproject", envTplData, composeTplData)
	if err != nil {
		t.Fatalf("CreateProjectFiles() error = %v", err)
	}

	envCtnt, err := os.ReadFile(store.GetProjectEnvFilePath("testproject"))
	if err != nil {
		t.Fatal(err)
	}

	envCtntStr := string(envCtnt)
	envChecks := []string{
		`ENABLE_SSH="false"`,
	}
	for _, check := range envChecks {
		if !strings.Contains(envCtntStr, check) {
			t.Errorf("env file should not enable SSH: %s", check)
		}
	}

	composeCtnt, err := os.ReadFile(store.GetProjectComposeFilePath("testproject"))
	if err != nil {
		t.Fatal(err)
	}

	composeCtntStr := string(composeCtnt)
	if strings.Contains(composeCtntStr, `"22:22"`) {
		t.Error("compose file should not contain SSH port when disabled")
	}
	if strings.Contains(composeCtntStr, "authorized_keys") {
		t.Error("compose file should not contain SSH key mount when disabled")
	}
}
