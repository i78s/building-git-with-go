package merge

import "building-git/lib/database"

type CommonAncestors struct {
	db      *database.Database
	flags   map[string]map[string]bool
	queue   []*database.Commit
	results []*database.Commit
}

func NewCommonAncestors(db *database.Database, one string, twos []string) *CommonAncestors {
	ca := &CommonAncestors{
		db:      db,
		flags:   map[string]map[string]bool{},
		queue:   make([]*database.Commit, 0),
		results: make([]*database.Commit, 0),
	}

	commitOne, _ := db.Load(one)
	ca.queue = ca.insertByDate(ca.queue, commitOne.(*database.Commit))
	ca.setFlag(commitOne.Oid(), "parent1")

	for _, two := range twos {
		commitTwo, _ := db.Load(two)
		ca.queue = ca.insertByDate(ca.queue, commitTwo.(*database.Commit))
		ca.setFlag(commitTwo.Oid(), "parent2")
	}

	return ca
}

func (ca *CommonAncestors) Find() []string {
	for !ca.isAllStale() {
		ca.processQueue()
	}

	oids := []string{}
	for _, c := range ca.results {
		oid := c.Oid()
		if !ca.IsMarked(oid, "stale") {
			oids = append(oids, oid)
		}
	}
	return oids
}

func (ca *CommonAncestors) IsMarked(oid string, flag string) bool {
	flags, exists := ca.flags[oid]
	if !exists {
		return false
	}
	return flags[flag]
}

func (ca *CommonAncestors) isAllStale() bool {
	for _, c := range ca.queue {
		if !ca.IsMarked(c.Oid(), "stale") {
			return false
		}
	}
	return true
}

func (ca *CommonAncestors) processQueue() {
	commit := ca.queue[0]
	ca.queue = ca.queue[1:]
	flags := ca.flags[commit.Oid()]

	if flags["parent1"] && flags["parent2"] && len(flags) == 2 {
		ca.setFlag(commit.Oid(), "result")
		ca.results = ca.insertByDate(ca.results, commit)

		newFlags := make(map[string]bool)
		for flag := range flags {
			newFlags[flag] = true
		}
		newFlags["stale"] = true
		ca.addParents(commit, newFlags)
	} else {
		ca.addParents(commit, flags)
	}
}

func (ca *CommonAncestors) addParents(commit *database.Commit, flags map[string]bool) {
	for _, parent := range commit.Parents {
		existingFlags := ca.flags[parent]

		isSuperset := true
		for flag := range flags {
			if !existingFlags[flag] {
				isSuperset = false
				break
			}
		}
		if isSuperset {
			continue
		}

		for flag := range flags {
			ca.setFlag(parent, flag)
		}
		gitObj, _ := ca.db.Load(parent)
		ca.queue = ca.insertByDate(ca.queue, gitObj.(*database.Commit))
	}
}

func (ca *CommonAncestors) insertByDate(list []*database.Commit, commit *database.Commit) []*database.Commit {
	index := -1
	for i, c := range list {
		if c.Date().Before(commit.Date()) {
			index = i
			break
		}
	}
	if index == -1 {
		return append(list, commit)
	}
	newList := append(list[:index+1], list[index:]...)
	newList[index] = commit
	return newList
}

func (ca *CommonAncestors) setFlag(oid, flag string) {
	if _, ok := ca.flags[oid]; !ok {
		ca.flags[oid] = make(map[string]bool)
	}
	ca.flags[oid][flag] = true
}
