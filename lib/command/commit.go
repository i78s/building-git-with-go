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

func Commit(dir string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintf(stderr, "fatal: %v", err)
		return 1
	}
	repo := repository.NewRepository(rootPath)

	repo.Index.Load()

	root := database.BuildTree(repo.Index.EachEntry())
	root.Traverse(func(t database.TreeObject) {
		if gitObj, ok := t.(database.GitObject); ok {
			if err := repo.Database.Store(gitObj); err != nil {
				log.Fatalf("Failed to store object: %v", err)
			}
		} else {
			log.Fatalf("Object does not implement GitObject interface")
		}
	})

	parent, _ := repo.Refs.ReadHead()
	name, exists := os.LookupEnv("GIT_AUTHOR_NAME")
	if !exists {
		fmt.Fprintf(stderr, "GIT_AUTHOR_NAME is not set")
		return 1
	}
	email, exists := os.LookupEnv("GIT_AUTHOR_EMAIL")
	if !exists {
		fmt.Fprintf(stderr, "GIT_AUTHOR_EMAIL is not set")
		return 1
	}

	author := database.NewAuthor(name, email, time.Now())
	reader := bufio.NewReader(stdin)
	message, _ := reader.ReadString('\n')

	commit := database.NewCommit(parent, root.Oid(), author.String(), message)
	repo.Database.Store(commit)
	repo.Refs.UpdateHead(commit.Oid())

	messageLines := strings.Split(message, "\n")
	isRoot := ""
	if root == nil {
		isRoot = "[(root-commit)]"
	}
	fmt.Fprintf(stdout, "%s%s %s\n", isRoot, commit.Oid(), messageLines[0])

	return 0
}
