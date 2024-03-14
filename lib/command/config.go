package command

import (
	"building-git/lib/config"
	"building-git/lib/repository"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

type KeyDoesNotContainError struct {
	message string
}

func (c KeyDoesNotContainError) Error() string {
	return c.message
}

type InvalidKeyError struct {
	message string
}

func (c InvalidKeyError) Error() string {
	return c.message
}

type ConfigOption struct {
	File          string
	Add           string
	Replace       string
	GetAll        string
	RemoveSection string
	Unset         string
	UnsetAll      string
}

type Config struct {
	rootPath string
	args     []string
	options  ConfigOption
	repo     *repository.Repository
	stdout   io.Writer
	stderr   io.Writer
}

func NewConfig(dir string, args []string, options ConfigOption, stdout, stderr io.Writer) (*Config, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRepository(rootPath)

	return &Config{
		rootPath: rootPath,
		args:     args,
		options:  options,
		repo:     repo,
		stdout:   stdout,
		stderr:   stderr,
	}, nil
}

func (c *Config) Run() int {
	if c.options.Add != "" {
		c.addVariable()
	} else if c.options.Replace != "" {
		c.replaceVariable()
		return 0
	} else if c.options.GetAll != "" {
		err := c.getAllValues()
		if err != nil {
			return 1
		}
		return 0
	} else if c.options.Unset != "" {
		err := c.unsetSingle()
		if err != nil {
			return 5
		}
		return 0
	} else if c.options.UnsetAll != "" {
		err := c.unsetAll()
		if err != nil {
			return 1
		}
		return 0
	} else if c.options.RemoveSection != "" {
		c.removeSection()
		return 0
	}

	key, err := c.parseKey(c.args[0])
	if err != nil {
		if _, ok := err.(*KeyDoesNotContainError); ok {
			fmt.Fprintf(c.stderr, "%s\n", err)
			return 2
		}
		fmt.Fprintf(c.stderr, "%s\n", err)
		return 3
	}
	if len(c.args) > 1 {
		value := c.args[1]
		err := c.editConfig(func(config *config.Config) error {
			return config.Set(key, value)
		})
		if err != nil {
			if _, ok := err.(*config.ConflictError); ok {
				return 5
			}
			return 1
		}
	} else {
		err := c.readConfig(func(config StackConfig) ([]interface{}, error) {
			val, err := config.Get(key)
			if val == nil {
				return []interface{}{}, fmt.Errorf("error: value is empty")
			}
			return []interface{}{val}, err
		})
		if err != nil {
			if _, ok := err.(*config.ParseError); ok {
				return 3
			}
			return 1
		}
	}

	return 0
}

func (c *Config) addVariable() error {
	key, err := c.parseKey(c.options.Add)
	if err != nil {
		return err
	}
	err = c.editConfig(func(config *config.Config) error {
		config.Add(key, c.args[0])
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) replaceVariable() error {
	key, err := c.parseKey(c.options.Replace)
	if err != nil {
		return err
	}
	err = c.editConfig(func(config *config.Config) error {
		config.ReplaceAll(key, c.args[0])
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) unsetSingle() error {
	key, err := c.parseKey(c.options.Unset)
	if err != nil {
		return err
	}
	err = c.editConfig(func(config *config.Config) error {
		return config.Unset(key)
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) unsetAll() error {
	key, err := c.parseKey(c.options.UnsetAll)
	if err != nil {
		return err
	}
	err = c.editConfig(func(conf *config.Config) error {
		return conf.UnsetAll(key, func(lines []*config.Line) error {
			return nil
		})
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) removeSection() {
	key := strings.SplitN(c.options.RemoveSection, ".", 2)
	c.editConfig(func(config *config.Config) error {
		config.RemoveSection(key)
		return nil
	})
}

func (c *Config) getAllValues() error {
	key, err := c.parseKey(c.options.GetAll)
	if err != nil {
		return err
	}
	err = c.readConfig(func(config StackConfig) ([]interface{}, error) {
		return config.GetAll(key)
	})
	if err != nil {
		return err
	}
	return nil
}

type StackConfig interface {
	Open() error
	Get(key []string) (interface{}, error)
	GetAll(key []string) ([]interface{}, error)
}

func (c *Config) readConfig(fn func(config StackConfig) ([]interface{}, error)) error {
	var conf StackConfig = c.repo.Config
	if c.options.File != "" {
		conf = config.StackFile(c.options.File, c.repo.Config)
	}

	conf.Open()
	values, err := fn(conf)
	if err != nil {
		return err
	}
	if len(values) == 0 {
		return fmt.Errorf("error: values are empty")
	}

	for _, v := range values {
		fmt.Fprintf(c.stdout, "%v\n", v)
	}
	return nil
}

func (c *Config) editConfig(fn func(config *config.Config) error) error {
	name := "local"
	if c.options.File != "" {
		name = c.options.File
	}
	config := config.StackFile(name, c.repo.Config)
	config.OpenForUpdate()
	err := fn(config)
	if err != nil {
		return err
	}
	config.Save()
	return nil
}

func (c *Config) parseKey(name string) ([]string, error) {
	parsedName := strings.Split(name, ".")
	if len(parsedName) < 2 {
		return nil, &KeyDoesNotContainError{fmt.Sprintf("error: key does not contain a section: %s", name)}
	}

	section := parsedName[0]
	subsection := parsedName[1 : len(parsedName)-1]
	varName := parsedName[len(parsedName)-1]

	if !config.ConfigValidKey([]string{section, varName}) {
		return nil, &InvalidKeyError{fmt.Sprintf("error: invalid key: %s", name)}
	}

	if len(subsection) == 0 {
		return []string{section, varName}, nil
	}
	return []string{section, strings.Join(subsection, "."), varName}, nil
}
