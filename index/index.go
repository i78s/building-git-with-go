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
	lockfile *jit.Lockfile
	digest   hash.Hash
}

func NewIndex(pathname string) *Index {
	return &Index{
		entries:  make(map[string]*Entry),
		lockfile: jit.NewLockfile(pathname),
	}
}

func (i *Index) Add(pathname, oid string, stat fs.FileInfo) {
	entry := CreateEntry(pathname, oid, stat)
	i.entries[pathname] = entry
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

	var paths []string
	for path := range i.entries {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		i.write([]byte(i.entries[path].String()))
	}

	i.finishWrite()

	return true
}

func (i *Index) beginWrite() {
	i.digest = sha1.New()
}

func (i *Index) write(data []byte) {
	i.lockfile.Write(string(data))
	i.digest.Write(data)
}

func (i *Index) finishWrite() {
	i.lockfile.Write(string(i.digest.Sum(nil)))
	i.lockfile.Commit()
}
