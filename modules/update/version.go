package update

import (
	"fmt"
	"strconv"
	"strings"
)

// Version represents a parsed semantic version
type Version struct {
	Raw   string
	IsDev bool
	major int
	minor int
	patch int
	pre   string
	build string
}

// ParseVersion parses a version string into a Version struct
func ParseVersion(v string) (*Version, error) {
	if v == "" {
		return nil, fmt.Errorf("empty version string")
	}

	version := &Version{
		Raw:   v,
		IsDev: v == "dev",
	}

	if version.IsDev {
		return version, nil
	}

	// Remove "v" prefix if present
	clean := strings.TrimPrefix(v, "v")

	// Split by + for build metadata
	parts := strings.SplitN(clean, "+", 2)
	core := parts[0]
	if len(parts) == 2 {
		version.build = parts[1]
	}

	// Split by - for prerelease
	parts = strings.SplitN(core, "-", 2)
	numbers := parts[0]
	if len(parts) == 2 {
		version.pre = parts[1]
	}

	// Parse major.minor.patch
	nums := strings.Split(numbers, ".")
	if len(nums) != 3 {
		return nil, fmt.Errorf("invalid semver format: %s", v)
	}

	var err error
	version.major, err = strconv.Atoi(nums[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %s", nums[0])
	}

	version.minor, err = strconv.Atoi(nums[1])
	if err != nil {
		return nil, fmt.Errorf("invalid minor version: %s", nums[1])
	}

	version.patch, err = strconv.Atoi(nums[2])
	if err != nil {
		return nil, fmt.Errorf("invalid patch version: %s", nums[2])
	}

	return version, nil
}

// String returns the version string without "v" prefix
func (v *Version) String() string {
	if v.IsDev {
		return "dev"
	}

	s := fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
	if v.pre != "" {
		s += "-" + v.pre
	}
	if v.build != "" {
		s += "+" + v.build
	}
	return s
}

// Compare compares two versions and returns:
// -1 if v < other
//  0 if v == other
//  1 if v > other
func (v *Version) Compare(other *Version) int {
	// dev is always older than any release
	if v.IsDev && other.IsDev {
		return 0
	}
	if v.IsDev {
		return -1
	}
	if other.IsDev {
		return 1
	}

	// Compare major
	if v.major != other.major {
		if v.major < other.major {
			return -1
		}
		return 1
	}

	// Compare minor
	if v.minor != other.minor {
		if v.minor < other.minor {
			return -1
		}
		return 1
	}

	// Compare patch
	if v.patch != other.patch {
		if v.patch < other.patch {
			return -1
		}
		return 1
	}

	// Compare prerelease (no prerelease is higher than prerelease)
	if v.pre != "" && other.pre == "" {
		return -1
	}
	if v.pre == "" && other.pre != "" {
		return 1
	}
	if v.pre != other.pre {
		if v.pre < other.pre {
			return -1
		}
		return 1
	}

	// Build metadata is ignored in comparison
	return 0
}

// IsOlderThan returns true if v is older than other
func (v *Version) IsOlderThan(other *Version) bool {
	return v.Compare(other) < 0
}

// GetCurrentVersion returns the current version from the provided version string
func GetCurrentVersion(versionStr string) *Version {
	v, err := ParseVersion(versionStr)
	if err != nil {
		// If parsing fails, treat as dev
		return &Version{
			Raw:   versionStr,
			IsDev: true,
		}
	}
	return v
}
