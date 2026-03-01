package runbooks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/store"
	"gopkg.in/yaml.v3"
)

type Definition struct {
	Name        string            `yaml:"name" json:"name"`
	Description string            `yaml:"description" json:"description"`
	Match       map[string]string `yaml:"match" json:"match"`
	Steps       []Step            `yaml:"steps" json:"steps"`
}

type Step struct {
	Name string            `yaml:"name" json:"name"`
	Tool string            `yaml:"tool" json:"tool"`
	Args map[string]string `yaml:"args" json:"args"`
}

type Store interface {
	GetRunbooks(ctx context.Context) ([]store.Runbook, error)
}

type Loader struct {
	Mode  string
	Path  string
	Store Store
}

func (l Loader) Load(ctx context.Context) ([]Definition, error) {
	switch l.Mode {
	case "db":
		return l.loadFromDB(ctx)
	default:
		return l.loadFromFiles()
	}
}

func (l Loader) loadFromDB(ctx context.Context) ([]Definition, error) {
	if l.Store == nil {
		return nil, fmt.Errorf("runbook store is nil in db mode")
	}
	rows, err := l.Store.GetRunbooks(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]Definition, 0, len(rows))
	for _, row := range rows {
		var rb Definition
		if err := json.Unmarshal(row.Spec, &rb); err != nil {
			return nil, err
		}
		result = append(result, rb)
	}
	return result, nil
}

func (l Loader) loadFromFiles() ([]Definition, error) {
	path := filepath.Clean(filepath.Join(l.Path, "*.yaml"))
	files, err := filepath.Glob(path)
	if err != nil {
		return nil, err
	}
	result := make([]Definition, 0, len(files))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		var rb Definition
		if err := yaml.Unmarshal(data, &rb); err != nil {
			return nil, err
		}
		if rb.Name == "" {
			return nil, fmt.Errorf("runbook missing name: %s", file)
		}
		result = append(result, rb)
	}
	return result, nil
}
