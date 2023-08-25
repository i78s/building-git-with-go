package command

import (
	"building-git/lib/repository"
	"bytes"
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

var baseFiles = map[string]string{
	"1.txt":             "1",
	"outer/2.txt":       "2",
	"outer/inner/3.txt": "3",
}

func TestCheckOutWithSetOfFiles(t *testing.T) {
	setup := func() (tmpDir string, stdout, stderr *bytes.Buffer, commitAndCheckout func(revision string)) {
		tmpDir, stdout, stderr = SetupTestEnvironment(t)

		for name, content := range baseFiles {
			WriteFile(t, tmpDir, name, content)
		}

		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
		TestCommit(t, tmpDir, "first")

		commitAll := func() {
			Delete(t, tmpDir, ".git/index")
			Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
			TestCommit(t, tmpDir, "change")
		}

		commitAndCheckout = func(revision string) {
			commitAll()
			options := CheckOutOption{}
			checkout, _ := NewCheckOut(tmpDir, []string{revision}, options, stdout, stderr)
			checkout.Run()
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
}
