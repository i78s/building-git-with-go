package command

import (
	"building-git/lib/database"
	"bytes"
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestBranchWithChainOfCommits(t *testing.T) {
	setup := func() (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		messages := []string{"first", "second", "third"}
		for _, msg := range messages {
			writeFile(t, tmpDir, "file.txt", msg)
			Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
			commit(t, tmpDir, msg)
		}

		return
	}

	t.Run("creates a branch pointing at HEAD", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewBranch(tmpDir, []string{"topic"}, BranchOption{}, stdout, stderr)
		cmd.Run()

		expected, _ := cmd.repo.Refs.ReadHead()
		result, _ := cmd.repo.Refs.ReadRef("topic")

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("want %q, but got %q", expected, result)
		}
	})

	t.Run("fails for invalid branch names", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewBranch(tmpDir, []string{"^"}, BranchOption{}, stdout, stderr)
		cmd.Run()

		expected := `fatal: '^' is not a valid branch name.`

		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("fails for existing branch names", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewBranch(tmpDir, []string{"topic"}, BranchOption{}, stdout, stderr)
		cmd.Run()
		cmd.Run() // call twice

		expected := `fatal: A branch named 'topic' already exists.`

		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("creates a branch pointing at HEAD's parent", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewBranch(tmpDir, []string{"topic", "HEAD^"}, BranchOption{}, stdout, stderr)
		cmd.Run()

		oid, _ := cmd.repo.Refs.ReadHead()
		headObj, _ := cmd.repo.Database.Load(oid)
		head, _ := headObj.(*database.Commit)
		result := head.Parent()

		expected, _ := cmd.repo.Refs.ReadRef("topic")

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("want %q, but got %q", expected, result)
		}
	})

	t.Run("creates a branch pointing at HEAD's grandparent", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewBranch(tmpDir, []string{"topic", "@~2"}, BranchOption{}, stdout, stderr)
		cmd.Run()

		oid, _ := cmd.repo.Refs.ReadHead()
		headObj, _ := cmd.repo.Database.Load(oid)
		head, _ := headObj.(*database.Commit)
		parentObj, _ := cmd.repo.Database.Load(head.Parent())
		parent, _ := parentObj.(*database.Commit)
		result := parent.Parent()

		expected, _ := cmd.repo.Refs.ReadRef("topic")

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("want %q, but got %q", expected, result)
		}
	})

	t.Run("creates a branch relative to another one", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewBranch(tmpDir, []string{"topic", "@~1"}, BranchOption{}, stdout, stderr)
		cmd.Run()
		cmd, _ = NewBranch(tmpDir, []string{"another", "topic^"}, BranchOption{}, stdout, stderr)
		cmd.Run()

		result, _ := resolveRevision(t, tmpDir, "HEAD~2")
		expected, _ := cmd.repo.Refs.ReadRef("another")

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("want %q, but got %q", expected, result)
		}
	})

	t.Run("creates a branch from a short commit ID", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		commitId, _ := resolveRevision(t, tmpDir, "@~2")
		r := repo(t, tmpDir)

		a := r.Database.ShortOid(commitId)
		cmd, _ := NewBranch(tmpDir, []string{"topic", a}, BranchOption{}, stdout, stderr)
		cmd.Run()

		expected, _ := r.Refs.ReadRef("topic")

		if !reflect.DeepEqual(commitId, expected) {
			t.Errorf("want %q, but got %q", expected, commitId)
		}
	})

	t.Run("fails for invalid revisions", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewBranch(tmpDir, []string{"topic", "^"}, BranchOption{}, stdout, stderr)
		cmd.Run()

		expected := `fatal: Not a valid object name: '^'.`

		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("fails for invalid refs", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewBranch(tmpDir, []string{"topic", "no-such-branch"}, BranchOption{}, stdout, stderr)
		cmd.Run()

		expected := `fatal: Not a valid object name: 'no-such-branch'.`

		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("fails for invalid parents", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewBranch(tmpDir, []string{"topic", "@^^^^"}, BranchOption{}, stdout, stderr)
		cmd.Run()

		expected := `fatal: Not a valid object name: '@^^^^'.`

		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("fails for invalid ancestors", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewBranch(tmpDir, []string{"topic", "@~50"}, BranchOption{}, stdout, stderr)
		cmd.Run()

		expected := `fatal: Not a valid object name: '@~50'.`

		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("fails for revisions that are not commits", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		r := repo(t, tmpDir)
		head, _ := r.Refs.ReadHead()
		treeObj, _ := r.Database.Load(head)
		treeId := treeObj.(*database.Commit).Tree()
		cmd, _ := NewBranch(tmpDir, []string{"topic", treeId}, BranchOption{}, stdout, stderr)
		cmd.Run()

		expected := fmt.Sprintf("error: object %s is a tree, not a commit\nfatal: Not a valid object name: '%s'.", treeId, treeId)

		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("fails for parents of revisions that are not commits", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		r := repo(t, tmpDir)
		head, _ := r.Refs.ReadHead()
		treeObj, _ := r.Database.Load(head)
		treeId := treeObj.(*database.Commit).Tree()
		cmd, _ := NewBranch(tmpDir, []string{"topic", treeId + "^^"}, BranchOption{}, stdout, stderr)
		cmd.Run()

		expected := fmt.Sprintf("error: object %s is a tree, not a commit\nfatal: Not a valid object name: '%s'.", treeId, treeId+"^^")

		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})
}
