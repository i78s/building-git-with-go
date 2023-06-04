package command

import (
	"fmt"
	"os"
	"path/filepath"
)

func Init(args []string) {
	var path string
	if len(args) == 0 {
		var err error
		path, err = os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		path = args[0]
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
}
