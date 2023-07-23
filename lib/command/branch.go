package command

import (
	"building-git/lib/repository"

	"fmt"
	"io"
	"path/filepath"
)

type Branch struct {
	rootPath string
	args     []string
	options  BranchOption
	repo     *repository.Repository
	status   *repository.Status
	stdout   io.Writer
	stderr   io.Writer
}

type BranchOption struct {
}

func NewBranch(dir string, args []string, options BranchOption, stdout, stderr io.Writer) (*Branch, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRepository(rootPath)

	return &Branch{
		rootPath: rootPath,
		args:     args,
		options:  options,
		repo:     repo,
		stdout:   stdout,
		stderr:   stderr,
	}, nil
}

func (b *Branch) Run() int {
	err := b.createBranch()

	if err != nil {
		fmt.Fprintf(b.stderr, "fatal: %v", err)
		return 128
	}

	return 0
}

func (b *Branch) createBranch() error {
	branchName := b.args[0]
	startOid := ""
	var err error
	if len(b.args) > 1 {
		startPoint := b.args[1]
		revision := repository.NewRevision(b.repo, startPoint)
		startOid, err = revision.Resolve()
	} else {
		startOid, err = b.repo.Refs.ReadHead()
	}

	if err != nil {
		return err
	}

	return b.repo.Refs.CreateBranch(branchName, startOid)
}
