package command

import (
	"building-git/lib/command/write_commit"
	"building-git/lib/repository"
	"fmt"
	"io"
	"path/filepath"
	"time"
)

type CommitOption struct {
	write_commit.ReadOption
}

type Commit struct {
	rootPath string
	args     []string
	options  CommitOption
	repo     *repository.Repository
	stdout   io.Writer
	stderr   io.Writer
}

func NewCommit(dir string, args []string, options CommitOption, stdout, stderr io.Writer) (*Commit, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRepository(rootPath)

	return &Commit{
		rootPath: rootPath,
		args:     args,
		options:  options,
		repo:     repo,
		stdout:   stdout,
		stderr:   stderr,
	}, nil
}

func (c *Commit) Run(now time.Time) int {
	writeCommit := write_commit.NewWriteCommit(c.repo)
	c.repo.Index.Load()

	if writeCommit.PendingCommit().InProgress() {
		err := writeCommit.ResumeMerge()
		if err != nil {
			fmt.Fprintf(c.stderr, "%s", err.Error())
			return 128
		}
		return 0
	}

	parent, _ := c.repo.Refs.ReadHead()
	message, err := writeCommit.ReadMessage(c.options.ReadOption)
	if err != nil {
		return 1
	}

	parents := []string{}
	if parent != "" {
		parents = append(parents, parent)
	}

	commit := writeCommit.WriteCommit(parents, message, now)
	writeCommit.PrintCommit(commit, c.stdout)

	return 0
}
