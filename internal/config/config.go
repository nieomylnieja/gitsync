package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nieomylnieja/gitsync/internal/diff"
)

const defaultRef = "origin/main"

type Config struct {
	StorePath    string        `json:"storePath,omitempty"`
	Root         *Repository   `json:"root"`
	Ignore       []*IgnoreRule `json:"ignore,omitempty"`
	Repositories []*Repository `json:"syncRepositories"`
	SyncFiles    []*File       `json:"syncFiles"`

	path              string
	resolvedStorePath string
}

func (c *Config) GetPath() string {
	return c.path
}

func (c *Config) GetStorePath() string {
	return c.resolvedStorePath
}

type Repository struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Ref  string `json:"ref,omitempty"`

	path       string
	defaultRef string
}

func (r *Repository) GetPath() string {
	return r.path
}

func (r *Repository) GetRef() string {
	if r.Ref != "" {
		return r.Ref
	}
	return r.defaultRef
}

type File struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type IgnoreRule struct {
	RepositoryName *string    `json:"repositoryName,omitempty"`
	FileName       *string    `json:"fileName,omitempty"`
	Regex          *string    `json:"regex,omitempty"`
	Hunk           *diff.Hunk `json:"hunk,omitempty"`
}

func ReadConfig(configPath string) (*Config, error) {
	// #nosec G304
	f, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	defer func() { _ = f.Close() }()
	var config Config
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err = dec.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON config: %w", err)
	}
	if err = config.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	if err = config.setDefaults(); err != nil {
		return nil, fmt.Errorf("failed to set default values: %w", err)
	}
	config.path = configPath
	return &config, nil
}

func (c *Config) Save() error {
	f, err := os.Create(c.path)
	if err != nil {
		return fmt.Errorf("failed to open/create config file: %w", err)
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err = enc.Encode(c); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	return nil
}

func (c *Config) setDefaults() error {
	if c.StorePath == "" {
		if xdgConfigHome, ok := os.LookupEnv("XDG_DATA_HOME"); ok {
			c.resolvedStorePath = filepath.Join(xdgConfigHome, "gitsync")
		} else {
			c.resolvedStorePath = os.ExpandEnv(filepath.Join("$HOME", ".local", "share", "gitsync"))
		}
	} else {
		c.resolvedStorePath = os.ExpandEnv(c.StorePath)
	}
	if strings.HasPrefix(c.resolvedStorePath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		c.resolvedStorePath = filepath.Join(home, c.resolvedStorePath[2:])
	}
	for _, repo := range c.Repositories {
		repo.path = filepath.Join(c.GetStorePath(), repo.Name)
		if repo.Ref == "" {
			repo.defaultRef = defaultRef
		}
	}
	c.Root.path = filepath.Join(c.GetStorePath(), c.Root.Name)
	if c.Root.Ref == "" {
		c.Root.defaultRef = defaultRef
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
			return fmt.Errorf("repository name '%s' is not unique", repo.Name)
		} else {
			unique[repo.Name] = struct{}{}
		}
		if repo.Name == "" {
			return errors.New("repository name is required")
		}
		if repo.URL == "" {
			return errors.New("repository URL is required")
		}
	}
	unique = make(map[string]struct{})
	for _, file := range c.SyncFiles {
		if _, ok := unique[file.Name]; ok {
			return fmt.Errorf("file name '%s' is not unique", file.Name)
		} else {
			unique[file.Name] = struct{}{}
		}
		if err := file.validate(); err != nil {
			return fmt.Errorf("file %s validation failed: %w", file.Name, err)
		}
	}
	for _, ignore := range c.Ignore {
		if err := ignore.validate(); err != nil {
			return fmt.Errorf("ignore rule validation failed: %w", err)
		}
	}
	return nil
}

func (f *File) validate() error {
	if f.Name == "" {
		return errors.New("file name is required")
	}
	if f.Path == "" {
		return errors.New("file path is required")
	}
	return nil
}

func (i *IgnoreRule) validate() error {
	if i.Regex == nil && i.Hunk == nil {
		return errors.New("either 'regex' or 'hunk' needs to be defined")
	}
	return nil
}
