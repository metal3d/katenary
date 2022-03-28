package update

import (
	"fmt"
	"os"
	"testing"
)

func TestDownloadLatestRelease(t *testing.T) {

	// Reset the version to test the latest release
	Version = "0.0.0"

	// change "exe" to /tmp/test-katenary
	exe = "/tmp/test-katenary"
	defer os.Remove(exe)

	// Now call the CheckLatestVersion function
	version, assets, err := CheckLatestVersion()

	if err != nil {
		t.Errorf("Error: %s", err)
	}

	fmt.Println("Version found", version)

	// Touch exe binary
	f, _ := os.OpenFile(exe, os.O_RDONLY|os.O_CREATE, 0755)
	f.Write(nil)
	f.Close()

	err = DownloadLatestVersion(assets)
	if err != nil {
		t.Errorf("Error: %s", err)
	}
}
