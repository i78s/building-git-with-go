package merge

import (
	"building-git/lib/repository"
)

type Resolve struct {
	repo   *repository.Repository
	inputs *Inputs
}

func NewResolve(repo *repository.Repository, inputs *Inputs) *Resolve {
	return &Resolve{repo: repo, inputs: inputs}
}

func (r *Resolve) Execute() error {
	baseOid := ""
	if len(r.inputs.BaseOids) > 0 {
		baseOid = r.inputs.BaseOids[0]
	}
	treeDiff := r.repo.Database.TreeDiff(baseOid, r.inputs.RightOid, nil)
	migration := r.repo.Migration(treeDiff)

	if err := migration.ApplyChanges(); err != nil {
		return err
	}
	return nil
}
