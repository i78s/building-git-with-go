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

func assertStatus(t *testing.T, tmpDir string, stdout *bytes.Buffer, stderr *bytes.Buffer, expected string) {
	args := StatusOption{
		Porcelain: true,
	}
	statusCmd, err := NewStatus(tmpDir, args, stdout, stderr)
	if err != nil {
		t.Fatal(err)
	}
	statusCmd.Run()

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

	expected := `?? file.txt
`
	assertStatus(t, tmpDir, stdout, stderr, expected)
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

	expected := `?? another.txt
?? file.txt
`
	assertStatus(t, tmpDir, stdout, stderr, expected)
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

	expected := `?? dir/
?? file.txt
`
	assertStatus(t, tmpDir, stdout, stderr, expected)
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

	expected := `?? a/b/c/
?? a/outer.txt
`
	assertStatus(t, tmpDir, stdout, stderr, expected)
}

func TestStatusDoesNotListEmptyUntrackedDirectories(t *testing.T) {
	tmpDir, stdout, stderr := commandtest.SetupTestEnvironment(t)
	defer os.RemoveAll(tmpDir)

	commandtest.Mkdir(t, tmpDir, "outer")

	expected := ``
	assertStatus(t, tmpDir, stdout, stderr, expected)
}

func TestStatusListUntrackedDirectoriesIndirectlyContainFiles(t *testing.T) {
	tmpDir, stdout, stderr := commandtest.SetupTestEnvironment(t)
	defer os.RemoveAll(tmpDir)

	commandtest.WriteFile(t, tmpDir, "outer/inner/file.txt", "")

	expected := `?? outer/
`
	assertStatus(t, tmpDir, stdout, stderr, expected)
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

		expected := ``
		assertStatus(t, tmpDir, stdout, stderr, expected)
	})

	t.Run("reports files with modified contents", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		commandtest.WriteFile(t, tmpDir, "1.txt", "changed")
		commandtest.WriteFile(t, tmpDir, "a/2.txt", "modified")

		expected := ` M 1.txt
 M a/2.txt
`
		assertStatus(t, tmpDir, stdout, stderr, expected)
	})

	t.Run("reports modified files with unchanged size", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		commandtest.WriteFile(t, tmpDir, "a/b/3.txt", "hello")

		expected := ` M a/b/3.txt
`
		assertStatus(t, tmpDir, stdout, stderr, expected)
	})

	t.Run("reports files with changed modes", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		commandtest.MakeExecutable(t, tmpDir, "a/2.txt")

		expected := ` M a/2.txt
`
		assertStatus(t, tmpDir, stdout, stderr, expected)
	})

	t.Run("prints nothing if a file is touched", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		commandtest.Touch(t, tmpDir, "1.txt")

		expected := ``
		assertStatus(t, tmpDir, stdout, stderr, expected)
	})

	t.Run("reports deleted files", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		commandtest.Delete(t, tmpDir, "a/2.txt")

		expected := ` D a/2.txt
`
		assertStatus(t, tmpDir, stdout, stderr, expected)
	})

	t.Run("reports files in deleted directories", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		commandtest.Delete(t, tmpDir, "a")

		expected := ` D a/2.txt
 D a/b/3.txt
`
		assertStatus(t, tmpDir, stdout, stderr, expected)
	})
}

func TestStatusHeadIndexChanges(t *testing.T) {
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

	t.Run("reports a file added to a tracked directory", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		commandtest.WriteFile(t, tmpDir, "a/4.txt", "four")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))

		expected := `A  a/4.txt
`
		assertStatus(t, tmpDir, stdout, stderr, expected)
	})

	t.Run("reports a file added to an untracked directory", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		commandtest.WriteFile(t, tmpDir, "d/e/5.txt", "five")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))

		expected := `A  d/e/5.txt
`
		assertStatus(t, tmpDir, stdout, stderr, expected)
	})

	t.Run("reports modified modes", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		commandtest.MakeExecutable(t, tmpDir, "1.txt")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))

		expected := `M  1.txt
`
		assertStatus(t, tmpDir, stdout, stderr, expected)
	})

	t.Run("reports modified contents", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		commandtest.WriteFile(t, tmpDir, "a/b/3.txt", "changed")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))

		expected := `M  a/b/3.txt
`
		assertStatus(t, tmpDir, stdout, stderr, expected)
	})

	t.Run("reports deleted files", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		commandtest.Delete(t, tmpDir, "1.txt")
		commandtest.Delete(t, tmpDir, ".git/index")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))

		expected := `D  1.txt
`
		assertStatus(t, tmpDir, stdout, stderr, expected)
	})

	t.Run("reports all deleted files inside directories", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		commandtest.Delete(t, tmpDir, "a")
		commandtest.Delete(t, tmpDir, ".git/index")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))

		expected := `D  a/2.txt
D  a/b/3.txt
`
		assertStatus(t, tmpDir, stdout, stderr, expected)
	})
}
