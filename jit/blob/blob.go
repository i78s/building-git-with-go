package blob

type Blob struct {
	Oid  string
	Data string
}

func NewBlob(data string) *Blob {
	return &Blob{
		Data: data,
	}
}

func (b *Blob) Type() string {
	return "blob"
}

func (b *Blob) String() string {
	return b.Data
}
