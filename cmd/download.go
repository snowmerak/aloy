package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aloy-io/aloy/internal/git"
	"github.com/aloy-io/aloy/internal/parser"
	"github.com/aloy-io/aloy/internal/resolver"
	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download dependencies from lock file",
	Long:  "Clones exact versions from aloy.lock without generating CMake files. Useful for CI.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		lf, err := parser.LoadLockFile(dir)
		if err != nil {
			return fmt.Errorf("failed to load aloy.lock: %w", err)
		}
		if len(lf.Packages) == 0 {
			fmt.Println("No packages in lock file. Run 'aloy sync' first.")
			return nil
		}

		modulesBase := filepath.Join(dir, resolver.ModulesDir)
		if err := os.MkdirAll(modulesBase, 0755); err != nil {
			return err
		}

		downloaded := 0
		for _, pkg := range lf.Packages {
			if pkg.IsSystem {
				fmt.Printf("  Skipping system package %s\n", pkg.Name)
				continue
			}
			if pkg.GitURL == "" || pkg.CommitSHA == "" {
				fmt.Fprintf(os.Stderr, "warning: skipping %s (missing git URL or commit SHA)\n", pkg.Name)
				continue
			}
			dest := filepath.Join(modulesBase, pkg.Name)
			shortSHA := pkg.CommitSHA
			if len(shortSHA) > 8 {
				shortSHA = shortSHA[:8]
			}
			fmt.Printf("  Downloading %s @ %s ...\n", pkg.Name, shortSHA)
			if err := git.CloneFull(pkg.GitURL, dest); err != nil {
				return fmt.Errorf("failed to clone %s: %w", pkg.Name, err)
			}
			if err := git.Checkout(dest, pkg.CommitSHA); err != nil {
				return fmt.Errorf("failed to checkout %s @ %s: %w", pkg.Name, pkg.CommitSHA, err)
			}
			downloaded++
		}

		fmt.Printf("Downloaded %d packages.\n", downloaded)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)
}
