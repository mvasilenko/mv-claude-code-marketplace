package installer

import (
	"fmt"
	"strconv"
	"strings"
)

// Version represents a semantic version
type Version struct {
	Major int
	Minor int
	Patch int
}

// ParseVersion parses a version string (with or without 'v' prefix)
// Supports semantic versioning with pre-release and build metadata:
// - 1.2.3
// - v1.2.3
// - 1.2.3-dirty
// - 1.2.3-alpha.1
// - 1.2.3+build.123
func ParseVersion(v string) (Version, error) {
	v = strings.TrimPrefix(v, "v")

	// Strip pre-release and build metadata (everything after - or +)
	if idx := strings.IndexAny(v, "-+"); idx != -1 {
		v = v[:idx]
	}

	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid version format: %s (expected MAJOR.MINOR.PATCH)", v)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	return Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, nil
}

// Compare compares this version with another version
// Returns: -1 if v < other, 0 if v == other, 1 if v > other
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}

	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}

	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}

	return 0
}

// IsNewer returns true if this version is newer than the other version
func (v Version) IsNewer(other Version) bool {
	return v.Compare(other) > 0
}

// String returns the string representation of the version
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}
