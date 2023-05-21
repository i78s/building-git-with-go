package jit

import (
	"io/fs"
	"path/filepath"
)

const (
	REGULAR_MODE    = "100644"
	EXECUTABLE_MODE = "100755"
	DIRECTORY_MODE  = "40000"
)

type Entry struct {
	oid  string
	Name string
	stat fs.FileInfo
}

func NewEntry(name, oid string, stat fs.FileInfo) *Entry {
	return &Entry{
		oid:  oid,
		Name: name,
		stat: stat,
	}
}

func (e *Entry) Mode() string {
	if e.stat.Mode()&0111 == 0 {
		return REGULAR_MODE
	} else {
		return EXECUTABLE_MODE
	}
}

func (e *Entry) ParentDirectories() []string {
	var dirs []string
	path := e.Name
	for {
		path = filepath.Dir(path)
		if path == "." || path == string(filepath.Separator) {
			break
		}
		dirs = append(dirs, path)
	}
	return dirs
}

func (e *Entry) Basename() string {
	return filepath.Base(e.Name)
}

func (e *Entry) GetOid() string {
	return e.oid
}
