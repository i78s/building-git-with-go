package database

import (
	"io/fs"
	"path/filepath"
	"testing"
)

type MockEntryObject struct {
	oid  string
	path string
	mode int
}

func (m *MockEntryObject) Oid() string { return m.oid }
func (m *MockEntryObject) Key() string { return m.path }
func (m *MockEntryObject) Mode() int   { return m.mode }
func (m *MockEntryObject) ParentDirectories() []string {
	var dirs []string
	path := m.path
	for {
		path = filepath.Dir(path)
		if path == "." || path == string(filepath.Separator) {
			break
		}
		dirs = append([]string{path}, dirs...)
	}
	return dirs
}
func (m *MockEntryObject) Basename() string                   { return filepath.Base(m.path) }
func (m *MockEntryObject) IsStatMatch(stat fs.FileInfo) bool  { return false }
func (m *MockEntryObject) UpdateStat(stat fs.FileInfo)        {}
func (m *MockEntryObject) IsTimesMatch(stat fs.FileInfo) bool { return false }
func (m *MockEntryObject) IsNil() bool                        { return m == nil }

func TestBuildTree(t *testing.T) {
	entries := []EntryObject{
		&MockEntryObject{oid: "1", path: "1.txt"},
		&MockEntryObject{oid: "2", path: "a/2.txt"},
		&MockEntryObject{oid: "3", path: "a/b/3.txt"},
	}

	tree := BuildTree(entries)

	if _, ok := tree.Entries["1.txt"]; !ok {
		t.Errorf("Expected to find '1.txt' in tree entries")
	}

	if subtree, ok := tree.Entries["a"]; ok {
		dir1 := subtree.(*Tree)

		if _, ok := dir1.Entries["b"]; !ok {
			t.Errorf("Expected to find 'file2' in dir1 entries")
		}

		if subsubtree, ok := dir1.Entries["b"]; ok {
			dir2 := subsubtree.(*Tree)

			if _, ok := dir2.Entries["3.txt"]; !ok {
				t.Errorf("Expected to find '3.txt' in dir2 entries")
			}
		} else {
			t.Errorf("Expected to find 'dir2' in dir1 entries")
		}
	} else {
		t.Errorf("Expected to find 'dir1' in tree entries")
	}
}
