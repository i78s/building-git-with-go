package command

import (
	"bytes"
	"os"
	"testing"
)

func assertDiff(t *testing.T, tmpDir string, args DiffOption, stdout *bytes.Buffer, stderr *bytes.Buffer, expected string) {
	cmd, err := NewDiff(tmpDir, args, stdout, stderr)
	if err != nil {
		t.Fatal(err)
	}
	cmd.Run()

	if got := stdout.String(); got != expected {
		t.Errorf("want %q, but got %q", expected, got)
	}
}

func TestDiffWithFileInTheIndex(t *testing.T) {
	setup := func() (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = SetupTestEnvironment(t)

		WriteFile(t, tmpDir, "file.txt", "contents\n")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))

		return
	}

	t.Run("diffs a file with modified contents", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		WriteFile(t, tmpDir, "file.txt", "changed\n")

		expected := `diff --git a/file.txt b/file.txt
index 12f00e9..5ea2ed4 100644
--- a/file.txt
+++ b/file.txt
-contents
+changed
 
`
		assertDiff(t, tmpDir, DiffOption{}, stdout, stderr, expected)
	})

	t.Run("diffs a file with changed mode", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		MakeExecutable(t, tmpDir, "file.txt")

		expected := `diff --git a/file.txt b/file.txt
old mode 100644
new mode 100755
`
		assertDiff(t, tmpDir, DiffOption{}, stdout, stderr, expected)
	})

	t.Run("diffs a file with changed mode and contents", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		MakeExecutable(t, tmpDir, "file.txt")
		WriteFile(t, tmpDir, "file.txt", "changed\n")

		expected := `diff --git a/file.txt b/file.txt
old mode 100644
new mode 100755
index 12f00e9..5ea2ed4
--- a/file.txt
+++ b/file.txt
-contents
+changed
 
`
		assertDiff(t, tmpDir, DiffOption{}, stdout, stderr, expected)
	})

	t.Run("diffs a deleted file", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		Delete(t, tmpDir, "file.txt")

		expected := `diff --git a/file.txt b/file.txt
deleted file mode 100644
index 12f00e9..0000000
--- a/file.txt
+++ /dev/null
-contents
 
`
		assertDiff(t, tmpDir, DiffOption{}, stdout, stderr, expected)
	})
}

func TestDiffWithHeadCommit(t *testing.T) {
	setup := func() (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = SetupTestEnvironment(t)

		WriteFile(t, tmpDir, "file.txt", "contents\n")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))
		TestCommit(t, tmpDir, "first commit")

		return
	}

	t.Run("diffs a file with modified contents", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		WriteFile(t, tmpDir, "file.txt", "changed\n")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))

		expected := `diff --git a/file.txt b/file.txt
index 12f00e9..5ea2ed4 100644
--- a/file.txt
+++ b/file.txt
-contents
+changed
 
`

		assertDiff(t, tmpDir, DiffOption{Cached: true}, stdout, stderr, expected)
	})

	t.Run("diffs a file with changed mode", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		MakeExecutable(t, tmpDir, "file.txt")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))

		expected := `diff --git a/file.txt b/file.txt
old mode 100644
new mode 100755
`

		assertDiff(t, tmpDir, DiffOption{Cached: true}, stdout, stderr, expected)
	})

	t.Run("diffs a file with changed mode and contents", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		MakeExecutable(t, tmpDir, "file.txt")
		WriteFile(t, tmpDir, "file.txt", "changed\n")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))

		expected := `diff --git a/file.txt b/file.txt
old mode 100644
new mode 100755
index 12f00e9..5ea2ed4
--- a/file.txt
+++ b/file.txt
-contents
+changed
 
`

		assertDiff(t, tmpDir, DiffOption{Cached: true}, stdout, stderr, expected)
	})

	t.Run("diffs a deleted file", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		Delete(t, tmpDir, "file.txt")
		Delete(t, tmpDir, ".git/index")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))

		expected := `diff --git a/file.txt b/file.txt
deleted file mode 100644
index 12f00e9..0000000
--- a/file.txt
+++ /dev/null
-contents
 
`

		assertDiff(t, tmpDir, DiffOption{Cached: true}, stdout, stderr, expected)
	})

	t.Run("diffs an added file", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		WriteFile(t, tmpDir, "another.txt", "hello\n")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))

		expected := `diff --git a/another.txt b/another.txt
new file mode 100644
index 0000000..ce01362
--- /dev/null
+++ b/another.txt
+hello
 
`

		assertDiff(t, tmpDir, DiffOption{Cached: true}, stdout, stderr, expected)
	})
}
