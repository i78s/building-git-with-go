package workspace

import (
	"io/ioutil"
	"os"
	"path"
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

func (w *Workspace) ListFiles() ([]string, error) {
	var files []string
	ignore := map[string]struct{}{
		".":    {},
		"..":   {},
		".git": {},
	}
	err := filepath.Walk(w.pathname,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if _, ok := ignore[info.Name()]; ok {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if !info.IsDir() {
				rel, err := filepath.Rel(w.pathname, path)
				if err != nil {
					return err
				}
				files = append(files, rel)
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (ws *Workspace) ReadFile(filePath string) (string, error) {
	data, err := ioutil.ReadFile(path.Join(ws.pathname, filePath))
	if err != nil {
		return "", err
	}
	return string(data), nil
}
