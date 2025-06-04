package utils

import "testing"

func TestHash(t *testing.T) {
	h, err := HashComposefiles([]string{"./hash.go"})
	if err != nil {
		t.Fatalf("failed to hash compose files: %v", err)
	}
	if len(h) == 0 {
		t.Fatal("hash should not be empty")
	}
}
