package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/nieomylnieja/gitsync/internal/diff"
)

type Config struct {
	StorePath    string              `json:"storePath"`
	Root         *RepositoryConfig   `json:"root"`
	Ignore       []*IgnoreConfig     `json:"ignore,omitempty"`
	Repositories []*RepositoryConfig `json:"syncRepositories"`
	SyncFiles    []*FileConfig       `json:"syncFiles"`

	path string
}

type RepositoryConfig struct {
	Name   string          `json:"name"`
	URL    string          `json:"url"`
	Ref    string          `json:"ref,omitempty"`
	Ignore []*IgnoreConfig `json:"ignore,omitempty"`

	path       string
	defaultRef string
}

func (r *RepositoryConfig) GetPath() string {
	return r.path
}

func (r *RepositoryConfig) GetRef() string {
	if r.Ref != "" {
		return r.Ref
	}
	return r.defaultRef
}

type FileConfig struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type IgnoreConfig struct {
	Regex *string    `json:"regex,omitempty"`
	Hunk  *diff.Hunk `json:"hunk,omitempty"`
}

func ReadConfig(configPath string) (*Config, error) {
	// #nosec G304
	f, err := os.Open(configPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read config file")
	}
	defer func() { _ = f.Close() }()
	var config Config
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err = dec.Decode(&config); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal JSON config")
	}
	if err = config.validate(); err != nil {
		return nil, errors.Wrap(err, "config validation failed")
	}
	if err = config.setDefaults(); err != nil {
		return nil, errors.Wrap(err, "failed to set default values")
	}
	config.path = configPath
	return &config, nil
}

func (c *Config) Save() error {
	f, err := os.Create(c.path)
	if err != nil {
		return errors.Wrap(err, "failed to open/create config file")
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err = enc.Encode(c); err != nil {
		return errors.Wrap(err, "failed to save config")
	}
	return nil
}

func (c *Config) setDefaults() error {
	if c.StorePath == "" {
		if xdgConfigHome, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
			c.StorePath = filepath.Join(xdgConfigHome, "gitsync")
		} else {
			c.StorePath = os.ExpandEnv(filepath.Join("$HOME", ".config", "gitsync"))
		}
	} else {
		c.StorePath = os.ExpandEnv(c.StorePath)
	}
	if strings.HasPrefix(c.StorePath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return errors.Wrap(err, "failed to get user home directory")
		}
		c.StorePath = filepath.Join(home, c.StorePath[2:])
	}
	for _, repo := range c.Repositories {
		repo.path = filepath.Join(c.StorePath, repo.Name)
		if repo.Ref == "" {
			repo.defaultRef = "main"
		}
	}
	c.Root.path = filepath.Join(c.StorePath, c.Root.Name)
	if c.Root.Ignore != nil {
		return errors.New("root repository cannot have ignore rules")
	}
	if c.Root.Ref == "" {
		c.Root.defaultRef = "main"
	}
	return nil
}

func (c *Config) validate() error {
	if len(c.Repositories) == 0 {
		return errors.New("at least one repository is required")
	}
	if len(c.SyncFiles) == 0 {
		return errors.New("at least one file to keep in sync is required")
	}
	unique := make(map[string]struct{})
	for _, repo := range append(c.Repositories, c.Root) {
		if _, ok := unique[repo.Name]; ok {
			return errors.Errorf("repository name '%s' is not unique", repo.Name)
		} else {
			unique[repo.Name] = struct{}{}
		}
		if repo.Name == "" {
			return errors.New("repository name is required")
		}
		if repo.URL == "" {
			return errors.New("repository URL is required")
		}
		for _, file := range c.SyncFiles {
			if err := file.validate(); err != nil {
				return errors.Wrapf(err, "file %s validation failed", file.Name)
			}
		}
	}
	return nil
}

func (f *FileConfig) validate() error {
	if f.Name == "" {
		return errors.New("file name is required")
	}
	if f.Path == "" {
		return errors.New("file path is required")
	}
	return nil
}
