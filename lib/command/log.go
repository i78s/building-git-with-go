package command

import (
	"building-git/lib/database"
	"building-git/lib/repository"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

type LogOption struct {
	Abbrev bool
}

type Log struct {
	rootPath string
	args     []string
	options  LogOption
	repo     *repository.Repository
	stdout   io.Writer
	stderr   io.Writer
}

func NewLog(dir string, args []string, options LogOption, stdout, stderr io.Writer) (*Log, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRepository(rootPath)

	return &Log{
		rootPath: rootPath,
		args:     args,
		options:  options,
		repo:     repo,
		stdout:   stdout,
		stderr:   stderr,
	}, nil
}

func (l *Log) Run() int {
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
	author := commit.Author()

	if blankLine {
		fmt.Fprintf(l.stdout, "\n")
	}
	color.New(color.FgYellow).Fprintf(l.stdout, "commit %s\n", l.abbrev(commit))
	fmt.Fprintf(l.stdout, "Author: %s <%s>\n", author.Name, author.Email)
	fmt.Fprintf(l.stdout, "Date:  %s\n", author.ReadableTime())
	fmt.Fprintf(l.stdout, "\n")
	for _, line := range strings.Split(commit.Message(), "\n") {
		fmt.Fprintf(l.stdout, "    %s\n", line)
	}
}

func (l *Log) abbrev(commit *database.Commit) string {
	if l.options.Abbrev {
		return l.repo.Database.ShortOid(commit.Oid())
	}
	return commit.Oid()
}
