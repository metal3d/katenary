package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"katenary/cmd"
	"net/http"
	"os"
	"runtime"
	"time"
)

var exe, _ = os.Executable()

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
		TagName string  `json:"tag_name"`
		Assets  []Asset `json:"assets"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&release)
	if err != nil {
		return "", nil, err
	}

	if release.TagName == "" {
		return "", nil, errors.New("No release found")
	}

	if cmd.Version == release.TagName {
		fmt.Println("You are using the latest version")
		return "", nil, errors.New("You are using the latest version")
	}

	return release.TagName, release.Assets, nil
}

func DownloadLatestVersion(assets []Asset) error {
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
