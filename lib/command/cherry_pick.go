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
	"strings"
	"time"
)

type CherryPickOption struct {
	Mode      MergeMode
	EditorCmd func(path string) editor.Executable
	IsTTY     bool
}

type CherryPick struct {
	rootPath    string
	args        []string
	options     CherryPickOption
	repo        *repository.Repository
	writeCommit *write_commit.WriteCommit
	sequencer   *repository.Sequencer
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
	sequencer := repository.NewSequencer(repo)
	return &CherryPick{
		rootPath:    rootPath,
		args:        args,
		options:     options,
		repo:        repo,
		writeCommit: writeCommit,
		sequencer:   sequencer,
		stdout:      stdout,
		stderr:      stderr,
	}, nil
}

func (c *CherryPick) Run() int {
	if c.options.Mode == Continue {
		err := c.handleContinue()
		if err != nil {
			if _, ok := err.(*repository.PendingCommitError); ok {
				fmt.Fprintf(c.stderr, "fatal: %v", err)
			} else {
				fmt.Fprint(c.stderr, err)
			}
			return 128
		}
		return 0
	}

	c.sequencer.Start()
	c.storeCommitSequence()
	err := c.resumeSequencer()
	if err != nil {
		return 1
	}
	return 0
}

func (c *CherryPick) storeCommitSequence() {
	for i := 0; i < len(c.args)/2; i++ {
		c.args[i], c.args[len(c.args)-i-1] = c.args[len(c.args)-i-1], c.args[i]
	}

	walk := false
	commits := repository.NewRevList(c.repo, c.args, repository.RevListOption{Walk: &walk})
	for _, commit := range commits.ReverseEach() {
		c.sequencer.Pick(commit)
	}
}

func (c *CherryPick) mergeType() repository.MergeType {
	return repository.CherryPick
}

func (c *CherryPick) pick(commit *database.Commit) error {
	inputs := c.pickMergeInputs(commit)
	err := c.resolveMerge(inputs)
	if err != nil {
		return err
	}

	if c.repo.Index.IsConflict() {
		c.failOnConflict(inputs, commit.Message())
		return fmt.Errorf("detect conflict")
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

func (c *CherryPick) failOnConflict(inputs merge.ResolveInputs, message string) {
	c.sequencer.Dump()
	pendingCommit := c.writeCommit.PendingCommit()
	pendingCommit.Start(inputs.RightOid(), c.mergeType())

	path := pendingCommit.MessagePath
	editor.EditFile(path, c.options.EditorCmd(path), c.options.IsTTY, func(e *editor.Editor) {
		e.Puts(message)
		e.Puts("")
		e.Note("Conflicts:")
		for name := range c.repo.Index.ConflictPaths() {
			e.Note("\t" + name)
		}
		e.Close()
	})

	fmt.Fprintf(c.stdout, "error: could not apply %s\n", inputs.RightName())
	for _, line := range strings.Split(write_commit.CONFLICT_NOTES, "\n") {
		fmt.Fprintf(c.stdout, "hint: %s\n", line)
	}
}

func (c *CherryPick) finishCommit(commit *database.Commit) {
	c.repo.Database.Store(commit)
	c.repo.Refs.UpdateHead(commit.Oid())
	c.writeCommit.PrintCommit(commit, c.stdout)
}

func (c *CherryPick) handleContinue() error {
	c.repo.Index.Load()
	if c.writeCommit.PendingCommit().InProgress() {
		err := c.writeCommit.WriteCherryPickCommit(c.options.IsTTY)
		if err != nil {
			return err
		}
	}
	c.sequencer.Load()
	c.sequencer.DropCommand()
	return c.resumeSequencer()
}

func (c *CherryPick) resumeSequencer() error {
	for {
		commit := c.sequencer.NextCommand()
		if commit == nil {
			break
		}
		err := c.pick(commit)
		if err != nil {
			return err
		}
		c.sequencer.DropCommand()
	}
	c.sequencer.Quit()
	return nil
}
