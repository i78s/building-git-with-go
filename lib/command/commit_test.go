package command

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func setUpForTestCommittingToBranches(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
	tmpDir, stdout, stderr = setupTestEnvironment(t)

	messages := []string{"first", "second", "third"}
	for _, message := range messages {
		writeFile(t, tmpDir, "file.txt", message)
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
		commit(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), message, time.Now())
	}

	brunchCmd, _ := NewBranch(tmpDir, []string{"topic"}, BranchOption{}, new(bytes.Buffer), new(bytes.Buffer))
	brunchCmd.Run()
	checkout(tmpDir, stdout, stderr, "topic")

	return
}

func commitChange(t *testing.T, tmpDir, content string) {
	writeFile(t, tmpDir, "file.txt", content)
	Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
	commit(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), content, time.Now())
}

func TestCommittingToBranchesOnBranch(t *testing.T) {
	t.Run("advances a branch pointer", func(t *testing.T) {
		tmpDir, _, _ := setUpForTestCommittingToBranches(t)
		defer os.RemoveAll(tmpDir)

		r := repo(t, tmpDir)
		headBefore, _ := r.Refs.ReadRef("HEAD")
		commitChange(t, tmpDir, "change")
		headAfter, _ := r.Refs.ReadRef("HEAD")
		branchAfter, _ := r.Refs.ReadRef("topic")

		if headBefore == headAfter {
			t.Errorf("want %q, but got %q", headAfter, headBefore)
		}

		if headAfter != branchAfter {
			t.Errorf("want %q, but got %q", headAfter, branchAfter)
		}

		expected, _ := resolveRevision(t, tmpDir, "@^")
		if headBefore != expected {
			t.Errorf("want %q, but got %q", expected, headBefore)
		}
	})
}

func TestCommittingToBranchesWithDetachedHEAD(t *testing.T) {
	t.Run("advances HEAD", func(t *testing.T) {
		tmpDir, _, _ := setUpForTestCommittingToBranches(t)
		defer os.RemoveAll(tmpDir)
		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "@")

		r := repo(t, tmpDir)
		headBefore, _ := r.Refs.ReadRef("HEAD")
		commitChange(t, tmpDir, "change")
		headAfter, _ := r.Refs.ReadRef("HEAD")

		if headBefore == headAfter {
			t.Errorf("want %q, but got %q", headAfter, headBefore)
		}
	})

	t.Run("does not advance the detached branch", func(t *testing.T) {
		tmpDir, _, _ := setUpForTestCommittingToBranches(t)
		defer os.RemoveAll(tmpDir)
		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "@")

		r := repo(t, tmpDir)
		headBefore, _ := r.Refs.ReadRef("topic")
		commitChange(t, tmpDir, "change")
		branchAfter, _ := r.Refs.ReadRef("topic")

		if headBefore != branchAfter {
			t.Errorf("want %q, but got %q", headBefore, branchAfter)
		}
	})

	t.Run("leaves HEAD a commit ahead of the branch", func(t *testing.T) {
		tmpDir, _, _ := setUpForTestCommittingToBranches(t)
		defer os.RemoveAll(tmpDir)
		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "@")

		commitChange(t, tmpDir, "change")
		r := repo(t, tmpDir)
		branchAfter, _ := r.Refs.ReadRef("topic")

		expected, _ := resolveRevision(t, tmpDir, "@^")
		if branchAfter != expected {
			t.Errorf("want %q, but got %q", expected, branchAfter)
		}
	})

}

func TestCommittingToBranchesWithConcurrentBranches(t *testing.T) {
	t.Run("advances HEAD", func(t *testing.T) {
		tmpDir, _, _ := setUpForTestCommittingToBranches(t)
		defer os.RemoveAll(tmpDir)
		brunchCmd, _ := NewBranch(tmpDir, []string{"fork", "@^"}, BranchOption{}, new(bytes.Buffer), new(bytes.Buffer))
		brunchCmd.Run()

		commitChange(t, tmpDir, "A")
		commitChange(t, tmpDir, "B")
		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "fork")
		commitChange(t, tmpDir, "C")

		rev1, _ := resolveRevision(t, tmpDir, "topic")
		rev2, _ := resolveRevision(t, tmpDir, "fork")
		if rev1 == rev2 {
			t.Errorf("want %q, but got %q", rev1, rev2)
		}

		topic, _ := resolveRevision(t, tmpDir, "topic")
		fork, _ := resolveRevision(t, tmpDir, "fork")
		if topic == fork {
			t.Errorf("want %q, but got %q", topic, fork)
		}

		topicThreeCommitsAgo, _ := resolveRevision(t, tmpDir, "topic~3")
		forkParentCommit, _ := resolveRevision(t, tmpDir, "fork^")
		if topicThreeCommitsAgo != forkParentCommit {
			t.Errorf("want %q, but got %q", topic, fork)
		}
	})
}
