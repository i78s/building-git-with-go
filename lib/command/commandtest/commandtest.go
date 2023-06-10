package commandtest

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func SetupTestEnvironment(t *testing.T) (string, *bytes.Buffer, *bytes.Buffer) {
	tmpDir, err := ioutil.TempDir("", "jit")
	if err != nil {
		t.Fatal(err)
	}
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)

	return tmpDir, outStream, errStream
}

func SetupRepo(t *testing.T, path string) {
	t.Helper()

	rootPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	gitPath := filepath.Join(rootPath, ".git")
	dirs := []string{"objects", "refs"}

	for _, dir := range dirs {
		err := os.MkdirAll(filepath.Join(gitPath, dir), os.ModePerm)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fatal: %s\n", err.Error())
		}
	}
}

func WriteFile(t *testing.T, path, name, content string) {
	t.Helper()

	dir := filepath.Join(path, filepath.Dir(name))
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %s", err)
	}

	// Write the content to the file
	file, err := os.Create(filepath.Join(path, name))
	if err != nil {
		t.Fatalf("Failed to create file: %s", err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		t.Fatalf("Failed to write to file: %s", err)
	}
}

func MakeExecutable(t *testing.T, path, name string) {
	t.Helper()

	err := os.Chmod(filepath.Join(path, name), 0755)
	if err != nil {
		t.Fatalf("Failed to make file executable: %s", err)
	}
}

func MakeUnreadable(t *testing.T, path, name string) {
	t.Helper()

	err := os.Chmod(filepath.Join(path, name), 0200)
	if err != nil {
		t.Fatalf("Failed to make file unreadable: %s", err)
	}
}
