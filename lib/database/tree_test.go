package database

import (
	"io/fs"
	"testing"
)

// Mock implementation of EntryObject for testing
type MockEntryObject struct {
	oid        string
	key        string
	mode       int
	parentDirs []string
	basename   string
}

func (m *MockEntryObject) Oid() string                        { return m.oid }
func (m *MockEntryObject) Key() string                        { return m.key }
func (m *MockEntryObject) Mode() int                          { return m.mode }
func (m *MockEntryObject) ParentDirectories() []string        { return m.parentDirs }
func (m *MockEntryObject) Basename() string                   { return m.basename }
func (m *MockEntryObject) IsStatMatch(stat fs.FileInfo) bool  { return false }
func (m *MockEntryObject) UpdateStat(stat fs.FileInfo)        {}
func (m *MockEntryObject) IsTimesMatch(stat fs.FileInfo) bool { return false }

func TestBuildTree(t *testing.T) {
	// Create mock entries

	entries := []EntryObject{
		&MockEntryObject{oid: "1", parentDirs: []string{}, basename: "1.txt"},
		&MockEntryObject{oid: "2", parentDirs: []string{"a"}, basename: "2.txt"},
		&MockEntryObject{oid: "3", parentDirs: []string{"a", "b"}, basename: "3.txt"},
	}

	// Build the tree
	tree := BuildTree(entries)

	// Verify the structure of the tree
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
