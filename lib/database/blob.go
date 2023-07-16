package database

import (
	"bufio"
	"io"
)

type Blob struct {
	oid  string
	data string
}

func NewBlob(data string) *Blob {
	return &Blob{
		data: data,
	}
}

func ParseBlob(reader *bufio.Reader) *Blob {
	data, _ := io.ReadAll(reader)
	return NewBlob(string(data))
}

func (b *Blob) Type() string {
	return "blob"
}

func (b *Blob) String() string {
	return b.data
}

func (b *Blob) Oid() string {
	return b.oid
}

func (b *Blob) SetOid(oid string) {
	b.oid = oid
}
