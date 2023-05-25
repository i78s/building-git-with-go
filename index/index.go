package index

import (
	jit "building-git"
	"crypto/sha1"
	"encoding/binary"
	"hash"
	"io/fs"
	"sort"
)

const (
	HEADER_FORMAT    = "a4N2"
	ENTRY_BLOCK_SIZE = 8
)

type Index struct {
	entries  map[string]*Entry
	keys     []string
	lockfile *jit.Lockfile
	digest   hash.Hash
}

func NewIndex(pathname string) *Index {
	return &Index{
		entries:  make(map[string]*Entry),
		keys:     make([]string, 0),
		lockfile: jit.NewLockfile(pathname),
	}
}

func (i *Index) Add(pathname, oid string, stat fs.FileInfo) {
	if _, exists := i.entries[pathname]; !exists {
		entry := CreateEntry(pathname, oid, stat)
		i.entries[entry.Key()] = entry

		index := sort.SearchStrings(i.keys, pathname)
		i.keys = append(i.keys, "")
		copy(i.keys[index+1:], i.keys[index:])
		i.keys[index] = pathname
	}
}

func (i *Index) WriteUpdates() bool {
	if _, err := i.lockfile.HoldForUpdate(); err != nil {
		return false
	}

	i.beginWrite()

	header := make([]byte, 12)
	copy(header[0:4], "DIRC")
	binary.BigEndian.PutUint32(header[4:8], 2)
	binary.BigEndian.PutUint32(header[8:12], uint32(len(i.entries)))
	i.write(header)

	for _, key := range i.keys {
		i.write([]byte(i.entries[key].String()))
	}

	i.finishWrite()

	return true
}

func (i *Index) beginWrite() {
	i.digest = sha1.New()
}

func (i *Index) write(data []byte) {
	i.lockfile.Write(data)
	i.digest.Write(data)
}

func (i *Index) finishWrite() {
	i.lockfile.Write(i.digest.Sum(nil))
	i.lockfile.Commit()
}
