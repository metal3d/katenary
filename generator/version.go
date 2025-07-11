package generator

import (
	"regexp"
	"runtime/debug"
)

// Version is the version of katenary. It is set at compile time.
var Version = "master" // changed at compile time

// GetVersion return the version of katneary. It's important to understand that
// the version is set at compile time for the github release. But, it the user get
// katneary using `go install`, the version should be different.
func GetVersion() string {
	// try to get the semantic version from the Version variable (theorically set at compile time)
	if reg := regexp.MustCompile(`^v?\d+.\d+.\d+.*|^release-.*`); reg.MatchString(Version) {
		return Version
	}

	// OK... let's try to get the version from the build info
	// get the version from the build info (when installed with go install)
	if v, ok := debug.ReadBuildInfo(); ok {
		return v.Main.Version + "-" + v.GoVersion
	}

	// OK... none worked, so we return the default version
	return Version
}
