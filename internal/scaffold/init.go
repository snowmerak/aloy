package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aloy-io/aloy/internal/model"
	"github.com/aloy-io/aloy/internal/parser"
)

// Init creates a new aloy project in the given directory.
func Init(dir string, targetType string) error {
	projectName := filepath.Base(dir)

	// Validate target type
	switch targetType {
	case "executable", "library", "shared_library", "header_only":
		// ok
	default:
		return fmt.Errorf("invalid target type %q: must be executable, library, shared_library, or header_only", targetType)
	}

	target := model.Target{
		Type: targetType,
		Includes: model.IncludeConfig{
			Public: []string{"include/"},
		},
	}
	if targetType != "header_only" {
		if targetType == "executable" {
			target.Sources = []string{"src/main.cpp"}
		} else {
			target.Sources = []string{"src/**/*.cpp"}
		}
	}

	cfg := &model.ProjectConfig{
		Project: model.ProjectMeta{
			Name:        projectName,
			Version:     "0.1.0",
			CXXStandard: 17,
		},
		Targets: map[string]model.Target{
			projectName: target,
			projectName + "_test": {
				Type:    "test",
				Sources: []string{"tests/**/*.cpp"},
			},
		},
	}

	// Create directories
	dirs := []string{filepath.Join(dir, "include"), filepath.Join(dir, "tests")}
	if targetType != "header_only" {
		dirs = append(dirs, filepath.Join(dir, "src"))
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", d, err)
		}
	}

	// Create starter source file
	var starterFile, starterContent string
	switch targetType {
	case "executable":
		starterFile = filepath.Join(dir, "src", "main.cpp")
		starterContent = `#include <iostream>

int main() {
    std::cout << "Hello from ` + projectName + `!" << std::endl;
    return 0;
}
`
	case "library", "shared_library":
		starterFile = filepath.Join(dir, "src", projectName+".cpp")
		starterContent = `#include "` + projectName + `.h"
`
		// Also create header
		headerFile := filepath.Join(dir, "include", projectName+".h")
		if _, err := os.Stat(headerFile); os.IsNotExist(err) {
			headerGuard := strings.ToUpper(projectName) + "_H"
			hContent := "#ifndef " + headerGuard + "\n#define " + headerGuard + "\n\n// TODO: add declarations\n\n#endif // " + headerGuard + "\n"
			if err := os.WriteFile(headerFile, []byte(hContent), 0644); err != nil {
				return fmt.Errorf("failed to create header: %w", err)
			}
		}
	case "header_only":
		starterFile = filepath.Join(dir, "include", projectName+".h")
		headerGuard := strings.ToUpper(projectName) + "_H"
		starterContent = "#ifndef " + headerGuard + "\n#define " + headerGuard + "\n\n// TODO: add declarations\n\n#endif // " + headerGuard + "\n"
	}

	if starterFile != "" {
		if _, err := os.Stat(starterFile); os.IsNotExist(err) {
			if err := os.WriteFile(starterFile, []byte(starterContent), 0644); err != nil {
				return fmt.Errorf("failed to create starter file: %w", err)
			}
		}
	}

	// Create starter test file
	testFile := filepath.Join(dir, "tests", "test.cpp")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		testContent := `#include <iostream>

int main() {
    std::cout << "Running tests for ` + projectName + `..." << std::endl;
    // TODO: Add your assertions here
    return 0; // Success
}
`
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			return fmt.Errorf("failed to create test file: %w", err)
		}
	}

	// Write project.yaml
	if err := parser.SaveProject(dir, cfg); err != nil {
		return err
	}

	// Append to .gitignore
	appendGitignore(dir)

	fmt.Printf("Initialized aloy project: %s (type: %s)\n", projectName, targetType)
	fmt.Println("  project.yaml created")
	return nil
}

func appendGitignore(dir string) {
	path := filepath.Join(dir, ".gitignore")
	entries := "\n# aloy\n.my_modules/\nbuild/\n"

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not update .gitignore: %v\n", err)
		return
	}
	defer f.Close()
	if _, err := f.WriteString(entries); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not write to .gitignore: %v\n", err)
	}
}
