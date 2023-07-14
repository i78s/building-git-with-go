package repository

import (
	"building-git/lib"
	"building-git/lib/database"
	"fmt"
	"io/fs"
	"path/filepath"
)

type ChangeType int

const (
	Added ChangeType = iota
	Deleted
	Modified
)

type Status struct {
	repo             *Repository
	stats            map[string]fs.FileInfo
	Changed          *lib.SortedMap[struct{}]
	IndexChanges     *lib.SortedMap[ChangeType]
	WorkspaceChanges *lib.SortedMap[ChangeType]
	Untracked        *lib.SortedMap[struct{}]
	headTree         map[string]*database.Entry
}

func NewStatus(repo *Repository) (*Status, error) {
	s := &Status{
		repo:             repo,
		stats:            map[string]fs.FileInfo{},
		Changed:          lib.NewSortedMap[struct{}](),
		IndexChanges:     lib.NewSortedMap[ChangeType](),
		WorkspaceChanges: lib.NewSortedMap[ChangeType](),
		Untracked:        lib.NewSortedMap[struct{}](),
		headTree:         make(map[string]*database.Entry),
	}

	err := s.scanWorkspace("")
	if err != nil {
		return nil, err
	}
	s.loadHeadTree()
	s.checkIndexEntries()
	s.collectDeletedHeadFiles()

	return s, nil
}

func (s *Status) recordChange(path string, smap *lib.SortedMap[ChangeType], ctype ChangeType) {
	s.Changed.Set(path, struct{}{})
	smap.Set(path, ctype)
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
		} else if s.isTrackableFile(path, stat) {
			if stat.IsDir() {
				path += string(filepath.Separator)
			}
			s.Untracked.Set(path, struct{}{})
		}
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

func (s *Status) loadHeadTree() error {
	s.headTree = make(map[string]*database.Entry)

	headOid, err := s.repo.Refs.ReadHead()
	if err != nil {
		return err
	}
	if headOid == "" {
		return nil
	}
	commitObj, _ := s.repo.Database.Load(headOid)

	commit, ok := commitObj.(*database.Commit)
	if !ok {
		return fmt.Errorf("Failed to cast to Commit")
	}

	s.readTree(commit.Tree(), "")
	return nil
}

func (s *Status) readTree(treeOid, pathname string) error {
	treeObj, _ := s.repo.Database.Load(treeOid)
	tree, ok := treeObj.(*database.Tree)
	if !ok {
		return fmt.Errorf("Failed to cast to Tree")
	}

	for name, e := range tree.Entries {
		path := filepath.Join(pathname, name)
		entry, ok := e.(*database.Entry)
		if !ok {
			return fmt.Errorf("Failed to cast to Entry")
		}
		if entry.IsTree() {
			err := s.readTree(entry.Oid(), name)
			if err != nil {
				return err
			}
		} else {
			s.headTree[path] = entry
		}
	}

	return nil
}

func (s *Status) checkIndexEntries() {
	for _, entry := range s.repo.Index.EachEntry() {
		s.checkIndexAgainstWorkspace(entry)
		s.checkIndexAgainstHeadTree(entry)
	}
}

func (s *Status) checkIndexAgainstWorkspace(entry database.EntryObject) {
	stat, exists := s.stats[entry.Key()]

	if !exists {
		s.recordChange(entry.Key(), s.WorkspaceChanges, Deleted)
		return
	}

	if !entry.IsStatMatch(stat) {
		s.recordChange(entry.Key(), s.WorkspaceChanges, Modified)
		return
	}
	if entry.IsTimesMatch(stat) {
		return
	}

	data, _ := s.repo.Workspace.ReadFile(entry.Key())
	blob := database.NewBlob(data)
	oid, _ := s.repo.Database.HashObject(blob)

	if entry.Oid() == oid {
		s.repo.Index.UpdateEntryStat(entry, stat)
		return
	}
	s.recordChange(entry.Key(), s.WorkspaceChanges, Modified)
}

func (s *Status) checkIndexAgainstHeadTree(entry database.EntryObject) {
	item := s.headTree[entry.Key()]

	if item != nil {
		if entry.Mode() != item.Mode() || entry.Oid() != item.Oid() {
			s.recordChange(entry.Key(), s.IndexChanges, Modified)
		}
		return
	}
	s.recordChange(entry.Key(), s.IndexChanges, Added)
}

func (s *Status) collectDeletedHeadFiles() {
	for path := range s.headTree {
		if s.repo.Index.IsTrackedFile(path) {
			continue
		}
		s.recordChange(path, s.IndexChanges, Deleted)
	}
}