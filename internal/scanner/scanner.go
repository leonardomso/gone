// Package scanner finds markdown files in a directory
package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

// FindMarkdownFiles walks a directory and returns all .md file paths
// It skips hidden directories (starting with .) like .git.
func FindMarkdownFiles(root string) ([]string, error) {
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

		// Check if this is a markdown file
		// strings.HasSuffix checks if a string ends with a given suffix
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			files = append(files, path)
		}

		return nil
	})

	// In Go, we always return the error to let the caller decide how to handle it
	if err != nil {
		return nil, err
	}

	return files, nil
}
