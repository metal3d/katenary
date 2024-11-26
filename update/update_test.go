package update

// TODO: fix this tests, and possibly the update function

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
		t.Logf("Error getting latest version: %s", err)
	}

	fmt.Println("Version found", version)

	// Touch exe binary
	f, _ := os.OpenFile(exe, os.O_RDONLY|os.O_CREATE, 0o755)
	f.Write(nil)
	f.Close()

	err = DownloadLatestVersion(assets)
	if err != nil {
		t.Logf("Error: %s", err)
	}
}

func TestAlreadyUpToDate(t *testing.T) {
	Version = "99999.999.99"
	exe = "/tmp/test-katenary"
	defer os.Remove(exe)

	// Call the version check
	version, _, err := CheckLatestVersion()

	if err == nil {
		t.Logf("Error: %v", err)
	}

	t.Log("Version is already the most recent", version)
}
