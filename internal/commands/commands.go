package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	constants "github.com/peaberberian/paul-envs/internal"
	"github.com/peaberberian/paul-envs/internal/args"
	"github.com/peaberberian/paul-envs/internal/config"
	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
	"github.com/peaberberian/paul-envs/internal/utils"
)

func Create(argsList []string, filestore *files.FileStore, console *console.Console) error {
	cfg, noPrompt, err := args.ParseAndPrompt(argsList, console, filestore)
	if err != nil {
		return err
	}

	if err := ensureProjectPath(cfg.ProjectHostPath, noPrompt, console); err != nil {
		return err
	}

	if err := generateProjectFiles(&cfg, filestore, console); err != nil {
		return err
	}

	printNextSteps(&cfg, filestore, console)
	return nil
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

func printNextSteps(cfg *config.Config, filestore *files.FileStore, console *console.Console) {
	console.Success("Created project '%s'", cfg.ProjectName)
	console.WriteLn("")
	console.WriteLn("Next steps:")
	console.WriteLn("  1. Review/edit configuration:")
	console.WriteLn("     - %s", filestore.GetEnvFilePathFor(cfg.ProjectName))
	console.WriteLn("     - %s", filestore.GetComposeFilePathFor(cfg.ProjectName))
	// TODO:
	// console.WriteLn("  2. Put the $HOME dotfiles you want to port in:")
	// console.WriteLn("     - %s/configs/", app.binaryDir)
	console.WriteLn("  2. Build the environment:")
	console.WriteLn("     paul-envs build %s", cfg.ProjectName)
	console.WriteLn("  3. Run the environment:")
	console.WriteLn("     paul-envs run %s", cfg.ProjectName)
}

func List(filestore *files.FileStore, console *console.Console) error {
	// TODO: All that in filestore?
	if _, err := os.Stat(filestore.GetBaseComposeFile()); os.IsNotExist(err) {
		return fmt.Errorf("base compose.yaml not found at %s", filestore.GetBaseComposeFile())
	}

	dirBase := filestore.GetProjectDirBase()
	if _, err := os.Stat(dirBase); os.IsNotExist(err) {
		console.WriteLn("No project created yet")
		console.WriteLn("Hint: Create one with 'paul-envs create <path>'")
		return nil
	}

	entries, err := os.ReadDir(dirBase)
	if err != nil {
		return fmt.Errorf("reading project directory failed: %w", err)
	}

	console.WriteLn("Projects created:")
	found := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if printProjectInfo(entry.Name(), filestore, console) {
			found = true
		}
	}

	if !found {
		console.WriteLn("  (no project found)")
		console.WriteLn("Hint: Create one with 'paul-envs create <path>'")
	}
	return nil
}

func printProjectInfo(name string, filestore *files.FileStore, console *console.Console) bool {
	composeFile := filestore.GetComposeFilePathFor(name)
	if _, err := os.Stat(composeFile); err != nil {
		return false
	}

	envFile := filestore.GetEnvFilePathFor(name)
	path := parseProjectPath(envFile)

	console.WriteLn("  - %s", name)
	console.WriteLn("      Path: %s", path)
	return true
}

func parseProjectPath(envFile string) string {
	f, err := os.Open(envFile)
	if err != nil {
		return ""
	}
	defer f.Close()

	re := regexp.MustCompile(`PROJECT_PATH="([^"]*)"`)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if matches := re.FindStringSubmatch(scanner.Text()); len(matches) == 2 {
			return matches[1]
		}
	}
	return ""
}

func Build(ctx context.Context, args []string, filestore *files.FileStore, console *console.Console) error {
	if err := ensureBaseComposeExists(filestore); err != nil {
		return err
	}

	name, err := getProjectName(args, filestore, console, "build")
	if err != nil {
		return err
	}

	if err := validateProjectFiles(filestore, name); err != nil {
		return err
	}

	if err := createSharedCacheVolume(ctx); err != nil {
		return err
	}

	if err := dockerComposeBuild(ctx, filestore, name); err != nil {
		return err
	}

	console.Success("Built project '%s'", name)
	return resetVolumes(ctx, filestore, name, console)
}

func ensureBaseComposeExists(filestore *files.FileStore) error {
	if _, err := os.Stat(filestore.GetBaseComposeFile()); os.IsNotExist(err) {
		return fmt.Errorf("base compose.yaml not found at %s", filestore.GetBaseComposeFile())
	}
	return nil
}

func getProjectName(args []string, filestore *files.FileStore, console *console.Console, action string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}

	if err := List(filestore, console); err != nil {
		return "", fmt.Errorf("failed to list projects: %w", err)
	}

	name, err := console.AskString(fmt.Sprintf("Enter project name to %s", action), "")
	if err != nil {
		return "", err
	}
	if name == "" {
		return "", errors.New("no project name provided")
	}
	return name, nil
}

func Run(ctx context.Context, args []string, filestore *files.FileStore, console *console.Console) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	baseCompose := filestore.GetBaseComposeFile()
	if _, err := os.Stat(baseCompose); os.IsNotExist(err) {
		return fmt.Errorf("Base compose.yaml not found at %s", baseCompose)
	}

	var name string
	var cmdArgs []string

	if len(args) == 0 {
		var err error
		err = List(filestore, console)
		if err != nil {
			return fmt.Errorf("no project name given, and failed to list other projects: %w", err)
		}
		console.WriteLn("No project name given, listing projects...")
		console.WriteLn("")
		name, err = console.AskString("Enter project name to run", "")
		if err != nil {
			return err
		}
		if name == "" {
			return fmt.Errorf("No project name provided")
		}
	} else {
		name = args[0]
		if len(args) > 1 {
			cmdArgs = args[1:]
		}
	}

	if err := utils.ValidateProjectName(name); err != nil {
		return err
	}

	composeFile := filestore.GetComposeFilePathFor(name)
	envFile := filestore.GetEnvFilePathFor(name)
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("Project '%s' not found\nHint: Use 'paul-envs list' to see available projects", name)
	}
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		return fmt.Errorf("Project '%s' not found\nHint: Use 'paul-envs list' to see available projects", name)
	}

	args = []string{"compose", "-f", baseCompose, "-f", composeFile, "--env-file", envFile, "run", "--rm", "paulenv"}
	args = append(args, cmdArgs...)

	if err := ctx.Err(); err != nil {
		return err
	}

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

func Remove(args []string, filestore *files.FileStore, console *console.Console) error {
	var name string
	if len(args) == 0 {
		var err error
		err = List(filestore, console)
		if err != nil {
			return fmt.Errorf("no project name given, and failed to list other projects: %w", err)
		}
		console.WriteLn("No project name given, listing projects...")
		console.WriteLn("")
		name, err = console.AskString("Enter project name to remove", "")
		if err != nil {
			return err
		}
		if name == "" {
			return fmt.Errorf("No project name provided")
		}
	} else {
		name = args[0]
	}

	if err := utils.ValidateProjectName(name); err != nil {
		return err
	}

	projectDir := filestore.GetProjectDir(name)
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return fmt.Errorf("Project '%s' not found\nHint: Use 'paul-envs list' to see available projects", name)
	}

	choice, err := console.AskYesNo(fmt.Sprintf("Remove project '%s'?", name), false)
	if err != nil {
		return err
	}
	if !choice {
		return nil
	}

	if err := os.RemoveAll(projectDir); err != nil {
		return fmt.Errorf("Failed to remove project: %w", err)
	}
	console.Success("Removed project '%s'", name)
	console.WriteLn("Note: Docker volumes are preserved. To remove them, run:")
	console.WriteLn("  docker volume rm paulenv-%s-local", name)
	return nil
}

func validateProjectFiles(filestore *files.FileStore, name string) error {
	composeFile := filestore.GetComposeFilePathFor(name)
	envFile := filestore.GetEnvFilePathFor(name)
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("project '%s' not found; use 'paul-envs list' to see available projects", name)
	}
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		return fmt.Errorf("project '%s' not found; use 'paul-envs list' to see available projects", name)
	}
	return utils.ValidateProjectName(name)
}

func createSharedCacheVolume(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "volume", "create", "paulenv-shared-cache")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to create shared volume: %w.", err)
	}
	return nil
}

func dockerComposeBuild(ctx context.Context, filestore *files.FileStore, name string) error {
	base := filestore.GetBaseComposeFile()
	compose := filestore.GetComposeFilePathFor(name)
	env := filestore.GetEnvFilePathFor(name)

	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", base, "-f", compose, "--env-file", env, "build")
	cmd.Env = append(os.Environ(), "COMPOSE_PROJECT_NAME=paulenv-"+name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func resetVolumes(ctx context.Context, filestore *files.FileStore, name string, console *console.Console) error {
	console.WriteLn("\nResetting persistent volumes...")
	base := filestore.GetBaseComposeFile()
	compose := filestore.GetComposeFilePathFor(name)
	env := filestore.GetEnvFilePathFor(name)

	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", base, "-f", compose, "--env-file", env,
		"--profile", "reset", "up", "reset-cache", "reset-local")
	cmd.Env = append(os.Environ(), "COMPOSE_PROJECT_NAME=paulenv-"+name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reset volumes: %w", err)
	}
	console.Success("Volumes reset complete")
	return nil
}

func Version(ctx context.Context, console *console.Console) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	console.WriteLn("paul-envs version %s", constants.Version)
	console.WriteLn("Go version: %s", runtime.Version())
	cmd := exec.CommandContext(ctx, "docker", "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("Failed to obtain docker version: %w", err)
	}
	console.WriteLn("Docker version: %s", output)
	return nil
}

func Help(filestore *files.FileStore, console *console.Console) {
	console.WriteLn(`paul-envs - Development Environment Manager

Usage:
  paul-envs create <path> [options]
  paul-envs list
  paul-envs build <name>
  paul-envs run <name> [commands]
  paul-envs remove <name>
  paul-envs version
  paul-envs help
  paul-envs interactive
  paul-envs clean

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
  --package PKG_NAME       Additional Ubuntu package (prompted if not specified, can be repeated)
  --port PORT              Expose container port (prompted if not specified, can be repeated)
  --volume HOST:CONT[:ro]  Mount volume (prompted if not specified, can be repeated)

Windows/Git Bash Notes:
  - UID/GID default to 1000 on Windows (Docker Desktop requirement)

Creating a configuration in interactive Mode (default):
  paul-envs create ~/projects/myapp
  # Will prompt for all unspecified options

Creating a configuration in a non-Interactive Mode:
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
    --package ripgrep \
    --package ripgrep \
    --port 3000 \
    --port 5432 \
    --volume ~/.git-credentials:/home/dev/.git-credentials:ro

Location of stored files created by this tool:
  Base compose       : ` + filestore.GetBaseComposeFile() + `
  Projects directory : ` + filestore.GetProjectDirBase() + `

NOTE: To start a guided prompt, you can also just run:
  paul-envs interactive
`)
}

func generateProjectFiles(cfg *config.Config, filestore *files.FileStore, console *console.Console) error {
	if filestore.CheckProjectNameAvailable(cfg.ProjectName, console) != nil {
		return errors.New("project name already taken")
	}

	envData := files.EnvTemplateData{
		ProjectComposeFilename: utils.EscapeEnvValue(files.ProjectComposeFilename),
		ProjectID:              utils.EscapeEnvValue(cfg.ProjectName),
		ProjectDestPath:        utils.EscapeEnvValue(cfg.ProjectDestPath),
		ProjectHostPath:        utils.EscapeEnvValue(cfg.ProjectHostPath),
		HostUID:                utils.EscapeEnvValue(cfg.UID),
		HostGID:                utils.EscapeEnvValue(cfg.GID),
		Username:               utils.EscapeEnvValue(cfg.Username),
		Shell:                  string(cfg.Shell),
		InstallNode:            utils.EscapeEnvValue(cfg.InstallNode),
		InstallRust:            utils.EscapeEnvValue(cfg.InstallRust),
		InstallPython:          utils.EscapeEnvValue(cfg.InstallPython),
		InstallGo:              utils.EscapeEnvValue(cfg.InstallGo),
		EnableWasm:             strconv.FormatBool(cfg.EnableWasm),
		EnableSSH:              strconv.FormatBool(cfg.EnableSsh),
		EnableSudo:             strconv.FormatBool(cfg.EnableSudo),
		Packages:               utils.EscapeEnvValue(strings.Join(cfg.Packages, " ")),
		InstallNeovim:          strconv.FormatBool(cfg.InstallNeovim),
		InstallStarship:        strconv.FormatBool(cfg.InstallStarship),
		InstallAtuin:           strconv.FormatBool(cfg.InstallAtuin),
		InstallMise:            strconv.FormatBool(cfg.InstallMise),
		InstallZellij:          strconv.FormatBool(cfg.InstallZellij),
		InstallJujutsu:         strconv.FormatBool(cfg.InstallJujutsu),
		GitName:                utils.EscapeEnvValue(cfg.GitName),
		GitEmail:               utils.EscapeEnvValue(cfg.GitEmail),
	}

	if err := filestore.CreateProjectEnvFile(cfg.ProjectName, envData); err != nil {
		return fmt.Errorf("failed to create project env file: %w", err)
	}

	composeData := files.ComposeTemplateData{
		ProjectName: cfg.ProjectName,
		Ports:       cfg.Ports,
		EnableSSH:   cfg.EnableSsh,
		SSHKeyPath:  cfg.SshKeyPath,
		Volumes:     cfg.Volumes,
	}

	if err := filestore.CreateProjectComposeFile(cfg.ProjectName, composeData); err != nil {
		return fmt.Errorf("failed to create project compose file: %w", err)
	}

	return nil
}

func Interactive(ctx context.Context, fs *files.FileStore, c *console.Console) error {
	for {
		c.WriteLn("")
		c.Info("Available commands:")
		c.WriteLn("  1. create  - Create a new configuration for a project directory")
		c.WriteLn("  2. list    - List all created configurations")
		c.WriteLn("  3. build   - Build an image according to your configuration")
		c.WriteLn("  4. run     - Run a container based on a built image")
		c.WriteLn("  5. remove  - Remove a configuration and its data")
		c.WriteLn("  6. version - Show the current version")
		c.WriteLn("  7. clean   - Remove all stored paul-envs data from your computer")
		c.WriteLn("  8. exit    - Exit interactive mode")
		c.WriteLn("")

		choice, err := c.AskString("Select a command (1-7 or name)", "")
		if err != nil {
			return err
		}

		choice = strings.ToLower(strings.TrimSpace(choice))

		var cmdErr error
		switch choice {
		case "1", "create":
			path, err := c.AskString("Project path", "")
			if err != nil {
				return err
			}
			cmdErr = Create([]string{path}, fs, c)
		case "2", "list", "ls":
			cmdErr = List(fs, c)
		case "3", "build":
			cmdErr = Build(ctx, []string{}, fs, c)
		case "4", "run":
			cmdErr = Run(ctx, []string{}, fs, c)
		case "5", "remove", "rm":
			cmdErr = Remove([]string{}, fs, c)
		case "6", "version":
			cmdErr = Version(ctx, c)
		case "7", "clean":
			cmdErr = Clean(ctx, fs, c)
		case "8", "exit", "quit", "q":
			c.Success("Goodbye!")
			return nil
		default:
			c.Error("Invalid choice: %s", choice)
			continue
		}

		if cmdErr != nil {
			c.Error("Command failed: %v", cmdErr)
			continuePrompt, err := c.AskYesNo("Continue?", true)
			if err != nil || !continuePrompt {
				return err
			}
		}
	}
}
