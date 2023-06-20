package command

import (
	"building-git/lib/command/commandtest"
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestStatusListUntrackedFilesInNameOrder(t *testing.T) {
	tmpDir, stdout, stderr := commandtest.SetupTestEnvironment(t)
	defer os.RemoveAll(tmpDir)

	filesToAdd := []*filesToAdd{
		{name: "file.txt", content: ""},
		{name: "another.txt", content: ""},
	}
	for _, file := range filesToAdd {
		commandtest.WriteFile(t, tmpDir, file.name, file.content)
	}

	statusCmd, err := NewStatus(tmpDir, stdout, stderr)
	if err != nil {
		t.Fatal(err)
	}
	statusCmd.Run()

	expected := `?? another.txt
?? file.txt
`
	if got := stdout.String(); got != expected {
		t.Errorf("want %q, but got %q", expected, got)
	}
}

func TestStatusListFilesAsUntrackedIfTheyAreNotInTheIndex(t *testing.T) {
	tmpDir, stdout, stderr := commandtest.SetupTestEnvironment(t)
	defer os.RemoveAll(tmpDir)

	commandtest.SetupRepo(t, tmpDir)
	commandtest.WriteFile(t, tmpDir, "committed.txt", "")

	Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))

	os.Setenv("GIT_AUTHOR_NAME", "A. U. Thor")
	os.Setenv("GIT_AUTHOR_EMAIL", "author@example.com")
	defer os.Unsetenv("GIT_AUTHOR_NAME")
	defer os.Unsetenv("GIT_AUTHOR_EMAIL")
	stdin := strings.NewReader("commit message")
	Commit(tmpDir, []string{}, stdin, new(bytes.Buffer), new(bytes.Buffer))

	commandtest.WriteFile(t, tmpDir, "file.txt", "")
	statusCmd, err := NewStatus(tmpDir, stdout, stderr)
	if err != nil {
		t.Fatal(err)
	}
	statusCmd.Run()

	expected := `?? file.txt
`
	if got := stdout.String(); got != expected {
		t.Errorf("want %q, but got %q", expected, got)
	}
}

func TestStatusListUntrackedDirectoriesNotTheirContents(t *testing.T) {
	tmpDir, stdout, stderr := commandtest.SetupTestEnvironment(t)
	defer os.RemoveAll(tmpDir)

	filesToAdd := []*filesToAdd{
		{name: "file.txt", content: ""},
		{name: "dir/another.txt", content: ""},
	}
	for _, file := range filesToAdd {
		commandtest.WriteFile(t, tmpDir, file.name, file.content)
	}

	statusCmd, err := NewStatus(tmpDir, stdout, stderr)
	if err != nil {
		t.Fatal(err)
	}
	statusCmd.Run()

	expected := `?? dir/
?? file.txt
`
	if got := stdout.String(); got != expected {
		t.Errorf("want %q, but got %q", expected, got)
	}
}

func TestStatusListUntrackedFilesInsideTrackedDirectories(t *testing.T) {
	tmpDir, stdout, stderr := commandtest.SetupTestEnvironment(t)
	defer os.RemoveAll(tmpDir)

	commandtest.SetupRepo(t, tmpDir)
	commandtest.WriteFile(t, tmpDir, "a/b/inner.txt", "")
	Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
	os.Setenv("GIT_AUTHOR_NAME", "A. U. Thor")
	os.Setenv("GIT_AUTHOR_EMAIL", "author@example.com")
	defer os.Unsetenv("GIT_AUTHOR_NAME")
	defer os.Unsetenv("GIT_AUTHOR_EMAIL")
	stdin := strings.NewReader("commit message")
	Commit(tmpDir, []string{}, stdin, new(bytes.Buffer), new(bytes.Buffer))

	filesToAdd := []*filesToAdd{
		{name: "a/outer.txt", content: ""},
		{name: "a/b/c/file.txt", content: ""},
	}
	for _, file := range filesToAdd {
		commandtest.WriteFile(t, tmpDir, file.name, file.content)
	}
	statusCmd, err := NewStatus(tmpDir, stdout, stderr)
	if err != nil {
		t.Fatal(err)
	}
	statusCmd.Run()

	expected := `?? a/b/c/
?? a/outer.txt
`
	if got := stdout.String(); got != expected {
		t.Errorf("want %q, but got %q", expected, got)
	}
}

func TestStatusDoesNotListEmptyUntrackedDirectories(t *testing.T) {
	tmpDir, stdout, stderr := commandtest.SetupTestEnvironment(t)
	defer os.RemoveAll(tmpDir)

	commandtest.Mkdir(t, tmpDir, "outer")

	statusCmd, err := NewStatus(tmpDir, stdout, stderr)
	if err != nil {
		t.Fatal(err)
	}
	statusCmd.Run()

	expected := ``
	if got := stdout.String(); got != expected {
		t.Errorf("want %q, but got %q", expected, got)
	}
}

func TestStatusListUntrackedDirectoriesIndirectlyContainFiles(t *testing.T) {
	tmpDir, stdout, stderr := commandtest.SetupTestEnvironment(t)
	defer os.RemoveAll(tmpDir)

	commandtest.WriteFile(t, tmpDir, "outer/inner/file.txt", "")

	statusCmd, err := NewStatus(tmpDir, stdout, stderr)
	if err != nil {
		t.Fatal(err)
	}
	statusCmd.Run()

	expected := `?? outer/
`
	if got := stdout.String(); got != expected {
		t.Errorf("want %q, but got %q", expected, got)
	}
}
