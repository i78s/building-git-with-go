package command

import (
	"building-git/lib/database"
	"building-git/lib/repository"
	"sort"
	"strings"

	"fmt"
	"io"
	"path/filepath"

	"github.com/fatih/color"
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
	Verbose bool
	Delete  bool
	Force   bool
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
	var err error
	if b.options.Delete {
		err = b.deleteBranches()
		if err != nil {
			return 1
		}
	} else if len(b.args) == 0 {
		b.listBranch()
	} else {
		err = b.createBranch()
	}

	if err != nil {
		return 128
	}
	return 0
}

func (b *Branch) listBranch() {
	current, _ := b.repo.Refs.CurrentRef("")
	branches, _ := b.repo.Refs.ListBranches()
	sort.Slice(branches, func(i, j int) bool {
		return branches[i].Path < branches[j].Path
	})
	maxWidth := 0
	for _, b := range branches {
		name, _ := b.ShortName()
		if maxWidth < len(name) {
			maxWidth = len(name)
		}
	}

	for _, ref := range branches {
		info := b.formatRef(ref, current)
		info += b.extendedBranchInfo(ref, maxWidth)
		fmt.Fprintf(b.stdout, "%s\n", info)
	}
}

func (b *Branch) formatRef(ref, current *repository.SymRef) string {
	shortName, _ := ref.ShortName()
	if ref.Path == current.Path {
		return "* " + color.GreenString(shortName)
	}
	return fmt.Sprintf("  %s", shortName)
}

func (b *Branch) extendedBranchInfo(ref *repository.SymRef, maxWidth int) string {
	if !b.options.Verbose {
		return ""
	}
	oid, _ := ref.ReadOid()
	commit, _ := b.repo.Database.Load(oid)
	short := b.repo.Database.ShortOid(commit.Oid())
	size, _ := ref.ShortName()
	space := strings.Repeat(" ", maxWidth-len(size))

	return fmt.Sprintf("%s %s %s", space, short, commit.(*database.Commit).TitleLine())
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

func (b *Branch) deleteBranches() error {
	for _, branch := range b.args {
		err := b.deleteBranch(branch)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Branch) deleteBranch(branchName string) error {
	if !b.options.Force {
		return nil
	}

	oid, err := b.repo.Refs.DeleteBranch(branchName)
	if err != nil {
		fmt.Fprintf(b.stderr, "error: branch '%s' not found.\n", branchName)
		return err
	}
	short := b.repo.Database.ShortOid(oid)
	fmt.Fprintf(b.stdout, "Deleted branch %s (was %s).\n", branchName, short)
	return nil
}
