package command

import (
	"bufio"
	"building-git/lib"
	"building-git/lib/database"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Commit(args []string) {
	path, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	rootPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	repo := lib.NewRepository(rootPath)

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
		fmt.Println("GIT_AUTHOR_NAME is not set")
		os.Exit(1)
	}

	email, exists := os.LookupEnv("GIT_AUTHOR_EMAIL")
	if !exists {
		fmt.Println("GIT_AUTHOR_EMAIL is not set")
		os.Exit(1)
	}

	author := database.NewAuthor(name, email, time.Now())
	reader := bufio.NewReader(os.Stdin)
	message, _ := reader.ReadString('\n')

	commit := database.NewCommit(parent, root.GetOid(), author, message)
	repo.Database.Store(commit)
	repo.Refs.UpdateHead(commit.GetOid())

	f, err := os.OpenFile(filepath.Join(filepath.Join(rootPath, ".git"), "HEAD"), os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	_, err = f.WriteString(commit.GetOid() + "\n")
	if err != nil {
		log.Fatal(err)
	}

	messageLines := strings.Split(message, "\n")
	isRoot := ""
	if root == nil {
		isRoot = "[(root-commit)]"
	}
	fmt.Printf("%s%s] %s\n", isRoot, commit.GetOid(), messageLines[0])

	os.Exit(0)
}
