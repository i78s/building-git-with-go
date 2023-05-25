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
	Ctime, CtimeNsec uint32
	Mtime, MtimeNsec uint32
	Dev, Ino         uint32
	Mode             uint32
	Uid, Gid         uint32
	Size             uint32
	Oid              string
	Flags            uint16
	Path             string
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
		Ctime:     uint32(stat.ModTime().Unix()),
		CtimeNsec: uint32(stat.ModTime().UnixNano() - stat.ModTime().Unix()*int64(time.Second)),
		Mtime:     uint32(stat.ModTime().Unix()),
		MtimeNsec: uint32(stat.ModTime().UnixNano() - stat.ModTime().Unix()*int64(time.Second)),
		Dev:       uint32(stat.Sys().(*syscall.Stat_t).Dev),
		Ino:       uint32(stat.Sys().(*syscall.Stat_t).Ino),
		Mode:      mode,
		Uid:       uint32(stat.Sys().(*syscall.Stat_t).Uid),
		Gid:       uint32(stat.Sys().(*syscall.Stat_t).Gid),
		Size:      uint32(stat.Size()),
		Oid:       oid,
		Flags:     flags,
		Path:      pathname,
	}
}

func (e *Entry) String() string {
	data := make([]byte, 62+len(e.Path)+1)
	binary.BigEndian.PutUint32(data[0:4], e.Ctime)
	binary.BigEndian.PutUint32(data[4:8], e.CtimeNsec)
	binary.BigEndian.PutUint32(data[8:12], e.Mtime)
	binary.BigEndian.PutUint32(data[12:16], e.MtimeNsec)
	binary.BigEndian.PutUint32(data[16:20], e.Dev)
	binary.BigEndian.PutUint32(data[20:24], e.Ino)
	binary.BigEndian.PutUint32(data[24:28], e.Mode)
	binary.BigEndian.PutUint32(data[28:32], e.Uid)
	binary.BigEndian.PutUint32(data[32:36], e.Gid)
	binary.BigEndian.PutUint32(data[36:40], e.Size)

	h, _ := hex.DecodeString(e.Oid)
	copy(data[40:60], h)

	binary.BigEndian.PutUint16(data[60:62], e.Flags)
	copy(data[62:], e.Path)

	for len(data)%8 != 0 {
		data = append(data, 0)
	}

	return string(data)
}
