package jit

import (
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
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
	err := w.listFilesRecursive(path, &files)
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (w *Workspace) listFilesRecursive(path string, files *[]string) error {
	ignore := map[string]struct{}{
		".":    {},
		"..":   {},
		".git": {},
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if _, ok := ignore[entry.Name()]; ok {
			continue
		}

		fullPath := filepath.Join(path, entry.Name())

		if entry.IsDir() {
			err := w.listFilesRecursive(fullPath, files)
			if err != nil {
				return err
			}
		} else {
			rel, err := filepath.Rel(w.pathname, fullPath)
			if err != nil {
				return err
			}
			*files = append(*files, rel)
		}
	}
	return nil
}

func (ws *Workspace) ReadFile(filePath string) (string, error) {
	data, err := ioutil.ReadFile(filepath.Join(ws.pathname, filePath))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (ws *Workspace) StatFile(filePath string) (fs.FileInfo, error) {
	return os.Stat(filepath.Join(ws.pathname, filePath))
}
