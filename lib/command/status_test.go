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

	Status(tmpDir, stdout, stderr)

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
	Status(tmpDir, stdout, stderr)

	expected := `?? file.txt
`
	if got := stdout.String(); got != expected {
		t.Errorf("want %q, but got %q", expected, got)
	}
}
