package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func openConfig(path string) *Config {
	config := NewConfig(path)
	config.Open()

	return config
}

func setUpForTestConfig(t *testing.T) (path string, config *Config) {
	t.Helper()
	tmpDir, _ := os.MkdirTemp("", "jit-test-config")
	path = filepath.Join(tmpDir, "test-config")

	config = openConfig(path)

	return
}

func TestConfigInMemory(t *testing.T) {
	t.Run("returns nil for an unknown key", func(t *testing.T) {
		path, config := setUpForTestConfig(t)
		defer os.RemoveAll(path)

		value, err := config.Get([]string{"core", "editor"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if value != nil {
			t.Errorf("expected nil, got %s", value)
		}
	})

	t.Run("returns the value for a known key", func(t *testing.T) {
		path, config := setUpForTestConfig(t)
		defer os.RemoveAll(path)

		err := config.Set([]string{"core", "editor"}, "ed")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		value, err := config.Get([]string{"core", "editor"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if value != "ed" {
			t.Errorf("expected %s, got %s", "ed", value)
		}
	})

	t.Run("treats section names as case-insensitive", func(t *testing.T) {
		path, config := setUpForTestConfig(t)
		defer os.RemoveAll(path)

		err := config.Set([]string{"core", "editor"}, "ed")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		value, err := config.Get([]string{"Core", "editor"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if value != "ed" {
			t.Errorf("expected %s, got %s", "ed", value)
		}
	})

	t.Run("treats variable names as case-insensitive", func(t *testing.T) {
		path, config := setUpForTestConfig(t)
		defer os.RemoveAll(path)

		err := config.Set([]string{"core", "editor"}, "ed")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		value, err := config.Get([]string{"core", "Editor"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if value != "ed" {
			t.Errorf("expected %s, got %s", "ed", value)
		}
	})

	t.Run("retrieves values from subsections", func(t *testing.T) {
		path, config := setUpForTestConfig(t)
		defer os.RemoveAll(path)

		err := config.Set([]string{"branch", "master", "remote"}, "origin")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		value, err := config.Get([]string{"branch", "master", "remote"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if value != "origin" {
			t.Errorf("expected %s, got %s", "origin", value)
		}
	})

	t.Run("treats subsection names as case-sensitive", func(t *testing.T) {
		path, config := setUpForTestConfig(t)
		defer os.RemoveAll(path)

		err := config.Set([]string{"branch", "master", "remote"}, "origin")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		value, err := config.Get([]string{"branch", "Master", "remote"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if value != nil {
			t.Errorf("expected %v, got %s", nil, value)
		}
	})

	t.Run("adds multiple values for a key", func(t *testing.T) {
		path, config := setUpForTestConfig(t)
		defer os.RemoveAll(path)

		key := []string{"remote", "origin", "fetch"}
		config.Add(key, "master")
		config.Add(key, "topic")

		value, err := config.Get(key)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if value != "topic" {
			t.Errorf("expected %s, got %s", "topic", value)
		}
		values, _ := config.GetAll(key)
		if !reflect.DeepEqual(values[0], "master") && !reflect.DeepEqual(values[1], "topic") {
			t.Errorf("expected %v, got %v", []string{"master", "topic"}, values)
		}
	})

	t.Run("refuses to set a value for a multi-valued key", func(t *testing.T) {
		path, config := setUpForTestConfig(t)
		defer os.RemoveAll(path)

		key := []string{"remote", "origin", "fetch"}
		config.Add(key, "master")
		config.Add(key, "topic")

		err := config.Set(key, "new-value")
		if _, ok := err.(*ConflictError); !ok {
			t.Errorf("expected %s, got %s", &ConflictError{}, err)
		}
	})

	t.Run("replaces all the values for a multi-valued key", func(t *testing.T) {
		path, config := setUpForTestConfig(t)
		defer os.RemoveAll(path)

		defer os.RemoveAll(path)
		key := []string{"remote", "origin", "fetch"}
		config.Add(key, "master")
		config.Add(key, "topic")

		config.ReplaceAll(key, "new-value")
		values, _ := config.GetAll(key)
		if !reflect.DeepEqual(values[0], "new-value") {
			t.Errorf("expected %v, got %v", []string{"new-value"}, values)
		}
	})

}

func TestConfigFileStorage(t *testing.T) {
	var assertFile = func(t *testing.T, path, contents string) {
		t.Helper()
		c, _ := os.ReadFile(path)
		if contents != string(c) {
			t.Errorf("expected %s, got %s", string(c), contents)
		}
	}

	var before = func(t *testing.T) (path string, config *Config) {
		path, config = setUpForTestConfig(t)
		config.OpenForUpdate()
		return
	}

	t.Run("writes a single setting", func(t *testing.T) {
		path, config := before(t)
		defer os.RemoveAll(path)

		config.Set([]string{"core", "editor"}, "ed")
		config.Save()

		expected := `[core]
	editor = ed
`
		assertFile(t, path, expected)
	})

	t.Run("writes multiple settings", func(t *testing.T) {
		path, config := before(t)
		defer os.RemoveAll(path)

		config.Set([]string{"core", "editor"}, "ed")
		config.Set([]string{"user", "name"}, "A. U. Thor")
		config.Set([]string{"Core", "bare"}, true)
		config.Save()

		expected := `[core]
	editor = ed
	bare = true
[user]
	name = A. U. Thor
`
		assertFile(t, path, expected)
	})

	t.Run("writes multiple subsections", func(t *testing.T) {
		path, config := before(t)
		defer os.RemoveAll(path)

		config.Set([]string{"branch", "master", "remote"}, "origin")
		config.Set([]string{"branch", "Master", "remote"}, "another")
		config.Save()

		expected := `[branch "master"]
	remote = origin
[branch "Master"]
	remote = another
`
		assertFile(t, path, expected)
	})

	t.Run("overwrites a variable with a matching name", func(t *testing.T) {
		path, config := before(t)
		defer os.RemoveAll(path)

		config.Set([]string{"merge", "conflictstyle"}, "diff3")
		config.Set([]string{"merge", "ConflictStyle"}, "none")
		config.Save()

		expected := `[merge]
	ConflictStyle = none
`
		assertFile(t, path, expected)
	})

	t.Run("removes a section", func(t *testing.T) {
		path, config := before(t)
		defer os.RemoveAll(path)

		config.Set([]string{"core", "editor"}, "ed")
		config.Set([]string{"remote", "origin", "url"}, "ssh://example.com/repo")
		config.RemoveSection([]string{"core"})
		config.Save()

		expected := `[remote "origin"]
	url = ssh://example.com/repo
`
		assertFile(t, path, expected)
	})

	t.Run("removes a subsection", func(t *testing.T) {
		path, config := before(t)
		defer os.RemoveAll(path)

		config.Set([]string{"core", "editor"}, "ed")
		config.Set([]string{"remote", "origin", "url"}, "ssh://example.com/repo")
		config.RemoveSection([]string{"remote", "origin"})
		config.Save()

		expected := `[core]
	editor = ed
`
		assertFile(t, path, expected)
	})

	t.Run("retrieves persisted settings", func(t *testing.T) {
		path, config := before(t)
		defer os.RemoveAll(path)

		config.Set([]string{"core", "editor"}, "ed")
		config.Save()

		value, err := config.Get([]string{"core", "editor"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if value != "ed" {
			t.Errorf("expected %s, got %s", "ed", value)
		}
	})

	t.Run("retrieves variables from subsections", func(t *testing.T) {
		path, config := before(t)
		defer os.RemoveAll(path)

		config.Set([]string{"branch", "master", "remote"}, "origin")
		config.Set([]string{"branch", "Master", "remote"}, "another")
		config.Save()

		value, err := config.Get([]string{"branch", "master", "remote"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if value != "origin" {
			t.Errorf("expected %s, got %s", "origin", value)
		}

		value, err = config.Get([]string{"branch", "Master", "remote"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if value != "another" {
			t.Errorf("expected %s, got %s", "another", value)
		}
	})

	t.Run("retrieves variables from subsections including dots", func(t *testing.T) {
		path, config := before(t)
		defer os.RemoveAll(path)

		config.Set([]string{"url", "git@github.com:", "insteadOf"}, "gh:")
		config.Save()

		value, err := config.Get([]string{"url", "git@github.com:", "insteadOf"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if value != "gh:" {
			t.Errorf("expected %s, got %s", "gh:", value)
		}
	})

	t.Run("retains the formatting of existing settings", func(t *testing.T) {
		path, config := before(t)
		defer os.RemoveAll(path)

		config.Set([]string{"core", "Editor"}, "ed")
		config.Set([]string{"user", "Name"}, "A. U. Thor")
		config.Set([]string{"core", "Bare"}, true)
		config.Save()

		config = openConfig(path)
		config.OpenForUpdate()
		config.Set([]string{"core", "Bare"}, false)
		config.Save()

		expected := `[core]
	Editor = ed
	Bare = false
[user]
	Name = A. U. Thor
`
		assertFile(t, path, expected)
	})
}
