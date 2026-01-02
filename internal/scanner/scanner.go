// Package scanner finds files in a directory based on their extensions.
package scanner

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gobwas/glob"
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
		// Handle special cases for types with multiple extensions
		switch strings.ToLower(t) {
		case "yaml":
			// yaml type should match both .yaml and .yml
			extensions[i] = ".yaml"
		case "md":
			// md type should match .md, .mdx, and .markdown
			extensions[i] = ".md"
		default:
			extensions[i] = "." + strings.ToLower(t)
		}
	}

	// Add additional extensions for types that have multiple file extensions
	if slices.Contains(types, "yaml") {
		extensions = append(extensions, ".yml")
	}
	if slices.Contains(types, "md") {
		extensions = append(extensions, ".mdx", ".markdown")
	}

	return FindFiles(root, extensions)
}

// ScanOptions holds options for scanning files with filtering.
type ScanOptions struct {
	// Root is the directory to scan.
	Root string

	// Types are the file types to include (e.g., "md", "json", "yaml").
	Types []string

	// Include patterns (glob) - if set, only matching files are included.
	Include []string

	// Exclude patterns (glob) - matching files are excluded.
	Exclude []string
}

// FindFilesWithOptions scans for files with include/exclude filtering.
// This is the recommended function for scanning with full configuration support.
func FindFilesWithOptions(opts ScanOptions) ([]string, error) {
	// Get base files by type
	files, err := FindFilesByTypes(opts.Root, opts.Types)
	if err != nil {
		return nil, err
	}

	// Apply include filter (if any patterns specified)
	if len(opts.Include) > 0 {
		files, err = filterByGlobPatterns(files, opts.Root, opts.Include, true)
		if err != nil {
			return nil, err
		}
	}

	// Apply exclude filter
	if len(opts.Exclude) > 0 {
		files, err = filterByGlobPatterns(files, opts.Root, opts.Exclude, false)
		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

// filterByGlobPatterns filters files by glob patterns.
// If include=true, keeps only files matching any pattern.
// If include=false, removes files matching any pattern.
func filterByGlobPatterns(files []string, root string, patterns []string, include bool) ([]string, error) {
	if len(patterns) == 0 {
		return files, nil
	}

	// Compile patterns
	compiled := make([]glob.Glob, 0, len(patterns))
	for _, p := range patterns {
		g, err := glob.Compile(p)
		if err != nil {
			return nil, err
		}
		compiled = append(compiled, g)
	}

	result := make([]string, 0, len(files))
	for _, f := range files {
		// Get relative path for matching (relative to root)
		relPath, err := filepath.Rel(root, f)
		if err != nil {
			relPath = f // Fall back to absolute path
		}
		// Normalize path separators for cross-platform glob matching
		relPath = filepath.ToSlash(relPath)

		matches := matchesAnyGlob(relPath, compiled)

		// For include mode: keep files that match any pattern
		// For exclude mode: keep files that don't match any pattern
		if include && matches {
			result = append(result, f)
		} else if !include && !matches {
			result = append(result, f)
		}
	}

	return result, nil
}

// matchesAnyGlob checks if a path matches any of the compiled glob patterns.
func matchesAnyGlob(path string, patterns []glob.Glob) bool {
	for _, g := range patterns {
		if g.Match(path) {
			return true
		}
	}
	return false
}
