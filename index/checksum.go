package index

import (
	"crypto/sha1"
	"fmt"
	"hash"
	"io"
	"os"
)

const (
	CHECKSUM_SIZE = 20
)

type Checksum struct {
	file   os.File
	digest hash.Hash
}

func NewChecksum(file os.File) *Checksum {
	return &Checksum{
		file:   file,
		digest: sha1.New(),
	}
}

func (c *Checksum) Write(data []byte) {
	c.file.Write(data)
	c.digest.Write(data)
}

func (c *Checksum) WriteChecksum() {
	c.file.Write(c.digest.Sum(nil))
}

func (c *Checksum) Read(size int) ([]byte, error) {
	data := make([]byte, size)
	if _, err := c.file.Read(data); err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("Unexpected end-of-file while reading index")
		}
		return nil, err
	}
	c.digest.Write(data)

	return data, nil
}

func (c *Checksum) VerifyChecksum() error {
	data := make([]byte, CHECKSUM_SIZE)
	sum, err := c.file.Read(data)
	if err != nil {
		if err == io.EOF {
			return fmt.Errorf("Unexpected end-of-file while reading index")
		}
		return err
	}
	if sum == c.digest.Size() {
		return fmt.Errorf("Checksum does not match value stored on disk")
	}

	return nil
}
