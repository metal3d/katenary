package utils

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os"
	"sort"
)

// HashComposefiles returns a hash of the compose files.
func HashComposefiles(files []string) (string, error) {
	sort.Strings(files) // ensure the order is always the same
	sha := sha1.New()
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			return "", err
		}
		defer f.Close()
		if _, err := io.Copy(sha, f); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(sha.Sum(nil)), nil
}
