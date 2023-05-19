package entry

import "io/fs"

const (
	REGULAR_MODE    = "100644"
	EXECUTABLE_MODE = "100755"
)

type Entry struct {
	Oid  string
	Name string
	stat fs.FileInfo
}

func NewEntry(name, oid string, stat fs.FileInfo) *Entry {
	return &Entry{
		Oid:  oid,
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
