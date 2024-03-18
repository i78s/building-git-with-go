package command

import (
	"bytes"
	"os"
	"testing"
)

func TestRemoteAddingRemote(t *testing.T) {
	before := func() (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		cmd, _ := NewRemote(tmpDir, []string{"add", "origin", "ssh://example.com/repo"}, RemoteOption{}, new(bytes.Buffer), new(bytes.Buffer))
		cmd.Run()
		return
	}

	t.Run("fails to add an existing remote", func(t *testing.T) {
		tmpDir, stdout, stderr := before()
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewRemote(tmpDir, []string{"add", "origin", "url"}, RemoteOption{}, stdout, stderr)
		status := cmd.Run()

		if status != 128 {
			t.Errorf("want %d, but got %d", 128, status)
		}
		expected := "fatal: remote origin already exists.\n"
		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("lists the remote", func(t *testing.T) {
		tmpDir, stdout, stderr := before()
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewRemote(tmpDir, []string{}, RemoteOption{}, stdout, stderr)
		cmd.Run()

		expected := "origin\n"
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("lists the remote with its URLs", func(t *testing.T) {
		tmpDir, stdout, stderr := before()
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewRemote(tmpDir, []string{}, RemoteOption{Verbose: true}, stdout, stderr)
		cmd.Run()

		expected := `origin	ssh://example.com/repo (fetch)
origin	ssh://example.com/repo (push)
`
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("sets a catch-all fetch refspec", func(t *testing.T) {
		tmpDir, stdout, stderr := before()
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewConfig(tmpDir, []string{}, ConfigOption{File: "local", GetAll: "remote.origin.fetch"}, stdout, stderr)
		cmd.Run()

		expected := "+refs/heads/*:refs/remotes/origin/*\n"
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})
}

func TestRemoteAddingRemoteWithTrackingBrances(t *testing.T) {
	before := func() (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		options := RemoteOption{Tracked: []string{"master", "topic"}}
		cmd, _ := NewRemote(tmpDir, []string{"add", "origin", "ssh://example.com/repo"}, options, new(bytes.Buffer), new(bytes.Buffer))
		cmd.Run()
		return
	}

	t.Run("sets a catch-all fetch refspec", func(t *testing.T) {
		tmpDir, stdout, stderr := before()
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewConfig(tmpDir, []string{}, ConfigOption{File: "local", GetAll: "remote.origin.fetch"}, stdout, stderr)
		cmd.Run()

		expected := "+refs/heads/master:refs/remotes/origin/master\n+refs/heads/topic:refs/remotes/origin/topic\n"
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})
}

func TestRemoteRemovingRemote(t *testing.T) {
	before := func() (tmpDir string, stdout, stderr *bytes.Buffer) {
		tmpDir, stdout, stderr = setupTestEnvironment(t)

		cmd, _ := NewRemote(tmpDir, []string{"add", "origin", "ssh://example.com/repo"}, RemoteOption{}, new(bytes.Buffer), new(bytes.Buffer))
		cmd.Run()
		return
	}

	t.Run("removes the remote", func(t *testing.T) {
		tmpDir, stdout, stderr := before()
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewRemote(tmpDir, []string{"remove", "origin"}, RemoteOption{}, stdout, stderr)
		status := cmd.Run()

		if status != 0 {
			t.Errorf("want %d, but got %d", 0, status)
		}
		expected := ""
		if got := stdout.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("fails to remove a missing remote", func(t *testing.T) {
		tmpDir, stdout, stderr := before()
		defer os.RemoveAll(tmpDir)

		cmd, _ := NewRemote(tmpDir, []string{"remove", "no-such"}, RemoteOption{}, stdout, stderr)
		status := cmd.Run()

		if status != 128 {
			t.Errorf("want %d, but got %d", 128, status)
		}
		expected := "fatal: No such remote: no-such\n"
		if got := stderr.String(); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})
}
