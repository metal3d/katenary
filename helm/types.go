package helm

import (
	"os"
	"strings"
)

const K = "katenary.io"

var (
	Appname = ""    // set at runtime
	Version = "1.0" // should be set from main.Version
)

// Kinded represent an object with a kind.
type Kinded interface {
	// Get must resturn the kind name.
	Get() string
}

// Signable represents an object with a signature.
type Signable interface {
	// BuildSHA must return the signature.
	BuildSHA(filename string)
}

// Named represents an object with a name.
type Named interface {
	// Name must return the name of the object (from metadata).
	Name() string
}

// GetProjectName returns the name of the project.
func GetProjectName() string {
	if len(Appname) > 0 {
		return Appname
	}
	p, _ := os.Getwd()
	path := strings.Split(p, string(os.PathSeparator))
	return path[len(path)-1]
}
