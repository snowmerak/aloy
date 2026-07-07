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

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a dependency from the project",
	Long:  "Removes a dependency from project.yaml and deletes its source from .my_modules/.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		cfg, err := parser.LoadProject(dir)
		if err != nil {
			return fmt.Errorf("failed to load project.yaml: %w", err)
		}

		name := args[0]
		found := false
		var removedDirs []string  // module directories to delete
		var removedNames []string // original names for lockfile cleanup

		for targetName, target := range cfg.Targets {
			// Filter out the dependency
			var newDepList []model.Dependency
			for _, dep := range target.Dependencies {
				if dep.Name == name || dep.Alias == name {
					found = true
					removedDirs = append(removedDirs, dep.ModuleDir())
					removedNames = append(removedNames, dep.Name)
					continue
				}
				newDepList = append(newDepList, dep)
			}

			if len(newDepList) != len(target.Dependencies) {
				target.Dependencies = newDepList
				cfg.Targets[targetName] = target
			}
		}

		if !found {
			return fmt.Errorf("dependency %q not found in any target", name)
		}

		// Save updated config
		if err := parser.SaveProject(dir, cfg); err != nil {
			return err
		}

		// Remove module directories
		for _, dirName := range removedDirs {
			modulePath := filepath.Join(dir, resolver.ModulesDir, dirName)
			if _, err := os.Stat(modulePath); err == nil {
				if err := os.RemoveAll(modulePath); err != nil {
					return fmt.Errorf("failed to remove %s: %w", modulePath, err)
				}
				fmt.Printf("Removed %s/%s/\n", resolver.ModulesDir, dirName)
			}
		}

		// Remove from lock file (match by original name or module dir)
		nameSet := make(map[string]bool)
		for _, n := range removedNames {
			nameSet[n] = true
		}
		for _, d := range removedDirs {
			nameSet[d] = true
		}
		lf, err := parser.LoadLockFile(dir)
		if err != nil {
			return fmt.Errorf("failed to load aloy.lock: %w", err)
		}
		var remaining []model.LockedPackage
		for _, pkg := range lf.Packages {
			if !nameSet[pkg.Name] {
				remaining = append(remaining, pkg)
			}
		}
		lf.Packages = remaining
		if err := parser.SaveLockFile(dir, lf); err != nil {
			return err
		}

		fmt.Printf("Removed dependency %q\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
