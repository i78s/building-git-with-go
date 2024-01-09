package repository

type HardReset struct {
	oid    string
	repo   *Repository
	status *Status
}

func NewHardReset(repo *Repository, oid string) *HardReset {
	status, _ := repo.Status(oid)
	hr := &HardReset{
		oid:    oid,
		repo:   repo,
		status: status,
	}
	return hr
}

func (hr *HardReset) Execute() {
	changed := []string{}
	hr.status.Changed.Iterate(func(path string, _ struct{}) {
		changed = append(changed, path)
	})
	for _, path := range changed {
		hr.resetPath(path)
	}
}

func (hr *HardReset) resetPath(path string) {
	hr.repo.Index.Remove(path)
	hr.repo.Workspace.Remove(path)

	entry := hr.status.HeadTree[path]
	if entry == nil || entry.IsNil() {
		return
	}

	blob, _ := hr.repo.Database.Load(entry.Oid())
	hr.repo.Workspace.WriteFile(path, []byte(blob.String()), entry.Mode(), true)

	stat, _ := hr.repo.Workspace.StatFile(path)
	hr.repo.Index.Add(path, entry.Oid(), stat)
}
