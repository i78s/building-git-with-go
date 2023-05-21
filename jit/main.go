package main

import (
	"bufio"
	"building-git/jit/author"
	"building-git/jit/blob"
	"building-git/jit/commit"
	"building-git/jit/database"
	"building-git/jit/entry"
	"building-git/jit/refs"
	"building-git/jit/tree"
	"building-git/jit/workspace"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "jit: '%s' is not a jit command.\n", os.Args[1])
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "init":
		path := ""
		if len(os.Args) > 2 {
			path = os.Args[2]
		}
		if path == "" {
			var err error
			path, err = os.Getwd()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}

		rootPath, err := filepath.Abs(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		gitPath := filepath.Join(rootPath, ".git")
		dirs := []string{"objects", "refs"}

		for _, dir := range dirs {
			err := os.MkdirAll(filepath.Join(gitPath, dir), os.ModePerm)
			if err != nil {
				fmt.Fprintf(os.Stderr, "fatal: %s\n", err.Error())
				os.Exit(1)
			}
		}

		fmt.Printf("Initialized empty Jit repository in %s\n", gitPath)
	case "commit":
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
		gitPath := filepath.Join(rootPath, ".git")
		dbPath := filepath.Join(gitPath, "objects")

		ws := workspace.NewWorkspace(rootPath)
		db := database.NewDatabase(dbPath)
		r := refs.NewRefs(gitPath)
		entries := make([]*entry.Entry, 0)
		files, _ := ws.ListFiles()
		for _, path := range files {
			data, _ := ws.ReadFile(path)
			b := blob.NewBlob(data)
			db.Store(b)
			stat, _ := ws.StatFile(path)

			entries = append(entries, entry.NewEntry(path, b.GetOid(), stat))
		}

		root := tree.Build(entries)
		root.Traverse(func(t tree.TreeObject) {
			if gitObj, ok := t.(database.GitObject); ok {
				if err := db.Store(gitObj); err != nil {
					log.Fatalf("Failed to store object: %v", err)
				}
			} else {
				log.Fatalf("Object does not implement GitObject interface")
			}
		})

		parent, _ := r.ReadHead()
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

		a := author.NewAuthor(name, email, time.Now())

		reader := bufio.NewReader(os.Stdin)
		message, _ := reader.ReadString('\n')

		c := commit.NewCommit(parent, root.GetOid(), a, message)
		db.Store(c)

		f, err := os.OpenFile(filepath.Join(gitPath, "HEAD"), os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		_, err = f.WriteString(c.GetOid() + "\n")
		if err != nil {
			log.Fatal(err)
		}

		messageLines := strings.Split(message, "\n")
		fmt.Printf("[(root-commit) %s] %s\n", c.GetOid(), messageLines[0])

		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "jit: '%s' is not a jit command.\n", command)
		os.Exit(1)
	}
}
