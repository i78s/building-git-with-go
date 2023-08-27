package repository

import (
	"building-git/lib/database"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

type changeType string

const (
	create changeType = "create"
	update changeType = "update"
	delete changeType = "delete"
)

type conflictType string

const (
	staleFile            conflictType = "staleFile"
	staleDirectory       conflictType = "staleDirectory"
	untrackedOverwritten conflictType = "untrackedOverwritten"
	untrackedRemoved     conflictType = "untrackedRemoved"
)

var migrationMessages = map[conflictType][2]string{
	staleFile: {
		"Your local changes to the following files would be overwritten by checkout:",
		"Please commit your changes or stash them before you switch branches.",
	},
	staleDirectory: {
		"Updating the following directories would lose untracked files in them:",
		"",
	},
	untrackedOverwritten: {
		"The following untracked working tree files would be overwritten by checkout:",
		"Please move or remove them before you switch branches.",
	},
	untrackedRemoved: {
		"The following untracked working tree files would be removed by checkout:",
		"Please move or remove them before you switch branches.",
	},
}

type plan struct {
	path string
	item database.TreeObject
}

type Migration struct {
	repo      *Repository
	diff      map[string][2]database.TreeObject
	inspector *Inspector
	Changes   map[changeType][]plan
	Mkdirs    map[string]string
	Rmdirs    map[string]string
	Errors    []string
	conflicts map[conflictType][]string
}

func NewMigration(repository *Repository, treeDiff map[string][2]database.TreeObject) *Migration {
	return &Migration{
		repo:      repository,
		diff:      treeDiff,
		inspector: NewInspector(repository),
		Changes: map[changeType][]plan{
			create: {},
			update: {},
			delete: {},
		},
		Mkdirs: make(map[string]string),
		Rmdirs: make(map[string]string),
		Errors: []string{},
		conflicts: map[conflictType][]string{
			staleFile:            {},
			staleDirectory:       {},
			untrackedOverwritten: {},
			untrackedRemoved:     {},
		},
	}
}

func (m *Migration) ApplyChanges() error {
	err := m.planChanges()
	if err != nil {
		return err
	}
	m.updateWorkspace()
	m.updateIndex()
	return nil
}

func (m *Migration) BlobData(oid string) (string, error) {
	blob, err := m.repo.Database.Load(oid)
	return blob.String(), err
}

func (m *Migration) planChanges() error {
	for path, diff := range m.diff {
		m.checkForConflict(path, diff[0], diff[1])
		m.recordChange(path, diff[0], diff[1])
	}
	return m.collectErrors()
}

func (m *Migration) updateWorkspace() {
	m.repo.Workspace.ApplyMigration(m)
}

func (m *Migration) updateIndex() {
	for _, p := range m.Changes[delete] {
		m.repo.Index.Remove(p.path)
	}

	for _, p := range m.Changes[create] {
		stat, _ := m.repo.Workspace.StatFile(p.path)
		m.repo.Index.Add(p.path, p.item.Oid(), stat)
	}
	for _, p := range m.Changes[update] {
		stat, _ := m.repo.Workspace.StatFile(p.path)
		m.repo.Index.Add(p.path, p.item.Oid(), stat)
	}
}

func (m *Migration) recordChange(path string, oldItem, newItem database.TreeObject) {
	dirs := descend(filepath.Dir(path))

	if oldItem == nil || oldItem != nil && oldItem.IsNil() {
		for _, dir := range dirs {
			m.Mkdirs[dir] = dir
		}
		m.Changes[create] = append(m.Changes[create], plan{path: path, item: newItem})
	} else if newItem == nil || newItem.IsNil() {
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

func (m *Migration) checkForConflict(path string, oldItem, newItem database.TreeObject) {
	entry := m.repo.Index.EntryForPath(path)

	if m.indexDiffersFromTrees(entry, oldItem, newItem) {
		m.conflicts[staleFile] = append(m.conflicts[staleFile], path)
		return
	}

	stat, _ := m.repo.Workspace.StatFile(path)
	etype := m.getErrorType(stat, entry, newItem)

	if stat == nil {
		parent := m.untrackedParent(path)
		if parent != "" {
			if !entry.IsNil() {
				m.conflicts[etype] = append(m.conflicts[etype], path)
			} else {
				m.conflicts[etype] = append(m.conflicts[etype], parent)
			}
		}
	} else if stat.Mode().IsRegular() {
		changed := m.inspector.compareIndexToWorkspace(entry, stat)
		if changed != Unmodified {
			m.conflicts[etype] = append(m.conflicts[etype], path)
		}
	} else if stat.IsDir() {
		trackable := m.inspector.isTrackableFile(path, stat)
		if trackable {
			m.conflicts[etype] = append(m.conflicts[etype], path)
		}
	}
}

func (m *Migration) getErrorType(stat fs.FileInfo, entry, item database.TreeObject) conflictType {
	if !entry.IsNil() {
		return staleFile
	} else if stat != nil && stat.IsDir() {
		return staleDirectory
	} else if item != nil {
		return untrackedOverwritten
	} else if item != nil && !item.IsNil() {
		return untrackedOverwritten
	}
	return untrackedRemoved
}

func (m *Migration) indexDiffersFromTrees(entry, oldItem, newItem database.TreeObject) bool {
	if m.inspector.compareTreeToIndex(oldItem, entry) != Unmodified &&
		m.inspector.compareTreeToIndex(newItem, entry) != Unmodified {
		return true
	}
	return false
}

func (m *Migration) untrackedParent(path string) string {
	dirname := filepath.Dir(path)
	for _, parent := range ascend(dirname) {
		parentStat, err := m.repo.Workspace.StatFile(parent)
		if err != nil || parentStat.IsDir() {
			continue
		}
		if m.inspector.isTrackableFile(parent, parentStat) {
			return parent
		}
	}
	return ""
}

func (m *Migration) collectErrors() error {
	for ctype, paths := range m.conflicts {
		if len(paths) == 0 {
			continue
		}

		msg := migrationMessages[ctype]
		header := msg[0]
		footer := msg[1]

		lines := []string{header}
		for _, path := range paths {
			lines = append(lines, "\t"+path)
		}
		lines = append(lines, footer)

		m.Errors = append(m.Errors, strings.Join(lines, "\n"))
	}

	if len(m.Errors) == 0 {
		return nil
	}
	return fmt.Errorf("conflicts detected")
}

func ascend(path string) []string {
	var parts []string
	for path != "." && path != "/" {
		parts = append(parts, path)
		path = filepath.Dir(path)
	}
	return parts
}

func descend(path string) []string {
	var parts []string
	for path != "." && path != "/" {
		parts = append([]string{path}, parts...)
		path = filepath.Dir(path)
	}
	return parts
}
