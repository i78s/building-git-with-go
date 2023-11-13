package pathutils

import "path/filepath"

func Ascend(path string) []string {
	var dirs []string
	for {
		dirs = append(dirs, path)
		parent := filepath.Dir(path)
		if parent == path || parent == "." {
			break
		}
		path = parent
	}
	return dirs
}

func Descend(path string) []string {
	var dirs []string
	for path != "." && path != string(filepath.Separator) {
		dirs = append([]string{path}, dirs...)
		path = filepath.Dir(path)
	}
	return dirs
}
