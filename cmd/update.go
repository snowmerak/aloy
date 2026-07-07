package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aloy-io/aloy/internal/git"
	"github.com/aloy-io/aloy/internal/model"
	"github.com/aloy-io/aloy/internal/parser"
	"github.com/aloy-io/aloy/internal/resolver"
	"github.com/spf13/cobra"
)

var updateDryRun bool

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update dependencies to latest versions within SemVer range",
	Long:  "Fetches latest tags and updates aloy.lock to the newest compatible versions.",
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
			return fmt.Errorf("failed to load aloy.lock: %w", err)
		}

		modulesBase := filepath.Join(dir, resolver.ModulesDir)

		// Deduplicate direct dependencies across all targets
		type depEntry struct {
			dep     model.Dependency
			dirName string
		}
		seen := make(map[string]bool)
		var deps []depEntry
		for _, target := range cfg.Targets {
			for _, dep := range target.Dependencies {
				key := dep.ModuleDir()
				if seen[key] || dep.Type == "system" || dep.Git == "" {
					continue
				}
				seen[key] = true
				deps = append(deps, depEntry{dep: dep, dirName: key})
			}
		}

		var updated int
		for _, e := range deps {
			destPath := filepath.Join(modulesBase, e.dirName)
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				fmt.Printf("  %s: not cloned, skipping (run sync first)\n", e.dep.Name)
				continue
			}

			if err := git.FetchTags(destPath); err != nil {
				fmt.Printf("  %s: failed to fetch tags: %v\n", e.dep.Name, err)
				continue
			}

			tag, ver, err := git.FindBestTag(destPath, e.dep.Version)
			if err != nil {
				fmt.Printf("  %s: no matching tag: %v\n", e.dep.Name, err)
				continue
			}

			currentSHA, _ := git.GetHeadSHA(destPath)
			tagSHA, _ := git.GetTagSHA(destPath, tag)

			if currentSHA == tagSHA {
				fmt.Printf("  %s: already at %s\n", e.dep.Name, ver)
				continue
			}

			if updateDryRun {
				fmt.Printf("  %s: would update to %s (%s)\n", e.dep.Name, ver, tag)
				updated++
				continue
			}

			if err := git.Checkout(destPath, tag); err != nil {
				fmt.Printf("  %s: failed to checkout %s: %v\n", e.dep.Name, tag, err)
				continue
			}

			newSHA, _ := git.GetHeadSHA(destPath)

			// Update lockfile entry in-place
			pkg := lf.FindPackage(e.dep.Name)
			if pkg == nil {
				pkg = lf.FindPackage(e.dirName)
			}
			if pkg != nil {
				pkg.ResolvedVersion = ver.String()
				pkg.CommitSHA = newSHA
			}

			fmt.Printf("  %s: updated to %s (%s)\n", e.dep.Name, ver, tag)
			updated++
		}

		if updateDryRun {
			fmt.Printf("Dry run: %d packages would be updated.\n", updated)
		} else if updated > 0 {
			if err := parser.SaveLockFile(dir, lf); err != nil {
				return err
			}
			fmt.Printf("Updated %d packages.\n", updated)
		} else {
			fmt.Println("All dependencies are up to date.")
		}

		return nil
	},
}

func init() {
	updateCmd.Flags().BoolVar(&updateDryRun, "dry-run", false, "Show what would be updated without making changes")
	rootCmd.AddCommand(updateCmd)
}
