package parser

import (
	"fmt"
	"os"

	"github.com/aloy-io/aloy/internal/model"
	"gopkg.in/yaml.v3"
)

const LockFileName = "aloy.lock"

// LoadLockFile reads and parses an aloy.lock from the given directory.
func LoadLockFile(dir string) (*model.LockFile, error) {
	path := dir + "/" + LockFileName
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &model.LockFile{Version: 1}, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	var lf model.LockFile
	if err := yaml.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return &lf, nil
}

// SaveLockFile writes a LockFile to aloy.lock in the given directory.
func SaveLockFile(dir string, lf *model.LockFile) error {
	path := dir + "/" + LockFileName
	data, err := yaml.Marshal(lf)
	if err != nil {
		return fmt.Errorf("failed to marshal lock file: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}
