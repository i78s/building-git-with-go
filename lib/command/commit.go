package command

import (
	"bufio"
	"building-git/lib/database"
	"building-git/lib/repository"
	"fmt"
	"io"
	"log"
	"os"
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
	c.repo.Index.Load()

	root := database.BuildTree(c.repo.Index.EachEntry())
	root.Traverse(func(t database.TreeObject) {
		if gitObj, ok := t.(database.GitObject); ok {
			if err := c.repo.Database.Store(gitObj); err != nil {
				log.Fatalf("Failed to store object: %v", err)
			}
		} else {
			log.Fatalf("Object does not implement GitObject interface")
		}
	})

	parent, _ := c.repo.Refs.ReadHead()
	name, exists := os.LookupEnv("GIT_AUTHOR_NAME")
	if !exists {
		fmt.Fprintf(c.stderr, "GIT_AUTHOR_NAME is not set")
		return 1
	}
	email, exists := os.LookupEnv("GIT_AUTHOR_EMAIL")
	if !exists {
		fmt.Fprintf(c.stderr, "GIT_AUTHOR_EMAIL is not set")
		return 1
	}

	author := database.NewAuthor(name, email, now)
	reader := bufio.NewReader(c.stdin)
	message, _ := reader.ReadString('\n')

	commit := database.NewCommit(parent, root.Oid(), author, message)
	c.repo.Database.Store(commit)
	c.repo.Refs.UpdateHead(commit.Oid())

	messageLines := strings.Split(message, "\n")
	isRoot := ""
	if root == nil {
		isRoot = "[(root-commit)]"
	}
	fmt.Fprintf(c.stdout, "%s%s %s\n", isRoot, commit.Oid(), messageLines[0])

	return 0
}
