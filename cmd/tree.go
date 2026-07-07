package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aloy-io/aloy/internal/model"
	"github.com/aloy-io/aloy/internal/parser"
	"github.com/aloy-io/aloy/internal/resolver"
	"github.com/spf13/cobra"
)

var treeCmd = &cobra.Command{
	Use:   "tree",
	Short: "Show dependency tree",
	Long:  "Displays the dependency hierarchy of the project.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		cfg, err := parser.LoadProject(dir)
		if err != nil {
			return fmt.Errorf("failed to load project.yaml: %w", err)
		}

		lf, err := parser.LoadLockFile(dir)
		if err != nil {
			lf = &model.LockFile{}
		}

		fmt.Printf("%s (%s)\n", cfg.Project.Name, cfg.Project.Version)

		deps := collectDirectDependencies(cfg)
		for i, dep := range deps {
			isLast := i == len(deps)-1
			printTree(dir, dep, lf, "", isLast)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(treeCmd)
}

func collectDirectDependencies(cfg *model.ProjectConfig) []model.Dependency {
	seen := make(map[string]bool)
	var deps []model.Dependency
	for _, target := range cfg.Targets {
		for _, d := range target.Dependencies {
			logicalName := d.ModuleDir()
			if !seen[logicalName] {
				seen[logicalName] = true
				if d.Type != "system" {
					deps = append(deps, d)
				}
			}
		}
	}
	return deps
}

func printTree(rootDir string, dep model.Dependency, lf *model.LockFile, prefix string, isLast bool) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	locked := lf.FindPackage(dep.ModuleDir())
	if locked == nil {
		locked = lf.FindPackage(dep.Name)
	}

	versionStr := dep.Version
	if locked != nil && locked.ResolvedVersion != "" {
		versionStr = locked.ResolvedVersion
		if len(locked.CommitSHA) >= 7 {
			versionStr += fmt.Sprintf(" %s", locked.CommitSHA[:7])
		}
	} else if versionStr == "" {
		versionStr = "latest"
	}

	fmt.Printf("%s%s%s (%s)\n", prefix, connector, dep.Name, versionStr)

	childPrefix := prefix + "│   "
	if isLast {
		childPrefix = prefix + "    "
	}

	if locked != nil && locked.IsAloyPackage {
		subDir := filepath.Join(rootDir, resolver.ModulesDir, locked.RepoDir, locked.Subdir)
		subCfg, err := parser.LoadProject(subDir)
		if err == nil {
			subDeps := collectDirectDependencies(subCfg)
			for i, subDep := range subDeps {
				subIsLast := i == len(subDeps)-1
				printTree(rootDir, subDep, lf, childPrefix, subIsLast)
			}
		}
	}
}
