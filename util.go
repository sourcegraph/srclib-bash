package main

import (
	"log"
	"path/filepath"
)

func relPath(base, path string) string {
	rp, err := filepath.Rel(evalSymlinks(base), evalSymlinks(path))
	if err != nil {
		log.Fatalf("Failed to make path %q relative to %q: %s", path, base, err)
	}
	return filepath.ToSlash(rp)
}

func evalSymlinks(path string) string {
	newPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return newPath
}
