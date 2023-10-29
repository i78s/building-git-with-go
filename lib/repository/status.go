package repository

import (
	"building-git/lib/database"
	"building-git/lib/sortedmap"
	"fmt"
	"io/fs"
	"path/filepath"
)

type ChangeType int

const (
	Added ChangeType = iota
	Deleted
	Modified
	Unmodified
	Untracked
)

type Status struct {
	repo             *Repository
	Stats            map[string]fs.FileInfo
	inspector        *Inspector
	Changed          *sortedmap.SortedMap[struct{}]
	IndexChanges     *sortedmap.SortedMap[ChangeType]
	WorkspaceChanges *sortedmap.SortedMap[ChangeType]
	Untracked        *sortedmap.SortedMap[struct{}]
	HeadTree         map[string]*database.Entry
}

func NewStatus(repo *Repository) (*Status, error) {
	s := &Status{
		repo:             repo,
		Stats:            map[string]fs.FileInfo{},
		inspector:        NewInspector(repo),
		Changed:          sortedmap.NewSortedMap[struct{}](),
		IndexChanges:     sortedmap.NewSortedMap[ChangeType](),
		WorkspaceChanges: sortedmap.NewSortedMap[ChangeType](),
		Untracked:        sortedmap.NewSortedMap[struct{}](),
		HeadTree:         make(map[string]*database.Entry),
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

func (s *Status) recordChange(path string, smap *sortedmap.SortedMap[ChangeType], ctype ChangeType) {
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
				s.Stats[path] = stat
			}
			if stat.IsDir() {
				s.scanWorkspace(path)
			}
			continue
		} else if s.inspector.isTrackableFile(path, stat) {
			if stat.IsDir() {
				path += string(filepath.Separator)
			}
			s.Untracked.Set(path, struct{}{})
		}
	}
	return nil
}

func (s *Status) loadHeadTree() error {
	s.HeadTree = make(map[string]*database.Entry)

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
		return fmt.Errorf("failed to cast to commit")
	}

	s.readTree(commit.Tree(), "")
	return nil
}

func (s *Status) readTree(treeOid, pathname string) error {
	treeObj, _ := s.repo.Database.Load(treeOid)
	tree, ok := treeObj.(*database.Tree)
	if !ok {
		return fmt.Errorf("failed to cast to tree")
	}

	for name, e := range tree.Entries {
		path := filepath.Join(pathname, name)
		entry, ok := e.(*database.Entry)
		if !ok {
			return fmt.Errorf("failed to cast to entry")
		}
		if entry.IsTree() {
			err := s.readTree(entry.Oid(), path)
			if err != nil {
				return err
			}
		} else {
			s.HeadTree[path] = entry
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
	stat := s.Stats[entry.Path()]
	status := s.inspector.compareIndexToWorkspace(entry, stat)

	if status != Unmodified {
		s.recordChange(entry.Path(), s.WorkspaceChanges, status)
		return
	}
	s.repo.Index.UpdateEntryStat(entry, stat)
}

func (s *Status) checkIndexAgainstHeadTree(entry database.EntryObject) {
	item := s.HeadTree[entry.Path()]
	status := s.inspector.compareTreeToIndex(item, entry)

	if status == Unmodified {
		return
	}
	s.recordChange(entry.Path(), s.IndexChanges, status)
}

func (s *Status) collectDeletedHeadFiles() {
	for path := range s.HeadTree {
		if s.repo.Index.IsTrackedFile(path) {
			continue
		}
		s.recordChange(path, s.IndexChanges, Deleted)
	}
}
