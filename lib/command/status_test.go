package command

import (
	"building-git/lib/command/commandtest"
	"os"
	"testing"
)

func TestStatus(t *testing.T) {
	tmpDir, stdout, stderr := commandtest.SetupTestEnvironment(t)
	defer os.RemoveAll(tmpDir)

	filesToAdd := []*filesToAdd{
		{name: "file.txt", content: ""},
		{name: "another.txt", content: ""},
	}
	for _, file := range filesToAdd {
		commandtest.WriteFile(t, tmpDir, file.name, file.content)
	}

	Status(tmpDir, stdout, stderr)

	expected := `?? another.txt
?? file.txt
`
	if got := stdout.String(); got != expected {
		t.Errorf("want %q, but got %q", expected, got)
	}
}
