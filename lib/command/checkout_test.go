package command

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func assertNoent(t *testing.T, repoPath, filename string) {
	_, err := os.Stat(filepath.Join(repoPath, filename))
	if !os.IsNotExist(err) {
		t.Errorf("File %s should not exist", filename)
	}
}

func assertStaleFile(t *testing.T, stderr *bytes.Buffer, filename string) {
	expected := fmt.Sprintf(`error: Your local changes to the following files would be overwritten by checkout:
	%s
Please commit your changes or stash them before you switch branches.
Aborting`, filename)

	if got := stderr.String(); got != expected {
		t.Errorf("want %q, but got %q", expected, got)
	}
}

func assertStaleDirectory(t *testing.T, stderr *bytes.Buffer, filename string) {
	expected := fmt.Sprintf(`error: Updating the following directories would lose untracked files in them:
	%s

Aborting`, filename)

	if got := stderr.String(); got != expected {
		t.Errorf("want %q, but got %q", expected, got)
	}
}

func assertOverwriteConflict(t *testing.T, stderr *bytes.Buffer, filename string) {
	expected := fmt.Sprintf(`error: The following untracked working tree files would be overwritten by checkout:
	%s
Please move or remove them before you switch branches.
Aborting`, filename)

	if got := stderr.String(); got != expected {
		t.Errorf("want %q, but got %q", expected, got)
	}
}

func assertRemoveConflict(t *testing.T, stderr *bytes.Buffer, filename string) {
	expected := fmt.Sprintf(`error: The following untracked working tree files would be removed by checkout:
	%s
Please move or remove them before you switch branches.
Aborting`, filename)

	if got := stderr.String(); got != expected {
		t.Errorf("want %q, but got %q", expected, got)
	}
}

var baseFiles = map[string]string{
	"1.txt":             "1",
	"outer/2.txt":       "2",
	"outer/inner/3.txt": "3",
}

func TestCheckOutWithSetOfFiles(t *testing.T) {
	commitAll := func(t *testing.T, tmpDir string) {
		delete(t, tmpDir, ".git/index")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
		commit(t, tmpDir, "change", time.Now())
	}

	setup := func() (tmpDir string, stdout, stderr *bytes.Buffer, commitAndCheckout func(revision string)) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		for name, content := range baseFiles {
			writeFile(t, tmpDir, name, content)
		}

		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
		commit(t, tmpDir, "first", time.Now())

		commitAndCheckout = func(revision string) {
			commitAll(t, tmpDir)
			checkout(tmpDir, stdout, stderr, revision)
		}

		return
	}

	t.Run("updates a changed file", func(t *testing.T) {
		tmpDir, stdout, stderr, commitAndCheckout := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "1.txt", "changed")
		commitAndCheckout("@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("fails to update a modified file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "1.txt", "changed")
		commitAll(t, tmpDir)

		writeFile(t, tmpDir, "1.txt", "conflict")
		checkout(tmpDir, stdout, stderr, "@^")

		assertStaleFile(t, stderr, "1.txt")
	})

	t.Run("fails to update a modified-equal file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "1.txt", "changed")
		commitAll(t, tmpDir)

		writeFile(t, tmpDir, "1.txt", "1")
		checkout(tmpDir, stdout, stderr, "@^")

		assertStaleFile(t, stderr, "1.txt")
	})

	t.Run("fails to update a changed-mode file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "1.txt", "changed")
		commitAll(t, tmpDir)

		makeExecutable(t, tmpDir, "1.txt")
		checkout(tmpDir, stdout, stderr, "@^")

		assertStaleFile(t, stderr, "1.txt")
	})

	t.Run("restores a deleted file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "1.txt", "changed")
		commitAll(t, tmpDir)

		delete(t, tmpDir, "1.txt")
		checkout(tmpDir, stdout, stderr, "@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("restores files from a deleted directory", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/inner/3.txt", "changed")
		commitAll(t, tmpDir)

		delete(t, tmpDir, "outer")
		checkout(tmpDir, stdout, stderr, "@^")

		assertWorkspace(t, tmpDir, map[string]string{
			"1.txt":             "1",
			"outer/inner/3.txt": "3",
		})
		assertStatus(t, tmpDir, stdout, stderr, ` D outer/2.txt
`)
	})

	t.Run("fails to update a staged file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "1.txt", "changed")
		commitAll(t, tmpDir)

		writeFile(t, tmpDir, "1.txt", "conflict")
		Add(tmpDir, []string{"."}, stdout, stderr)

		checkout(tmpDir, stdout, stderr, "@^")
		assertStaleFile(t, stderr, "1.txt")
	})

	t.Run("updates a staged-equal file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "1.txt", "changed")
		commitAll(t, tmpDir)

		writeFile(t, tmpDir, "1.txt", "1")
		Add(tmpDir, []string{"."}, stdout, stderr)
		checkout(tmpDir, stdout, stderr, "@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("fails to update a staged changed-mode file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "1.txt", "changed")
		commitAll(t, tmpDir)

		makeExecutable(t, tmpDir, "1.txt")
		Add(tmpDir, []string{"."}, stdout, stderr)

		checkout(tmpDir, stdout, stderr, "@^")
		assertStaleFile(t, stderr, "1.txt")
	})

	t.Run("fails to update an unindexed file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "1.txt", "changed")
		commitAll(t, tmpDir)

		delete(t, tmpDir, "1.txt")
		delete(t, tmpDir, ".git/index")
		Add(tmpDir, []string{"."}, stdout, stderr)

		checkout(tmpDir, stdout, stderr, "@^")
		assertStaleFile(t, stderr, "1.txt")
	})

	t.Run("fails to update an unindexed and untracked file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "1.txt", "changed")
		commitAll(t, tmpDir)

		delete(t, tmpDir, "1.txt")
		delete(t, tmpDir, ".git/index")
		Add(tmpDir, []string{"."}, stdout, stderr)
		writeFile(t, tmpDir, "1.txt", "conflict")

		checkout(tmpDir, stdout, stderr, "@^")
		assertStaleFile(t, stderr, "1.txt")
	})

	t.Run("fails to update an unindexed directory", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/inner/3.txt", "changed")
		commitAll(t, tmpDir)

		delete(t, tmpDir, "outer/inner")
		delete(t, tmpDir, ".git/index")
		Add(tmpDir, []string{"."}, stdout, stderr)

		checkout(tmpDir, stdout, stderr, "@^")
		assertStaleFile(t, stderr, "outer/inner/3.txt")
	})

	t.Run("fails to update with a file at a parent path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/inner/3.txt", "changed")
		commitAll(t, tmpDir)

		delete(t, tmpDir, "outer/inner")
		writeFile(t, tmpDir, "outer/inner", "conflict")

		checkout(tmpDir, stdout, stderr, "@^")
		assertStaleFile(t, stderr, "outer/inner/3.txt")
	})

	t.Run("fails to update with a staged file at a parent path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/inner/3.txt", "changed")
		commitAll(t, tmpDir)

		delete(t, tmpDir, "outer/inner")
		writeFile(t, tmpDir, "outer/inner", "conflict")
		Add(tmpDir, []string{"."}, stdout, stderr)

		checkout(tmpDir, stdout, stderr, "@^")
		assertStaleFile(t, stderr, "outer/inner/3.txt")
	})

	t.Run("fails to update with an unstaged file at a parent path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/inner/3.txt", "changed")
		commitAll(t, tmpDir)

		delete(t, tmpDir, "outer/inner")
		delete(t, tmpDir, ".git/index")
		Add(tmpDir, []string{"."}, stdout, stderr)
		writeFile(t, tmpDir, "outer/inner", "conflict")

		checkout(tmpDir, stdout, stderr, "@^")
		assertStaleFile(t, stderr, "outer/inner/3.txt")
	})

	t.Run("fails to update with a file at a child path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/2.txt", "changed")
		commitAll(t, tmpDir)

		delete(t, tmpDir, "outer/2.txt")
		writeFile(t, tmpDir, "outer/2.txt/extra.log", "conflict")

		checkout(tmpDir, stdout, stderr, "@^")
		assertStaleFile(t, stderr, "outer/2.txt")
	})

	t.Run("fails to update with a staged file at a child path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/2.txt", "changed")
		commitAll(t, tmpDir)

		delete(t, tmpDir, "outer/2.txt")
		writeFile(t, tmpDir, "outer/2.txt/extra.log", "conflict")
		Add(tmpDir, []string{"."}, stdout, stderr)

		checkout(tmpDir, stdout, stderr, "@^")
		assertStaleFile(t, stderr, "outer/2.txt")
	})

	t.Run("removes a file", func(t *testing.T) {
		tmpDir, stdout, stderr, commitAndCheckout := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "94.txt", "94")
		commitAndCheckout("@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("removes a file from an existing directory", func(t *testing.T) {
		tmpDir, stdout, stderr, commitAndCheckout := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/94.txt", "94")
		commitAndCheckout("@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("removes a file from a new directory", func(t *testing.T) {
		tmpDir, stdout, stderr, commitAndCheckout := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "new/94.txt", "94")
		commitAndCheckout("@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertNoent(t, tmpDir, "new")
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("removes a file from a new nested directory", func(t *testing.T) {
		tmpDir, stdout, stderr, commitAndCheckout := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "new/inner/94.txt", "94")
		commitAndCheckout("@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertNoent(t, tmpDir, "new")
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("removes a file from a non-empty directory", func(t *testing.T) {
		tmpDir, stdout, stderr, commitAndCheckout := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/94.txt", "94")
		commitAndCheckout("@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("fails to remove a modified file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/94.txt", "94")
		commitAll(t, tmpDir)

		writeFile(t, tmpDir, "outer/94.txt", "conflict")
		checkout(tmpDir, stdout, stderr, "@^")

		assertStaleFile(t, stderr, "outer/94.txt")
	})

	t.Run("fails to remove a changed-mode file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/94.txt", "94")
		commitAll(t, tmpDir)

		makeExecutable(t, tmpDir, "outer/94.txt")
		checkout(tmpDir, stdout, stderr, "@^")

		assertStaleFile(t, stderr, "outer/94.txt")
	})

	t.Run("leaves a deleted file deleted", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/94.txt", "94")
		commitAll(t, tmpDir)

		delete(t, tmpDir, "outer/94.txt")
		checkout(tmpDir, stdout, stderr, "@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("leaves a deleted directory deleted", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/inner/94.txt", "94")
		commitAll(t, tmpDir)

		delete(t, tmpDir, "outer/inner")
		checkout(tmpDir, stdout, stderr, "@^")

		assertWorkspace(t, tmpDir, map[string]string{
			"1.txt":       "1",
			"outer/2.txt": "2",
		})
		assertStatus(t, tmpDir, stdout, stderr, ` D outer/inner/3.txt
`)
	})

	t.Run("fails to remove a staged file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/94.txt", "94")
		commitAll(t, tmpDir)

		writeFile(t, tmpDir, "outer/94.txt", "conflict")
		Add(tmpDir, []string{"."}, stdout, stderr)
		checkout(tmpDir, stdout, stderr, "@^")

		assertStaleFile(t, stderr, "outer/94.txt")
	})

	t.Run("fails to remove a staged changed-mode file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/94.txt", "94")
		commitAll(t, tmpDir)

		makeExecutable(t, tmpDir, "outer/94.txt")
		Add(tmpDir, []string{"."}, stdout, stderr)
		checkout(tmpDir, stdout, stderr, "@^")

		assertStaleFile(t, stderr, "outer/94.txt")
	})

	t.Run("leaves an unindexed file deleted", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/94.txt", "94")
		commitAll(t, tmpDir)

		delete(t, tmpDir, "outer/94.txt")
		delete(t, tmpDir, ".git/index")
		Add(tmpDir, []string{"."}, stdout, stderr)
		checkout(tmpDir, stdout, stderr, "@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("fails to remove an unindexed and untracked file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/94.txt", "94")
		commitAll(t, tmpDir)

		delete(t, tmpDir, "outer/94.txt")
		delete(t, tmpDir, ".git/index")
		Add(tmpDir, []string{"."}, stdout, stderr)
		writeFile(t, tmpDir, "outer/94.txt", "conflict")
		checkout(tmpDir, stdout, stderr, "@^")

		assertRemoveConflict(t, stderr, "outer/94.txt")
	})

	t.Run("leaves an unindexed directory deleted", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/inner/94.txt", "94")
		commitAll(t, tmpDir)

		delete(t, tmpDir, "outer/inner")
		delete(t, tmpDir, ".git/index")
		Add(tmpDir, []string{"."}, stdout, stderr)
		checkout(tmpDir, stdout, stderr, "@^")

		assertWorkspace(t, tmpDir, map[string]string{
			"1.txt":       "1",
			"outer/2.txt": "2",
		})
		assertStatus(t, tmpDir, stdout, stderr, `D  outer/inner/3.txt
`)
	})

	t.Run("fails to remove with a file at a parent path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/inner/94.txt", "94")
		commitAll(t, tmpDir)

		delete(t, tmpDir, "outer/inner")
		writeFile(t, tmpDir, "outer/inner", "conflict")
		checkout(tmpDir, stdout, stderr, "@^")

		assertStaleFile(t, stderr, "outer/inner/94.txt")
	})

	t.Run("removes a file with a staged file at a parent path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/inner/94.txt", "94")
		commitAll(t, tmpDir)

		delete(t, tmpDir, "outer/inner")
		writeFile(t, tmpDir, "outer/inner", "conflict")
		Add(tmpDir, []string{"."}, stdout, stderr)
		checkout(tmpDir, stdout, stderr, "@^")

		assertWorkspace(t, tmpDir, map[string]string{
			"1.txt":       "1",
			"outer/2.txt": "2",
			"outer/inner": "conflict",
		})
		assertStatus(t, tmpDir, stdout, stderr, `A  outer/inner
D  outer/inner/3.txt
`)
	})

	t.Run("fails to remove with an unstaged file at a parent path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/inner/94.txt", "94")
		commitAll(t, tmpDir)

		delete(t, tmpDir, "outer/inner")
		delete(t, tmpDir, ".git/index")
		Add(tmpDir, []string{"."}, stdout, stderr)
		writeFile(t, tmpDir, "outer/inner", "conflict")
		checkout(tmpDir, stdout, stderr, "@^")

		assertRemoveConflict(t, stderr, "outer/inner")
	})

	t.Run("fails to remove with a file at a child path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/94.txt", "94")
		commitAll(t, tmpDir)

		delete(t, tmpDir, "outer/94.txt")
		writeFile(t, tmpDir, "outer/94.txt/extra.log", "conflict")
		checkout(tmpDir, stdout, stderr, "@^")

		assertStaleFile(t, stderr, "outer/94.txt")
	})

	t.Run("removes a file with a staged file at a child path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/94.txt", "94")
		commitAll(t, tmpDir)

		delete(t, tmpDir, "outer/94.txt")
		writeFile(t, tmpDir, "outer/94.txt/extra.log", "conflict")
		Add(tmpDir, []string{"."}, stdout, stderr)
		checkout(tmpDir, stdout, stderr, "@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("add a file", func(t *testing.T) {
		tmpDir, stdout, stderr, commitAndCheckout := setup()
		defer os.RemoveAll(tmpDir)

		delete(t, tmpDir, "1.txt")
		commitAndCheckout("@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("add a file to a directory", func(t *testing.T) {
		tmpDir, stdout, stderr, commitAndCheckout := setup()
		defer os.RemoveAll(tmpDir)

		delete(t, tmpDir, "outer/2.txt")
		commitAndCheckout("@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("add a directory", func(t *testing.T) {
		tmpDir, stdout, stderr, commitAndCheckout := setup()
		defer os.RemoveAll(tmpDir)

		delete(t, tmpDir, "outer")
		commitAndCheckout("@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("fails to add an untracked file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		delete(t, tmpDir, "outer/2.txt")
		commitAll(t, tmpDir)

		writeFile(t, tmpDir, "outer/2.txt", "conflict")
		checkout(tmpDir, stdout, stderr, "@^")

		assertOverwriteConflict(t, stderr, "outer/2.txt")
	})

	t.Run("fails to add an added file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		delete(t, tmpDir, "outer/2.txt")
		commitAll(t, tmpDir)

		writeFile(t, tmpDir, "outer/2.txt", "conflict")
		Add(tmpDir, []string{"."}, stdout, stderr)
		checkout(tmpDir, stdout, stderr, "@^")

		assertStaleFile(t, stderr, "outer/2.txt")
	})

	t.Run("adds a staged-equal file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		delete(t, tmpDir, "outer/2.txt")
		commitAll(t, tmpDir)

		writeFile(t, tmpDir, "outer/2.txt", "2")
		Add(tmpDir, []string{"."}, stdout, stderr)
		checkout(tmpDir, stdout, stderr, "@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("fails to add with an untracked file at a parent path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		delete(t, tmpDir, "outer/inner/3.txt")
		commitAll(t, tmpDir)

		delete(t, tmpDir, "outer/inner")
		writeFile(t, tmpDir, "outer/inner", "conflict")
		checkout(tmpDir, stdout, stderr, "@^")

		assertOverwriteConflict(t, stderr, "outer/inner")
	})

	t.Run("adds a file with an added file at a parent path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		delete(t, tmpDir, "outer/inner/3.txt")
		commitAll(t, tmpDir)

		delete(t, tmpDir, "outer/inner")
		writeFile(t, tmpDir, "outer/inner", "conflict")
		Add(tmpDir, []string{"."}, stdout, stderr)
		checkout(tmpDir, stdout, stderr, "@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("fails to add with an untracked file at a child path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		delete(t, tmpDir, "outer/2.txt")
		commitAll(t, tmpDir)

		writeFile(t, tmpDir, "outer/2.txt/extra.log", "conflict")
		checkout(tmpDir, stdout, stderr, "@^")

		assertStaleDirectory(t, stderr, "outer/2.txt")
	})

	t.Run("adds a file with an added file at a child path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		delete(t, tmpDir, "outer/2.txt")
		commitAll(t, tmpDir)

		writeFile(t, tmpDir, "outer/2.txt/extra.log", "conflict")
		Add(tmpDir, []string{"."}, stdout, stderr)
		checkout(tmpDir, stdout, stderr, "@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("replaces a file with a directory", func(t *testing.T) {
		tmpDir, stdout, stderr, commitAndCheckout := setup()
		defer os.RemoveAll(tmpDir)

		delete(t, tmpDir, "outer/inner")
		writeFile(t, tmpDir, "outer/inner", "in")
		commitAndCheckout("@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("replaces a directory with a file", func(t *testing.T) {
		tmpDir, stdout, stderr, commitAndCheckout := setup()
		defer os.RemoveAll(tmpDir)

		delete(t, tmpDir, "outer/2.txt")
		writeFile(t, tmpDir, "outer/2.txt/nested.log", "nested")
		commitAndCheckout("@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("maintains workspace modifications", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "1.txt", "changed")
		commitAll(t, tmpDir)

		writeFile(t, tmpDir, "outer/2.txt", "hello")
		delete(t, tmpDir, "outer/inner")
		checkout(tmpDir, stdout, stderr, "@^")

		assertWorkspace(t, tmpDir, map[string]string{
			"1.txt":       "1",
			"outer/2.txt": "hello",
		})
		assertStatus(t, tmpDir, stdout, stderr, ` M outer/2.txt
 D outer/inner/3.txt
`)
	})

	t.Run("maintains index modifications", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "1.txt", "changed")
		commitAll(t, tmpDir)

		writeFile(t, tmpDir, "outer/2.txt", "hello")
		writeFile(t, tmpDir, "outer/inner/4.txt", "world")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
		checkout(tmpDir, stdout, stderr, "@^")

		expectedFiles := baseFiles
		for path, content := range map[string]string{
			"outer/2.txt":       "hello",
			"outer/inner/4.txt": "world",
		} {
			expectedFiles[path] = content
		}
		assertWorkspace(t, tmpDir, expectedFiles)
		assertStatus(t, tmpDir, stdout, stderr, `M  outer/2.txt
A  outer/inner/4.txt
`)
	})
}

func setupForTestCheckOutWithChainOfCommits(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
	tmpDir, stdout, stderr = setupTestEnvironment(t)

	messages := []string{"first", "second", "third"}
	for _, message := range messages {
		writeFile(t, tmpDir, "file.txt", message)
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
		commit(t, tmpDir, message, time.Now())
	}

	brunchCmd, _ := NewBranch(tmpDir, []string{"topic"}, BranchOption{}, new(bytes.Buffer), new(bytes.Buffer))
	brunchCmd.Run()
	brunchCmd, _ = NewBranch(tmpDir, []string{"second", "@^"}, BranchOption{}, new(bytes.Buffer), new(bytes.Buffer))
	brunchCmd.Run()

	return
}

func TestCheckOutWithChainOfCommitsCheckingOutBranch(t *testing.T) {
	setup := func() (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupForTestCheckOutWithChainOfCommits(t)

		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "topic")

		return
	}

	t.Run("links HEAD to the branch", func(t *testing.T) {
		tmpDir, _, _ := setup()
		defer os.RemoveAll(tmpDir)

		r := repo(t, tmpDir)
		expected := "refs/heads/topic"

		ref, _ := r.Refs.CurrentRef("")
		if got := ref.Path; got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("resolves HEAD to the same object as the branch", func(t *testing.T) {
		tmpDir, _, _ := setup()
		defer os.RemoveAll(tmpDir)

		r := repo(t, tmpDir)
		expected, _ := r.Refs.ReadHead()

		ref, _ := r.Refs.ReadRef("topic")
		if got := ref; got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("prints a message when switching to the same branch", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		checkout(tmpDir, stdout, stderr, "topic")
		expected := `Already on 'topic'
`
		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("prints a message when switching to another branch", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		checkout(tmpDir, stdout, stderr, "second")
		expected := `Switched to branch 'second'
`
		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("prints a warning when detaching HEAD", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		rev, _ := resolveRevision(t, tmpDir, "@")
		shortOid := repo(t, tmpDir).Database.ShortOid(rev)

		checkout(tmpDir, stdout, stderr, "@")

		expected := fmt.Sprintf(`Note: checking out '@'.
You are in 'detached HEAD' state. You can look around, make experimental
changes and commit them, and you can discard any commits you make in this
state without impacting any branches by performing another checkout.
If you want to create a new branch to retain commits you create, you may
do so (now or later) by using the branch command. Example:

	jit branch <new-branch-name>
HEAD is now at %s third
`, shortOid)

		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})
}

func TestCheckOutWithChainOfCommitsCheckingOutRelativeRevision(t *testing.T) {
	setup := func() (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupForTestCheckOutWithChainOfCommits(t)

		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "topic^")

		return
	}

	t.Run("detaches HEAD", func(t *testing.T) {
		tmpDir, _, _ := setup()
		defer os.RemoveAll(tmpDir)

		r := repo(t, tmpDir)
		expected := "HEAD"

		ref, _ := r.Refs.CurrentRef("")
		if got := ref.Path; got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("puts the revision's value in HEAD", func(t *testing.T) {
		tmpDir, _, _ := setup()
		defer os.RemoveAll(tmpDir)

		r := repo(t, tmpDir)
		expected, _ := r.Refs.ReadHead()

		ref, _ := resolveRevision(t, tmpDir, "topic^")
		if got := ref; got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("prints a message when switching to the same commit", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		rev, _ := resolveRevision(t, tmpDir, "@")
		shortOid := repo(t, tmpDir).Database.ShortOid(rev)

		checkout(tmpDir, stdout, stderr, "@")

		expected := fmt.Sprintf(`HEAD is now at %s second
`, shortOid)

		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("prints a message when switching to a different commit", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		rev, _ := resolveRevision(t, tmpDir, "@")
		a := repo(t, tmpDir).Database.ShortOid(rev)
		rev, _ = resolveRevision(t, tmpDir, "@^")
		b := repo(t, tmpDir).Database.ShortOid(rev)

		checkout(tmpDir, stdout, stderr, "@^")

		expected := fmt.Sprintf(`Previous HEAD position was %s second
HEAD is now at %s first
`, a, b)

		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("prints a message when switching to a branch with the same ID", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		checkout(tmpDir, stdout, stderr, "second")

		expected := "Switched to branch 'second'\n"

		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("prints a message when switching to a different branch", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		rev, _ := resolveRevision(t, tmpDir, "@")
		shortOid := repo(t, tmpDir).Database.ShortOid(rev)

		checkout(tmpDir, stdout, stderr, "topic")

		expected := fmt.Sprintf(`Previous HEAD position was %s second
Switched to branch 'topic'
`, shortOid)

		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})
}
