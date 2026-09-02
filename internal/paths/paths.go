package paths

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func ResolveUnderRoot(root, relativePath string) (string, error) {
	if relativePath == "" || filepath.IsAbs(relativePath) || filepath.VolumeName(relativePath) != "" {
		return "", errors.New("path must be relative")
	}
	cleanPath := filepath.Clean(relativePath)
	if cleanPath == "." || strings.HasPrefix(cleanPath, "..") {
		return "", errors.New("path escapes the root")
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	candidate := filepath.Join(resolvedRoot, cleanPath)
	resolvedCandidate, err := resolveExistingParent(candidate)
	if err != nil {
		return "", err
	}
	relativeCandidate, err := filepath.Rel(resolvedRoot, resolvedCandidate)
	if err != nil || relativeCandidate == ".." || strings.HasPrefix(relativeCandidate, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes the root")
	}
	return candidate, nil
}

func resolveExistingParent(path string) (string, error) {
	current := path
	for {
		_, err := os.Lstat(current)
		if err == nil {
			return filepath.EvalSymlinks(current)
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", err
		}
		current = parent
	}
}
