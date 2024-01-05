package command

import (
	"bytes"
	"os"
	"reflect"
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
			t.Errorf("want %d, but got %d", 0, status)
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

		if status != 1 {
			t.Errorf("want %q, but got %q", 1, status)
		}

		r := repo(t, tmpDir)
		r.Index.Load()

		isTrackedFile := r.Index.IsTrackedFile("f.txt")
		if !isTrackedFile {
			t.Errorf("want %v, but got %v", true, isTrackedFile)
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

	t.Run("forces removal of unstaged changes", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "f.txt", "2")
		rm, _ := NewRm(tmpDir, []string{"f.txt"}, RmOption{Force: true}, stdout, stderr)
		rm.Run()

		r := repo(t, tmpDir)
		r.Index.Load()

		expected := r.Index.IsTrackedFile("f.txt")
		if expected {
			t.Errorf("want %v, but got %v", true, expected)
		}

		assertWorkspace(t, tmpDir, map[string]string{})
	})

	t.Run("forces removal of uncommitted changes", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "f.txt", "2")
		Add(tmpDir, []string{"f.txt"}, new(bytes.Buffer), new(bytes.Buffer))
		rm, _ := NewRm(tmpDir, []string{"f.txt"}, RmOption{Force: true}, stdout, stderr)
		rm.Run()

		r := repo(t, tmpDir)
		r.Index.Load()

		expected := r.Index.IsTrackedFile("f.txt")
		if expected {
			t.Errorf("want %v, but got %v", true, expected)
		}

		assertWorkspace(t, tmpDir, map[string]string{})
	})

	t.Run("removes a file only from the index", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		rm, _ := NewRm(tmpDir, []string{"f.txt"}, RmOption{Cached: true}, stdout, stderr)
		rm.Run()

		r := repo(t, tmpDir)
		r.Index.Load()
		expected := r.Index.IsTrackedFile("f.txt")

		if expected {
			t.Errorf("want %v, but got %v", true, expected)
		}

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "1",
		})
	})

	t.Run("removes a file from the index if it has unstaged changes", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "f.txt", "2")
		rm, _ := NewRm(tmpDir, []string{"f.txt"}, RmOption{Cached: true}, stdout, stderr)
		rm.Run()

		r := repo(t, tmpDir)
		r.Index.Load()
		expected := r.Index.IsTrackedFile("f.txt")

		if expected {
			t.Errorf("want %v, but got %v", true, expected)
		}

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "2",
		})
	})

	t.Run("removes a file from the index if it has uncommitted changes", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "f.txt", "2")
		Add(tmpDir, []string{"f.txt"}, new(bytes.Buffer), new(bytes.Buffer))
		rm, _ := NewRm(tmpDir, []string{"f.txt"}, RmOption{Cached: true}, stdout, stderr)
		rm.Run()

		r := repo(t, tmpDir)
		r.Index.Load()
		expected := r.Index.IsTrackedFile("f.txt")

		if expected {
			t.Errorf("want %v, but got %v", true, expected)
		}

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "2",
		})
	})

	t.Run("does not remove a file with both uncommitted and unstaged changes", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "f.txt", "2")
		Add(tmpDir, []string{"f.txt"}, new(bytes.Buffer), new(bytes.Buffer))
		writeFile(t, tmpDir, "f.txt", "3")
		rm, _ := NewRm(tmpDir, []string{"f.txt"}, RmOption{Cached: true}, stdout, stderr)
		status := rm.Run()

		expected := `error: the following file has staged content different from both the file and the HEAD:
    f.txt
`
		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}

		if status != 1 {
			t.Errorf("want %q, but got %q", 1, status)
		}

		r := repo(t, tmpDir)
		r.Index.Load()

		isTrackedFile := r.Index.IsTrackedFile("f.txt")
		if !isTrackedFile {
			t.Errorf("want %v, but got %v", true, isTrackedFile)
		}

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "3",
		})
	})
}

func TestRmWithTree(t *testing.T) {
	setup := func() (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		writeFile(t, tmpDir, "f.txt", "1")
		writeFile(t, tmpDir, "outer/g.txt", "2")
		writeFile(t, tmpDir, "outer/inner/h.txt", "3")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
		commit(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), "first", time.Now())

		return
	}

	t.Run("removes multiple files", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		rm, _ := NewRm(tmpDir, []string{"f.txt", "outer/inner/h.txt"}, RmOption{}, stdout, stderr)
		rm.Run()

		r := repo(t, tmpDir)
		r.Index.Load()

		expandPaths := []string{"outer/g.txt"}
		actualPaths := []string{}
		for _, e := range r.Index.EachEntry() {
			actualPaths = append(actualPaths, e.Path())
		}

		if !reflect.DeepEqual(actualPaths, expandPaths) {
			t.Errorf("expected %v, got %v", expandPaths, actualPaths)
		}
	})

	t.Run("refuses to remove a directory", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		rm, _ := NewRm(tmpDir, []string{"f.txt", "outer"}, RmOption{}, stdout, stderr)
		status := rm.Run()

		expected := "fatal: not removing 'outer' recursively without -r\n"
		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}

		if status != 128 {
			t.Errorf("want %q, but got %q", 128, status)
		}

		r := repo(t, tmpDir)
		r.Index.Load()

		expandPaths := []string{"f.txt", "outer/g.txt", "outer/inner/h.txt"}
		actualPaths := []string{}
		for _, e := range r.Index.EachEntry() {
			actualPaths = append(actualPaths, e.Path())
		}

		if !reflect.DeepEqual(actualPaths, expandPaths) {
			t.Errorf("expected %v, got %v", expandPaths, actualPaths)
		}

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt":             "1",
			"outer/g.txt":       "2",
			"outer/inner/h.txt": "3",
		})
	})

	t.Run("does not remove a file replaced with a directory", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		delete(t, tmpDir, "f.txt")
		writeFile(t, tmpDir, "f.txt/nested", "keep me")

		rm, _ := NewRm(tmpDir, []string{"f.txt"}, RmOption{}, stdout, stderr)
		status := rm.Run()

		expected := "fatal: jit rm: 'f.txt': Operation not permitted\n"
		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}

		if status != 128 {
			t.Errorf("want %q, but got %q", 128, status)
		}

		r := repo(t, tmpDir)
		r.Index.Load()

		expandPaths := []string{"f.txt", "outer/g.txt", "outer/inner/h.txt"}
		actualPaths := []string{}
		for _, e := range r.Index.EachEntry() {
			actualPaths = append(actualPaths, e.Path())
		}

		if !reflect.DeepEqual(actualPaths, expandPaths) {
			t.Errorf("expected %v, got %v", expandPaths, actualPaths)
		}

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt/nested":      "keep me",
			"outer/g.txt":       "2",
			"outer/inner/h.txt": "3",
		})
	})

	t.Run("removes a directory with -r", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		rm, _ := NewRm(tmpDir, []string{"outer"}, RmOption{Recursive: true}, stdout, stderr)
		rm.Run()

		r := repo(t, tmpDir)
		r.Index.Load()

		expandPaths := []string{"f.txt"}
		actualPaths := []string{}
		for _, e := range r.Index.EachEntry() {
			actualPaths = append(actualPaths, e.Path())
		}

		if !reflect.DeepEqual(actualPaths, expandPaths) {
			t.Errorf("expected %v, got %v", expandPaths, actualPaths)
		}

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "1",
		})
	})

	t.Run("does not remove untracked files", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "outer/inner/j.txt", "4")
		rm, _ := NewRm(tmpDir, []string{"outer"}, RmOption{Recursive: true}, stdout, stderr)
		rm.Run()

		r := repo(t, tmpDir)
		r.Index.Load()

		expandPaths := []string{"f.txt"}
		actualPaths := []string{}
		for _, e := range r.Index.EachEntry() {
			actualPaths = append(actualPaths, e.Path())
		}

		if !reflect.DeepEqual(actualPaths, expandPaths) {
			t.Errorf("expected %v, got %v", expandPaths, actualPaths)
		}

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt":             "1",
			"outer/inner/j.txt": "4",
		})
	})
}
