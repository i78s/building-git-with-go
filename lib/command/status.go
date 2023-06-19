package command

import (
	"building-git/lib"
	"fmt"
	"io"
	"path/filepath"
	"sort"
)

type Status struct {
	rootPath  string
	repo      *lib.Repository
	untracked map[string]struct{}
	stdout    io.Writer
	stderr    io.Writer
}

func NewStatus(dir string, stdout, stderr io.Writer) (*Status, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := lib.NewRepository(rootPath)

	return &Status{
		rootPath:  rootPath,
		repo:      repo,
		untracked: map[string]struct{}{},
		stdout:    stdout,
		stderr:    stderr,
	}, nil
}

func (s *Status) Run() int {
	s.repo.Index.Load()
	err := s.scanWorkspace("")
	if err != nil {
		fmt.Fprintf(s.stderr, "fatal: %v", err)
		return 128
	}

	untracked := []string{}
	for filename := range s.untracked {
		untracked = append(untracked, filename)
	}
	sort.Strings(untracked)

	for _, filename := range untracked {
		fmt.Fprintf(s.stdout, "?? %s\n", filename)
	}

	return 0
}

func (s *Status) scanWorkspace(prefix string) error {
	files, err := s.repo.Workspace.ListDir(prefix)
	if err != nil {
		return err
	}

	for path, stat := range files {
		if s.repo.Index.IsTracked(path) {
			if stat.IsDir() {
				s.scanWorkspace(path)
			}
			continue
		}

		if stat.IsDir() {
			path += string(filepath.Separator)
		}
		s.untracked[path] = struct{}{}
	}
	return nil
}
