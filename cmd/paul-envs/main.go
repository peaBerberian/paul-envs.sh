package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"text/template"
)

const version = "0.1.0"

const (
	// ANSI color codes
	colorRed    = "\033[0;31m"
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[1;33m"
	colorBlue   = "\033[0;34m"
	colorReset  = "\033[0m"

	shellBash = "bash"
	shellZsh  = "zsh"
	shellFish = "fish"

	versionNone   = "none"
	versionLatest = "latest"

	baseComposeFilename    = "compose.yaml"
	projectComposeFilename = "compose.yaml"
	projectEnvFilename     = ".env"
)

var (
	projectNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_-]{0,127}$`)
	versionRegex     = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
)

type ContainerConfig struct {
	hostUID         string
	hostGID         string
	username        string
	shell           string
	installNode     string
	installRust     string
	installPython   string
	installGo       string
	enableWasm      string
	enableSSH       string
	enableSudo      string
	gitName         string
	gitEmail        string
	packages        string
	installNeovim   string
	installStarship string
	installAtuin    string
	installMise     string
	installZellij   string
	installJujutsu  string
	projectHostPath string
	projectDestPath string
	sshKeyPath      string
}

type App struct {
	scriptDir   string
	projectsDir string
	io          *IoCtrl
}

type IoCtrl struct {
	reader *bufio.Reader
	writer io.Writer
}

func NewIoCtrl(rd io.Reader, w io.Writer) *IoCtrl {
	return &IoCtrl{
		reader: bufio.NewReader(rd),
		writer: w,
	}
}

func NewApp() (*App, error) {
	// Get script directory
	// TODO: Rely on something like XDG_DATA_HOME, XDG_CONFIG_HOME etc.
	ex, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	return &App{
		scriptDir:   filepath.Dir(ex),
		projectsDir: filepath.Join(filepath.Dir(ex), "projects"),
		io:          NewIoCtrl(os.Stdin, os.Stdout),
	}, nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer cancel()

	app, err := NewApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		printUsage(app)
		os.Exit(0)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var cmdErr error
	switch cmd {
	case "create":
		app.Create(args)
	case "list", "ls":
		cmdErr = app.List()
	case "build":
		cmdErr = app.Build(ctx, args)
	case "run":
		cmdErr = app.Run(ctx, args)
	case "remove", "rm":
		cmdErr = app.Remove(args)
	case "version", "--version", "-v":
		cmdErr = app.Version(ctx)
	default:
		printUsage(app)
	}

	if cmdErr != nil {
		if errors.Is(cmdErr, context.Canceled) {
			fmt.Fprintln(os.Stderr, "Operation cancelled")
			os.Exit(130)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", cmdErr)
		os.Exit(1)
	}
}

// Output functions
func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorRed+"Error: "+format+colorReset+"\n", args...)
	os.Exit(1)
}

func (io *IoCtrl) success(format string, args ...any) {
	fmt.Fprintf(io.writer, colorGreen+format+colorReset+"\n", args...)
}

func (io *IoCtrl) warn(format string, args ...any) {
	fmt.Fprintf(io.writer, colorYellow+format+colorReset+"\n", args...)
}

func (io *IoCtrl) info(format string, args ...any) {
	fmt.Fprintf(io.writer, colorBlue+format+colorReset+"\n", args...)
}

// Validation functions
func validateProjectName(name string) error {
	if name == "" {
		return errors.New("project name cannot be empty")
	}
	if !projectNameRegex.MatchString(name) {
		return fmt.Errorf("invalid project name '%s'. Must be 1-128 characters, start with alphanumeric or underscore, and contain only alphanumeric, hyphens, and underscores", name)
	}
	return nil
}

func sanitizeProjectName(input string) string {
	sanitized := strings.ToLower(input)
	re := regexp.MustCompile(`[^a-z0-9_-]`)
	sanitized = re.ReplaceAllString(sanitized, "-")

	// Remove leading non-alphanumeric
	for len(sanitized) > 0 && !regexp.MustCompile(`^[a-z0-9]`).MatchString(sanitized) {
		sanitized = sanitized[1:]
	}

	// Collapse consecutive hyphens
	for strings.Contains(sanitized, "--") {
		sanitized = strings.ReplaceAll(sanitized, "--", "-")
	}

	// Truncate to 128 chars
	if len(sanitized) > 128 {
		sanitized = sanitized[:128]
	}

	// Remove trailing hyphens
	sanitized = strings.TrimRight(sanitized, "-")

	if sanitized == "" {
		sanitized = "project"
	}

	return sanitized
}

func validateShell(shell string) error {
	valid := map[string]bool{"bash": true, "zsh": true, "fish": true}
	if !valid[shell] {
		return fmt.Errorf("invalid shell '%s'. Must be one of: bash, zsh, fish", shell)
	}
	return nil
}

func validateVersionArg(version string) error {
	if version == "" || version == versionLatest || version == versionNone {
		return nil
	}
	if !versionRegex.MatchString(version) {
		return fmt.Errorf("invalid version argument: '%s'. Must be either \"none\", \"latest\" or semantic versioning (e.g., 20.10.0)", version)
	}
	return nil
}

func validateAptPackageNames(packages string) error {
	re := regexp.MustCompile(`^[a-z0-9+._-]+$`)
	for pkg := range strings.FieldsSeq(packages) {
		if !re.MatchString(pkg) {
			return fmt.Errorf("invalid package name '%s'. Package names must contain only lowercase letters, digits, hyphens, periods, and plus signs", pkg)
		}
	}
	return nil
}

func validateGitName(name string) error {
	if strings.ContainsAny(name, "\"\n\r") {
		return errors.New("invalid git name. Cannot contain quotes or newlines")
	}
	if len(name) > 100 {
		return errors.New("git name too long (max 100 characters)")
	}
	return nil
}

func validateGitEmail(email string) error {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !re.MatchString(email) {
		return fmt.Errorf("invalid git email format '%s'", email)
	}
	return nil
}

func validateUsername(username string) error {
	re := regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)
	if !re.MatchString(username) {
		return fmt.Errorf("invalid username '%s'. Must start with lowercase letter or underscore, followed by lowercase letters, digits, underscores, or hyphens", username)
	}
	if len(username) > 32 {
		return errors.New("username too long (max 32 characters)")
	}
	return nil
}

func validatePort(port string) error {
	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		return fmt.Errorf("invalid port '%s'. Must be a number between 1 and 65535", port)
	}
	return nil
}

func validateUIDGID(id, idType string) error {
	i, err := strconv.Atoi(id)
	if err != nil || i < 0 || i > 65535 {
		return fmt.Errorf("invalid %s '%s'. Must be a number between 0 and 65535", idType, id)
	}
	return nil
}

// ContainerConfig functions
func newConfig() *ContainerConfig {
	cfg := &ContainerConfig{
		username:        "dev",
		shell:           "",
		installNode:     "",
		installRust:     "",
		installPython:   "",
		installGo:       "",
		enableWasm:      "",
		enableSSH:       "",
		enableSudo:      "",
		gitName:         "",
		gitEmail:        "",
		packages:        "",
		installNeovim:   "",
		installStarship: "",
		installAtuin:    "",
		installMise:     "",
		installZellij:   "",
		installJujutsu:  "",
	}

	// Set UID/GID
	// TODO: special platform file?
	if runtime.GOOS == "windows" {
		cfg.hostUID = "1000"
		cfg.hostGID = "1000"
	} else {
		cfg.hostUID = fmt.Sprintf("%d", os.Getuid())
		cfg.hostGID = fmt.Sprintf("%d", os.Getgid())
	}

	return cfg
}

func lightSanitize(str string) string {
	str = strings.ReplaceAll(str, "\n", "")
	str = strings.ReplaceAll(str, "\r", "")
	str = strings.ReplaceAll(str, "\"", "\\\"")
	return str
}

// File path helpers
func (app *App) getBaseCompose() string {
	return filepath.Join(app.scriptDir, baseComposeFilename)
}

func (app *App) getProjectDir(name string) string {
	return filepath.Join(app.projectsDir, name)
}

func (app *App) getProjectCompose(name string) string {
	return filepath.Join(app.projectsDir, name, projectComposeFilename)
}

func (app *App) getProjectEnv(name string) string {
	return filepath.Join(app.projectsDir, name, projectEnvFilename)
}

func (app *App) assertInexistantProject(name string) {
	composeFile := app.getProjectCompose(name)
	envFile := app.getProjectEnv(name)

	if _, err := os.Stat(composeFile); err == nil {
		fatal("Project '%s' already exists. You can have multiple configurations for the same project by calling 'create' with the '--name' flag. Hint: Use 'paul-envs list' to see all projects or 'paul-envs remove %s' to delete it", name, name)
	}
	if _, err := os.Stat(envFile); err == nil {
		fatal("Project '%s' already exists. You can have multiple configurations for the same project by calling 'create' with the '--name' flag. Hint: Use 'paul-envs list' to see all projects or 'paul-envs remove %s' to delete it", name, name)
	}
}

func doesLangVersionNeedsMise(version string) bool {
	return version != versionNone && version != versionLatest && version != ""
}

func miseCheck(cfg *ContainerConfig, noPrompt bool, io *IoCtrl) {
	needsMiseWarning := false
	if doesLangVersionNeedsMise(cfg.installNode) ||
		doesLangVersionNeedsMise(cfg.installRust) ||
		doesLangVersionNeedsMise(cfg.installPython) ||
		doesLangVersionNeedsMise(cfg.installGo) {
		if cfg.installMise != "true" {
			needsMiseWarning = true
		}
	}

	if needsMiseWarning {
		fmt.Println()
		io.warn("WARNING: You specified exact version(s) for language runtimes, but Mise is not enabled.")
		io.warn("Exact versions require Mise to be installed. Without Mise, Ubuntu's default packages will be used instead.")
		if !noPrompt {
			choice := io.askYesNo("Would you like to enable Mise now?", "Y")
			if choice {
				cfg.installMise = "true"
				io.success("Mise enabled")
			}
		}
	}
}

func (io *IoCtrl) askYesNo(prompt, defaultVal string) bool {
	fmt.Printf("%s (%s/n): ", prompt, defaultVal)
	input, err := io.reader.ReadString('\n')
	if err != nil {
		fatal("Failed to read user input: %v", err)
	}
	input = strings.TrimSpace(input)
	if input == "" {
		input = defaultVal
	}
	return strings.ToUpper(input) == "Y"
}

func (io *IoCtrl) askString(prompt, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultVal)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	input, err := io.reader.ReadString('\n')
	if err != nil {
		fatal("Failed to read user input: %v", err)
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func promptShellConfig(io *IoCtrl, cfg *ContainerConfig) {
	io.info("=== Shell Selection ===")
	fmt.Println("Select shell:")
	fmt.Println("  1) bash (default)")
	fmt.Println("  2) zsh")
	fmt.Println("  3) fish")
	choice := io.askString("Choice", "1")

	switch choice {
	case "1":
		cfg.shell = shellBash
	case "2":
		cfg.shell = shellZsh
	case "3":
		cfg.shell = shellFish
	default:
		cfg.shell = shellBash
	}
}

func promptLanguagesConfig(io *IoCtrl, cfg *ContainerConfig) {
	io.info("=== Language Runtimes ===")
	fmt.Println("Which language runtimes do you need? (space-separated numbers, or Enter to skip)")
	fmt.Println("  1) Node.js")
	fmt.Println("  2) Rust")
	fmt.Println("  3) Python")
	fmt.Println("  4) Go")
	fmt.Println("  5) WebAssembly tools (Binaryen, Rust WASM target if Rust is enabled)")
	langChoices := io.askString("Choice", "none")

	cfg.installNode = versionNone
	cfg.installRust = versionNone
	cfg.installPython = versionNone
	cfg.installGo = versionNone

	for choice := range strings.FieldsSeq(langChoices) {
		switch choice {
		case "1":
			nodeVer := io.askString("Node.js version (latest/none/X.Y.Z)", versionLatest)
			cfg.installNode = nodeVer
		case "2":
			rustVer := io.askString("Rust version (latest/none/X.Y.Z)", versionLatest)
			cfg.installRust = rustVer
		case "3":
			pythonVer := io.askString("Python version (latest/none/X.Y.Z)", versionLatest)
			cfg.installPython = pythonVer
		case "4":
			goVer := io.askString("Go version (latest/none/X.Y.Z)", versionLatest)
			cfg.installGo = goVer
		case "5":
			cfg.enableWasm = "true"
		case "none":
			cfg.installNode = versionNone
			cfg.installRust = versionNone
			cfg.installPython = versionNone
			cfg.installGo = versionNone
			return
		default:
			io.warn("Unknown choice: %s (skipped)", choice)
		}
	}
}

func promptPackagesConfig(io *IoCtrl, cfg *ContainerConfig) {
	io.info("=== Additional Packages ===")
	fmt.Println("The following packages are already installed on top of an Ubuntu:24.04 image:")
	fmt.Println("curl git build-essential")
	fmt.Println()
	fmt.Println("Enter additional Ubuntu packages (space-separated, or Enter to skip):")
	fmt.Println("Examples: ripgrep fzf htop")
	packages := io.askString("Packages", "")

	if packages != "" {
		if err := validateAptPackageNames(packages); err != nil {
			fatal(err.Error())
		}
		cfg.packages = packages
	}
}

func promptToolsConfig(io *IoCtrl, cfg *ContainerConfig) {
	io.info("=== Development Tools ===")
	fmt.Println("Some dev tools are not pulled from Ubuntu's repositories to get their latest version instead.")
	fmt.Println("Which of those tools do you want to install? (space-separated numbers, or Enter to skip all)")
	fmt.Println("  1) Neovim (text editor)")
	fmt.Println("  2) Starship (prompt)")
	fmt.Println("  3) Atuin (shell history)")
	fmt.Println("  4) Mise (version manager - required for specific language versions)")
	fmt.Println("  5) Zellij (terminal multiplexer)")
	fmt.Println("  6) Jujutsu (Git-compatible VCS)")
	toolChoices := io.askString("Choice", "none")

	cfg.installNeovim = "false"
	cfg.installStarship = "false"
	cfg.installAtuin = "false"
	cfg.installMise = "false"
	cfg.installZellij = "false"
	cfg.installJujutsu = "false"

	for choice := range strings.FieldsSeq(toolChoices) {
		switch choice {
		case "1":
			cfg.installNeovim = "true"
		case "2":
			cfg.installStarship = "true"
		case "3":
			cfg.installAtuin = "true"
		case "4":
			cfg.installMise = "true"
		case "5":
			cfg.installZellij = "true"
		case "6":
			cfg.installJujutsu = "true"
		case "none":
			cfg.installNeovim = "false"
			cfg.installStarship = "false"
			cfg.installAtuin = "false"
			cfg.installMise = "false"
			cfg.installZellij = "false"
			cfg.installJujutsu = "false"
			return
		default:
			io.warn("Unknown choice: %s (skipped)", choice)
		}
	}
}

func promptSudoConfig(io *IoCtrl, cfg *ContainerConfig) {
	io.info("=== Sudo Access ===")
	if io.askYesNo("Enable sudo access in container (password:\"dev\")?", "N") {
		cfg.enableSudo = "true"
	} else {
		cfg.enableSudo = "false"
	}
}

func promptSshConfig(io *IoCtrl, cfg *ContainerConfig) {
	io.info("=== SSH Access ===")
	if io.askYesNo("Enable ssh access to container?", "N") {
		cfg.enableSSH = "true"
		promptSshKeysConfig(io, cfg)
	} else {
		cfg.enableSSH = "false"
	}
}

func promptSshKeysConfig(io *IoCtrl, cfg *ContainerConfig) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		io.warn("Unable to obtain home dir for ssh prompt: %v", err)
		cfg.sshKeyPath = ""
		return
	}
	sshDir := filepath.Join(homeDir, ".ssh")

	var pubKeys []string
	if info, err := os.Stat(sshDir); err == nil && info.IsDir() {
		files, err := filepath.Glob(filepath.Join(sshDir, "*.pub"))
		if err != nil {
			io.warn("Unable to list ssh public keys, skipping ssh keys prompt: %v", err)
			cfg.sshKeyPath = ""
			return
		}
		pubKeys = files
	}

	if len(pubKeys) == 0 {
		io.warn("No SSH public keys found in ~/.ssh/")
		cfg.sshKeyPath = ""
		return
	}

	fmt.Println()
	fmt.Println("Select SSH public key to mount:")
	for i, key := range pubKeys {
		fmt.Printf("  %d) %s\n", i+1, filepath.Base(key))
	}
	fmt.Printf("  %d) Custom path\n", len(pubKeys)+1)
	fmt.Printf("  %d) Skip (add manually later)\n", len(pubKeys)+2)

	choice := io.askString("Choice", "1")
	choiceNum, err := strconv.Atoi(choice)
	if err != nil {
		io.warn("Invalid value: %s. Skipping...", choice)
		cfg.sshKeyPath = ""
	} else if choiceNum >= 1 && choiceNum <= len(pubKeys) {
		cfg.sshKeyPath = pubKeys[choiceNum-1]
	} else if choiceNum == len(pubKeys)+1 {
		customKey := io.askString("Enter path to public key", "")
		if _, err := os.Stat(customKey); err == nil {
			cfg.sshKeyPath = customKey
		} else {
			io.warn("File not found: %s", customKey)
			cfg.sshKeyPath = ""
		}
	} else {
		cfg.sshKeyPath = ""
	}
}

func promptPortsWanted(io *IoCtrl) []string {
	io.info("=== Port Forwarding ===")
	fmt.Println("Enter supplementary container ports to expose (space-separated, or Enter to skip):")
	fmt.Println("Examples: 3000 5432 8080")
	portInput := io.askString("Ports", "")

	var ports []string
	for port := range strings.FieldsSeq(portInput) {
		if err := validatePort(port); err != nil {
			fatal(err.Error())
		}
		ports = append(ports, port)
	}
	return ports
}

func promptVolumesWanted(io *IoCtrl) []string {
	io.info("=== Credentials & Volumes ===")
	fmt.Println("Mount common credentials/configs? (space-separated numbers, or Enter to skip)")
	fmt.Println("  1) SSH keys (~/.ssh)")
	fmt.Println("  2) Git credentials (~/.git-credentials)")
	fmt.Println("  3) AWS credentials (~/.aws)")
	fmt.Println("  4) Custom CA certificates (/etc/ssl/certs/custom-ca.crt)")
	choices := io.askString("Choice", "none")

	var volumes []string

	homeDir, err := os.UserHomeDir()
	if err != nil {
		io.warn("Unable to obtain home dir for volume prompt: %v", err)
		return volumes
	}
	for choice := range strings.FieldsSeq(choices) {
		switch choice {
		case "1":
			volumes = append(volumes, fmt.Sprintf("%s/.ssh:/home/${USERNAME}/.ssh:ro", homeDir))
		case "2":
			volumes = append(volumes, fmt.Sprintf("%s/.git-credentials:/home/${USERNAME}/.git-credentials:ro", homeDir))
		case "3":
			volumes = append(volumes, fmt.Sprintf("%s/.aws:/home/${USERNAME}/.aws:ro", homeDir))
		case "4":
			volumes = append(volumes, "/etc/ssl/certs/custom-ca.crt:/usr/local/share/ca-certificates/custom-ca.crt:ro")
		case "none":
			volumes = volumes[:0]
			return volumes
		default:
			io.warn("Unknown choice: %s (skipped)", choice)
		}
	}

	fmt.Println()
	fmt.Println("Add custom volumes? (one per line, Enter on empty line to finish)")
	fmt.Println("Format: /host/path:/container/path[:ro]")
	for {
		vol := io.askString("Volume", "")
		if vol == "" {
			break
		}
		// Normalize host path
		if idx := strings.Index(vol, ":"); idx != -1 {
			hostPart := vol[:idx]
			containerPart := vol[idx:]
			vol = hostPart + containerPart
		}
		volumes = append(volumes, vol)
	}

	return volumes
}

const envFileTpl = `# "Env file" for your project, which will be fed to docker compose
# alongside {{.ProjectComposeFilename}} in the same directory.
#
# Can be freely updated.

# Uniquely identify this container.
# *SHOULD NOT BE UPDATED*
PROJECT_ID="{{.ProjectID}}"

# Name of the project directory inside the container.
# A PROJECT_DIRNAME should always be set
PROJECT_DIRNAME="{{.ProjectDestPath}}"

# Path to the project you want to mount in this container
# Will be mounted in "$HOME/projects/<PROJECT_DIRNAME>" inside that container.
# A PROJECT_PATH should always be set
PROJECT_PATH="{{.ProjectHostPath}}"

# To align with your current uid.
# This is to ensure the mounted volume from your host has compatible
# permissions.
# On POSIX-like systems, just run 'id -u' with the wanted user to know it.
HOST_UID="{{.HostUID}}"

# To align with your current gid (same reason than for "uid").
# On POSIX-like systems, just run 'id -g' with the wanted user to know it.
HOST_GID="{{.HostGID}}"

# Username created in the container.
# Not really important, just set it if you want something other than "dev".
USERNAME="{{.Username}}"

# The default shell wanted.
# Only "bash", "zsh" or "fish" are supported for now.
USER_SHELL="{{.Shell}}"

# Whether to install Node.js, and the version wanted.
# Note that a WebAssembly target is also automatically ready.
#
# Values can be:
# - if 'none': don't install Node.js
# - if 'latest': Install Ubuntu's default package for Node.js
# - If anything else: The exact version to install (e.g. "1.90.0").
#   That last type of value will only work if INSTALL_MISE is 'true'.
INSTALL_NODE="{{.InstallNode}}"

# Whether to install Rust and Cargo, and the version wanted.
# Note that a WebAssembly target is also automatically ready.
#
# Values can be:
# - if 'none': don't install Rust
# - if 'latest': Install Ubuntu's default package for Rust
#   Ubuntu base's repositories
# - If anything else: The exact version to install (e.g. "1.90.0").
#   That last type of value will only work if INSTALL_MISE is 'true'.
INSTALL_RUST="{{.InstallRust}}"

# Whether to install Python, and the version wanted.
#
# Values can be:
# - if 'none': don't install Python
# - if 'latest': Install Ubuntu's default package for Python
# - If anything else: The exact version to install (e.g. "3.12.0").
#   That last type of value will only work if INSTALL_MISE is 'true'.
INSTALL_PYTHON="{{.InstallPython}}"

# Whether to install Go, and the version wanted.
# Note that GOPATH is automatically set to ~/go
#
# Values can be:
# - if 'none': don't install Go
# - if 'latest': Install Ubuntu's default package for Go
# - If anything else: The exact version to install (e.g. "1.21.5").
#   That last type of value will only work if INSTALL_MISE is 'true'.
INSTALL_GO="{{.InstallGo}}"

# If 'true', add WebAssembly-specialized tools such as binaryen and a
# WebAssembly target for Rust if it is installed.
ENABLE_WASM="{{.EnableWasm}}"

# If 'true', openssh will be installed, and the container will listen for ssh
# connections at port 22.
ENABLE_SSH="{{.EnableSSH}}"

# If 'true', sudo will be installed, with a password set to "dev".
ENABLE_SUDO="{{.EnableSudo}}"

# Additional packages outside the core base, separated by a space.
# Have to be in Ubuntu's default repository
# (e.g. "ripgrep fzf". Can be left empty for no supplementary packages)
SUPPLEMENTARY_PACKAGES="{{.Packages}}"

# Tools toggle.
# "true" == install it
# anything else == don't.
INSTALL_NEOVIM="{{.InstallNeovim}}"
INSTALL_STARSHIP="{{.InstallStarship}}"
INSTALL_ATUIN="{{.InstallAtuin}}"
INSTALL_MISE="{{.InstallMise}}"
INSTALL_ZELLIJ="{{.InstallZellij}}"
INSTALL_JUJUTSU="{{.InstallJujutsu}}"

# Git author and committer name used inside the container
# Can also be empty to not set that in the container.
GIT_AUTHOR_NAME="{{.GitName}}"

# Git author and committer e-mail used inside the container
# Can also be empty to not set that in the container.
GIT_AUTHOR_EMAIL="{{.GitEmail}}"
`

func generateProjectCompose(app *App, name string, cfg *ContainerConfig, ports, volumes []string) {
	composeFile := app.getProjectCompose(name)
	envFile := app.getProjectEnv(name)

	app.assertInexistantProject(name)

	os.MkdirAll(filepath.Dir(composeFile), 0755)

	// Generate .env file using template
	tmpl := template.Must(template.New("env").Parse(envFileTpl))

	data := struct {
		ProjectComposeFilename string
		ProjectID              string
		ProjectDestPath        string
		ProjectHostPath        string
		HostUID                string
		HostGID                string
		Username               string
		Shell                  string
		InstallNode            string
		InstallRust            string
		InstallPython          string
		InstallGo              string
		EnableWasm             string
		EnableSSH              string
		EnableSudo             string
		Packages               string
		InstallNeovim          string
		InstallStarship        string
		InstallAtuin           string
		InstallMise            string
		InstallZellij          string
		InstallJujutsu         string
		GitName                string
		GitEmail               string
	}{
		ProjectComposeFilename: projectComposeFilename,
		ProjectID:              name,
		ProjectDestPath:        cfg.projectDestPath,
		ProjectHostPath:        cfg.projectHostPath,
		HostUID:                cfg.hostUID,
		HostGID:                cfg.hostGID,
		Username:               cfg.username,
		Shell:                  cfg.shell,
		InstallNode:            cfg.installNode,
		InstallRust:            cfg.installRust,
		InstallPython:          cfg.installPython,
		InstallGo:              cfg.installGo,
		EnableWasm:             cfg.enableWasm,
		EnableSSH:              cfg.enableSSH,
		EnableSudo:             cfg.enableSudo,
		Packages:               lightSanitize(cfg.packages),
		InstallNeovim:          cfg.installNeovim,
		InstallStarship:        cfg.installStarship,
		InstallAtuin:           cfg.installAtuin,
		InstallMise:            cfg.installMise,
		InstallZellij:          cfg.installZellij,
		InstallJujutsu:         cfg.installJujutsu,
		GitName:                lightSanitize(cfg.gitName),
		GitEmail:               lightSanitize(cfg.gitEmail),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		fatal("Failed to generate env file: %v", err)
	}

	if err := os.WriteFile(envFile, buf.Bytes(), 0644); err != nil {
		fatal("Failed to write env file: %v", err)
	}

	// Generate compose.yaml
	var composeBuilder strings.Builder
	// TODO: rely on projectEnvFilename
	composeBuilder.WriteString(`# Compose file for your project, which will be fed to docker compose alongside
# ".env" in the same directory.
#
# Can be freely updated to update ports, volumes etc.
services:
  paulenv:
    build:
    image: paulenv:` + name + "\n")

	// Add ports
	if len(ports) > 0 || cfg.enableSSH == "true" {
		composeBuilder.WriteString("    ports:\n")
		for _, port := range ports {
			composeBuilder.WriteString(fmt.Sprintf("      - \"%s:%s\"\n", port, port))
		}
		if cfg.enableSSH == "true" {
			composeBuilder.WriteString("      # to listen for ssh connections\n")
			composeBuilder.WriteString("      - \"22:22\"\n")
		}
	}

	// Add volumes
	composeBuilder.WriteString("    volumes:\n")
	for _, vol := range volumes {
		composeBuilder.WriteString(fmt.Sprintf("      - %s\n", vol))
	}

	if cfg.enableSSH == "true" && cfg.sshKeyPath != "" {
		sshKeyPath := cfg.sshKeyPath
		composeBuilder.WriteString("      # Your local public key for ssh:\n")
		composeBuilder.WriteString(fmt.Sprintf("      - %s:/etc/ssh/authorized_keys/${USERNAME}:ro\n", sshKeyPath))
	} else if cfg.enableSSH == "true" {
		app.io.warn("No SSH key configured")
		app.io.warn("Adding a note to your compose file: %s", composeFile)
		composeBuilder.WriteString("      # Add your SSH public key here, for example:\n")
		composeBuilder.WriteString("      # - ~/.ssh/id_ed25519.pub:/etc/ssh/authorized_keys/${USERNAME}:ro\n")
	}
	composeBuilder.WriteString("\n")

	if err := os.WriteFile(composeFile, []byte(composeBuilder.String()), 0644); err != nil {
		fatal("Failed to write compose file: %v", err)
	}
}

// Command implementations
func (app *App) Create(args []string) {
	cfg := newConfig()
	var name string
	var ports, volumes []string
	var noPrompt bool

	// Parse flags
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	fs.BoolVar(&noPrompt, "no-prompt", false, "Non-interactive mode")
	fs.StringVar(&name, "name", "", "Project name")
	fs.StringVar(&cfg.hostUID, "uid", cfg.hostUID, "Container UID")
	fs.StringVar(&cfg.hostGID, "gid", cfg.hostGID, "Container GID")
	fs.StringVar(&cfg.username, "username", cfg.username, "Container username")
	fs.StringVar(&cfg.shell, "shell", "", "User shell")
	fs.StringVar(&cfg.installNode, "nodejs", "", "Node.js version")
	fs.StringVar(&cfg.installRust, "rust", "", "Rust version")
	fs.StringVar(&cfg.installPython, "python", "", "Python version")
	fs.StringVar(&cfg.installGo, "go", "", "Go version")
	fs.BoolFunc("enable-wasm", "Enable WebAssembly tools", func(s string) error {
		cfg.enableWasm = "true"
		return nil
	})
	fs.BoolFunc("wasm", "Enable WebAssembly tools", func(s string) error {
		cfg.enableWasm = "true"
		return nil
	})
	fs.BoolFunc("enable-ssh", "Enable SSH access", func(s string) error {
		cfg.enableSSH = "true"
		return nil
	})
	fs.BoolFunc("ssh", "Enable SSH access", func(s string) error {
		cfg.enableSSH = "true"
		return nil
	})
	fs.BoolFunc("enable-sudo", "Enable sudo access", func(s string) error {
		cfg.enableSudo = "true"
		return nil
	})
	fs.BoolFunc("sudo", "Enable sudo access", func(s string) error {
		cfg.enableSudo = "true"
		return nil
	})
	fs.StringVar(&cfg.gitName, "git-name", "", "Git user name")
	fs.StringVar(&cfg.gitEmail, "git-email", "", "Git user email")
	fs.StringVar(&cfg.packages, "packages", "", "Additional packages")
	fs.BoolFunc("neovim", "Install Neovim", func(s string) error {
		cfg.installNeovim = "true"
		return nil
	})
	fs.BoolFunc("starship", "Install Starship", func(s string) error {
		cfg.installStarship = "true"
		return nil
	})
	fs.BoolFunc("atuin", "Install Atuin", func(s string) error {
		cfg.installAtuin = "true"
		return nil
	})
	fs.BoolFunc("mise", "Install Mise", func(s string) error {
		cfg.installMise = "true"
		return nil
	})
	fs.BoolFunc("zellij", "Install Zellij", func(s string) error {
		cfg.installZellij = "true"
		return nil
	})
	fs.BoolFunc("jujutsu", "Install Jujutsu", func(s string) error {
		cfg.installJujutsu = "true"
		return nil
	})

	// Custom parsing for repeatable flags
	var i int
	for i = 1; i < len(args); i++ {
		if args[i] == "--port" && i+1 < len(args) {
			ports = append(ports, args[i+1])
			i++
		} else if args[i] == "--volume" && i+1 < len(args) {
			vol := args[i+1]
			if idx := strings.Index(vol, ":"); idx != -1 {
				hostPart := vol[:idx]
				containerPart := vol[idx:]
				vol = hostPart + containerPart
			}
			volumes = append(volumes, vol)
			i++
		} else {
			break
		}
	}

	// Get project path
	if len(args) == 0 {
		fatal("Usage: paul-envs create <project-path> [options]")
	}

	projectPath, err := filepath.Abs(args[0])
	if err != nil {
		fatal("Invalid project path: %v", err)
	}
	cfg.projectHostPath = projectPath

	// Parse remaining flags
	fs.Parse(args[1:])

	// Validate inputs
	if cfg.hostUID != "" {
		if err := validateUIDGID(cfg.hostUID, "UID"); err != nil {
			fatal(err.Error())
		}
	}
	if cfg.hostGID != "" {
		if err := validateUIDGID(cfg.hostGID, "GID"); err != nil {
			fatal(err.Error())
		}
	}
	if cfg.username != "" {
		if err := validateUsername(cfg.username); err != nil {
			fatal(err.Error())
		}
	}
	if cfg.shell != "" {
		if err := validateShell(cfg.shell); err != nil {
			fatal(err.Error())
		}
	}
	if cfg.installNode != "" {
		if err := validateVersionArg(cfg.installNode); err != nil {
			fatal(err.Error())
		}
	}
	if cfg.installRust != "" {
		if err := validateVersionArg(cfg.installRust); err != nil {
			fatal(err.Error())
		}
	}
	if cfg.installPython != "" {
		if err := validateVersionArg(cfg.installPython); err != nil {
			fatal(err.Error())
		}
	}
	if cfg.installGo != "" {
		if err := validateVersionArg(cfg.installGo); err != nil {
			fatal(err.Error())
		}
	}
	if cfg.gitName != "" {
		if err := validateGitName(cfg.gitName); err != nil {
			fatal(err.Error())
		}
	}
	if cfg.gitEmail != "" {
		if err := validateGitEmail(cfg.gitEmail); err != nil {
			fatal(err.Error())
		}
	}
	if cfg.packages != "" {
		if err := validateAptPackageNames(cfg.packages); err != nil {
			fatal(err.Error())
		}
	}
	for _, port := range ports {
		if err := validatePort(port); err != nil {
			fatal(err.Error())
		}
	}

	// Determine project name
	if name == "" {
		name = filepath.Base(cfg.projectHostPath)
	}
	if err := validateProjectName(name); err != nil {
		name = sanitizeProjectName(name)
	} else {
		name = sanitizeProjectName(name)
	}

	cfg.projectDestPath = name
	app.assertInexistantProject(name)

	// Set defaults or prompt
	if noPrompt {
		if cfg.shell == "" {
			cfg.shell = shellBash
		}
		if cfg.installNode == "" {
			cfg.installNode = versionNone
		}
		if cfg.installRust == "" {
			cfg.installRust = versionNone
		}
		if cfg.installPython == "" {
			cfg.installPython = versionNone
		}
		if cfg.installGo == "" {
			cfg.installGo = versionNone
		}
		if cfg.enableWasm == "" {
			cfg.enableWasm = "false"
		}
		if cfg.enableSSH == "" {
			cfg.enableSSH = "false"
		}
		if cfg.enableSudo == "" {
			cfg.enableSudo = "false"
		}
		if cfg.installNeovim == "" {
			cfg.installNeovim = "false"
		}
		if cfg.installStarship == "" {
			cfg.installStarship = "false"
		}
		if cfg.installAtuin == "" {
			cfg.installAtuin = "false"
		}
		if cfg.installMise == "" {
			cfg.installMise = "false"
		}
		if cfg.installZellij == "" {
			cfg.installZellij = "false"
		}
		if cfg.installJujutsu == "" {
			cfg.installJujutsu = "false"
		}
		miseCheck(cfg, noPrompt, app.io)
	} else {
		if cfg.shell == "" {
			fmt.Println()
			promptShellConfig(app.io, cfg)
		}

		hasAnyLang := cfg.installNode != "" || cfg.installRust != "" ||
			cfg.installPython != "" || cfg.installGo != ""

		if hasAnyLang {
			if cfg.installNode == "" {
				cfg.installNode = versionNone
			}
			if cfg.installRust == "" {
				cfg.installRust = versionNone
			}
			if cfg.installPython == "" {
				cfg.installPython = versionNone
			}
			if cfg.installGo == "" {
				cfg.installGo = versionNone
			}
		} else {
			fmt.Println()
			promptLanguagesConfig(app.io, cfg)
		}

		toolsSet := cfg.installNeovim != "" || cfg.installStarship != "" ||
			cfg.installAtuin != "" || cfg.installMise != "" ||
			cfg.installZellij != "" || cfg.installJujutsu != ""

		if toolsSet {
			if cfg.installNeovim == "" {
				cfg.installNeovim = "false"
			}
			if cfg.installStarship == "" {
				cfg.installStarship = "false"
			}
			if cfg.installAtuin == "" {
				cfg.installAtuin = "false"
			}
			if cfg.installMise == "" {
				cfg.installMise = "false"
			}
			if cfg.installZellij == "" {
				cfg.installZellij = "false"
			}
			if cfg.installJujutsu == "" {
				cfg.installJujutsu = "false"
			}
		} else {
			fmt.Println()
			promptToolsConfig(app.io, cfg)
		}

		miseCheck(cfg, noPrompt, app.io)

		if cfg.enableSudo == "" {
			fmt.Println()
			promptSudoConfig(app.io, cfg)
		}
		if cfg.enableSSH == "" {
			fmt.Println()
			promptSshConfig(app.io, cfg)
		}
		if cfg.packages == "" {
			fmt.Println()
			promptPackagesConfig(app.io, cfg)
		}

		if len(ports) == 0 {
			fmt.Println()
			ports = promptPortsWanted(app.io)
		}
		if len(volumes) == 0 {
			fmt.Println()
			volumes = promptVolumesWanted(app.io)
		}
	}

	// Validate path exists or warn
	os.MkdirAll(app.projectsDir, 0755)

	if _, err := os.Stat(cfg.projectHostPath); os.IsNotExist(err) && !noPrompt {
		app.io.warn("Warning: Path %s does not exist", cfg.projectHostPath)
		if !app.io.askYesNo("Create config anyway?", "N") {
			os.Exit(1)
		}
	}

	// Generate config
	generateProjectCompose(app, name, cfg, ports, volumes)

	app.io.success("Created project '%s'", name)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Review/edit configuration:\n")
	fmt.Printf("     - %s\n", app.getProjectEnv(name))
	fmt.Printf("     - %s\n", app.getProjectCompose(name))
	fmt.Printf("  2. Put the $HOME dotfiles you want to port in:\n")
	fmt.Printf("     - %s/configs/\n", app.scriptDir)
	fmt.Printf("  3. Build the environment:\n")
	fmt.Printf("     paul-envs build %s\n", name)
	fmt.Printf("  4. Run the environment:\n")
	fmt.Printf("     paul-envs run %s\n", name)
}

func (app *App) List() error {
	baseCompose := app.getBaseCompose()
	if _, err := os.Stat(baseCompose); os.IsNotExist(err) {
		return fmt.Errorf("Base compose.yaml not found at %s", baseCompose)
	}

	if _, err := os.Stat(app.projectsDir); os.IsNotExist(err) {
		fmt.Println("No project created yet")
		fmt.Println("Hint: Create one with 'paul-envs create <path>'")
		return nil
	}

	entries, err := os.ReadDir(app.projectsDir)
	if err != nil {
		return fmt.Errorf("Failed to read project directory: %w", err)
	}

	fmt.Println("Projects created:")
	found := false
	for _, entry := range entries {
		if entry.IsDir() {
			composeFile := app.getProjectCompose(entry.Name())
			if _, err := os.Stat(composeFile); err == nil {
				found = true
				envFile := app.getProjectEnv(entry.Name())
				envData, _ := os.ReadFile(envFile)
				pathRe := regexp.MustCompile(`PROJECT_PATH="([^"]*)"`)
				matches := pathRe.FindStringSubmatch(string(envData))
				path := ""
				if len(matches) > 1 {
					path = matches[1]
				}
				fmt.Printf("  - %s\n", entry.Name())
				fmt.Printf("      Path: %s\n", path)
			}
		}
	}

	if !found {
		fmt.Println("  (no project found)")
		fmt.Println("Hint: Create one with 'paul-envs create <path>'")
	}
	return nil
}

func (app *App) Build(ctx context.Context, args []string) error {
	baseCompose := app.getBaseCompose()
	if _, err := os.Stat(baseCompose); os.IsNotExist(err) {
		return fmt.Errorf("Base compose.yaml not found at %s", baseCompose)
	}

	var name string
	if len(args) == 0 {
		fmt.Println("No project name given, listing projects...")
		fmt.Println()
		app.List()
		fmt.Println()
		name = app.io.askString("Enter project name to build", "")
		if name == "" {
			return errors.New("No project name provided")
		}
	} else {
		name = args[0]
	}

	if err := validateProjectName(name); err != nil {
		return err
	}

	composeFile := app.getProjectCompose(name)
	envFile := app.getProjectEnv(name)
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("Project '%s' not found. Hint: Use 'paul-envs.sh list' to see available projects or 'paul-envs create' to make a new one", name)
	}
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		return fmt.Errorf("Project '%s' not found. Hint: Use 'paul-envs list' to see available projects or 'paul-envs create' to make a new one", name)
	}

	// Create shared cache volume
	if err := exec.CommandContext(ctx, "docker", "volume", "create", "paulenv-shared-cache").Run(); err != nil {
		return fmt.Errorf("Failed to create shared cache volume: %w", err)
	}

	// Build
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", baseCompose, "-f", composeFile, "--env-file", envFile, "build")
	cmd.Env = append(os.Environ(), "COMPOSE_PROJECT_NAME=paulenv-"+name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Build failed: %w", err)
	}
	app.io.success("Built project '%s'", name)

	fmt.Println()
	app.io.warn("Resetting persistent volumes...")
	cmd = exec.CommandContext(ctx, "docker", "compose", "-f", baseCompose, "-f", composeFile, "--env-file", envFile, "--profile", "reset", "up", "reset-cache", "reset-local")
	cmd.Env = append(os.Environ(), "COMPOSE_PROJECT_NAME=paulenv-"+name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to setup volumes: %w", err)
	}
	app.io.success("Volumes reset complete")
	return nil
}

func (app *App) Run(ctx context.Context, args []string) error {
	baseCompose := app.getBaseCompose()
	if _, err := os.Stat(baseCompose); os.IsNotExist(err) {
		return fmt.Errorf("Base compose.yaml not found at %s", baseCompose)
	}

	var name string
	var cmdArgs []string

	if len(args) == 0 {
		fmt.Println("No project name given, listing projects...")
		fmt.Println()
		app.List()
		fmt.Println()
		name = app.io.askString("Enter project name to run", "")
		if name == "" {
			return fmt.Errorf("No project name provided")
		}
	} else {
		name = args[0]
		if len(args) > 1 {
			cmdArgs = args[1:]
		}
	}

	if err := validateProjectName(name); err != nil {
		return err
	}

	composeFile := app.getProjectCompose(name)
	envFile := app.getProjectEnv(name)
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("Project '%s' not found\nHint: Use 'paul-envs list' to see available projects", name)
	}
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		return fmt.Errorf("Project '%s' not found\nHint: Use 'paul-envs list' to see available projects", name)
	}

	args = []string{"compose", "-f", baseCompose, "-f", composeFile, "--env-file", envFile, "run", "--rm", "paulenv"}
	args = append(args, cmdArgs...)

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Env = append(os.Environ(), "COMPOSE_PROJECT_NAME=paulenv-"+name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Run failed: %w", err)
	}

	return nil
}

func (app *App) Remove(args []string) error {
	var name string
	if len(args) == 0 {
		fmt.Println("No project name given, listing projects...")
		fmt.Println()
		app.List()
		fmt.Println()
		name = app.io.askString("Enter project name to remove", "")
		if name == "" {
			return fmt.Errorf("No project name provided")
		}
	} else {
		name = args[0]
	}

	if err := validateProjectName(name); err != nil {
		return err
	}

	projectDir := app.getProjectDir(name)
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return fmt.Errorf("Project '%s' not found\nHint: Use 'paul-envs list' to see available projects", name)
	}

	if !app.io.askYesNo(fmt.Sprintf("Remove project '%s'?", name), "N") {
		return nil
	}

	if err := os.RemoveAll(projectDir); err != nil {
		return fmt.Errorf("Failed to remove project: %w", err)
	}
	app.io.success("Removed project '%s'", name)
	fmt.Println("Note: Docker volumes are preserved. To remove them, run:")
	fmt.Printf("  docker volume rm paulenv-%s-local\n", name)
	return nil
}

func (app *App) Version(ctx context.Context) error {
	fmt.Printf("paul-envs version %s\n", version)
	fmt.Printf("Go version: %s\n", runtime.Version())
	cmd := exec.CommandContext(ctx, "docker", "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("Failed to obtain version: %w", err)
	}
	fmt.Printf("Docker version: %s", output)
	return nil
}

func printUsage(app *App) {
	fmt.Println(`paul-envs - Development Environment Manager

Usage:
  paul-envs create <path> [options]
  paul-envs list
  paul-envs build <name>
  paul-envs run <name> [commands]
  paul-envs remove <name>

Options for create (all optional):
  --no-prompt              Non-interactive mode (uses defaults)
  --name NAME              Name of this project (default: directory name)
  --uid UID                Container UID (default: current user - or 1000 on windows)
  --gid GID                Container GID (default: current group - or 1000 on windows)
  --username NAME          Container username (default: dev)
  --shell SHELL            User shell: bash|zsh|fish (prompted if not specified)
  --nodejs VERSION         Node.js installation:
                             'none' - skip installation of Node.js
                             'latest' - use Ubuntu default package
                             '20.10.0' - specific version (requires mise)
                           (prompted if no language specified)
  --rust VERSION           Rust installation:
                             'none' - skip installation of Rust
                             'latest' - latest stable via rustup
                             '1.75.0' - specific version (requires mise)
                           (prompted if no language specified)
  --python VERSION         Python installation:
                             'none' - skip installation of Python
                             'latest' - use Ubuntu default package
                             '3.12.0' - specific version (requires mise)
                           (prompted if no language specified)
  --go VERSION             Go installation:
                             'none' - skip installation of Go
                             'latest' - use Ubuntu default package
                             '1.21.5' - specific version (requires mise)
                           (prompted if no language specified)
  --enable-wasm            Add WASM-specialized tools (binaryen, Rust wasm target if enabled)
                           (prompted if no language specified)
  --enable-ssh             Enable ssh access on port 22 (E.g. to access files from your host)
                           (prompted if not specified)
  --enable-sudo            Enable sudo access in container with a "dev" password
                           (prompted if not specified)
  --git-name NAME          Git user.name (optional)
  --git-email EMAIL        Git user.email (optional)
  --neovim                 Install Neovim (text editor)
                           (prompted if no tool specified)
  --starship               Install Starship (prompt)
                           (prompted if no tool specified)
  --atuin                  Install Atuin (shell history)
                           (prompted if no tool specified)
  --mise                   Install Mise (version manager - required for specific language versions)
                           (prompted if no tool specified)
  --zellij                 Install Zellij (terminal multiplexer)
                           (prompted if no tool specified)
  --jujutsu                Install Jujutsu (Git-compatible VCS)
                           (prompted if no tool specified)
  --packages "PKG1 PKG2"   Additional Ubuntu packages (prompted if not specified)
  --port PORT              Expose container port (prompted if not specified, can be repeated)
  --volume HOST:CONT[:ro]  Mount volume (prompted if not specified, can be repeated)

Windows/Git Bash Notes:
  - Paths are automatically converted (C:\Users\... -> /c/Users/...)
  - UID/GID default to 1000 on Windows (Docker Desktop requirement)
  - Use forward slashes or let the script normalize paths for you

Interactive Mode (default):
  paul-envs create ~/projects/myapp
  # Will prompt for all unspecified options

Non-Interactive Mode:
  paul-envs create ~/projects/myapp --no-prompt --shell bash --nodejs latest

Mixed Mode (some flags + prompts):
  paul-envs create ~/projects/myapp --nodejs 20.10.0 --rust latest --mise
  # Will prompt for shell, sudo, packages, ports, and volumes

Full Configuration Example:
  paul-envs create ~/work/api \
    --name myApp \
    --shell zsh \
    --nodejs 20.10.0 \
    --rust latest \
    --python 3.12.0 \
    --go latest \
    --mise \
    --neovim \
    --starship \
    --zellij \
    --jujutsu \
    --enable-ssh \
    --enable-sudo \
    --git-name "John Doe" \
    --git-email "john@example.com" \
    --packages "ripgrep fzf" \
    --port 3000 \
    --port 5432 \
    --volume ~/.git-credentials:/home/dev/.git-credentials:ro

Configuration:
  Base compose: ` + app.getBaseCompose() + `
  Projects directory: ` + app.projectsDir)
}
