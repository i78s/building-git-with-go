package command

import (
	"building-git/lib/command/write_commit"
	"building-git/lib/database"
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"
)

func commitFile(t *testing.T, tmpDir, message string, now time.Time) {
	writeFile(t, tmpDir, "file.txt", message)
	Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
	commit(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), message, now)
}

func TestLogWithChainOfCommits(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer, commits []*database.Commit) {
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
	t.Run("advances a branch pointer", func(t *testing.T) {
		tmpDir, stdout, stderr, commits := setUp(t)
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
		tmpDir, stdout, stderr, commits := setUp(t)
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
		tmpDir, stdout, stderr, commits := setUp(t)
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
		tmpDir, stdout, stderr, commits := setUp(t)
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
		tmpDir, stdout, stderr, commits := setUp(t)
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
		tmpDir, stdout, stderr, commits := setUp(t)
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
		tmpDir, stdout, stderr, commits := setUp(t)
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
		tmpDir, stdout, stderr, commits := setUp(t)
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
		tmpDir, stdout, stderr, commits := setUp(t)
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

func TestLogWithCommitsChangingDifferentFiles(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer, commits []*database.Commit) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		commitTree(t, tmpDir, "first", map[string]string{
			"a/1.txt":   "1",
			"b/c/2.txt": "2",
		}, time.Now())

		commitTree(t, tmpDir, "second", map[string]string{
			"a/1.txt": "10",
			"b/3.txt": "3",
		}, time.Now())

		commitTree(t, tmpDir, "third", map[string]string{
			"b/c/2.txt": "4",
		}, time.Now())

		commits = []*database.Commit{}
		for _, rev := range []string{"@^^", "@^", "@"} {
			cobj, _ := loadCommit(t, tmpDir, rev)
			c, _ := cobj.(*database.Commit)
			commits = append(commits, c)
		}

		return
	}

	t.Run("logs commits that change a file", func(t *testing.T) {
		tmpDir, stdout, stderr, commits := setUp(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{"a/1.txt"}, LogOption{Format: "oneline", IsTty: false, Decorate: "auto"}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`%s second
%s first
`, commits[1].Oid(),
			commits[0].Oid(),
		)
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("logs commits that change a directory", func(t *testing.T) {
		tmpDir, stdout, stderr, commits := setUp(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{"b"}, LogOption{Format: "oneline", IsTty: false, Decorate: "auto"}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`%s third
%s second
%s first
`, commits[2].Oid(),
			commits[1].Oid(),
			commits[0].Oid(),
		)
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("logs commits that change a directory and one of its files", func(t *testing.T) {
		tmpDir, stdout, stderr, commits := setUp(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{"b", "b/3.txt"}, LogOption{Format: "oneline", IsTty: false, Decorate: "auto"}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`%s third
%s second
%s first
`, commits[2].Oid(),
			commits[1].Oid(),
			commits[0].Oid(),
		)
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("logs commits that change a nested directory", func(t *testing.T) {
		tmpDir, stdout, stderr, commits := setUp(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{"b/c"}, LogOption{Format: "oneline", IsTty: false, Decorate: "auto"}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`%s third
%s first
`, commits[2].Oid(),
			commits[0].Oid(),
		)
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("logs commits with patches for selected files", func(t *testing.T) {
		tmpDir, stdout, stderr, commits := setUp(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{"a/1.txt"}, LogOption{Format: "oneline", IsTty: false, Decorate: "auto", Patch: true}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`%s second
diff --git a/a/1.txt b/a/1.txt
index 56a6051..9a03714 100644
--- a/a/1.txt
+++ b/a/1.txt
@@ -1,1 +1,1 @@
-1
+10
%s first
diff --git a/a/1.txt b/a/1.txt
new file mode 100644
index 0000000..56a6051
--- /dev/null
+++ b/a/1.txt
@@ -0,0 +1,1 @@
+1
`, commits[1].Oid(),
			commits[0].Oid(),
		)
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

	var branchTime = time.Now().Add(time.Second * 10)
	var setUp = func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer, master, topic []string) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		for i := 1; i <= 3; i++ {
			commitFile(t, tmpDir, fmt.Sprintf("master-%d", i), time.Now())
		}

		brunchCmd, _ := NewBranch(tmpDir, []string{"topic", "master^"}, BranchOption{}, new(bytes.Buffer), new(bytes.Buffer))
		brunchCmd.Run()
		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "topic")

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

	t.Run("logs the difference from one one branch to another", func(t *testing.T) {
		tmpDir, stdout, stderr, _, topic := setUp(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{"master..topic"}, LogOption{Format: "oneline", IsTty: false, Decorate: "auto"}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`%s topic-4
%s topic-3
%s topic-2
%s topic-1
`, topic[0],
			topic[1],
			topic[2],
			topic[3],
		)
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("excludes a long branch when commit times are equal", func(t *testing.T) {
		tmpDir, stdout, stderr, _, topic := setUp(t)
		defer os.RemoveAll(tmpDir)

		brunchCmd, _ := NewBranch(tmpDir, []string{"side", "topic^^"}, BranchOption{}, new(bytes.Buffer), new(bytes.Buffer))
		brunchCmd.Run()
		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "side")

		for i := 1; i <= 10; i++ {
			commitFile(t, tmpDir, fmt.Sprintf("side-%d", i), branchTime)
		}

		log, _ := NewLog(tmpDir, []string{"side..topic", "^master"}, LogOption{Format: "oneline", IsTty: false, Decorate: "auto"}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`%s topic-4
%s topic-3
`, topic[0],
			topic[1],
		)
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("logs the last few commits on a branch", func(t *testing.T) {
		tmpDir, stdout, stderr, _, topic := setUp(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{"@~3.."}, LogOption{Format: "oneline", IsTty: false, Decorate: "auto"}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`%s topic-4
%s topic-3
%s topic-2
`, topic[0],
			topic[1],
			topic[2],
		)
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})
}

func TestLogWithGraphOfCommits(t *testing.T) {
	//   A   B   C   D   J   K
	//   o---o---o---o---o---o [master]
	//        \         /
	//         o---o---o---o [topic]
	//         E   F   G   H

	var branchTime = time.Now()
	var setUp = func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer, master, topic []string) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		commitTree(t, tmpDir, "A", map[string]string{"f.txt": "0", "g.txt": "0"}, branchTime)
		commitTree(t, tmpDir, "B", map[string]string{
			"f.txt": "B",
			"h.txt": `one
two
three
`,
		}, branchTime)

		for c := 'C'; c <= 'D'; c++ {
			commitTree(t, tmpDir, string(c), map[string]string{
				"f.txt": string(c),
				"h.txt": fmt.Sprintf(`%s
two
three
`, string(c)),
			}, branchTime.Add(time.Second))
		}

		brunchCmd, _ := NewBranch(tmpDir, []string{"topic", "master~2"}, BranchOption{}, new(bytes.Buffer), new(bytes.Buffer))
		brunchCmd.Run()
		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "topic")

		for c := 'E'; c <= 'H'; c++ {
			commitTree(t, tmpDir, string(c), map[string]string{
				"g.txt": string(c),
				"h.txt": fmt.Sprintf(`one
two
%s
`, string(c)),
			}, branchTime.Add(2*time.Second))
		}

		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "master")
		mergeCommit(t, tmpDir, "topic^", MergeOption{ReadOption: write_commit.ReadOption{Message: "J"}}, new(bytes.Buffer), new(bytes.Buffer))

		commitTree(t, tmpDir, "K", map[string]string{"f.txt": "K"}, branchTime.Add(3*time.Second))

		master = []string{}
		for i := 0; i <= 5; i++ {
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

	t.Run("logs concurrent branches leading to a merge", func(t *testing.T) {
		tmpDir, stdout, stderr, master, topic := setUp(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{}, LogOption{Format: "oneline", IsTty: false, Decorate: "auto"}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`%s K
%s J
%s G
%s F
%s E
%s D
%s C
%s B
%s A
`, master[0],
			master[1],
			topic[1],
			topic[2],
			topic[3],
			master[2],
			master[3],
			master[4],
			master[5],
		)
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("logs the first parent of a merge", func(t *testing.T) {
		tmpDir, stdout, stderr, master, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{"master^^"}, LogOption{Format: "oneline", IsTty: false, Decorate: "auto"}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`%s D
%s C
%s B
%s A
`, master[2],
			master[3],
			master[4],
			master[5],
		)
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("logs the second parent of a merge", func(t *testing.T) {
		tmpDir, stdout, stderr, master, topic := setUp(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{"master^^2"}, LogOption{Format: "oneline", IsTty: false, Decorate: "auto"}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`%s G
%s F
%s E
%s B
%s A
`, topic[1],
			topic[2],
			topic[3],
			master[4],
			master[5],
		)
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("logs unmerged commits on a branch", func(t *testing.T) {
		tmpDir, stdout, stderr, _, topic := setUp(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{"master..topic"}, LogOption{Format: "oneline", IsTty: false, Decorate: "auto"}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf("%s H\n", topic[0])
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("does not show patches for merge commits", func(t *testing.T) {
		tmpDir, stdout, stderr, master, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{"topic..master", "^master^^^"}, LogOption{Format: "oneline", IsTty: false, Decorate: "auto", Patch: true}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`%s K
diff --git a/f.txt b/f.txt
index 02358d2..449e49e 100644
--- a/f.txt
+++ b/f.txt
@@ -1,1 +1,1 @@
-D
+K
%s J
%s D
diff --git a/f.txt b/f.txt
index 96d80cd..02358d2 100644
--- a/f.txt
+++ b/f.txt
@@ -1,1 +1,1 @@
-C
+D
diff --git a/h.txt b/h.txt
index 4e5ce14..4139691 100644
--- a/h.txt
+++ b/h.txt
@@ -1,3 +1,3 @@
-C
+D
 two
 three
`, master[0],
			master[1],
			master[2])
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("shows combined patches for merges", func(t *testing.T) {
		tmpDir, stdout, stderr, master, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{"topic..master", "^master^^^"}, LogOption{Format: "oneline", IsTty: false, Decorate: "auto", Patch: true, Combined: true}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`%s K
diff --git a/f.txt b/f.txt
index 02358d2..449e49e 100644
--- a/f.txt
+++ b/f.txt
@@ -1,1 +1,1 @@
-D
+K
%s J
diff --cc h.txt
index 4139691,f3e97ee..4e78f4f
--- a/h.txt
+++ b/h.txt
@@@ -1,3 -1,3 +1,3 @@@
 -one
 +D
  two
- three
+ G
%s D
diff --git a/f.txt b/f.txt
index 96d80cd..02358d2 100644
--- a/f.txt
+++ b/f.txt
@@ -1,1 +1,1 @@
-C
+D
diff --git a/h.txt b/h.txt
index 4e5ce14..4139691 100644
--- a/h.txt
+++ b/h.txt
@@ -1,3 +1,3 @@
-C
+D
 two
 three
`, master[0],
			master[1],
			master[2])
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("does not list merges with treesame parents for prune paths", func(t *testing.T) {
		tmpDir, stdout, stderr, master, topic := setUp(t)
		defer os.RemoveAll(tmpDir)

		log, _ := NewLog(tmpDir, []string{"g.txt"}, LogOption{Format: "oneline", IsTty: false, Decorate: "auto"}, stdout, stderr)
		log.Run()

		expected := fmt.Sprintf(`%s G
%s F
%s E
%s A
`, topic[1],
			topic[2],
			topic[3],
			master[5])
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("with changes that are undone on a branch leading to a merge", func(t *testing.T) {
		t.Run("does not list commits on the filtered branch", func(t *testing.T) {
			tmpDir, stdout, stderr, _, _ := setUp(t)
			defer os.RemoveAll(tmpDir)

			brunchCmd, _ := NewBranch(tmpDir, []string{"aba", "master~4"}, BranchOption{}, new(bytes.Buffer), new(bytes.Buffer))
			brunchCmd.Run()
			checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "aba")

			for _, n := range []string{"C", "0"} {
				commitTree(t, tmpDir, n, map[string]string{"g.txt": n}, branchTime.Add(time.Second))
			}

			mergeCommit(t, tmpDir, "topic^", MergeOption{ReadOption: write_commit.ReadOption{Message: "J"}}, stdout, stderr)

			commitTree(t, tmpDir, "K", map[string]string{"f.txt": "K"}, branchTime.Add(3*time.Second))

			master := []string{}
			for i := 0; i <= 5; i++ {
				rev, _ := resolveRevision(t, tmpDir, fmt.Sprintf("master~%d", i))
				master = append(master, rev)
			}
			topic := []string{}
			for i := 0; i <= 3; i++ {
				rev, _ := resolveRevision(t, tmpDir, fmt.Sprintf("topic~%d", i))
				topic = append(topic, rev)
			}

			log, _ := NewLog(tmpDir, []string{"g.txt"}, LogOption{Format: "oneline", IsTty: false, Decorate: "auto"}, stdout, stderr)
			log.Run()

			expected := fmt.Sprintf(`%s G
%s F
%s E
%s A
`, topic[1],
				topic[2],
				topic[3],
				master[5])
			if got := stdout.String(); got != expected {
				t.Errorf("want %q, but got %q", expected, got)
			}
		})
	})
}
