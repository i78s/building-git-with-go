package command

import (
	"building-git/lib/repository"
	"fmt"
	"io"
	"path/filepath"
)

type RemoteOption struct {
	Verbose bool
	Tracked []string
}

type Remote struct {
	rootPath string
	args     []string
	options  RemoteOption
	repo     *repository.Repository
	stdout   io.Writer
	stderr   io.Writer
}

func NewRemote(dir string, args []string, options RemoteOption, stdout, stderr io.Writer) (*Remote, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRepository(rootPath)

	return &Remote{
		rootPath: rootPath,
		args:     args,
		options:  options,
		repo:     repo,
		stdout:   stdout,
		stderr:   stderr,
	}, nil
}

func (r *Remote) Run() int {
	if len(r.args) == 0 {
		r.listRemotes()
		return 0
	}

	c := r.args[0]
	r.args = r.args[1:]
	switch c {
	case "add":
		err := r.addRemote()
		if err != nil {
			fmt.Fprintf(r.stderr, "fatal: %s\n", err)
			return 128
		}
	case "remove":
		err := r.removeRemote()
		if err != nil {
			fmt.Fprintf(r.stderr, "fatal: %s\n", err)
			return 128
		}
	}
	return 0
}

func (r *Remote) addRemote() error {
	name, url := r.args[0], r.args[1]
	return r.repo.Remotes().Add(name, url, r.options.Tracked)
}

func (r *Remote) removeRemote() error {
	err := r.repo.Remotes().Remove(r.args[0])
	if err != nil {
		return err
	}
	return nil
}

func (r *Remote) listRemotes() {
	for _, name := range r.repo.Remotes().ListRemotes() {
		r.listRemote(name)
	}
}

func (r *Remote) listRemote(name string) {
	if !r.options.Verbose {
		fmt.Fprintf(r.stdout, "%s\n", name)
		return
	}
	remote := r.repo.Remotes().Get(name)
	if remote == nil {
		return
	}
	fetch, _ := remote.FetchUrl()
	push, _ := remote.PushUrl()

	fmt.Fprintf(r.stdout, "%s\t%s (fetch)\n", name, fetch)
	fmt.Fprintf(r.stdout, "%s\t%s (push)\n", name, push)
}
