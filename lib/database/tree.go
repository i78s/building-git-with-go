package database

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

const TREE_MODE = 0o40000

type Tree struct {
	oid     string
	Entries map[string]TreeObject
}

type TreeObject interface {
	Oid() string
	Mode() int
}

type EntryObject interface {
	Oid() string
	Key() string
	Mode() int
	ParentDirectories() []string
	Basename() string
	IsStatMatch(stat fs.FileInfo) bool
	UpdateStat(stat fs.FileInfo)
	IsTimesMatch(stat fs.FileInfo) bool
}

func NewTree(entries map[string]TreeObject) *Tree {
	return &Tree{
		Entries: entries,
	}
}

func ParseTree(reader *bufio.Reader) (*Tree, error) {
	entries := make(map[string]TreeObject)

	for {
		modeStr, err := reader.ReadString(' ')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		mode, err := strconv.ParseInt(strings.TrimSpace(modeStr), 8, 32)
		if err != nil {
			return nil, err
		}

		name, err := reader.ReadString('\x00')
		if err != nil {
			return nil, err
		}
		name = strings.TrimSuffix(name, "\x00")

		oidBytes := make([]byte, 20)
		if _, err = io.ReadFull(reader, oidBytes); err != nil {
			return nil, err
		}

		oid := hex.EncodeToString(oidBytes)
		entries[name] = NewEntry(
			oid, int(mode),
		)
	}
	return NewTree(entries), nil
}

func BuildTree(entries []EntryObject) *Tree {
	root := NewTree(make(map[string]TreeObject))
	for _, e := range entries {
		root.addEntry(e.ParentDirectories(), e)
	}

	return root
}

func (t *Tree) addEntry(parents []string, e TreeObject) {
	if len(parents) == 0 {
		t.Entries[e.(EntryObject).Basename()] = e
	} else {
		subtree, exists := t.Entries[parents[0]]
		if !exists {
			subtree = NewTree(make(map[string]TreeObject))
			t.Entries[parents[0]] = subtree
		}
		subtree.(*Tree).addEntry(parents[1:], e)
	}
}

func (t *Tree) Traverse(fn func(TreeObject)) {
	for _, entry := range t.Entries {
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
	keys := make([]string, 0, len(t.Entries))
	for k := range t.Entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, name := range keys {
		entry := t.Entries[name]
		entryOidBytes, _ := hex.DecodeString(entry.Oid())

		entryString := fmt.Sprintf("%o %s", entry.Mode(), name)
		buf.WriteString(entryString)
		buf.WriteByte(0)
		buf.Write(entryOidBytes)
	}
	return buf.String()
}

func (t *Tree) Oid() string {
	return t.oid
}

func (t *Tree) SetOid(oid string) {
	t.oid = oid
}
