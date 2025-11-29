package utils

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Version struct {
	major int
	minor int
	patch int
}

func NewVersion(major, minor, patch int) Version {
	return Version{major: major, minor: minor, patch: patch}
}

// ParseVersion takes a version string in the form "x.y.z" and splits it into integers
func ParseVersion(version string) (Version, error) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return Version{major: 0, minor: 0, patch: 0}, errors.New("invalid version format, expected x.y.z")
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{major: 0, minor: 0, patch: 0}, fmt.Errorf("invalid major version: %v", err)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{major: 0, minor: 0, patch: 0}, fmt.Errorf("invalid minor version: %v", err)
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{major: 0, minor: 0, patch: 0}, fmt.Errorf("invalid patch version: %v", err)
	}

	return Version{major: major, minor: minor, patch: patch}, nil
}

// compareVersions compares two versions: v1 and v2, each in "x.y.z" format
// Returns:
//
//	-1 if v1 < v2
//	 0 if v1 == v2
//	 1 if v1 > v2
func compareVersions(v1, v2 *Version) int {
	if v1.major != v2.major {
		if v1.major < v2.major {
			return -1
		}
		return 1
	}
	if v1.minor != v2.minor {
		if v1.minor < v2.minor {
			return -1
		}
		return 1
	}
	if v1.patch != v2.patch {
		if v1.patch < v2.patch {
			return -1
		}
		return 1
	}

	// Versions are equal
	return 0
}

func (v *Version) IsCompatibleWithBase(base Version) bool {
	result := compareVersions(&base, v)
	return base.major == v.major && result >= 0
}

func (v *Version) ToString() string {
	return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
}
