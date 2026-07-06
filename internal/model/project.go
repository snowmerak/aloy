package model

// ProjectConfig represents the root project.yaml structure.
type ProjectConfig struct {
	Project     ProjectMeta       `yaml:"project"`
	Targets     map[string]Target `yaml:"targets,omitempty"`
	InjectCMake string            `yaml:"inject_cmake,omitempty"`
}

// ProjectMeta holds the top-level project metadata.
type ProjectMeta struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	CXXStandard int    `yaml:"cxx_standard"`
}

// Target defines an executable or library build target.
type Target struct {
	Type         string                    `yaml:"type"` // executable, library, shared_library, header_only, test
	Sources      []string                  `yaml:"sources,omitempty"`
	Includes     IncludeConfig             `yaml:"includes,omitempty"`
	Platforms    map[string]PlatformConfig `yaml:"platforms,omitempty"`
	Dependencies []Dependency              `yaml:"dependencies,omitempty"`
	Pch          string                    `yaml:"pch,omitempty"`
}

// IncludeConfig separates public and private include directories.
type IncludeConfig struct {
	Public  []string `yaml:"public,omitempty"`
	Private []string `yaml:"private,omitempty"`
}

// PlatformConfig holds platform-specific build settings.
type PlatformConfig struct {
	Sources       []string `yaml:"sources,omitempty"`
	CompilerFlags []string `yaml:"compiler_flags,omitempty"`
	LinkerFlags   []string `yaml:"linker_flags,omitempty"`
}

// Dependency describes a single package dependency.
type Dependency struct {
	Name         string            `yaml:"name"`
	Git          string            `yaml:"git,omitempty"`
	Version      string            `yaml:"version,omitempty"`
	Type         string            `yaml:"type,omitempty"` // "" (git), "system", "cmake", "meson", "aloy"
	Alias        string            `yaml:"alias,omitempty"`
	Subdir       string            `yaml:"subdir,omitempty"`
	CMakeTarget  string            `yaml:"cmake_target,omitempty"`
	CMakeOptions map[string]string `yaml:"cmake_options,omitempty"`
}

// TargetName returns the alias if set, otherwise the name.
func (d Dependency) TargetName() string {
	if d.Alias != "" {
		return d.Alias
	}
	return d.Name
}

// ModuleDir returns the directory name under .my_modules/.
func (d Dependency) ModuleDir() string {
	if d.Alias != "" {
		return d.Alias
	}
	return d.Name
}
