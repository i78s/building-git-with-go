package merge

import (
	"building-git/lib/database"
	"building-git/lib/repository"
	"fmt"
	"strings"
)

type Resolve struct {
	repo      *repository.Repository
	inputs    *Inputs
	leftDiff  map[string][2]database.TreeObject
	rightDiff map[string][2]database.TreeObject
	cleanDiff map[string][2]database.TreeObject
	conflicts map[string][3]database.TreeObject
}

func NewResolve(repo *repository.Repository, inputs *Inputs) *Resolve {
	return &Resolve{repo: repo, inputs: inputs}
}

func (r *Resolve) Execute() error {
	r.prepareTreeDiffs()

	migration := r.repo.Migration(r.cleanDiff)
	if err := migration.ApplyChanges(); err != nil {
		return err
	}

	r.addConflictsToIndex()
	return nil
}

func (r *Resolve) prepareTreeDiffs() {
	baseOid := ""
	if len(r.inputs.BaseOids) > 0 {
		baseOid = r.inputs.BaseOids[0]
	}

	r.leftDiff = r.repo.Database.TreeDiff(baseOid, r.inputs.LeftOid, nil)
	r.rightDiff = r.repo.Database.TreeDiff(baseOid, r.inputs.RightOid, nil)
	r.cleanDiff = map[string][2]database.TreeObject{}
	r.conflicts = map[string][3]database.TreeObject{}

	for path, images := range r.rightDiff {
		r.samePathConflict(path, images[0], images[1])
	}
}

func (r *Resolve) samePathConflict(path string, base, right database.TreeObject) {
	if _, exists := r.leftDiff[path]; !exists {
		r.cleanDiff[path] = [2]database.TreeObject{
			base,
			right,
		}
		return
	}

	left := r.leftDiff[path][1]
	if left == right || // Both are nil
		left != nil && right != nil && left.Oid() == right.Oid() && left.Mode() == right.Mode() {
		return
	}

	oidOk, oid := r.mergeBlobs(base, left, right)
	modeOk, mode := r.mergedModes(base, left, right)

	r.cleanDiff[path] = [2]database.TreeObject{left, database.NewEntry(oid, mode)}
	if !(oidOk && modeOk) {
		r.conflicts[path] = [3]database.TreeObject{
			base,
			left,
			right,
		}
	}
}

func (r *Resolve) mergeBlobs(base, left, right database.TreeObject) (bool, string) {
	result, err := r.merge3ForBlobs(base, left, right)
	if err != nil {
		blob := database.NewBlob(r.mergedData(left, right))
		r.repo.Database.Store(blob)
		return false, blob.Oid()
	}
	return result.mergedCleanly, result.obj.Oid()
}

func (r *Resolve) mergedData(left, right database.TreeObject) string {
	var leftOid, rightOid string
	if left != nil && !left.IsNil() {
		leftOid = left.Oid()
	}
	if right != nil && !right.IsNil() {
		rightOid = right.Oid()
	}
	leftBlob, _ := r.repo.Database.Load(leftOid)
	rightBlob, _ := r.repo.Database.Load(rightOid)

	return strings.Join([]string{
		fmt.Sprintf("<<<<<<< %s\n", r.inputs.LeftName),
		leftBlob.String(),
		"=======\n",
		rightBlob.String(),
		fmt.Sprintf("<<<<<<< %s\n", r.inputs.RightName),
	}, "")
}

func (r *Resolve) mergedModes(base, left, right database.TreeObject) (bool, int) {
	result, err := r.merge3ForModes(base, left, right)
	if err != nil {
		return false, left.Mode()
	}
	return result.mergedCleanly, result.obj.Mode()
}

type merge3 struct {
	mergedCleanly bool
	obj           database.TreeObject
}

func (r *Resolve) merge3ForBlobs(base, left, right database.TreeObject) (merge3, error) {
	if left == nil || left.IsNil() {
		return merge3{false, right}, nil
	}
	if right == nil || right.IsNil() {
		return merge3{false, left}, nil
	}

	if base != nil && left.Oid() == base.Oid() || left.Oid() == right.Oid() {
		return merge3{true, right}, nil
	} else if base != nil && right.Oid() == base.Oid() {
		return merge3{true, left}, nil
	}
	return merge3{}, fmt.Errorf("detect conflict")
}

func (r *Resolve) merge3ForModes(base, left, right database.TreeObject) (merge3, error) {
	if left == nil || left.IsNil() {
		return merge3{false, right}, nil
	}
	if right == nil || right.IsNil() {
		return merge3{false, left}, nil
	}

	if base != nil && left.Mode() == base.Mode() || left.Mode() == right.Mode() {
		return merge3{true, right}, nil
	} else if base != nil && right.Mode() == base.Mode() {
		return merge3{true, left}, nil
	}
	return merge3{}, fmt.Errorf("detect conflict")
}

func (r *Resolve) addConflictsToIndex() {
	for path, items := range r.conflicts {
		r.repo.Index.AddConflictSet(path, items[:])
	}
}
