package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/snowmerak/aloy/internal/cmake"
	"github.com/snowmerak/aloy/internal/parser"
	"github.com/snowmerak/aloy/internal/resolver"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Resolve dependencies, generate CMake, and configure the build",
	Long:  "Clones/updates dependencies, generates CMakeLists.txt files, and runs cmake -B build.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		return runSync(dir)
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(dir string) error {
	// 1. Load project config
	cfg, err := parser.LoadProject(dir)
	if err != nil {
		return fmt.Errorf("failed to load project.yaml: %w", err)
	}

	// 1.5 Validate
	if err := parser.ValidateProject(cfg); err != nil {
		return fmt.Errorf("invalid project.yaml: %w", err)
	}

	// 2. Resolve dependency graph
	fmt.Println("Resolving dependencies...")
	resolvedDeps, err := resolver.ResolveGraph(dir, cfg)
	if err != nil {
		return fmt.Errorf("dependency resolution failed: %w", err)
	}
	fmt.Printf("  Resolved %d dependencies\n", len(resolvedDeps))

	// 3. Generate lock file
	lf := resolver.BuildLockFile(resolvedDeps)
	if err := parser.SaveLockFile(dir, lf); err != nil {
		return fmt.Errorf("failed to save lock file: %w", err)
	}

	// 4. Generate CMakeLists.txt for aloy sub-packages or build meson dependencies
	for _, dep := range resolvedDeps {
		if dep.IsSystem {
			continue
		}
		if dep.Type == "meson" {
			if err := buildMesonDependency(dir, dep); err != nil {
				return fmt.Errorf("failed to build meson dependency %s: %w", dep.Name, err)
			}
			continue
		}
		if dep.IsAloyPackage {
			modulePath := filepath.Join(dir, resolver.ModulesDir, dep.RepoDir, dep.Subdir)
			fmt.Printf("  Generating CMake for %s...\n", dep.Name)
			if err := cmake.GenerateForModule(modulePath); err != nil {
				return fmt.Errorf("failed to generate CMake for %s: %w", dep.Name, err)
			}
		}
	}

	// 5. Generate master CMakeLists.txt
	fmt.Println("Generating root CMakeLists.txt...")
	if err := cmake.GenerateMaster(dir, cfg, resolvedDeps); err != nil {
		return fmt.Errorf("failed to generate root CMakeLists.txt: %w", err)
	}

	// 6. Run cmake -B build
	fmt.Println("Configuring build...")
	cmakeCmd := exec.Command("cmake", "-B", "build", "-S", ".")
	cmakeCmd.Dir = dir
	cmakeCmd.Stdout = os.Stdout
	cmakeCmd.Stderr = os.Stderr
	cmakeCmd.Env = append(os.Environ(), "VSLANG=1033", "PYTHONIOENCODING=utf-8", "LANG=en_US.UTF-8", "LC_ALL=en_US.UTF-8")
	if err := cmakeCmd.Run(); err != nil {
		return fmt.Errorf("cmake configuration failed: %w", err)
	}

	fmt.Println("Sync complete!")
	return nil
}

func buildMesonDependency(dir string, dep resolver.ResolvedDep) error {
	if _, err := exec.LookPath("meson"); err != nil {
		return fmt.Errorf("meson not found in PATH. Please install Meson and Ninja to build meson dependencies")
	}

	sourcePath := filepath.Join(dir, resolver.ModulesDir, dep.RepoDir, dep.Subdir)
	installDir := filepath.Join(dir, resolver.ModulesDir, dep.LogicalName+"_install")
	absInstallDir, err := filepath.Abs(installDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of install dir: %w", err)
	}

	buildDir := filepath.Join(sourcePath, "build")
	privateFile := filepath.Join(buildDir, "meson-private", "meson.private")

	env := append(os.Environ(), "VSLANG=1033", "PYTHONIOENCODING=utf-8", "LANG=en_US.UTF-8", "LC_ALL=en_US.UTF-8")

	if _, err := os.Stat(privateFile); os.IsNotExist(err) {
		fmt.Printf("  [Meson] Configuring %s...\n", dep.Name)
		cmd := exec.Command("meson", "setup", "build", "--prefix="+absInstallDir)
		cmd.Dir = sourcePath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = env
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("meson setup failed: %w", err)
		}
	} else {
		fmt.Printf("  [Meson] %s already configured, skipping setup.\n", dep.Name)
	}

	fmt.Printf("  [Meson] Compiling %s...\n", dep.Name)
	compileCmd := exec.Command("meson", "compile", "-C", "build")
	compileCmd.Dir = sourcePath
	compileCmd.Stdout = os.Stdout
	compileCmd.Stderr = os.Stderr
	compileCmd.Env = env
	if err := compileCmd.Run(); err != nil {
		// Fallback to ninja
		ninjaCmd := exec.Command("ninja", "-C", "build")
		ninjaCmd.Dir = sourcePath
		ninjaCmd.Stdout = os.Stdout
		ninjaCmd.Stderr = os.Stderr
		ninjaCmd.Env = env
		if err2 := ninjaCmd.Run(); err2 != nil {
			return fmt.Errorf("meson compile and ninja both failed: %w (ninja: %v)", err, err2)
		}
	}

	fmt.Printf("  [Meson] Installing %s to local staging...\n", dep.Name)
	installCmd := exec.Command("meson", "install", "-C", "build")
	installCmd.Dir = sourcePath
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	installCmd.Env = env
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("meson install failed: %w", err)
	}

	return nil
}
