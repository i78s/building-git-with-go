package command

import (
	"building-git/lib/repository"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func SetupTestEnvironment(t *testing.T) (string, *bytes.Buffer, *bytes.Buffer) {
	tmpDir, err := ioutil.TempDir("", "jit")
	if err != nil {
		t.Fatal(err)
	}
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)

	setupRepo(t, tmpDir)

	return tmpDir, outStream, errStream
}

func TestCommit(t *testing.T, dir string, message string) {
	t.Helper()

	os.Setenv("GIT_AUTHOR_NAME", "A. U. Thor")
	os.Setenv("GIT_AUTHOR_EMAIL", "author@example.com")
	defer os.Unsetenv("GIT_AUTHOR_NAME")
	defer os.Unsetenv("GIT_AUTHOR_EMAIL")

	stdin := strings.NewReader(message)
	Commit(dir, []string{}, stdin, new(bytes.Buffer), new(bytes.Buffer))
}

func setupRepo(t *testing.T, path string) {
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

func Repo(t *testing.T, path string) *repository.Repository {
	t.Helper()
	rootPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	return repository.NewRepository(rootPath)
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

func Mkdir(t *testing.T, path, name string) {
	t.Helper()

	dir := filepath.Join(path, name)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %s", err)
	}
}

func Touch(t *testing.T, path, name string) {
	t.Helper()

	filePath := filepath.Join(path, name)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		_, err := os.Create(filePath)
		if err != nil {
			t.Fatalf("Failed to create file: %s", err)
		}
	} else if err != nil {
		t.Fatalf("Failed to touch file: %s", err)
	}

	currentTime := time.Now()
	err := os.Chtimes(filePath, currentTime, currentTime)
	if err != nil {
		t.Fatalf("Failed to touch file: %s", err)
	}
}

func Delete(t *testing.T, path, name string) {
	t.Helper()

	filePath := filepath.Join(path, name)
	err := os.RemoveAll(filePath)
	if err != nil {
		t.Fatalf("Failed to delete file or directory: %s", err)
	}
}
