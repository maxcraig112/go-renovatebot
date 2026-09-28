// Package versiontag converts between npm's unprefixed semver strings
// (e.g. "41.20.3") and the "v"-prefixed git tags this repository uses
// (e.g. "v41.20.3"), and provides ordering over both.
package versiontag

import (
	"sort"
	"strings"

	"golang.org/x/mod/semver"
)

// Tag returns the git tag for a given npm version string.
func Tag(npmVersion string) string {
	if strings.HasPrefix(npmVersion, "v") {
		return npmVersion
	}
	return "v" + npmVersion
}

// NpmVersion returns the npm version string for a given git tag.
func NpmVersion(tag string) string {
	return strings.TrimPrefix(tag, "v")
}

// Less reports whether npm version a sorts before npm version b.
func Less(a, b string) bool {
	return semver.Compare(Tag(a), Tag(b)) < 0
}

// IsValid reports whether the npm version string is valid semver.
func IsValid(npmVersion string) bool {
	return semver.IsValid(Tag(npmVersion))
}

// SortAscending sorts npm version strings in place, ascending.
func SortAscending(versions []string) {
	sort.Slice(versions, func(i, j int) bool {
		return Less(versions[i], versions[j])
	})
}
