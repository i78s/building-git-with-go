package command

import (
	"building-git/lib/command/write_commit"
	"building-git/lib/repository"
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCherryPickWithTwoBranches(t *testing.T) {
	var setUp = func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		for _, message := range []string{"one", "two", "three", "four"} {
			commitTree(t, tmpDir, message, map[string]string{
				"f.txt": message,
			}, time.Now())
		}
		brunchCmd, _ := NewBranch(tmpDir, []string{"topic", "@~2"}, BranchOption{}, new(bytes.Buffer), new(bytes.Buffer))
		brunchCmd.Run()
		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "topic")

		commitTree(t, tmpDir, "five", map[string]string{
			"g.txt": "five",
		}, time.Now())
		commitTree(t, tmpDir, "six", map[string]string{
			"f.txt": "six",
		}, time.Now())
		commitTree(t, tmpDir, "seven", map[string]string{
			"g.txt": "seven",
		}, time.Now())
		commitTree(t, tmpDir, "eight", map[string]string{
			"g.txt": "eight",
		}, time.Now())

		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "master")

		return
	}
	t.Run("applies a commit on top of the current HEAD", func(t *testing.T) {
		tmpDir, stdout, stderr := setUp(t)
		defer os.RemoveAll(tmpDir)

		status := cherryPick(t, tmpDir, stdout, stderr, []string{"topic~3"}, CherryPickOption{})

		if status != 0 {
			t.Errorf("want %d, but got %d", 0, status)
		}

		revs := repository.NewRevList(repo(t, tmpDir), []string{"@~3.."})

		actual := []string{}
		for _, c := range revs.Each() {
			s := strings.Split(c.Message(), "\n")
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
		tmpDir, stdout, stderr := setUp(t)
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
		tmpDir, stdout, stderr := setUp(t)
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
		tmpDir, stdout, stderr := setUp(t)
		defer os.RemoveAll(tmpDir)

		cherryPick(t, tmpDir, stdout, stderr, []string{"topic"}, CherryPickOption{})
		Add(tmpDir, []string{"g.txt"}, new(bytes.Buffer), new(bytes.Buffer))

		status := cherryPick(t, tmpDir, stdout, stderr, []string{}, CherryPickOption{Mode: Continue})
		if status != 0 {
			t.Errorf("want %d, but got %d", 0, status)
		}

		commits := repository.NewRevList(repo(t, tmpDir), []string{"@~3.."}).Each()
		if !reflect.DeepEqual([]string{commits[1].Oid()}, commits[0].Parents) {
			t.Errorf("expected %v, got %v", []string{commits[1].Oid()}, commits[0].Parents)
		}
		messages := []string{}
		for _, c := range commits {
			messages = append(messages, strings.Split(c.Message(), "\n")[0])
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
		tmpDir, stdout, stderr := setUp(t)
		defer os.RemoveAll(tmpDir)

		cherryPick(t, tmpDir, stdout, stderr, []string{"topic"}, CherryPickOption{})
		Add(tmpDir, []string{"g.txt"}, new(bytes.Buffer), new(bytes.Buffer))
		status := commit(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), CommitOption{
			ReadOption: write_commit.ReadOption{Message: "commit"},
		}, time.Now())

		if status != 0 {
			t.Errorf("want %d, but got %d", 0, status)
		}

		commits := repository.NewRevList(repo(t, tmpDir), []string{"@~3.."}).Each()
		if !reflect.DeepEqual([]string{commits[1].Oid()}, commits[0].Parents) {
			t.Errorf("expected %v, got %v", []string{commits[1].Oid()}, commits[0].Parents)
		}
		messages := []string{}
		for _, c := range commits {
			messages = append(messages, strings.Split(c.Message(), "\n")[0])
		}
		if !reflect.DeepEqual([]string{"eight", "four", "three"}, messages) {
			t.Errorf("expected %v, got %v", []string{"eight", "four", "three"}, messages)
		}
	})
}
