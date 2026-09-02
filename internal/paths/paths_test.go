package paths

import (
	"path/filepath"
	"testing"
)

func TestResolveUnderRootRejectsEscapingPaths(t *testing.T) {
	root := t.TempDir()
	_, err := ResolveUnderRoot(root, "../outside.wav")
	if err == nil {
		t.Fatal("ResolveUnderRoot accepted a traversal path")
	}
}

func TestResolveUnderRootReturnsContainedPath(t *testing.T) {
	root := t.TempDir()
	path, err := ResolveUnderRoot(root, filepath.Join("recordings", "set.wav"))
	if err != nil {
		t.Fatalf("ResolveUnderRoot returned an error: %v", err)
	}
	relativePath, err := filepath.Rel(root, path)
	if err != nil {
		t.Fatalf("Rel returned an error: %v", err)
	}
	if relativePath != filepath.Join("recordings", "set.wav") {
		t.Fatalf("ResolveUnderRoot returned %q", relativePath)
	}
}
