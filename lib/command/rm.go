package command

import (
	"building-git/lib/repository"
	"fmt"
	"io"
	"path/filepath"
)

type RmOption struct {
}

type Rm struct {
	rootPath    string
	args        []string
	options     RmOption
	repo        *repository.Repository
	headOid     string
	inspector   *repository.Inspector
	uncommitted []string
	unstaged    []string
	stdout      io.Writer
	stderr      io.Writer
}

func NewRm(dir string, args []string, options RmOption, stdout, stderr io.Writer) (*Rm, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRepository(rootPath)
	inspector := repository.NewInspector(repo)
	headOid, _ := repo.Refs.ReadHead()

	return &Rm{
		rootPath:    rootPath,
		args:        args,
		options:     options,
		repo:        repo,
		headOid:     headOid,
		inspector:   inspector,
		unstaged:    []string{},
		uncommitted: []string{},
		stdout:      stdout,
		stderr:      stderr,
	}, nil
}

func (r *Rm) Run() int {
	r.repo.Index.LoadForUpdate()
	for _, path := range r.args {
		err := r.planRemoval(path)
		if err != nil {
			r.repo.Index.ReleaseLock()
			fmt.Fprintf(r.stderr, "fatal: %v\n", err)
			return 128
		}
	}
	if err := r.exitOnErrors(); err != nil {
		return 1
	}

	for _, path := range r.args {
		r.removeFile(path)
	}
	r.repo.Index.WriteUpdates()

	return 0
}

func (r *Rm) planRemoval(path string) error {
	if !r.repo.Index.IsTrackedFile(path) {
		return fmt.Errorf("pathspec '%s' did not match any files", path)
	}
	item := r.repo.Database.LoadTreeEntry(r.headOid, path)
	entry := r.repo.Index.EntryForPath(path, "0")
	stat, _ := r.repo.Workspace.StatFile(path)

	if r.inspector.CompareTreeToIndex(item, entry) != repository.Unmodified {
		r.uncommitted = append(r.uncommitted, path)
	} else if stat != nil && r.inspector.CompareIndexToWorkspace(entry, stat) != repository.Unmodified {
		r.unstaged = append(r.unstaged, path)
	}
	return nil
}

func (r *Rm) removeFile(path string) {
	r.repo.Index.Remove(path)
	r.repo.Workspace.Remove(path)
	fmt.Fprintf(r.stdout, "rm %s\n", path)
}

func (r *Rm) exitOnErrors() error {
	if len(r.uncommitted) == 0 && len(r.unstaged) == 0 {
		return nil
	}
	r.printErrors(r.uncommitted, "changes staged in the index")
	r.printErrors(r.unstaged, "local modifications")

	r.repo.Index.ReleaseLock()
	return fmt.Errorf("")
}

func (r *Rm) printErrors(paths []string, message string) {
	if len(paths) == 0 {
		return
	}

	filesHave := "file has"
	if len(paths) > 1 {
		filesHave = "files have"
	}
	fmt.Fprintf(r.stderr, "error: the following %s %s:\n", filesHave, message)
	for _, path := range paths {
		fmt.Fprintf(r.stderr, "    %s\n", path)
	}
}
