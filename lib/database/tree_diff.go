package database

import (
	"fmt"
	"path/filepath"
)

type TreeDiff struct {
	database *Database
	changes  map[string][2]TreeObject
}

func NewTreeDiff(database *Database) *TreeDiff {
	return &TreeDiff{
		database: database,
		changes:  map[string][2]TreeObject{},
	}
}

func (t *TreeDiff) compareOids(a, b, prefix string) {
	if a == b {
		return
	}

	aEntries := map[string]TreeObject{}
	bEntries := map[string]TreeObject{}
	if a != "" {
		tree, _ := t.oidToTree(a)
		aEntries = tree.Entries
	}
	if b != "" {
		tree, _ := t.oidToTree(b)
		bEntries = tree.Entries
	}

	t.detectDeletions(aEntries, bEntries, prefix)
	t.detectAdditions(aEntries, bEntries, prefix)
}

func (t *TreeDiff) oidToTree(oid string) (*Tree, error) {
	object, err := t.database.Load(oid)

	if err != nil {
		return nil, err
	}

	switch v := object.(type) {
	case *Commit:
		tree, err := t.database.Load(v.Tree())
		return tree.(*Tree), err
	case *Tree:
		return v, nil
	}

	return nil, fmt.Errorf("")
}

func (t *TreeDiff) detectDeletions(a, b map[string]TreeObject, prefix string) {
	for name, entry := range a {
		path := filepath.Join(prefix, name)
		other := b[name]

		if entry == nil && other == nil {
			continue
		}
		if entry != nil && other != nil && other.Oid() == entry.Oid() && other.Mode() == entry.Mode() {
			continue
		}

		treeA, treeB := "", ""
		if entry != nil && entry.(*Entry).IsTree() {
			treeA = entry.Oid()
		}
		if other != nil && other.(*Entry).IsTree() {
			treeB = other.Oid()
		}
		t.compareOids(treeA, treeB, path)

		blobs := [2]TreeObject{}
		for i, e := range []TreeObject{entry, other} {
			if e == nil || !e.(*Entry).IsTree() {
				blobs[i] = e
			}
		}
		if blobs[0] != nil || blobs[1] != nil {
			t.changes[path] = blobs
		}
	}
}

func (t *TreeDiff) detectAdditions(a, b map[string]TreeObject, prefix string) {
	for name, entry := range b {
		path := filepath.Join(prefix, name)
		_, ok := a[name]
		if !ok {
			if entry.(*Entry).IsTree() {
				t.compareOids("", entry.Oid(), path)
			} else {
				t.changes[path] = [2]TreeObject{nil, entry}
			}
		}
	}
}
