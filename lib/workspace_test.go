package lib

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
