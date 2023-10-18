package merge

import "building-git/lib/database"

type Bases struct {
	db        *database.Database
	common    *CommonAncestors
	redundant map[string]bool
	commits   []string
}

func NewBases(db *database.Database, one, two string) *Bases {
	return &Bases{
		db:        db,
		common:    NewCommonAncestors(db, one, []string{two}),
		redundant: map[string]bool{},
		commits:   make([]string, 0),
	}
}

func (b *Bases) Find() []string {
	b.commits = b.common.Find()
	if len(b.commits) <= 1 {
		return b.commits
	}

	for _, commit := range b.commits {
		b.filterCommit(commit)
	}

	result := []string{}
	for _, commit := range b.commits {
		if b.redundant[commit] {
			continue
		}
		result = append(result, commit)
	}
	return result
}

func (b *Bases) filterCommit(commit string) {
	if b.redundant[commit] {
		return
	}

	others := []string{}
	for _, other := range b.commits {
		if other != commit && !b.redundant[other] {
			others = append(others, other)
		}
	}

	common := NewCommonAncestors(b.db, commit, others)
	common.Find()

	if common.IsMarked(commit, "parent2") {
		b.redundant[commit] = true
	}

	for _, other := range others {
		if common.IsMarked(other, "parent1") {
			b.redundant[other] = true
		}
	}
}
