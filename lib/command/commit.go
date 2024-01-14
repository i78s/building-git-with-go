package command

import (
	"building-git/lib/command/write_commit"
	"building-git/lib/editor"
	"building-git/lib/repository"
	"fmt"
	"io"
	"path/filepath"
	"time"
)

type CommitOption struct {
	write_commit.ReadOption
	Edit  bool
	IsTTY bool
}

type Commit struct {
	rootPath    string
	args        []string
	options     CommitOption
	repo        *repository.Repository
	writeCommit *write_commit.WriteCommit
	stdout      io.Writer
	stderr      io.Writer
}

func NewCommit(dir string, args []string, options CommitOption, stdout, stderr io.Writer) (*Commit, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRepository(rootPath)
	writeCommit := write_commit.NewWriteCommit(repo)
	return &Commit{
		rootPath:    rootPath,
		args:        args,
		options:     options,
		repo:        repo,
		writeCommit: writeCommit,
		stdout:      stdout,
		stderr:      stderr,
	}, nil
}

func (c *Commit) Run(now time.Time) int {
	c.repo.Index.Load()

	if c.writeCommit.PendingCommit().InProgress() {
		err := c.writeCommit.ResumeMerge(c.options.IsTTY)
		if err != nil {
			fmt.Fprintf(c.stderr, "%s", err.Error())
			return 128
		}
		return 0
	}

	parent, _ := c.repo.Refs.ReadHead()
	message, err := c.writeCommit.ReadMessage(c.options.ReadOption)
	if err != nil {
		return 1
	}
	message = c.composeMessage(message)
	parents := []string{}
	if parent != "" {
		parents = append(parents, parent)
	}

	commit, err := c.writeCommit.WriteCommit(parents, message, now)
	if err != nil {
		fmt.Fprintf(c.stderr, "%s", err.Error())
		return 1
	}
	c.writeCommit.PrintCommit(commit, c.stdout)

	return 0
}

func (c *Commit) composeMessage(message string) string {
	return editor.EditFile(c.writeCommit.CommitMessagePath(), c.options.IsTTY, func(e *editor.Editor) {
		e.Puts(message)
		e.Puts("")
		e.Note(write_commit.COMMIT_NOTES)

		if !c.options.Edit {
			e.Close()
		}
	})
}
