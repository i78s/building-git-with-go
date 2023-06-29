package lib

import (
	"building-git/lib/database"
	"building-git/lib/index"
	"path/filepath"
)

type Repository struct {
	Database  *database.Database
	Index     *index.Index
	Refs      *database.Refs
	Workspace *Workspace
}

func NewRepository(rootPath string) *Repository {
	gitPath := filepath.Join(rootPath, ".git")
	return &Repository{
		Database:  database.NewDatabase(filepath.Join(gitPath, "objects")),
		Index:     index.NewIndex(filepath.Join(gitPath, "index")),
		Refs:      database.NewRefs(gitPath),
		Workspace: NewWorkspace(rootPath),
	}
}
