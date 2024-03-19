package repository

import (
	"building-git/lib/database"
	"fmt"
	"regexp"
	"sort"
)

type RevListFlag int

const (
	added RevListFlag = iota
	seen
	uninteresting
	treesame
)

var (
	RANGE   = regexp.MustCompile(`^(.*)\.\.(.*)$`)
	EXCLUDE = regexp.MustCompile(`^\^(.+)$`)
)

type RevList struct {
	repo    *Repository
	commits map[string]*database.Commit
	flags   map[string]map[RevListFlag]bool
	queue   []*database.Commit
	pending []*database.Entry
	limited bool
	prune   []string
	diffs   map[[2]string]map[string][2]database.TreeObject
	output  []*database.Commit
	filter  *database.PathFilter
	walk    bool
	all     bool
	objects bool
	missing bool
}

type RevListOption struct {
	Walk    *bool
	All     bool
	Objects bool
	Missing bool
}

func NewRevList(repo *Repository, revs []string, options RevListOption) (*RevList, error) {
	revList := &RevList{
		repo:    repo,
		commits: map[string]*database.Commit{},
		flags:   map[string]map[RevListFlag]bool{},
		queue:   make([]*database.Commit, 0),
		prune:   make([]string, 0),
		diffs:   make(map[[2]string]map[string][2]database.TreeObject),
		output:  make([]*database.Commit, 0),
		objects: options.Objects,
		all:     options.All,
		missing: options.Missing,
	}
	if options.Walk == nil {
		revList.walk = true
	} else {
		revList.walk = *options.Walk
	}

	if revList.all {
		err := revList.includeRefs(repo.Refs.listAllRefs())
		if err != nil {
			return nil, err
		}
	}

	for _, rev := range revs {
		err := revList.handleRevision(rev)
		if err != nil {
			return nil, err
		}
	}
	if len(revList.queue) == 0 {
		err := revList.handleRevision(HEAD)
		if err != nil {
			return nil, err
		}
	}
	revList.filter = database.PathFilterBuild(revList.prune)

	return revList, nil
}

type RevListObject interface {
	Oid() string
	SetOid(string)
	Type() string
}

func (r *RevList) Each() []RevListObject {
	if r.limited {
		r.limitList()
	}
	if r.objects {
		r.markEdgesUninteresting()
	}

	var objects []RevListObject
	r.traverseCommits(func(c *database.Commit) {
		objects = append(objects, c)
	})
	r.traversePending(func(obj database.TreeObject) {
		objects = append(objects, obj.(RevListObject))
	})
	return objects
}

func (r *RevList) ReverseEach() []RevListObject {
	objects := r.Each()
	for i := 0; i < len(objects)/2; i++ {
		objects[i], objects[len(objects)-i-1] = objects[len(objects)-i-1], objects[i]
	}
	return objects
}

func (r *RevList) TreeDiff(oldOid, newOid string, differ *database.PathFilter) map[string][2]database.TreeObject {
	key := [2]string{oldOid, newOid}
	diff, ok := r.diffs[key]
	if ok {
		return diff
	}

	r.diffs[key] = r.repo.Database.TreeDiff(oldOid, newOid, r.filter)
	return r.diffs[key]
}

func (r *RevList) includeRefs(refs []*SymRef) error {
	oids := []string{}
	for _, symRef := range refs {
		oid, err := symRef.ReadOid()
		if err != nil {
			continue
		}
		oids = append(oids, oid)
	}
	for _, oid := range oids {
		err := r.handleRevision(oid)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *RevList) handleRevision(rev string) error {
	if stat, _ := r.repo.Workspace.StatFile(rev); stat != nil {
		r.prune = append(r.prune, rev)
	} else if match := RANGE.FindStringSubmatch(rev); match != nil {
		r.setStartPoint(match[1], false)
		r.setStartPoint(match[2], true)
		r.walk = true
	} else if match := EXCLUDE.FindStringSubmatch(rev); match != nil {
		r.setStartPoint(match[1], false)
		r.walk = true
	} else {
		return r.setStartPoint(rev, true)
	}
	return nil
}

func (r *RevList) setStartPoint(rev string, interesting bool) error {
	if rev == "" {
		rev = HEAD
	}

	oid, err := NewRevision(r.repo, rev).Resolve(COMMIT)
	if err != nil && !r.missing {
		return err
	}
	commit := r.loadCommit(oid)

	r.enqueueCommit(commit)

	if !interesting {
		r.limited = true
		r.mark(oid, uninteresting)
		r.markParentsUninteresting(commit)
	}
	return nil
}

func (r *RevList) enqueueCommit(commit *database.Commit) {
	if !r.mark(commit.Oid(), seen) {
		return
	}
	if r.walk {
		index := sort.Search(len(r.queue), func(i int) bool {
			return r.queue[i].Date().Before(commit.Date())
		})
		r.queue = append(r.queue[:index], append([]*database.Commit{commit}, r.queue[index:]...)...)
	} else {
		r.queue = append(r.queue, commit)
	}
}

func (r *RevList) limitList() {
	for r.isStillInteresting() {
		commit := r.queue[0]
		r.queue = r.queue[1:]
		r.addParents(commit)

		if !r.isMarked(commit.Oid(), uninteresting) {
			r.output = append(r.output, commit)
		}
	}
	r.queue = r.output
}

func (r *RevList) isStillInteresting() bool {
	if len(r.queue) == 0 {
		return false
	}

	var oldestOut *database.Commit
	if len(r.output) > 0 {
		oldestOut = r.output[len(r.output)-1]
	}
	newestIn := r.queue[0]

	fmt.Println(oldestOut)

	if oldestOut != nil &&
		(oldestOut.Date().Before(newestIn.Date()) ||
			oldestOut.Date().Equal(newestIn.Date())) {
		return true
	}

	for _, commit := range r.queue {
		if !r.isMarked(commit.Oid(), uninteresting) {
			return true
		}
	}

	return false
}

func (r *RevList) addParents(commit *database.Commit) {
	if !r.walk || !r.mark(commit.Oid(), added) {
		return
	}

	var parents []*database.Commit
	if r.isMarked(commit.Oid(), uninteresting) {
		for _, parent := range commit.Parents {
			parents = append(parents, r.loadCommit(parent))
		}

		for _, parent := range parents {
			r.markParentsUninteresting(parent)
		}
	} else {
		for _, oid := range r.simplifyCommit(commit) {
			parents = append(parents, r.loadCommit(oid))
		}
	}
	for _, parent := range parents {
		r.enqueueCommit(parent)
	}
}

func (r *RevList) markParentsUninteresting(commit *database.Commit) {
	queue := make([]string, len(commit.Parents))
	copy(queue, commit.Parents)

	for len(queue) > 0 {
		oid := queue[0]
		queue = queue[1:]

		if !r.mark(oid, uninteresting) {
			continue
		}

		commit, exists := r.commits[oid]
		if exists {
			queue = append(queue, commit.Parents...)
		}
	}
}

func (r *RevList) markEdgesUninteresting() {
	for _, c := range r.queue {
		if r.mark(c.Oid(), uninteresting) {
			r.markTreeUninteresting(c.Tree())
		}

		for _, oid := range c.Parents {
			if !r.isMarked(oid, uninteresting) {
				return
			}
			parent := r.loadCommit(oid)
			r.markTreeUninteresting(parent.Tree())
		}
	}
}

func (r *RevList) markTreeUninteresting(treeOid string) {
	entry := r.repo.Database.TreeEntry(treeOid)
	r.traverseTree(entry, func(object *database.Entry) bool {
		return r.mark(object.Oid(), uninteresting)
	})
}

func (r *RevList) simplifyCommit(commit *database.Commit) []string {
	if len(r.prune) == 0 {
		return commit.Parents
	}

	parents := commit.Parents
	if len(parents) == 0 {
		parents = append(parents, "")
	}

	for _, oid := range parents {
		td := r.TreeDiff(oid, commit.Oid(), nil)
		if len(td) != 0 {
			continue
		}
		r.mark(commit.Oid(), treesame)
		return []string{oid}
	}

	return commit.Parents
}

func (r *RevList) traverseCommits(fn func(*database.Commit)) {
	for len(r.queue) > 0 {
		commit := r.queue[0]
		r.queue = r.queue[1:]

		if !r.limited {
			r.addParents(commit)
		}
		if r.isMarked(commit.Oid(), uninteresting) {
			continue
		}
		if r.isMarked(commit.Oid(), treesame) {
			continue
		}

		r.pending = append(r.pending, r.repo.Database.TreeEntry(commit.Tree()))
		fn(commit)
	}
}

func (r *RevList) traversePending(fn func(entry database.TreeObject)) {
	if !r.objects {
		return
	}

	for _, entry := range r.pending {
		r.traverseTree(entry, func(object *database.Entry) bool {
			if r.isMarked(object.Oid(), uninteresting) {
				return false
			}
			if !r.mark(object.Oid(), seen) {
				return false
			}
			fn(object)
			return true
		})
	}
}

func (r *RevList) traverseTree(entry *database.Entry, fn func(entry *database.Entry) bool) {
	if !fn(entry) {
		return
	}
	if !entry.IsTree() {
		return
	}

	tree, _ := r.repo.Database.Load(entry.Oid())
	for _, item := range tree.(*database.Tree).Entries {
		r.traverseTree(item.(*database.Entry), func(object *database.Entry) bool {
			return fn(object)
		})
	}
}

func (r *RevList) loadCommit(oid string) *database.Commit {
	if oid == "" {
		return nil
	}
	if commit, exists := r.commits[oid]; exists {
		return commit
	}
	commitObj, _ := r.repo.Database.Load(oid)
	commit := commitObj.(*database.Commit)

	return commit
}

func (r *RevList) mark(oid string, flag RevListFlag) bool {
	if _, exists := r.flags[oid]; !exists {
		r.flags[oid] = map[RevListFlag]bool{}
	}
	exists := r.flags[oid][flag]
	r.flags[oid][flag] = true
	return !exists
}

func (r *RevList) isMarked(oid string, flag RevListFlag) bool {
	flags, ok := r.flags[oid]
	return ok && flags[flag]
}
