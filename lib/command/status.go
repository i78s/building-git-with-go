package command

import (
	"building-git/lib"
	"building-git/lib/database"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"sort"
)

type Status struct {
	rootPath  string
	repo      *lib.Repository
	stats     map[string]fs.FileInfo
	changed   map[string]struct{}
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
		stats:     map[string]fs.FileInfo{},
		changed:   map[string]struct{}{},
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

	s.detectWorkspaceChanges()

	changed := []string{}
	for filename := range s.changed {
		changed = append(changed, filename)
	}
	sort.Strings(changed)
	for _, filename := range changed {
		fmt.Fprintf(s.stdout, " M %s\n", filename)
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
			if stat.Mode().IsRegular() {
				s.stats[path] = stat
			}
			if stat.IsDir() {
				s.scanWorkspace(path)
			}
			continue
		}
		if !s.isTrackableFile(path, stat) {
			continue
		}
		if stat.IsDir() {
			path += string(filepath.Separator)
		}
		s.untracked[path] = struct{}{}
	}
	return nil
}

func (st *Status) isTrackableFile(path string, stat fs.FileInfo) bool {
	if stat.Mode().IsRegular() {
		return !st.repo.Index.IsTracked(path)
	}
	if !stat.IsDir() {
		return false
	}

	items, _ := st.repo.Workspace.ListDir(path)
	files := map[string]fs.FileInfo{}
	dirs := map[string]fs.FileInfo{}
	for p, s := range items {
		if s.Mode().IsRegular() {
			files[p] = s
		}
		if s.IsDir() {
			dirs[p] = s
		}
	}

	for p, s := range files {
		if st.isTrackableFile(p, s) {
			return true
		}
	}
	for p, s := range dirs {
		if st.isTrackableFile(p, s) {
			return true
		}
	}
	return false
}

func (s *Status) detectWorkspaceChanges() {
	for _, entry := range s.repo.Index.EachEntry() {
		s.checkIndexEntry(entry)
	}
}

func (s *Status) checkIndexEntry(entry database.EntryObject) {
	if stat, exists := s.stats[entry.Key()]; exists {
		if entry.IsStatMatch(stat) {
			return
		}
		s.changed[entry.Key()] = struct{}{}
	}
}
