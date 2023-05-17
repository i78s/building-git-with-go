package tree

import (
	"building-git/jit/entry"
	"bytes"
	"encoding/hex"
	"fmt"
	"sort"
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

	var buf bytes.Buffer
	for _, entry := range t.entries {
		entryOidBytes, _ := hex.DecodeString(entry.Oid)
		entryString := fmt.Sprintf("100644 %s", entry.Name)
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
