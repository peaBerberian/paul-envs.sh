package utils

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	VersionNone   = "none"
	VersionLatest = "latest"
)

var (
	// Debian/Ubuntu package name regex:
	// ^[a-z0-9]          → must start with letter or digit
	// [a-z0-9+.-]{1,254}$ → remaining allowed chars
	pkgNameRe        = regexp.MustCompile(`^[a-z0-9][a-z0-9+.-]{1,254}$`)
	projectNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_-]{0,127}$`)
	versionRegex     = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	usernameRegex    = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)
	gitEmailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

func ValidateProjectName(name string) error {
	if name == "" {
		return errors.New("project name cannot be empty")
	}
	if !projectNameRegex.MatchString(name) {
		return fmt.Errorf("invalid project name '%s'. Must be 1-128 characters, start with alphanumeric or underscore, and contain only alphanumeric, hyphens, and underscores", name)
	}
	return nil
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
