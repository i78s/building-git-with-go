package merge

import (
	"building-git/lib/database"
	"building-git/lib/pathutils"
	"building-git/lib/repository"
	"fmt"
	"path/filepath"
	"strings"
)

type Resolve struct {
	repo       *repository.Repository
	inputs     *Inputs
	leftDiff   map[string][2]database.TreeObject
	rightDiff  map[string][2]database.TreeObject
	cleanDiff  map[string][2]database.TreeObject
	conflicts  map[string][3]database.TreeObject
	untracked  map[string]database.TreeObject
	onProgress func(fn func() string)
}

func NewResolve(repo *repository.Repository, inputs *Inputs, onProgress func(fn func() string)) *Resolve {
	return &Resolve{
		repo:       repo,
		inputs:     inputs,
		onProgress: onProgress,
	}
}

func (r *Resolve) Execute() error {
	r.prepareTreeDiffs()

	migration := r.repo.Migration(r.cleanDiff)
	if err := migration.ApplyChanges(); err != nil {
		return err
	}

	r.addConflictsToIndex()
	r.writeUntrackedFiles()
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
	r.untracked = map[string]database.TreeObject{}

	for path, images := range r.rightDiff {
		oldItem, newItem := images[0], images[1]
		if newItem != nil && !newItem.IsNil() {
			r.fileDirConflict(path, r.leftDiff, r.inputs.LeftName)
		}
		r.samePathConflict(path, oldItem, newItem)
	}
	for path, images := range r.leftDiff {
		newItem := images[1]
		if newItem != nil && !newItem.IsNil() {
			r.fileDirConflict(path, r.rightDiff, r.inputs.RightName)
		}
	}
}

func (r *Resolve) samePathConflict(path string, base, right database.TreeObject) {
	if _, exists := r.conflicts[path]; exists {
		return
	}

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

	if left != nil && right != nil {
		r.onProgress(func() string {
			return fmt.Sprintf("Auto-merging %s", path)
		})
	}

	oidOk, oid := r.mergeBlobs(base, left, right)
	modeOk, mode := r.mergedModes(base, left, right)

	r.cleanDiff[path] = [2]database.TreeObject{left, database.NewEntry(oid, mode)}
	if oidOk && modeOk {
		return
	}

	r.conflicts[path] = [3]database.TreeObject{
		base,
		left,
		right,
	}
	r.logConflict([]string{path})
}

func (r *Resolve) mergeBlobs(base, left, right database.TreeObject) (bool, string) {
	result, err := r.merge3ForBlobs(base, left, right)
	if err == nil {
		return result.mergedCleanly, result.obj.Oid()
	}

	oids := []database.TreeObject{base, left, right}
	blobs := []string{}
	for _, obj := range oids {
		if obj == nil {
			blobs = append(blobs, "")
			continue
		}
		obj, _ := r.repo.Database.Load(obj.Oid())
		blobs = append(blobs, obj.String())
	}

	merge := Merge(blobs[0], blobs[1], blobs[2])
	data := merge.String(r.inputs.LeftName, r.inputs.RightName)
	blob := database.NewBlob(data)
	r.repo.Database.Store(blob)

	return merge.isClean(), blob.Oid()
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

func (r *Resolve) fileDirConflict(path string, diff map[string][2]database.TreeObject, name string) {
	for _, parent := range pathutils.Ascend(filepath.Dir(path)) {
		oldItem, newItem := diff[parent][0], diff[parent][1]
		if newItem == nil || newItem.IsNil() {
			continue
		}

		switch name {
		case r.inputs.LeftName:
			r.conflicts[parent] = [3]database.TreeObject{oldItem, newItem, nil}
		case r.inputs.RightName:
			r.conflicts[parent] = [3]database.TreeObject{oldItem, nil, newItem}
		}

		delete(r.cleanDiff, parent)
		rename := fmt.Sprintf("%s~%s", parent, name)
		r.untracked[rename] = newItem

		if _, exsists := diff[path]; !exsists {
			r.onProgress(func() string {
				return fmt.Sprintf("Adding %s", path)
			})
		}
		r.logConflict([]string{parent, rename})
	}
}

func (r *Resolve) addConflictsToIndex() {
	for path, items := range r.conflicts {
		r.repo.Index.AddConflictSet(path, items[:])
	}
}

func (r *Resolve) writeUntrackedFiles() {
	for path, item := range r.untracked {
		blob, _ := r.repo.Database.Load(item.Oid())
		r.repo.Workspace.WriteFile(path, []byte(blob.String()), 0, false)
	}
}

func (r *Resolve) logConflict(args []string) {
	path := args[0]
	rename := ""
	if len(args) > 1 {
		rename = args[1]
	}

	conflict := r.conflicts[path]
	base, left, right := conflict[0], conflict[1], conflict[2]

	if left != nil && right != nil {
		r.logLeftRightConflict(path)
	} else if base != nil && (left != nil || right != nil) {
		r.logModifyDeleteConflict(path, rename)
	} else {
		r.logFileDirectoryConflict(path, rename)
	}
}

func (r *Resolve) logLeftRightConflict(path string) {
	conflictType := "add/add"
	if r.conflicts[path][0] != nil {
		conflictType = "content"
	}
	r.onProgress(func() string {
		return fmt.Sprintf("CONFLICT (%s): Merge conflict in %s", conflictType, path)
	})
}

func (r *Resolve) logModifyDeleteConflict(path, rename string) {
	names := r.logBranchNames(path)
	deleted, modified := names[0], names[1]

	if rename != "" {
		rename = fmt.Sprintf(" at %s", rename)
	}
	r.onProgress(func() string {
		return strings.Join([]string{
			fmt.Sprintf("CONFLICT (modify/delete): %s", path),
			fmt.Sprintf("deleted in %s and modified in %s.", deleted, modified),
			fmt.Sprintf("Version %s of %s left in tree%s.", modified, path, rename),
		}, " ")
	})
}

func (r *Resolve) logFileDirectoryConflict(path, rename string) {
	conflictType := "directory/file"
	if r.conflicts[path][1] != nil {
		conflictType = "file/directory"
	}
	branch := r.logBranchNames(path)[0]

	r.onProgress(func() string {
		return strings.Join([]string{
			fmt.Sprintf("CONFLICT (%s): There is a directory", conflictType),
			fmt.Sprintf("with name %s in %s.", path, branch),
			fmt.Sprintf("Adding %s as %s", path, rename),
		}, " ")
	})
}

func (r *Resolve) logBranchNames(path string) [2]string {
	a, b := r.inputs.LeftName, r.inputs.RightName
	if r.conflicts[path][1] != nil {
		return [2]string{b, a}
	}
	return [2]string{a, b}
}
