package repository

import (
	"building-git/lib/errors"
	"building-git/lib/lockfile"
	"regexp"
	"strings"

	"fmt"
	"os"
	"path/filepath"
)

const HEAD = "HEAD"

var symRefRegexp = regexp.MustCompile(`^ref: (.+)$`)

type InvalidBranchError struct {
	msg string
}

func (e *InvalidBranchError) Error() string {
	return fmt.Sprintf("%s", e.msg)
}

type SymRef struct {
	Path string
}

type Ref struct {
	Oid string
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
	return r.readSymRef(filepath.Join(r.pathname, HEAD))
}

func (r *Refs) UpdateHead(oid string) error {
	return r.updateRefFile(filepath.Join(r.pathname, HEAD), oid)
}

func (r *Refs) SetHead(revision, oid string) error {
	head := filepath.Join(r.pathname, HEAD)
	path := filepath.Join(r.headsPath, revision)

	if fileInfo, err := os.Stat(path); err == nil && fileInfo.Mode().IsRegular() {
		relative, err := relativePathFrom(r.pathname, path)
		if err != nil {
			return err
		}
		return r.updateRefFile(head, fmt.Sprintf("ref: %s", relative))
	}
	return r.updateRefFile(head, oid)
}

func relativePathFrom(base, target string) (string, error) {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return "", err
	}
	return rel, nil
}

func (r *Refs) ReadRef(name string) (string, error) {
	path, err := r.pathForName(name)

	if err != nil {
		return "", err
	}
	return r.readSymRef(path)
}

func (r *Refs) CreateBranch(branchName, startOid string) error {
	path := filepath.Join(r.headsPath, branchName)
	if !IsValidRef(branchName) {
		return &InvalidBranchError{
			msg: fmt.Sprintf("'%s' is not a valid branch name.", branchName),
		}
	}

	if _, err := os.Stat(path); err == nil {
		return &InvalidBranchError{
			msg: fmt.Sprintf("A branch named '%s' already exists.", branchName),
		}
	}

	return r.updateRefFile(path, startOid)
}

func (r *Refs) CurrentRef(source string) (interface{}, error) {
	if source == "" {
		source = HEAD
	}
	ref, err := r.readOidOrSymRef(filepath.Join(r.pathname, source))
	if err != nil {
		return nil, err
	}

	switch v := ref.(type) {
	case SymRef:
		return r.CurrentRef(v.Path)
	default:
		return SymRef{Path: source}, nil
	}
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

func (r *Refs) readOidOrSymRef(path string) (interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	trimedData := strings.TrimSpace(string(data))
	matches := symRefRegexp.FindStringSubmatch(trimedData)
	if matches != nil {
		return SymRef{Path: matches[1]}, nil
	}
	return Ref{Oid: trimedData}, nil
}

func (r *Refs) readSymRef(path string) (string, error) {
	ref, err := r.readOidOrSymRef(path)
	if err != nil {
		return "", err
	}

	switch v := ref.(type) {
	case SymRef:
		return r.readSymRef(filepath.Join(r.pathname, v.Path))
	case Ref:
		return v.Oid, nil
	}
	return "", fmt.Errorf("")
}

func (r *Refs) updateRefFile(path, oid string) error {
	lockfile := lockfile.NewLockfile(path)

	for {
		err := lockfile.HoldForUpdate()
		if err != nil {
			if _, ok := err.(*errors.MissingParentError); ok {
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
