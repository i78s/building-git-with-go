package command

import (
	"building-git/lib/command/write_commit"
	"building-git/lib/database"
	"bytes"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"
)

func commitTreeHelper(t *testing.T, tmpDir, message string, files map[string]interface{}, now time.Time) {
	t.Helper()

	for path, contents := range files {
		if contents != ":x" {
			delete(t, tmpDir, path)
		}
		switch v := contents.(type) {
		case string:
			if v == ":x" {
				makeExecutable(t, tmpDir, path)
			} else {
				writeFile(t, tmpDir, path, v)
			}
		case []string:
			writeFile(t, tmpDir, path, v[0])
			makeExecutable(t, tmpDir, path)
		}
	}
	delete(t, tmpDir, ".git/index")
	Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
	options := CommitOption{
		ReadOption: write_commit.ReadOption{
			Message: message,
		},
	}
	commit(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), options, now)
}

// A   B   M
// o---o---o [master]
//
//	\     /
//	 `---o [topic]
//	     C
func merge3(
	t *testing.T,
	tmpDir string,
	base, left, right map[string]interface{},
	stdout, stderr *bytes.Buffer,
) {
	commitTreeHelper(t, tmpDir, "A", base, time.Now())
	commitTreeHelper(t, tmpDir, "B", left, time.Now())

	brunchCmd, _ := NewBranch(tmpDir, []string{"topic", "master^"}, BranchOption{}, new(bytes.Buffer), new(bytes.Buffer))
	brunchCmd.Run()
	checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "topic")
	commitTreeHelper(t, tmpDir, "C", right, time.Now())

	checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "master")
	options := MergeOption{
		ReadOption: write_commit.ReadOption{
			Message: "M",
		},
	}
	mergeCommit(t, tmpDir, "topic", options, stdout, stderr)
}

func assertCleanMerge(t *testing.T, tmpDir string) {
	assertGitStatus(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), "")

	commitObj, _ := loadCommit(t, tmpDir, "@")
	commit := commitObj.(*database.Commit)
	oldHead, _ := loadCommit(t, tmpDir, "@^")
	mergeHead, _ := loadCommit(t, tmpDir, "topic")

	expected := []string{"M\n"}
	if got := commit.Message(); got != expected[0] {
		t.Errorf("want %q, but got %q", expected, got)
	}

	expected = []string{oldHead.Oid(), mergeHead.Oid()}
	if commit.Parents[0] != expected[0] || commit.Parents[1] != expected[1] {
		t.Errorf("want %q, but got %q", expected, commit.Parents)
	}
}

func assertNoMerge(t *testing.T, tmpDir string) {
	commitObj, _ := loadCommit(t, tmpDir, "@")
	commit := commitObj.(*database.Commit)

	expected := []string{"B\n"}
	if got := commit.Message(); got != expected[0] {
		t.Errorf("want %q, but got %q", expected, got)
	}

	if len(commit.Parents) != 1 {
		t.Errorf("want %q, but got %q", 1, len(commit.Parents))
	}
}

func assertIndexEntriesPathWithStage(t *testing.T, tmpDir string, entries []struct {
	path  string
	stage string
}) {
	r := repo(t, tmpDir)
	r.Index.Load()

	actual := []struct {
		path  string
		stage string
	}{}
	for _, entry := range r.Index.EachEntry() {
		actual = append(actual, struct {
			path  string
			stage string
		}{entry.Path(), entry.Stage()})
	}

	if !reflect.DeepEqual(actual, entries) {
		t.Errorf("expected %v, got %v", entries, actual)
	}
}

func TestMergeMergingAncestor(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		now := time.Now()
		commitTree(t, tmpDir, "A", map[string]string{
			"f.txt": "1",
		}, now)
		commitTree(t, tmpDir, "B", map[string]string{
			"f.txt": "2",
		}, now)
		commitTree(t, tmpDir, "C", map[string]string{
			"f.txt": "3",
		}, now)

		options := MergeOption{
			ReadOption: write_commit.ReadOption{
				Message: "D",
			},
		}
		mergeCommit(t, tmpDir, "@^", options, stdout, stderr)

		return
	}

	t.Run("prints the up-to-date message", func(t *testing.T) {
		tmpDir, stdout, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := "Already up to date.\n"
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("does not change the repository state", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		gitObj, _ := loadCommit(t, tmpDir, "@")

		expected := "C\n"
		if got := gitObj.(*database.Commit).Message(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}

		assertGitStatus(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), "")
	})
}

func TestMergeFastForwardMerge(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		now := time.Now()
		commitTree(t, tmpDir, "A", map[string]string{
			"f.txt": "1",
		}, now)
		commitTree(t, tmpDir, "B", map[string]string{
			"f.txt": "2",
		}, now)
		commitTree(t, tmpDir, "C", map[string]string{
			"f.txt": "3",
		}, now)

		brunchCmd, _ := NewBranch(tmpDir, []string{"topic", "@^^"}, BranchOption{}, new(bytes.Buffer), new(bytes.Buffer))
		brunchCmd.Run()
		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "topic")

		options := MergeOption{
			ReadOption: write_commit.ReadOption{
				Message: "M",
			},
		}
		mergeCommit(t, tmpDir, "master", options, stdout, stderr)

		return
	}

	t.Run("prints the fast-forward message", func(t *testing.T) {
		tmpDir, stdout, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		a, _ := resolveRevision(t, tmpDir, "master^^")
		b, _ := resolveRevision(t, tmpDir, "master")

		r := repo(t, tmpDir)

		expected := fmt.Sprintf("Updating %s..%s\nFast-forward\n",
			r.Database.ShortOid(a),
			r.Database.ShortOid(b),
		)
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("updates the current branch HEAD", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		gitObj, _ := loadCommit(t, tmpDir, "@")

		expected := "C\n"
		if got := gitObj.(*database.Commit).Message(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}

		assertGitStatus(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), "")
	})
}

func TestMergeUnconflictedMergeWithTwoFiles(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		merge3(t, tmpDir, map[string]interface{}{
			"f.txt": "1", "g.txt": "1",
		}, map[string]interface{}{
			"f.txt": "2",
		}, map[string]interface{}{
			"g.txt": "2",
		}, stdout, stderr)

		return
	}

	t.Run("puts the combined changes in the workspace", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "2",
			"g.txt": "2",
		})
	})

	t.Run("creates a clean merge", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertCleanMerge(t, tmpDir)
	})
}

func TestMergeUnconflictedMergeWithDeletedFile(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		merge3(t, tmpDir, map[string]interface{}{
			"f.txt": "1", "g.txt": "1",
		}, map[string]interface{}{
			"f.txt": "2",
		}, map[string]interface{}{
			"g.txt": nil,
		}, stdout, stderr)

		return
	}

	t.Run("puts the combined changes in the workspace", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "2",
		})
	})

	t.Run("creates a clean merge", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertCleanMerge(t, tmpDir)
	})
}

func TestMergeUnconflictedMergeSameAdditionOnBothSides(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		merge3(t, tmpDir, map[string]interface{}{
			"f.txt": "1",
		}, map[string]interface{}{
			"g.txt": "2",
		}, map[string]interface{}{
			"g.txt": "2",
		}, stdout, stderr)

		return
	}

	t.Run("puts the combined changes in the workspace", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "1",
			"g.txt": "2",
		})
	})

	t.Run("creates a clean merge", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertCleanMerge(t, tmpDir)
	})
}

func TestMergeUnconflictedMergeSameEditOnBothSides(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		merge3(t, tmpDir, map[string]interface{}{
			"f.txt": "1",
		}, map[string]interface{}{
			"f.txt": "2",
		}, map[string]interface{}{
			"f.txt": "2",
		}, stdout, stderr)

		return
	}

	t.Run("puts the combined changes in the workspace", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "2",
		})
	})

	t.Run("creates a clean merge", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertCleanMerge(t, tmpDir)
	})
}

func TestMergeUnconflictedMergeInFileMergePossible(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		merge3(t, tmpDir, map[string]interface{}{
			"f.txt": "1\n2\n3\n",
		}, map[string]interface{}{
			"f.txt": "4\n2\n3\n",
		}, map[string]interface{}{
			"f.txt": "1\n2\n5\n",
		}, stdout, stderr)

		return
	}

	t.Run("puts the combined changes in the workspace", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "4\n2\n5\n",
		})
	})

	t.Run("creates a clean merge", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertCleanMerge(t, tmpDir)
	})
}

func TestMergeUnconflictedMergeEditAndModeChange(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		merge3(t, tmpDir, map[string]interface{}{
			"f.txt": "1",
		}, map[string]interface{}{
			"f.txt": "2",
		}, map[string]interface{}{
			"f.txt": ":x",
		}, stdout, stderr)

		return
	}

	t.Run("puts the combined changes in the workspace", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "2",
		})
		assertExecutable(t, tmpDir, "f.txt")
	})

	t.Run("creates a clean merge", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertCleanMerge(t, tmpDir)
	})
}

func TestMergeUnconflictedMergeModeChangeAndEdit(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		merge3(t, tmpDir, map[string]interface{}{
			"f.txt": "1",
		}, map[string]interface{}{
			"f.txt": ":x",
		}, map[string]interface{}{
			"f.txt": "3",
		}, stdout, stderr)

		return
	}

	t.Run("puts the combined changes in the workspace", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "3",
		})
		assertExecutable(t, tmpDir, "f.txt")
	})

	t.Run("creates a clean merge", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertCleanMerge(t, tmpDir)
	})
}

func TestMergeUnconflictedMergeSameDeletionOnBothSides(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		merge3(t, tmpDir, map[string]interface{}{
			"f.txt": "1", "g.txt": "1",
		}, map[string]interface{}{
			"g.txt": nil,
		}, map[string]interface{}{
			"g.txt": nil,
		}, stdout, stderr)

		return
	}

	t.Run("puts the combined changes in the workspace", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "1",
		})
	})

	t.Run("creates a clean merge", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertCleanMerge(t, tmpDir)
	})
}

func TestMergeUnconflictedMergeDeleteAddParent(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		merge3(t, tmpDir, map[string]interface{}{
			"nest/f.txt": "1",
		}, map[string]interface{}{
			"nest/f.txt": nil,
		}, map[string]interface{}{
			"nest": "3",
		}, stdout, stderr)

		return
	}

	t.Run("puts the combined changes in the workspace", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertWorkspace(t, tmpDir, map[string]string{
			"nest": "3",
		})
	})

	t.Run("creates a clean merge", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertCleanMerge(t, tmpDir)
	})
}

func TestMergeUnconflictedMergeDeleteAddChild(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		merge3(t, tmpDir, map[string]interface{}{
			"nest/f.txt": "1",
		}, map[string]interface{}{
			"nest/f.txt": nil,
		}, map[string]interface{}{
			"nest/f.txt": nil, "nest/f.txt/g.txt": "3",
		}, stdout, stderr)

		return
	}

	t.Run("puts the combined changes in the workspace", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertWorkspace(t, tmpDir, map[string]string{
			"nest/f.txt/g.txt": "3",
		})
	})

	t.Run("creates a clean merge", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertCleanMerge(t, tmpDir)
	})
}

func TestMergeConflictedMergeAddAdd(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		merge3(t, tmpDir, map[string]interface{}{
			"f.txt": "1",
		}, map[string]interface{}{
			"g.txt": "2\n",
		}, map[string]interface{}{
			"g.txt": "3\n",
		}, stdout, stderr)

		return
	}

	t.Run("prints the merge conflicts", func(t *testing.T) {
		tmpDir, stdout, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `Auto-merging g.txt
CONFLICT (add/add): Merge conflict in g.txt
Automatic merge failed; fix conflicts and then commit the result.`
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("puts the conflicted file in the workspace", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "1",
			"g.txt": `<<<<<<< HEAD
2
=======
3
>>>>>>> topic
`,
		})
	})

	t.Run("records the conflict in the index", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertIndexEntriesPathWithStage(t, tmpDir, []struct {
			path  string
			stage string
		}{
			{"f.txt", "0"},
			{"g.txt", "2"},
			{"g.txt", "3"},
		})
	})

	t.Run("does not write a merge commit", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertNoMerge(t, tmpDir)
	})

	t.Run("reports the conflict in the status", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `AA g.txt
`
		assertGitStatus(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), expected)
	})

	t.Run("shows the combined diff against stages 2 and 3", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `diff --cc g.txt
index 0cfbf08,00750ed..2603ab2
--- a/g.txt
+++ b/g.txt
@@@ -1,1 -1,1 +1,5 @@@
++<<<<<<< HEAD
 +2
++=======
+ 3
++>>>>>>> topic
`
		assertDiff(t, tmpDir, []string{}, DiffOption{Patch: true}, new(bytes.Buffer), new(bytes.Buffer), expected)
	})

	t.Run("shows the diff against our version", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `* Unmerged path g.txt
diff --git a/g.txt b/g.txt
index 0cfbf08..2603ab2 100644
--- a/g.txt
+++ b/g.txt
@@ -1,1 +1,5 @@
+<<<<<<< HEAD
 2
+=======
+3
+>>>>>>> topic
`
		assertDiff(t, tmpDir, []string{}, DiffOption{Patch: true, Stage: "2"}, new(bytes.Buffer), new(bytes.Buffer), expected)
	})

	t.Run("shows the diff against their version", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `* Unmerged path g.txt
diff --git a/g.txt b/g.txt
index 00750ed..2603ab2 100644
--- a/g.txt
+++ b/g.txt
@@ -1,1 +1,5 @@
+<<<<<<< HEAD
+2
+=======
 3
+>>>>>>> topic
`
		assertDiff(t, tmpDir, []string{}, DiffOption{Patch: true, Stage: "3"}, new(bytes.Buffer), new(bytes.Buffer), expected)
	})
}

func TestMergeConflictedMergeAddAddModeConflict(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		merge3(t, tmpDir, map[string]interface{}{
			"f.txt": "1",
		}, map[string]interface{}{
			"g.txt": "2",
		}, map[string]interface{}{
			"g.txt": []string{"2"},
		}, stdout, stderr)

		return
	}

	t.Run("prints the merge conflicts", func(t *testing.T) {
		tmpDir, stdout, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `Auto-merging g.txt
CONFLICT (add/add): Merge conflict in g.txt
Automatic merge failed; fix conflicts and then commit the result.`
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("puts the conflicted file in the workspace", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "1",
			"g.txt": "2",
		})
	})

	t.Run("records the conflict in the index", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertIndexEntriesPathWithStage(t, tmpDir, []struct {
			path  string
			stage string
		}{
			{"f.txt", "0"},
			{"g.txt", "2"},
			{"g.txt", "3"},
		})
	})

	t.Run("does not write a merge commit", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertNoMerge(t, tmpDir)
	})

	t.Run("shows the combined diff against stages 2 and 3", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `diff --cc g.txt
index d8263ee,d8263ee..d8263ee
mode 100644,100755..100644
--- a/g.txt
+++ b/g.txt
`
		assertDiff(t, tmpDir, []string{}, DiffOption{Patch: true}, new(bytes.Buffer), new(bytes.Buffer), expected)
	})

	t.Run("reports the mode change in the appropriate diff", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `* Unmerged path g.txt
diff --git a/g.txt b/g.txt
old mode 100755
new mode 100644
`
		assertDiff(t, tmpDir, []string{}, DiffOption{Patch: true, Stage: "3"}, new(bytes.Buffer), new(bytes.Buffer), expected)
	})
}

func TestMergeConflictedMergeFileDirectoryAddition(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		merge3(t, tmpDir, map[string]interface{}{
			"f.txt": "1",
		}, map[string]interface{}{
			"g.txt": "2",
		}, map[string]interface{}{
			"g.txt/nested.txt": "3",
		}, stdout, stderr)

		return
	}

	t.Run("prints the merge conflicts", func(t *testing.T) {
		tmpDir, stdout, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `Adding g.txt/nested.txt
CONFLICT (file/directory): There is a directory with name g.txt in topic. Adding g.txt as g.txt~HEAD
Automatic merge failed; fix conflicts and then commit the result.`
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("puts a namespaced copy of the conflicted file in the workspace", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt":            "1",
			"g.txt~HEAD":       "2",
			"g.txt/nested.txt": "3",
		})
	})

	t.Run("records the conflict in the index", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertIndexEntriesPathWithStage(t, tmpDir, []struct {
			path  string
			stage string
		}{
			{"f.txt", "0"},
			{"g.txt", "2"},
			{"g.txt/nested.txt", "0"},
		})
	})

	t.Run("does not write a merge commit", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertNoMerge(t, tmpDir)
	})

	t.Run("reports the conflict in the status", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `AU g.txt
A  g.txt/nested.txt
?? g.txt~HEAD
`
		assertGitStatus(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), expected)
	})

	t.Run("lists the file as unmerged in the diff", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `* Unmerged path g.txt
`
		assertDiff(t, tmpDir, []string{}, DiffOption{Patch: true}, new(bytes.Buffer), new(bytes.Buffer), expected)
	})
}

func TestMergeConflictedMergeDirectoryFileAddition(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		merge3(t, tmpDir, map[string]interface{}{
			"f.txt": "1",
		}, map[string]interface{}{
			"g.txt/nested.txt": "2",
		}, map[string]interface{}{
			"g.txt": "3",
		}, stdout, stderr)

		return
	}

	t.Run("prints the merge conflicts", func(t *testing.T) {
		tmpDir, stdout, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `Adding g.txt/nested.txt
CONFLICT (directory/file): There is a directory with name g.txt in HEAD. Adding g.txt as g.txt~topic
Automatic merge failed; fix conflicts and then commit the result.`
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("puts a namespaced copy of the conflicted file in the workspace", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt":            "1",
			"g.txt~topic":      "3",
			"g.txt/nested.txt": "2",
		})
	})

	t.Run("records the conflict in the index", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertIndexEntriesPathWithStage(t, tmpDir, []struct {
			path  string
			stage string
		}{
			{"f.txt", "0"},
			{"g.txt", "3"},
			{"g.txt/nested.txt", "0"},
		})
	})

	t.Run("does not write a merge commit", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertNoMerge(t, tmpDir)
	})

	t.Run("reports the conflict in the status", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `UA g.txt
?? g.txt~topic
`
		assertGitStatus(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), expected)
	})

	t.Run("lists the file as unmerged in the diff", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `* Unmerged path g.txt
`
		assertDiff(t, tmpDir, []string{}, DiffOption{Patch: true}, new(bytes.Buffer), new(bytes.Buffer), expected)
	})
}

func TestMergeConflictedMergeEditEdit(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		merge3(t, tmpDir, map[string]interface{}{
			"f.txt": "1\n",
		}, map[string]interface{}{
			"f.txt": "2\n",
		}, map[string]interface{}{
			"f.txt": "3\n",
		}, stdout, stderr)

		return
	}

	t.Run("prints the merge conflicts", func(t *testing.T) {
		tmpDir, stdout, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `Auto-merging f.txt
CONFLICT (content): Merge conflict in f.txt
Automatic merge failed; fix conflicts and then commit the result.`
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("puts the conflicted file in the workspace", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": `<<<<<<< HEAD
2
=======
3
>>>>>>> topic
`,
		})
	})

	t.Run("records the conflict in the index", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertIndexEntriesPathWithStage(t, tmpDir, []struct {
			path  string
			stage string
		}{
			{"f.txt", "1"},
			{"f.txt", "2"},
			{"f.txt", "3"},
		})
	})

	t.Run("does not write a merge commit", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertNoMerge(t, tmpDir)
	})

	t.Run("reports the conflict in the status", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `UU f.txt
`
		assertGitStatus(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), expected)
	})

	t.Run("shows the combined diff against stages 2 and 3", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `diff --cc f.txt
index 0cfbf08,00750ed..2603ab2
--- a/f.txt
+++ b/f.txt
@@@ -1,1 -1,1 +1,5 @@@
++<<<<<<< HEAD
 +2
++=======
+ 3
++>>>>>>> topic
`
		assertDiff(t, tmpDir, []string{}, DiffOption{Patch: true}, new(bytes.Buffer), new(bytes.Buffer), expected)
	})
}

func TestMergeConflictedMergeEditDelete(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		merge3(t, tmpDir, map[string]interface{}{
			"f.txt": "1",
		}, map[string]interface{}{
			"f.txt": "2",
		}, map[string]interface{}{
			"f.txt": nil,
		}, stdout, stderr)

		return
	}

	t.Run("prints the merge conflicts", func(t *testing.T) {
		tmpDir, stdout, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `CONFLICT (modify/delete): f.txt deleted in topic and modified in HEAD. Version HEAD of f.txt left in tree.
Automatic merge failed; fix conflicts and then commit the result.`
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("puts the left version in the workspace", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "2",
		})
	})

	t.Run("records the conflict in the index", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertIndexEntriesPathWithStage(t, tmpDir, []struct {
			path  string
			stage string
		}{
			{"f.txt", "1"},
			{"f.txt", "2"},
		})
	})

	t.Run("does not write a merge commit", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertNoMerge(t, tmpDir)
	})

	t.Run("reports the conflict in the status", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `UD f.txt
`
		assertGitStatus(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), expected)
	})

	t.Run("lists the file as unmerged in the diff", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `* Unmerged path f.txt
`
		assertDiff(t, tmpDir, []string{}, DiffOption{Patch: true}, new(bytes.Buffer), new(bytes.Buffer), expected)
	})
}

func TestMergeConflictedMergeDeleteEdit(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		merge3(t, tmpDir, map[string]interface{}{
			"f.txt": "1",
		}, map[string]interface{}{
			"f.txt": nil,
		}, map[string]interface{}{
			"f.txt": "3",
		}, stdout, stderr)

		return
	}

	t.Run("prints the merge conflicts", func(t *testing.T) {
		tmpDir, stdout, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `CONFLICT (modify/delete): f.txt deleted in HEAD and modified in topic. Version topic of f.txt left in tree.
Automatic merge failed; fix conflicts and then commit the result.`
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("puts the right version in the workspace", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "3",
		})
	})

	t.Run("records the conflict in the index", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertIndexEntriesPathWithStage(t, tmpDir, []struct {
			path  string
			stage string
		}{
			{"f.txt", "1"},
			{"f.txt", "3"},
		})
	})

	t.Run("does not write a merge commit", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertNoMerge(t, tmpDir)
	})

	t.Run("reports the conflict in the status", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `DU f.txt
`
		assertGitStatus(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), expected)
	})

	t.Run("lists the file as unmerged in the diff", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `* Unmerged path f.txt
`
		assertDiff(t, tmpDir, []string{}, DiffOption{Patch: true}, new(bytes.Buffer), new(bytes.Buffer), expected)
	})
}

func TestMergeConflictedMergeEditAddParent(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		merge3(t, tmpDir, map[string]interface{}{
			"nest/f.txt": "1",
		}, map[string]interface{}{
			"nest/f.txt": "2",
		}, map[string]interface{}{
			"nest": "3",
		}, stdout, stderr)

		return
	}

	t.Run("prints the merge conflicts", func(t *testing.T) {
		tmpDir, stdout, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `CONFLICT (modify/delete): nest/f.txt deleted in topic and modified in HEAD. Version HEAD of nest/f.txt left in tree.
CONFLICT (directory/file): There is a directory with name nest in HEAD. Adding nest as nest~topic
Automatic merge failed; fix conflicts and then commit the result.`
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("puts a namespaced copy of the conflicted file in the workspace", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertWorkspace(t, tmpDir, map[string]string{
			"nest/f.txt": "2",
			"nest~topic": "3",
		})
	})

	t.Run("records the conflict in the index", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertIndexEntriesPathWithStage(t, tmpDir, []struct {
			path  string
			stage string
		}{
			{"nest", "3"},
			{"nest/f.txt", "1"},
			{"nest/f.txt", "2"},
		})
	})

	t.Run("does not write a merge commit", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertNoMerge(t, tmpDir)
	})

	t.Run("reports the conflict in the status", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `UA nest
UD nest/f.txt
?? nest~topic
`
		assertGitStatus(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), expected)
	})

	t.Run("lists the file as unmerged in the diff", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `* Unmerged path nest
* Unmerged path nest/f.txt
`
		assertDiff(t, tmpDir, []string{}, DiffOption{Patch: true}, new(bytes.Buffer), new(bytes.Buffer), expected)
	})
}

func TestMergeConflictedMergeEditAddChild(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		merge3(t, tmpDir, map[string]interface{}{
			"nest/f.txt": "1",
		}, map[string]interface{}{
			"nest/f.txt": "2",
		}, map[string]interface{}{
			"nest/f.txt": nil, "nest/f.txt/g.txt": "3",
		}, stdout, stderr)

		return
	}

	t.Run("prints the merge conflicts", func(t *testing.T) {
		tmpDir, stdout, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `Adding nest/f.txt/g.txt
CONFLICT (modify/delete): nest/f.txt deleted in topic and modified in HEAD. Version HEAD of nest/f.txt left in tree at nest/f.txt~HEAD.
Automatic merge failed; fix conflicts and then commit the result.`
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("puts a namespaced copy of the conflicted file in the workspace", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertWorkspace(t, tmpDir, map[string]string{
			"nest/f.txt~HEAD":  "2",
			"nest/f.txt/g.txt": "3",
		})
	})

	t.Run("records the conflict in the index", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertIndexEntriesPathWithStage(t, tmpDir, []struct {
			path  string
			stage string
		}{
			{"nest/f.txt", "1"},
			{"nest/f.txt", "2"},
			{"nest/f.txt/g.txt", "0"},
		})
	})

	t.Run("does not write a merge commit", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		assertNoMerge(t, tmpDir)
	})

	t.Run("reports the conflict in the status", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `UD nest/f.txt
A  nest/f.txt/g.txt
?? nest/f.txt~HEAD
`
		assertGitStatus(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), expected)
	})

	t.Run("lists the file as unmerged in the diff", func(t *testing.T) {
		tmpDir, _, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		expected := `* Unmerged path nest/f.txt
`
		assertDiff(t, tmpDir, []string{}, DiffOption{Patch: true}, new(bytes.Buffer), new(bytes.Buffer), expected)
	})
}

func TestMergeMultipleCommonAncestors(t *testing.T) {
	//   A   B   C       M1  H   M2
	//   o---o---o-------o---o---o
	//        \         /       /
	//         o---o---o G     /
	//         D  E \         /
	//               `-------o
	//                       F

	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		commitTree(t, tmpDir, "A", map[string]string{
			"f.txt": "1",
		}, time.Now())
		commitTree(t, tmpDir, "B", map[string]string{
			"f.txt": "2",
		}, time.Now())
		commitTree(t, tmpDir, "C", map[string]string{
			"f.txt": "3",
		}, time.Now())

		brunchCmd, _ := NewBranch(tmpDir, []string{"topic", "master^"}, BranchOption{}, new(bytes.Buffer), new(bytes.Buffer))
		brunchCmd.Run()
		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "topic")
		commitTree(t, tmpDir, "D", map[string]string{
			"g.txt": "1",
		}, time.Now())
		commitTree(t, tmpDir, "E", map[string]string{
			"g.txt": "2",
		}, time.Now())
		commitTree(t, tmpDir, "F", map[string]string{
			"g.txt": "3",
		}, time.Now())

		brunchCmd, _ = NewBranch(tmpDir, []string{"joiner", "topic^"}, BranchOption{}, new(bytes.Buffer), new(bytes.Buffer))
		brunchCmd.Run()
		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "joiner")
		commitTree(t, tmpDir, "G", map[string]string{
			"h.txt": "1",
		}, time.Now())

		checkout(tmpDir, new(bytes.Buffer), new(bytes.Buffer), "master")

		return
	}

	t.Run("performs the first merge", func(t *testing.T) {
		tmpDir, stdout, stderr := setUp(t)
		defer os.RemoveAll(tmpDir)

		options := MergeOption{
			ReadOption: write_commit.ReadOption{Message: "merge joiner"},
		}
		status := mergeCommit(t, tmpDir, "joiner", options, stdout, stderr)

		if status != 0 {
			t.Errorf("want %q, but got %q", 0, status)
		}

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "3",
			"g.txt": "2",
			"h.txt": "1",
		})

		assertGitStatus(t, tmpDir, stdout, stderr, "")
	})

	t.Run("performs the second merge", func(t *testing.T) {
		tmpDir, stdout, stderr := setUp(t)
		defer os.RemoveAll(tmpDir)

		options := MergeOption{
			ReadOption: write_commit.ReadOption{Message: "merge joiner"},
		}
		mergeCommit(t, tmpDir, "joiner", options, stdout, stderr)

		commitTree(t, tmpDir, "H", map[string]string{
			"f.txt": "4",
		}, time.Now())

		options = MergeOption{
			ReadOption: write_commit.ReadOption{Message: "merge topic"},
		}
		status := mergeCommit(t, tmpDir, "topic", options, stdout, stderr)
		if status != 0 {
			t.Errorf("want %q, but got %q", 0, status)
		}

		assertWorkspace(t, tmpDir, map[string]string{
			"f.txt": "4",
			"g.txt": "3",
			"h.txt": "1",
		})

		assertGitStatus(t, tmpDir, stdout, stderr, "")
	})
}

func TestMergeConflictResolution(t *testing.T) {
	setUp := func(t *testing.T) (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		merge3(t, tmpDir, map[string]interface{}{
			"f.txt": "1\n",
		}, map[string]interface{}{
			"f.txt": "2\n",
		}, map[string]interface{}{
			"f.txt": "3\n",
		}, stdout, stderr)

		return
	}

	t.Run("prevents commits with unmerged entries", func(t *testing.T) {
		tmpDir, stdout, stderr := setUp(t)
		defer os.RemoveAll(tmpDir)

		options := CommitOption{
			ReadOption: write_commit.ReadOption{
				Message: "commit",
			},
		}
		status := commit(t, tmpDir, stdout, stderr, options, time.Now())

		expectedError := `error: Committing is not possible because you have unmerged files.
hint: Fix them up in the work tree, and then use 'jit add/rm <file>'
hint: as appropriate to mark resolution and make a commit.
fatal: Exiting because of an unresolved conflict.`
		if got := stderr.String(); got != expectedError {
			t.Errorf("want %q, but got %q", expectedError, got)
		}

		if status != 128 {
			t.Errorf("want %q, but got %q", status, 128)
		}

		commitObj, _ := loadCommit(t, tmpDir, "@")
		commit := commitObj.(*database.Commit)
		expected := []string{"B\n"}
		if got := commit.Message(); got != expected[0] {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("prevents merge --continue with unmerged entries", func(t *testing.T) {
		tmpDir, stdout, stderr := setUp(t)
		defer os.RemoveAll(tmpDir)

		options := MergeOption{
			Mode: "continue",
			ReadOption: write_commit.ReadOption{
				Message: "",
			},
		}
		status := mergeCommit(t, tmpDir, "", options, stdout, stderr)

		expectedError := `error: Committing is not possible because you have unmerged files.
hint: Fix them up in the work tree, and then use 'jit add/rm <file>'
hint: as appropriate to mark resolution and make a commit.
fatal: Exiting because of an unresolved conflict.
`
		if got := stderr.String(); got != expectedError {
			t.Errorf("want %q, but got %q", expectedError, got)
		}

		if status != 128 {
			t.Errorf("want %q, but got %q", status, 128)
		}

		commitObj, _ := loadCommit(t, tmpDir, "@")
		commit := commitObj.(*database.Commit)
		expected := []string{"B\n"}
		if got := commit.Message(); got != expected[0] {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("commits a merge after resolving conflicts", func(t *testing.T) {
		tmpDir, stdout, stderr := setUp(t)
		defer os.RemoveAll(tmpDir)

		Add(tmpDir, []string{"f.txt"}, new(bytes.Buffer), new(bytes.Buffer))
		options := CommitOption{
			ReadOption: write_commit.ReadOption{
				Message: "commit",
			},
		}
		status := commit(t, tmpDir, stdout, stderr, options, time.Now())
		if status != 0 {
			t.Errorf("want %q, but got %q", status, 0)
		}

		commitObj, _ := loadCommit(t, tmpDir, "@")
		commit := commitObj.(*database.Commit)
		expected := []string{"M\n"}
		if got := commit.Message(); got != expected[0] {
			t.Errorf("want %q, but got %q", expected, got)
		}

		parents := []string{}
		for _, oid := range commit.Parents {
			commitObj, _ := loadCommit(t, tmpDir, oid)
			parents = append(parents, commitObj.(*database.Commit).Message())
		}
		expected = []string{"B\n", "C\n"}
		if parents[0] != expected[0] || parents[1] != expected[1] {
			t.Errorf("want %q, but got %q", expected, parents)
		}
	})

	t.Run("allows merge --continue after resolving conflicts", func(t *testing.T) {
		tmpDir, stdout, stderr := setUp(t)
		defer os.RemoveAll(tmpDir)

		Add(tmpDir, []string{"f.txt"}, new(bytes.Buffer), new(bytes.Buffer))
		options := MergeOption{
			Mode: "continue",
			ReadOption: write_commit.ReadOption{
				Message: "",
			},
		}
		status := mergeCommit(t, tmpDir, "", options, stdout, stderr)
		if status != 0 {
			t.Errorf("want %q, but got %q", status, 0)
		}

		commitObj, _ := loadCommit(t, tmpDir, "@")
		commit := commitObj.(*database.Commit)
		expected := []string{"M\n"}
		if got := commit.Message(); got != expected[0] {
			t.Errorf("want %q, but got %q", expected, got)
		}

		parents := []string{}
		for _, oid := range commit.Parents {
			commitObj, _ := loadCommit(t, tmpDir, oid)
			parents = append(parents, commitObj.(*database.Commit).Message())
		}
		expected = []string{"B\n", "C\n"}
		if parents[0] != expected[0] || parents[1] != expected[1] {
			t.Errorf("want %q, but got %q", expected, parents)
		}
	})

	t.Run("prevents merge --continue when none is in progress", func(t *testing.T) {
		tmpDir, stdout, stderr := setUp(t)
		defer os.RemoveAll(tmpDir)

		Add(tmpDir, []string{"f.txt"}, new(bytes.Buffer), new(bytes.Buffer))
		options := MergeOption{
			Mode: "continue",
			ReadOption: write_commit.ReadOption{
				Message: "",
			},
		}
		mergeCommit(t, tmpDir, "", options, new(bytes.Buffer), new(bytes.Buffer))
		status := mergeCommit(t, tmpDir, "", options, stdout, stderr)

		expectedError := "fatal: There is no merge in progress (MERGE_HEAD missing).\n"
		if got := stderr.String(); got != expectedError {
			t.Errorf("want %q, but got %q", expectedError, got)
		}
		if status != 128 {
			t.Errorf("want %q, but got %q", status, 128)
		}
	})

	t.Run("aborts the merge", func(t *testing.T) {
		tmpDir, stdout, stderr := setUp(t)
		defer os.RemoveAll(tmpDir)

		option := MergeOption{
			Mode:       Abort,
			ReadOption: write_commit.ReadOption{Message: ""},
		}
		mergeCmd, _ := NewMerge(tmpDir, []string{}, option, stdout, stderr)
		mergeCmd.Run()

		assertGitStatus(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), "")
	})

	t.Run("prevents aborting a merge when none is in progress", func(t *testing.T) {
		tmpDir, stdout, stderr := setUp(t)
		defer os.RemoveAll(tmpDir)

		option := MergeOption{
			Mode:       Abort,
			ReadOption: write_commit.ReadOption{Message: ""},
		}
		mergeCmd, _ := NewMerge(tmpDir, []string{}, option, stdout, stderr)
		mergeCmd.Run()
		status := mergeCmd.Run()

		expectedError := "fatal: There is no merge to abort (MERGE_HEAD missing).\n"
		if got := stderr.String(); got != expectedError {
			t.Errorf("want %q, but got %q", expectedError, got)
		}
		if status != 128 {
			t.Errorf("want %q, but got %q", status, 128)
		}
	})

	t.Run("prevents starting a new merge while one is in progress", func(t *testing.T) {
		tmpDir, stdout, stderr := setUp(t)
		defer os.RemoveAll(tmpDir)

		options := MergeOption{
			ReadOption: write_commit.ReadOption{
				Message: "",
			},
		}
		status := mergeCommit(t, tmpDir, "", options, stdout, stderr)

		expectedError := `error: Merging is not possible because you have unmerged files.
hint: Fix them up in the work tree, and then use 'jit add/rm <file>'
hint: as appropriate to mark resolution and make a commit.
fatal: Exiting because of an unresolved conflict.
`
		if got := stderr.String(); got != expectedError {
			t.Errorf("want %q, but got %q", expectedError, got)
		}
		if status != 128 {
			t.Errorf("want %q, but got %q", status, 128)
		}
	})
}
