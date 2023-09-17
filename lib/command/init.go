package command

import (
	"building-git/lib/repository"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var DEFAULT_BRANCH = "master"

func Init(args []string, stdout, stderr io.Writer) int {
	var path string
	if len(args) == 0 {
		var err error
		path, err = os.Getwd()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	} else {
		path = args[0]
	}

	rootPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	gitPath := filepath.Join(rootPath, ".git")
	dirs := []string{"objects", "refs/heads"}

	for _, dir := range dirs {
		err := os.MkdirAll(filepath.Join(gitPath, dir), os.ModePerm)
		if err != nil {
			fmt.Fprintf(stderr, "fatal: %s\n", err.Error())
			return 1
		}
	}

	refs := repository.NewRefs(gitPath)
	headPath := filepath.Join("refs", "heads", DEFAULT_BRANCH)
	refs.UpdateHead(fmt.Sprintf("ref: %s", headPath))

	fmt.Fprintf(stdout, "Initialized empty Jit repository in %s\n", gitPath)
	return 0
}
