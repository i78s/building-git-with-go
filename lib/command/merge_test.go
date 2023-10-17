package command

import (
	"building-git/lib/database"
	"bytes"
	"os"
	"strings"
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

		os.Setenv("GIT_AUTHOR_NAME", "A. U. Thor")
		os.Setenv("GIT_AUTHOR_EMAIL", "author@example.com")
		defer os.Unsetenv("GIT_AUTHOR_NAME")
		defer os.Unsetenv("GIT_AUTHOR_EMAIL")
		options := MergeOption{}
		stdin := strings.NewReader("merge topic branch")
		mergeCmd, _ := NewMerge(tmpDir, []string{"topic"}, options, stdin, stdout, stderr)
		mergeCmd.Run()

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

		assertStatus(t, tmpDir, stdout, stderr, "")
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
