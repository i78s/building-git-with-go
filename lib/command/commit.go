package command

import (
	"bufio"
	"building-git/lib/command/write_commit"
	"building-git/lib/repository"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"
)

type CommitOption struct {
}

type Commit struct {
	rootPath string
	args     []string
	options  CommitOption
	repo     *repository.Repository
	stdin    io.Reader
	stdout   io.Writer
	stderr   io.Writer
}

func NewCommit(dir string, args []string, options CommitOption, stdin io.Reader, stdout, stderr io.Writer) (*Commit, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRepository(rootPath)

	return &Commit{
		rootPath: rootPath,
		args:     args,
		options:  options,
		repo:     repo,
		stdin:    stdin,
		stdout:   stdout,
		stderr:   stderr,
	}, nil
}

func (c *Commit) Run(now time.Time) int {
	writeCommit := write_commit.NewWriteCommit(c.repo)
	c.repo.Index.Load()

	if writeCommit.PendingCommit().InProgress() {
		err := writeCommit.ResumeMerge()
		if err != nil {
			fmt.Fprintf(c.stderr, "%s", err.Error())
			return 128
		}
		return 0
	}

	parent, _ := c.repo.Refs.ReadHead()
	reader := bufio.NewReader(c.stdin)
	message, _ := reader.ReadString('\n')

	parents := []string{}
	if parent != "" {
		parents = append(parents, parent)
	}

	commit := writeCommit.WriteCommit(parents, message, now)

	messageLines := strings.Split(message, "\n")
	isRoot := ""
	if parent == "" {
		isRoot = "[(root-commit)]"
	}
	fmt.Fprintf(c.stdout, "%s%s %s\n", isRoot, commit.Oid(), messageLines[0])

	return 0
}
