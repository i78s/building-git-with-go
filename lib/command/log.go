package command

import (
	"building-git/lib/command/print_diff"
	"building-git/lib/database"
	"building-git/lib/repository"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

type LogOption struct {
	Abbrev   bool
	Format   string
	Decorate string
	IsTty    bool
	Patch    bool
}

type Log struct {
	rootPath    string
	args        []string
	options     LogOption
	repo        *repository.Repository
	stdout      io.Writer
	stderr      io.Writer
	currentRef  *repository.SymRef
	reverseRefs map[string][]*repository.SymRef
	prindDiff   *print_diff.PrintDiff
}

func NewLog(dir string, args []string, options LogOption, stdout, stderr io.Writer) (*Log, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRepository(rootPath)
	prindDiff, _ := print_diff.NewPrintDiff(dir, stdout, stderr)

	return &Log{
		rootPath:  rootPath,
		args:      args,
		options:   options,
		repo:      repo,
		stdout:    stdout,
		stderr:    stderr,
		prindDiff: prindDiff,
	}, nil
}

func (l *Log) Run() int {
	l.reverseRefs = l.repo.Refs.ReverseRefs()
	l.currentRef, _ = l.repo.Refs.CurrentRef("")

	blankLine := false
	for commit := range l.eachCommit() {
		l.showCommit(blankLine, commit)
		blankLine = true
	}

	return 0
}

func (l *Log) eachCommit() chan *database.Commit {
	ch := make(chan *database.Commit)

	go func() {
		oid, _ := l.repo.Refs.ReadHead()

		for oid != "" {
			commitObj, _ := l.repo.Database.Load(oid)
			commit := commitObj.(*database.Commit)
			ch <- commit
			oid = commit.Parent()
		}
		close(ch)
	}()

	return ch
}

func (l *Log) showCommit(blankLine bool, commit *database.Commit) {
	switch l.options.Format {
	case "", "medium":
		l.showCommitMedium(blankLine, commit)
		blankLine = false
	case "oneline":
		l.showCommitOneLine(commit)
		blankLine = false
	}

	l.showPatch(blankLine, commit)
}

func (l *Log) showCommitMedium(blankLine bool, commit *database.Commit) {
	author := commit.Author()

	if l.options.Format != "oneline" && blankLine {
		fmt.Fprintf(l.stdout, "\n")
	}
	fmt.Fprintf(l.stdout,
		color.New(color.FgYellow).Sprintf("commit %s", l.abbrev(commit))+
			l.decorate(commit),
	)

	fmt.Fprintf(l.stdout, "\n")

	fmt.Fprintf(l.stdout, "Author: %s <%s>\n", author.Name, author.Email)
	fmt.Fprintf(l.stdout, "Date:  %s\n", author.ReadableTime())
	fmt.Fprintf(l.stdout, "\n")
	for _, line := range strings.Split(commit.Message(), "\n") {
		fmt.Fprintf(l.stdout, "    %s\n", line)
	}
}

func (l *Log) showCommitOneLine(commit *database.Commit) {
	id := fmt.Sprintf(
		color.New(color.FgYellow).Sprintf("commit %s", l.abbrev(commit)) +
			l.decorate(commit),
	)
	fmt.Fprintf(l.stdout, "%s %s\n", id, commit.TitleLine())
}

func (l *Log) abbrev(commit *database.Commit) string {
	if l.options.Abbrev {
		return l.repo.Database.ShortOid(commit.Oid())
	}
	return commit.Oid()
}

func (l *Log) decorate(commit *database.Commit) string {
	switch l.options.Decorate {
	case "auto":
		if !l.options.IsTty {
			return ""
		}
	case "no":
		return ""
	}

	refs, ok := l.reverseRefs[commit.Oid()]
	if !ok {
		return ""
	}

	var head *repository.SymRef
	var otherRefs []*repository.SymRef
	for _, ref := range refs {
		if ref.IsHead() && !l.currentRef.IsHead() {
			head = ref
		} else {
			otherRefs = append(otherRefs, ref)
		}
	}

	var names []string
	for _, ref := range otherRefs {
		names = append(names, l.decorationName(head, ref))
	}

	return fmt.Sprint(
		color.New(color.FgYellow).Sprint(" ("),
		strings.Join(names, color.New(color.FgYellow).Sprint(", ")),
		color.New(color.FgYellow).Sprint(")"),
	)
}

func (l *Log) decorationName(head, ref *repository.SymRef) string {
	var name string
	switch l.options.Decorate {
	case "short", "auto":
		name, _ = ref.ShortName()
	case "full":
		name = ref.Path
	}

	name = l.refColor(ref)(name)

	if head != nil && ref.Path == l.currentRef.Path {
		name = fmt.Sprintf("%s -> %s", head.Path, name)
		name = l.refColor(head)(name)
	}

	return name
}

func (l *Log) refColor(ref *repository.SymRef) func(a ...interface{}) string {
	if ref.IsHead() {
		return color.New(color.FgCyan, color.Bold).SprintFunc()
	}
	return color.New(color.FgGreen, color.Bold).SprintFunc()
}

func (l *Log) showPatch(blankLine bool, commit *database.Commit) {
	if !l.options.Patch {
		return
	}

	if blankLine {
		fmt.Fprintf(l.stdout, "\n")
	}
	l.prindDiff.PrintCommitDiff(commit.Parent(), commit.Oid())
}
