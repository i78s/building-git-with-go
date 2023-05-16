package entry

type Entry struct {
	Oid  string
	Name string
}

func NewEntry(name, oid string) *Entry {
	return &Entry{
		Oid:  oid,
		Name: name,
	}
}
