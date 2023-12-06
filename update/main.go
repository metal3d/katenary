/* Update package is used to check if a new version of katenary is available.*/
package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"

	"golang.org/x/mod/semver"
)

var exe, _ = os.Executable()
var Version = "master" // reset by cmd/main.go

// Asset is a github asset from release url.
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

// CheckLatestVersion check katenary latest version from release and propose to download it
func CheckLatestVersion() (string, []Asset, error) {

	githuburl := "https://api.github.com/repos/metal3d/katenary/releases/latest"
	// Create a HTTP client with 1s timeout
	client := &http.Client{
		Timeout: time.Second * 1,
	}
	// Create a request
	req, err := http.NewRequest("GET", githuburl, nil)
	if err != nil {
		return "", nil, err
	}

	// Send the request via a client
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	// Get tag_name from the json response
	var release = struct {
		TagName    string  `json:"tag_name"`
		Assets     []Asset `json:"assets"`
		PreRelease bool    `json:"prerelease"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&release)
	if err != nil {
		return "", nil, err
	}

	// if it's a prerelease, don't update
	if release.PreRelease {
		return "", nil, errors.New("Prerelease detected, not updating")
	}

	// no tag, don't update
	if release.TagName == "" {
		return "", nil, errors.New("No release found")
	}

	// compare the current version, if the current version is the same or lower than the latest version, don't update
	versions := []string{Version, release.TagName}
	semver.Sort(versions)
	if versions[1] == Version {
		return "", nil, errors.New("Current version is the latest version")
	}

	return release.TagName, release.Assets, nil
}

// DownloadLatestVersion will download the latest version of katenary.
func DownloadLatestVersion(assets []Asset) error {

	defer func() {
		if r := recover(); r != nil {
			os.Rename(exe+".old", exe)
		}
	}()

	// Download the latest version
	fmt.Println("Downloading the latest version...")

	// ok, replace this from the current version to the latest version
	err := os.Rename(exe, exe+".old")
	if err != nil {
		return err
	}

	// Download the latest version for the current OS
	for _, asset := range assets {
		switch runtime.GOOS {
		case "windows":
			if asset.Name == "katenary.exe" {
				err = DownloadFile(asset.URL, exe)
			}
		case "linux":
			switch runtime.GOARCH {
			case "amd64":
				if asset.Name == "katenary-linux-amd64" {
					err = DownloadFile(asset.URL, exe)
				}
			case "arm64":
				if asset.Name == "katenary-linux-arm64" {
					err = DownloadFile(asset.URL, exe)
				}
			}
		case "darwin":
			if asset.Name == "katenary-darwin" {
				err = DownloadFile(asset.URL, exe)
			}
		default:
			fmt.Println("Unsupported OS")
			err = errors.New("Unsupported OS")
		}
	}
	if err == nil {
		// remove the old version
		os.Remove(exe + ".old")
	} else {
		// restore the old version
		os.Rename(exe+".old", exe)
	}
	return err
}

// DownloadFile will download a url to a local file. It also ensure that the file is executable.
func DownloadFile(url, exe string) error {
	// Download the url binary to exe path
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	fp, err := os.OpenFile(exe, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer fp.Close()
	_, err = io.Copy(fp, resp.Body)
	return err
}
