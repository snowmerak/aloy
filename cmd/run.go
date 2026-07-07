package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/aloy-io/aloy/internal/parser"
	"github.com/spf13/cobra"
)

var runConfig string

var runCmd = &cobra.Command{
	Use:   "run [target] [-- <args>...]",
	Short: "Build and run an executable target",
	Long:  "Automatically builds the project and runs the specified executable target.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		cfg, err := parser.LoadProject(dir)
		if err != nil {
			return fmt.Errorf("failed to load project.yaml: %w", err)
		}

		var targetName string
		var runArgs []string

		// Filter out first "--" if left by cobra
		var cleanArgs []string
		for _, a := range args {
			if a == "--" && len(cleanArgs) == 0 && targetName == "" {
				continue
			}
			cleanArgs = append(cleanArgs, a)
		}

		if len(cleanArgs) > 0 {
			if t, ok := cfg.Targets[cleanArgs[0]]; ok && t.Type == "executable" {
				targetName = cleanArgs[0]
				runArgs = cleanArgs[1:]
			} else {
				runArgs = cleanArgs
			}
		}

		if targetName == "" {
			var executables []string
			for name, t := range cfg.Targets {
				if t.Type == "executable" {
					executables = append(executables, name)
				}
			}
			if len(executables) == 0 {
				return fmt.Errorf("no executable targets found in project.yaml")
			}
			if len(executables) > 1 {
				return fmt.Errorf("multiple executable targets found; specify one explicitly: aloy run <target>")
			}
			targetName = executables[0]
		}

		if needsSync(dir) {
			fmt.Println("Running sync first...")
			if err := runSync(dir); err != nil {
				return err
			}
		}

		fmt.Printf("Building (%s)...\n", runConfig)
		cmakeCmd := exec.Command("cmake", "--build", "build", "--config", runConfig)
		cmakeCmd.Dir = dir
		cmakeCmd.Stdout = os.Stdout
		cmakeCmd.Stderr = os.Stderr
		if err := cmakeCmd.Run(); err != nil {
			return fmt.Errorf("build failed: %w", err)
		}

		exeName := targetName
		if runtime.GOOS == "windows" {
			exeName += ".exe"
		}

		searchPaths := []string{
			filepath.Join(dir, "build", runConfig, exeName),
			filepath.Join(dir, "build", exeName),
			filepath.Join(dir, "build", "Debug", exeName),
			filepath.Join(dir, "build", "Release", exeName),
		}

		var exePath string
		for _, p := range searchPaths {
			if _, err := os.Stat(p); err == nil {
				exePath = p
				break
			}
		}

		if exePath == "" {
			return fmt.Errorf("executable %s not found after build", exeName)
		}

		fmt.Printf("Running: %s\n", exePath)
		fmt.Println("----------------------------------------")
		runExecCmd := exec.Command(exePath, runArgs...)
		runExecCmd.Dir = dir
		runExecCmd.Stdout = os.Stdout
		runExecCmd.Stderr = os.Stderr
		runExecCmd.Stdin = os.Stdin
		return runExecCmd.Run()
	},
}

func init() {
	runCmd.Flags().SetInterspersed(false)
	runCmd.Flags().StringVar(&runConfig, "config", "Release", "Build configuration (Debug, Release, RelWithDebInfo, MinSizeRel)")
	rootCmd.AddCommand(runCmd)
}
