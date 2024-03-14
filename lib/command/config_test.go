package command

import (
	"bytes"
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	t.Run("returns 1 for unknown variables", func(t *testing.T) {
		tmpDir, stdout, stderr := setupTestEnvironment(t)
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewConfig(tmpDir, []string{"no.such"}, ConfigOption{File: "local"}, stdout, stderr)
		status := cmd.Run()

		if status != 1 {
			t.Errorf("want %d, but got %d", 1, status)
		}
	})

	t.Run("returns 1 when the key is invalid", func(t *testing.T) {
		tmpDir, stdout, stderr := setupTestEnvironment(t)
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewConfig(tmpDir, []string{"0.0"}, ConfigOption{File: "local"}, stdout, stderr)
		status := cmd.Run()

		if status != 3 {
			t.Errorf("want %d, but got %d", 3, status)
		}
		expected := "error: invalid key: 0.0\n"
		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("returns 2 when no section is given", func(t *testing.T) {
		tmpDir, stdout, stderr := setupTestEnvironment(t)
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewConfig(tmpDir, []string{"no"}, ConfigOption{File: "local"}, stdout, stderr)
		status := cmd.Run()

		if status != 2 {
			t.Errorf("want %d, but got %d", 2, status)
		}
		expected := "error: key does not contain a section: no\n"
		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("returns the value of a set variable", func(t *testing.T) {
		tmpDir, stdout, stderr := setupTestEnvironment(t)
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewConfig(tmpDir, []string{"core.editor", "ed"}, ConfigOption{}, new(bytes.Buffer), new(bytes.Buffer))
		cmd.Run()

		cmd, _ = NewConfig(tmpDir, []string{"Core.Editor"}, ConfigOption{File: "local"}, stdout, stderr)
		status := cmd.Run()

		if status != 0 {
			t.Errorf("want %d, but got %d", 0, status)
		}
		expected := "ed\n"
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("returns the value of a set variable in a subsection", func(t *testing.T) {
		tmpDir, stdout, stderr := setupTestEnvironment(t)
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewConfig(tmpDir, []string{"remote.origin.url", "git@github.com:jcoglan.jit"}, ConfigOption{}, new(bytes.Buffer), new(bytes.Buffer))
		cmd.Run()

		cmd, _ = NewConfig(tmpDir, []string{"Remote.origin.URL"}, ConfigOption{File: "local"}, stdout, stderr)
		status := cmd.Run()

		if status != 0 {
			t.Errorf("want %d, but got %d", 0, status)
		}
		expected := "git@github.com:jcoglan.jit\n"
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("unset a variable", func(t *testing.T) {
		tmpDir, stdout, stderr := setupTestEnvironment(t)
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewConfig(tmpDir, []string{"core.editor", "ed"}, ConfigOption{}, new(bytes.Buffer), new(bytes.Buffer))
		cmd.Run()
		cmd, _ = NewConfig(tmpDir, []string{}, ConfigOption{Unset: "core.editor"}, new(bytes.Buffer), new(bytes.Buffer))
		cmd.Run()

		cmd, _ = NewConfig(tmpDir, []string{"Core.Editor"}, ConfigOption{File: "local"}, stdout, stderr)
		status := cmd.Run()

		if status != 1 {
			t.Errorf("want %d, but got %d", 1, status)
		}
	})
}

func TestConfigWithMultiValuedVariables(t *testing.T) {
	before := func(t *testing.T) (string, *bytes.Buffer, *bytes.Buffer) {
		tmpDir, stdout, stderr := setupTestEnvironment(t)

		cmd, _ := NewConfig(tmpDir, []string{"master"}, ConfigOption{Add: "remote.origin.fetch"}, new(bytes.Buffer), new(bytes.Buffer))
		cmd.Run()
		cmd, _ = NewConfig(tmpDir, []string{"topic"}, ConfigOption{Add: "remote.origin.fetch"}, new(bytes.Buffer), new(bytes.Buffer))
		cmd.Run()

		return tmpDir, stdout, stderr
	}

	t.Run("returns the last value", func(t *testing.T) {
		tmpDir, stdout, stderr := before(t)
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewConfig(tmpDir, []string{"remote.origin.fetch"}, ConfigOption{}, stdout, stderr)
		status := cmd.Run()

		if status != 0 {
			t.Errorf("want %d, but got %d", 0, status)
		}
		expected := "topic\n"
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("returns all the values", func(t *testing.T) {
		tmpDir, stdout, stderr := before(t)
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewConfig(tmpDir, []string{}, ConfigOption{GetAll: "remote.origin.fetch"}, stdout, stderr)
		status := cmd.Run()

		if status != 0 {
			t.Errorf("want %d, but got %d", 0, status)
		}
		expected := "master\ntopic\n"
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("returns 5 on trying to set a variable", func(t *testing.T) {
		tmpDir, stdout, stderr := before(t)
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewConfig(tmpDir, []string{"remote.origin.fetch", "new-value"}, ConfigOption{}, stdout, stderr)
		status := cmd.Run()

		if status != 5 {
			t.Errorf("want %d, but got %d", 5, status)
		}

		cmd, _ = NewConfig(tmpDir, []string{}, ConfigOption{GetAll: "remote.origin.fetch"}, stdout, stderr)
		status = cmd.Run()

		if status != 0 {
			t.Errorf("want %d, but got %d", 0, status)
		}
		expected := "master\ntopic\n"
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("replaces a variable", func(t *testing.T) {
		tmpDir, stdout, stderr := before(t)
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewConfig(tmpDir, []string{"new-value"}, ConfigOption{Replace: "remote.origin.fetch"}, stdout, stderr)
		cmd.Run()

		cmd, _ = NewConfig(tmpDir, []string{}, ConfigOption{GetAll: "remote.origin.fetch"}, stdout, stderr)
		status := cmd.Run()

		if status != 0 {
			t.Errorf("want %d, but got %d", 0, status)
		}
		expected := "new-value\n"
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("returns 5 on trying to unset a variable", func(t *testing.T) {
		tmpDir, stdout, stderr := before(t)
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewConfig(tmpDir, []string{}, ConfigOption{Unset: "remote.origin.fetch"}, stdout, stderr)
		status := cmd.Run()

		if status != 5 {
			t.Errorf("want %d, but got %d", 5, status)
		}

		cmd, _ = NewConfig(tmpDir, []string{}, ConfigOption{GetAll: "remote.origin.fetch"}, stdout, stderr)
		status = cmd.Run()

		if status != 0 {
			t.Errorf("want %d, but got %d", 0, status)
		}
		expected := "master\ntopic\n"
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("unsets a variable", func(t *testing.T) {
		tmpDir, stdout, stderr := before(t)
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewConfig(tmpDir, []string{}, ConfigOption{UnsetAll: "remote.origin.fetch"}, stdout, stderr)
		cmd.Run()

		cmd, _ = NewConfig(tmpDir, []string{}, ConfigOption{GetAll: "remote.origin.fetch"}, stdout, stderr)
		status := cmd.Run()

		if status != 1 {
			t.Errorf("want %d, but got %d", 1, status)
		}
	})

	t.Run("removes a section", func(t *testing.T) {
		tmpDir, stdout, stderr := before(t)
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewConfig(tmpDir, []string{"core.editor", "ed"}, ConfigOption{}, new(bytes.Buffer), new(bytes.Buffer))
		cmd.Run()
		cmd, _ = NewConfig(tmpDir, []string{"remote.origin.url", "ssh://example.com/repo"}, ConfigOption{}, new(bytes.Buffer), new(bytes.Buffer))
		cmd.Run()

		cmd, _ = NewConfig(tmpDir, []string{}, ConfigOption{RemoveSection: "core"}, stdout, stderr)
		cmd.Run()

		cmd, _ = NewConfig(tmpDir, []string{"remote.origin.url"}, ConfigOption{File: "local"}, stdout, stderr)
		status := cmd.Run()

		if status != 0 {
			t.Errorf("want %d, but got %d", 0, status)
		}
		expected := "ssh://example.com/repo\n"
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}

		cmd, _ = NewConfig(tmpDir, []string{"core.editor"}, ConfigOption{File: "local"}, stdout, stderr)
		status = cmd.Run()

		if status != 1 {
			t.Errorf("want %d, but got %d", 1, status)
		}
	})

	t.Run("removes a subsection", func(t *testing.T) {
		tmpDir, stdout, stderr := before(t)
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewConfig(tmpDir, []string{"core.editor", "ed"}, ConfigOption{}, new(bytes.Buffer), new(bytes.Buffer))
		cmd.Run()
		cmd, _ = NewConfig(tmpDir, []string{"remote.origin.url", "ssh://example.com/repo"}, ConfigOption{}, new(bytes.Buffer), new(bytes.Buffer))
		cmd.Run()

		cmd, _ = NewConfig(tmpDir, []string{}, ConfigOption{RemoveSection: "remote.origin"}, stdout, stderr)
		cmd.Run()

		cmd, _ = NewConfig(tmpDir, []string{"core.editor"}, ConfigOption{File: "local"}, stdout, stderr)
		status := cmd.Run()

		if status != 0 {
			t.Errorf("want %d, but got %d", 0, status)
		}
		expected := "ed\n"
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}

		cmd, _ = NewConfig(tmpDir, []string{"remote.origin.url"}, ConfigOption{File: "local"}, stdout, stderr)
		status = cmd.Run()

		if status != 1 {
			t.Errorf("want %d, but got %d", 1, status)
		}
	})
}
