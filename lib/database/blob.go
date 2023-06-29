package database

type Blob struct {
	oid  string
	Data string
}

func NewBlob(data string) *Blob {
	return &Blob{
		Data: data,
	}
}

func ParseBlob(data string) GitObject {
	return NewBlob(data)
}

func (b *Blob) Type() string {
	return "blob"
}

func (b *Blob) String() string {
	return b.Data
}

func (b *Blob) Oid() string {
	return b.oid
}

func (b *Blob) SetOid(oid string) {
	b.oid = oid
}
