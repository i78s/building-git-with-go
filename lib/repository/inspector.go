package repository

import (
	"building-git/lib/database"
	"io/fs"
)

type Inspector struct {
	repo *Repository
}

func NewInspector(repo *Repository) *Inspector {
	return &Inspector{
		repo: repo,
	}
}

func (i *Inspector) isTrackableFile(path string, stat fs.FileInfo) bool {
	if stat == nil {
		return false
	}
	if stat.Mode().IsRegular() {
		return !i.repo.Index.IsTrackedFile(path)
	}
	if !stat.IsDir() {
		return false
	}

	items, _ := i.repo.Workspace.ListDir(path)
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
		if i.isTrackableFile(p, s) {
			return true
		}
	}
	for p, s := range dirs {
		if i.isTrackableFile(p, s) {
			return true
		}
	}
	return false
}

func (i *Inspector) compareIndexToWorkspace(entry database.EntryObject, stat fs.FileInfo) ChangeType {
	if entry.IsNil() {
		return Untracked
	}
	if stat == nil {
		return Deleted
	}
	if !entry.IsStatMatch(stat) {
		return Modified
	}
	if entry.IsTimesMatch(stat) {
		return Unmodified
	}

	data, _ := i.repo.Workspace.ReadFile(entry.Path())
	blob := database.NewBlob(data)
	oid, _ := i.repo.Database.HashObject(blob)

	if entry.Oid() != oid {
		return Modified
	}
	return Unmodified
}

func (i *Inspector) compareTreeToIndex(item, entry database.TreeObject) ChangeType {
	if item == nil && entry == nil ||
		item == nil && entry != nil && entry.IsNil() ||
		item != nil && item.IsNil() && entry == nil ||
		item != nil && item.IsNil() && entry != nil && entry.IsNil() {
		return Unmodified
	}
	if item == nil || item.IsNil() {
		return Added
	}
	if entry == nil || entry.IsNil() {
		return Deleted
	}
	if entry.Mode() != item.Mode() || entry.Oid() != item.Oid() {
		return Modified
	}
	return Unmodified
}
