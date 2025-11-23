package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const version = "0.1.0"

// ANSI color codes
const (
	colorRed    = "\033[0;31m"
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[1;33m"
	colorBlue   = "\033[0;34m"
	colorReset  = "\033[0m"
)

type Config struct {
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

var (
	isWindows   bool
	scriptDir   string
	projectsDir string
	reader      *bufio.Reader
)

func init() {
	reader = bufio.NewReader(os.Stdin)
	isWindows = runtime.GOOS == "windows"

	// Get script directory
	ex, err := os.Executable()
	if err != nil {
		fatal("Failed to get executable path: %v", err)
	}
	scriptDir = filepath.Dir(ex)

	// Normalize for Windows
	if isWindows {
		scriptDir = normalizePath(scriptDir)
	}

	projectsDir = filepath.Join(scriptDir, "projects")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "create":
		cmdCreate(args)
	case "list", "ls":
		cmdList()
	case "build":
		cmdBuild(args)
	case "run":
		cmdRun(args)
	case "remove", "rm":
		cmdRemove(args)
	case "version", "--version", "-v":
		cmdVersion()
	default:
		printUsage()
	}
}

// Output functions
func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, colorRed+"Error: "+format+colorReset+"\n", args...)
	os.Exit(1)
}

func success(format string, args ...interface{}) {
	fmt.Printf(colorGreen+format+colorReset+"\n", args...)
}

func warn(format string, args ...interface{}) {
	fmt.Printf(colorYellow+format+colorReset+"\n", args...)
}

func info(format string, args ...interface{}) {
	fmt.Printf(colorBlue+format+colorReset+"\n", args...)
}

// Path normalization for Windows
func normalizePath(path string) string {
	if !isWindows {
		return path
	}

	// Convert backslashes to forward slashes
	path = strings.ReplaceAll(path, "\\", "/")

	// Handle Windows drive letters (C:\ -> /c/)
	driveRe := regexp.MustCompile(`^([A-Za-z]):[\\/](.*)$`)
	if matches := driveRe.FindStringSubmatch(path); matches != nil {
		drive := strings.ToLower(matches[1])
		rest := matches[2]
		return "/" + drive + "/" + rest
	}

	// Handle Git Bash paths (/c/Users/...)
	if strings.HasPrefix(path, "/") {
		return path
	}

	return path
}

func getAbsolutePath(path string) (string, error) {
	path = strings.ReplaceAll(path, "\\", "/")

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return normalizePath(absPath), nil
}

// Validation functions
func validateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	re := regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_-]{0,127}$`)
	if !re.MatchString(name) {
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
	if version == "" || version == "latest" || version == "none" {
		return nil
	}
	re := regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	if !re.MatchString(version) {
		return fmt.Errorf("invalid version argument: '%s'. Must be either \"none\", \"latest\" or semantic versioning (e.g., 20.10.0)", version)
	}
	return nil
}

func validateAptPackageNames(packages string) error {
	re := regexp.MustCompile(`^[a-z0-9+._-]+$`)
	for _, pkg := range strings.Fields(packages) {
		if !re.MatchString(pkg) {
			return fmt.Errorf("invalid package name '%s'. Package names must contain only lowercase letters, digits, hyphens, periods, and plus signs", pkg)
		}
	}
	return nil
}

func validateGitName(name string) error {
	if strings.ContainsAny(name, "\"\n\r") {
		return fmt.Errorf("invalid git name. Cannot contain quotes or newlines")
	}
	if len(name) > 100 {
		return fmt.Errorf("git name too long (max 100 characters)")
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
		return fmt.Errorf("username too long (max 32 characters)")
	}
	return nil
}

func validatePort(port string) error {
	var p int
	_, err := fmt.Sscanf(port, "%d", &p)
	if err != nil || p < 1 || p > 65535 {
		return fmt.Errorf("invalid port '%s'. Must be a number between 1 and 65535", port)
	}
	return nil
}

func validateUIDGID(id, idType string) error {
	var i int
	_, err := fmt.Sscanf(id, "%d", &i)
	if err != nil || i < 0 || i > 65535 {
		return fmt.Errorf("invalid %s '%s'. Must be a number between 0 and 65535", idType, id)
	}
	return nil
}

// Config functions
func newConfig() *Config {
	cfg := &Config{
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
	if isWindows {
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
func getProjectDir(name string) string {
	return filepath.Join(projectsDir, name)
}

func getProjectCompose(name string) string {
	return filepath.Join(projectsDir, name, "compose.yaml")
}

func getProjectEnv(name string) string {
	return filepath.Join(projectsDir, name, ".env")
}

func checkInexistentName(name string) {
	composeFile := getProjectCompose(name)
	envFile := getProjectEnv(name)

	if _, err := os.Stat(composeFile); err == nil {
		fatal("Project '%s' already exists. You can have multiple configurations for the same project by calling 'create' with the '--name' flag. Hint: Use 'paul-envs.sh list' to see all projects or 'paul-envs.sh remove %s' to delete it", name, name)
	}
	if _, err := os.Stat(envFile); err == nil {
		fatal("Project '%s' already exists. You can have multiple configurations for the same project by calling 'create' with the '--name' flag. Hint: Use 'paul-envs.sh list' to see all projects or 'paul-envs.sh remove %s' to delete it", name, name)
	}
}

func doesLangVersionNeedsMise(version string) bool {
	return version != "none" && version != "latest" && version != ""
}

func miseCheck(cfg *Config, noPrompt bool) {
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
		warn("WARNING: You specified exact version(s) for language runtimes, but Mise is not enabled.")
		warn("Exact versions require Mise to be installed. Without Mise, Ubuntu's default packages will be used instead.")
		if !noPrompt {
			choice := promptYN("Would you like to enable Mise now?", "Y")
			if choice {
				cfg.installMise = "true"
				success("Mise enabled")
			}
		}
	}
}

func promptYN(prompt, defaultVal string) bool {
	fmt.Printf("%s (%s/n): ", prompt, defaultVal)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		input = defaultVal
	}
	return strings.ToUpper(input) == "Y"
}

func promptString(prompt, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultVal)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func promptShell(cfg *Config) {
	if cfg.shell != "" {
		return
	}

	fmt.Println()
	info("=== Shell Selection ===")
	fmt.Println("Select shell:")
	fmt.Println("  1) bash (default)")
	fmt.Println("  2) zsh")
	fmt.Println("  3) fish")
	choice := promptString("Choice", "1")

	switch choice {
	case "1":
		cfg.shell = "bash"
	case "2":
		cfg.shell = "zsh"
	case "3":
		cfg.shell = "fish"
	default:
		cfg.shell = "bash"
	}
}

func promptLanguages(cfg *Config) {
	hasAnyLang := cfg.installNode != "" || cfg.installRust != "" ||
		cfg.installPython != "" || cfg.installGo != ""

	if hasAnyLang {
		if cfg.installNode == "" {
			cfg.installNode = "none"
		}
		if cfg.installRust == "" {
			cfg.installRust = "none"
		}
		if cfg.installPython == "" {
			cfg.installPython = "none"
		}
		if cfg.installGo == "" {
			cfg.installGo = "none"
		}
		return
	}

	fmt.Println()
	info("=== Language Runtimes ===")
	fmt.Println("Which language runtimes do you need? (space-separated numbers, or Enter to skip)")
	fmt.Println("  1) Node.js")
	fmt.Println("  2) Rust")
	fmt.Println("  3) Python")
	fmt.Println("  4) Go")
	fmt.Println("  5) WebAssembly tools (Binaryen, Rust WASM target if Rust is enabled)")
	langChoices := promptString("Choice", "none")

	cfg.installNode = "none"
	cfg.installRust = "none"
	cfg.installPython = "none"
	cfg.installGo = "none"

	for _, choice := range strings.Fields(langChoices) {
		switch choice {
		case "1":
			nodeVer := promptString("Node.js version (latest/none/X.Y.Z)", "latest")
			cfg.installNode = nodeVer
		case "2":
			rustVer := promptString("Rust version (latest/none/X.Y.Z)", "latest")
			cfg.installRust = rustVer
		case "3":
			pythonVer := promptString("Python version (latest/none/X.Y.Z)", "latest")
			cfg.installPython = pythonVer
		case "4":
			goVer := promptString("Go version (latest/none/X.Y.Z)", "latest")
			cfg.installGo = goVer
		case "5":
			cfg.enableWasm = "true"
		default:
			warn("Unknown choice: %s (skipped)", choice)
		}
	}
}

func promptPackages(cfg *Config) {
	if cfg.packages != "" {
		return
	}

	fmt.Println()
	info("=== Additional Packages ===")
	fmt.Println("The following packages are already installed on top of an Ubuntu:24.04 image:")
	fmt.Println("curl git build-essential")
	fmt.Println()
	fmt.Println("Enter additional Ubuntu packages (space-separated, or Enter to skip):")
	fmt.Println("Examples: ripgrep fzf htop")
	packages := promptString("Packages", "")

	if packages != "" {
		if err := validateAptPackageNames(packages); err != nil {
			fatal(err.Error())
		}
		cfg.packages = packages
	}
}

func promptTools(cfg *Config) {
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
		return
	}

	fmt.Println()
	info("=== Development Tools ===")
	fmt.Println("Some dev tools are not pulled from Ubuntu's repositories to get their latest version instead.")
	fmt.Println("Which of those tools do you want to install? (space-separated numbers, or Enter to skip all)")
	fmt.Println("  1) Neovim (text editor)")
	fmt.Println("  2) Starship (prompt)")
	fmt.Println("  3) Atuin (shell history)")
	fmt.Println("  4) Mise (version manager - required for specific language versions)")
	fmt.Println("  5) Zellij (terminal multiplexer)")
	fmt.Println("  6) Jujutsu (Git-compatible VCS)")
	toolChoices := promptString("Choice", "none")

	cfg.installNeovim = "false"
	cfg.installStarship = "false"
	cfg.installAtuin = "false"
	cfg.installMise = "false"
	cfg.installZellij = "false"
	cfg.installJujutsu = "false"

	for _, choice := range strings.Fields(toolChoices) {
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
		default:
			warn("Unknown choice: %s (skipped)", choice)
		}
	}
}

func promptSudo(cfg *Config) {
	if cfg.enableSudo != "" {
		return
	}

	fmt.Println()
	info("=== Sudo Access ===")
	if promptYN("Enable sudo access in container (password:\"dev\")?", "N") {
		cfg.enableSudo = "true"
	} else {
		cfg.enableSudo = "false"
	}
}

func promptSSH(cfg *Config) {
	if cfg.enableSSH != "" {
		return
	}

	fmt.Println()
	info("=== SSH Access ===")
	if promptYN("Enable ssh access to container?", "N") {
		cfg.enableSSH = "true"
		promptSSHKey(cfg)
	} else {
		cfg.enableSSH = "false"
	}
}

func promptSSHKey(cfg *Config) {
	homeDir, _ := os.UserHomeDir()
	sshDir := filepath.Join(homeDir, ".ssh")

	var pubKeys []string
	if info, err := os.Stat(sshDir); err == nil && info.IsDir() {
		files, _ := filepath.Glob(filepath.Join(sshDir, "*.pub"))
		pubKeys = files
	}

	if len(pubKeys) == 0 {
		warn("No SSH public keys found in ~/.ssh/")
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

	choice := promptString("Choice", "1")
	var choiceNum int
	fmt.Sscanf(choice, "%d", &choiceNum)

	if choiceNum >= 1 && choiceNum <= len(pubKeys) {
		cfg.sshKeyPath = pubKeys[choiceNum-1]
	} else if choiceNum == len(pubKeys)+1 {
		customKey := promptString("Enter path to public key", "")
		if _, err := os.Stat(customKey); err == nil {
			cfg.sshKeyPath = customKey
		} else {
			warn("File not found: %s", customKey)
			cfg.sshKeyPath = ""
		}
	} else {
		cfg.sshKeyPath = ""
	}
}

func promptPorts() []string {
	fmt.Println()
	info("=== Port Forwarding ===")
	fmt.Println("Enter supplementary container ports to expose (space-separated, or Enter to skip):")
	fmt.Println("Examples: 3000 5432 8080")
	portInput := promptString("Ports", "")

	var ports []string
	for _, port := range strings.Fields(portInput) {
		if err := validatePort(port); err != nil {
			fatal(err.Error())
		}
		ports = append(ports, port)
	}
	return ports
}

func promptVolumes() []string {
	fmt.Println()
	info("=== Credentials & Volumes ===")
	fmt.Println("Mount common credentials/configs? (space-separated numbers, or Enter to skip)")
	fmt.Println("  1) SSH keys (~/.ssh)")
	fmt.Println("  2) Git credentials (~/.git-credentials)")
	fmt.Println("  3) AWS credentials (~/.aws)")
	fmt.Println("  4) Custom CA certificates (/etc/ssl/certs/custom-ca.crt)")
	choices := promptString("Choice", "none")

	homeDir, _ := os.UserHomeDir()
	if isWindows {
		homeDir = normalizePath(homeDir)
	}

	var volumes []string
	for _, choice := range strings.Fields(choices) {
		switch choice {
		case "1":
			volumes = append(volumes, fmt.Sprintf("%s/.ssh:/home/${USERNAME}/.ssh:ro", homeDir))
		case "2":
			volumes = append(volumes, fmt.Sprintf("%s/.git-credentials:/home/${USERNAME}/.git-credentials:ro", homeDir))
		case "3":
			volumes = append(volumes, fmt.Sprintf("%s/.aws:/home/${USERNAME}/.aws:ro", homeDir))
		case "4":
			volumes = append(volumes, "/etc/ssl/certs/custom-ca.crt:/usr/local/share/ca-certificates/custom-ca.crt:ro")
		default:
			warn("Unknown choice: %s (skipped)", choice)
		}
	}

	fmt.Println()
	fmt.Println("Add custom volumes? (one per line, Enter on empty line to finish)")
	fmt.Println("Format: /host/path:/container/path[:ro]")
	for {
		vol := promptString("Volume", "")
		if vol == "" {
			break
		}
		// Normalize host path
		if idx := strings.Index(vol, ":"); idx != -1 {
			hostPart := vol[:idx]
			containerPart := vol[idx:]
			hostPart = normalizePath(hostPart)
			vol = hostPart + containerPart
		}
		volumes = append(volumes, vol)
	}

	return volumes
}

func generateProjectCompose(name string, cfg *Config, ports, volumes []string) {
	composeFile := getProjectCompose(name)
	envFile := getProjectEnv(name)

	checkInexistentName(name)

	safeGitName := lightSanitize(cfg.gitName)
	safeGitEmail := lightSanitize(cfg.gitEmail)
	safePackages := lightSanitize(cfg.packages)

	os.MkdirAll(filepath.Dir(composeFile), 0755)

	// Generate .env file
	envContent := fmt.Sprintf(`# "Env file" for your project, which will be fed to docker compose
# alongside "compose.yaml" in the same directory.
#
# Can be freely updated.

# Uniquely identify this container.
# *SHOULD NOT BE UPDATED*
PROJECT_ID="%s"

# Name of the project directory inside the container.
# A PROJECT_DIRNAME should always be set
PROJECT_DIRNAME="%s"

# Path to the project you want to mount in this container
# Will be mounted in "$HOME/projects/<PROJECT_DIRNAME>" inside that container.
# A PROJECT_PATH should always be set
PROJECT_PATH="%s"

# To align with your current uid.
# This is to ensure the mounted volume from your host has compatible
# permissions.
# On POSIX-like systems, just run 'id -u' with the wanted user to know it.
HOST_UID="%s"

# To align with your current gid (same reason than for "uid").
# On POSIX-like systems, just run 'id -g' with the wanted user to know it.
HOST_GID="%s"

# Username created in the container.
# Not really important, just set it if you want something other than "dev".
USERNAME="%s"

# The default shell wanted.
# Only "bash", "zsh" or "fish" are supported for now.
USER_SHELL="%s"

# Whether to install Node.js, and the version wanted.
# Note that a WebAssembly target is also automatically ready.
#
# Values can be:
# - if 'none': don't install Node.js
# - if 'latest': Install Ubuntu's default package for Node.js
# - If anything else: The exact version to install (e.g. "1.90.0").
#   That last type of value will only work if INSTALL_MISE is 'true'.
INSTALL_NODE="%s"

# Whether to install Rust and Cargo, and the version wanted.
# Note that a WebAssembly target is also automatically ready.
#
# Values can be:
# - if 'none': don't install Rust
# - if 'latest': Install Ubuntu's default package for Rust
#   Ubuntu base's repositories
# - If anything else: The exact version to install (e.g. "1.90.0").
#   That last type of value will only work if INSTALL_MISE is 'true'.
INSTALL_RUST="%s"

# Whether to install Python, and the version wanted.
#
# Values can be:
# - if 'none': don't install Python
# - if 'latest': Install Ubuntu's default package for Python
# - If anything else: The exact version to install (e.g. "3.12.0").
#   That last type of value will only work if INSTALL_MISE is 'true'.
INSTALL_PYTHON="%s"

# Whether to install Go, and the version wanted.
# Note that GOPATH is automatically set to ~/go
#
# Values can be:
# - if 'none': don't install Go
# - if 'latest': Install Ubuntu's default package for Go
# - If anything else: The exact version to install (e.g. "1.21.5").
#   That last type of value will only work if INSTALL_MISE is 'true'.
INSTALL_GO="%s"

# If 'true', add WebAssembly-specialized tools such as binaryen and a
# WebAssembly target for Rust if it is installed.
ENABLE_WASM="%s"

# If 'true', openssh will be installed, and the container will listen for ssh
# connections at port 22.
ENABLE_SSH="%s"

# If 'true', sudo will be installed, with a password set to "dev".
ENABLE_SUDO="%s"

# Additional packages outside the core base, separated by a space.
# Have to be in Ubuntu's default repository
# (e.g. "ripgrep fzf". Can be left empty for no supplementary packages)
SUPPLEMENTARY_PACKAGES="%s"

# Tools toggle.
# "true" == install it
# anything else == don't.
INSTALL_NEOVIM="%s"
INSTALL_STARSHIP="%s"
INSTALL_ATUIN="%s"
INSTALL_MISE="%s"
INSTALL_ZELLIJ="%s"
INSTALL_JUJUTSU="%s"

# Git author and committer name used inside the container
# Can also be empty to not set that in the container.
GIT_AUTHOR_NAME="%s"

# Git author and committer e-mail used inside the container
# Can also be empty to not set that in the container.
GIT_AUTHOR_EMAIL="%s"
`, name, cfg.projectDestPath, cfg.projectHostPath,
		cfg.hostUID, cfg.hostGID, cfg.username, cfg.shell,
		cfg.installNode, cfg.installRust, cfg.installPython, cfg.installGo,
		cfg.enableWasm, cfg.enableSSH, cfg.enableSudo, safePackages,
		cfg.installNeovim, cfg.installStarship, cfg.installAtuin,
		cfg.installMise, cfg.installZellij, cfg.installJujutsu,
		safeGitName, safeGitEmail)

	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		fatal("Failed to write env file: %v", err)
	}

	// Generate compose.yaml
	var composeBuilder strings.Builder
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
		sshKeyPath := normalizePath(cfg.sshKeyPath)
		composeBuilder.WriteString("      # Your local public key for ssh:\n")
		composeBuilder.WriteString(fmt.Sprintf("      - %s:/etc/ssh/authorized_keys/${USERNAME}:ro\n", sshKeyPath))
	} else if cfg.enableSSH == "true" {
		warn("No SSH key configured")
		warn("Adding a note to your compose file: %s", composeFile)
		composeBuilder.WriteString("      # Add your SSH public key here, for example:\n")
		composeBuilder.WriteString("      # - ~/.ssh/id_ed25519.pub:/etc/ssh/authorized_keys/${USERNAME}:ro\n")
	}
	composeBuilder.WriteString("\n")

	if err := os.WriteFile(composeFile, []byte(composeBuilder.String()), 0644); err != nil {
		fatal("Failed to write compose file: %v", err)
	}
}

// Command implementations
func cmdCreate(args []string) {
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
	for i = 0; i < len(args); i++ {
		if args[i] == "--port" && i+1 < len(args) {
			ports = append(ports, args[i+1])
			i++
		} else if args[i] == "--volume" && i+1 < len(args) {
			vol := args[i+1]
			if idx := strings.Index(vol, ":"); idx != -1 {
				hostPart := vol[:idx]
				containerPart := vol[idx:]
				hostPart = normalizePath(hostPart)
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
		fatal("Usage: paul-envs.sh create <project-path> [options]")
	}

	projectPath, err := getAbsolutePath(args[0])
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
	checkInexistentName(name)

	// Set defaults or prompt
	if noPrompt {
		if cfg.shell == "" {
			cfg.shell = "bash"
		}
		if cfg.installNode == "" {
			cfg.installNode = "none"
		}
		if cfg.installRust == "" {
			cfg.installRust = "none"
		}
		if cfg.installPython == "" {
			cfg.installPython = "none"
		}
		if cfg.installGo == "" {
			cfg.installGo = "none"
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
		miseCheck(cfg, noPrompt)
	} else {
		promptShell(cfg)
		promptLanguages(cfg)
		promptTools(cfg)
		miseCheck(cfg, noPrompt)
		promptSudo(cfg)
		promptSSH(cfg)
		promptPackages(cfg)

		if len(ports) == 0 {
			ports = promptPorts()
		}
		if len(volumes) == 0 {
			volumes = promptVolumes()
		}
	}

	// Validate path exists or warn
	os.MkdirAll(projectsDir, 0755)

	if _, err := os.Stat(cfg.projectHostPath); os.IsNotExist(err) && !noPrompt {
		warn("Warning: Path %s does not exist", cfg.projectHostPath)
		if !promptYN("Create config anyway?", "N") {
			os.Exit(1)
		}
	}

	// Generate config
	generateProjectCompose(name, cfg, ports, volumes)

	success("Created project '%s'", name)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Review/edit configuration:\n")
	fmt.Printf("     - %s\n", getProjectEnv(name))
	fmt.Printf("     - %s\n", getProjectCompose(name))
	fmt.Printf("  2. Put the $HOME dotfiles you want to port in:\n")
	fmt.Printf("     - %s/configs/\n", scriptDir)
	fmt.Printf("  3. Build the environment:\n")
	fmt.Printf("     paul-envs.sh build %s\n", name)
	fmt.Printf("  4. Run the environment:\n")
	fmt.Printf("     paul-envs.sh run %s\n", name)
}

func cmdList() {
	baseCompose := filepath.Join(scriptDir, "compose.yaml")
	if _, err := os.Stat(baseCompose); os.IsNotExist(err) {
		fatal("Base compose.yaml not found at %s", baseCompose)
	}

	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		fmt.Println("No project created yet")
		fmt.Println("Hint: Create one with 'paul-envs.sh create <path>'")
		return
	}

	fmt.Println("Projects created:")
	found := false
	entries, _ := os.ReadDir(projectsDir)
	for _, entry := range entries {
		if entry.IsDir() {
			composeFile := filepath.Join(projectsDir, entry.Name(), "compose.yaml")
			if _, err := os.Stat(composeFile); err == nil {
				found = true
				envFile := filepath.Join(projectsDir, entry.Name(), ".env")
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
		fmt.Println("Hint: Create one with 'paul-envs.sh create <path>'")
	}
}

func cmdBuild(args []string) {
	baseCompose := filepath.Join(scriptDir, "compose.yaml")
	if _, err := os.Stat(baseCompose); os.IsNotExist(err) {
		fatal("Base compose.yaml not found at %s", baseCompose)
	}

	var name string
	if len(args) == 0 {
		fmt.Println("No project name given, listing projects...")
		fmt.Println()
		cmdList()
		fmt.Println()
		name = promptString("Enter project name to build", "")
		if name == "" {
			fatal("No project name provided")
		}
	} else {
		name = args[0]
	}

	if err := validateProjectName(name); err != nil {
		fatal(err.Error())
	}

	composeFile := getProjectCompose(name)
	envFile := getProjectEnv(name)
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		fatal("Project '%s' not found. Hint: Use 'paul-envs.sh list' to see available projects or 'paul-envs.sh create' to make a new one", name)
	}
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		fatal("Project '%s' not found. Hint: Use 'paul-envs.sh list' to see available projects or 'paul-envs.sh create' to make a new one", name)
	}

	// Create shared cache volume
	exec.Command("docker", "volume", "create", "paulenv-shared-cache").Run()

	// Build
	os.Setenv("COMPOSE_PROJECT_NAME", "paulenv-"+name)
	cmd := exec.Command("docker", "compose", "-f", baseCompose, "-f", composeFile, "--env-file", envFile, "build")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatal("Build failed: %v", err)
	}
	success("Built project '%s'", name)

	fmt.Println()
	warn("Resetting persistent volumes...")
	cmd = exec.Command("docker", "compose", "-f", baseCompose, "-f", composeFile, "--env-file", envFile, "--profile", "reset", "up", "reset-cache", "reset-local")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
	success("Volumes reset complete")
}

func cmdRun(args []string) {
	baseCompose := filepath.Join(scriptDir, "compose.yaml")
	if _, err := os.Stat(baseCompose); os.IsNotExist(err) {
		fatal("Base compose.yaml not found at %s", baseCompose)
	}

	var name string
	var cmdArgs []string

	if len(args) == 0 {
		fmt.Println("No project name given, listing projects...")
		fmt.Println()
		cmdList()
		fmt.Println()
		name = promptString("Enter project name to run", "")
		if name == "" {
			fatal("No project name provided")
		}
	} else {
		name = args[0]
		if len(args) > 1 {
			cmdArgs = args[1:]
		}
	}

	if err := validateProjectName(name); err != nil {
		fatal(err.Error())
	}

	composeFile := getProjectCompose(name)
	envFile := getProjectEnv(name)
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		fatal("Project '%s' not found\nHint: Use 'paul-envs.sh list' to see available projects", name)
	}
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		fatal("Project '%s' not found\nHint: Use 'paul-envs.sh list' to see available projects", name)
	}

	os.Setenv("COMPOSE_PROJECT_NAME", "paulenv-"+name)

	args = []string{"compose", "-f", baseCompose, "-f", composeFile, "--env-file", envFile, "run", "--rm", "paulenv"}
	args = append(args, cmdArgs...)

	cmd := exec.Command("docker", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func cmdRemove(args []string) {
	var name string
	if len(args) == 0 {
		fmt.Println("No project name given, listing projects...")
		fmt.Println()
		cmdList()
		fmt.Println()
		name = promptString("Enter project name to remove", "")
		if name == "" {
			fatal("No project name provided")
		}
	} else {
		name = args[0]
	}

	if err := validateProjectName(name); err != nil {
		fatal(err.Error())
	}

	projectDir := getProjectDir(name)
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		fatal("Project '%s' not found\nHint: Use 'paul-envs.sh list' to see available projects", name)
	}

	if !promptYN(fmt.Sprintf("Remove project '%s'?", name), "N") {
		return
	}

	if err := os.RemoveAll(projectDir); err != nil {
		fatal("Failed to remove project: %v", err)
	}
	success("Removed project '%s'", name)
	fmt.Println("Note: Docker volumes are preserved. To remove them, run:")
	fmt.Printf("  docker volume rm paulenv-%s-local\n", name)
}

func cmdVersion() {
	fmt.Printf("paul-envs.sh version %s\n", version)
	fmt.Printf("Go version: %s\n", runtime.Version())
	cmd := exec.Command("docker", "--version")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Docker version: not installed")
	} else {
		fmt.Printf("Docker version: %s", output)
	}
}

func printUsage() {
	fmt.Println(`paul-envs.sh - Development Environment Manager

Usage:
  paul-envs.sh create <path> [options]
  paul-envs.sh list
  paul-envs.sh build <name>
  paul-envs.sh run <name> [command]
  paul-envs.sh remove <name>

Options for create (all optional):
  --no-prompt              Non-interactive mode (uses defaults)
  --name NAME              Name of this project (default: directory name)
  --uid UID                Container UID (default: current user - or 1000 on windows)
  --gid GID                Container GID (default: current group - or 1000 on windows)
  --username NAME          Container username (default: dev)
  --shell SHELL            User shell: bash|zsh|fish (prompted if not specified)
  --nodejs VERSION         Node.js installation
  --rust VERSION           Rust installation
  --python VERSION         Python installation
  --go VERSION             Go installation
  --enable-wasm            Add WASM-specialized tools
  --enable-ssh             Enable ssh access on port 22
  --enable-sudo            Enable sudo access in container
  --git-name NAME          Git user.name (optional)
  --git-email EMAIL        Git user.email (optional)
  --neovim                 Install Neovim
  --starship               Install Starship
  --atuin                  Install Atuin
  --mise                   Install Mise
  --zellij                 Install Zellij
  --jujutsu                Install Jujutsu
  --packages "PKG1 PKG2"   Additional Ubuntu packages
  --port PORT              Expose container port (can be repeated)
  --volume HOST:CONT[:ro]  Mount volume (can be repeated)

Configuration:
  Base compose: ` + filepath.Join(scriptDir, "compose.yaml") + `
  Projects directory: ` + projectsDir)
}
