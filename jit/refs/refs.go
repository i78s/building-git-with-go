package refs

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Refs struct {
	pathname string
}

func NewRefs(pathname string) *Refs {
	return &Refs{pathname: pathname}
}

func (r *Refs) ReadHead() (string, error) {
	headPath := r.getHeadPath()

	if _, err := os.Stat(headPath); os.IsNotExist(err) {
		return "", nil
	} else if err != nil {
		return "", err
	}

	data, err := ioutil.ReadFile(headPath)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func (r *Refs) UpdateHead(oid string) error {
	headPath := r.getHeadPath()

	f, err := os.OpenFile(headPath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(oid + "\n")
	return err
}

func (r *Refs) getHeadPath() string {
	return filepath.Join(r.pathname, "HEAD")
}
