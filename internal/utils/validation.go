package utils

import (
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

const (
	VersionNone   = "none"
	VersionLatest = "latest"
)

// Reserved filename on windows. Projects should not be called one of those
var reservedWin = map[string]struct{}{
	"CON": {}, "PRN": {}, "AUX": {}, "NUL": {},
	"COM1": {}, "COM2": {}, "COM3": {}, "COM4": {}, "COM5": {}, "COM6": {}, "COM7": {}, "COM8": {}, "COM9": {},
	"LPT1": {}, "LPT2": {}, "LPT3": {}, "LPT4": {}, "LPT5": {}, "LPT6": {}, "LPT7": {}, "LPT8": {}, "LPT9": {},
}

var (
	// Debian/Ubuntu package name regex:
	// ^[a-z0-9]          → must start with letter or digit
	// [a-z0-9+.-]{1,254}$ → remaining allowed chars
	pkgNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9+.-]{1,254}$`)
	// Docker image name component rules (reference component):
	// - Lowercase letters, digits, hyphens, underscores only
	// - Must start/end with alphanumeric
	// - Max 128 chars per component
	projectNameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9_-]*[a-z0-9])?$`)
	versionRegex     = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	usernameRegex    = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)
	gitEmailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

func ValidateProjectName(name string) error {
	if name == "" {
		return errors.New("project name cannot be empty")
	}

	// Length constraints
	if len(name) > 128 {
		return fmt.Errorf("project name '%s' is too long (max 128 characters)", name)
	}

	// Docker image name rules (most restrictive)
	// Must be lowercase for image names
	if !projectNameRegex.MatchString(name) {
		return fmt.Errorf("invalid project name '%s': must be lowercase, start and end with alphanumeric, and contain only lowercase letters, digits, hyphens, and underscores", name)
	}

	// Filesystem restrictions - no path separators
	if strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("invalid project name '%s': cannot contain path separators", name)
	}

	// No special filesystem names
	if name == "." || name == ".." {
		return fmt.Errorf("invalid project name '%s': reserved filesystem name", name)
	}

	// Windows reserved names
	if runtime.GOOS == "windows" {
		// Check base name (without extension)
		baseName := name
		if idx := strings.IndexByte(name, '.'); idx != -1 {
			baseName = name[:idx]
		}
		if _, reserved := reservedWin[strings.ToUpper(baseName)]; reserved {
			return fmt.Errorf("invalid project name '%s': reserved by Windows", name)
		}

		// Windows forbidden characters
		if strings.ContainsAny(name, `<>:"|?*`) {
			return fmt.Errorf("invalid project name '%s': contains Windows forbidden characters", name)
		}

		// No trailing dots or spaces on Windows
		if strings.HasSuffix(name, ".") || strings.HasSuffix(name, " ") {
			return fmt.Errorf("invalid project name '%s': cannot end with dot or space on Windows", name)
		}
	}

	// Docker Compose project name restrictions
	// Project names become container names as: <project>-<service>-<replica>
	// Container names have additional restrictions
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return fmt.Errorf("invalid project name '%s': cannot start or end with hyphen", name)
	}

	// Consecutive separators can cause issues
	if strings.Contains(name, "--") || strings.Contains(name, "__") {
		return fmt.Errorf("invalid project name '%s': cannot contain consecutive separators", name)
	}

	// Shell variable safety (for .env usage)
	// While PROJECT_ID is quoted, avoid potential issues
	if strings.ContainsAny(name, "$`\"'\\!") {
		return fmt.Errorf("invalid project name '%s': contains shell metacharacters", name)
	}

	return nil
}

// SanitizeProjectName attempts to convert a directory name to a valid project name
func SanitizeProjectName(dirName string) (string, error) {
	if dirName == "" {
		return "", errors.New("directory name cannot be empty")
	}

	// Convert to lowercase
	name := strings.ToLower(dirName)

	// Replace invalid characters with hyphens
	name = regexp.MustCompile(`[^a-z0-9_-]+`).ReplaceAllString(name, "-")

	// Remove leading/trailing separators
	name = strings.Trim(name, "-_")

	// Collapse consecutive separators
	name = regexp.MustCompile(`[-_]{2,}`).ReplaceAllString(name, "-")

	// Truncate if too long
	if len(name) > 128 {
		name = name[:128]
		name = strings.TrimRight(name, "-_")
	}

	// Validate the result
	if err := ValidateProjectName(name); err != nil {
		return "", fmt.Errorf("unable to sanitize '%s': %w", dirName, err)
	}

	return name, nil
}

func ValidateVersionArg(version string) error {
	if version == "" || version == VersionLatest || version == VersionNone {
		return nil
	}
	if !versionRegex.MatchString(version) {
		return fmt.Errorf("invalid version argument: '%s'. Must be either \"none\", \"latest\" or semantic versioning (e.g., 20.10.0)", version)
	}
	return nil
}

func ValidateUIDGID(id string) error {
	i, err := strconv.Atoi(id)
	if err != nil || i < 0 || i > 65535 {
		return fmt.Errorf("must be a number between 0 and 65535")
	}
	return nil
}

func ValidateUsername(username string) error {
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("invalid username '%s'. Must start with lowercase letter or underscore, followed by lowercase letters, digits, underscores, or hyphens", username)
	}
	if len(username) > 32 {
		return errors.New("username too long (max 32 characters)")
	}
	return nil
}

func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port '%d'. Must be a number between 1 and 65535", port)
	}
	return nil
}

func ValidateGitName(name string) error {
	if strings.ContainsAny(name, "\"\n\r") {
		return errors.New("invalid git name. Cannot contain quotes or newlines")
	}
	if len(name) > 100 {
		return errors.New("git name too long (max 100 characters)")
	}
	return nil
}

func ValidateGitEmail(email string) error {
	if !gitEmailRegex.MatchString(email) {
		return fmt.Errorf("invalid git email format '%s'", email)
	}
	return nil
}

func EscapeEnvValue(str string) string {
	// Remove actual newlines/carriage returns
	str = strings.ReplaceAll(str, "\n", "")
	str = strings.ReplaceAll(str, "\r", "")

	// Escape backslashes first (before other escapes)
	str = strings.ReplaceAll(str, `\`, `\\`)

	// Escape double quotes
	str = strings.ReplaceAll(str, `"`, `\"`)

	// Escape dollar signs to prevent variable expansion
	str = strings.ReplaceAll(str, `$`, `\$`)

	return str
}

// IsValidUbuntuPackageName returns true if the name complies with Ubuntu/Debian package rules.
func IsValidUbuntuPackageName(name string) bool {
	return pkgNameRe.MatchString(name)
}
