package database

import (
	"fmt"
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

func (t *TreeDiff) compareOids(a, b string, filter *PathFilter) {
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

	t.detectDeletions(aEntries, bEntries, filter)
	t.detectAdditions(aEntries, bEntries, filter)
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

func (t *TreeDiff) detectDeletions(a, b map[string]TreeObject, filter *PathFilter) {
	filter.EachEntry(a, func(name string, entry TreeObject) {
		other := b[name]

		if entry == nil && other == nil {
			return
		}
		if entry != nil && other != nil && other.Oid() == entry.Oid() && other.Mode() == entry.Mode() {
			return
		}

		subFilter := filter.Join(name)
		treeA, treeB := "", ""
		if entry != nil && entry.(*Entry).IsTree() {
			treeA = entry.Oid()
		}
		if other != nil && other.(*Entry).IsTree() {
			treeB = other.Oid()
		}
		t.compareOids(treeA, treeB, subFilter)

		blobs := [2]TreeObject{}
		for i, e := range []TreeObject{entry, other} {
			if e == nil || !e.(*Entry).IsTree() {
				blobs[i] = e
			}
		}
		if blobs[0] != nil || blobs[1] != nil {
			t.changes[subFilter.Path] = blobs
		}
	})
}

func (t *TreeDiff) detectAdditions(a, b map[string]TreeObject, filter *PathFilter) {
	filter.EachEntry(b, func(name string, entry TreeObject) {
		_, ok := a[name]
		if ok {
			return
		}

		subFilter := filter.Join(name)
		if entry.(*Entry).IsTree() {
			t.compareOids("", entry.Oid(), subFilter)
		} else {
			t.changes[subFilter.Path] = [2]TreeObject{nil, entry}
		}
	})
}
