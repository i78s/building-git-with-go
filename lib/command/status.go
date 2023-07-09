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

type changeType int

const (
	added changeType = iota
	deleted
	modified
)

var SHORT_STATUS = map[changeType]string{
	added:    "A",
	deleted:  "D",
	modified: "M",
}

type Status struct {
	rootPath         string
	repo             *lib.Repository
	stats            map[string]fs.FileInfo
	changed          map[string]struct{}
	indexChanges     map[string]changeType
	workspaceChanges map[string]changeType
	untracked        map[string]struct{}
	headTree         map[string]*database.Entry
	stdout           io.Writer
	stderr           io.Writer
}

func NewStatus(dir string, stdout, stderr io.Writer) (*Status, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := lib.NewRepository(rootPath)

	return &Status{
		rootPath:         rootPath,
		repo:             repo,
		stats:            map[string]fs.FileInfo{},
		changed:          map[string]struct{}{},
		indexChanges:     map[string]changeType{},
		workspaceChanges: map[string]changeType{},
		untracked:        map[string]struct{}{},
		headTree:         make(map[string]*database.Entry),
		stdout:           stdout,
		stderr:           stderr,
	}, nil
}

func (s *Status) Run() int {
	s.repo.Index.LoadForUpdate()
	err := s.scanWorkspace("")
	if err != nil {
		fmt.Fprintf(s.stderr, "fatal: %v", err)
		return 128
	}
	s.loadHeadTree()
	s.checkIndexEntries()
	s.collectDeletedHeadFiles()

	s.repo.Index.WriteUpdates()
	s.printResults()

	return 0
}

func (s *Status) printResults() {
	changed := []string{}
	for filename := range s.changed {
		changed = append(changed, filename)
	}
	sort.Strings(changed)
	for _, filename := range changed {
		status := s.statusFor(filename)
		fmt.Fprintf(s.stdout, "%s %s\n", status, filename)
	}

	untracked := []string{}
	for filename := range s.untracked {
		untracked = append(untracked, filename)
	}
	sort.Strings(untracked)
	for _, filename := range untracked {
		fmt.Fprintf(s.stdout, "?? %s\n", filename)
	}
}

func (s *Status) statusFor(path string) string {
	left := " "
	if ctype, exists := s.indexChanges[path]; exists {
		if status, exists := SHORT_STATUS[ctype]; exists {
			left = status
		}
	}

	right := " "
	if ctype, exists := s.workspaceChanges[path]; exists {
		if status, exists := SHORT_STATUS[ctype]; exists {
			right = status
		}
	}

	return left + right
}

func (s *Status) recordChange(path string, set map[string]changeType, ctype changeType) {
	s.changed[path] = struct{}{}
	set[path] = ctype
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
			s.untracked[path] = struct{}{}
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
		s.recordChange(entry.Key(), s.workspaceChanges, deleted)
		return
	}

	if !entry.IsStatMatch(stat) {
		s.recordChange(entry.Key(), s.workspaceChanges, modified)
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
	s.recordChange(entry.Key(), s.workspaceChanges, modified)
}

func (s *Status) checkIndexAgainstHeadTree(entry database.EntryObject) {
	item := s.headTree[entry.Key()]

	if item != nil {
		if entry.Mode() != item.Mode() || entry.Oid() != item.Oid() {
			s.recordChange(entry.Key(), s.indexChanges, modified)
		}
		return
	}
	s.recordChange(entry.Key(), s.indexChanges, added)
}

func (s *Status) collectDeletedHeadFiles() {
	for path := range s.headTree {
		if s.repo.Index.IsTrackedFile(path) {
			continue
		}
		s.recordChange(path, s.indexChanges, deleted)
	}
}
