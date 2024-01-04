package command

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func TestRmWithSingleFile(t *testing.T) {
	setup := func() (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		writeFile(t, tmpDir, "f.txt", "1")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
		commit(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), "first", time.Now())

		return
	}

	t.Run("exits successfully", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		rm, _ := NewRm(tmpDir, []string{"f.txt"}, RmOption{}, stdout, stderr)
		status := rm.Run()

		if status != 0 {
			t.Errorf("want %q, but got %q", 0, status)
		}
	})

	t.Run("removes a file from the index", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		rm, _ := NewRm(tmpDir, []string{"f.txt"}, RmOption{}, stdout, stderr)
		rm.Run()

		r := repo(t, tmpDir)
		r.Index.Load()
		expected := r.Index.IsTrackedFile("f.txt")

		if expected {
			t.Errorf("want %v, but got %v", false, expected)
		}
	})

	t.Run("removes a file from the workspace", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		rm, _ := NewRm(tmpDir, []string{"f.txt"}, RmOption{}, stdout, stderr)
		rm.Run()

		assertWorkspace(t, tmpDir, map[string]string{})
	})

	t.Run("succeeds if the file is not in the workspace", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		delete(t, tmpDir, "f.txt")
		rm, _ := NewRm(tmpDir, []string{"f.txt"}, RmOption{}, stdout, stderr)
		status := rm.Run()

		if status != 0 {
			t.Errorf("want %q, but got %q", 0, status)
		}

		r := repo(t, tmpDir)
		r.Index.Load()
		expected := r.Index.IsTrackedFile("f.txt")

		if expected {
			t.Errorf("want %v, but got %v", false, expected)
		}
	})

	t.Run("fails if the file is not in the index", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		rm, _ := NewRm(tmpDir, []string{"nope.txt"}, RmOption{}, stdout, stderr)
		status := rm.Run()

		expected := "fatal: pathspec 'nope.txt' did not match any files\n"
		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}

		if status != 128 {
			t.Errorf("want %q, but got %q", 128, status)
		}
	})

	t.Run("fails if the file has unstaged changes", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "f.txt", "2")
		rm, _ := NewRm(tmpDir, []string{"f.txt"}, RmOption{}, stdout, stderr)
		status := rm.Run()

		expected := `error: the following file has local modifications:
    f.txt
`
		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}

		if status == 0 {
			t.Errorf("want %q, but got %q", 1, status)
		}

		r := repo(t, tmpDir)
		r.Index.Load()

		if !r.Index.IsTrackedFile("f.txt") {
			t.Errorf("want %v, but got %v", true, expected)
		}

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "2",
		})
	})

	t.Run("fails if the file has uncommitted changes", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "f.txt", "2")
		Add(tmpDir, []string{"f.txt"}, new(bytes.Buffer), new(bytes.Buffer))
		rm, _ := NewRm(tmpDir, []string{"f.txt"}, RmOption{}, stdout, stderr)
		status := rm.Run()

		expected := `error: the following file has changes staged in the index:
    f.txt
`
		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}

		if status == 0 {
			t.Errorf("want %q, but got %q", 1, status)
		}

		r := repo(t, tmpDir)
		r.Index.Load()

		if !r.Index.IsTrackedFile("f.txt") {
			t.Errorf("want %v, but got %v", true, expected)
		}

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "2",
		})
	})
}
