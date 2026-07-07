package resolver

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/aloy-io/aloy/internal/git"
	"github.com/aloy-io/aloy/internal/model"
	"github.com/aloy-io/aloy/internal/parser"
)

const ModulesDir = ".my_modules"

var cloneMu sync.Mutex

// ResolvedDep holds the final resolved information for a dependency.
type ResolvedDep struct {
	Name            string
	GitURL          string
	ResolvedVersion string
	CommitSHA       string
	LogicalName     string // used as the unique ID for the dependency in aloy (alias if present, else name)
	RepoDir         string // directory where the repo is cloned (e.g. hash-version)
	Subdir          string // subdirectory inside the repo
	CMakeTarget     string // detected or overridden CMake project name
	IsAloyPackage   bool   // has project.yaml
	IsSystem        bool   // type: system
	Type            string // "aloy", "cmake", "meson", "system", "header_only", "custom"
	BuildCommand    string
	IncludeDirs     []string
	LibDirs         []string
}

// resolveSingle resolves version from cache, clones referencing cache, and returns result.
func resolveSingle(dep model.Dependency, modulesBase, cachePath string) (*ResolvedDep, error) {
	logicalName := dep.ModuleDir()

	// 1. Resolve version using cachePath (bare repo)
	var resolvedVersion string

	if dep.Version != "" {
		tag, ver, err := git.FindBestTag(cachePath, dep.Version)
		if err != nil {
			branch, brErr := git.DefaultBranch(cachePath)
			if brErr != nil {
				return nil, fmt.Errorf("no matching version for %s (%s) and no default branch: %w", dep.Name, dep.Version, err)
			}
			fmt.Printf("  Warning: no semver tags for %s, using branch %s\n", dep.Name, branch)
			resolvedVersion = branch
		} else {
			// Used to be ver.String(), but we need the exact tag string for checkout
			resolvedVersion = tag
			_ = ver // silences unused var
		}
	} else {
		branch, err := git.DefaultBranch(cachePath)
		if err != nil {
			return nil, fmt.Errorf("no default branch for %s: %w", dep.Name, err)
		}
		resolvedVersion = branch
	}

	cacheBase := filepath.Base(cachePath)
	cacheBase = strings.TrimSuffix(cacheBase, ".git")
	safeVersion := strings.ReplaceAll(resolvedVersion, "/", "_")
	repoDir := fmt.Sprintf("%s-%s", cacheBase, safeVersion)

	destPath := filepath.Join(modulesBase, repoDir)

	fmt.Printf("  Resolving %s (version: %s)...\n", dep.Name, resolvedVersion)

	cloneMu.Lock()
	if err := git.CloneFromCache(cachePath, dep.Git, destPath); err != nil {
		cloneMu.Unlock()
		return nil, fmt.Errorf("failed to clone %s: %w", dep.Name, err)
	}
	if err := git.Checkout(destPath, resolvedVersion); err != nil {
		cloneMu.Unlock()
		return nil, err
	}
	cloneMu.Unlock()

	sha, err := git.GetHeadSHA(destPath)
	if err != nil {
		return nil, err
	}

	isAloy := false
	resolvedType := ""

	if dep.Type != "" {
		resolvedType = dep.Type
		if resolvedType == "aloy" {
			isAloy = true
		}
	} else {
		projectYamlPath := filepath.Join(destPath, dep.Subdir, parser.ProjectFileName)
		mesonBuildPath := filepath.Join(destPath, dep.Subdir, "meson.build")
		cmakeListsPath := filepath.Join(destPath, dep.Subdir, "CMakeLists.txt")

		if _, err := os.Stat(projectYamlPath); err == nil {
			isAloy = true
			resolvedType = "aloy"
		} else if _, err := os.Stat(mesonBuildPath); err == nil {
			resolvedType = "meson"
		} else if _, err := os.Stat(cmakeListsPath); err == nil {
			resolvedType = "cmake"
		} else {
			resolvedType = "header_only"
		}
	}

	cmakeTarget := dep.Name
	if dep.CMakeTarget != "" {
		cmakeTarget = dep.CMakeTarget
	}

	return &ResolvedDep{
		Name:            dep.Name,
		GitURL:          dep.Git,
		ResolvedVersion: resolvedVersion,
		CommitSHA:       sha,
		LogicalName:     logicalName,
		RepoDir:         repoDir,
		Subdir:          dep.Subdir,
		CMakeTarget:     cmakeTarget,
		IsAloyPackage:   isAloy,
		IsSystem:        false,
		Type:            resolvedType,
		BuildCommand:    dep.BuildCommand,
		IncludeDirs:     dep.IncludeDirs,
		LibDirs:         dep.LibDirs,
	}, nil
}

// ResolveGraph performs BFS dependency resolution starting from the root project.
func ResolveGraph(projectRoot string, cfg *model.ProjectConfig) ([]ResolvedDep, error) {
	modulesBase := filepath.Join(projectRoot, ModulesDir)
	if err := os.MkdirAll(modulesBase, 0755); err != nil {
		return nil, fmt.Errorf("failed to create %s: %w", modulesBase, err)
	}

	type queueItem struct {
		dep     model.Dependency
		fromPkg string
	}
	var queue []queueItem
	for _, target := range cfg.Targets {
		for _, d := range target.Dependencies {
			queue = append(queue, queueItem{dep: d, fromPkg: "root"})
		}
	}

	resolved := make(map[string]*ResolvedDep)

	for len(queue) > 0 {
		currentLevel := queue
		queue = nil

		type fetchJob struct {
			item        queueItem
			logicalName string
		}
		var toFetch []fetchJob
		seen := make(map[string]bool)

		uniqueGitURLs := make(map[string]bool)

		for _, item := range currentLevel {
			dep := item.dep
			logicalName := dep.ModuleDir()

			if dep.Type == "system" {
				if _, exists := resolved[logicalName]; !exists {
					resolved[logicalName] = &ResolvedDep{
						Name:        dep.Name,
						LogicalName: logicalName,
						IsSystem:    true,
						Type:        "system",
					}
				}
				continue
			}

			if dep.Git == "" {
				return nil, fmt.Errorf("dependency %q from %s: git URL is required for non-system packages", dep.Name, item.fromPkg)
			}

			if existing, exists := resolved[logicalName]; exists {
				if dep.Version != "" && existing.ResolvedVersion != "" {
					if IsMajorConflict(dep.Version, existing.ResolvedVersion) {
						return nil, fmt.Errorf(
							"major version conflict for %q: %s (from %s) vs %s (previously resolved)\n"+
								"  Hint: use 'alias' field to import both versions under different names",
							dep.Name, dep.Version, item.fromPkg, existing.ResolvedVersion,
						)
					}
				}
				continue
			}

			if seen[logicalName] {
				continue
			}
			seen[logicalName] = true

			uniqueGitURLs[dep.Git] = true
			toFetch = append(toFetch, fetchJob{item: item, logicalName: logicalName})
		}

		if len(toFetch) == 0 {
			continue
		}

		// 1. Fetch caches concurrently
		type cacheResult struct {
			url       string
			cachePath string
			err       error
		}
		cacheResults := make(chan cacheResult, len(uniqueGitURLs))

		for url := range uniqueGitURLs {
			go func(u string) {
				cp, err := git.FetchCache(u)
				cacheResults <- cacheResult{u, cp, err}
			}(url)
		}

		cachePaths := make(map[string]string)
		for i := 0; i < len(uniqueGitURLs); i++ {
			res := <-cacheResults
			if res.err != nil {
				return nil, res.err
			}
			cachePaths[res.url] = res.cachePath
		}

		// 2. Resolve uniquely and parallel
		type fetchResult struct {
			rd  *ResolvedDep
			dep model.Dependency
			err error
		}
		results := make([]fetchResult, len(toFetch))
		var wg sync.WaitGroup
		wg.Add(len(toFetch))

		for i, job := range toFetch {
			go func(idx int, j fetchJob) {
				defer wg.Done()
				cp := cachePaths[j.item.dep.Git]
				rd, err := resolveSingle(j.item.dep, modulesBase, cp)
				results[idx] = fetchResult{rd: rd, dep: j.item.dep, err: err}
			}(i, job)
		}
		wg.Wait()

		// 3. Collect results
		for _, r := range results {
			if r.err != nil {
				return nil, r.err
			}
			resolved[r.rd.LogicalName] = r.rd

			if r.rd.IsAloyPackage {
				destPath := filepath.Join(modulesBase, r.rd.RepoDir, r.rd.Subdir)
				subCfg, err := parser.LoadProject(destPath)
				if err != nil {
					return nil, fmt.Errorf("failed to parse %s/project.yaml: %w", r.dep.Name, err)
				}
				for _, target := range subCfg.Targets {
					for _, subDep := range target.Dependencies {
						queue = append(queue, queueItem{dep: subDep, fromPkg: r.dep.Name})
					}
				}
			}
		}
	}

	var result []ResolvedDep
	for _, rd := range resolved {
		result = append(result, *rd)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

// BuildLockFile creates a LockFile from resolved dependencies.
func BuildLockFile(deps []ResolvedDep) *model.LockFile {
	lf := &model.LockFile{Version: 1}
	for _, d := range deps {
		lf.Packages = append(lf.Packages, model.LockedPackage{
			Name:            d.Name,
			LogicalName:     d.LogicalName,
			RepoDir:         d.RepoDir,
			Subdir:          d.Subdir,
			GitURL:          d.GitURL,
			ResolvedVersion: d.ResolvedVersion,
			CommitSHA:       d.CommitSHA,
			CMakeTarget:     d.CMakeTarget,
			IsAloyPackage:   d.IsAloyPackage,
			IsSystem:        d.IsSystem,
			Type:            d.Type,
			BuildCommand:    d.BuildCommand,
			IncludeDirs:     d.IncludeDirs,
			LibDirs:         d.LibDirs,
		})
	}
	return lf
}
