package write_commit

import (
	"building-git/lib/database"
	"building-git/lib/repository"
	"log"
	"os"
	"time"
)

func WriteCommit(repo *repository.Repository, parents []string, message string, now time.Time) *database.Commit {
	tree := writeTree(repo)
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
	repo.Database.Store(commit)
	repo.Refs.UpdateHead(commit.Oid())

	return commit
}

func writeTree(repo *repository.Repository) *database.Tree {
	root := database.BuildTree(repo.Index.EachEntry())
	root.Traverse(func(t database.TreeObject) {
		if gitObj, ok := t.(database.GitObject); ok {
			if err := repo.Database.Store(gitObj); err != nil {
				log.Fatalf("Failed to store object: %v", err)
			}
		} else {
			log.Fatalf("Object does not implement GitObject interface")
		}
	})
	return root
}
