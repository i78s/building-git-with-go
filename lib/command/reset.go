package command

import (
	"building-git/lib/repository"
	"io"
	"path/filepath"
)

type ResetOption struct {
}

type Reset struct {
	rootPath string
	args     []string
	options  ResetOption
	repo     *repository.Repository
	headOid  string
	stdout   io.Writer
	stderr   io.Writer
}

func NewReset(dir string, args []string, options ResetOption, stdout, stderr io.Writer) (*Reset, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRepository(rootPath)

	return &Reset{
		rootPath: rootPath,
		args:     args,
		options:  options,
		repo:     repo,
		stdout:   stdout,
		stderr:   stderr,
	}, nil
}

func (r *Reset) Run() int {
	headOid, _ := r.repo.Refs.ReadHead()
	r.headOid = headOid

	r.repo.Index.LoadForUpdate()
	for _, path := range r.args {
		r.resetPath(path)
	}
	r.repo.Index.WriteUpdates()

	return 0
}

func (r *Reset) resetPath(pathname string) {
	listing := r.repo.Database.LoadTreeList(r.headOid, pathname)
	r.repo.Index.Remove(pathname)

	for path, entry := range listing {
		r.repo.Index.AddFromDb(path, entry)
	}
}
