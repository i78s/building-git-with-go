package config

import (
	"os"
	"path/filepath"
)

type Stack struct {
	configs map[string]*Config
}

func NewStack(gitPath string) *Stack {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic("cannot find user home directory")
	}
	return &Stack{
		configs: map[string]*Config{
			"local":  NewConfig(filepath.Join(gitPath, "config")),
			"global": NewConfig(filepath.Join(homeDir, ".gitconfig")),
			"system": NewConfig("/etc/gitconfig"),
		},
	}
}

func StackFile(name string, stack *Stack) *Config {
	if config, ok := stack.configs[name]; ok {
		return config
	}
	return NewConfig(name)
}

func (s *Stack) Open() error {
	for _, c := range s.configs {
		err := c.Open()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Stack) Get(key []string) (interface{}, error) {
	all, err := s.GetAll(key)
	if err != nil {
		return nil, err
	}
	return all[len(all)-1], nil
}

func (s *Stack) GetAll(key []string) ([]interface{}, error) {
	values := []interface{}{}
	for _, name := range []string{"system", "global", "local"} {
		err := s.configs[name].Open()
		if err != nil {
			return []interface{}{}, err
		}
		vals, _ := s.configs[name].GetAll(key)
		values = append(values, vals...)
	}
	return values, nil
}
