package main

import "testing"

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
