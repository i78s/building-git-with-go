package merge

import "building-git/lib/repository"

type Inputs struct {
	LeftName  string
	RightName string
	LeftOid   string
	RightOid  string
	BaseOids  []string
	repo      *repository.Repository
}

func NewInputs(repo *repository.Repository, leftName, rightName string) (*Inputs, error) {
	inputs := &Inputs{
		repo:      repo,
		LeftName:  leftName,
		RightName: rightName,
	}

	LeftOid, err := inputs.resolveRev(leftName)
	if err != nil {
		return nil, err
	}
	inputs.LeftOid = LeftOid

	rightOid, err := inputs.resolveRev(rightName)
	if err != nil {
		return nil, err
	}
	inputs.RightOid = rightOid

	common := NewBases(repo.Database, LeftOid, rightOid)
	inputs.BaseOids = common.Find()

	return inputs, nil
}

func (i *Inputs) resolveRev(rev string) (string, error) {
	oid, err := repository.NewRevision(i.repo, rev).Resolve(repository.COMMIT)
	if err != nil {
		return "", err
	}
	return oid, nil
}
