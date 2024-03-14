package repository

import (
	"building-git/lib/config"
	"building-git/lib/database"
	"building-git/lib/index"

	"path/filepath"
)

type Repository struct {
	GitPath       string
	Config        *config.Stack
	Database      *database.Database
	Index         *index.Index
	Refs          *Refs
	Workspace     *Workspace
	PendingCommit *PendingCommit
}

func NewRepository(rootPath string) *Repository {
	gitPath := filepath.Join(rootPath, ".git")
	return &Repository{
		GitPath:       gitPath,
		Config:        config.NewStack(gitPath),
		Database:      database.NewDatabase(filepath.Join(gitPath, "objects")),
		Index:         index.NewIndex(filepath.Join(gitPath, "index")),
		Refs:          NewRefs(gitPath),
		Workspace:     NewWorkspace(rootPath),
		PendingCommit: NewPendingCommit(gitPath),
	}
}

func (r *Repository) HardReset(oid string) {
	NewHardReset(r, oid).Execute()
}

func (r *Repository) Status(commitOid string) (*Status, error) {
	status, err := NewStatus(r, commitOid)
	return status, err
}

func (r *Repository) Migration(treeDiff map[string][2]database.TreeObject) *Migration {
	return NewMigration(r, treeDiff)
}
