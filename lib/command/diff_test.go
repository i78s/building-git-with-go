package command

import (
	"building-git/lib/command/commandtest"
	"bytes"
	"os"
	"testing"
)

func assertDiff(t *testing.T, tmpDir string, stdout *bytes.Buffer, stderr *bytes.Buffer, expected string) {
	args := StatusOption{
		Porcelain: true,
	}
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
		tmpDir, stdout, stderr = commandtest.SetupTestEnvironment(t)

		commandtest.WriteFile(t, tmpDir, "file.txt", "contents\n")
		Add(tmpDir, []string{"."}, new(bytes.Buffer), new(bytes.Buffer))

		return
	}

	t.Run("diffs a file with modified contents", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		commandtest.WriteFile(t, tmpDir, "file.txt", "changed\n")

		expected := `diff --git a/file.txt b/file.txt
index 12f00e9..5ea2ed4 100644
--- a/file.txt
+++ b/file.txt
`
		assertDiff(t, tmpDir, stdout, stderr, expected)
	})

	t.Run("diffs a file with changed mode", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		commandtest.MakeExecutable(t, tmpDir, "file.txt")

		expected := `diff --git a/file.txt b/file.txt
old mode 100644
new mode 100755
`
		assertDiff(t, tmpDir, stdout, stderr, expected)
	})

	t.Run("diffs a file with changed mode and contents", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		commandtest.MakeExecutable(t, tmpDir, "file.txt")
		commandtest.WriteFile(t, tmpDir, "file.txt", "changed\n")

		expected := `diff --git a/file.txt b/file.txt
old mode 100644
new mode 100755
index 12f00e9..5ea2ed4
--- a/file.txt
+++ b/file.txt
`
		assertDiff(t, tmpDir, stdout, stderr, expected)
	})

	t.Run("diffs a deleted file", func(t *testing.T) {
		tmpDir, stdout, stderr := setup()
		defer os.RemoveAll(tmpDir)
		commandtest.Delete(t, tmpDir, "file.txt")

		expected := `diff --git a/file.txt b/file.txt
deleted file mode 100644
index 12f00e9..0000000
--- a/file.txt
+++ /dev/null
`
		assertDiff(t, tmpDir, stdout, stderr, expected)
	})
}
