package runbooks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/store"
	"gopkg.in/yaml.v3"
)

type Definition struct {
	SchemaVersion string            `yaml:"schemaVersion" json:"schemaVersion"`
	Name          string            `yaml:"name" json:"name"`
	Description   string            `yaml:"description" json:"description"`
	Match         map[string]string `yaml:"match" json:"match"`
	Steps         []Step            `yaml:"steps" json:"steps"`
	VersionHash   string            `yaml:"-" json:"versionHash"`
}

type Step struct {
	Name                 string            `yaml:"name" json:"name"`
	Kind                 string            `yaml:"kind" json:"kind"`
	Tool                 string            `yaml:"tool" json:"tool"`
	Args                 map[string]string `yaml:"args" json:"args"`
	Required             bool              `yaml:"required" json:"required"`
	Timeout              time.Duration     `yaml:"timeout" json:"timeout"`
	Retries              int               `yaml:"retries" json:"retries"`
	RecommendationOnFail string            `yaml:"recommendation_on_fail" json:"recommendation_on_fail"`
	RecommendationOnPass string            `yaml:"recommendation_on_pass" json:"recommendation_on_pass"`
	AWXTemplateID        int64             `yaml:"awx_template_id" json:"awx_template_id"`
	InventoryID          int64             `yaml:"inventory_id" json:"inventory_id"`
	ExtraVars            map[string]any    `yaml:"extra_vars" json:"extra_vars"`
	PlaybookRef          string            `yaml:"playbook_ref" json:"playbook_ref"`
	RequiresApproval     bool              `yaml:"requires_approval" json:"requires_approval"`
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
		normalizeDefinition(&rb)
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
		normalizeDefinition(&rb)
		if rb.Name == "" {
			return nil, fmt.Errorf("runbook missing name: %s", file)
		}
		result = append(result, rb)
	}
	return result, nil
}

func normalizeDefinition(rb *Definition) {
	if strings.TrimSpace(rb.SchemaVersion) == "" {
		rb.SchemaVersion = "v1"
	}
	if rb.Match == nil {
		rb.Match = map[string]string{}
	}
	for i := range rb.Steps {
		s := &rb.Steps[i]
		if strings.TrimSpace(s.Kind) == "" {
			s.Kind = "check"
		}
		if s.Kind == "check" && strings.TrimSpace(s.Tool) == "" {
			s.Tool = "http_get"
		}
		if s.Args == nil {
			s.Args = map[string]string{}
		}
		if s.Retries < 0 {
			s.Retries = 0
		}
		if strings.TrimSpace(s.Name) == "" {
			s.Name = s.Tool
		}
	}
	blob, _ := json.Marshal(struct {
		SchemaVersion string            `json:"schemaVersion"`
		Name          string            `json:"name"`
		Description   string            `json:"description"`
		Match         map[string]string `json:"match"`
		Steps         []Step            `json:"steps"`
	}{
		SchemaVersion: rb.SchemaVersion,
		Name:          rb.Name,
		Description:   rb.Description,
		Match:         rb.Match,
		Steps:         rb.Steps,
	})
	sum := sha256.Sum256(blob)
	rb.VersionHash = hex.EncodeToString(sum[:])
}
