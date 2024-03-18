package repository

import (
	"building-git/lib/config"
	"building-git/lib/repository/remotes"
	"fmt"
	"path/filepath"
)

type InvalidRemoteError struct {
	msg string
}

func (e *InvalidRemoteError) Error() string {
	return fmt.Sprint(e.msg)
}

const DEFAULT_REMOTE = "origin"

type Remotes struct {
	config *config.Config
}

func NewRemotes(config *config.Config) *Remotes {
	return &Remotes{
		config: config,
	}
}

func (r *Remotes) Add(name, url string, branches []string) error {
	if len(branches) == 0 {
		branches = []string{"*"}
	}
	r.config.OpenForUpdate()

	v, _ := r.config.Get([]string{"remote", name, "url"})
	if v != nil {
		r.config.Save()
		return &InvalidBranchError{fmt.Sprintf("remote %s already exists.", name)}
	}

	r.config.Set([]string{"remote", name, "url"}, url)

	for _, branch := range branches {
		source := filepath.Join(HeadsDir(), branch)
		target := filepath.Join(RemotesDir(), name, branch)
		refspec := remotes.NewRefSpec(source, target, true)
		r.config.Add([]string{"remote", name, "fetch"}, refspec.String())
	}
	r.config.Save()
	return nil
}

func (r *Remotes) Remove(name string) error {
	r.config.OpenForUpdate()
	defer r.config.Save()

	if !r.config.RemoveSection([]string{"remote", name}) {
		return &InvalidBranchError{fmt.Sprintf("No such remote: %s", name)}
	}
	return nil
}

func (r *Remotes) ListRemotes() []string {
	r.config.Open()
	return r.config.Subsection("remote")
}

func (r *Remotes) Get(name string) *remotes.Remote {
	r.config.Open()
	if !r.config.HasSection([]string{"remote", name}) {
		return nil
	}
	return remotes.NewRemote(r.config, name)
}
