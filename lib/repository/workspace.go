package repository

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var IGNORE = map[string]struct{}{
	".":    {},
	"..":   {},
	".git": {},
}

type Workspace struct {
	pathname string
}

func NewWorkspace(pathname string) *Workspace {
	return &Workspace{
		pathname: pathname,
	}
}

func (w *Workspace) ListDir(dirname string) (map[string]os.FileInfo, error) {
	path := filepath.Join(w.pathname, dirname)
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	stats := map[string]os.FileInfo{}
	for _, file := range files {
		if _, exists := IGNORE[file.Name()]; exists {
			continue
		}
		relativePath, err := filepath.Rel(w.pathname, filepath.Join(path, file.Name()))
		if err != nil {
			return nil, err
		}
		stats[relativePath] = file
	}
	return stats, nil
}

func (w *Workspace) ListFiles(path string) ([]string, error) {
	var files []string

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(w.pathname, path)
		if err != nil {
			return err
		}
		parts := strings.Split(relativePath, "/")
		for _, part := range parts {
			if _, ok := IGNORE[part]; ok {
				return nil
			}
		}
		if info.Mode().IsRegular() {
			relative, err := filepath.Rel(w.pathname, path)
			if err != nil {
				return err
			}
			files = append(files, relative)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("pathspec '%s' did not match any files", path)
	}

	return files, nil
}

func (ws *Workspace) ReadFile(filePath string) (string, error) {
	data, err := ioutil.ReadFile(filepath.Join(ws.pathname, filePath))
	if err != nil {
		if os.IsPermission(err) {
			return "", fmt.Errorf("open('%s'): Permission denied", filePath)
		}
		return "", err
	}
	return string(data), nil
}

func (ws *Workspace) StatFile(filePath string) (fs.FileInfo, error) {
	info, err := os.Stat(filepath.Join(ws.pathname, filePath))

	if err != nil {
		if os.IsPermission(err) {
			return nil, fmt.Errorf("stat('%s'): Permission denied", filePath)
		}
		return nil, err
	}
	return info, nil
}
