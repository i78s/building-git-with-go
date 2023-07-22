package lib

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var INVALID_NAME = regexp.MustCompile(`^\.|\/\.|\.\.|^\/|\/$|\.lock$|@\{|[\x00-\x20*:?\[\\^~\x7f]`)

const HEAD = "HEAD"

type InvalidBranchError struct {
	msg string
}

func (e *InvalidBranchError) Error() string {
	return fmt.Sprintf("%s", e.msg)
}

type Refs struct {
	pathname  string
	refsPath  string
	headsPath string
}

func NewRefs(pathname string) *Refs {
	refsPath := filepath.Join(pathname, "refs")
	return &Refs{
		pathname:  pathname,
		refsPath:  refsPath,
		headsPath: filepath.Join(refsPath, "heads"),
	}
}

func (r *Refs) ReadHead() (string, error) {
	return r.readRefFile(filepath.Join(r.pathname, "HEAD"))
}

func (r *Refs) UpdateHead(oid string) error {
	return r.updateRefFile(filepath.Join(r.pathname, "HEAD"), oid)
}

func (r *Refs) ReadRef(name string) (string, error) {
	path, err := r.pathForName(name)

	if err != nil {
		return "", err
	}
	return r.readRefFile(path)
}

func (r *Refs) CreateBranch(branchName string) error {
	path := filepath.Join(r.headsPath, branchName)

	if INVALID_NAME.MatchString(branchName) {
		return &InvalidBranchError{
			msg: fmt.Sprintf("'%s' is not a valid branch name.", branchName),
		}
	}

	if _, err := os.Stat(path); err == nil {
		return &InvalidBranchError{
			msg: fmt.Sprintf("A branch named '%s' already exists.", branchName),
		}
	}

	head, err := r.ReadHead()
	if err != nil {
		return err
	}
	return r.updateRefFile(path, head)
}

func (r *Refs) pathForName(name string) (string, error) {
	prefixes := []string{r.pathname, r.refsPath, r.headsPath}

	var err error
	for _, prefix := range prefixes {
		path := filepath.Join(prefix, name)
		if _, err = os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", err
}

func (r *Refs) readRefFile(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", nil
	} else if err != nil {
		return "", err
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func (r *Refs) updateRefFile(path, oid string) error {
	lockfile := NewLockfile(path)

	for {
		err := lockfile.HoldForUpdate()
		if err != nil {
			if _, ok := err.(*MissingParentError); ok {
				err := os.MkdirAll(filepath.Dir(path), 0755)
				if err != nil {
					return err
				}
				continue
			} else {
				return err
			}
		}
		break
	}

	err := lockfile.Write([]byte(oid + "\n"))
	if err != nil {
		return err
	}
	err = lockfile.Commit()
	if err != nil {
		return err
	}

	return nil
}
