package commands

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

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
