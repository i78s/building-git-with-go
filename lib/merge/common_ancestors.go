package merge

import "building-git/lib/database"

type CommonAncestors struct {
	db    *database.Database
	flags map[string]map[string]bool
	queue []*database.Commit
}

func NewCommonAncestors(db *database.Database, one, two string) *CommonAncestors {
	ca := &CommonAncestors{
		db:    db,
		flags: map[string]map[string]bool{},
		queue: make([]*database.Commit, 0),
	}

	commitOne, _ := db.Load(one)
	ca.insertByDate(commitOne.(*database.Commit))
	ca.setFlag(commitOne.Oid(), "parent1")

	commitTwo, _ := db.Load(two)
	ca.insertByDate(commitTwo.(*database.Commit))
	ca.setFlag(commitTwo.Oid(), "parent2")

	return ca
}

func (ca *CommonAncestors) Find() string {
	for len(ca.queue) > 0 {
		commit := ca.queue[0]
		ca.queue = ca.queue[1:]
		flags := ca.flags[commit.Oid()]

		if flags["parent1"] && flags["parent2"] {
			return commit.Oid()
		}

		ca.addParents(commit, flags)
	}

	return ""
}

func (ca *CommonAncestors) addParents(commit *database.Commit, flags map[string]bool) {
	if commit.Parent() == "" {
		return
	}

	parent, _ := ca.db.Load(commit.Parent())
	existingFlags := ca.flags[parent.Oid()]

	isSuperset := true
	for flag := range flags {
		if !existingFlags[flag] {
			isSuperset = false
			break
		}
	}
	if isSuperset {
		return
	}

	for flag := range flags {
		ca.setFlag(parent.Oid(), flag)
	}
	ca.insertByDate(parent.(*database.Commit))
}

func (ca *CommonAncestors) insertByDate(commit *database.Commit) {
	index := -1
	for i, c := range ca.queue {
		if c.Date().Before(commit.Date()) {
			index = i
			break
		}
	}
	if index == -1 {
		ca.queue = append(ca.queue, commit)
		return
	}
	ca.queue = append(ca.queue[:index+1], ca.queue[index:]...)
	ca.queue[index] = commit
}

func (ca *CommonAncestors) setFlag(oid, flag string) {
	if _, ok := ca.flags[oid]; !ok {
		ca.flags[oid] = make(map[string]bool)
	}
	ca.flags[oid][flag] = true
}
