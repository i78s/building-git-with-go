package repository

import (
	"building-git/lib/database"
	"building-git/lib/sortedmap"
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
	Conflicts        *sortedmap.SortedMap[[]string]
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
		Conflicts:        sortedmap.NewSortedMap[[]string](),
		WorkspaceChanges: sortedmap.NewSortedMap[ChangeType](),
		Untracked:        sortedmap.NewSortedMap[struct{}](),
		HeadTree:         make(map[string]*database.Entry),
	}

	head, _ := repo.Refs.ReadHead()
	s.HeadTree = repo.Database.LoadTreeList(head, "")

	err := s.scanWorkspace("")
	if err != nil {
		return nil, err
	}
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

func (s *Status) checkIndexEntries() {
	for _, entry := range s.repo.Index.EachEntry() {
		if entry.Stage() == "0" {
			s.checkIndexAgainstWorkspace(entry)
			s.checkIndexAgainstHeadTree(entry)
		} else {
			s.Changed.Set(entry.Path(), struct{}{})
			stgs, _ := s.Conflicts.Get(entry.Path())
			s.Conflicts.Set(entry.Path(), append(stgs, entry.Stage()))
		}
	}
}

func (s *Status) checkIndexAgainstWorkspace(entry database.EntryObject) {
	stat := s.Stats[entry.Path()]
	status := s.inspector.CompareIndexToWorkspace(entry, stat)

	if status != Unmodified {
		s.recordChange(entry.Path(), s.WorkspaceChanges, status)
		return
	}
	s.repo.Index.UpdateEntryStat(entry, stat)
}

func (s *Status) checkIndexAgainstHeadTree(entry database.EntryObject) {
	item := s.HeadTree[entry.Path()]
	status := s.inspector.CompareTreeToIndex(item, entry)

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
