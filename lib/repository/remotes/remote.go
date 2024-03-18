package remotes

import "building-git/lib/config"

type Remote struct {
	config *config.Config
	name   string
}

func NewRemote(config *config.Config, name string) *Remote {
	config.Open()

	return &Remote{
		config: config,
		name:   name,
	}
}

func (r *Remote) FetchUrl() (string, error) {
	v, err := r.config.Get([]string{"remote", r.name, "url"})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

func (r *Remote) FetchSpecs() (string, error) {
	v, err := r.config.Get([]string{"remote", r.name, "fetch"})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

func (r *Remote) PushUrl() (string, error) {
	v, err := r.config.Get([]string{"remote", r.name, "url"})
	if err == nil {
		return v.(string), nil
	}
	v, _ = r.FetchUrl()
	return v.(string), nil
}

func (r *Remote) Uploader() (string, error) {
	v, err := r.config.Get([]string{"remote", r.name, "uploadpack"})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}
