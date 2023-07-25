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
		return 128
	}

	return 0
}

func (b *Branch) createBranch() error {
	branchName := b.args[0]
	startOid := ""
	var revision *repository.Revision
	var err error
	if len(b.args) > 1 {
		startPoint := b.args[1]
		revision = repository.NewRevision(b.repo, startPoint)
		startOid, err = revision.Resolve(repository.COMMIT)
	} else {
		startOid, err = b.repo.Refs.ReadHead()
	}

	if err != nil {
		if _, ok := err.(*repository.InvalidObjectError); ok {
			for _, he := range revision.Errors {
				fmt.Fprintf(b.stderr, "error: %v\n", he.Message)
				for _, line := range he.Hint {
					fmt.Fprintf(b.stderr, "hint: %v\n", line)
				}
			}
		}
		fmt.Fprintf(b.stderr, "fatal: %v", err)
		return err
	}

	err = b.repo.Refs.CreateBranch(branchName, startOid)
	if err != nil {
		if _, ok := err.(*repository.InvalidBranchError); ok {
			fmt.Fprintf(b.stderr, "fatal: %v", err)
		}
		return err
	}

	return nil
}
