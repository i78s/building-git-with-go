package command

import (
	"building-git/lib/database"
	"bytes"
	"os"
	"testing"
)

func TestMergeUnconflictedMergeWithTwoFiles(t *testing.T) {
	//   A   B   M
	//   o---o---o
	//    \     /
	//     `---o
	//         C

	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		commitTree(t, tmpDir, "root", map[string]string{
			"f.txt": "1",
			"g.txt": "1",
		})

		brunchCmd, _ := NewBranch(tmpDir, []string{"topic"}, BranchOption{}, new(bytes.Buffer), new(bytes.Buffer))
		brunchCmd.Run()
		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "topic")
		commitTree(t, tmpDir, "right", map[string]string{
			"g.txt": "2",
		})

		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "master")
		commitTree(t, tmpDir, "left", map[string]string{
			"f.txt": "2",
		})

		options := MergeOption{}
		mergeCommit(t, tmpDir, "topic", "merge topic branch", options, stdout, stderr)

		return
	}

	t.Run("puts the combined changes in the workspace", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "2",
			"g.txt": "2",
		})
	})

	t.Run("leaves the status clean", func(t *testing.T) {
		tmpDir, stdout, stderr := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertGitStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("writes a commit with the old HEAD and the merged commit as parents", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		gitObj, _ := loadCommit(t, tmpDir, "@")
		commit := gitObj.(*database.Commit)
		oldHead, _ := loadCommit(t, tmpDir, "@^")
		mergeHead, _ := loadCommit(t, tmpDir, "topic")

		expected := []string{oldHead.Oid(), mergeHead.Oid()}
		if commit.Parents[0] != expected[0] || commit.Parents[1] != expected[1] {
			t.Errorf("want %q, but got %q", expected, commit.Parents)
		}
	})
}

func TestMergeMultipleCommonAncestors(t *testing.T) {
	//   A   B   C       M1  H   M2
	//   o---o---o-------o---o---o
	//        \         /       /
	//         o---o---o G     /
	//         D  E \         /
	//               `-------o
	//                       F

	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		commitTree(t, tmpDir, "A", map[string]string{
			"f.txt": "1",
		})
		commitTree(t, tmpDir, "B", map[string]string{
			"f.txt": "2",
		})
		commitTree(t, tmpDir, "C", map[string]string{
			"f.txt": "3",
		})

		brunchCmd, _ := NewBranch(tmpDir, []string{"topic", "master^"}, BranchOption{}, new(bytes.Buffer), new(bytes.Buffer))
		brunchCmd.Run()
		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "topic")
		commitTree(t, tmpDir, "D", map[string]string{
			"g.txt": "1",
		})
		commitTree(t, tmpDir, "E", map[string]string{
			"g.txt": "2",
		})
		commitTree(t, tmpDir, "F", map[string]string{
			"g.txt": "3",
		})

		brunchCmd, _ = NewBranch(tmpDir, []string{"joiner", "topic^"}, BranchOption{}, new(bytes.Buffer), new(bytes.Buffer))
		brunchCmd.Run()
		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "joiner")
		commitTree(t, tmpDir, "G", map[string]string{
			"h.txt": "1",
		})

		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "master")

		return
	}

	t.Run("performs the first merge", func(t *testing.T) {
		tmpDir, stdout, stderr := setUp(t)
		defer os.RemoveAll(tmpDir)

		options := MergeOption{}
		status := mergeCommit(t, tmpDir, "joiner", "merge joiner", options, stdout, stderr)

		if status != 0 {
			t.Errorf("want %q, but got %q", 0, status)
		}

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "3",
			"g.txt": "2",
			"h.txt": "1",
		})

		assertGitStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("performs the second merge", func(t *testing.T) {
		tmpDir, stdout, stderr := setUp(t)
		defer os.RemoveAll(tmpDir)

		options := MergeOption{}
		mergeCommit(t, tmpDir, "joiner", "merge joiner", options, stdout, stderr)

		commitTree(t, tmpDir, "H", map[string]string{
			"f.txt": "4",
		})

		status := mergeCommit(t, tmpDir, "topic", "merge topic", options, stdout, stderr)
		if status != 0 {
			t.Errorf("want %q, but got %q", 0, status)
		}

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "4",
			"g.txt": "3",
			"h.txt": "1",
		})

		assertGitStatus(t, tmpDir, stdout, stderr, "")
	})
}
