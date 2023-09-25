package command

import (
	"building-git/lib/database"
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

func setupTestEnvironment(t *testing.T) (string, *bytes.Buffer, *bytes.Buffer) {
	tmpDir, err := ioutil.TempDir("", "jit")
	if err != nil {
		t.Fatal(err)
	}
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)

	setupRepo(t, tmpDir)

	return tmpDir, outStream, errStream
}

func commit(t *testing.T, dir string, message string) {
	t.Helper()

	os.Setenv("GIT_AUTHOR_NAME", "A. U. Thor")
	os.Setenv("GIT_AUTHOR_EMAIL", "author@example.com")
	defer os.Unsetenv("GIT_AUTHOR_NAME")
	defer os.Unsetenv("GIT_AUTHOR_EMAIL")

	stdin := strings.NewReader(message)
	Commit(dir, []string{}, stdin, new(bytes.Buffer), new(bytes.Buffer))
}

func checkout(tmpDir string, stdout, stderr *bytes.Buffer, revision string) {
	options := CheckOutOption{}
	checkout, _ := NewCheckOut(tmpDir, []string{revision}, options, stdout, stderr)
	checkout.Run()
}

func resolveRevision(t *testing.T, tmpDir, expression string) (string, error) {
	t.Helper()
	return repository.NewRevision(repo(t, tmpDir), expression).Resolve("")
}

func loadCommit(t *testing.T, tmpDir string, expression string) (database.GitObject, error) {
	t.Helper()
	rev, _ := resolveRevision(t, tmpDir, expression)
	return repo(t, tmpDir).Database.Load(rev)
}

func setupRepo(t *testing.T, tmpDir string) {
	t.Helper()

	Init([]string{tmpDir}, new(bytes.Buffer), new(bytes.Buffer))
}

func repo(t *testing.T, path string) *repository.Repository {
	t.Helper()
	rootPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	return repository.NewRepository(rootPath)
}

func writeFile(t *testing.T, path, name, content string) {
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

func makeExecutable(t *testing.T, path, name string) {
	t.Helper()

	err := os.Chmod(filepath.Join(path, name), 0755)
	if err != nil {
		t.Fatalf("Failed to make file executable: %s", err)
	}
}

func makeUnreadable(t *testing.T, path, name string) {
	t.Helper()

	err := os.Chmod(filepath.Join(path, name), 0200)
	if err != nil {
		t.Fatalf("Failed to make file unreadable: %s", err)
	}
}

func mkdir(t *testing.T, path, name string) {
	t.Helper()

	dir := filepath.Join(path, name)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %s", err)
	}
}

func touch(t *testing.T, path, name string) {
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

func delete(t *testing.T, path, name string) {
	t.Helper()

	filePath := filepath.Join(path, name)
	err := os.RemoveAll(filePath)
	if err != nil {
		t.Fatalf("Failed to delete file or directory: %s", err)
	}
}
