package command

import (
	"building-git/lib/database"
	"building-git/lib/diff"
	"building-git/lib/index"
	"building-git/lib/repository"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/fatih/color"
)

const (
	NULL_OID  = "0000000000000000000000000000000000000000"
	NULL_PATH = "/dev/null"
)

type Diff struct {
	rootPath string
	args     []string
	options  DiffOption
	repo     *repository.Repository
	status   *repository.Status
	stdout   io.Writer
	stderr   io.Writer
}

type DiffOption struct {
	Cached bool
}

func NewDiff(dir string, args []string, options DiffOption, stdout, stderr io.Writer) (*Diff, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRepository(rootPath)

	return &Diff{
		rootPath: rootPath,
		args:     args,
		options:  options,
		repo:     repo,
		stdout:   stdout,
		stderr:   stderr,
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
	} else {
		d.diffIndexWorkspace()
	}

	return 0
}

func (d *Diff) diffHeadIndex() {
	d.status.IndexChanges.Iterate(func(path string, state repository.ChangeType) {
		switch state {
		case repository.Added:
			d.printDiff(d.fromNothing(path), d.fromIndex(path))
			break
		case repository.Modified:
			d.printDiff(d.fromHead(path), d.fromIndex(path))
			break
		case repository.Deleted:
			d.printDiff(d.fromHead(path), d.fromNothing(path))
			break
		}
	})
}

func (d *Diff) diffIndexWorkspace() {
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
}

func (d *Diff) fromHead(path string) *Target {
	entry := d.status.HeadTree[path]
	aOid := entry.Oid()
	aMode := fmt.Sprintf("%o", entry.Mode())
	blob, _ := d.repo.Database.Load(aOid)

	return NewTarget(path, aOid, aMode, blob.String())
}

func (d *Diff) fromIndex(path string) *Target {
	entry := d.repo.Index.EntryForPath(path)
	aOid := entry.Oid()
	aMode := fmt.Sprintf("%o", entry.Mode())
	blob, _ := d.repo.Database.Load(aOid)

	return NewTarget(path, aOid, aMode, blob.String())
}

func (d *Diff) fromFile(path string) *Target {
	file, _ := d.repo.Workspace.ReadFile(path)
	blob := database.NewBlob(file)
	bOid, _ := d.repo.Database.HashObject(blob)

	stat, _ := d.status.Stats[path]
	bMode := fmt.Sprintf("%o", index.ModeForStat(stat))

	return NewTarget(path, bOid, bMode, blob.String())
}

func (d *Diff) fromNothing(path string) *Target {
	return NewTarget(path, NULL_OID, "", "")
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
	d.printDiffMode(a, b)
	d.printDiffContent(a, b)
}

func (d *Diff) printDiffMode(a, b *Target) {
	if a.mode == "" {
		color.New(color.Bold).Fprintf(d.stdout, "new file mode %s\n", b.mode)
		return
	}
	if b.mode == "" {
		color.New(color.Bold).Fprintf(d.stdout, "deleted file mode %s\n", a.mode)
		return
	}
	if a.mode != b.mode {
		color.New(color.Bold).Fprintf(d.stdout, "old mode %s\n", a.mode)
		color.New(color.Bold).Fprintf(d.stdout, "new mode %s\n", b.mode)
	}
}

func (d *Diff) printDiffContent(a, b *Target) {
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

	hunks := diff.DiffHunk(a.data, b.data)
	for _, hunk := range hunks {
		d.printDiffHunk(hunk)
	}
}

func (d *Diff) printDiffHunk(hunk *diff.Hunk) {
	color.New(color.FgCyan).Fprintf(d.stdout, "%s\n", hunk.Header())

	for _, edit := range hunk.Edits {
		text := strings.TrimRightFunc(edit.String(), unicode.IsSpace)

		switch edit.Type {
		case diff.EQL:
			fmt.Fprintf(d.stdout, "%s\n", text)
			break
		case diff.INS:
			color.New(color.FgGreen).Fprintf(d.stdout, "%s\n", text)
			break
		case diff.DEL:
			color.New(color.FgRed).Fprintf(d.stdout, "%s\n", text)
			break
		}
	}
}

type Target struct {
	path string
	oid  string
	mode string
	data string
}

func NewTarget(path, oid, mode, data string) *Target {
	return &Target{
		path: path,
		oid:  oid,
		mode: mode,
		data: data,
	}
}

func (t *Target) diffPath() string {
	if t.mode == "" {
		return NULL_PATH
	}
	return t.path
}
