package repository

import (
	"building-git/lib/database"
	"sort"
)

type RevListFlag int

const (
	added RevListFlag = iota
	seen
)

type RevList struct {
	repo    *Repository
	commits map[string]*database.Commit
	flags   map[string]map[RevListFlag]bool
	queue   []*database.Commit
}

func NewRevList(repo *Repository, revs []string) *RevList {
	revList := &RevList{
		repo:    repo,
		commits: map[string]*database.Commit{},
		flags:   map[string]map[RevListFlag]bool{},
		queue:   make([]*database.Commit, 0),
	}

	for _, rev := range revs {
		revList.handleRevision(rev)
	}
	if len(revList.queue) == 0 {
		revList.handleRevision(HEAD)
	}

	return revList
}

func (r *RevList) Each() []*database.Commit {
	var commits []*database.Commit
	r.traverseCommits(func(c *database.Commit) {
		commits = append(commits, c)
	})
	return commits
}

func (r *RevList) handleRevision(rev string) {
	oid, _ := NewRevision(r.repo, rev).Resolve(COMMIT)
	commitObj, _ := r.repo.Database.Load(oid)
	commit := commitObj.(*database.Commit)

	r.enqueueCommit(commit)
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

func (r *RevList) addParents(commit *database.Commit) {
	if !r.mark(commit.Oid(), added) {
		return
	}
	parent := r.loadCommit(commit.Parent())
	if parent != nil {
		r.enqueueCommit(parent)
	}
}

func (r *RevList) traverseCommits(fn func(*database.Commit)) {
	for len(r.queue) > 0 {
		commit := r.queue[0]
		r.queue = r.queue[1:]
		r.addParents(commit)
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
