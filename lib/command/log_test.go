package command

import (
	"building-git/lib/database"
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"
)

func setUpForTestLogWithChainOfCommits(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer, commits []*database.Commit) {
	tmpDir, stdout, stderr = setupTestEnvironment(t)

	messages := []string{"A", "B", "C"}
	for _, message := range messages {
		commitFile(t, tmpDir, message, time.Now())
	}

	brunchCmd, _ := NewBranch(tmpDir, []string{"topic", "@^^"}, BranchOption{}, new(bytes.Buffer), new(bytes.Buffer))
	brunchCmd.Run()

	commits = []*database.Commit{}
	for _, rev := range []string{"@", "@^", "@^^"} {
		cobj, _ := loadCommit(t, tmpDir, rev)
		c, _ := cobj.(*database.Commit)
		commits = append(commits, c)
	}

	return
}

func commitFile(t *testing.T, tmpDir, message string, now time.Time) {
	writeFile(t, tmpDir, "file.txt", message)
	Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
	commit(t, tmpDir, message, now)
}

func TestLogWithChainOfCommits(t *testing.T) {
	t.Run("advances a branch pointer", func(t *testing.T) {
		tmpDir, stdout, stderr, commits := setUpForTestLogWithChainOfCommits(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{}, LogOption{IsTty: false, Decorate: "auto"}, stdout, stderr)
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

		log, _ := NewLog(tmpDir, []string{}, LogOption{Abbrev: true, IsTty: false, Decorate: "auto"}, stdout, stderr)
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

		log, _ := NewLog(tmpDir, []string{}, LogOption{Abbrev: true, Format: "oneline", IsTty: false, Decorate: "auto"}, stdout, stderr)
		log.Run()

		r := repo(t, tmpDir)

		expected := fmt.Sprintf(`%s C
%s B
%s A
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

		log, _ := NewLog(tmpDir, []string{}, LogOption{Format: "oneline", IsTty: false, Decorate: "auto"}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`%s C
%s B
%s A
`, commits[0].Oid(),
			commits[1].Oid(),
			commits[2].Oid())
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("prints a log starting from a specified commit", func(t *testing.T) {
		tmpDir, stdout, stderr, commits := setUpForTestLogWithChainOfCommits(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{"@^"}, LogOption{Format: "oneline", IsTty: false, Decorate: "auto"}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`%s B
%s A
`, commits[1].Oid(),
			commits[2].Oid())
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("prints a log with short decorations", func(t *testing.T) {
		tmpDir, stdout, stderr, commits := setUpForTestLogWithChainOfCommits(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{}, LogOption{Format: "oneline", IsTty: false, Decorate: "short"}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`%s (HEAD -> master) C
%s B
%s (topic) A
`, commits[0].Oid(),
			commits[1].Oid(),
			commits[2].Oid())
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("prints a log with detached HEAD", func(t *testing.T) {
		tmpDir, stdout, stderr, commits := setUpForTestLogWithChainOfCommits(t)
		defer os.RemoveAll(tmpDir)

		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "@")
		log, _ := NewLog(tmpDir, []string{}, LogOption{Format: "oneline", IsTty: false, Decorate: "short"}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`%s (HEAD, master) C
%s B
%s (topic) A
`, commits[0].Oid(),
			commits[1].Oid(),
			commits[2].Oid())
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("prints a log with full decorations", func(t *testing.T) {
		tmpDir, stdout, stderr, commits := setUpForTestLogWithChainOfCommits(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{}, LogOption{Format: "oneline", IsTty: false, Decorate: "full"}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`%s (HEAD -> refs/heads/master) C
%s B
%s (refs/heads/topic) A
`, commits[0].Oid(),
			commits[1].Oid(),
			commits[2].Oid())
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("prints a log with patches", func(t *testing.T) {
		tmpDir, stdout, stderr, commits := setUpForTestLogWithChainOfCommits(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{}, LogOption{Format: "oneline", IsTty: false, Patch: true, Decorate: "auto"}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`%s C
diff --git a/file.txt b/file.txt
index 7371f47..96d80cd 100644
--- a/file.txt
+++ b/file.txt
@@ -1,1 +1,1 @@
-B
+C
%s B
diff --git a/file.txt b/file.txt
index 8c7e5a6..7371f47 100644
--- a/file.txt
+++ b/file.txt
@@ -1,1 +1,1 @@
-A
+B
%s A
diff --git a/file.txt b/file.txt
new file mode 100644
index 0000000..8c7e5a6
--- /dev/null
+++ b/file.txt
@@ -0,0 +1,1 @@
+A
`, commits[0].Oid(),
			commits[1].Oid(),
			commits[2].Oid())
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})
}

func TestLogWithTreeOfCommits(t *testing.T) {
	//  m1  m2  m3
	//   o---o---o [master]
	//        \
	//         o---o---o---o [topic]
	//        t1  t2  t3  t4

	var setUp = func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer, master, topic []string) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		for i := 1; i <= 3; i++ {
			commitFile(t, tmpDir, fmt.Sprintf("master-%d", i), time.Now())
		}

		brunchCmd, _ := NewBranch(tmpDir, []string{"topic", "master^"}, BranchOption{}, new(bytes.Buffer), new(bytes.Buffer))
		brunchCmd.Run()
		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "topic")

		branchTime := time.Now().Add(time.Second * 10)
		for i := 1; i <= 4; i++ {
			commitFile(t, tmpDir, fmt.Sprintf("topic-%d", i), branchTime)
		}

		master = []string{}
		for i := 0; i <= 2; i++ {
			rev, _ := resolveRevision(t, tmpDir, fmt.Sprintf("master~%d", i))
			master = append(master, rev)
		}
		topic = []string{}
		for i := 0; i <= 3; i++ {
			rev, _ := resolveRevision(t, tmpDir, fmt.Sprintf("topic~%d", i))
			topic = append(topic, rev)
		}

		return
	}

	t.Run("logs the combined history of multiple branches", func(t *testing.T) {
		tmpDir, stdout, stderr, master, topic := setUp(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{"master", "topic"}, LogOption{Format: "oneline", IsTty: false, Decorate: "short"}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`%s (HEAD -> topic) topic-4
%s topic-3
%s topic-2
%s topic-1
%s (master) master-3
%s master-2
%s master-1
`, topic[0],
			topic[1],
			topic[2],
			topic[3],
			master[0],
			master[1],
			master[2],
		)
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})
}
