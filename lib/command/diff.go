package command

import (
	"building-git/lib/database"
	"building-git/lib/index"
	"building-git/lib/repository"
	"fmt"
	"io"
	"path/filepath"
)

const (
	NULL_OID  = "0000000000000000000000000000000000000000"
	NULL_PATH = "/dev/null"
)

type Diff struct {
	rootPath string
	args     StatusOption
	repo     *repository.Repository
	status   *repository.Status
	stdout   io.Writer
	stderr   io.Writer
}

func NewDiff(dir string, args StatusOption, stdout, stderr io.Writer) (*Diff, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRepository(rootPath)

	return &Diff{
		rootPath: rootPath,
		args:     args,
		repo:     repo,
		stdout:   stdout,
		stderr:   stderr,
	}, nil
}

func (d *Diff) Run() int {
	d.repo.Index.LoadForUpdate()

	status, err := d.repo.Status()
	d.status = status
	if err != nil {
		fmt.Fprintf(d.stderr, "fatal: %v", err)
		return 128
	}

	d.status.WorkspaceChanges.Iterate(func(path string, state repository.ChangeType) {
		switch state {
		case repository.Modified:
			d.printDiff(d.fromIndex(path), d.fromFile(path))
			break
		case repository.Deleted:
			d.printDiff(d.fromIndex(path), d.fromNothing(path))
			break
		}
	})

	return 0
}

func (d *Diff) fromIndex(path string) *Target {
	entry := d.repo.Index.EntryForPath(path)
	aOid := entry.Oid()
	aMode := fmt.Sprintf("%o", entry.Mode())

	return NewTarget(path, aOid, aMode)
}

func (d *Diff) fromFile(path string) *Target {
	file, _ := d.repo.Workspace.ReadFile(path)
	blob := database.NewBlob(file)
	bOid, _ := d.repo.Database.HashObject(blob)

	stat, _ := d.status.Stats[path]
	bMode := fmt.Sprintf("%o", index.ModeForStat(stat))

	return NewTarget(path, bOid, bMode)
}

func (d *Diff) fromNothing(path string) *Target {
	return NewTarget(path, NULL_OID, "")
}

func (d *Diff) short(oid string) string {
	return d.repo.Database.ShortOid(oid)
}

func (d *Diff) printDiff(a, b *Target) {
	if a.oid == b.oid && a.mode == b.mode {
		return
	}

	a.path = filepath.Join("a", a.path)
	b.path = filepath.Join("b", b.path)

	fmt.Fprintf(d.stdout, "diff --git %s %s\n", a.path, b.path)
	d.printMode(a, b)
	d.printContent(a, b)
}

func (d *Diff) printMode(a, b *Target) {
	if b.mode == "" {
		fmt.Fprintf(d.stdout, "deleted file mode %s\n", a.mode)
		return
	}
	if a.mode != b.mode {
		fmt.Fprintf(d.stdout, "old mode %s\n", a.mode)
		fmt.Fprintf(d.stdout, "new mode %s\n", b.mode)
	}
}

func (d *Diff) printContent(a, b *Target) {
	if a.oid == b.oid {
		return
	}

	oidRange := fmt.Sprintf("index %s..%s", d.short(a.oid), d.short(b.oid))
	if a.mode == b.mode {
		oidRange += fmt.Sprintf(" %s", a.mode)
	}
	fmt.Fprintf(d.stdout, "%s\n", oidRange)
	fmt.Fprintf(d.stdout, "--- %s\n", a.diffPath())
	fmt.Fprintf(d.stdout, "+++ %s\n", b.diffPath())
}

type Target struct {
	path string
	oid  string
	mode string
}

func NewTarget(path, oid, mode string) *Target {
	return &Target{
		path: path,
		oid:  oid,
		mode: mode,
	}
}

func (t *Target) diffPath() string {
	if t.mode == "" {
		return NULL_PATH
	}
	return t.path
}
