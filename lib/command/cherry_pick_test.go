package command

import (
	"building-git/lib/command/write_commit"
	"building-git/lib/database"
	"building-git/lib/editor"
	"building-git/lib/repository"
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

var now time.Time

func getTime() time.Time {
	if now.IsZero() {
		now = time.Now()
	}
	now = now.Add(time.Second * 10)
	return now
}

func setUpForTestCherryPickWithTwoBranches(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
	tmpDir, stdout, stderr = setupTestEnvironment(t)

	for _, message := range []string{"one", "two", "three", "four"} {
		commitTree(t, tmpDir, message, map[string]string{
			"f.txt": message,
		}, getTime())
	}
	brunchCmd, _ := NewBranch(tmpDir, []string{"topic", "@~2"}, BranchOption{}, new(bytes.Buffer), new(bytes.Buffer))
	brunchCmd.Run()
	checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "topic")

	commitTree(t, tmpDir, "five", map[string]string{
		"g.txt": "five",
	}, getTime())
	commitTree(t, tmpDir, "six", map[string]string{
		"f.txt": "six",
	}, getTime())
	commitTree(t, tmpDir, "seven", map[string]string{
		"g.txt": "seven",
	}, getTime())
	commitTree(t, tmpDir, "eight", map[string]string{
		"g.txt": "eight",
	}, getTime())

	checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "master")

	return
}

func TestCherryPickWithTwoBranches(t *testing.T) {
	t.Run("applies a commit on top of the current HEAD", func(t *testing.T) {
		tmpDir, stdout, stderr := setUpForTestCherryPickWithTwoBranches(t)
		defer os.RemoveAll(tmpDir)

		status := cherryPick(t, tmpDir, stdout, stderr, []string{"topic~3"}, CherryPickOption{})

		if status != 0 {
			t.Errorf("want %d, but got %d", 0, status)
		}

		revs, _ := repository.NewRevList(repo(t, tmpDir), []string{"@~3.."}, repository.RevListOption{})

		actual := []string{}
		for _, c := range revs.Each() {
			s := strings.Split(c.(*database.Commit).Message(), "\n")
			actual = append(actual, s[0])
		}
		expected := []string{"five", "four", "three"}

		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("expected %v, got %v", expected, actual)
		}

		assertIndexEntries(t, tmpDir, map[string]string{
			"f.txt": "four",
			"g.txt": "five",
		})

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "four",
			"g.txt": "five",
		})
	})

	t.Run("fails to apply a content conflict", func(t *testing.T) {
		tmpDir, stdout, stderr := setUpForTestCherryPickWithTwoBranches(t)
		defer os.RemoveAll(tmpDir)

		status := cherryPick(t, tmpDir, stdout, stderr, []string{"topic^^"}, CherryPickOption{})
		if status != 1 {
			t.Errorf("want %d, but got %d", 1, status)
		}

		rev, _ := resolveRevision(t, tmpDir, "topic^^")
		short := repo(t, tmpDir).Database.ShortOid(rev)
		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": fmt.Sprintf(`<<<<<<< HEAD
four=======
six>>>>>>> %s... six
`, short),
		})

		assertGitStatus(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), `UU f.txt
`)
	})

	t.Run("fails to apply a modify/delete conflict", func(t *testing.T) {
		tmpDir, stdout, stderr := setUpForTestCherryPickWithTwoBranches(t)
		defer os.RemoveAll(tmpDir)

		status := cherryPick(t, tmpDir, stdout, stderr, []string{"topic"}, CherryPickOption{})
		if status != 1 {
			t.Errorf("want %d, but got %d", 1, status)
		}

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "four",
			"g.txt": "eight",
		})

		assertGitStatus(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), `DU g.txt
`)
	})

	t.Run("continues a conflicted cherry-pick", func(t *testing.T) {
		tmpDir, stdout, stderr := setUpForTestCherryPickWithTwoBranches(t)
		defer os.RemoveAll(tmpDir)

		cherryPick(t, tmpDir, stdout, stderr, []string{"topic"}, CherryPickOption{})
		Add(tmpDir, []string{"g.txt"}, new(bytes.Buffer), new(bytes.Buffer))

		status := cherryPick(t, tmpDir, stdout, stderr, []string{}, CherryPickOption{Mode: Continue})
		if status != 0 {
			t.Errorf("want %d, but got %d", 0, status)
		}

		revs, _ := repository.NewRevList(repo(t, tmpDir), []string{"@~3.."}, repository.RevListOption{})
		commits := revs.Each()
		if !reflect.DeepEqual([]string{commits[1].Oid()}, commits[0].(*database.Commit).Parents) {
			t.Errorf("expected %v, got %v", []string{commits[1].Oid()}, commits[0].(*database.Commit).Parents)
		}
		messages := []string{}
		for _, c := range commits {
			messages = append(messages, strings.Split(c.(*database.Commit).Message(), "\n")[0])
		}
		if !reflect.DeepEqual([]string{"eight", "four", "three"}, messages) {
			t.Errorf("expected %v, got %v", []string{"eight", "four", "three"}, messages)
		}

		assertIndexEntries(t, tmpDir, map[string]string{
			"f.txt": "four",
			"g.txt": "eight",
		})

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "four",
			"g.txt": "eight",
		})
	})

	t.Run("commits after a conflicted cherry-pick", func(t *testing.T) {
		tmpDir, stdout, stderr := setUpForTestCherryPickWithTwoBranches(t)
		defer os.RemoveAll(tmpDir)

		cherryPick(t, tmpDir, stdout, stderr, []string{"topic"}, CherryPickOption{})
		Add(tmpDir, []string{"g.txt"}, new(bytes.Buffer), new(bytes.Buffer))
		status := commit(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), CommitOption{
			ReadOption: write_commit.ReadOption{Message: "commit"},
		}, getTime())

		if status != 0 {
			t.Errorf("want %d, but got %d", 0, status)
		}

		revs, _ := repository.NewRevList(repo(t, tmpDir), []string{"@~3.."}, repository.RevListOption{})
		commits := revs.Each()
		if !reflect.DeepEqual([]string{commits[1].Oid()}, commits[0].(*database.Commit).Parents) {
			t.Errorf("expected %v, got %v", []string{commits[1].Oid()}, commits[0].(*database.Commit).Parents)
		}
		messages := []string{}
		for _, c := range commits {
			messages = append(messages, strings.Split(c.(*database.Commit).Message(), "\n")[0])
		}
		if !reflect.DeepEqual([]string{"eight", "four", "three"}, messages) {
			t.Errorf("expected %v, got %v", []string{"eight", "four", "three"}, messages)
		}
	})

	t.Run("applies multiple non-conflicting commits", func(t *testing.T) {
		tmpDir, stdout, stderr := setUpForTestCherryPickWithTwoBranches(t)
		defer os.RemoveAll(tmpDir)

		status := cherryPick(t, tmpDir, stdout, stderr, []string{"topic~3", "topic^", "topic"}, CherryPickOption{})
		if status != 0 {
			t.Errorf("want %d, but got %d", 0, status)
		}

		revs, _ := repository.NewRevList(repo(t, tmpDir), []string{"@~4.."}, repository.RevListOption{})
		commits := revs.Each()
		messages := []string{}
		for _, c := range commits {
			messages = append(messages, strings.Split(c.(*database.Commit).Message(), "\n")[0])
		}
		if !reflect.DeepEqual([]string{"eight", "seven", "five", "four"}, messages) {
			t.Errorf("expected %v, got %v", []string{"eight", "seven", "five", "four"}, messages)
		}

		assertIndexEntries(t, tmpDir, map[string]string{
			"f.txt": "four",
			"g.txt": "eight",
		})

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "four",
			"g.txt": "eight",
		})
	})

	t.Run("stops when a list of commits includes a conflict", func(t *testing.T) {
		tmpDir, stdout, stderr := setUpForTestCherryPickWithTwoBranches(t)
		defer os.RemoveAll(tmpDir)

		status := cherryPick(t, tmpDir, stdout, stderr, []string{"topic^", "topic~3"}, CherryPickOption{})
		if status != 1 {
			t.Errorf("want %d, but got %d", 1, status)
		}

		assertGitStatus(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), "DU g.txt\n")
	})

	t.Run("stops when a range of commits includes a conflict", func(t *testing.T) {
		tmpDir, stdout, stderr := setUpForTestCherryPickWithTwoBranches(t)
		defer os.RemoveAll(tmpDir)

		status := cherryPick(t, tmpDir, stdout, stderr, []string{"..topic"}, CherryPickOption{})
		if status != 1 {
			t.Errorf("want %d, but got %d", 1, status)
		}

		assertGitStatus(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), "UU f.txt\n")
	})

	t.Run("refuses to commit in a conflicted state", func(t *testing.T) {
		tmpDir, stdout, stderr := setUpForTestCherryPickWithTwoBranches(t)
		defer os.RemoveAll(tmpDir)

		cherryPick(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), []string{"..topic"}, CherryPickOption{})
		status := commit(t, tmpDir, stdout, stderr, CommitOption{}, getTime())
		if status != 128 {
			t.Errorf("want %d, but got %d", 128, status)
		}

		expected := `error: Committing is not possible because you have unmerged files.
hint: Fix them up in the work tree, and then use 'jit add/rm <file>'
hint: as appropriate to mark resolution and make a commit.
fatal: Exiting because of an unresolved conflict.`

		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("refuses to continue in a conflicted state", func(t *testing.T) {
		tmpDir, stdout, stderr := setUpForTestCherryPickWithTwoBranches(t)
		defer os.RemoveAll(tmpDir)

		cherryPick(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), []string{"..topic"}, CherryPickOption{})
		status := cherryPick(t, tmpDir, stdout, stderr, []string{}, CherryPickOption{Mode: Continue})
		if status != 128 {
			t.Errorf("want %d, but got %d", 128, status)
		}

		expected := `error: Committing is not possible because you have unmerged files.
hint: Fix them up in the work tree, and then use 'jit add/rm <file>'
hint: as appropriate to mark resolution and make a commit.
fatal: Exiting because of an unresolved conflict.`

		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("can continue after resolving the conflicts", func(t *testing.T) {
		tmpDir, stdout, stderr := setUpForTestCherryPickWithTwoBranches(t)
		defer os.RemoveAll(tmpDir)

		cherryPick(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), []string{"..topic"}, CherryPickOption{})
		writeFile(t, tmpDir, "f.txt", "six")
		Add(tmpDir, []string{"f.txt"}, new(bytes.Buffer), new(bytes.Buffer))

		status := cherryPick(t, tmpDir, stdout, stderr, []string{}, CherryPickOption{Mode: Continue})
		if status != 0 {
			t.Errorf("want %d, but got %d", 0, status)
		}

		revs, _ := repository.NewRevList(repo(t, tmpDir), []string{"@~5.."}, repository.RevListOption{})
		commits := revs.Each()
		if !reflect.DeepEqual([]string{commits[1].Oid()}, commits[0].(*database.Commit).Parents) {
			t.Errorf("expected %v, got %v", []string{commits[1].Oid()}, commits[0].(*database.Commit).Parents)
		}
		messages := []string{}
		for _, c := range commits {
			messages = append(messages, strings.Split(c.(*database.Commit).Message(), "\n")[0])
		}
		if !reflect.DeepEqual([]string{"eight", "seven", "six", "five", "four"}, messages) {
			t.Errorf("expected %v, got %v", []string{"eight", "seven", "six", "five", "four"}, messages)
		}

		assertIndexEntries(t, tmpDir, map[string]string{
			"f.txt": "six",
			"g.txt": "eight",
		})

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "six",
			"g.txt": "eight",
		})
	})

	t.Run("can continue after commiting the resolved tree", func(t *testing.T) {
		tmpDir, stdout, stderr := setUpForTestCherryPickWithTwoBranches(t)
		defer os.RemoveAll(tmpDir)

		cherryPick(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), []string{"..topic"}, CherryPickOption{})
		writeFile(t, tmpDir, "f.txt", "six")
		Add(tmpDir, []string{"f.txt"}, new(bytes.Buffer), new(bytes.Buffer))
		commit(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), CommitOption{}, getTime())

		status := cherryPick(t, tmpDir, stdout, stderr, []string{}, CherryPickOption{Mode: Continue})
		if status != 0 {
			t.Errorf("want %d, but got %d", 0, status)
		}

		revs, _ := repository.NewRevList(repo(t, tmpDir), []string{"@~5.."}, repository.RevListOption{})
		commits := revs.Each()
		if !reflect.DeepEqual([]string{commits[1].Oid()}, commits[0].(*database.Commit).Parents) {
			t.Errorf("expected %v, got %v", []string{commits[1].Oid()}, commits[0].(*database.Commit).Parents)
		}
		messages := []string{}
		for _, c := range commits {
			messages = append(messages, strings.Split(c.(*database.Commit).Message(), "\n")[0])
		}
		if !reflect.DeepEqual([]string{"eight", "seven", "six", "five", "four"}, messages) {
			t.Errorf("expected %v, got %v", []string{"eight", "seven", "six", "five", "four"}, messages)
		}

		assertIndexEntries(t, tmpDir, map[string]string{
			"f.txt": "six",
			"g.txt": "eight",
		})

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "six",
			"g.txt": "eight",
		})
	})
}

func TestCherryPickWithTwoBranchesAbortingInConflictedState(t *testing.T) {
	var before = func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer, status int) {
		tmpDir, stdout, stderr = setUpForTestCherryPickWithTwoBranches(t)

		cherryPick(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), []string{"..topic"}, CherryPickOption{})
		cherryPick(t, tmpDir, stdout, stderr, []string{}, CherryPickOption{Mode: Abort})

		return
	}
	t.Run("exits successfully", func(t *testing.T) {
		tmpDir, _, stderr, status := before(t)
		defer os.RemoveAll(tmpDir)

		if status != 0 {
			t.Errorf("want %d, but got %d", 0, status)
		}
		expected := ""
		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("resets to the old HEAD", func(t *testing.T) {
		tmpDir, _, _, _ := before(t)
		defer os.RemoveAll(tmpDir)

		obj, _ := loadCommit(t, tmpDir, "HEAD")

		expected := "four"
		if got := strings.TrimSpace(obj.(*database.Commit).Message()); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}

		assertGitStatus(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), "")
	})

	t.Run("removes the merge state", func(t *testing.T) {
		tmpDir, _, _, _ := before(t)
		defer os.RemoveAll(tmpDir)

		inProgress := repo(t, tmpDir).PendingCommit.InProgress()
		if inProgress {
			t.Errorf("want %v, but got %v", false, inProgress)
		}
	})
}

func TestCherryPickWithTwoBranchesAbortingInCommittedState(t *testing.T) {
	var before = func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer, status int) {
		tmpDir, stdout, stderr = setUpForTestCherryPickWithTwoBranches(t)

		cherryPick(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), []string{"..topic"}, CherryPickOption{})
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
		options := CommitOption{
			IsTTY: true,
			Edit:  true,
			EditorCmd: func(path string) editor.Executable {
				return NewMockEditor(path, "picked\n")
			},
		}
		commit(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), options, time.Now())

		cherryPick(t, tmpDir, stdout, stderr, []string{}, CherryPickOption{Mode: Abort})

		return
	}
	t.Run("exits with a warning", func(t *testing.T) {
		tmpDir, _, stderr, status := before(t)
		defer os.RemoveAll(tmpDir)

		if status != 0 {
			t.Errorf("want %d, but got %d", 0, status)
		}
		expected := "warning: You seem to have moved HEAD. Not rewinding, check your HEAD!\n"
		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("does not reset HEAD", func(t *testing.T) {
		tmpDir, _, _, _ := before(t)
		defer os.RemoveAll(tmpDir)

		obj, _ := loadCommit(t, tmpDir, "HEAD")

		expected := "picked"
		if got := strings.TrimSpace(obj.(*database.Commit).Message()); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}

		assertGitStatus(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), "")
	})

	t.Run("removes the merge state", func(t *testing.T) {
		tmpDir, _, _, _ := before(t)
		defer os.RemoveAll(tmpDir)

		inProgress := repo(t, tmpDir).PendingCommit.InProgress()
		if inProgress {
			t.Errorf("want %v, but got %v", false, inProgress)
		}
	})
}
