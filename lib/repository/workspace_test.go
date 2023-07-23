package repository

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestListFiles(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "jit")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	workspace := NewWorkspace(tmpDir)

	err = ioutil.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile(filepath.Join(subDir, "subtest.txt"), []byte("test"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	gitDir := filepath.Join(tmpDir, ".git")
	err = os.Mkdir(gitDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile(filepath.Join(gitDir, "index"), []byte("index"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	files, err := workspace.ListFiles(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	sort.Strings(files) // Ensure order for comparison
	expected := []string{"subdir/subtest.txt", "test.txt"}
	if !reflect.DeepEqual(files, expected) {
		t.Errorf("ListFiles() = %v, want %v", files, expected)
	}
}

func TestListDir(t *testing.T) {
	// Setup a temporary workspace
	tmpDir, err := ioutil.TempDir("", "workspace")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir) // clean up

	// Create some files and directories in the temporary workspace
	structure := []string{
		"file1.txt",
		"dir1/file2.txt",
		"dir1/dir2/file3.txt",
		".git/config",
	}
	for _, path := range structure {
		fullPath := filepath.Join(tmpDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatal(err)
		}
		_, err = os.Create(fullPath)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Initialize the Workspace struct
	w := &Workspace{pathname: tmpDir}

	// Call the method we're testing
	stats, err := w.ListDir("")
	if err != nil {
		t.Fatal(err)
	}

	// Check the results
	expected := []string{
		"file1.txt",
		"dir1",
	}
	for _, path := range expected {
		if _, exists := stats[path]; !exists {
			t.Errorf("Expected %s to be listed, but it wasn't", path)
		}
	}
	for path := range stats {
		found := false
		for _, expectedPath := range expected {
			if path == expectedPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Did not expect %s to be listed, but it was", path)
		}
	}
}
