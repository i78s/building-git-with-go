package lib

import (
	"building-git/lib/database"
	"path/filepath"
)

type Repository struct {
	Database  *database.Database
	Index     *database.Index
	Refs      *database.Refs
	Workspace *Workspace
}

func NewRepository(rootPath string) *Repository {
	gitPath := filepath.Join(rootPath, ".git")
	return &Repository{
		Database:  database.NewDatabase(filepath.Join(gitPath, "objects")),
		Index:     database.NewIndex(filepath.Join(gitPath, "index")),
		Refs:      database.NewRefs(gitPath),
		Workspace: NewWorkspace(rootPath),
	}
}
