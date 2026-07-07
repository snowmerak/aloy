package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aloy-io/aloy/internal/resolver"
	"github.com/spf13/cobra"
)

var cleanAll bool
var cleanCache bool

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove build artifacts",
	Long:  "Deletes the build/ directory. With --all, also removes .my_modules/.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		buildDir := filepath.Join(dir, "build")
		if err := os.RemoveAll(buildDir); err != nil {
			return fmt.Errorf("failed to remove build/: %w", err)
		}
		fmt.Println("Removed build/")

		if cleanAll {
			modulesDir := filepath.Join(dir, resolver.ModulesDir)
			if err := os.RemoveAll(modulesDir); err != nil {
				return fmt.Errorf("failed to remove %s: %w", resolver.ModulesDir, err)
			}
			fmt.Printf("Removed %s/\n", resolver.ModulesDir)

			cmakeLists := filepath.Join(dir, "CMakeLists.txt")
			os.Remove(cmakeLists)
			fmt.Println("Removed CMakeLists.txt")
		}

		if cleanCache {
			home, err := os.UserHomeDir()
			if err == nil {
				cacheDir := filepath.Join(home, ".aloy", "cache")
				if err := os.RemoveAll(cacheDir); err == nil {
					fmt.Println("Removed global cache ~/.aloy/cache/")
				}
			}
		}

		fmt.Println("Clean complete!")
		return nil
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Also remove .my_modules/ and generated CMakeLists.txt")
	cleanCmd.Flags().BoolVar(&cleanCache, "cache", false, "Remove the global git cache (~/.aloy/cache/)")
	rootCmd.AddCommand(cleanCmd)
}
