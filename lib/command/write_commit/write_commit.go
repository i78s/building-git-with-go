package write_commit

import (
	"building-git/lib/database"
	"building-git/lib/repository"
	"fmt"
	"log"
	"os"
	"time"
)

const CONFLICT_MESSAGE = `hint: Fix them up in the work tree, and then use 'jit add <file>'
hint: as appropriate to mark resolution and make a commit.
fatal: Exiting because of an unresolved conflict.`

type WriteCommit struct {
	repo *repository.Repository
}

func NewWriteCommit(repo *repository.Repository) *WriteCommit {
	return &WriteCommit{
		repo: repo,
	}
}

func (wc *WriteCommit) WriteCommit(parents []string, message string, now time.Time) *database.Commit {
	tree := wc.writeTree()
	name, exists := os.LookupEnv("GIT_AUTHOR_NAME")
	if !exists {
		log.Fatalf("GIT_AUTHOR_NAME is not set")
	}
	email, exists := os.LookupEnv("GIT_AUTHOR_EMAIL")
	if !exists {
		log.Fatalf("GIT_AUTHOR_EMAIL is not set")
	}

	author := database.NewAuthor(name, email, now)

	commit := database.NewCommit(parents, tree.Oid(), author, message)
	wc.repo.Database.Store(commit)
	wc.repo.Refs.UpdateHead(commit.Oid())

	return commit
}

func (wc *WriteCommit) writeTree() *database.Tree {
	root := database.BuildTree(wc.repo.Index.EachEntry())
	root.Traverse(func(t database.TreeObject) {
		if gitObj, ok := t.(database.GitObject); ok {
			if err := wc.repo.Database.Store(gitObj); err != nil {
				log.Fatalf("Failed to store object: %v", err)
			}
		} else {
			log.Fatalf("Object does not implement GitObject interface")
		}
	})
	return root
}

func (wc *WriteCommit) PendingCommit() *repository.PendingCommit {
	return wc.repo.PendingCommit
}

func (wc *WriteCommit) ResumeMerge() error {
	err := wc.HandleConflictedIndex()
	if err != nil {
		return err
	}
	head, _ := wc.repo.Refs.ReadHead()
	mergeOid, err := wc.PendingCommit().MergeOID()
	if err != nil {
		return err
	}
	parants := []string{head, mergeOid}
	message, err := wc.PendingCommit().MergeMessage()
	if err != nil {
		return err
	}
	wc.WriteCommit(parants, message, time.Now())
	wc.PendingCommit().Clear()
	return nil
}

func (wc *WriteCommit) HandleConflictedIndex() error {
	if !wc.repo.Index.IsConflict() {
		return nil
	}
	message := "Committing is not possible because you have unmerged files."
	return fmt.Errorf(`error: %s
%s`, message, CONFLICT_MESSAGE)
}
