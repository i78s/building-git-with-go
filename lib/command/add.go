package command

import (
	"building-git/lib"
	"building-git/lib/database"
	"fmt"
	"io"
	"path/filepath"
)

func Add(dir string, args []string, stdout, stderr io.Writer) int {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintf(stderr, "fatal: %v", err)
		return 1
	}
	repo := lib.NewRepository(rootPath)

	err = repo.Index.LoadForUpdate()
	if err != nil {
		fmt.Fprintf(stderr, `fatal: %v
		Another git process seems to be running in this repository.
		Please make sure all processes are terminated then try again.
		If it still fails, a git process may have crashed in this
		repository earlier: remove the file manually to continue.`, err)
		return 128
	}

	var paths []string
	for _, path := range args {
		absPath, err := filepath.Abs(filepath.Join(dir, path))
		if err != nil {
			fmt.Fprintf(stderr, "fatal: %v", err)
			return 128
		}
		files, err := repo.Workspace.ListFiles(absPath)
		if err != nil {
			fmt.Fprintf(stderr, "fatal: %v", err)
			return 128
		}
		paths = append(paths, files...)
	}

	for _, pathname := range paths {
		data, err := repo.Workspace.ReadFile(pathname)
		if err != nil {
			repo.Index.ReleaseLock()
			fmt.Fprintf(stderr, "fatal: %v", err)
			return 128
		}
		stat, err := repo.Workspace.StatFile(pathname)
		if err != nil {
			repo.Index.ReleaseLock()
			fmt.Fprintf(stderr, "fatal: %v", err)
			return 128
		}
		blob := database.NewBlob(data)
		repo.Database.Store(blob)
		repo.Index.Add(pathname, blob.Oid(), stat)
	}
	repo.Index.WriteUpdates()
	return 0
}
