package database

import (
	"os"
	"path/filepath"
	"testing"
)

var testDir = filepath.Join(os.TempDir(), "test-objects")

func storeTree(contents map[string]string) string {
	db := NewDatabase(testDir)

	entries := []EntryObject{}
	for path, data := range contents {
		blob := NewBlob(data)
		db.Store(blob)

		entries = append(entries, &MockEntryObject{
			oid:  blob.oid,
			path: path,
			mode: 0o100644,
		})
	}

	tree := BuildTree(entries)
	tree.Traverse(func(t TreeObject) {
		if gitObj, ok := t.(GitObject); ok {
			db.Store(gitObj)
		}
	})

	return tree.oid
}

func treeDiff(a, b string) map[string][2]TreeObject {
	db := NewDatabase(testDir)
	return db.TreeDiff(a, b, nil)
}

func TestTreeDiff(t *testing.T) {
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	t.Run("reports a changed file", func(t *testing.T) {
		treeA := storeTree(map[string]string{
			"alice.txt": "alice",
			"bob.txt":   "bob",
		})

		treeB := storeTree(map[string]string{
			"alice.txt": "changed",
			"bob.txt":   "bob",
		})

		changes := treeDiff(treeA, treeB)

		expected := map[string][2]TreeObject{
			"alice.txt": {
				NewEntry("ca56b59dbf8c0884b1b9ceb306873b24b73de969", 0o100644),
				NewEntry("21fb1eca31e64cd3914025058b21992ab76edcf9", 0o100644),
			},
		}

		if !compareEntriesMap(changes, expected) {
			t.Errorf("expected %v, got %v", expected, changes)
		}
	})

	t.Run("reports an added file", func(t *testing.T) {
		treeA := storeTree(map[string]string{
			"alice.txt": "alice",
		})

		treeB := storeTree(map[string]string{
			"alice.txt": "alice",
			"bob.txt":   "bob",
		})

		changes := treeDiff(treeA, treeB)

		expected := map[string][2]TreeObject{
			"bob.txt": {
				nil,
				NewEntry("2529de8969e5ee206e572ed72a0389c3115ad95c", 0o100644),
			},
		}

		if !compareEntriesMap(changes, expected) {
			t.Errorf("expected %v, got %v", expected, changes)
		}
	})

	t.Run("reports a deleted file", func(t *testing.T) {
		treeA := storeTree(map[string]string{
			"alice.txt": "alice",
			"bob.txt":   "bob",
		})

		treeB := storeTree(map[string]string{
			"alice.txt": "alice",
		})

		changes := treeDiff(treeA, treeB)

		expected := map[string][2]TreeObject{
			"bob.txt": {
				NewEntry("2529de8969e5ee206e572ed72a0389c3115ad95c", 0o100644),
				nil,
			},
		}

		if !compareEntriesMap(changes, expected) {
			t.Errorf("expected %v, got %v", expected, changes)
		}
	})

	t.Run("reports an added file inside a directory", func(t *testing.T) {
		treeA := storeTree(map[string]string{
			"1.txt":       "1",
			"outer/2.txt": "2",
		})

		treeB := storeTree(map[string]string{
			"1.txt":           "1",
			"outer/2.txt":     "2",
			"outer/new/4.txt": "4",
		})

		changes := treeDiff(treeA, treeB)

		expected := map[string][2]TreeObject{
			"outer/new/4.txt": {
				nil,
				NewEntry("bf0d87ab1b2b0ec1a11a3973d2845b42413d9767", 0o100644),
			},
		}

		if !compareEntriesMap(changes, expected) {
			t.Errorf("expected %v, got %v", expected, changes)
		}
	})

	t.Run("reports a deleted file inside a directory", func(t *testing.T) {
		treeA := storeTree(map[string]string{
			"1.txt":             "1",
			"outer/2.txt":       "2",
			"outer/inner/3.txt": "3",
		})

		treeB := storeTree(map[string]string{
			"1.txt":       "1",
			"outer/2.txt": "2",
		})

		changes := treeDiff(treeA, treeB)

		expected := map[string][2]TreeObject{
			"outer/inner/3.txt": {
				NewEntry("e440e5c842586965a7fb77deda2eca68612b1f53", 0o100644),
				nil,
			},
		}

		if !compareEntriesMap(changes, expected) {
			t.Errorf("expected %v, got %v", expected, changes)
		}
	})
}

func compareEntriesMap(a, b map[string][2]TreeObject) bool {
	for key, entriesA := range a {
		entriesB, ok := b[key]
		if !ok {
			return false
		}

		if len(entriesA) != len(entriesB) {
			return false
		}

		for i := range entriesA {
			if entriesA[i] != nil && entriesB[i] != nil {
				if entriesA[i].Oid() != entriesB[i].Oid() || entriesA[i].Mode() != entriesB[i].Mode() {
					return false
				}
			} else if entriesA[i] != entriesB[i] {
				return false
			}
		}
	}

	return true
}
