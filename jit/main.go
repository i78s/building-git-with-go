package main

import (
	"building-git/jit/blob"
	"building-git/jit/database"
	"building-git/jit/workspace"
	"fmt"
	"os"
	"path/filepath"
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
		files, _ := ws.ListFiles()
		for _, path := range files {
			data, _ := ws.ReadFile(path)
			db.Store(blob.NewBlob(data))
		}
	default:
		fmt.Fprintf(os.Stderr, "jit: '%s' is not a jit command.\n", command)
		os.Exit(1)
	}
}
