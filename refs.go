package jit

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
	lf := NewLockfile(headPath)
	err := lf.HoldForUpdate()

	if err != nil {
		return err
	}

	err = lf.Write([]byte(oid + "\n"))
	if err != nil {
		return err
	}
	return lf.Commit()
}

func (r *Refs) getHeadPath() string {
	return filepath.Join(r.pathname, "HEAD")
}
