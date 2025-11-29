package commands

import (
	"bufio"
	"fmt"
	"os"
	"regexp"

	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
)

func List(filestore *files.FileStore, console *console.Console) error {
	// TODO: All that in filestore?
	if _, err := os.Stat(filestore.GetBaseComposeFilePath()); os.IsNotExist(err) {
		return fmt.Errorf("base compose.yaml not found at %s", filestore.GetBaseComposeFilePath())
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
