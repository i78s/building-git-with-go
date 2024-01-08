package command

import (
	"building-git/lib/repository"
	"io"
	"path/filepath"
)

type ResetOption struct {
}

type Reset struct {
	rootPath  string
	args      []string
	options   ResetOption
	repo      *repository.Repository
	commitOid string
	stdout    io.Writer
	stderr    io.Writer
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
	r.selectCommitOid()

	r.repo.Index.LoadForUpdate()
	for _, path := range r.args {
		r.resetPath(path)
	}
	r.repo.Index.WriteUpdates()

	return 0
}

func (r *Reset) selectCommitOid() {
	revision := repository.HEAD
	if len(r.args) > 0 {
		revision = r.args[0]
	}
	rev := repository.NewRevision(r.repo, revision)
	oid, err := rev.Resolve(repository.COMMIT)

	if err != nil {
		headOid, _ := r.repo.Refs.ReadHead()
		r.commitOid = headOid
		return
	}
	r.args = r.args[1:]
	r.commitOid = oid
}

func (r *Reset) resetPath(pathname string) {
	listing := r.repo.Database.LoadTreeList(r.commitOid, pathname)
	r.repo.Index.Remove(pathname)

	for path, entry := range listing {
		r.repo.Index.AddFromDb(path, entry)
	}
}
