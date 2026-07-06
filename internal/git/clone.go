package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CloneFull performs a full (non-shallow) clone to allow tag enumeration.
func CloneFull(url, dest string) error {
	if _, err := os.Stat(dest); err == nil {
		return nil
	}
	cmd := exec.Command("git", "clone", "--recurse-submodules", url, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone %s: %w", url, err)
	}
	return nil
}

// FetchTags fetches all tags for a repository.
func FetchTags(repoPath string) error {
	cmd := exec.Command("git", "fetch", "--tags")
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git fetch --tags in %s: %w", repoPath, err)
	}
	return nil
}

// Checkout performs a detached HEAD checkout to the given ref (tag, SHA, branch).
func Checkout(repoPath, ref string) error {
	cmd := exec.Command("git", "checkout", "--detach", ref)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		cmdOrigin := exec.Command("git", "checkout", "--detach", "origin/"+ref)
		cmdOrigin.Dir = repoPath
		cmdOrigin.Stdout = os.Stdout
		cmdOrigin.Stderr = os.Stderr
		if err2 := cmdOrigin.Run(); err2 != nil {
			return fmt.Errorf("git checkout %s (and origin/%s) failed in %s: %w (origin err: %v)", ref, ref, repoPath, err, err2)
		}
	}
	return nil
}

// GetHeadSHA returns the current HEAD commit SHA.
func GetHeadSHA(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD in %s: %w", repoPath, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// DefaultBranch returns the default branch name (main or master).
func DefaultBranch(repoPath string) (string, error) {
	for _, branch := range []string{"main", "master"} {
		cmd := exec.Command("git", "rev-parse", "--verify", branch)
		cmd.Dir = repoPath
		if err := cmd.Run(); err == nil {
			return branch, nil
		}
	}
	return "", fmt.Errorf("no default branch (main/master) found in %s", repoPath)
}
