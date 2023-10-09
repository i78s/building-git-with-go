package print_diff

import (
	"building-git/lib/database"
	"building-git/lib/diff"
	"building-git/lib/repository"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/fatih/color"
)

const (
	NULL_OID  = "0000000000000000000000000000000000000000"
	NULL_PATH = "/dev/null"
)

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

type PrintDiff struct {
	rootPath string
	repo     *repository.Repository
	stdout   io.Writer
	stderr   io.Writer
}

func NewPrintDiff(dir string, stdout, stderr io.Writer) (*PrintDiff, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRepository(rootPath)

	return &PrintDiff{
		rootPath: rootPath,
		repo:     repo,
		stdout:   stdout,
		stderr:   stderr,
	}, nil
}

func (p *PrintDiff) FromEntry(path string, entry database.TreeObject) *Target {
	if entry == nil || entry.IsNil() {
		return p.FromNothing(path)
	}

	aOid := entry.Oid()
	aMode := fmt.Sprintf("%o", entry.Mode())
	blob, _ := p.repo.Database.Load(aOid)

	return NewTarget(path, aOid, aMode, blob.String())
}

func (p *PrintDiff) FromNothing(path string) *Target {
	return NewTarget(path, NULL_OID, "", "")
}

func (p *PrintDiff) short(oid string) string {
	return p.repo.Database.ShortOid(oid)
}

type Differ interface {
	TreeDiff(a, b string, differ *database.PathFilter) map[string][2]database.TreeObject
}

func (p *PrintDiff) PrintCommitDiff(a, b string, differ Differ) {
	if differ == nil {
		differ = p.repo.Database
	}
	diff := differ.TreeDiff(a, b, nil)
	paths := []string{}
	for k := range diff {
		paths = append(paths, k)
	}
	sort.Strings(paths)

	for _, path := range paths {
		oldEntry, newEntry := diff[path][0], diff[path][1]
		p.PrintDiff(p.FromEntry(path, oldEntry), p.FromEntry(path, newEntry))
	}
}

func (p *PrintDiff) PrintDiff(a, b *Target) {
	if a.oid == b.oid && a.mode == b.mode {
		return
	}

	a.path = filepath.Join("a", a.path)
	b.path = filepath.Join("b", b.path)

	fmt.Fprintf(p.stdout, "diff --git %s %s\n", a.path, b.path)
	p.printDiffMode(a, b)
	p.printDiffContent(a, b)
}

func (p *PrintDiff) printDiffMode(a, b *Target) {
	if a.mode == "" {
		color.New(color.Bold).Fprintf(p.stdout, "new file mode %s\n", b.mode)
		return
	}
	if b.mode == "" {
		color.New(color.Bold).Fprintf(p.stdout, "deleted file mode %s\n", a.mode)
		return
	}
	if a.mode != b.mode {
		color.New(color.Bold).Fprintf(p.stdout, "old mode %s\n", a.mode)
		color.New(color.Bold).Fprintf(p.stdout, "new mode %s\n", b.mode)
	}
}

func (p *PrintDiff) printDiffContent(a, b *Target) {
	if a.oid == b.oid {
		return
	}

	oidRange := fmt.Sprintf("index %s..%s", p.short(a.oid), p.short(b.oid))
	if a.mode == b.mode {
		oidRange += fmt.Sprintf(" %s", a.mode)
	}
	fmt.Fprintf(p.stdout, "%s\n", oidRange)
	fmt.Fprintf(p.stdout, "--- %s\n", a.diffPath())
	fmt.Fprintf(p.stdout, "+++ %s\n", b.diffPath())

	hunks := diff.DiffHunk(a.data, b.data)
	for _, hunk := range hunks {
		p.printDiffHunk(hunk)
	}
}

func (p *PrintDiff) printDiffHunk(hunk *diff.Hunk) {
	color.New(color.FgCyan).Fprintf(p.stdout, "%s\n", hunk.Header())

	for _, edit := range hunk.Edits {
		text := strings.TrimRightFunc(edit.String(), unicode.IsSpace)

		switch edit.Type {
		case diff.EQL:
			fmt.Fprintf(p.stdout, "%s\n", text)
		case diff.INS:
			color.New(color.FgGreen).Fprintf(p.stdout, "%s\n", text)
		case diff.DEL:
			color.New(color.FgRed).Fprintf(p.stdout, "%s\n", text)
		}
	}
}
