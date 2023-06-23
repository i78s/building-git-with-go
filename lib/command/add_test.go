package command

import (
	"building-git/lib"
	"building-git/lib/command/commandtest"
	"os"
	"reflect"
	"regexp"
	"testing"
)

type filesToAdd struct {
	name       string
	content    string
	executable bool
}

type indexEntry struct {
	mode int
	path string
}

func assertIndex(t *testing.T, tmpDir string, expected []*indexEntry) {
	repo := lib.NewRepository(tmpDir)
	repo.Index.Load()
	actual := []*indexEntry{}
	for _, entry := range repo.Index.EachEntry() {
		actual = append(actual, &indexEntry{mode: entry.Mode(), path: entry.Key()})
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}

func TestAddRegularFileToIndex(t *testing.T) {
	tmpDir, stdout, stderr := commandtest.SetupTestEnvironment(t)
	defer os.RemoveAll(tmpDir)

	filesToAdd := []*filesToAdd{
		{name: "hello.txt", content: "hello"},
	}
	expected := []*indexEntry{
		{mode: 0o100644, path: "hello.txt"},
	}

	fileNames := make([]string, len(filesToAdd))
	for i, file := range filesToAdd {
		fileNames[i] = file.name
		commandtest.WriteFile(t, tmpDir, file.name, file.content)
	}

	Add(tmpDir, fileNames, stdout, stderr)
	assertIndex(t, tmpDir, expected)
}

func TestAddExecutableFileToIndex(t *testing.T) {
	tmpDir, stdout, stderr := commandtest.SetupTestEnvironment(t)
	defer os.RemoveAll(tmpDir)

	filesToAdd := []*filesToAdd{
		{name: "hello.txt", content: "hello"},
	}
	expected := []*indexEntry{
		{mode: 0o100755, path: "hello.txt"},
	}

	fileNames := make([]string, len(filesToAdd))
	for i, file := range filesToAdd {
		fileNames[i] = file.name
		commandtest.WriteFile(t, tmpDir, file.name, file.content)
		commandtest.MakeExecutable(t, tmpDir, file.name)
	}

	Add(tmpDir, fileNames, stdout, stderr)
	assertIndex(t, tmpDir, expected)
}

func TestAddMultipleFilesToIndex(t *testing.T) {
	tmpDir, stdout, stderr := commandtest.SetupTestEnvironment(t)
	defer os.RemoveAll(tmpDir)

	filesToAdd := []*filesToAdd{
		{name: "hello.txt", content: "hello"},
		{name: "world.txt", content: "world"},
	}
	expected := []*indexEntry{
		{mode: 0o100644, path: "hello.txt"},
		{mode: 0o100644, path: "world.txt"},
	}

	fileNames := make([]string, len(filesToAdd))
	for i, file := range filesToAdd {
		fileNames[i] = file.name
		commandtest.WriteFile(t, tmpDir, file.name, file.content)
	}

	Add(tmpDir, fileNames, stdout, stderr)
	assertIndex(t, tmpDir, expected)
}

func TestIncrementallyAddFilesToIndex(t *testing.T) {
	tmpDir, stdout, stderr := commandtest.SetupTestEnvironment(t)
	defer os.RemoveAll(tmpDir)

	filesToAdd := []*filesToAdd{
		{name: "hello.txt", content: "hello"},
		{name: "world.txt", content: "world"},
	}

	fileNames := make([]string, len(filesToAdd))
	for i, file := range filesToAdd {
		fileNames[i] = file.name
		commandtest.WriteFile(t, tmpDir, file.name, file.content)
	}

	Add(tmpDir, []string{fileNames[0]}, stdout, stderr)
	assertIndex(t, tmpDir, []*indexEntry{
		{mode: 0o100644, path: "hello.txt"},
	})
	Add(tmpDir, []string{fileNames[1]}, stdout, stderr)
	assertIndex(t, tmpDir, []*indexEntry{
		{mode: 0o100644, path: "hello.txt"},
		{mode: 0o100644, path: "world.txt"},
	})
}

func TestAddDirectoryToIndex(t *testing.T) {
	tmpDir, stdout, stderr := commandtest.SetupTestEnvironment(t)
	defer os.RemoveAll(tmpDir)

	filesToAdd := []*filesToAdd{
		{name: "a-dir/nested.txt", content: "content"},
	}

	fileNames := make([]string, len(filesToAdd))
	for i, file := range filesToAdd {
		fileNames[i] = file.name
		commandtest.WriteFile(t, tmpDir, file.name, file.content)
	}

	Add(tmpDir, []string{"a-dir"}, stdout, stderr)
	assertIndex(t, tmpDir, []*indexEntry{
		{mode: 0o100644, path: "a-dir/nested.txt"},
	})
}

func TestAddRepositoryRootToIndex(t *testing.T) {
	tmpDir, stdout, stderr := commandtest.SetupTestEnvironment(t)
	defer os.RemoveAll(tmpDir)

	filesToAdd := []*filesToAdd{
		{name: "a/b/c/file.txt", content: "content"},
	}

	fileNames := make([]string, len(filesToAdd))
	for i, file := range filesToAdd {
		fileNames[i] = file.name
		commandtest.WriteFile(t, tmpDir, file.name, file.content)
	}

	Add(tmpDir, []string{"."}, stdout, stderr)
	assertIndex(t, tmpDir, []*indexEntry{
		{mode: 0o100644, path: "a/b/c/file.txt"},
	})
}

func TestIsSilentOnSuccess(t *testing.T) {
	tmpDir, stdout, stderr := commandtest.SetupTestEnvironment(t)
	defer os.RemoveAll(tmpDir)

	filesToAdd := []*filesToAdd{
		{name: "hello.txt", content: "hello"},
	}

	fileNames := make([]string, len(filesToAdd))
	for i, file := range filesToAdd {
		fileNames[i] = file.name
		commandtest.WriteFile(t, tmpDir, file.name, file.content)
	}

	code := Add(tmpDir, []string{"hello.txt"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("Add returned error: %v", code)
	}
	if got := stdout.String(); got != "" {
		t.Errorf("want %q, but got %q", "", got)
	}
	if got := stderr.String(); got != "" {
		t.Errorf("want %q, but got %q", "", got)
	}
}

func TestFailsForNonExistentFiles(t *testing.T) {
	tmpDir, stdout, stderr := commandtest.SetupTestEnvironment(t)
	defer os.RemoveAll(tmpDir)

	code := Add(tmpDir, []string{"no-such-file"}, stdout, stderr)
	if code != 128 {
		t.Fatalf("Add returned error: %v", code)
	}
	if got := stdout.String(); got != "" {
		t.Errorf("want %q, but got %q", "", got)
	}
	matched, err := regexp.MatchString("fatal: pathspec .* did not match any files", stderr.String())
	if err != nil {
		t.Fatalf("Failed to execute regex match: %v", err)
	}
	if !matched {
		t.Errorf("stderr output did not match expected pattern")
	}
	assertIndex(t, tmpDir, []*indexEntry{})
}

func TestFailsForUnreadableFiles(t *testing.T) {
	tmpDir, stdout, stderr := commandtest.SetupTestEnvironment(t)
	defer os.RemoveAll(tmpDir)

	filesToAdd := []*filesToAdd{
		{name: "secret.txt", content: ""},
	}

	fileNames := make([]string, len(filesToAdd))
	for i, file := range filesToAdd {
		fileNames[i] = file.name
		commandtest.WriteFile(t, tmpDir, file.name, file.content)
		commandtest.MakeUnreadable(t, tmpDir, file.name)
	}

	code := Add(tmpDir, []string{"secret.txt"}, stdout, stderr)
	if code != 128 {
		t.Fatalf("Add returned error: %v", code)
	}
	if got := stdout.String(); got != "" {
		t.Errorf("want %q, but got %q", "", got)
	}
	if got := stderr.String(); got != "fatal: open('secret.txt'): Permission denied" {
		t.Errorf("want %q, but got %q", "fatal: open('secret.txt'): Permission denied", got)
	}
	assertIndex(t, tmpDir, []*indexEntry{})
}

func TestFailsIfIndexIsLocked(t *testing.T) {
	tmpDir, stdout, stderr := commandtest.SetupTestEnvironment(t)
	defer os.RemoveAll(tmpDir)

	filesToAdd := []*filesToAdd{
		{name: "file.txt", content: ""},
		{name: ".git/index.lock", content: ""},
	}

	fileNames := make([]string, len(filesToAdd))
	for i, file := range filesToAdd {
		fileNames[i] = file.name
		commandtest.WriteFile(t, tmpDir, file.name, file.content)
	}

	code := Add(tmpDir, []string{"file.txt"}, stdout, stderr)
	if code != 128 {
		t.Fatalf("Add returned error: %v", code)
	}
	if got := stdout.String(); got != "" {
		t.Errorf("want %q, but got %q", "", got)
	}
	matched, err := regexp.MatchString("fatal: open .*.git/index.lock: file exists", stderr.String())
	if err != nil {
		t.Fatalf("Failed to execute regex match: %v", err)
	}
	if !matched {
		t.Errorf("stderr output did not match expected pattern")
	}
	assertIndex(t, tmpDir, []*indexEntry{})
}
