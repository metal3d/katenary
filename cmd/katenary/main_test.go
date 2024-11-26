package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestBuildCommand(t *testing.T) {
	rootCmd := buildRootCmd()
	if rootCmd == nil {
		t.Errorf("Expected rootCmd to be defined")
	}
	if rootCmd.Use != "katenary" {
		t.Errorf("Expected rootCmd.Use to be katenary, got %s", rootCmd.Use)
	}
	numCommands := 6
	if len(rootCmd.Commands()) != numCommands {
		t.Errorf("Expected %d command, got %d", numCommands, len(rootCmd.Commands()))
	}
}

func TestGetVersion(t *testing.T) {
	cmd := buildRootCmd()
	if cmd == nil {
		t.Errorf("Expected cmd to be defined")
	}
	version := generateVersionCommand()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	version.Run(cmd, nil)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()
	if !strings.Contains(output, "(devel)") {
		t.Errorf("Expected output to contain '(devel)', got %s", output)
	}
}

func TestSchemaCommand(t *testing.T) {
	cmd := buildRootCmd()
	if cmd == nil {
		t.Errorf("Expected cmd to be defined")
	}
	schema := generateSchemaCommand()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	schema.Run(cmd, nil)
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// try to parse json
	schemaContent := make(map[string]interface{})
	if err := json.Unmarshal([]byte(output), &schemaContent); err != nil {
		t.Errorf("Expected valid json, got %s", output)
	}
}
