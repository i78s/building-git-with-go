package command

import (
	"building-git/lib/repository"
	"fmt"
	"io"
	"path/filepath"
)

type CheckOutOption struct {
}

type CheckOut struct {
	rootPath string
	args     []string
	options  CheckOutOption
	repo     *repository.Repository
	stdout   io.Writer
	stderr   io.Writer
}

func NewCheckOut(dir string, args []string, options CheckOutOption, stdout, stderr io.Writer) (*CheckOut, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRepository(rootPath)

	return &CheckOut{
		rootPath: rootPath,
		args:     args,
		repo:     repo,
		stdout:   stdout,
		stderr:   stderr,
	}, nil
}

func (c *CheckOut) Run() int {
	target := c.args[0]
	currentOid, _ := c.repo.Refs.ReadHead()

	revision := repository.NewRevision(c.repo, target)
	targetOid, err := revision.Resolve(repository.COMMIT)

	c.repo.Index.LoadForUpdate()

	if err != nil {
		c.handleInvalidObject(revision, err)
		return 1
	}

	treeDiff := c.repo.Database.TreeDiff(currentOid, targetOid)
	migration := c.repo.Migration(treeDiff)
	migration.ApplyChanges()

	c.repo.Index.WriteUpdates()
	c.repo.Refs.UpdateHead(targetOid)

	return 0
}

func (c *CheckOut) handleInvalidObject(revision *repository.Revision, err error) {
	for _, err := range revision.Errors {
		fmt.Fprintf(c.stderr, "error: %v\n", err.Message)
		for _, line := range err.Hint {
			fmt.Fprintf(c.stderr, "hint: %v\n", line)
		}
	}
	fmt.Fprintf(c.stderr, "error: %v\n", err)
}
