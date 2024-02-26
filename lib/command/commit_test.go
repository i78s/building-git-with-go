package command

import (
	"building-git/lib/command/write_commit"
	"building-git/lib/database"
	"building-git/lib/editor"
	"building-git/lib/repository"
	"bytes"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func setUpForTestCommittingToBranches(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
	tmpDir, stdout, stderr = setupTestEnvironment(t)

	messages := []string{"first", "second", "third"}
	for _, message := range messages {
		writeFile(t, tmpDir, "file.txt", message)
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
		options := CommitOption{
			ReadOption: write_commit.ReadOption{
				Message: message,
			},
		}
		commit(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), options, time.Now())
	}

	brunchCmd, _ := NewBranch(tmpDir, []string{"topic"}, BranchOption{}, new(bytes.Buffer), new(bytes.Buffer))
	brunchCmd.Run()
	checkout(tmpDir, stdout, stderr, "topic")

	return
}

func commitChange(t *testing.T, tmpDir, content string) {
	writeFile(t, tmpDir, "file.txt", content)
	Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
	options := CommitOption{
		ReadOption: write_commit.ReadOption{
			Message: content,
		},
	}
	commit(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), options, time.Now())
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

func TestCommitReusingMessages(t *testing.T) {
	var setUp = func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		writeFile(t, tmpDir, "file.txt", "1")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
		options := CommitOption{
			ReadOption: write_commit.ReadOption{
				Message: "first",
			},
		}
		commit(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), options, time.Now())

		return
	}
	t.Run("uses the message from another commit", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "file.txt", "2")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
		options := CommitOption{
			Edit:  false,
			Reuse: "@",
		}
		commit(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), options, time.Now())

		revs := repository.NewRevList(repo(t, tmpDir), []string{"HEAD"}, repository.RevListOption{})

		actual := []string{}
		for _, c := range revs.Each() {
			s := strings.Split(c.Message(), "\n")
			actual = append(actual, s[0])
		}
		expected := []string{"first", "first"}

		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("expected %v, got %v", expected, actual)
		}
	})
}

func TestCommitAmendingCommits(t *testing.T) {
	var setUp = func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		for _, message := range []string{"first", "second", "third"} {
			writeFile(t, tmpDir, "file.txt", message)
			Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
			options := CommitOption{
				ReadOption: write_commit.ReadOption{
					Message: message,
				},
			}
			commit(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), options, time.Now())
		}

		return
	}
	t.Run("replaces the last commit's message", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		options := CommitOption{
			IsTTY: true,
			Amend: true,
			Edit:  true,
			EditorCmd: func(path string) editor.Executable {
				return NewMockEditor(path, "third [amended]\n")
			},
		}
		commit(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), options, time.Now())

		revs := repository.NewRevList(repo(t, tmpDir), []string{"HEAD"}, repository.RevListOption{})

		actual := []string{}
		for _, c := range revs.Each() {
			s := strings.Split(c.Message(), "\n")
			actual = append(actual, s[0])
		}
		expected := []string{"third [amended]", "second", "first"}

		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("expected %v, got %v", expected, actual)
		}
	})

	t.Run("replaces the last commit's tree", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		writeFile(t, tmpDir, "another.txt", "1")
		Add(tmpDir, []string{"another.txt"}, new(bytes.Buffer), new(bytes.Buffer))
		options := CommitOption{
			IsTTY: true,
			Amend: true,
			Edit:  true,
			EditorCmd: func(path string) editor.Executable {
				return NewMockEditor(path, "third [amended]\n")
			},
		}
		commit(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), options, time.Now())

		obj, _ := loadCommit(t, tmpDir, "HEAD")
		commit := obj.(*database.Commit)
		diff := repo(t, tmpDir).Database.TreeDiff(commit.Parent(), commit.Oid(), nil)

		actual := []string{}
		for k := range diff {
			actual = append(actual, k)
		}
		sort.Slice(actual, func(i, j int) bool {
			return actual[i] < actual[j]
		})
		expected := []string{"another.txt", "file.txt"}

		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("expected %v, got %v", expected, actual)
		}
	})
}
