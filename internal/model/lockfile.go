package model

// LockFile represents the aloy.lock structure for reproducible builds.
type LockFile struct {
	Version  int             `yaml:"version"`
	Packages []LockedPackage `yaml:"packages,omitempty"`
}

// LockedPackage records a resolved dependency at a specific commit.
type LockedPackage struct {
	Name            string `yaml:"name"`
	LogicalName     string `yaml:"logical_name,omitempty"`
	RepoDir         string `yaml:"repo_dir,omitempty"`
	Subdir          string `yaml:"subdir,omitempty"`
	GitURL          string `yaml:"git_url"`
	ResolvedVersion string `yaml:"resolved_version"`
	CommitSHA       string `yaml:"commit_sha"`
	CMakeTarget     string `yaml:"cmake_target,omitempty"`
	IsAloyPackage   bool   `yaml:"is_aloy_package,omitempty"`
	IsSystem        bool   `yaml:"is_system,omitempty"`
	Type            string `yaml:"type,omitempty"`
}

// FindPackage returns the locked package by name, or nil if not found.
func (lf *LockFile) FindPackage(name string) *LockedPackage {
	for i := range lf.Packages {
		if lf.Packages[i].Name == name {
			return &lf.Packages[i]
		}
	}
	return nil
}
