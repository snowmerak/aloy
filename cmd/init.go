package cmd

import (
	"fmt"
	"os"

	"github.com/aloy-io/aloy/internal/scaffold"
	"github.com/spf13/cobra"
)

var initType string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new aloy project",
	Long:  "Creates project.yaml, src/, include/ directories, and a starter source file.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		return scaffold.Init(dir, initType)
	},
}

func init() {
	initCmd.Flags().StringVarP(&initType, "type", "t", "executable", "Target type: executable, library, shared_library, header_only")
	rootCmd.AddCommand(initCmd)
}
