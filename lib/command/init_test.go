package command

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestInit(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "jit")
	if err != nil {
		t.Fatal(err)
	}
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
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
