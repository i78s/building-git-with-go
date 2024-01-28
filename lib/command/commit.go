package command

import (
	"building-git/lib/command/write_commit"
	"building-git/lib/database"
	"building-git/lib/editor"
	"building-git/lib/repository"
	"fmt"
	"io"
	"path/filepath"
	"time"
)

type CommitOption struct {
	write_commit.ReadOption
	Edit      bool
	Reuse     string
	Amend     bool
	IsTTY     bool
	EditorCmd func(path string) editor.Executable
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
	writeCommit := write_commit.NewWriteCommit(repo, options.EditorCmd)
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

	if c.options.Amend {
		c.handleAmend()
		return 0
	}
	mergeType := c.writeCommit.PendingCommit().MergeType()
	if mergeType != -1 {
		err := c.writeCommit.ResumeMerge(mergeType, c.options.IsTTY)
		if err != nil {
			fmt.Fprintf(c.stderr, "%s", err.Error())
			return 128
		}
		return 0
	}

	parent, _ := c.repo.Refs.ReadHead()
	message, err := c.writeCommit.ReadMessage(c.options.ReadOption)
	reusedMessage := c.reusedMessage()
	if err != nil {
		if reusedMessage == "" {
			return 1
		}
		message = reusedMessage
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
	path := c.writeCommit.CommitMessagePath()
	return editor.EditFile(path, c.options.EditorCmd(path), c.options.IsTTY, func(e *editor.Editor) {
		e.Puts(message)
		e.Puts("")
		e.Note(write_commit.COMMIT_NOTES)

		if !c.options.Edit {
			e.Close()
		}
	})
}

func (c *Commit) reusedMessage() string {
	if c.options.Reuse == "" {
		return ""
	}

	revision := repository.NewRevision(c.repo, c.options.Reuse)
	rev, err := revision.Resolve(repository.COMMIT)
	if err != nil {
		return ""
	}
	commitObj, _ := c.repo.Database.Load(rev)
	commit := commitObj.(*database.Commit)

	return commit.Message()
}

func (c *Commit) handleAmend() {
	head, _ := c.repo.Refs.ReadHead()
	obj, _ := c.repo.Database.Load(head)
	old := obj.(*database.Commit)
	tree := c.writeCommit.WriteTree()

	message := c.composeMessage(old.Message())
	committer := c.writeCommit.CurrentAuthor(time.Now())

	new := database.NewCommit(old.Parents, tree.Oid(), old.Author(), committer, message)
	c.repo.Database.Store(new)
	c.repo.Refs.UpdateHead(new.Oid())

	c.writeCommit.PrintCommit(new, c.stdout)
}
