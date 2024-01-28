package command

import (
	"building-git/lib/command/write_commit"
	"building-git/lib/database"
	"building-git/lib/editor"
	"building-git/lib/merge"
	"building-git/lib/repository"
	"fmt"
	"io"
	"path/filepath"
	"time"
)

type CherryPickOption struct {
	EditorCmd func(path string) editor.Executable
}

type CherryPick struct {
	rootPath    string
	args        []string
	options     CherryPickOption
	repo        *repository.Repository
	writeCommit *write_commit.WriteCommit
	stdout      io.Writer
	stderr      io.Writer
}

func NewCherryPick(dir string, args []string, options CherryPickOption, stdout, stderr io.Writer) (*CherryPick, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRepository(rootPath)
	writeCommit := write_commit.NewWriteCommit(repo, options.EditorCmd)
	return &CherryPick{
		rootPath:    rootPath,
		args:        args,
		options:     options,
		repo:        repo,
		writeCommit: writeCommit,
		stdout:      stdout,
		stderr:      stderr,
	}, nil
}

func (c *CherryPick) Run() int {
	revision := repository.NewRevision(c.repo, c.args[0])
	rev, err := revision.Resolve(repository.COMMIT)
	if err != nil {
		return 1
	}
	commitObj, _ := c.repo.Database.Load(rev)
	commit := commitObj.(*database.Commit)

	err = c.pick(commit)
	if err != nil {
		return 1
	}
	return 0
}

func (c *CherryPick) pick(commit *database.Commit) error {
	inputs := c.pickMergeInputs(commit)
	err := c.resolveMerge(inputs)
	if err != nil {
		return err
	}

	picked := database.NewCommit(
		[]string{inputs.LeftOid()},
		c.writeCommit.WriteTree().Oid(),
		commit.Author(),
		c.writeCommit.CurrentAuthor(time.Now()),
		commit.Message(),
	)
	c.finishCommit(picked)
	return nil
}

func (c *CherryPick) pickMergeInputs(commit *database.Commit) merge.ResolveInputs {
	short := c.repo.Database.ShortOid(commit.Oid())
	leftName := repository.HEAD
	leftOid, _ := c.repo.Refs.ReadHead()
	rightName := fmt.Sprintf("%s... %s", short, commit.TitleLine())
	rightOid := commit.Oid()

	return merge.NewCherryPick(
		c.repo,
		leftName, rightName,
		leftOid, rightOid,
		[]string{commit.Parent()},
	)
}

func (c *CherryPick) resolveMerge(inputs merge.ResolveInputs) error {
	c.repo.Index.LoadForUpdate()
	resolve := merge.NewResolve(c.repo, inputs, func(fn func() string) {
		info := fn()
		fmt.Fprintf(c.stdout, info+"\n")
	})
	err := resolve.Execute()
	if err != nil {
		return err
	}
	c.repo.Index.WriteUpdates()
	return nil
}

func (c *CherryPick) finishCommit(commit *database.Commit) {
	c.repo.Database.Store(commit)
	c.repo.Refs.UpdateHead(commit.Oid())
	c.writeCommit.PrintCommit(commit, c.stdout)
}
