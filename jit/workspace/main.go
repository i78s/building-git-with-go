package workspace

import "io/ioutil"

type Workspace struct {
	pathname string
}

func NewWorkspace(pathname string) *Workspace {
	return &Workspace{
		pathname: pathname,
	}
}

func (w *Workspace) ListFiles() ([]string, error) {
	files, err := ioutil.ReadDir(w.pathname)
	if err != nil {
		return nil, err
	}

	var fileNames []string
	for _, f := range files {
		name := f.Name()
		if name != "." && name != ".." && name != ".git" {
			fileNames = append(fileNames, name)
		}
	}
	return fileNames, nil
}
