package tags

import (
	"github.com/Masterminds/semver/v3"
)

var ALLOWED_TAGS = []string{"latest", "pre", "beta", "alpha"}

// IsTag returns true if a version is a tag
func IsTag(version string) bool {
	for _, tag := range ALLOWED_TAGS {
		if version == tag {
			return true
		}
	}
	return false
}

// IsValid returns true if a version matches gitbook-cli's requirements
func IsValid(version string, gitbookVersionConstraint string) bool {
	if IsTag(version) {
		return true
	}

	v, err := semver.NewVersion(version)
	if err != nil {
		return false
	}

	// Remove prerelease part for constraint checking
	constraint, err := semver.NewConstraint(gitbookVersionConstraint)
	if err != nil {
		return false
	}

	return constraint.Check(v)
}

// GetTag extracts prerelease tag from a version
func GetTag(version string) string {
	if IsTag(version) {
		return version
	}

	v, err := semver.NewVersion(version)
	if err != nil {
		return "latest"
	}

	if v.Prerelease() != "" {
		// Extract the first part of prerelease (e.g., "beta.1" -> "beta")
		prerelease := v.Prerelease()
		for _, tag := range ALLOWED_TAGS {
			if len(prerelease) >= len(tag) && prerelease[:len(tag)] == tag {
				return tag
			}
		}
	}

	return "latest"
}

// Sort compares two versions (takes prerelease tags into consideration)
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func Sort(a, b string) int {
	aIsTag := IsTag(a)
	bIsTag := IsTag(b)

	if aIsTag && bIsTag {
		aIndex := getTagIndex(a)
		bIndex := getTagIndex(b)
		if aIndex > bIndex {
			return -1
		}
		if bIndex > aIndex {
			return 1
		}
		return 0
	}

	if aIsTag {
		return -1
	}
	if bIsTag {
		return 1
	}

	va, errA := semver.NewVersion(a)
	vb, errB := semver.NewVersion(b)

	if errA != nil || errB != nil {
		if errA != nil && errB != nil {
			return 0
		}
		if errA != nil {
			return -1
		}
		return 1
	}

	if va.LessThan(vb) {
		return 1
	}
	if vb.LessThan(va) {
		return -1
	}
	return 0
}

// Satisfies returns true if a version satisfies a condition
func Satisfies(version, condition string, acceptTagCondition bool) bool {
	if IsTag(version) {
		return condition == "*" || version == condition
	}

	// Condition is a tag ('beta', 'latest')
	if acceptTagCondition {
		tag := GetTag(version)
		if tag == condition {
			return true
		}
	}

	v, err := semver.NewVersion(version)
	if err != nil {
		return false
	}

	constraint, err := semver.NewConstraint(condition)
	if err != nil {
		return false
	}

	return constraint.Check(v)
}

func getTagIndex(tag string) int {
	for i, t := range ALLOWED_TAGS {
		if t == tag {
			return i
		}
	}
	return -1
}
