package repository

import (
	"building-git/lib/database"
	"building-git/lib/index"

	"path/filepath"
)

type Repository struct {
	Database  *database.Database
	Index     *index.Index
	Refs      *Refs
	Workspace *Workspace
}

func NewRepository(rootPath string) *Repository {
	gitPath := filepath.Join(rootPath, ".git")
	return &Repository{
		Database:  database.NewDatabase(filepath.Join(gitPath, "objects")),
		Index:     index.NewIndex(filepath.Join(gitPath, "index")),
		Refs:      NewRefs(gitPath),
		Workspace: NewWorkspace(rootPath),
	}
}

func (r *Repository) Status() (*Status, error) {
	status, err := NewStatus(r)
	return status, err
}
