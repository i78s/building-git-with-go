package command

import (
	"bytes"
	"os"
	"reflect"
	"testing"
)

func branch(
	t *testing.T,
	tmpDir string,
	args []string,
	options BranchOption,
	stdout *bytes.Buffer,
	stderr *bytes.Buffer,
) *Branch {
	cmd, err := NewBranch(tmpDir, args, options, stdout, stderr)
	if err != nil {
		t.Fatal(err)
	}
	return cmd
}

func TestBranchWithChainOfCommits(t *testing.T) {
	setup := func() (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = SetupTestEnvironment(t)

		messages := []string{"first", "second", "third"}
		for _, msg := range messages {
			WriteFile(t, tmpDir, "file.txt", msg)
			Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
			TestCommit(t, tmpDir, msg)
		}

		return
	}

	t.Run("creates a branch pointing at HEAD", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		cmd := branch(t, tmpDir, []string{"topic"}, BranchOption{}, stdout, stderr)
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

		cmd := branch(t, tmpDir, []string{"^"}, BranchOption{}, stdout, stderr)
		cmd.Run()

		expected := `fatal: '^' is not a valid branch name.`

		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("fails for existing branch names", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)

		cmd := branch(t, tmpDir, []string{"topic"}, BranchOption{}, stdout, stderr)
		cmd.Run()
		cmd.Run()

		expected := `fatal: A branch named 'topic' already exists.`

		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})
}
