package commands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/peaberberian/paul-envs/internal/args"
	"github.com/peaberberian/paul-envs/internal/config"
	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
	"github.com/peaberberian/paul-envs/internal/utils"
)

func Create(argsList []string, filestore *files.FileStore, console *console.Console) error {
	cfg, err := args.ParseAndPrompt(argsList, console, filestore)
	if err != nil {
		return err
	}

	if err := generateProjectFiles(&cfg, filestore); err != nil {
		return err
	}
	dotfilesDir, err := filestore.InitGlobalDotfilesDir()
	if err != nil {
		return err
	}

	printNextSteps(&cfg, dotfilesDir, filestore, console)
	return nil
}

func generateProjectFiles(cfg *config.Config, filestore *files.FileStore) error {
	if filestore.DoesProjectExist(cfg.ProjectName) {
		return errors.New("project name already taken")
	}

	// TODO: Should those template definitions be moved to the `FileStore` code?
	// It could only take the Config as argument
	envData := files.EnvTemplateData{
		ProjectID:       utils.EscapeEnvValue(cfg.ProjectName),
		ProjectDestPath: utils.EscapeEnvValue(cfg.ProjectDestPath),
		ProjectHostPath: utils.EscapeEnvValue(cfg.ProjectHostPath),
		HostUID:         utils.EscapeEnvValue(cfg.UID),
		HostGID:         utils.EscapeEnvValue(cfg.GID),
		Username:        utils.EscapeEnvValue(cfg.Username),
		Shell:           string(cfg.Shell),
		InstallNode:     utils.EscapeEnvValue(cfg.InstallNode),
		InstallRust:     utils.EscapeEnvValue(cfg.InstallRust),
		InstallPython:   utils.EscapeEnvValue(cfg.InstallPython),
		InstallGo:       utils.EscapeEnvValue(cfg.InstallGo),
		EnableWasm:      strconv.FormatBool(cfg.EnableWasm),
		EnableSSH:       strconv.FormatBool(cfg.EnableSsh),
		EnableSudo:      strconv.FormatBool(cfg.EnableSudo),
		Packages:        utils.EscapeEnvValue(strings.Join(cfg.Packages, " ")),
		InstallNeovim:   strconv.FormatBool(cfg.InstallNeovim),
		InstallStarship: strconv.FormatBool(cfg.InstallStarship),
		InstallAtuin:    strconv.FormatBool(cfg.InstallAtuin),
		InstallMise:     strconv.FormatBool(cfg.InstallMise),
		InstallZellij:   strconv.FormatBool(cfg.InstallZellij),
		InstallJujutsu:  strconv.FormatBool(cfg.InstallJujutsu),
		GitName:         utils.EscapeEnvValue(cfg.GitName),
		GitEmail:        utils.EscapeEnvValue(cfg.GitEmail),
	}

	composeData := files.ComposeTemplateData{
		ProjectName: cfg.ProjectName,
		Ports:       cfg.Ports,
		EnableSSH:   cfg.EnableSsh,
		SSHKeyPath:  cfg.SshKeyPath,
		Volumes:     cfg.Volumes,
	}

	err := filestore.CreateProjectFiles(cfg.ProjectName, envData, composeData)
	if err != nil {
		return fmt.Errorf("failed to create project files: %w", err)
	}
	return nil
}

func printNextSteps(cfg *config.Config, dotfilesDir string, filestore *files.FileStore, console *console.Console) {
	console.Success("Created project '%s'", cfg.ProjectName)
	console.WriteLn("")
	console.WriteLn("Next steps:")
	console.WriteLn("  1. Review/edit configuration:")
	// TODO: rely on just `GetProject` instead
	console.WriteLn("     - %s", filestore.GetProjectEnvFilePath(cfg.ProjectName))
	console.WriteLn("     - %s", filestore.GetProjectComposeFilePath(cfg.ProjectName))
	console.WriteLn("  2. Put the $HOME dotfiles you want to port in:")
	console.WriteLn("     - %s", dotfilesDir)
	console.WriteLn("  3. Build the environment:")
	console.WriteLn("     paul-envs build %s", cfg.ProjectName)
	console.WriteLn("  4. Run the environment:")
	console.WriteLn("     paul-envs run %s", cfg.ProjectName)
}
