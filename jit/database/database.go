package database

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

type Database struct {
	pathname string
}

type GitObject interface {
	GetOid() string
	SetOid(string)
	Type() string
	String() string
}

func NewDatabase(pathname string) *Database {
	return &Database{
		pathname: pathname,
	}
}

func (d *Database) Store(object GitObject) error {
	data := []byte(object.String())
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%s %d\x00", object.Type(), len(data))
	buf.Write(data)
	cont := buf.Bytes()

	hash := sha1.New()
	_, err := hash.Write(cont)
	if err != nil {
		return err
	}

	object.SetOid(fmt.Sprintf("%x", hash.Sum(nil)))
	d.writeObject(object.GetOid(), cont)
	return nil
}

func (d *Database) writeObject(oid string, content []byte) error {
	objectPath := filepath.Join(d.pathname, oid[:2], oid[2:])
	dirname := filepath.Dir(objectPath)
	tempPath := filepath.Join(dirname, generateTempName())

	if _, err := os.Stat(dirname); os.IsNotExist(err) {
		if err := os.MkdirAll(dirname, os.ModePerm); err != nil {
			return err
		}
	}

	// Create a new file with os.O_RDWR|os.O_CREATE|os.O_EXCL flags
	file, err := os.OpenFile(tempPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	compressor, err := zlib.NewWriterLevel(file, zlib.BestSpeed)
	if err != nil {
		return err
	}
	if _, err := compressor.Write(content); err != nil {
		return err
	}
	if err := compressor.Close(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	if err := os.Rename(file.Name(), objectPath); err != nil {
		return err
	}
	return nil
}

const TEMP_CHARS = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateTempName() string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 6)
	for i := range b {
		b[i] = TEMP_CHARS[rand.Intn(len(TEMP_CHARS))]
	}
	return "tmp_obj_" + string(b)
}
