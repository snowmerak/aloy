package parser

import (
	"fmt"

	"github.com/aloy-io/aloy/internal/model"
)

var validTargetTypes = map[string]bool{
	"executable":     true,
	"library":        true,
	"shared_library": true,
	"header_only":    true,
	"test":           true,
}

// ValidateProject checks a ProjectConfig for common mistakes.
func ValidateProject(cfg *model.ProjectConfig) error {
	if cfg.Project.Name == "" {
		return fmt.Errorf("project.name is required")
	}
	if cfg.Project.Version == "" {
		return fmt.Errorf("project.version is required")
	}
	if len(cfg.Targets) == 0 {
		return fmt.Errorf("at least one target must be defined")
	}
	for name, target := range cfg.Targets {
		if name == "" {
			return fmt.Errorf("target name cannot be empty")
		}
		if !validTargetTypes[target.Type] {
			return fmt.Errorf("target %q: invalid type %q (must be executable, library, shared_library, header_only, or test)", name, target.Type)
		}
		if target.Type != "header_only" && len(target.Sources) == 0 {
			return fmt.Errorf("target %q: sources are required for type %q", name, target.Type)
		}
	}
	return nil
}
