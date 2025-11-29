package args

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/peaberberian/paul-envs/internal/config"
	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
	"github.com/peaberberian/paul-envs/internal/utils"
)

// ParseAndPrompt parses command-line arguments and prompts for missing configuration
func ParseAndPrompt(args []string, cons *console.Console, filestor *files.FileStore) (config.Config, error) {
	if len(args) == 0 {
		// TODO: Here prompt path to project dir?
		return config.Config{}, errors.New("no project path provided. Use --help for more info")
	}

	projectPath, err := filepath.Abs(args[0])
	if err != nil {
		return config.Config{}, fmt.Errorf("invalid project path: %w", err)
	}

	parsed, noPrompt, err := parseFlags(args[1:])
	if err != nil {
		return config.Config{}, err
	}

	if err := ensureProjectPath(projectPath, noPrompt, cons); err != nil {
		return config.Config{}, err
	}

	// Build initial config
	cfg, err := buildConfig(projectPath, parsed)
	if err != nil {
		return config.Config{}, err
	}

	// Validate project name
	if err := validateProjectName(cfg.ProjectDestPath, filestor, cons); err != nil {
		return config.Config{}, err
	}

	// Prompt for missing values if interactive
	if !noPrompt {
		if err := promptMissing(cons, &cfg); err != nil {
			return config.Config{}, err
		}
	}

	// Final validation for mise requirement
	if !cfg.InstallMise && !noPrompt {
		checkMiseRequirement(cons, &cfg)
	}

	return cfg, nil
}

// parsedFlags holds raw flag values
type parsedFlags struct {
	noPrompt        bool
	name            string
	uid             string
	gid             string
	username        string
	shell           string
	nodeVersion     string
	rustVersion     string
	pythonVersion   string
	goVersion       string
	enableWasm      bool
	enableSsh       bool
	enableSudo      bool
	gitName         string
	gitEmail        string
	installNeovim   bool
	installStarship bool
	installAtuin    bool
	installMise     bool
	installZellij   bool
	installJujutsu  bool
	packages        []string
	ports           []string
	volumes         []string
}

func parseFlags(args []string) (*parsedFlags, bool, error) {
	var noPrompt bool
	p := &parsedFlags{}

	flagset := flag.NewFlagSet("create", flag.ContinueOnError)
	flagset.BoolVar(&noPrompt, "no-prompt", false, "Non-interactive mode")
	flagset.StringVar(&p.name, "name", "", "Project name")
	flagset.StringVar(&p.uid, "uid", "", "Container UID")
	flagset.StringVar(&p.gid, "gid", "", "Container GID")
	flagset.StringVar(&p.username, "username", "", "Container username")
	flagset.StringVar(&p.shell, "shell", "", "User shell")
	flagset.StringVar(&p.nodeVersion, "nodejs", "", "Node.js version")
	flagset.StringVar(&p.rustVersion, "rust", "", "Rust version")
	flagset.StringVar(&p.pythonVersion, "python", "", "Python version")
	flagset.StringVar(&p.goVersion, "go", "", "Go version")
	flagset.BoolVar(&p.enableWasm, "enable-wasm", false, "Enable WebAssembly tools")
	flagset.BoolVar(&p.enableWasm, "wasm", false, "Enable WebAssembly tools")
	flagset.BoolVar(&p.enableSsh, "enable-ssh", false, "Enable SSH access")
	flagset.BoolVar(&p.enableSsh, "ssh", false, "Enable SSH access")
	flagset.BoolVar(&p.enableSudo, "enable-sudo", false, "Enable sudo access")
	flagset.BoolVar(&p.enableSudo, "sudo", false, "Enable sudo access")
	flagset.StringVar(&p.gitName, "git-name", "", "Git user name")
	flagset.StringVar(&p.gitEmail, "git-email", "", "Git user email")
	flagset.BoolVar(&p.installNeovim, "neovim", false, "Install Neovim")
	flagset.BoolVar(&p.installStarship, "starship", false, "Install Starship")
	flagset.BoolVar(&p.installAtuin, "atuin", false, "Install Atuin")
	flagset.BoolVar(&p.installMise, "mise", false, "Install Mise")
	flagset.BoolVar(&p.installZellij, "zellij", false, "Install Zellij")
	flagset.BoolVar(&p.installJujutsu, "jujutsu", false, "Install Jujutsu")

	// Parse repeatable flags manually
	filtered := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--port" && i+1 < len(args) {
			p.ports = append(p.ports, args[i+1])
			i++
		} else if args[i] == "--volume" && i+1 < len(args) {
			p.volumes = append(p.volumes, args[i+1])
			i++
		} else if args[i] == "--package" && i+1 < len(args) {
			p.packages = append(p.packages, args[i+1])
			i++
		} else {
			filtered = append(filtered, args[i])
		}
	}

	if err := flagset.Parse(filtered); err != nil {
		return nil, false, err
	}

	return p, noPrompt, nil
}

func buildConfig(projectPath string, p *parsedFlags) (config.Config, error) {
	cfg := config.New("dev", config.ShellBash)
	cfg.ProjectHostPath = projectPath

	// Set values from flags with validation
	if p.uid != "" {
		if err := utils.ValidateUIDGID(p.uid); err != nil {
			return config.Config{}, fmt.Errorf("invalid UID '%s': %w", p.uid, err)
		}
		cfg.UID = p.uid
	}
	if p.gid != "" {
		if err := utils.ValidateUIDGID(p.gid); err != nil {
			return config.Config{}, fmt.Errorf("invalid GID '%s': %w", p.gid, err)
		}
		cfg.GID = p.gid
	}
	if p.username != "" {
		if err := utils.ValidateUsername(p.username); err == nil {
			cfg.Username = p.username
		}
	}
	if p.shell != "" {
		if shell, err := parseShell(p.shell); err == nil {
			cfg.Shell = shell
		}
	}

	// Language versions
	if p.nodeVersion != "" && utils.ValidateVersionArg(p.nodeVersion) == nil {
		cfg.InstallNode = p.nodeVersion
	}
	if p.rustVersion != "" && utils.ValidateVersionArg(p.rustVersion) == nil {
		cfg.InstallRust = p.rustVersion
	}
	if p.pythonVersion != "" && utils.ValidateVersionArg(p.pythonVersion) == nil {
		cfg.InstallPython = p.pythonVersion
	}
	if p.goVersion != "" && utils.ValidateVersionArg(p.goVersion) == nil {
		cfg.InstallGo = p.goVersion
	}

	cfg.EnableWasm = p.enableWasm
	cfg.EnableSsh = p.enableSsh
	cfg.EnableSudo = p.enableSudo

	// Git config
	if p.gitName != "" && utils.ValidateGitName(p.gitName) == nil {
		cfg.GitName = p.gitName
	}
	if p.gitEmail != "" && utils.ValidateGitEmail(p.gitEmail) == nil {
		cfg.GitEmail = p.gitEmail
	}

	// Packages
	validPackages, invalidPackages := filterValidPackages(p.packages)
	if len(invalidPackages) > 0 {
		return config.Config{}, fmt.Errorf("invalid package list: %s", strings.Join(invalidPackages, " "))
	}
	cfg.Packages = validPackages

	// Tools
	cfg.InstallNeovim = p.installNeovim
	cfg.InstallStarship = p.installStarship
	cfg.InstallAtuin = p.installAtuin
	cfg.InstallMise = p.installMise
	cfg.InstallZellij = p.installZellij
	cfg.InstallJujutsu = p.installJujutsu

	// Project name
	projectName := p.name
	if projectName == "" {
		projectName = filepath.Base(projectPath)
	}
	cfg.ProjectName = projectName
	cfg.ProjectDestPath = sanitizeProjectName(projectName)

	// Ports and volumes
	validPorts, invalidPorts := filterValidPorts(p.ports)
	if len(invalidPorts) > 0 {
		return config.Config{}, fmt.Errorf("invalid port list: %s", strings.Join(invalidPorts, " "))
	}
	cfg.Ports = validPorts

	// TODO: sanitization?
	cfg.Volumes = p.volumes

	return cfg, nil
}

func validateProjectName(name string, filestor *files.FileStore, cons *console.Console) error {
	if err := utils.ValidateProjectName(name); err != nil {
		return fmt.Errorf("invalid project name: %w", err)
	}
	if filestor.CheckProjectNameAvailable(name, cons) != nil {
		return errors.New("project name already taken")
	}
	return nil
}

func promptMissing(cons *console.Console, cfg *config.Config) error {
	// Shell
	if cfg.Shell == config.ShellBash {
		cons.WriteLn("")
		if err := promptShell(cons, cfg); err != nil {
			return err
		}
	}

	// Languages
	if !hasAnyLanguage(cfg) {
		cons.WriteLn("")
		if err := promptLanguages(cons, cfg); err != nil {
			return err
		}
	}

	// Set more explicit default values for disabled languages
	if cfg.InstallNode == "" {
		cfg.InstallNode = "none"
	}
	if cfg.InstallRust == "" {
		cfg.InstallRust = "none"
	}
	if cfg.InstallPython == "" {
		cfg.InstallPython = "none"
	}
	if cfg.InstallGo == "" {
		cfg.InstallGo = "none"
	}

	// Tools
	if !hasAnyTool(cfg) {
		cons.WriteLn("")
		if err := promptTools(cons, cfg); err != nil {
			return err
		}
	}

	// Sudo
	if !cfg.EnableSudo {
		cons.WriteLn("")
		if err := promptSudo(cons, cfg); err != nil {
			return err
		}
	}

	// SSH
	if !cfg.EnableSsh {
		cons.WriteLn("")
		if err := promptSSH(cons, cfg); err != nil {
			return err
		}
	}

	// Packages
	if len(cfg.Packages) == 0 {
		cons.WriteLn("")
		packages, err := promptPackages(cons)
		if err != nil {
			return err
		}
		cfg.Packages = packages
	}

	// Ports
	if len(cfg.Ports) == 0 {
		cons.WriteLn("")
		ports, err := promptPorts(cons)
		if err != nil {
			return err
		}
		cfg.Ports = ports
	}

	// Volumes
	if len(cfg.Volumes) == 0 {
		cons.WriteLn("")
		volumes, err := promptVolumes(cons)
		if err != nil {
			return err
		}
		cfg.Volumes = volumes
	}

	return nil
}

func checkMiseRequirement(cons *console.Console, cfg *config.Config) {
	needsMise := needsExactVersion(cfg.InstallNode) ||
		needsExactVersion(cfg.InstallRust) ||
		needsExactVersion(cfg.InstallPython) ||
		needsExactVersion(cfg.InstallGo)

	if !needsMise {
		return
	}

	cons.WriteLn("")
	cons.Warn("WARNING: You specified exact version(s) for language runtimes, but Mise is not enabled.")
	cons.Warn("Exact versions require Mise to be installed. Without Mise, Ubuntu's default packages will be used instead.")

	choice, err := cons.AskYesNo("Would you like to enable Mise now?", true)
	if err != nil {
		cons.Warn("Error: %v", err)
		return
	}
	if choice {
		cfg.InstallMise = true
		cons.Success("Mise enabled")
	}
}

// Helper functions

func parseShell(s string) (config.Shell, error) {
	switch s {
	case "bash":
		return config.ShellBash, nil
	case "zsh":
		return config.ShellZsh, nil
	case "fish":
		return config.ShellFish, nil
	default:
		return config.ShellBash, fmt.Errorf("invalid shell '%s'. Must be one of: bash, zsh, fish", s)
	}
}

func sanitizeProjectName(input string) string {
	s := strings.ToLower(input)
	s = regexp.MustCompile(`[^a-z0-9_-]`).ReplaceAllString(s, "-")

	// Remove leading non-alphanumeric
	s = strings.TrimLeftFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})

	// Collapse consecutive hyphens
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}

	// Truncate and trim
	if len(s) > 128 {
		s = s[:128]
	}
	s = strings.TrimRight(s, "-")

	if s == "" {
		s = "project"
	}

	return s
}

func filterValidPorts(ports []string) ([]uint16, []string) {
	valid := make([]uint16, 0, len(ports))
	invalid := make([]string, 0)
	for _, port := range ports {
		p, err := strconv.Atoi(port)
		if err == nil && utils.ValidatePort(p) == nil {
			valid = append(valid, uint16(p))
		} else {
			invalid = append(invalid, port)
		}
	}
	return valid, invalid
}

func filterValidPackages(packages []string) ([]string, []string) {
	valid := make([]string, 0, len(packages))
	invalid := make([]string, 0)
	for _, p := range packages {
		if utils.IsValidUbuntuPackageName(p) {
			valid = append(valid, p)
		} else {
			invalid = append(invalid, p)
		}
	}
	return valid, invalid
}

func hasAnyLanguage(cfg *config.Config) bool {
	return cfg.InstallNode != "" || cfg.InstallRust != "" ||
		cfg.InstallPython != "" || cfg.InstallGo != "" || cfg.EnableWasm
}

func hasAnyTool(cfg *config.Config) bool {
	return cfg.InstallNeovim || cfg.InstallStarship ||
		cfg.InstallAtuin || cfg.InstallMise ||
		cfg.InstallZellij || cfg.InstallJujutsu
}

func needsExactVersion(version string) bool {
	return version != "" && version != config.VersionNone && version != config.VersionLatest
}

// Prompt functions

// TODO: Return Shell instead of filling Config itself?
func promptShell(cons *console.Console, cfg *config.Config) error {
	for {
		cons.Info("=== Shell Selection ===")
		cons.WriteLn("Select shell:")
		cons.WriteLn("  1) bash (default)")
		cons.WriteLn("  2) zsh")
		cons.WriteLn("  3) fish")

		choice, err := cons.AskString("Choice", "1")
		if err != nil {
			return fmt.Errorf("unable to prompt for shell choice: %w", err)
		}

		switch choice {
		case "1", "bash":
			cfg.Shell = config.ShellBash
			return nil
		case "2", "zsh":
			cfg.Shell = config.ShellZsh
			return nil
		case "3", "fish":
			cfg.Shell = config.ShellFish
			return nil
		default:
			cons.Warn("Unrecognized choice: \"%s\"", choice)
			cons.Warn("Please select an element from the list, or leave empty for the default.")
			cons.WriteLn("")
		}
	}
}

// TODO: Return languages instead through a new type?
func promptLanguages(cons *console.Console, cfg *config.Config) error {
	for {
		cons.Info("=== Language Runtimes ===")
		cons.WriteLn("Which language runtimes do you need? (space-separated numbers, or Enter to skip)")
		cons.WriteLn("  1) Node.js")
		cons.WriteLn("  2) Rust")
		cons.WriteLn("  3) Python")
		cons.WriteLn("  4) Go")
		cons.WriteLn("  5) WebAssembly tools (Binaryen, Rust WASM target if Rust is enabled)")

		choices, err := cons.AskString("Choice", "none")
		if err != nil {
			return fmt.Errorf("unable to prompt for language choice: %w", err)
		}

		// Track if we successfully processed all choices
		allValid := true
		selectedChoices := strings.FieldsSeq(choices)

		for choice := range selectedChoices {
			switch choice {
			case "1":
				ver, err := cons.AskString("Node.js version (latest/none/X.Y.Z)", config.VersionLatest)
				if err != nil {
					return fmt.Errorf("unable to prompt for Node.js version: %w", err)
				}
				if err := utils.ValidateVersionArg(ver); err != nil {
					// TODO: just reask version, not the whole thing
					cons.Warn("Invalid version format: %v", err)
					allValid = false
					break
				}
				cfg.InstallNode = ver
			case "2":
				ver, err := cons.AskString("Rust version (latest/none/X.Y.Z)", config.VersionLatest)
				if err != nil {
					return fmt.Errorf("unable to prompt for Rust version: %w", err)
				}
				if err := utils.ValidateVersionArg(ver); err != nil {
					// TODO: just reask version, not the whole thing
					cons.Warn("Invalid version format: %v", err)
					allValid = false
					break
				}
				cfg.InstallRust = ver
			case "3":
				ver, err := cons.AskString("Python version (latest/none/X.Y.Z)", config.VersionLatest)
				if err != nil {
					return fmt.Errorf("unable to prompt for Python version: %w", err)
				}
				if err := utils.ValidateVersionArg(ver); err != nil {
					// TODO: just reask version, not the whole thing
					cons.Warn("Invalid version format: %v", err)
					allValid = false
					break
				}
				cfg.InstallPython = ver
			case "4":
				ver, err := cons.AskString("Go version (latest/none/X.Y.Z)", config.VersionLatest)
				if err != nil {
					return fmt.Errorf("unable to prompt for Go version: %w", err)
				}
				if err := utils.ValidateVersionArg(ver); err != nil {
					// TODO: just reask version, not the whole thing
					cons.Warn("Invalid version format: %v", err)
					allValid = false
					break
				}
				cfg.InstallGo = ver
			case "5":
				cfg.EnableWasm = true
			case "none":
				return nil
			default:
				cons.Warn("Unrecognized choice: \"%s\"", choice)
				allValid = false
			}
		}

		if !allValid {
			cons.Warn("Please select valid elements from the list, or leave empty no language.")
			cons.WriteLn("")
			// Reset any partial changes
			cfg.InstallNode = ""
			cfg.InstallRust = ""
			cfg.InstallPython = ""
			cfg.InstallGo = ""
			cfg.EnableWasm = false
			continue
		}

		return nil
	}
}

// TODO: Return tools instead through a new type?
func promptTools(cons *console.Console, cfg *config.Config) error {
	for {
		cons.Info("=== Development Tools ===")
		cons.WriteLn("Some dev tools are not pulled from Ubuntu's repositories to get their latest version instead.")
		cons.WriteLn("Which of those tools do you want to install? (space-separated numbers, or Enter to skip all)")
		cons.WriteLn("  1) Neovim (text editor)")
		cons.WriteLn("  2) Starship (prompt)")
		cons.WriteLn("  3) Atuin (shell history)")
		cons.WriteLn("  4) Mise (version manager - required for specific language versions)")
		cons.WriteLn("  5) Zellij (terminal multiplexer)")
		cons.WriteLn("  6) Jujutsu (Git-compatible VCS)")

		choices, err := cons.AskString("Choice", "none")
		if err != nil {
			return fmt.Errorf("unable to prompt for tools choice: %w", err)
		}

		allValid := true
		selectedChoices := strings.FieldsSeq(choices)

		for choice := range selectedChoices {
			switch choice {
			case "1":
				cfg.InstallNeovim = true
			case "2":
				cfg.InstallStarship = true
			case "3":
				cfg.InstallAtuin = true
			case "4":
				cfg.InstallMise = true
			case "5":
				cfg.InstallZellij = true
			case "6":
				cfg.InstallJujutsu = true
			case "none":
				return nil
			default:
				cons.Warn("Unrecognized choice: \"%s\"", choice)
				allValid = false
			}
		}

		if !allValid {
			cons.Warn("Please select valid elements from the list, or leave empty for no tool.")
			cons.WriteLn("")
			// Reset any partial changes
			cfg.InstallNeovim = false
			cfg.InstallStarship = false
			cfg.InstallAtuin = false
			cfg.InstallMise = false
			cfg.InstallZellij = false
			cfg.InstallJujutsu = false
			continue
		}

		return nil
	}
}

// TODO: Return bool instead of filling Config itself?
func promptSudo(cons *console.Console, cfg *config.Config) error {
	cons.Info("=== Sudo Access ===")
	val, err := cons.AskYesNo("Enable sudo access in container (password:\"dev\")?", false)
	if err != nil {
		return fmt.Errorf("unable to prompt for sudo choice: %w", err)
	}
	cfg.EnableSudo = val
	return nil
}

// TODO: Return string instead of filling Config itself?
func promptSSH(cons *console.Console, cfg *config.Config) error {
	cons.Info("=== SSH Access ===")
	val, err := cons.AskYesNo("Enable ssh access to container?", false)
	if err != nil {
		return fmt.Errorf("unable to prompt for SSH choice: %w", err)
	}
	cfg.EnableSsh = val
	if val {
		sshKeyPath, err := promptSSHKeys(cons)
		if err != nil {
			cons.Warn("Failed to configure SSH key: %v", err)
		} else if sshKeyPath != "" {
			cfg.SshKeyPath = sshKeyPath
		}
	}
	return nil
}

func promptSSHKeys(cons *console.Console) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to obtain home dir: %w", err)
	}

	pubKeys, err := filepath.Glob(filepath.Join(homeDir, ".ssh", "*.pub"))
	if err != nil || len(pubKeys) == 0 {
		cons.Warn("No SSH public keys found in ~/.ssh/")
		return "", errors.New("no ssh public key found")
	}

	for {
		cons.WriteLn("")
		cons.WriteLn("Select SSH public key to mount:")
		for i, key := range pubKeys {
			cons.WriteLn("  %d) %s", i+1, filepath.Base(key))
		}
		cons.WriteLn("  %d) Custom path", len(pubKeys)+1)
		cons.WriteLn("  %d) Skip (add manually later)", len(pubKeys)+2)

		choice, err := cons.AskString("Choice", "")
		if err != nil {
			return "", fmt.Errorf("unable to prompt for SSH key choice: %w", err)
		}

		choiceNum, err := strconv.Atoi(choice)
		if err != nil {
			cons.Warn("Invalid choice: \"%s\"", choice)
			cons.Warn("Please Enter a valid number.")
			continue
		}

		if choiceNum >= 1 && choiceNum <= len(pubKeys) {
			return pubKeys[choiceNum-1], nil
		} else if choiceNum == len(pubKeys)+1 {
			customKey, err := cons.AskString("Enter path to public key", "")
			if err != nil {
				return "", fmt.Errorf("failed to ask for public key input: %w", err)
			}
			if _, err := os.Stat(customKey); err == nil {
				return customKey, nil
			}
			cons.Warn("File not found: %s", customKey)
		} else if choiceNum == len(pubKeys)+2 {
			return "", nil
		} else {
			cons.Warn("Invalid choice number: \"%s\"", choice)
			cons.Warn("Please Enter a listed number.")
		}
	}
}

func promptPackages(cons *console.Console) ([]string, error) {
	for {
		cons.Info("=== Additional Packages ===")
		cons.WriteLn("The following packages are already installed on top of an Ubuntu:24.04 image:")
		cons.WriteLn("curl git build-essential")
		cons.WriteLn("")
		cons.WriteLn("Enter additional Ubuntu packages (space-separated, or Enter to skip):")
		cons.WriteLn("Examples: ripgrep fzf htop")

		input, err := cons.AskString("Packages", "")
		if err != nil {
			return nil, fmt.Errorf("unable to prompt for packages: %w", err)
		}

		packages := strings.Fields(input)
		validPackages, invalidPackages := filterValidPackages(packages)
		if len(invalidPackages) > 0 {
			cons.Warn("Invalid package names: \"%s\"", strings.Join(invalidPackages, " "))
			cons.Warn("Please input a valid list of space-separated Ubuntu packages.")
			cons.WriteLn("")
			continue
		}
		return validPackages, nil
	}
}

func promptPorts(cons *console.Console) ([]uint16, error) {
	for {
		cons.Info("=== Port Forwarding ===")
		cons.WriteLn("Enter supplementary container ports to expose (space-separated, or Enter to skip):")
		cons.WriteLn("Examples: 3000 5432 8080")

		input, err := cons.AskString("Ports", "")
		if err != nil {
			return nil, fmt.Errorf("unable to prompt for ports: %w", err)
		}

		ports := strings.Fields(input)
		validPorts, invalidPorts := filterValidPorts(ports)
		if len(invalidPorts) > 0 {
			cons.Warn("Invalid port numbers: \"%s\"", strings.Join(invalidPorts, " "))
			cons.Warn("Please input a valid list of space-separated ports (1-65535).")
			cons.WriteLn("")
			continue
		}
		return validPorts, nil
	}
}

func promptVolumes(cons *console.Console) ([]string, error) {
	cons.Info("=== Credentials & Volumes ===")
	cons.WriteLn("Mount common credentials/configs? (space-separated numbers, or Enter to skip)")
	cons.WriteLn("  1) SSH keys (~/.ssh)")
	cons.WriteLn("  2) Git credentials (~/.git-credentials)")
	cons.WriteLn("  3) AWS credentials (~/.aws)")
	cons.WriteLn("  4) Custom CA certificates (/etc/ssl/certs/custom-ca.crt)")

	choices, err := cons.AskString("Choice", "none")
	if err != nil {
		return nil, fmt.Errorf("unable to prompt for volume choice: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("unable to obtain home dir: %w", err)
	}

	var volumes []string
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
			// Skip
		default:
			cons.Warn("Ignoring unrecognized choice: \"%s\"", choice)
		}
	}

	// Custom volumes
	cons.WriteLn("")
	cons.WriteLn("Add custom volumes? (one per line, Enter on empty line to finish)")
	cons.WriteLn("Format: /host/path:/container/path[:ro]")
	for {
		vol, err := cons.AskString("Volume", "")
		if err != nil {
			return nil, fmt.Errorf("failed to ask for volume input: %w", err)
		}
		if vol == "" {
			break
		}
		volumes = append(volumes, vol)
	}

	return volumes, nil
}

func ensureProjectPath(path string, noPrompt bool, console *console.Console) error {
	if _, err := os.Stat(path); os.IsNotExist(err) && !noPrompt {
		console.Warn("Warning: Path %s does not exist", path)
		confirm, err := console.AskYesNo("Create config anyway?", false)
		if err != nil {
			return fmt.Errorf("asking user confirmation failed: %w", err)
		}
		if !confirm {
			return errors.New("project creation aborted by user")
		}
	}
	return nil
}
