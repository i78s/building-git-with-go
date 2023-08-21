package repository

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
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

func (ws *Workspace) ApplyMigration(migration *Migration) error {
	ws.applyChangeList(migration, delete)

	rmdirKeys := make([]string, 0, len(migration.Rmdirs))
	for dir := range migration.Rmdirs {
		rmdirKeys = append(rmdirKeys, dir)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(rmdirKeys)))
	for _, dir := range rmdirKeys {
		if err := ws.removeDirectory(dir); err != nil {
			return err
		}
	}

	mkdirKeys := make([]string, 0, len(migration.Mkdirs))
	for dir := range migration.Mkdirs {
		mkdirKeys = append(mkdirKeys, dir)
	}
	sort.Strings(mkdirKeys)
	for _, dir := range mkdirKeys {
		if err := ws.makeDirectory(dir); err != nil {
			return err
		}
	}

	ws.applyChangeList(migration, update)
	ws.applyChangeList(migration, create)

	return nil
}

func (ws *Workspace) removeDirectory(dirname string) error {
	path := filepath.Join(ws.pathname, dirname)
	if err := os.Remove(path); err != nil {
		if pathErr, ok := err.(*os.PathError); ok {
			if pathErr.Err == syscall.ENOENT || pathErr.Err == syscall.ENOTDIR || pathErr.Err == syscall.ENOTEMPTY {
				return nil
			}
		}
		return err
	}
	return nil
}

func (ws *Workspace) makeDirectory(dirname string) error {
	path := filepath.Join(ws.pathname, dirname)
	stat, err := ws.StatFile(dirname)

	if err == nil && !stat.IsDir() {
		err := os.Remove(path)
		if err != nil {
			return err
		}
	}
	if os.IsNotExist(err) {
		err := os.Mkdir(path, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ws *Workspace) applyChangeList(migration *Migration, action changeType) error {
	for _, plan := range migration.Changes[action] {
		path := filepath.Join(ws.pathname, plan.path)

		err := os.RemoveAll(path)
		if err != nil {
			return err
		}
		if action == delete {
			continue
		}

		flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
		file, err := os.OpenFile(path, flags, fs.FileMode(plan.item.Mode()))
		if err != nil {
			return err
		}
		data, err := migration.BlobData(plan.item.Oid())
		if err != nil {
			return err
		}
		_, err = file.Write([]byte(data))
		if err != nil {
			return err
		}
		err = file.Chmod(fs.FileMode(plan.item.Mode()))
		if err != nil {
			return err
		}
		err = file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
