package index

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
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
