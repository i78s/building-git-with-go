package database

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestIndexAdd(t *testing.T) {
	testCases := []struct {
		Description string
		FilesToAdd  []string
		Expected    []string
	}{
		{
			Description: "AddSingleFile",
			FilesToAdd:  []string{"alice.txt"},
			Expected:    []string{"alice.txt"},
		},
		{
			Description: "ReplaceFileWithDirectory",
			FilesToAdd:  []string{"alice.txt", "bob.txt", "alice.txt/nested.txt"},
			Expected:    []string{"alice.txt/nested.txt", "bob.txt"},
		},
		{
			Description: "Replace directory with file",
			FilesToAdd:  []string{"alice.txt", "nested/bob.txt", "nested"},
			Expected:    []string{"alice.txt", "nested"},
		},
		{
			Description: "Recursively replace directory with file",
			FilesToAdd:  []string{"alice.txt", "nested/bob.txt", "nested/inner/claire.txt", "nested"},
			Expected:    []string{"alice.txt", "nested"},
		},
	}

	stat, err := os.Stat(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}

	oid := dummyOid()
	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {
			tmpDir, index, _ := setupTestEnvironment(t, []TestFile{})
			defer os.RemoveAll(tmpDir)

			for _, filename := range tc.FilesToAdd {
				index.Add(filename, oid, stat)
			}

			actual := make([]string, 0)
			for _, entry := range index.EachEntry() {
				actual = append(actual, entry.Key())
			}
			sort.Strings(actual) // assuming that order is not important

			if !reflect.DeepEqual(actual, tc.Expected) {
				t.Errorf("expected %v, got %v", tc.Expected, actual)
			}
		})
	}
}

func TestIndexWriteUpdates(t *testing.T) {
	testFiles := []TestFile{
		{Name: "test.txt", Content: "Test content"},
		{Name: "foo/bar.txt", Content: "Bar content"},
		{Name: "baz/boo/zoo.txt", Content: "Zoo content"},
	}
	tmpDir, index, workDir := setupTestEnvironment(t, testFiles)
	defer os.RemoveAll(tmpDir)

	err := index.LoadForUpdate()
	if err != nil {
		t.Fatalf("Failed to load for update: %s", err)
	}

	oid := dummyOid()
	for _, testFile := range testFiles {
		stat, err := os.Stat(filepath.Join(workDir, testFile.Name))
		if err != nil {
			t.Fatal(err)
		}

		index.Add(testFile.Name, oid, stat)
	}

	index.WriteUpdates()
	err = index.LoadForUpdate()
	if err != nil {
		t.Fatalf("Failed to load for update: %s", err)
	}
}

type TestFile struct {
	Name    string
	Content string
}

func setupTestEnvironment(t *testing.T, testFiles []TestFile) (string, *Index, string) {
	tmpDir, err := ioutil.TempDir("", "jit")
	if err != nil {
		t.Fatal(err)
	}

	gitDir := filepath.Join(tmpDir, ".git")
	err = os.Mkdir(gitDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(gitDir, "index")
	index := NewIndex(indexPath)

	workDir := filepath.Join(tmpDir, "tmp")
	err = os.Mkdir(workDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	for _, testFile := range testFiles {
		filePath := filepath.Join(workDir, testFile.Name)
		dirPath := filepath.Dir(filePath)

		err = os.MkdirAll(dirPath, 0755)
		if err != nil {
			t.Fatal(err)
		}

		err = ioutil.WriteFile(filePath, []byte(testFile.Content), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	return tmpDir, index, workDir
}

func dummyOid() string {
	oidBytes := make([]byte, 20)
	rand.Read(oidBytes)
	return fmt.Sprintf("%x", oidBytes)
}
