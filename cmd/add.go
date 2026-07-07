package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aloy-io/aloy/internal/model"
	"github.com/aloy-io/aloy/internal/parser"
	"github.com/spf13/cobra"
)

var (
	addVersion     string
	addAlias       string
	addSystem      bool
	addCMakeOpts   []string
	addTarget      string
	addCMakeTarget string
	addSubdir      string
	addType        string
)

var addCmd = &cobra.Command{
	Use:   "add <git_url>",
	Short: "Add a dependency to the project",
	Long:  "Adds a git-based or system dependency to project.yaml.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		cfg, err := parser.LoadProject(dir)
		if err != nil {
			return fmt.Errorf("failed to load project.yaml (run 'aloy init' first): %w", err)
		}

		gitURL := args[0]
		name := inferPackageName(gitURL)

		dep := model.Dependency{
			Name:    name,
			Version: addVersion,
			Alias:   addAlias,
			Subdir:  addSubdir,
			Type:    addType,
		}

		if addSystem {
			dep.Type = "system"
			dep.Name = gitURL // for system, the arg is the package name, not a URL
			dep.Git = ""
		} else {
			dep.Git = gitURL
		}

		// Parse cmake options (key=value)
		if len(addCMakeOpts) > 0 {
			dep.CMakeOptions = make(map[string]string)
			for _, opt := range addCMakeOpts {
				parts := strings.SplitN(opt, "=", 2)
				if len(parts) == 2 {
					dep.CMakeOptions[parts[0]] = parts[1]
				}
			}
		}

		if addCMakeTarget != "" {
			dep.CMakeTarget = addCMakeTarget
		}

		// Determine target to add dep to
		targetName := addTarget
		if targetName == "" {
			if len(cfg.Targets) == 0 {
				return fmt.Errorf("no targets defined in project.yaml; add a target first")
			}
			if len(cfg.Targets) > 1 {
				return fmt.Errorf("multiple targets exist; specify one with --target (-t)")
			}
			for k := range cfg.Targets {
				targetName = k
			}
		}

		target, ok := cfg.Targets[targetName]
		if !ok {
			return fmt.Errorf("target %q not found in project.yaml", targetName)
		}

		// Check for duplicates across ALL targets (name and alias collision)
		for tName, t := range cfg.Targets {
			for _, existing := range t.Dependencies {
				effective := dep.TargetName()
				existingEffective := existing.TargetName()
				if existing.Name == dep.Name {
					return fmt.Errorf("dependency %q already exists in target %q", dep.Name, tName)
				}
				if effective == existingEffective {
					return fmt.Errorf("name collision: %q (effective name %q) conflicts with existing %q in target %q",
						dep.Name, effective, existing.Name, tName)
				}
			}
		}

		target.Dependencies = append(target.Dependencies, dep)
		cfg.Targets[targetName] = target

		if err := parser.SaveProject(dir, cfg); err != nil {
			return err
		}

		fmt.Printf("Added %s to target %s\n", dep.Name, targetName)
		return nil
	},
}

func init() {
	addCmd.Flags().StringVarP(&addVersion, "version", "v", "", "SemVer constraint (e.g., ^1.2.0)")
	addCmd.Flags().StringVarP(&addAlias, "alias", "a", "", "Alias name for the dependency")
	addCmd.Flags().BoolVar(&addSystem, "system", false, "Mark as system dependency (find_package)")
	addCmd.Flags().StringArrayVar(&addCMakeOpts, "cmake-option", nil, "CMake option in key=value format")
	addCmd.Flags().StringVar(&addCMakeTarget, "cmake-target", "", "CMake target name for linking (if different from package name)")
	addCmd.Flags().StringVarP(&addTarget, "target", "t", "", "Target to add the dependency to (default: first target)")
	addCmd.Flags().StringVarP(&addSubdir, "subdir", "s", "", "Subdirectory path within the repository")
	addCmd.Flags().StringVar(&addType, "type", "", "Dependency type (system, cmake, meson, aloy)")
	rootCmd.AddCommand(addCmd)
}

func inferPackageName(gitURL string) string {
	// Handle SSH URLs: git@github.com:user/repo.git
	if idx := strings.LastIndex(gitURL, ":"); idx != -1 && !strings.Contains(gitURL, "://") {
		gitURL = gitURL[idx+1:]
	}

	name := filepath.Base(gitURL)
	name = strings.TrimSuffix(name, ".git")
	return name
}
