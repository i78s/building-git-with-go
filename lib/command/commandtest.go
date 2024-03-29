package command

import (
	"building-git/lib/command/write_commit"
	"building-git/lib/database"
	"building-git/lib/editor"
	"building-git/lib/repository"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
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

type MockEditor struct {
	path string
	edit string
}

func NewMockEditor(path, edit string) *MockEditor {
	return &MockEditor{
		path: path,
		edit: edit,
	}
}

func (m *MockEditor) Run() error {
	file, err := os.OpenFile(m.path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err = file.Write([]byte(m.edit)); err != nil {
		fmt.Println("Error writing to file:", err)
		return err
	}
	return nil
}

func commit(t *testing.T, dir string, stdout, stderr *bytes.Buffer, options CommitOption, now time.Time) int {
	t.Helper()

	os.Setenv("GIT_AUTHOR_NAME", "A. U. Thor")
	os.Setenv("GIT_AUTHOR_EMAIL", "author@example.com")
	defer os.Unsetenv("GIT_AUTHOR_NAME")
	defer os.Unsetenv("GIT_AUTHOR_EMAIL")

	if options.EditorCmd == nil {
		options.EditorCmd = func(path string) editor.Executable {
			return &MockEditor{path: path}
		}
	}

	c, _ := NewCommit(dir, []string{}, options, stdout, stderr)
	return c.Run(now)
}

func cherryPick(t *testing.T, dir string, stdout, stderr *bytes.Buffer, args []string, options CherryPickOption) int {
	t.Helper()

	os.Setenv("GIT_AUTHOR_NAME", "A. U. Thor")
	os.Setenv("GIT_AUTHOR_EMAIL", "author@example.com")
	defer os.Unsetenv("GIT_AUTHOR_NAME")
	defer os.Unsetenv("GIT_AUTHOR_EMAIL")

	if options.EditorCmd == nil {
		options.EditorCmd = func(path string) editor.Executable {
			return &MockEditor{path: path}
		}
	}

	c, _ := NewCherryPick(dir, args, options, stdout, stderr)
	return c.Run()
}

func commitTree(t *testing.T, tmpDir, message string, files map[string]string, now time.Time) {
	t.Helper()

	for path, contents := range files {
		writeFile(t, tmpDir, path, contents)
	}
	Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
	options := CommitOption{
		ReadOption: write_commit.ReadOption{
			Message: message,
		},
	}
	commit(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), options, now)
}

func checkout(tmpDir string, stdout, stderr *bytes.Buffer, revision string) {
	options := CheckOutOption{}
	checkout, _ := NewCheckOut(tmpDir, []string{revision}, options, stdout, stderr)
	checkout.Run()
}

func mergeCommit(t *testing.T, tmpDir, branch string, options MergeOption, stdout, stderr *bytes.Buffer) int {
	t.Helper()

	os.Setenv("GIT_AUTHOR_NAME", "A. U. Thor")
	os.Setenv("GIT_AUTHOR_EMAIL", "author@example.com")
	defer os.Unsetenv("GIT_AUTHOR_NAME")
	defer os.Unsetenv("GIT_AUTHOR_EMAIL")
	options.EditorCmd = func(path string) editor.Executable {
		return &MockEditor{path: path}
	}
	mergeCmd, _ := NewMerge(tmpDir, []string{branch}, options, stdout, stderr)
	return mergeCmd.Run()
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
	os.RemoveAll(filePath)
}

func assertExecutable(t *testing.T, path, name string) {
	filePath := filepath.Join(path, name)
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Cannot stat file: %v", err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Fatalf("The file is NOT executable: %s", filePath)
	}
}

func assertWorkspace(t *testing.T, dir string, expected map[string]string) {
	rootPath, _ := filepath.Abs(dir)
	repo := repository.NewRepository(rootPath)

	actual := make(map[string]string)

	files, _ := repo.Workspace.ListFiles(rootPath)
	sort.Strings(files)
	for _, path := range files {
		actual[path], _ = repo.Workspace.ReadFile(path)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("want %q, but got %q", expected, actual)
	}
}

func assertGitStatus(t *testing.T, tmpDir string, stdout *bytes.Buffer, stderr *bytes.Buffer, expected string) {
	args := []string{}
	options := StatusOption{
		Porcelain: true,
	}
	statusCmd, err := NewStatus(tmpDir, args, options, stdout, stderr)
	if err != nil {
		t.Fatal(err)
	}
	statusCmd.Run()

	if got := stdout.String(); got != expected {
		t.Errorf("want %q, but got %q", expected, got)
	}
}

func assertDiff(
	t *testing.T,
	tmpDir string,
	args []string,
	options DiffOption,
	stdout *bytes.Buffer,
	stderr *bytes.Buffer,
	expected string,
) {
	cmd, err := NewDiff(tmpDir, args, options, stdout, stderr)
	if err != nil {
		t.Fatal(err)
	}
	cmd.Run()

	if got := stdout.String(); got != expected {
		t.Errorf("want %q, but got %q", expected, got)
	}
}
