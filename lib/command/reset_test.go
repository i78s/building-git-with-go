package command

import (
	"building-git/lib/database"
	"bytes"
	"os"
	"reflect"
	"testing"
	"time"
)

func assertIndexEntries(t *testing.T, path string, expected map[string]string) {
	r := repo(t, path)
	r.Index.Load()

	actual := map[string]string{}
	for _, e := range r.Index.EachEntry() {
		obj, _ := r.Database.Load(e.Oid())
		actual[e.Path()] = obj.String()
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("want %q, but got %q", expected, actual)
	}
}

func TestResetWithNoHeadCommit(t *testing.T) {
	setup := func() (tmpDir string, stdout, stderr *bytes.Buffer, assertUnchangedWorkspace func()) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		writeFile(t, tmpDir, "a.txt", "1")
		writeFile(t, tmpDir, "outer/b.txt", "2")
		writeFile(t, tmpDir, "outer/inner/c.txt", "3")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))

		assertUnchangedWorkspace = func() {
			assertWorkspace(t, tmpDir, map[string]string{
				"a.txt":             "1",
				"outer/b.txt":       "2",
				"outer/inner/c.txt": "3",
			})
		}

		return
	}

	t.Run("removes everything from the index", func(t *testing.T) {
		tmpDir, stdout, stderr, assertUnchangedWorkspace := setup()
		defer os.RemoveAll(tmpDir)

		reset, _ := NewReset(tmpDir, []string{}, ResetOption{}, stdout, stderr)
		reset.Run()

		assertIndexEntries(t, tmpDir, map[string]string{})
		assertUnchangedWorkspace()
	})

	t.Run("removes a single file from the index", func(t *testing.T) {
		tmpDir, stdout, stderr, assertUnchangedWorkspace := setup()
		defer os.RemoveAll(tmpDir)

		reset, _ := NewReset(tmpDir, []string{"a.txt"}, ResetOption{}, stdout, stderr)
		reset.Run()

		assertIndexEntries(t, tmpDir, map[string]string{
			"outer/b.txt":       "2",
			"outer/inner/c.txt": "3",
		})
		assertUnchangedWorkspace()
	})

	t.Run("removes a directory from the index", func(t *testing.T) {
		tmpDir, stdout, stderr, assertUnchangedWorkspace := setup()
		defer os.RemoveAll(tmpDir)

		reset, _ := NewReset(tmpDir, []string{"outer"}, ResetOption{}, stdout, stderr)
		reset.Run()

		assertIndexEntries(t, tmpDir, map[string]string{
			"a.txt": "1",
		})
		assertUnchangedWorkspace()
	})
}

func TestResetWithHeadCommit(t *testing.T) {
	setup := func() (tmpDir string, stdout, stderr *bytes.Buffer, assertUnchangedWorkspace, assertUnchangedHead func(), headOid string) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		writeFile(t, tmpDir, "a.txt", "1")
		writeFile(t, tmpDir, "outer/b.txt", "2")
		writeFile(t, tmpDir, "outer/inner/c.txt", "3")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
		commit(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), "first", time.Now())

		writeFile(t, tmpDir, "outer/b.txt", "4")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
		commit(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), "second", time.Now())

		rm, _ := NewRm(tmpDir, []string{"a.txt"}, RmOption{}, stdout, stderr)
		rm.Run()
		writeFile(t, tmpDir, "outer/d.txt", "5")
		writeFile(t, tmpDir, "outer/inner/c.txt", "6")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
		writeFile(t, tmpDir, "outer/e.txt", "7")

		assertUnchangedWorkspace = func() {
			assertWorkspace(t, tmpDir, map[string]string{
				"outer/b.txt":       "4",
				"outer/d.txt":       "5",
				"outer/e.txt":       "7",
				"outer/inner/c.txt": "6",
			})
		}

		headOid, _ = repo(t, tmpDir).Refs.ReadHead()
		assertUnchangedHead = func() {
			oid, _ := repo(t, tmpDir).Refs.ReadHead()
			if headOid != oid {
				t.Errorf("want %v, but got %v", oid, headOid)
			}
		}

		return
	}

	t.Run("restores a file removed from the index", func(t *testing.T) {
		tmpDir, stdout, stderr, assertUnchangedWorkspace, assertUnchangedHead, _ := setup()
		defer os.RemoveAll(tmpDir)

		reset, _ := NewReset(tmpDir, []string{"a.txt"}, ResetOption{}, stdout, stderr)
		reset.Run()

		assertIndexEntries(t, tmpDir, map[string]string{
			"a.txt":             "1",
			"outer/b.txt":       "4",
			"outer/d.txt":       "5",
			"outer/inner/c.txt": "6",
		})
		assertUnchangedHead()
		assertUnchangedWorkspace()
	})

	t.Run("resets a file modified in the index", func(t *testing.T) {
		tmpDir, stdout, stderr, assertUnchangedWorkspace, assertUnchangedHead, _ := setup()
		defer os.RemoveAll(tmpDir)

		reset, _ := NewReset(tmpDir, []string{"outer/inner"}, ResetOption{}, stdout, stderr)
		reset.Run()

		assertIndexEntries(t, tmpDir, map[string]string{
			"outer/b.txt":       "4",
			"outer/d.txt":       "5",
			"outer/inner/c.txt": "3",
		})
		assertUnchangedHead()
		assertUnchangedWorkspace()
	})

	t.Run("removes a file added to the index", func(t *testing.T) {
		tmpDir, stdout, stderr, assertUnchangedWorkspace, assertUnchangedHead, _ := setup()
		defer os.RemoveAll(tmpDir)

		reset, _ := NewReset(tmpDir, []string{"outer/d.txt"}, ResetOption{}, stdout, stderr)
		reset.Run()

		assertIndexEntries(t, tmpDir, map[string]string{
			"outer/b.txt":       "4",
			"outer/inner/c.txt": "6",
		})
		assertUnchangedHead()
		assertUnchangedWorkspace()
	})

	t.Run("resets a file to a specific commit", func(t *testing.T) {
		tmpDir, stdout, stderr, assertUnchangedWorkspace, assertUnchangedHead, _ := setup()
		defer os.RemoveAll(tmpDir)

		reset, _ := NewReset(tmpDir, []string{"@^", "outer/b.txt"}, ResetOption{}, stdout, stderr)
		reset.Run()

		assertIndexEntries(t, tmpDir, map[string]string{
			"outer/b.txt":       "2",
			"outer/d.txt":       "5",
			"outer/inner/c.txt": "6",
		})
		assertUnchangedHead()
		assertUnchangedWorkspace()
	})

	t.Run("resets the whole index", func(t *testing.T) {
		tmpDir, stdout, stderr, assertUnchangedWorkspace, assertUnchangedHead, _ := setup()
		defer os.RemoveAll(tmpDir)

		reset, _ := NewReset(tmpDir, []string{}, ResetOption{}, stdout, stderr)
		reset.Run()

		assertIndexEntries(t, tmpDir, map[string]string{
			"a.txt":             "1",
			"outer/b.txt":       "4",
			"outer/inner/c.txt": "3",
		})
		assertUnchangedHead()
		assertUnchangedWorkspace()
	})

	t.Run("resets the whole index and moves HEAD", func(t *testing.T) {
		tmpDir, stdout, stderr, assertUnchangedWorkspace, _, headOid := setup()
		defer os.RemoveAll(tmpDir)

		reset, _ := NewReset(tmpDir, []string{"@^"}, ResetOption{}, stdout, stderr)
		reset.Run()

		assertIndexEntries(t, tmpDir, map[string]string{
			"a.txt":             "1",
			"outer/b.txt":       "2",
			"outer/inner/c.txt": "3",
		})

		r := repo(t, tmpDir)
		obj, _ := r.Database.Load(headOid)
		p := obj.(*database.Commit).Parent()
		head, _ := r.Refs.ReadHead()
		if p != head {
			t.Errorf("want %v, but got %v", p, head)
		}

		assertUnchangedWorkspace()
	})

	t.Run("moves HEAD and leaves the index unchanged", func(t *testing.T) {
		tmpDir, stdout, stderr, assertUnchangedWorkspace, _, headOid := setup()
		defer os.RemoveAll(tmpDir)

		reset, _ := NewReset(tmpDir, []string{"@^"}, ResetOption{Mode: Soft}, stdout, stderr)
		reset.Run()

		assertIndexEntries(t, tmpDir, map[string]string{
			"outer/b.txt":       "4",
			"outer/d.txt":       "5",
			"outer/inner/c.txt": "6",
		})

		r := repo(t, tmpDir)
		obj, _ := r.Database.Load(headOid)
		p := obj.(*database.Commit).Parent()
		head, _ := r.Refs.ReadHead()
		if p != head {
			t.Errorf("want %v, but got %v", p, head)
		}

		assertUnchangedWorkspace()
	})
}
