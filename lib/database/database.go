package database

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Database struct {
	pathname string
	objects  map[string]GitObject
}

type GitObject interface {
	Oid() string
	SetOid(string)
	Type() string
	String() string
}

func NewDatabase(pathname string) *Database {
	return &Database{
		pathname: pathname,
		objects:  map[string]GitObject{},
	}
}

func (d *Database) Store(object GitObject) error {
	cont := d.serializeObject(object)
	oid, err := d.hashContent(cont)
	if err != nil {
		return err
	}

	object.SetOid(oid)
	d.writeObject(object.Oid(), cont)
	return nil
}

func (d *Database) Load(oid string) (GitObject, error) {
	if obj, exists := d.objects[oid]; exists {
		return obj, nil
	}
	obj, err := d.readObject(oid)
	if err != nil {
		return nil, err
	}
	d.objects[oid] = obj

	return obj, nil
}

func (d *Database) HashObject(object GitObject) (string, error) {
	return d.hashContent(d.serializeObject(object))
}

func (d *Database) LoadTreeEntry(oid string, pathname string) TreeObject {
	commitObj, _ := d.Load(oid)
	commit := commitObj.(*Commit)
	root := NewEntry(commit.Tree(), TREE_MODE)

	if pathname == "" {
		return root
	}

	var entry TreeObject = root
	for _, name := range strings.Split(pathname, "/") {
		if entry != nil {
			obj, _ := d.Load(entry.Oid())
			entry = obj.(*Tree).Entries[name]
		} else {
			return nil
		}
	}
	return entry
}

func (d *Database) LoadTreeList(oid, pathname string) map[string]*Entry {
	list := make(map[string]*Entry)

	if oid == "" {
		return list
	}

	entry := d.LoadTreeEntry(oid, pathname)
	d.BuildList(list, entry, pathname)

	return list
}

func (d *Database) BuildList(list map[string]*Entry, entry TreeObject, prefix string) {
	if entry == nil || entry.IsNil() {
		return
	}

	e := entry.(*Entry)
	if !e.IsTree() {
		list[prefix] = e
		return
	}

	obj, _ := d.Load(e.Oid())
	for name, item := range obj.(*Tree).Entries {
		d.BuildList(list, item.(*Entry), filepath.Join(prefix, name))
	}
}

func (d *Database) ShortOid(oid string) string {
	return oid[:7]
}

func (d *Database) PrefixMatch(name string) ([]string, error) {
	dirname := filepath.Dir(d.objectPath(name))
	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var oids []string
	for _, file := range files {
		oid := filepath.Base(dirname) + file.Name()
		if strings.HasPrefix(oid, name) {
			oids = append(oids, oid)
		}
	}

	return oids, nil
}

func (d *Database) TreeDiff(a, b string, filter *PathFilter) map[string][2]TreeObject {
	if filter == nil {
		filter = NewPathFilter(nil, "")
	}
	diff := NewTreeDiff(d)
	diff.compareOids(a, b, filter)
	return diff.changes
}

func (d *Database) serializeObject(object GitObject) []byte {
	data := []byte(object.String())
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%s %d\x00", object.Type(), len(data))
	buf.Write(data)
	return buf.Bytes()
}

func (d *Database) hashContent(content []byte) (string, error) {
	hash := sha1.New()
	_, err := hash.Write(content)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (d *Database) objectPath(oid string) string {
	return filepath.Join(d.pathname, oid[:2], oid[2:])
}

func (d *Database) writeObject(oid string, content []byte) error {
	objectPath := d.objectPath(oid)
	dirname := filepath.Dir(objectPath)
	tempPath := filepath.Join(dirname, generateTempName())

	if _, err := os.Stat(objectPath); err == nil {
		return nil
	}

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

func (d *Database) readObject(oid string) (GitObject, error) {
	data, err := ioutil.ReadFile(d.objectPath(oid))
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(data)
	zr, err := zlib.NewReader(reader)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	bufReader := bufio.NewReader(zr)
	line, err := bufReader.ReadString(' ')
	if err != nil {
		return nil, err
	}

	objectType := strings.TrimSpace(string(line))
	size, err := bufReader.ReadString(0)
	size = strings.TrimRight(size, "\x00")

	var object GitObject
	switch objectType {
	case "blob":
		object = ParseBlob(bufReader)
	case "tree":
		object, err = ParseTree(bufReader)
	case "commit":
		object, err = ParseCommit(bufReader)
	default:
		return nil, fmt.Errorf("unrecognized object type: %s", objectType)
	}

	if err != nil {
		return nil, err
	}

	object.SetOid(oid)

	return object, nil
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
