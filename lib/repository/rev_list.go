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
	limited bool
	prune   []string
	diffs   map[[2]string]map[string][2]database.TreeObject
	output  []*database.Commit
	filter  *database.PathFilter
}

func NewRevList(repo *Repository, revs []string) *RevList {
	revList := &RevList{
		repo:    repo,
		commits: map[string]*database.Commit{},
		flags:   map[string]map[RevListFlag]bool{},
		queue:   make([]*database.Commit, 0),
		prune:   make([]string, 0),
		diffs:   make(map[[2]string]map[string][2]database.TreeObject),
		output:  make([]*database.Commit, 0),
	}

	for _, rev := range revs {
		revList.handleRevision(rev)
	}
	if len(revList.queue) == 0 {
		revList.handleRevision(HEAD)
	}
	revList.filter = database.PathFilterBuild(revList.prune)

	return revList
}

func (r *RevList) Each() []*database.Commit {
	if r.limited {
		r.limitList()
	}

	var commits []*database.Commit
	r.traverseCommits(func(c *database.Commit) {
		commits = append(commits, c)
	})
	return commits
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

func (r *RevList) handleRevision(rev string) {
	if stat, _ := r.repo.Workspace.StatFile(rev); stat != nil {
		r.prune = append(r.prune, rev)
	} else if match := RANGE.FindStringSubmatch(rev); match != nil {
		r.setStartPoint(match[1], false)
		r.setStartPoint(match[2], true)
	} else if match := EXCLUDE.FindStringSubmatch(rev); match != nil {
		r.setStartPoint(match[1], false)
	} else {
		r.setStartPoint(rev, true)
	}
}

func (r *RevList) setStartPoint(rev string, interesting bool) {
	if rev == "" {
		rev = HEAD
	}

	oid, _ := NewRevision(r.repo, rev).Resolve(COMMIT)
	commit := r.loadCommit(oid)

	r.enqueueCommit(commit)

	if !interesting {
		r.limited = true
		r.mark(oid, uninteresting)
		r.markParentsUninteresting(commit)
	}
}

func (r *RevList) enqueueCommit(commit *database.Commit) {
	if !r.mark(commit.Oid(), seen) {
		return
	}

	index := sort.Search(len(r.queue), func(i int) bool {
		return r.queue[i].Date().Before(commit.Date())
	})

	r.queue = append(r.queue[:index], append([]*database.Commit{commit}, r.queue[index:]...)...)
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
	if !r.mark(commit.Oid(), added) {
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
		fn(commit)
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
