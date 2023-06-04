package command

import (
	"building-git/lib"
	"building-git/lib/database"
	"fmt"
	"os"
	"path/filepath"
)

func Add(args []string) {
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

	err = repo.Index.LoadForUpdate()
	if err != nil {
		fmt.Fprint(os.Stderr, `fatal: `, err.Error(), `
			Another git process seems to be running in this repository.
			Please make sure all processes are terminated then try again.
			If it still fails, a git process may have crashed in this
			repository earlier: remove the file manually to continue.
			`)
		os.Exit(128)
	}

	var paths []string
	for _, path := range args {
		absPath, err := filepath.Abs(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(128)
		}
		files, err := repo.Workspace.ListFiles(absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fatal: %s\n", err.Error())
			os.Exit(128)
		}
		paths = append(paths, files...)
	}

	for _, pathname := range paths {
		data, err := repo.Workspace.ReadFile(pathname)
		if err != nil {
			repo.Index.ReleaseLock()
			fmt.Fprintf(os.Stderr, "fatal: %s\n", err.Error())
			os.Exit(128)
		}
		stat, err := repo.Workspace.StatFile(pathname)
		if err != nil {
			repo.Index.ReleaseLock()
			fmt.Fprintf(os.Stderr, "fatal: %s\n", err.Error())
			os.Exit(128)
		}
		blob := database.NewBlob(data)
		repo.Database.Store(blob)
		repo.Index.Add(pathname, blob.GetOid(), stat)
	}
	repo.Index.WriteUpdates()

	os.Exit(0)
}
