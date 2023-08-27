package command

import (
	"building-git/lib/repository"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

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
		Delete(t, tmpDir, ".git/index")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
		TestCommit(t, tmpDir, "change")
	}
	checkout := func(tmpDir string, stdout, stderr *bytes.Buffer, revision string) {
		options := CheckOutOption{}
		checkout, _ := NewCheckOut(tmpDir, []string{revision}, options, stdout, stderr)
		checkout.Run()
	}

	setup := func() (tmpDir string, stdout, stderr *bytes.Buffer, commitAndCheckout func(revision string)) {
		tmpDir, stdout, stderr = SetupTestEnvironment(t)

		for name, content := range baseFiles {
			WriteFile(t, tmpDir, name, content)
		}

		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
		TestCommit(t, tmpDir, "first")

		commitAndCheckout = func(revision string) {
			commitAll(t, tmpDir)
			checkout(tmpDir, stdout, stderr, revision)
		}

		return
	}

	t.Run("updates a changed file", func(t *testing.T) {
		tmpDir, stdout, stderr, commitAndCheckout := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "1.txt", "changed")
		commitAndCheckout("@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("fails to update a modified file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "1.txt", "changed")
		commitAll(t, tmpDir)

		WriteFile(t, tmpDir, "1.txt", "conflict")
		checkout(tmpDir, stdout, stderr, "@^")

		assertStaleFile(t, stderr, "1.txt")
	})

	t.Run("fails to update a modified-equal file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "1.txt", "changed")
		commitAll(t, tmpDir)

		WriteFile(t, tmpDir, "1.txt", "1")
		checkout(tmpDir, stdout, stderr, "@^")

		assertStaleFile(t, stderr, "1.txt")
	})

	t.Run("fails to update a changed-mode file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "1.txt", "changed")
		commitAll(t, tmpDir)

		MakeExecutable(t, tmpDir, "1.txt")
		checkout(tmpDir, stdout, stderr, "@^")

		assertStaleFile(t, stderr, "1.txt")
	})

	t.Run("restores a deleted file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "1.txt", "changed")
		commitAll(t, tmpDir)

		Delete(t, tmpDir, "1.txt")
		checkout(tmpDir, stdout, stderr, "@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("restores files from a deleted directory", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "outer/inner/3.txt", "changed")
		commitAll(t, tmpDir)

		Delete(t, tmpDir, "outer")
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

		WriteFile(t, tmpDir, "1.txt", "changed")
		commitAll(t, tmpDir)

		WriteFile(t, tmpDir, "1.txt", "conflict")
		Add(tmpDir, []string{"."}, stdout, stderr)

		checkout(tmpDir, stdout, stderr, "@^")
		assertStaleFile(t, stderr, "1.txt")
	})

	t.Run("updates a staged-equal file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "1.txt", "changed")
		commitAll(t, tmpDir)

		WriteFile(t, tmpDir, "1.txt", "1")
		Add(tmpDir, []string{"."}, stdout, stderr)
		checkout(tmpDir, stdout, stderr, "@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("fails to update a staged changed-mode file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "1.txt", "changed")
		commitAll(t, tmpDir)

		MakeExecutable(t, tmpDir, "1.txt")
		Add(tmpDir, []string{"."}, stdout, stderr)

		checkout(tmpDir, stdout, stderr, "@^")
		assertStaleFile(t, stderr, "1.txt")
	})

	t.Run("fails to update an unindexed file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "1.txt", "changed")
		commitAll(t, tmpDir)

		Delete(t, tmpDir, "1.txt")
		Delete(t, tmpDir, ".git/index")
		Add(tmpDir, []string{"."}, stdout, stderr)

		checkout(tmpDir, stdout, stderr, "@^")
		assertStaleFile(t, stderr, "1.txt")
	})

	t.Run("fails to update an unindexed and untracked file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "1.txt", "changed")
		commitAll(t, tmpDir)

		Delete(t, tmpDir, "1.txt")
		Delete(t, tmpDir, ".git/index")
		Add(tmpDir, []string{"."}, stdout, stderr)
		WriteFile(t, tmpDir, "1.txt", "conflict")

		checkout(tmpDir, stdout, stderr, "@^")
		assertStaleFile(t, stderr, "1.txt")
	})

	t.Run("fails to update an unindexed directory", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "outer/inner/3.txt", "changed")
		commitAll(t, tmpDir)

		Delete(t, tmpDir, "outer/inner")
		Delete(t, tmpDir, ".git/index")
		Add(tmpDir, []string{"."}, stdout, stderr)

		checkout(tmpDir, stdout, stderr, "@^")
		assertStaleFile(t, stderr, "outer/inner/3.txt")
	})

	t.Run("fails to update with a file at a parent path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "outer/inner/3.txt", "changed")
		commitAll(t, tmpDir)

		Delete(t, tmpDir, "outer/inner")
		WriteFile(t, tmpDir, "outer/inner", "conflict")

		checkout(tmpDir, stdout, stderr, "@^")
		assertStaleFile(t, stderr, "outer/inner/3.txt")
	})

	t.Run("fails to update with a staged file at a parent path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "outer/inner/3.txt", "changed")
		commitAll(t, tmpDir)

		Delete(t, tmpDir, "outer/inner")
		WriteFile(t, tmpDir, "outer/inner", "conflict")
		Add(tmpDir, []string{"."}, stdout, stderr)

		checkout(tmpDir, stdout, stderr, "@^")
		assertStaleFile(t, stderr, "outer/inner/3.txt")
	})

	t.Run("fails to update with an unstaged file at a parent path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "outer/inner/3.txt", "changed")
		commitAll(t, tmpDir)

		Delete(t, tmpDir, "outer/inner")
		Delete(t, tmpDir, ".git/index")
		Add(tmpDir, []string{"."}, stdout, stderr)
		WriteFile(t, tmpDir, "outer/inner", "conflict")

		checkout(tmpDir, stdout, stderr, "@^")
		assertStaleFile(t, stderr, "outer/inner/3.txt")
	})

	t.Run("fails to update with a file at a child path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "outer/2.txt", "changed")
		commitAll(t, tmpDir)

		Delete(t, tmpDir, "outer/2.txt")
		WriteFile(t, tmpDir, "outer/2.txt/extra.log", "conflict")

		checkout(tmpDir, stdout, stderr, "@^")
		assertStaleFile(t, stderr, "outer/2.txt")
	})

	t.Run("fails to update with a staged file at a child path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "outer/2.txt", "changed")
		commitAll(t, tmpDir)

		Delete(t, tmpDir, "outer/2.txt")
		WriteFile(t, tmpDir, "outer/2.txt/extra.log", "conflict")
		Add(tmpDir, []string{"."}, stdout, stderr)

		checkout(tmpDir, stdout, stderr, "@^")
		assertStaleFile(t, stderr, "outer/2.txt")
	})

	t.Run("removes a file", func(t *testing.T) {
		tmpDir, stdout, stderr, commitAndCheckout := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "94.txt", "94")
		commitAndCheckout("@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("removes a file from an existing directory", func(t *testing.T) {
		tmpDir, stdout, stderr, commitAndCheckout := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "outer/94.txt", "94")
		commitAndCheckout("@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("removes a file from a new directory", func(t *testing.T) {
		tmpDir, stdout, stderr, commitAndCheckout := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "new/94.txt", "94")
		commitAndCheckout("@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertNoent(t, tmpDir, "new")
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("removes a file from a new nested directory", func(t *testing.T) {
		tmpDir, stdout, stderr, commitAndCheckout := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "new/inner/94.txt", "94")
		commitAndCheckout("@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertNoent(t, tmpDir, "new")
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("removes a file from a non-empty directory", func(t *testing.T) {
		tmpDir, stdout, stderr, commitAndCheckout := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "outer/94.txt", "94")
		commitAndCheckout("@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("fails to remove a modified file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "outer/94.txt", "94")
		commitAll(t, tmpDir)

		WriteFile(t, tmpDir, "outer/94.txt", "conflict")
		checkout(tmpDir, stdout, stderr, "@^")

		assertStaleFile(t, stderr, "outer/94.txt")
	})

	t.Run("fails to remove a changed-mode file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "outer/94.txt", "94")
		commitAll(t, tmpDir)

		MakeExecutable(t, tmpDir, "outer/94.txt")
		checkout(tmpDir, stdout, stderr, "@^")

		assertStaleFile(t, stderr, "outer/94.txt")
	})

	t.Run("leaves a deleted file deleted", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "outer/94.txt", "94")
		commitAll(t, tmpDir)

		Delete(t, tmpDir, "outer/94.txt")
		checkout(tmpDir, stdout, stderr, "@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("leaves a deleted directory deleted", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "outer/inner/94.txt", "94")
		commitAll(t, tmpDir)

		Delete(t, tmpDir, "outer/inner")
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

		WriteFile(t, tmpDir, "outer/94.txt", "94")
		commitAll(t, tmpDir)

		WriteFile(t, tmpDir, "outer/94.txt", "conflict")
		Add(tmpDir, []string{"."}, stdout, stderr)
		checkout(tmpDir, stdout, stderr, "@^")

		assertStaleFile(t, stderr, "outer/94.txt")
	})

	t.Run("fails to remove a staged changed-mode file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "outer/94.txt", "94")
		commitAll(t, tmpDir)

		MakeExecutable(t, tmpDir, "outer/94.txt")
		Add(tmpDir, []string{"."}, stdout, stderr)
		checkout(tmpDir, stdout, stderr, "@^")

		assertStaleFile(t, stderr, "outer/94.txt")
	})

	t.Run("leaves an unindexed file deleted", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "outer/94.txt", "94")
		commitAll(t, tmpDir)

		Delete(t, tmpDir, "outer/94.txt")
		Delete(t, tmpDir, ".git/index")
		Add(tmpDir, []string{"."}, stdout, stderr)
		checkout(tmpDir, stdout, stderr, "@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("fails to remove an unindexed and untracked file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "outer/94.txt", "94")
		commitAll(t, tmpDir)

		Delete(t, tmpDir, "outer/94.txt")
		Delete(t, tmpDir, ".git/index")
		Add(tmpDir, []string{"."}, stdout, stderr)
		WriteFile(t, tmpDir, "outer/94.txt", "conflict")
		checkout(tmpDir, stdout, stderr, "@^")

		assertRemoveConflict(t, stderr, "outer/94.txt")
	})

	t.Run("leaves an unindexed directory deleted", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "outer/inner/94.txt", "94")
		commitAll(t, tmpDir)

		Delete(t, tmpDir, "outer/inner")
		Delete(t, tmpDir, ".git/index")
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

		WriteFile(t, tmpDir, "outer/inner/94.txt", "94")
		commitAll(t, tmpDir)

		Delete(t, tmpDir, "outer/inner")
		WriteFile(t, tmpDir, "outer/inner", "conflict")
		checkout(tmpDir, stdout, stderr, "@^")

		assertStaleFile(t, stderr, "outer/inner/94.txt")
	})

	t.Run("removes a file with a staged file at a parent path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "outer/inner/94.txt", "94")
		commitAll(t, tmpDir)

		Delete(t, tmpDir, "outer/inner")
		WriteFile(t, tmpDir, "outer/inner", "conflict")
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

		WriteFile(t, tmpDir, "outer/inner/94.txt", "94")
		commitAll(t, tmpDir)

		Delete(t, tmpDir, "outer/inner")
		Delete(t, tmpDir, ".git/index")
		Add(tmpDir, []string{"."}, stdout, stderr)
		WriteFile(t, tmpDir, "outer/inner", "conflict")
		checkout(tmpDir, stdout, stderr, "@^")

		assertRemoveConflict(t, stderr, "outer/inner")
	})

	t.Run("fails to remove with a file at a child path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "outer/94.txt", "94")
		commitAll(t, tmpDir)

		Delete(t, tmpDir, "outer/94.txt")
		WriteFile(t, tmpDir, "outer/94.txt/extra.log", "conflict")
		checkout(tmpDir, stdout, stderr, "@^")

		assertStaleFile(t, stderr, "outer/94.txt")
	})

	t.Run("removes a file with a staged file at a child path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "outer/94.txt", "94")
		commitAll(t, tmpDir)

		Delete(t, tmpDir, "outer/94.txt")
		WriteFile(t, tmpDir, "outer/94.txt/extra.log", "conflict")
		Add(tmpDir, []string{"."}, stdout, stderr)
		checkout(tmpDir, stdout, stderr, "@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("add a file", func(t *testing.T) {
		tmpDir, stdout, stderr, commitAndCheckout := setup()
		defer os.RemoveAll(tmpDir)

		Delete(t, tmpDir, "1.txt")
		commitAndCheckout("@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("add a file to a directory", func(t *testing.T) {
		tmpDir, stdout, stderr, commitAndCheckout := setup()
		defer os.RemoveAll(tmpDir)

		Delete(t, tmpDir, "outer/2.txt")
		commitAndCheckout("@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("add a directory", func(t *testing.T) {
		tmpDir, stdout, stderr, commitAndCheckout := setup()
		defer os.RemoveAll(tmpDir)

		Delete(t, tmpDir, "outer")
		commitAndCheckout("@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("fails to add an untracked file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		Delete(t, tmpDir, "outer/2.txt")
		commitAll(t, tmpDir)

		WriteFile(t, tmpDir, "outer/2.txt", "conflict")
		checkout(tmpDir, stdout, stderr, "@^")

		assertOverwriteConflict(t, stderr, "outer/2.txt")
	})

	t.Run("fails to add an added file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		Delete(t, tmpDir, "outer/2.txt")
		commitAll(t, tmpDir)

		WriteFile(t, tmpDir, "outer/2.txt", "conflict")
		Add(tmpDir, []string{"."}, stdout, stderr)
		checkout(tmpDir, stdout, stderr, "@^")

		assertStaleFile(t, stderr, "outer/2.txt")
	})

	t.Run("adds a staged-equal file", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		Delete(t, tmpDir, "outer/2.txt")
		commitAll(t, tmpDir)

		WriteFile(t, tmpDir, "outer/2.txt", "2")
		Add(tmpDir, []string{"."}, stdout, stderr)
		checkout(tmpDir, stdout, stderr, "@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("fails to add with an untracked file at a parent path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		Delete(t, tmpDir, "outer/inner/3.txt")
		commitAll(t, tmpDir)

		Delete(t, tmpDir, "outer/inner")
		WriteFile(t, tmpDir, "outer/inner", "conflict")
		checkout(tmpDir, stdout, stderr, "@^")

		assertOverwriteConflict(t, stderr, "outer/inner")
	})

	t.Run("adds a file with an added file at a parent path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		Delete(t, tmpDir, "outer/inner/3.txt")
		commitAll(t, tmpDir)

		Delete(t, tmpDir, "outer/inner")
		WriteFile(t, tmpDir, "outer/inner", "conflict")
		Add(tmpDir, []string{"."}, stdout, stderr)
		checkout(tmpDir, stdout, stderr, "@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	// NG
	t.Run("fails to add with an untracked file at a child path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		Delete(t, tmpDir, "outer/2.txt")
		commitAll(t, tmpDir)

		WriteFile(t, tmpDir, "outer/2.txt/extra.log", "conflict")
		checkout(tmpDir, stdout, stderr, "@^")

		assertStaleDirectory(t, stderr, "outer/2.txt")
	})

	t.Run("adds a file with an added file at a child path", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		Delete(t, tmpDir, "outer/2.txt")
		commitAll(t, tmpDir)

		WriteFile(t, tmpDir, "outer/2.txt/extra.log", "conflict")
		Add(tmpDir, []string{"."}, stdout, stderr)
		checkout(tmpDir, stdout, stderr, "@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("replaces a file with a directory", func(t *testing.T) {
		tmpDir, stdout, stderr, commitAndCheckout := setup()
		defer os.RemoveAll(tmpDir)

		Delete(t, tmpDir, "outer/inner")
		WriteFile(t, tmpDir, "outer/inner", "in")
		commitAndCheckout("@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("replaces a directory with a file", func(t *testing.T) {
		tmpDir, stdout, stderr, commitAndCheckout := setup()
		defer os.RemoveAll(tmpDir)

		Delete(t, tmpDir, "outer/2.txt")
		WriteFile(t, tmpDir, "outer/2.txt/nested.log", "nested")
		commitAndCheckout("@^")

		assertWorkspace(t, tmpDir, baseFiles)
		assertStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("maintains workspace modifications", func(t *testing.T) {
		tmpDir, stdout, stderr, _ := setup()
		defer os.RemoveAll(tmpDir)

		WriteFile(t, tmpDir, "1.txt", "changed")
		commitAll(t, tmpDir)

		WriteFile(t, tmpDir, "outer/2.txt", "hello")
		Delete(t, tmpDir, "outer/inner")
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

		WriteFile(t, tmpDir, "1.txt", "changed")
		commitAll(t, tmpDir)

		WriteFile(t, tmpDir, "outer/2.txt", "hello")
		WriteFile(t, tmpDir, "outer/inner/4.txt", "world")
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
