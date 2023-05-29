package index

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestIndexAddSingleFile(t *testing.T) {
	oidBytes := make([]byte, 20)
	_, err := rand.Read(oidBytes)
	if err != nil {
		t.Fatal(err)
	}
	oid := fmt.Sprintf("%x", oidBytes)

	tmpPath := filepath.Join("..", "tmp")
	indexPath := filepath.Join(tmpPath, "index")
	index := NewIndex(indexPath)

	stat, err := os.Stat(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}

	index.Add("alice.txt", oid, stat)

	var found bool
	for _, entry := range index.EachEntry() {
		if entry.path == "alice.txt" {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("file not found in index")
	}
}

func TestIndexReplaceFileWithDirectory(t *testing.T) {
	oidBytes := make([]byte, 20)
	_, err := rand.Read(oidBytes)
	if err != nil {
		t.Fatal(err)
	}
	oid := fmt.Sprintf("%x", oidBytes)

	tmpPath := filepath.Join("..", "tmp")
	indexPath := filepath.Join(tmpPath, "index")
	index := NewIndex(indexPath)

	stat, err := os.Stat(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}

	index.Add("alice.txt", oid, stat)
	index.Add("bob.txt", oid, stat)
	index.Add("alice.txt/nested.txt", oid, stat)

	expectedPaths := []string{"alice.txt/nested.txt", "bob.txt"}
	actualPaths := []string{}
	for _, entry := range index.EachEntry() {
		actualPaths = append(actualPaths, entry.path)
	}

	if len(actualPaths) != len(expectedPaths) {
		t.Fatalf("expected %d entries, got %d", len(expectedPaths), len(actualPaths))
	}

	for i := range expectedPaths {
		if actualPaths[i] != expectedPaths[i] {
			t.Errorf("expected entry %d to be %s, got %s", i, expectedPaths[i], actualPaths[i])
		}
	}
}

func TestIndexReplaceDirectoryWithFile(t *testing.T) {
	oidBytes := make([]byte, 20)
	_, err := rand.Read(oidBytes)
	if err != nil {
		t.Fatal(err)
	}
	oid := fmt.Sprintf("%x", oidBytes)

	tmpPath := "../tmp"
	index := NewIndex(tmpPath)

	stat, err := os.Stat(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}

	index.Add("alice.txt", oid, stat)
	index.Add("nested/bob.txt", oid, stat)
	index.Add("nested", oid, stat)

	expected := []string{"alice.txt", "nested"}
	got := pathsFromEntries(index.EachEntry())

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

func TestIndexRecursivelyReplaceDirectoryWithFile(t *testing.T) {
	oidBytes := make([]byte, 20)
	_, err := rand.Read(oidBytes)
	if err != nil {
		t.Fatal(err)
	}
	oid := fmt.Sprintf("%x", oidBytes)

	tmpPath := "../tmp"
	index := NewIndex(tmpPath)

	stat, err := os.Stat(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}

	index.Add("alice.txt", oid, stat)
	index.Add("nested/bob.txt", oid, stat)
	index.Add("nested/inner/claire.txt", oid, stat)
	index.Add("nested", oid, stat)

	expected := []string{"alice.txt", "nested"}
	got := pathsFromEntries(index.EachEntry())

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

func pathsFromEntries(entries []*Entry) []string {
	paths := []string{}
	for _, entry := range entries {
		paths = append(paths, entry.Basename())
	}
	return paths
}
