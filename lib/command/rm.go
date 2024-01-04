package command

import (
	"building-git/lib/repository"
	"fmt"
	"io"
	"path/filepath"
)

type RmOption struct {
}

type Rm struct {
	rootPath string
	args     []string
	options  RmOption
	repo     *repository.Repository
	stdout   io.Writer
	stderr   io.Writer
}

func NewRm(dir string, args []string, options RmOption, stdout, stderr io.Writer) (*Rm, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRepository(rootPath)

	return &Rm{
		rootPath: rootPath,
		args:     args,
		options:  options,
		repo:     repo,
		stdout:   stdout,
		stderr:   stderr,
	}, nil
}

func (r *Rm) Run() int {
	r.repo.Index.LoadForUpdate()
	for _, path := range r.args {
		r.removeFile(path)
	}
	r.repo.Index.WriteUpdates()

	return 0
}

func (r *Rm) removeFile(path string) {
	r.repo.Index.Remove(path)
	r.repo.Workspace.Remove(path)
	fmt.Fprintf(r.stdout, "rm %s\n", path)
}
