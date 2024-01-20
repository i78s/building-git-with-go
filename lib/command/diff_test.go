package command

import (
	"building-git/lib/command/write_commit"
	"bytes"
	"os"
	"testing"
	"time"
)

func TestDiffWithFileInTheIndex(t *testing.T) {
	setup := func() (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		writeFile(t, tmpDir, "file.txt", "contents\n")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))

		return
	}

	t.Run("diffs a file with modified contents", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		writeFile(t, tmpDir, "file.txt", "changed\n")

		expected := `diff --git a/file.txt b/file.txt
index 12f00e9..5ea2ed4 100644
--- a/file.txt
+++ b/file.txt
@@ -1,1 +1,1 @@
-contents
+changed
`
		assertDiff(t, tmpDir, []string{}, DiffOption{Patch: true}, stdout, stderr, expected)
	})

	t.Run("diffs a file with changed mode", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		makeExecutable(t, tmpDir, "file.txt")

		expected := `diff --git a/file.txt b/file.txt
old mode 100644
new mode 100755
`
		assertDiff(t, tmpDir, []string{}, DiffOption{Patch: true}, stdout, stderr, expected)
	})

	t.Run("diffs a file with changed mode and contents", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		makeExecutable(t, tmpDir, "file.txt")
		writeFile(t, tmpDir, "file.txt", "changed\n")

		expected := `diff --git a/file.txt b/file.txt
old mode 100644
new mode 100755
index 12f00e9..5ea2ed4
--- a/file.txt
+++ b/file.txt
@@ -1,1 +1,1 @@
-contents
+changed
`
		assertDiff(t, tmpDir, []string{}, DiffOption{Patch: true}, stdout, stderr, expected)
	})

	t.Run("diffs a deleted file", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		delete(t, tmpDir, "file.txt")

		expected := `diff --git a/file.txt b/file.txt
deleted file mode 100644
index 12f00e9..0000000
--- a/file.txt
+++ /dev/null
@@ -1,1 +0,0 @@
-contents
`
		assertDiff(t, tmpDir, []string{}, DiffOption{Patch: true}, stdout, stderr, expected)
	})
}

func TestDiffWithHeadCommit(t *testing.T) {
	setup := func() (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		writeFile(t, tmpDir, "file.txt", "contents\n")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
		options := CommitOption{
			ReadOption: write_commit.ReadOption{
				Message: "first commit",
			},
		}
		commit(t, tmpDir, new(bytes.Buffer), new(bytes.Buffer), options, time.Now())

		return
	}

	t.Run("diffs a file with modified contents", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		writeFile(t, tmpDir, "file.txt", "changed\n")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))

		expected := `diff --git a/file.txt b/file.txt
index 12f00e9..5ea2ed4 100644
--- a/file.txt
+++ b/file.txt
@@ -1,1 +1,1 @@
-contents
+changed
`

		assertDiff(t, tmpDir, []string{}, DiffOption{Patch: true, Cached: true}, stdout, stderr, expected)
	})

	t.Run("diffs a file with changed mode", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		makeExecutable(t, tmpDir, "file.txt")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))

		expected := `diff --git a/file.txt b/file.txt
old mode 100644
new mode 100755
`

		assertDiff(t, tmpDir, []string{}, DiffOption{Patch: true, Cached: true}, stdout, stderr, expected)
	})

	t.Run("diffs a file with changed mode and contents", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		makeExecutable(t, tmpDir, "file.txt")
		writeFile(t, tmpDir, "file.txt", "changed\n")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))

		expected := `diff --git a/file.txt b/file.txt
old mode 100644
new mode 100755
index 12f00e9..5ea2ed4
--- a/file.txt
+++ b/file.txt
@@ -1,1 +1,1 @@
-contents
+changed
`

		assertDiff(t, tmpDir, []string{}, DiffOption{Patch: true, Cached: true}, stdout, stderr, expected)
	})

	t.Run("diffs a deleted file", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		delete(t, tmpDir, "file.txt")
		delete(t, tmpDir, ".git/index")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))

		expected := `diff --git a/file.txt b/file.txt
deleted file mode 100644
index 12f00e9..0000000
--- a/file.txt
+++ /dev/null
@@ -1,1 +0,0 @@
-contents
`

		assertDiff(t, tmpDir, []string{}, DiffOption{Patch: true, Cached: true}, stdout, stderr, expected)
	})

	t.Run("diffs an added file", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		writeFile(t, tmpDir, "another.txt", "hello\n")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))

		expected := `diff --git a/another.txt b/another.txt
new file mode 100644
index 0000000..ce01362
--- /dev/null
+++ b/another.txt
@@ -0,0 +1,1 @@
+hello
`

		assertDiff(t, tmpDir, []string{}, DiffOption{Patch: true, Cached: true}, stdout, stderr, expected)
	})
}
