package jit

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type LockDeniedError struct {
	Message string
}

func (e *LockDeniedError) Error() string {
	return e.Message
}

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
	_, err := lf.HoldForUpdate()

	if err != nil {
		if os.IsPermission(err) {
			return &LockDeniedError{err.Error()}
		}
		log.Fatalf("Could not acquire lock on file: %s", headPath)
	}

	err = lf.Write(oid + "\n")
	if err != nil {
		return err
	}
	return lf.Commit()
}

func (r *Refs) getHeadPath() string {
	return filepath.Join(r.pathname, "HEAD")
}
