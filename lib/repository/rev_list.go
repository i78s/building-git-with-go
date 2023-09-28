package repository

import "building-git/lib/database"

type RevList struct {
	repo  *Repository
	start string
}

func NewRevList(repo *Repository, start string) *RevList {
	return &RevList{
		repo:  repo,
		start: start,
	}
}

func (r *RevList) EachCommit() chan *database.Commit {
	ch := make(chan *database.Commit)

	go func() {
		oid, _ := NewRevision(r.repo, r.start).Resolve(COMMIT)
		for oid != "" {
			commitObj, _ := r.repo.Database.Load(oid)
			commit := commitObj.(*database.Commit)
			ch <- commit
			oid = commit.Parent()
		}
		close(ch)
	}()

	return ch
}
