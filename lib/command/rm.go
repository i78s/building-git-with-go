package command

import (
	"building-git/lib/repository"
	"fmt"
	"io"
	"path/filepath"
)

const BOTH_CHANGED = "staged content different from both the file and the HEAD"
const INDEX_CHANGED = "changes staged in the index"
const WORKSPACE_CHANGED = "local modifications"

type RmOption struct {
	Cached    bool
	Force     bool
	Recursive bool
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
	bothChanged []string
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
		bothChanged: []string{},
		stdout:      stdout,
		stderr:      stderr,
	}, nil
}

func (r *Rm) Run() int {
	r.repo.Index.LoadForUpdate()

	paths := []string{}
	for _, path := range r.args {
		ex, err := r.expandPath(path)
		if err != nil {
			r.repo.Index.ReleaseLock()
			fmt.Fprintf(r.stderr, "fatal: %v\n", err)
			return 128
		}
		paths = append(paths, ex...)
	}
	for _, path := range paths {
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

	for _, path := range paths {
		r.removeFile(path)
	}
	r.repo.Index.WriteUpdates()

	return 0
}

func (r *Rm) expandPath(path string) ([]string, error) {
	if r.repo.Index.IsTrackedDirectory(path) {
		if !r.options.Recursive {
			return nil, fmt.Errorf("not removing '%s' recursively without -r", path)
		}
		return r.repo.Index.ChildPaths(path), nil
	}
	if r.repo.Index.IsTrackedFile(path) {
		return []string{path}, nil
	}
	return nil, fmt.Errorf("pathspec '%s' did not match any files", path)
}

func (r *Rm) planRemoval(path string) error {
	if r.options.Force {
		return nil
	}

	stat, _ := r.repo.Workspace.StatFile(path)
	if stat != nil && stat.IsDir() {
		return fmt.Errorf("jit rm: '%s': Operation not permitted", path)
	}

	item := r.repo.Database.LoadTreeEntry(r.headOid, path)
	entry := r.repo.Index.EntryForPath(path, "0")

	stagedChange := r.inspector.CompareTreeToIndex(item, entry)
	unstagedChange := repository.Unmodified
	if stat != nil {
		unstagedChange = r.inspector.CompareIndexToWorkspace(entry, stat)
	}

	if stagedChange != repository.Unmodified && unstagedChange != repository.Unmodified {
		r.bothChanged = append(r.bothChanged, path)
	} else if stagedChange != repository.Unmodified {
		if !r.options.Cached {
			r.uncommitted = append(r.uncommitted, path)
		}
	} else if unstagedChange != repository.Unmodified {
		if !r.options.Cached {
			r.unstaged = append(r.unstaged, path)
		}
	}
	return nil
}

func (r *Rm) removeFile(path string) {
	r.repo.Index.Remove(path)

	if !r.options.Cached {
		r.repo.Workspace.Remove(path)
	}
	fmt.Fprintf(r.stdout, "rm %s\n", path)
}

func (r *Rm) exitOnErrors() error {
	if len(r.bothChanged) == 0 && len(r.uncommitted) == 0 && len(r.unstaged) == 0 {
		return nil
	}
	r.printErrors(r.bothChanged, BOTH_CHANGED)
	r.printErrors(r.uncommitted, INDEX_CHANGED)
	r.printErrors(r.unstaged, WORKSPACE_CHANGED)

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
