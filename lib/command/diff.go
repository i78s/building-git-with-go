package command

import (
	"building-git/lib/command/print_diff"
	"building-git/lib/database"
	"building-git/lib/index"
	"building-git/lib/repository"
	"fmt"
	"io"
	"path/filepath"
)

type Diff struct {
	rootPath  string
	args      []string
	options   DiffOption
	repo      *repository.Repository
	status    *repository.Status
	stdout    io.Writer
	stderr    io.Writer
	prindDiff *print_diff.PrintDiff
}

type DiffOption struct {
	Cached bool
	Patch  bool
}

func NewDiff(dir string, args []string, options DiffOption, stdout, stderr io.Writer) (*Diff, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRepository(rootPath)
	prindDiff, _ := print_diff.NewPrintDiff(dir, stdout, stderr)

	return &Diff{
		rootPath:  rootPath,
		args:      args,
		options:   options,
		repo:      repo,
		stdout:    stdout,
		stderr:    stderr,
		prindDiff: prindDiff,
	}, nil
}

func (d *Diff) Run() int {
	d.repo.Index.Load()

	status, err := d.repo.Status()
	d.status = status
	if err != nil {
		fmt.Fprintf(d.stderr, "fatal: %v", err)
		return 128
	}

	if d.options.Cached {
		d.diffHeadIndex()
	} else if len(d.args) == 2 {
		d.diffCommits()
	} else {
		d.diffIndexWorkspace()
	}

	return 0
}

func (d *Diff) diffCommits() {
	if !d.options.Patch {
		return
	}

	a, _ := repository.NewRevision(d.repo, d.args[0]).Resolve(repository.COMMIT)
	b, _ := repository.NewRevision(d.repo, d.args[1]).Resolve(repository.COMMIT)
	d.prindDiff.PrintCommitDiff(a, b, d.repo.Database)
}

func (d *Diff) diffHeadIndex() {
	if !d.options.Patch {
		return
	}
	d.status.IndexChanges.Iterate(func(path string, state repository.ChangeType) {
		switch state {
		case repository.Added:
			d.prindDiff.PrintDiff(d.prindDiff.FromNothing(path), d.fromIndex(path))
		case repository.Modified:
			d.prindDiff.PrintDiff(d.fromHead(path), d.fromIndex(path))
		case repository.Deleted:
			d.prindDiff.PrintDiff(d.fromHead(path), d.prindDiff.FromNothing(path))
		}
	})
}

func (d *Diff) diffIndexWorkspace() {
	if !d.options.Patch {
		return
	}

	paths := append(d.status.Conflicts.Keys, d.status.WorkspaceChanges.Keys...)
	for _, path := range paths {
		if _, exists := d.status.Conflicts.Get(path); exists {
			d.printConflictDiff(path)
		} else {
			d.printWorkspaceDiff(path)
		}
	}
}

func (d *Diff) printConflictDiff(path string) {
	fmt.Fprintf(d.stdout, "* Unmerged path %v\n", path)
}

func (d *Diff) printWorkspaceDiff(path string) {

	switch state, _ := d.status.WorkspaceChanges.Get(path); state {
	case repository.Modified:
		d.prindDiff.PrintDiff(d.fromIndex(path), d.fromFile(path))
	case repository.Deleted:
		d.prindDiff.PrintDiff(d.fromIndex(path), d.prindDiff.FromNothing(path))
	}
}

func (d *Diff) fromHead(path string) *print_diff.Target {
	entry := d.status.HeadTree[path]
	aOid := entry.Oid()
	aMode := fmt.Sprintf("%o", entry.Mode())
	blob, _ := d.repo.Database.Load(aOid)

	return print_diff.NewTarget(path, aOid, aMode, blob.String())
}

func (d *Diff) fromIndex(path string) *print_diff.Target {
	entry := d.repo.Index.EntryForPath(path)
	return d.prindDiff.FromEntry(path, entry)
}

func (d *Diff) fromFile(path string) *print_diff.Target {
	file, _ := d.repo.Workspace.ReadFile(path)
	blob := database.NewBlob(file)
	bOid, _ := d.repo.Database.HashObject(blob)

	stat := d.status.Stats[path]
	bMode := fmt.Sprintf("%o", index.ModeForStat(stat))

	return print_diff.NewTarget(path, bOid, bMode, blob.String())
}
