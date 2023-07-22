package repository

import (
	"building-git/lib"
	"building-git/lib/database"
	"building-git/lib/index"
	"path/filepath"
)

type Repository struct {
	Database  *database.Database
	Index     *index.Index
	Refs      *lib.Refs
	Workspace *lib.Workspace
}

func NewRepository(rootPath string) *Repository {
	gitPath := filepath.Join(rootPath, ".git")
	return &Repository{
		Database:  database.NewDatabase(filepath.Join(gitPath, "objects")),
		Index:     index.NewIndex(filepath.Join(gitPath, "index")),
		Refs:      lib.NewRefs(gitPath),
		Workspace: lib.NewWorkspace(rootPath),
	}
}

func (r *Repository) Status() (*Status, error) {
	status, err := NewStatus(r)
	return status, err
}
