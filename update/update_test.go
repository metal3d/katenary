package update

import (
	"fmt"
	"katenary/cmd"
	"os"
	"testing"
)

func TestDownloadLatestRelease(t *testing.T) {

	// Change the cmd.Version to "v0.0.0" to test the fallback to the latest release
	cmd.Version = "v0.0.0"

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
