package command

import (
	"building-git/lib/repository"
	"bytes"
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
			t.Errorf("want %q, but got %q", 0, status)
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
}
