package repository

import (
	"building-git/lib/database"
	"path/filepath"
)

type changeType string

const (
	create changeType = "create"
	update changeType = "update"
	delete changeType = "delete"
)

type plan struct {
	path string
	item database.TreeObject
}

type Migration struct {
	repo    *Repository
	diff    map[string][2]database.TreeObject
	Changes map[changeType][]plan
	Mkdirs  map[string]string
	Rmdirs  map[string]string
}

func NewMigration(repository *Repository, treeDiff map[string][2]database.TreeObject) *Migration {
	return &Migration{
		repo: repository,
		diff: treeDiff,
		Changes: map[changeType][]plan{
			create: {},
			update: {},
			delete: {},
		},
		Mkdirs: make(map[string]string),
		Rmdirs: make(map[string]string),
	}
}

func (m *Migration) ApplyChanges() {
	m.planChanges()
	m.updateWorkspace()
}

func (m *Migration) BlobData(oid string) (string, error) {
	blob, err := m.repo.Database.Load(oid)
	return blob.String(), err
}

func (m *Migration) planChanges() {
	for path, diff := range m.diff {
		m.recordChange(path, diff[0], diff[1])
	}
}

func (m *Migration) updateWorkspace() {
	m.repo.Workspace.ApplyMigration(m)
}

func (m *Migration) recordChange(path string, oldItem, newItem database.TreeObject) {
	dirs := descend(filepath.Dir(path))

	if oldItem == nil {
		for _, dir := range dirs {
			m.Mkdirs[dir] = dir
		}
		m.Changes[create] = append(m.Changes[create], plan{path: path, item: newItem})
	} else if newItem == nil {
		for _, dir := range dirs {
			m.Rmdirs[dir] = dir
		}
		m.Changes[delete] = append(m.Changes[delete], plan{path: path, item: newItem})
	} else {
		for _, dir := range dirs {
			m.Mkdirs[dir] = dir
		}
		m.Changes[update] = append(m.Changes[update], plan{path: path, item: newItem})
	}
}

func descend(path string) []string {
	var parts []string
	for path != "." && path != "/" {
		parts = append([]string{path}, parts...)
		path = filepath.Dir(path)
	}
	return parts
}
