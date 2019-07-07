package main

import (
	"io/ioutil"
	"os/exec"
	"testing"
)

func TestIntegration(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatal("failed to create a tempdir")
	}
	cmd := exec.Command("git", "clone", "https://github.com/kubernetes/enhancements")
	cmd.Dir = tempDir
	out, err := gitClone("https://github.com/kubernetes/enhancements", into string)
	// git clone
	// pass root
}
