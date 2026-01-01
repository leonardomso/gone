// Package fixer provides functionality to automatically fix redirect URLs in markdown files.
package fixer

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/leonardomso/gone/internal/checker"
	"github.com/leonardomso/gone/internal/parser"
)

// Fix represents a single URL replacement to be made.
type Fix struct {
	FilePath    string // File containing the URL
	OldURL      string // Original URL (redirect source)
	NewURL      string // Final URL (redirect destination)
	RefName     string // Reference name if IsRefDef (e.g., "myref" in [myref]: url)
	Line        int    // Line number where the URL appears
	Occurrences int    // How many times this exact URL appears in the file
	LinkType    parser.LinkType
	RefUsages   int  // How many places use this reference
	IsRefDef    bool // Is this a reference definition line?
}

// FileChanges groups all fixes for a single file.
type FileChanges struct {
	FilePath   string
	Fixes      []Fix
	TotalFixes int // Total number of replacements (accounting for occurrences)
}

// FixResult represents the outcome of applying fixes to a file.
type FixResult struct {
	Error       error
	FilePath    string
	ChangedURLs []URLChange
	Applied     int
	Skipped     int
}

// URLChange represents a single URL that was changed.
type URLChange struct {
	OldURL string
	NewURL string
	Line   int
}

// Fixer handles URL replacement in markdown files.
type Fixer struct {
	// Track parser links for reference info
	parserLinks []parser.Link
}

// New creates a new Fixer instance.
func New() *Fixer {
	return &Fixer{}
}

// SetParserLinks provides the original parser links for reference detection.
func (f *Fixer) SetParserLinks(links []parser.Link) {
	f.parserLinks = links
}

// FindFixes analyzes check results and returns fixable items grouped by file.
// Only redirects with a successful final destination (200) are considered fixable.
func (f *Fixer) FindFixes(results []checker.Result) []FileChanges {
	fileFixMap := map[string]map[string]*Fix{}
	urlToParserLink := f.buildURLToLinksMap()

	for _, r := range results {
		if !isFixableRedirect(r) {
			continue
		}

		f.addOrUpdateFix(fileFixMap, r, urlToParserLink)
	}

	return f.buildFileChanges(fileFixMap)
}

// buildURLToLinksMap creates a map from URL to parser links for quick lookup.
func (f *Fixer) buildURLToLinksMap() map[string][]parser.Link {
	urlToLinks := make(map[string][]parser.Link, len(f.parserLinks))
	for _, pl := range f.parserLinks {
		urlToLinks[pl.URL] = append(urlToLinks[pl.URL], pl)
	}
	return urlToLinks
}

// isFixableRedirect checks if a result is a fixable redirect.
func isFixableRedirect(r checker.Result) bool {
	return r.Status == checker.StatusRedirect &&
		r.FinalStatus == 200 &&
		r.FinalURL != "" &&
		r.FinalURL != r.Link.URL
}

// addOrUpdateFix adds a new fix or increments occurrence count for existing fix.
func (f *Fixer) addOrUpdateFix(
	fileFixMap map[string]map[string]*Fix,
	r checker.Result,
	urlToParserLink map[string][]parser.Link,
) {
	filePath := r.Link.FilePath
	oldURL := r.Link.URL

	if fileFixMap[filePath] == nil {
		fileFixMap[filePath] = map[string]*Fix{}
	}

	if existing, ok := fileFixMap[filePath][oldURL]; ok {
		existing.Occurrences++
		return
	}

	fix := f.createFix(r, urlToParserLink)
	fileFixMap[filePath][oldURL] = fix
}

// createFix creates a Fix from a checker result.
func (f *Fixer) createFix(r checker.Result, urlToParserLink map[string][]parser.Link) *Fix {
	fix := &Fix{
		FilePath:    r.Link.FilePath,
		Line:        r.Link.Line,
		OldURL:      r.Link.URL,
		NewURL:      r.FinalURL,
		Occurrences: 1,
		LinkType:    parser.LinkTypeInline,
	}

	if pLinks, ok := urlToParserLink[r.Link.URL]; ok {
		f.applyRefInfo(fix, pLinks)
	}

	return fix
}

// applyRefInfo applies reference definition info to a fix.
func (*Fixer) applyRefInfo(fix *Fix, pLinks []parser.Link) {
	// First pass: find link type and ref name for this file
	for _, pl := range pLinks {
		if pl.FilePath != fix.FilePath {
			continue
		}
		fix.LinkType = pl.Type
		if pl.Type == parser.LinkTypeReference && pl.RefDefLine > 0 {
			fix.RefName = pl.RefName
		}
	}

	// Second pass: find the reference definition line
	for _, pl := range pLinks {
		if pl.FilePath != fix.FilePath || pl.RefDefLine <= 0 {
			continue
		}
		fix.Line = pl.RefDefLine
		fix.IsRefDef = true
		fix.RefName = pl.RefName
		fix.RefUsages = countRefUsages(pLinks, fix.FilePath, pl.RefName)
		break
	}
}

// countRefUsages counts how many times a reference is used in a file.
func countRefUsages(links []parser.Link, filePath, refName string) int {
	count := 0
	for _, pl := range links {
		if pl.FilePath == filePath && pl.RefName == refName {
			count++
		}
	}
	return count
}

// buildFileChanges converts the internal map to a sorted slice of FileChanges.
func (*Fixer) buildFileChanges(fileFixMap map[string]map[string]*Fix) []FileChanges {
	result := make([]FileChanges, 0, len(fileFixMap))

	// Get sorted file paths
	filePaths := make([]string, 0, len(fileFixMap))
	for fp := range fileFixMap {
		filePaths = append(filePaths, fp)
	}
	sort.Strings(filePaths)

	for _, filePath := range filePaths {
		fixes := fileFixMap[filePath]

		// Convert map to slice and sort by line number
		fixSlice := make([]Fix, 0, len(fixes))
		totalFixes := 0
		for _, fix := range fixes {
			fixSlice = append(fixSlice, *fix)
			totalFixes += fix.Occurrences
		}

		sort.Slice(fixSlice, func(i, j int) bool {
			return fixSlice[i].Line < fixSlice[j].Line
		})

		result = append(result, FileChanges{
			FilePath:   filePath,
			Fixes:      fixSlice,
			TotalFixes: totalFixes,
		})
	}

	return result
}

// Preview returns a formatted string showing what changes would be made.
func (*Fixer) Preview(changes []FileChanges) string {
	if len(changes) == 0 {
		return "No fixable redirects found."
	}

	var b strings.Builder
	totalFixes := 0
	for _, fc := range changes {
		totalFixes += fc.TotalFixes
	}

	b.WriteString(fmt.Sprintf("Found %d fixable redirect(s) across %d file(s):\n\n",
		totalFixes, len(changes)))

	for _, fc := range changes {
		b.WriteString(fmt.Sprintf("%s (%d fix(es))\n", fc.FilePath, fc.TotalFixes))

		for _, fix := range fc.Fixes {
			lineInfo := fmt.Sprintf("  Line %d: ", fix.Line)

			if fix.IsRefDef {
				b.WriteString(fmt.Sprintf("%s[%s] %s\n", lineInfo, fix.RefName, fix.OldURL))
				b.WriteString(fmt.Sprintf("          -> %s", fix.NewURL))
				if fix.RefUsages > 0 {
					b.WriteString(fmt.Sprintf(" (used %d time(s))", fix.RefUsages))
				}
				b.WriteString("\n")
			} else {
				b.WriteString(fmt.Sprintf("%s%s\n", lineInfo, truncateURL(fix.OldURL, 60)))
				b.WriteString(fmt.Sprintf("          -> %s", truncateURL(fix.NewURL, 60)))
				if fix.Occurrences > 1 {
					b.WriteString(fmt.Sprintf(" (%d occurrence(s))", fix.Occurrences))
				}
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

// truncateURL shortens a URL for display.
func truncateURL(url string, maxLen int) string {
	if len(url) <= maxLen {
		return url
	}
	return url[:maxLen-3] + "..."
}

// ApplyToFile applies all fixes to a single file.
func (*Fixer) ApplyToFile(fc FileChanges) (*FixResult, error) {
	result := &FixResult{
		FilePath:    fc.FilePath,
		ChangedURLs: []URLChange{},
	}

	// Read file content
	content, err := os.ReadFile(fc.FilePath)
	if err != nil {
		result.Error = fmt.Errorf("reading file: %w", err)
		return result, result.Error
	}

	originalContent := string(content)
	modifiedContent := originalContent

	// Apply each fix
	// We need to be careful about overlapping replacements
	// Process fixes in reverse line order to avoid offset issues...
	// Actually, since we're doing string replacement on full URLs, order shouldn't matter
	// as long as URLs are unique

	for _, fix := range fc.Fixes {
		// Count occurrences before replacement
		countBefore := strings.Count(modifiedContent, fix.OldURL)

		if countBefore == 0 {
			result.Skipped++
			continue
		}

		// Replace all occurrences of the old URL with the new URL
		modifiedContent = strings.ReplaceAll(modifiedContent, fix.OldURL, fix.NewURL)

		// Verify replacement worked
		countAfter := strings.Count(modifiedContent, fix.OldURL)
		replaced := countBefore - countAfter

		if replaced > 0 {
			result.Applied += replaced
			result.ChangedURLs = append(result.ChangedURLs, URLChange{
				Line:   fix.Line,
				OldURL: fix.OldURL,
				NewURL: fix.NewURL,
			})
		}
	}

	// Only write if content changed
	if modifiedContent == originalContent {
		return result, nil
	}

	// Write modified content back to file
	err = os.WriteFile(fc.FilePath, []byte(modifiedContent), 0o600)
	if err != nil {
		result.Error = fmt.Errorf("writing file: %w", err)
		return result, result.Error
	}

	return result, nil
}

// ApplyAll applies fixes to all files and returns results.
func (f *Fixer) ApplyAll(changes []FileChanges) []FixResult {
	results := make([]FixResult, 0, len(changes))

	for _, fc := range changes {
		result, _ := f.ApplyToFile(fc)
		results = append(results, *result)
	}

	return results
}

// Summary returns a formatted summary of fix results.
func Summary(results []FixResult) string {
	var b strings.Builder

	totalApplied := 0
	totalSkipped := 0
	filesModified := 0
	var errors []string

	for _, r := range results {
		totalApplied += r.Applied
		totalSkipped += r.Skipped
		if r.Applied > 0 {
			filesModified++
		}
		if r.Error != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", r.FilePath, r.Error))
		}
	}

	if totalApplied == 0 && len(errors) == 0 {
		return "No changes made."
	}

	b.WriteString(fmt.Sprintf("Fixed %d redirect(s) across %d file(s).\n", totalApplied, filesModified))

	if totalSkipped > 0 {
		b.WriteString(fmt.Sprintf("Skipped %d (URL not found in file).\n", totalSkipped))
	}

	if len(errors) > 0 {
		b.WriteString("\nErrors:\n")
		for _, e := range errors {
			b.WriteString(fmt.Sprintf("  %s\n", e))
		}
	}

	return b.String()
}

// DetailedSummary returns a detailed summary showing each change.
func DetailedSummary(results []FixResult) string {
	var b strings.Builder

	totalApplied := 0
	filesModified := 0

	for _, r := range results {
		if r.Applied > 0 {
			totalApplied += r.Applied
			filesModified++
		}
	}

	if totalApplied == 0 {
		return "No changes made."
	}

	b.WriteString(fmt.Sprintf("Fixed %d redirect(s) across %d file(s):\n\n", totalApplied, filesModified))

	for _, r := range results {
		if r.Applied == 0 {
			continue
		}

		for _, change := range r.ChangedURLs {
			b.WriteString(fmt.Sprintf("  %s:%d\n", r.FilePath, change.Line))
			b.WriteString(fmt.Sprintf("    %s\n", truncateURL(change.OldURL, 70)))
			b.WriteString(fmt.Sprintf("    -> %s\n", truncateURL(change.NewURL, 70)))
		}
	}

	return b.String()
}
