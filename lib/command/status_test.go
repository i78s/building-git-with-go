package command

import (
	"building-git/lib/command/commandtest"
	"bytes"
	"os"
	"strings"
	"testing"
)

func commit(t *testing.T, dir string, message string) {
	t.Helper()

	os.Setenv("GIT_AUTHOR_NAME", "A. U. Thor")
	os.Setenv("GIT_AUTHOR_EMAIL", "author@example.com")
	defer os.Unsetenv("GIT_AUTHOR_NAME")
	defer os.Unsetenv("GIT_AUTHOR_EMAIL")

	stdin := strings.NewReader(message)
	Commit(dir, []string{}, stdin, new(bytes.Buffer), new(bytes.Buffer))
}

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

	commandtest.WriteFile(t, tmpDir, "committed.txt", "")

	Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
	commit(t, tmpDir, "commit message")

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

	commandtest.WriteFile(t, tmpDir, "a/b/inner.txt", "")
	Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
	commit(t, tmpDir, "commit message")

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

func TestStatusIndexWorkspaceChanges(t *testing.T) {
	setup := func() (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = commandtest.SetupTestEnvironment(t)

		filesToAdd := []*filesToAdd{
			{name: "1.txt", content: "one"},
			{name: "a/2.txt", content: "two"},
			{name: "a/b/3.txt", content: "three"},
		}
		for _, file := range filesToAdd {
			commandtest.WriteFile(t, tmpDir, file.name, file.content)
		}

		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
		commit(t, tmpDir, "commit message")

		return
	}

	t.Run("prints nothing when no files are changed", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		statusCmd, err := NewStatus(tmpDir, stdout, stderr)
		if err != nil {
			t.Fatal(err)
		}
		statusCmd.Run()

		expected := ``
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("reports files with modified contents", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		commandtest.WriteFile(t, tmpDir, "1.txt", "changed")
		commandtest.WriteFile(t, tmpDir, "a/2.txt", "modified")

		statusCmd, err := NewStatus(tmpDir, stdout, stderr)
		if err != nil {
			t.Fatal(err)
		}
		statusCmd.Run()

		expected := ` M 1.txt
 M a/2.txt
`
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("reports modified files with unchanged size", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		commandtest.WriteFile(t, tmpDir, "a/b/3.txt", "hello")

		statusCmd, err := NewStatus(tmpDir, stdout, stderr)
		if err != nil {
			t.Fatal(err)
		}
		statusCmd.Run()

		expected := ` M a/b/3.txt
`
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("reports files with changed modes", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		commandtest.MakeExecutable(t, tmpDir, "a/2.txt")

		statusCmd, err := NewStatus(tmpDir, stdout, stderr)
		if err != nil {
			t.Fatal(err)
		}
		statusCmd.Run()

		expected := ` M a/2.txt
`
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("prints nothing if a file is touched", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		commandtest.Touch(t, tmpDir, "1.txt")

		statusCmd, err := NewStatus(tmpDir, stdout, stderr)
		if err != nil {
			t.Fatal(err)
		}
		statusCmd.Run()

		expected := ``
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})
}
