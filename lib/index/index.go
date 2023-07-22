package index

import (
	"building-git/lib"
	"building-git/lib/database"
	"encoding/binary"
	"fmt"
	"hash"
	"io/fs"
	"os"
	"sort"
)

const (
	HEADER_SIZE   = 12
	HEADER_FORMAT = "a4N2"
	SIGNATURE     = "DIRC"
	VERSION       = 2
)

type Index struct {
	pathname string
	entries  map[string]*Entry
	keys     []string
	parents  map[string]map[string]struct{}
	lockfile *lib.Lockfile
	digest   hash.Hash
	changed  bool
}

func NewIndex(pathname string) *Index {
	return &Index{
		pathname: pathname,
		entries:  make(map[string]*Entry),
		keys:     make([]string, 0),
		parents:  make(map[string]map[string]struct{}),
		lockfile: lib.NewLockfile(pathname),
	}
}

func (i *Index) LoadForUpdate() error {
	err := i.lockfile.HoldForUpdate()
	i.Load()
	return err
}

func (i *Index) Load() {
	i.clear()
	file, err := i.openIndexFile()

	defer file.Close()

	if err != nil {
		return
	}

	reader := NewChecksum(*file)
	count, err := i.readHeader(reader)
	if err != nil {
		return
	}
	i.readEntries(reader, count)
	reader.VerifyChecksum()
}

func (i *Index) WriteUpdates() {
	if !i.changed {
		i.lockfile.Rollback()
		return
	}

	writer := NewChecksum(*i.lockfile.Lock)

	header := make([]byte, 12)
	copy(header[0:4], SIGNATURE)
	binary.BigEndian.PutUint32(header[4:8], VERSION)
	binary.BigEndian.PutUint32(header[8:12], uint32(len(i.entries)))
	writer.Write(header)

	for _, key := range i.keys {
		writer.Write([]byte(i.entries[key].String()))
	}
	writer.WriteChecksum()
	i.lockfile.Commit()

	i.changed = false
}

func (i *Index) ReleaseLock() {
	i.lockfile.Rollback()
}

func (i *Index) Add(pathname, oid string, stat fs.FileInfo) {
	entry := CreateEntry(pathname, oid, stat)
	i.discardConflicts(entry)
	i.storeEntry(entry)
	i.changed = true
}

func (i *Index) EachEntry() []database.EntryObject {
	entries := []database.EntryObject{}
	for _, key := range i.keys {
		entries = append(entries, i.entries[key])
	}
	return entries
}

func (i *Index) IsTrackedFile(path string) bool {
	_, existsInEntries := i.entries[path]
	return existsInEntries
}

func (i *Index) EntryForPath(path string) *Entry {
	return i.entries[path]
}

func (i *Index) IsTracked(path string) bool {
	_, existsInParents := i.parents[path]
	return i.IsTrackedFile(path) || existsInParents
}

func (i *Index) UpdateEntryStat(entry database.EntryObject, stat fs.FileInfo) {
	entry.UpdateStat(stat)
	i.changed = true
}

func (i *Index) clear() {
	i.entries = make(map[string]*Entry)
	i.keys = make([]string, 0)
	i.changed = false
}

func (i *Index) discardConflicts(entry *Entry) {
	for _, parent := range entry.ParentDirectories() {
		i.removeEntry(parent)
	}
	i.removeChildren(entry.Key())
}

func (i *Index) removeEntry(pathname string) {
	entry, ok := i.entries[pathname]
	if !ok {
		return
	}

	for index, key := range i.keys {
		if key == entry.Key() {
			i.keys = append(i.keys[:index], i.keys[index+1:]...)
			break
		}
	}
	delete(i.entries, entry.Key())

	for _, dirname := range entry.ParentDirectories() {
		delete(i.parents[dirname], entry.Key())
		if len(i.parents[dirname]) == 0 {
			delete(i.parents, dirname)
		}
	}
}

func (i *Index) removeChildren(path string) {
	children, ok := i.parents[path]
	if !ok {
		return
	}
	for child := range children {
		i.removeEntry(child)
	}
}

func (i *Index) storeEntry(entry *Entry) {
	key := entry.Key()

	_, exists := i.entries[key]
	if !exists {
		index := sort.SearchStrings(i.keys, key)
		i.keys = append(i.keys, "")
		copy(i.keys[index+1:], i.keys[index:])
		i.keys[index] = key
	}
	i.entries[key] = entry

	for _, dirname := range entry.ParentDirectories() {
		if i.parents[dirname] == nil {
			i.parents[dirname] = make(map[string]struct{})
		}
		i.parents[dirname][entry.Key()] = struct{}{}
	}
}

func (i *Index) openIndexFile() (*os.File, error) {
	return os.Open(i.pathname)
}

func (i *Index) readHeader(reader *Checksum) (int, error) {
	data, err := reader.Read(HEADER_SIZE)
	if err != nil {
		return 0, err
	}

	signature := string(data[:4])
	version := binary.BigEndian.Uint32(data[4:8])
	count := binary.BigEndian.Uint32(data[8:12])

	if signature != SIGNATURE {
		return 0, fmt.Errorf("Signature: expected '%s' but found '%s'", SIGNATURE, signature)
	}
	if version != VERSION {
		return 0, fmt.Errorf("Version: expected '%d' but found '%d'", VERSION, version)
	}

	return int(count), nil
}

func (i *Index) readEntries(reader *Checksum, count int) error {
	for ; count > 0; count-- {
		entry, err := reader.Read(ENTRY_MIN_SIZE)
		if err != nil {
			return err
		}

		for entry[len(entry)-1] != 0 {
			block, err := reader.Read(ENTRY_BLOCK_SIZE)
			if err != nil {
				return err
			}
			entry = append(entry, block...)
		}
		i.storeEntry(ParseEntry(entry))
	}
	return nil
}
