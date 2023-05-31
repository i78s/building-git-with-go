package jit

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Workspace struct {
	pathname string
}

func NewWorkspace(pathname string) *Workspace {
	return &Workspace{
		pathname: pathname,
	}
}

func (w *Workspace) ListFiles(path string) ([]string, error) {
	var files []string
	ignore := map[string]struct{}{
		".":    {},
		"..":   {},
		".git": {},
	}

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
			if _, ok := ignore[part]; ok {
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
