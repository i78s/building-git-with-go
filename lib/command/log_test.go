package command

import (
	"building-git/lib/database"
	"bytes"
	"fmt"
	"os"
	"testing"
)

func setUpForTestLogWithChainOfCommits(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer, commits []*database.Commit) {
	tmpDir, stdout, stderr = setupTestEnvironment(t)

	messages := []string{"A", "B", "C"}
	for _, message := range messages {
		commitFile(t, tmpDir, message)
	}

	commits = []*database.Commit{}
	for _, rev := range []string{"@", "@^", "@^^"} {
		cobj, _ := loadCommit(t, tmpDir, rev)
		c, _ := cobj.(*database.Commit)
		commits = append(commits, c)
	}

	return
}

func commitFile(t *testing.T, tmpDir, message string) {
	writeFile(t, tmpDir, "file.txt", message)
	Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
	commit(t, tmpDir, message)
}

func TestLogWithChainOfCommits(t *testing.T) {
	t.Run("advances a branch pointer", func(t *testing.T) {
		tmpDir, stdout, stderr, commits := setUpForTestLogWithChainOfCommits(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{}, LogOption{}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`commit %s
Author: A. U. Thor <author@example.com>
Date:  %s

    C

commit %s
Author: A. U. Thor <author@example.com>
Date:  %s

    B

commit %s
Author: A. U. Thor <author@example.com>
Date:  %s

    A
`, commits[0].Oid(), commits[0].Author().ReadableTime(),
			commits[1].Oid(), commits[1].Author().ReadableTime(),
			commits[2].Oid(), commits[2].Author().ReadableTime())
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("prints a log in medium format with abbreviated commit IDs", func(t *testing.T) {
		tmpDir, stdout, stderr, commits := setUpForTestLogWithChainOfCommits(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{}, LogOption{Abbrev: true}, stdout, stderr)
		log.Run()

		r := repo(t, tmpDir)

		expected := fmt.Sprintf(`commit %s
Author: A. U. Thor <author@example.com>
Date:  %s

    C

commit %s
Author: A. U. Thor <author@example.com>
Date:  %s

    B

commit %s
Author: A. U. Thor <author@example.com>
Date:  %s

    A
`, r.Database.ShortOid(commits[0].Oid()), commits[0].Author().ReadableTime(),
			r.Database.ShortOid(commits[1].Oid()), commits[1].Author().ReadableTime(),
			r.Database.ShortOid(commits[2].Oid()), commits[2].Author().ReadableTime())
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("prints a log in oneline format", func(t *testing.T) {
		tmpDir, stdout, stderr, commits := setUpForTestLogWithChainOfCommits(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{}, LogOption{Abbrev: true, Format: "oneline"}, stdout, stderr)
		log.Run()

		r := repo(t, tmpDir)

		expected := fmt.Sprintf(`commit %s C
commit %s B
commit %s A
`, r.Database.ShortOid(commits[0].Oid()),
			r.Database.ShortOid(commits[1].Oid()),
			r.Database.ShortOid(commits[2].Oid()))
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("prints a log in oneline format without abbreviated commit IDs", func(t *testing.T) {
		tmpDir, stdout, stderr, commits := setUpForTestLogWithChainOfCommits(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{}, LogOption{Format: "oneline"}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`commit %s C
commit %s B
commit %s A
`, commits[0].Oid(),
			commits[1].Oid(),
			commits[2].Oid())
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})
}