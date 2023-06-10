package command

import (
	"building-git/lib/command/commandtest"
	"os"
	"path/filepath"
	"testing"
)

func TestInit(t *testing.T) {
	tmpDir, stdout, stderr := commandtest.SetupTestEnvironment(t)
	defer os.RemoveAll(tmpDir)

	Init([]string{tmpDir}, stdout, stderr)

	gitDir := filepath.Join(tmpDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		t.Errorf("git directory not created: %s", gitDir)
	}

	subDirs := []string{"objects", "refs"}
	for _, dir := range subDirs {
		path := filepath.Join(gitDir, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("subdirectory not created: %s", path)
		}
	}
}
