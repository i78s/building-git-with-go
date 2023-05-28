package database

import (
	"building-git/index"
	"bytes"
	"encoding/hex"
	"fmt"
	"sort"
)

const TREE_MODE = 040000

type Tree struct {
	oid     string
	entries map[string]TreeObject
}

type TreeObject interface {
	GetOid() string
	Mode() int
}

func NewTree() *Tree {
	return &Tree{
		entries: make(map[string]TreeObject),
	}
}

func BuildTree(entries []*index.Entry) *Tree {
	root := NewTree()
	for _, e := range entries {
		root.addEntry(e.ParentDirectories(), e)
	}

	return root
}

func (t *Tree) addEntry(parents []string, e TreeObject) {
	if len(parents) == 0 {
		t.entries[e.(*index.Entry).Basename()] = e
	} else {
		subtree, exists := t.entries[parents[0]]
		if !exists {
			subtree = NewTree()
			t.entries[parents[0]] = subtree
		}
		subtree.(*Tree).addEntry(parents[1:], e)
	}
}

func (t *Tree) Traverse(fn func(TreeObject)) {
	for _, entry := range t.entries {
		if tree, ok := entry.(*Tree); ok {
			tree.Traverse(fn)
		}
	}
	fn(t)
}

func (t *Tree) Mode() int {
	return TREE_MODE
}

func (t *Tree) Type() string {
	return "tree"
}

func (t *Tree) String() string {
	var buf bytes.Buffer
	keys := make([]string, 0, len(t.entries))
	for k := range t.entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, name := range keys {
		entry := t.entries[name]
		entryOidBytes, _ := hex.DecodeString(entry.GetOid())

		entryString := fmt.Sprintf("%o %s", entry.Mode(), name)
		buf.WriteString(entryString)
		buf.WriteByte(0)
		buf.Write(entryOidBytes)
	}
	return buf.String()
}

func (t *Tree) GetOid() string {
	return t.oid
}

func (t *Tree) SetOid(oid string) {
	t.oid = oid
}
