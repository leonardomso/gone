// Package scanner finds files in a directory based on their extensions.
package scanner

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// FindMarkdownFiles walks a directory and returns all .md file paths
// It skips hidden directories (starting with .) like .git.
//
// Deprecated: Use FindFiles with extensions parameter instead.
func FindMarkdownFiles(root string) ([]string, error) {
	return FindFiles(root, []string{".md"})
}

// FindFiles walks a directory and returns all files matching the given extensions.
// Extensions should include the leading dot (e.g., ".md", ".json").
// It skips hidden directories (starting with .) like .git.
func FindFiles(root string, extensions []string) ([]string, error) {
	if len(extensions) == 0 {
		return nil, nil
	}

	// Normalize extensions to lowercase
	normalizedExts := make(map[string]bool, len(extensions))
	for _, ext := range extensions {
		normalizedExts[strings.ToLower(ext)] = true
	}

	var files []string

	// filepath.WalkDir traverses a directory tree
	// It calls the function we provide for each file/directory it finds
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		// If there was an error accessing this path, return it
		if err != nil {
			return err
		}

		// Skip hidden directories (like .git, .github, etc.)
		// d.IsDir() returns true if this entry is a directory
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && path != root {
			// filepath.SkipDir tells WalkDir to skip this entire directory
			return filepath.SkipDir
		}

		// Check if this file has a matching extension
		if !d.IsDir() {
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if normalizedExts[ext] {
				files = append(files, path)
			}
		}

		return nil
	})

	// In Go, we always return the error to let the caller decide how to handle it
	if err != nil {
		return nil, err
	}

	return files, nil
}

// FindFilesByTypes walks a directory and returns all files matching the given type names.
// Type names are without the leading dot (e.g., "md", "json", "yaml").
// It skips hidden directories (starting with .) like .git.
func FindFilesByTypes(root string, types []string) ([]string, error) {
	if len(types) == 0 {
		return nil, nil
	}

	// Convert type names to extensions
	extensions := make([]string, len(types))
	for i, t := range types {
		// Handle special case for yaml/yml
		if t == "yaml" {
			// yaml type should match both .yaml and .yml
			extensions[i] = ".yaml"
		} else {
			extensions[i] = "." + strings.ToLower(t)
		}
	}

	// For yaml type, we need to also include .yml
	if slices.Contains(types, "yaml") {
		extensions = append(extensions, ".yml")
	}

	return FindFiles(root, extensions)
}
