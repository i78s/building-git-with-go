package index

import (
	"encoding/binary"
	"encoding/hex"
	"os"
	"syscall"
	"time"
)

const (
	REGULAR_MODE    = 0100644
	EXECUTABLE_MODE = 0100755
	MAX_PATH_SIZE   = 0xfff
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
	var mode uint32
	if stat.Mode().Perm()&0111 == 0 {
		mode = REGULAR_MODE
	} else {
		mode = EXECUTABLE_MODE
	}
	flags := uint16(len(pathname))
	if flags > MAX_PATH_SIZE {
		flags = MAX_PATH_SIZE
	}

	return &Entry{
		ctime:     uint32(stat.ModTime().Unix()),
		ctimeNsec: uint32(stat.ModTime().UnixNano() - stat.ModTime().Unix()*int64(time.Second)),
		mtime:     uint32(stat.ModTime().Unix()),
		mtimeNsec: uint32(stat.ModTime().UnixNano() - stat.ModTime().Unix()*int64(time.Second)),
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

func (e *Entry) Key() string {
	return e.path
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
