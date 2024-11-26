package generator

import (
	"runtime/debug"
	"strings"
)

// Version is the version of katenary. It is set at compile time.
var Version = "master" // changed at compile time

// GetVersion return the version of katneary. It's important to understand that
// the version is set at compile time for the github release. But, it the user get
// katneary using `go install`, the version should be different.
func GetVersion() string {
	if strings.HasPrefix(Version, "release-") {
		return Version
	}
	// get the version from the build info
	v, ok := debug.ReadBuildInfo()
	if ok {
		return v.Main.Version + "-" + v.GoVersion
	}
	return Version
}
