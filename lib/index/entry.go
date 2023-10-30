package index

import (
	"building-git/lib/database"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"syscall"
)

const (
	ENTRY_BLOCK_SIZE = 8
	ENTRY_MIN_SIZE   = 64
	REGULAR_MODE     = 0o100644
	EXECUTABLE_MODE  = 0o100755
	MAX_PATH_SIZE    = 0xfff
)

type Entry struct {
	ctime, ctimeNsec uint32
	mtime, mtimeNsec uint32
	dev, ino         uint32
	mode             uint32
	uid, gid         uint32
	size             uint32
	oid              string
	flags            uint16
	path             string
}

func CreateEntry(pathname string, oid string, stat os.FileInfo) *Entry {
	mode := ModeForStat(stat)
	flags := uint16(len(pathname))
	if flags > MAX_PATH_SIZE {
		flags = MAX_PATH_SIZE
	}

	return &Entry{
		ctime:     uint32(stat.ModTime().Unix()),
		ctimeNsec: uint32(stat.ModTime().Nanosecond()),
		mtime:     uint32(stat.ModTime().Unix()),
		mtimeNsec: uint32(stat.ModTime().Nanosecond()),
		dev:       uint32(stat.Sys().(*syscall.Stat_t).Dev),
		ino:       uint32(stat.Sys().(*syscall.Stat_t).Ino),
		mode:      mode,
		uid:       uint32(stat.Sys().(*syscall.Stat_t).Uid),
		gid:       uint32(stat.Sys().(*syscall.Stat_t).Gid),
		size:      uint32(stat.Size()),
		oid:       oid,
		flags:     flags,
		path:      pathname,
	}
}

func CreateEntryFromDB(pathname string, item database.TreeObject, n int) *Entry {
	minPathSize := int(math.Min(float64(len(pathname)), float64(MAX_PATH_SIZE)))
	shiftedN := n << 12
	flags := shiftedN | minPathSize

	return &Entry{
		ctime:     uint32(0),
		ctimeNsec: uint32(0),
		mtime:     uint32(0),
		mtimeNsec: uint32(0),
		dev:       uint32(0),
		ino:       uint32(0),
		mode:      uint32(item.Mode()),
		uid:       uint32(0),
		gid:       uint32(0),
		size:      uint32(0),
		oid:       item.Oid(),
		flags:     uint16(flags),
		path:      pathname,
	}
}

func ModeForStat(stat fs.FileInfo) uint32 {
	if stat.Mode().Perm()&0111 == 0 {
		return REGULAR_MODE
	}
	return EXECUTABLE_MODE
}

func ParseEntry(data []byte) *Entry {
	nullPos := bytes.IndexByte(data[62:], 0)
	if nullPos == -1 {
		nullPos = len(data) - 62
	}

	return &Entry{
		ctime:     binary.BigEndian.Uint32(data[0:4]),
		ctimeNsec: binary.BigEndian.Uint32(data[4:8]),
		mtime:     binary.BigEndian.Uint32(data[8:12]),
		mtimeNsec: binary.BigEndian.Uint32(data[12:16]),
		dev:       binary.BigEndian.Uint32(data[16:20]),
		ino:       binary.BigEndian.Uint32(data[20:24]),
		mode:      binary.BigEndian.Uint32(data[24:28]),
		uid:       binary.BigEndian.Uint32(data[28:32]),
		gid:       binary.BigEndian.Uint32(data[32:36]),
		size:      binary.BigEndian.Uint32(data[36:40]),
		oid:       hex.EncodeToString(data[40:60]),
		flags:     binary.BigEndian.Uint16(data[60:62]),
		path:      string(data[62 : 62+nullPos]),
	}
}

func (e *Entry) Path() string {
	return e.path
}

func (e *Entry) Key() [2]string {
	return [2]string{e.path, e.Stage()}
}

func (e *Entry) Stage() string {
	return fmt.Sprint((e.flags >> 12) & 0x3)
}

func (e *Entry) Mode() int {
	return int(e.mode)
}

func (e *Entry) Oid() string {
	return e.oid
}

func (e *Entry) ParentDirectories() []string {
	var dirs []string
	path := e.path
	for {
		path = filepath.Dir(path)
		if path == "." || path == string(filepath.Separator) {
			break
		}
		dirs = append([]string{path}, dirs...)
	}
	return dirs
}

func (e *Entry) Basename() string {
	return filepath.Base(e.path)
}

func (e *Entry) IsStatMatch(stat fs.FileInfo) bool {
	return e.mode == ModeForStat(stat) &&
		(e.size == 0 || e.size == uint32(stat.Size()))
}

func (e *Entry) UpdateStat(stat fs.FileInfo) {
	e.mtime = uint32(stat.ModTime().Unix())
	e.mtimeNsec = uint32(stat.ModTime().Nanosecond())
	e.mode = ModeForStat(stat)
	e.size = uint32(stat.Size())
	if stat, ok := stat.Sys().(*syscall.Stat_t); ok {
		s, n := stat.Ctimespec.Unix()
		e.ctime = uint32(s)
		e.ctimeNsec = uint32(n)
		e.dev = uint32(stat.Dev)
		e.ino = uint32(stat.Ino)
		e.uid = stat.Uid
		e.gid = stat.Gid
	}
}

func (e *Entry) IsTimesMatch(stat fs.FileInfo) bool {
	mtime := uint32(stat.ModTime().Unix())
	mtimeNsec := uint32(stat.ModTime().Nanosecond())
	var ctime uint32
	var ctimeNsec uint32
	if stat, ok := stat.Sys().(*syscall.Stat_t); ok {
		s, n := stat.Ctimespec.Unix()
		ctime = uint32(s)
		ctimeNsec = uint32(n)
	}
	return e.ctime == ctime &&
		e.ctimeNsec == ctimeNsec &&
		e.mtime == mtime &&
		e.mtimeNsec == mtimeNsec
}

func (e *Entry) String() string {
	data := make([]byte, 62+len(e.path)+1)
	binary.BigEndian.PutUint32(data[0:4], e.ctime)
	binary.BigEndian.PutUint32(data[4:8], e.ctimeNsec)
	binary.BigEndian.PutUint32(data[8:12], e.mtime)
	binary.BigEndian.PutUint32(data[12:16], e.mtimeNsec)
	binary.BigEndian.PutUint32(data[16:20], e.dev)
	binary.BigEndian.PutUint32(data[20:24], e.ino)
	binary.BigEndian.PutUint32(data[24:28], e.mode)
	binary.BigEndian.PutUint32(data[28:32], e.uid)
	binary.BigEndian.PutUint32(data[32:36], e.gid)
	binary.BigEndian.PutUint32(data[36:40], e.size)

	h, _ := hex.DecodeString(e.oid)
	copy(data[40:60], h)

	binary.BigEndian.PutUint16(data[60:62], e.flags)
	copy(data[62:], e.path)

	for len(data)%8 != 0 {
		data = append(data, 0)
	}

	return string(data)
}

func (e *Entry) IsNil() bool {
	return e == nil
}
