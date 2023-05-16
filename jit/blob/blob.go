package blob

type Blob struct {
	oid  string
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

func (b *Blob) GetOid() string {
	return b.oid
}

func (b *Blob) SetOid(oid string) {
	b.oid = oid
}
