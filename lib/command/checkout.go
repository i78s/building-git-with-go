package command

import (
	"building-git/lib/database"
	"building-git/lib/repository"
	"fmt"
	"io"
	"path/filepath"
)

type CheckOutOption struct {
}

type CheckOut struct {
	rootPath   string
	args       []string
	options    CheckOutOption
	repo       *repository.Repository
	stdout     io.Writer
	stderr     io.Writer
	target     string
	targetOid  string
	currentRef *repository.SymRef
	currentOid string
	newRef     *repository.SymRef
}

var DETACHED_HEAD_MESSAGE = `You are in 'detached HEAD' state. You can look around, make experimental
changes and commit them, and you can discard any commits you make in this
state without impacting any branches by performing another checkout.
If you want to create a new branch to retain commits you create, you may
do so (now or later) by using the branch command. Example:

	jit branch <new-branch-name>
`

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
	c.target = c.args[0]
	currentRef, _ := c.repo.Refs.CurrentRef("")
	currentOid, _ := currentRef.ReadOid()
	c.currentRef = currentRef
	c.currentOid = currentOid

	revision := repository.NewRevision(c.repo, c.target)
	targetOid, err := revision.Resolve(repository.COMMIT)
	c.targetOid = targetOid

	c.repo.Index.LoadForUpdate()

	if err != nil {
		c.handleInvalidObject(revision, err)
		return 1
	}

	treeDiff := c.repo.Database.TreeDiff(currentOid, targetOid, nil)
	migration := c.repo.Migration(treeDiff)
	err = migration.ApplyChanges()
	if err != nil {
		c.handleMigrationConflict(migration)
		return 1
	}

	c.repo.Index.WriteUpdates()
	c.repo.Refs.SetHead(c.target, targetOid)
	newRef, _ := c.repo.Refs.CurrentRef("")
	c.newRef = newRef

	c.printPreviousHead()
	c.printDetachmentNotice()
	c.printNewHead()

	return 0
}

func (c *CheckOut) handleMigrationConflict(migration *repository.Migration) {
	c.repo.Index.ReleaseLock()

	for _, msg := range migration.Errors {
		fmt.Fprintf(c.stderr, "error: %v\n", msg)
	}
	fmt.Fprintf(c.stderr, "Aborting")
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

func (c *CheckOut) printPreviousHead() {
	if c.currentRef.IsHead() && c.currentOid != c.targetOid {
		c.printHeadPosition("Previous HEAD position was", c.currentOid)
	}
}

func (c *CheckOut) printDetachmentNotice() {
	if !(c.newRef.IsHead() && !c.currentRef.IsHead()) {
		return
	}
	fmt.Fprintf(c.stderr, "Note: checking out '%s'.\n", c.target)
	fmt.Fprintf(c.stderr, "%s", DETACHED_HEAD_MESSAGE)
}

func (c *CheckOut) printNewHead() {
	if c.newRef.IsHead() {
		c.printHeadPosition("HEAD is now at", c.targetOid)
		return
	}
	if c.newRef.Path == c.currentRef.Path {
		fmt.Fprintf(c.stderr, "Already on '%s'\n", c.target)
		return
	}
	fmt.Fprintf(c.stderr, "Switched to branch '%s'\n", c.target)
}

func (c *CheckOut) printHeadPosition(message, oid string) {
	commit, _ := c.repo.Database.Load(oid)
	short := c.repo.Database.ShortOid(commit.Oid())

	fmt.Fprintf(c.stderr, "%s %s %s\n", message, short, commit.(*database.Commit).TitleLine())
}
