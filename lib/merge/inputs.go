package merge

import "building-git/lib/repository"

type Inputs struct {
	leftName  string
	rightName string
	leftOid   string
	rightOid  string
	baseOids  []string
	repo      *repository.Repository
}

func NewInputs(repo *repository.Repository, leftName, rightName string) (*Inputs, error) {
	inputs := &Inputs{
		repo:      repo,
		leftName:  leftName,
		rightName: rightName,
	}

	leftOid, err := inputs.resolveRev(leftName)
	if err != nil {
		return nil, err
	}
	inputs.leftOid = leftOid

	rightOid, err := inputs.resolveRev(rightName)
	if err != nil {
		return nil, err
	}
	inputs.rightOid = rightOid

	common := NewBases(repo.Database, leftOid, rightOid)
	inputs.baseOids = common.Find()

	return inputs, nil
}

func (i *Inputs) LeftName() string {
	return i.leftName
}

func (i *Inputs) RightName() string {
	return i.rightName
}

func (i *Inputs) LeftOid() string {
	return i.leftOid
}

func (i *Inputs) RightOid() string {
	return i.rightOid
}

func (i *Inputs) BaseOids() []string {
	return i.baseOids
}

func (i *Inputs) IsAlreadyMerged() bool {
	return len(i.baseOids) == 1 && i.baseOids[0] == i.rightOid
}

func (i *Inputs) IsFastForward() bool {
	return len(i.baseOids) == 1 && i.baseOids[0] == i.leftOid
}

func (i *Inputs) resolveRev(rev string) (string, error) {
	oid, err := repository.NewRevision(i.repo, rev).Resolve(repository.COMMIT)
	if err != nil {
		return "", err
	}
	return oid, nil
}

type CherryPick struct {
	leftName  string
	rightName string
	leftOid   string
	rightOid  string
	baseOids  []string
	repo      *repository.Repository
}

func NewCherryPick(repo *repository.Repository, leftName, rightName, leftOid, rightOid string, baseOids []string) *CherryPick {
	return &CherryPick{
		repo:      repo,
		leftName:  leftName,
		rightName: rightName,
		leftOid:   leftOid,
		rightOid:  rightOid,
		baseOids:  baseOids,
	}
}

func (c *CherryPick) LeftName() string {
	return c.leftName
}

func (c *CherryPick) RightName() string {
	return c.rightName
}

func (c *CherryPick) LeftOid() string {
	return c.leftOid
}

func (c *CherryPick) RightOid() string {
	return c.rightOid
}

func (c *CherryPick) BaseOids() []string {
	return c.baseOids
}
