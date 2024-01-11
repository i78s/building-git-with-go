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
	Combined bool
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
	revList     *repository.RevList
}

func NewLog(dir string, args []string, options LogOption, stdout, stderr io.Writer) (*Log, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRepository(rootPath)
	prindDiff, _ := print_diff.NewPrintDiff(dir, stdout, stderr)

	revList := repository.NewRevList(repo, args)

	return &Log{
		rootPath:  rootPath,
		args:      args,
		options:   options,
		repo:      repo,
		stdout:    stdout,
		stderr:    stderr,
		prindDiff: prindDiff,
		revList:   revList,
	}, nil
}

func (l *Log) Run() int {
	l.reverseRefs = l.repo.Refs.ReverseRefs()
	l.currentRef, _ = l.repo.Refs.CurrentRef("")

	blankLine := false
	for _, commit := range l.revList.Each() {
		l.showCommit(blankLine, commit)
		blankLine = true
	}

	return 0
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
		color.New(color.FgYellow).Sprintf("commit %s\n", l.abbrev(commit))+
			l.decorate(commit),
	)

	if commit.IsMerge() {
		oids := []string{}
		for _, oid := range commit.Parents {
			oids = append(oids, l.repo.Database.ShortOid(oid))
		}
		fmt.Fprintf(l.stdout, "Merge: %s\n", strings.Join(oids, " "))
	}

	fmt.Fprintf(l.stdout, "Author: %s <%s>\n", author.Name, author.Email)
	fmt.Fprintf(l.stdout, "Date:  %s\n", author.ReadableTime())
	fmt.Fprintf(l.stdout, "\n")

	lines := strings.Split(commit.Message(), "\n")
	if lines[len(lines)-1] == "" { // Replicate Ruby's String#lines
		lines = lines[:len(lines)-1]
	}
	for _, line := range lines {
		fmt.Fprintf(l.stdout, "    %s\n", line)
	}
}

func (l *Log) showCommitOneLine(commit *database.Commit) {
	id := fmt.Sprintf(
		color.New(color.FgYellow).Sprint(l.abbrev(commit)) +
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
	if commit.IsMerge() {
		l.showMergePatch(commit)
		return
	}

	if blankLine {
		fmt.Fprintf(l.stdout, "\n")
	}
	l.prindDiff.PrintCommitDiff(commit.Parent(), commit.Oid(), l.revList)
}

func (l *Log) showMergePatch(commit *database.Commit) {
	if !l.options.Combined {
		return
	}

	diffs := []map[string][2]database.TreeObject{}
	for _, oid := range commit.Parents {
		diffs = append(diffs, l.revList.TreeDiff(oid, commit.Oid(), nil))
	}

	var paths []string
	for path := range diffs[0] {
		allHavePath := true
		for _, diff := range diffs[1:] {
			if _, exists := diff[path]; !exists {
				allHavePath = false
				break
			}
		}
		if allHavePath {
			paths = append(paths, path)
		}
	}

	for _, path := range paths {
		parents := []*print_diff.Target{}
		for _, diff := range diffs {
			parents = append(parents, l.prindDiff.FromEntry(path, diff[path][0]))
		}
		child := l.prindDiff.FromEntry(path, diffs[0][path][1])
		l.prindDiff.PrintCombinedDiff(parents, child)
	}
}
