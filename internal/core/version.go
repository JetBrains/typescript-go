// MODIFIED IN THIS FORK: 2025-05-21 Piotr Tomiak dfd337df3e64d6fb2bc18c46bf4de05bcbea5a64 Support WebStorm types in LSP mode
package core

import (
	"strings"
)

// This is a var so it can be overridden by ldflags.
var version = "7.0.0-dev"

func Version() string {
	return version + "-jb"
}

var versionMajorMinor = func() string {
	seenMajor := false
	i := strings.IndexFunc(version, func(r rune) bool {
		if r == '.' {
			if seenMajor {
				return true
			}
			seenMajor = true
		}
		return false
	})
	if i == -1 {
		panic("invalid version string: " + version)
	}
	return version[:i]
}()

func VersionMajorMinor() string {
	return versionMajorMinor
}
