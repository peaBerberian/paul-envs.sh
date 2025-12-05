package utils

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Version struct {
	Major int
	Minor int
	Patch int
}

// ParseVersion takes a version string in the form "x.y.z" and splits it into integers
func ParseVersion(version string) (Version, error) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return Version{Major: 0, Minor: 0, Patch: 0}, errors.New("invalid version format, expected x.y.z")
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{Major: 0, Minor: 0, Patch: 0}, fmt.Errorf("invalid major version: %v", err)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{Major: 0, Minor: 0, Patch: 0}, fmt.Errorf("invalid minor version: %v", err)
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{Major: 0, Minor: 0, Patch: 0}, fmt.Errorf("invalid patch version: %v", err)
	}

	return Version{Major: major, Minor: minor, Patch: patch}, nil
}

// compareVersions compares two versions: v1 and v2, each in "x.y.z" format
// Returns:
//
//	-1 if v1 < v2
//	 0 if v1 == v2
//	 1 if v1 > v2
func compareVersions(v1, v2 *Version) int {
	if v1.Major != v2.Major {
		if v1.Major < v2.Major {
			return -1
		}
		return 1
	}
	if v1.Minor != v2.Minor {
		if v1.Minor < v2.Minor {
			return -1
		}
		return 1
	}
	if v1.Patch != v2.Patch {
		if v1.Patch < v2.Patch {
			return -1
		}
		return 1
	}

	// Versions are equal
	return 0
}

func (v *Version) IsCompatibleWithBase(base Version) bool {
	result := compareVersions(&base, v)
	return base.Major == v.Major && result >= 0
}

func (v *Version) ToString() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}
