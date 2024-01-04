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
}
