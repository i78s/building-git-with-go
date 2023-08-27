package database

type Entry struct {
	oid  string
	mode int
}

func NewEntry(oid string, mode int) *Entry {
	return &Entry{
		oid:  oid,
		mode: mode,
	}
}

func (e *Entry) Mode() int {
	return e.mode
}

func (e *Entry) Oid() string {
	return e.oid
}

func (e *Entry) IsTree() bool {
	return e.mode == TREE_MODE
}

func (e *Entry) IsNil() bool {
	return e == nil
}
