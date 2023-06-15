package command

import (
	"building-git/lib"
	"fmt"
	"io"
	"path/filepath"
)

func Status(dir string, stdout, stderr io.Writer) int {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintf(stderr, "fatal: %v", err)
		return 1
	}
	repo := lib.NewRepository(rootPath)

	files, err := repo.Workspace.ListFiles(rootPath)
	if err != nil {
		fmt.Fprintf(stderr, "fatal: %v", err)
		return 128
	}

	for _, filename := range files {
		fmt.Fprintf(stdout, "?? %s\n", filename)
	}

	return 0
}
