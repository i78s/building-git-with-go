package tree

import (
	"building-git/jit/entry"
	"fmt"
	"sort"
	"strings"
)

type Tree struct {
	oid     string
	entries []entry.Entry
}

func NewTree(entries []entry.Entry) *Tree {
	return &Tree{
		entries: entries,
	}
}

func (t *Tree) Type() string {
	return "tree"
}

func (t *Tree) String() string {
	sort.Slice(t.entries, func(i, j int) bool {
		return t.entries[i].Name < t.entries[j].Name
	})

	var entries []string
	for _, entry := range t.entries {
		entryString := fmt.Sprintf("100644 %s\x00%s", entry.Name, entry.Oid)
		entries = append(entries, entryString)
	}
	return strings.Join(entries, "")
}

func (t *Tree) GetOid() string {
	return t.oid
}

func (t *Tree) SetOid(oid string) {
	t.oid = oid
}
