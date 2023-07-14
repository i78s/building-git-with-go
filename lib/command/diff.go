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
			d.diffFileModified(path)
			break
		case repository.Deleted:
			d.diffFileDeleted(path)
			break
		}

	})

	return 0
}

func (d *Diff) diffFileModified(path string) {
	entry := d.repo.Index.EntryForPath(path)
	aOid := entry.Oid()
	aMode := fmt.Sprintf("%o", entry.Mode())
	aPath := filepath.Join("a", path)

	file, _ := d.repo.Workspace.ReadFile(path)
	blob := database.NewBlob(file)
	bOid, _ := d.repo.Database.HashObject(blob)

	stat, _ := d.status.Stats[path]
	bMode := fmt.Sprintf("%o", index.ModeForStat(stat))
	bPath := filepath.Join("b", path)

	fmt.Fprintf(d.stdout, "diff --git %s %s\n", aPath, bPath)

	if aMode != bMode {
		fmt.Fprintf(d.stdout, "old mode %s\n", aMode)
		fmt.Fprintf(d.stdout, "new mode %s\n", bMode)
	}

	if aOid == bOid {
		return
	}

	oidRange := fmt.Sprintf("index %s..%s", d.short(aOid), d.short(bOid))
	if aMode == bMode {
		oidRange += fmt.Sprintf(" %s", aMode)
	}
	fmt.Fprintf(d.stdout, "%s\n", oidRange)
	fmt.Fprintf(d.stdout, "--- %s\n", aPath)
	fmt.Fprintf(d.stdout, "+++ %s\n", bPath)
}

func (d *Diff) diffFileDeleted(path string) {
	entry := d.repo.Index.EntryForPath(path)
	aOid := entry.Oid()
	aMode := fmt.Sprintf("%o", entry.Mode())
	aPath := filepath.Join("a", path)

	bOid := NULL_OID
	bPath := filepath.Join("b", path)

	fmt.Fprintf(d.stdout, "diff --git %s %s\n", aPath, bPath)
	fmt.Fprintf(d.stdout, "deleted file mode %s\n", aMode)
	fmt.Fprintf(d.stdout, "index %s..%s\n", d.short(aOid), d.short(bOid))
	fmt.Fprintf(d.stdout, "--- %s\n", aPath)
	fmt.Fprintf(d.stdout, "+++ %s\n", NULL_PATH)
}

func (d *Diff) short(oid string) string {
	return d.repo.Database.ShortOid(oid)
}
