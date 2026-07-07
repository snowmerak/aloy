package parser

import (
	"fmt"
	"os"

	"github.com/aloy-io/aloy/internal/model"
	"gopkg.in/yaml.v3"
)

const ProjectFileName = "project.yaml"

// LoadProject reads and parses a project.yaml from the given directory.
func LoadProject(dir string) (*model.ProjectConfig, error) {
	path := dir + "/" + ProjectFileName
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	var cfg model.ProjectConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return &cfg, nil
}

// SaveProject writes a ProjectConfig to project.yaml in the given directory.
func SaveProject(dir string, cfg *model.ProjectConfig) error {
	path := dir + "/" + ProjectFileName
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal project config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}
