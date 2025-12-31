package fixer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leonardomso/gone/internal/checker"
	"github.com/leonardomso/gone/internal/parser"
)

// =============================================================================
// FindFixes Tests
// =============================================================================

func TestFixer_FindFixes_Empty(t *testing.T) {
	t.Parallel()

	f := New()
	changes := f.FindFixes(nil)
	assert.Empty(t, changes)

	changes = f.FindFixes([]checker.Result{})
	assert.Empty(t, changes)
}

func TestFixer_FindFixes_NoRedirects(t *testing.T) {
	t.Parallel()

	results := []checker.Result{
		{
			Link:       checker.Link{URL: "https://example.com", FilePath: "test.md", Line: 1},
			Status:     checker.StatusAlive,
			StatusCode: 200,
		},
		{
			Link:       checker.Link{URL: "https://dead.com", FilePath: "test.md", Line: 5},
			Status:     checker.StatusDead,
			StatusCode: 404,
		},
	}

	f := New()
	changes := f.FindFixes(results)
	assert.Empty(t, changes)
}

func TestFixer_FindFixes_RedirectWith200Final(t *testing.T) {
	t.Parallel()

	results := []checker.Result{
		{
			Link:       checker.Link{URL: "https://old.com", FilePath: "test.md", Line: 10},
			Status:     checker.StatusRedirect,
			StatusCode: 301,
			RedirectChain: []checker.Redirect{
				{URL: "https://old.com", StatusCode: 301},
			},
			FinalURL:    "https://new.com",
			FinalStatus: 200,
		},
	}

	f := New()
	changes := f.FindFixes(results)

	require.Len(t, changes, 1)
	assert.Equal(t, "test.md", changes[0].FilePath)
	require.Len(t, changes[0].Fixes, 1)
	assert.Equal(t, "https://old.com", changes[0].Fixes[0].OldURL)
	assert.Equal(t, "https://new.com", changes[0].Fixes[0].NewURL)
	assert.Equal(t, 10, changes[0].Fixes[0].Line)
	assert.Equal(t, 1, changes[0].Fixes[0].Occurrences)
}

func TestFixer_FindFixes_RedirectWithNon200Final(t *testing.T) {
	t.Parallel()

	results := []checker.Result{
		{
			Link:        checker.Link{URL: "https://old.com", FilePath: "test.md", Line: 10},
			Status:      checker.StatusRedirect,
			StatusCode:  301,
			FinalURL:    "https://new.com",
			FinalStatus: 403, // Not 200, shouldn't be fixable
		},
	}

	f := New()
	changes := f.FindFixes(results)
	assert.Empty(t, changes)
}

func TestFixer_FindFixes_SameURLRedirect(t *testing.T) {
	t.Parallel()

	// Edge case: final URL is the same as original (shouldn't happen but handle it)
	results := []checker.Result{
		{
			Link:        checker.Link{URL: "https://example.com", FilePath: "test.md", Line: 10},
			Status:      checker.StatusRedirect,
			StatusCode:  301,
			FinalURL:    "https://example.com", // Same URL
			FinalStatus: 200,
		},
	}

	f := New()
	changes := f.FindFixes(results)
	assert.Empty(t, changes)
}

func TestFixer_FindFixes_EmptyFinalURL(t *testing.T) {
	t.Parallel()

	results := []checker.Result{
		{
			Link:        checker.Link{URL: "https://old.com", FilePath: "test.md", Line: 10},
			Status:      checker.StatusRedirect,
			StatusCode:  301,
			FinalURL:    "", // Empty final URL
			FinalStatus: 200,
		},
	}

	f := New()
	changes := f.FindFixes(results)
	assert.Empty(t, changes)
}

func TestFixer_FindFixes_MultipleRedirects(t *testing.T) {
	t.Parallel()

	results := []checker.Result{
		{
			Link:        checker.Link{URL: "https://old1.com", FilePath: "test.md", Line: 10},
			Status:      checker.StatusRedirect,
			FinalURL:    "https://new1.com",
			FinalStatus: 200,
		},
		{
			Link:        checker.Link{URL: "https://old2.com", FilePath: "test.md", Line: 20},
			Status:      checker.StatusRedirect,
			FinalURL:    "https://new2.com",
			FinalStatus: 200,
		},
	}

	f := New()
	changes := f.FindFixes(results)

	require.Len(t, changes, 1)
	assert.Equal(t, "test.md", changes[0].FilePath)
	assert.Len(t, changes[0].Fixes, 2)
	assert.Equal(t, 2, changes[0].TotalFixes)
}

func TestFixer_FindFixes_MultipleFiles(t *testing.T) {
	t.Parallel()

	results := []checker.Result{
		{
			Link:        checker.Link{URL: "https://old.com", FilePath: "a.md", Line: 10},
			Status:      checker.StatusRedirect,
			FinalURL:    "https://new.com",
			FinalStatus: 200,
		},
		{
			Link:        checker.Link{URL: "https://old.com", FilePath: "b.md", Line: 5},
			Status:      checker.StatusRedirect,
			FinalURL:    "https://new.com",
			FinalStatus: 200,
		},
	}

	f := New()
	changes := f.FindFixes(results)

	require.Len(t, changes, 2)
	// Should be sorted by filename
	assert.Equal(t, "a.md", changes[0].FilePath)
	assert.Equal(t, "b.md", changes[1].FilePath)
}

func TestFixer_FindFixes_DuplicateURLsInSameFile(t *testing.T) {
	t.Parallel()

	results := []checker.Result{
		{
			Link:        checker.Link{URL: "https://old.com", FilePath: "test.md", Line: 10},
			Status:      checker.StatusRedirect,
			FinalURL:    "https://new.com",
			FinalStatus: 200,
		},
		{
			Link:        checker.Link{URL: "https://old.com", FilePath: "test.md", Line: 20},
			Status:      checker.StatusRedirect,
			FinalURL:    "https://new.com",
			FinalStatus: 200,
		},
	}

	f := New()
	changes := f.FindFixes(results)

	require.Len(t, changes, 1)
	require.Len(t, changes[0].Fixes, 1) // Should be consolidated into one fix
	assert.Equal(t, 2, changes[0].Fixes[0].Occurrences)
	assert.Equal(t, 2, changes[0].TotalFixes)
}

func TestFixer_FindFixes_SortedByLine(t *testing.T) {
	t.Parallel()

	results := []checker.Result{
		{
			Link:        checker.Link{URL: "https://c.com", FilePath: "test.md", Line: 30},
			Status:      checker.StatusRedirect,
			FinalURL:    "https://new-c.com",
			FinalStatus: 200,
		},
		{
			Link:        checker.Link{URL: "https://a.com", FilePath: "test.md", Line: 10},
			Status:      checker.StatusRedirect,
			FinalURL:    "https://new-a.com",
			FinalStatus: 200,
		},
		{
			Link:        checker.Link{URL: "https://b.com", FilePath: "test.md", Line: 20},
			Status:      checker.StatusRedirect,
			FinalURL:    "https://new-b.com",
			FinalStatus: 200,
		},
	}

	f := New()
	changes := f.FindFixes(results)

	require.Len(t, changes[0].Fixes, 3)
	assert.Equal(t, 10, changes[0].Fixes[0].Line)
	assert.Equal(t, 20, changes[0].Fixes[1].Line)
	assert.Equal(t, 30, changes[0].Fixes[2].Line)
}

func TestFixer_FindFixes_WithParserLinks_Reference(t *testing.T) {
	t.Parallel()

	parserLinks := []parser.Link{
		{
			URL:        "https://old.com",
			FilePath:   "test.md",
			Line:       10,
			Type:       parser.LinkTypeReference,
			RefName:    "myref",
			RefDefLine: 100,
		},
	}

	results := []checker.Result{
		{
			Link:        checker.Link{URL: "https://old.com", FilePath: "test.md", Line: 10},
			Status:      checker.StatusRedirect,
			FinalURL:    "https://new.com",
			FinalStatus: 200,
		},
	}

	f := New()
	f.SetParserLinks(parserLinks)
	changes := f.FindFixes(results)

	require.Len(t, changes, 1)
	require.Len(t, changes[0].Fixes, 1)

	fix := changes[0].Fixes[0]
	assert.True(t, fix.IsRefDef)
	assert.Equal(t, "myref", fix.RefName)
	assert.Equal(t, 100, fix.Line) // Should use RefDefLine
}

// =============================================================================
// Preview Tests
// =============================================================================

func TestFixer_Preview_Empty(t *testing.T) {
	t.Parallel()

	f := New()
	preview := f.Preview(nil)
	assert.Equal(t, "No fixable redirects found.", preview)

	preview = f.Preview([]FileChanges{})
	assert.Equal(t, "No fixable redirects found.", preview)
}

func TestFixer_Preview_SingleFix(t *testing.T) {
	t.Parallel()

	changes := []FileChanges{
		{
			FilePath:   "test.md",
			TotalFixes: 1,
			Fixes: []Fix{
				{
					FilePath:    "test.md",
					Line:        10,
					OldURL:      "https://old.com",
					NewURL:      "https://new.com",
					Occurrences: 1,
				},
			},
		},
	}

	f := New()
	preview := f.Preview(changes)

	assert.Contains(t, preview, "Found 1 fixable redirect(s) across 1 file(s)")
	assert.Contains(t, preview, "test.md (1 fix(es))")
	assert.Contains(t, preview, "Line 10")
	assert.Contains(t, preview, "https://old.com")
	assert.Contains(t, preview, "https://new.com")
}

func TestFixer_Preview_MultipleOccurrences(t *testing.T) {
	t.Parallel()

	changes := []FileChanges{
		{
			FilePath:   "test.md",
			TotalFixes: 3,
			Fixes: []Fix{
				{
					FilePath:    "test.md",
					Line:        10,
					OldURL:      "https://old.com",
					NewURL:      "https://new.com",
					Occurrences: 3,
				},
			},
		},
	}

	f := New()
	preview := f.Preview(changes)

	assert.Contains(t, preview, "(3 occurrence(s))")
}

func TestFixer_Preview_ReferenceFix(t *testing.T) {
	t.Parallel()

	changes := []FileChanges{
		{
			FilePath:   "test.md",
			TotalFixes: 1,
			Fixes: []Fix{
				{
					FilePath:  "test.md",
					Line:      100,
					OldURL:    "https://old.com",
					NewURL:    "https://new.com",
					IsRefDef:  true,
					RefName:   "myref",
					RefUsages: 5,
				},
			},
		},
	}

	f := New()
	preview := f.Preview(changes)

	assert.Contains(t, preview, "[myref]")
	assert.Contains(t, preview, "(used 5 time(s))")
}

func TestFixer_Preview_LongURL(t *testing.T) {
	t.Parallel()

	longURL := "https://example.com/" + string(make([]byte, 100)) // Very long URL
	changes := []FileChanges{
		{
			FilePath:   "test.md",
			TotalFixes: 1,
			Fixes: []Fix{
				{
					FilePath: "test.md",
					Line:     10,
					OldURL:   longURL,
					NewURL:   "https://new.com",
				},
			},
		},
	}

	f := New()
	preview := f.Preview(changes)

	// URL should be truncated
	assert.Contains(t, preview, "...")
}

// =============================================================================
// ApplyToFile Tests
// =============================================================================

func TestFixer_ApplyToFile_Success(t *testing.T) {
	t.Parallel()

	// Create temp file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")

	content := `# Test
Check out [this link](https://old.com) for more info.
Also see [another](https://keep.com).
`
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o600))

	changes := FileChanges{
		FilePath:   filePath,
		TotalFixes: 1,
		Fixes: []Fix{
			{
				FilePath: filePath,
				Line:     2,
				OldURL:   "https://old.com",
				NewURL:   "https://new.com",
			},
		},
	}

	f := New()
	result, err := f.ApplyToFile(changes)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Applied)
	assert.Equal(t, 0, result.Skipped)
	assert.Len(t, result.ChangedURLs, 1)

	// Verify file content
	newContent, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Contains(t, string(newContent), "https://new.com")
	assert.NotContains(t, string(newContent), "https://old.com")
	assert.Contains(t, string(newContent), "https://keep.com") // Unchanged
}

func TestFixer_ApplyToFile_MultipleReplacements(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")

	content := `# Test
First: https://old.com
Second: https://old.com
Third: https://old.com
`
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o600))

	changes := FileChanges{
		FilePath:   filePath,
		TotalFixes: 3,
		Fixes: []Fix{
			{
				FilePath:    filePath,
				OldURL:      "https://old.com",
				NewURL:      "https://new.com",
				Occurrences: 3,
			},
		},
	}

	f := New()
	result, err := f.ApplyToFile(changes)

	require.NoError(t, err)
	assert.Equal(t, 3, result.Applied)

	// Verify all replaced
	newContent, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, 0, len(findAll(string(newContent), "https://old.com")))
	assert.Equal(t, 3, len(findAll(string(newContent), "https://new.com")))
}

func TestFixer_ApplyToFile_URLNotFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")

	content := `# Test
No matching URLs here.
`
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o600))

	changes := FileChanges{
		FilePath: filePath,
		Fixes: []Fix{
			{
				FilePath: filePath,
				OldURL:   "https://notfound.com",
				NewURL:   "https://new.com",
			},
		},
	}

	f := New()
	result, err := f.ApplyToFile(changes)

	require.NoError(t, err)
	assert.Equal(t, 0, result.Applied)
	assert.Equal(t, 1, result.Skipped)
}

func TestFixer_ApplyToFile_FileNotFound(t *testing.T) {
	t.Parallel()

	changes := FileChanges{
		FilePath: "/nonexistent/file.md",
		Fixes: []Fix{
			{
				FilePath: "/nonexistent/file.md",
				OldURL:   "https://old.com",
				NewURL:   "https://new.com",
			},
		},
	}

	f := New()
	result, err := f.ApplyToFile(changes)

	assert.Error(t, err)
	assert.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "reading file")
}

func TestFixer_ApplyToFile_NoChanges(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")

	content := `# Test
Nothing to change.
`
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o600))

	changes := FileChanges{
		FilePath: filePath,
		Fixes:    []Fix{}, // No fixes
	}

	f := New()
	result, err := f.ApplyToFile(changes)

	require.NoError(t, err)
	assert.Equal(t, 0, result.Applied)
}

// =============================================================================
// ApplyAll Tests
// =============================================================================

func TestFixer_ApplyAll(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create two files
	file1 := filepath.Join(tmpDir, "a.md")
	file2 := filepath.Join(tmpDir, "b.md")

	require.NoError(t, os.WriteFile(file1, []byte("Link: https://old1.com"), 0o600))
	require.NoError(t, os.WriteFile(file2, []byte("Link: https://old2.com"), 0o600))

	changes := []FileChanges{
		{
			FilePath: file1,
			Fixes: []Fix{
				{FilePath: file1, OldURL: "https://old1.com", NewURL: "https://new1.com"},
			},
		},
		{
			FilePath: file2,
			Fixes: []Fix{
				{FilePath: file2, OldURL: "https://old2.com", NewURL: "https://new2.com"},
			},
		},
	}

	f := New()
	results := f.ApplyAll(changes)

	require.Len(t, results, 2)
	assert.Equal(t, 1, results[0].Applied)
	assert.Equal(t, 1, results[1].Applied)
}

// =============================================================================
// Summary Tests
// =============================================================================

func TestSummary_NoChanges(t *testing.T) {
	t.Parallel()

	results := []FixResult{
		{FilePath: "a.md", Applied: 0},
		{FilePath: "b.md", Applied: 0},
	}

	summary := Summary(results)
	assert.Equal(t, "No changes made.", summary)
}

func TestSummary_WithChanges(t *testing.T) {
	t.Parallel()

	results := []FixResult{
		{FilePath: "a.md", Applied: 2},
		{FilePath: "b.md", Applied: 1},
	}

	summary := Summary(results)
	assert.Contains(t, summary, "Fixed 3 redirect(s) across 2 file(s)")
}

func TestSummary_WithSkipped(t *testing.T) {
	t.Parallel()

	results := []FixResult{
		{FilePath: "a.md", Applied: 1, Skipped: 2},
	}

	summary := Summary(results)
	assert.Contains(t, summary, "Skipped 2")
}

func TestSummary_WithErrors(t *testing.T) {
	t.Parallel()

	results := []FixResult{
		{FilePath: "a.md", Applied: 1, Error: nil},
		{FilePath: "b.md", Applied: 0, Error: os.ErrNotExist},
	}

	summary := Summary(results)
	assert.Contains(t, summary, "Errors:")
	assert.Contains(t, summary, "b.md")
}

func TestDetailedSummary_NoChanges(t *testing.T) {
	t.Parallel()

	results := []FixResult{
		{FilePath: "a.md", Applied: 0},
	}

	summary := DetailedSummary(results)
	assert.Equal(t, "No changes made.", summary)
}

func TestDetailedSummary_WithChanges(t *testing.T) {
	t.Parallel()

	results := []FixResult{
		{
			FilePath: "a.md",
			Applied:  1,
			ChangedURLs: []URLChange{
				{Line: 10, OldURL: "https://old.com", NewURL: "https://new.com"},
			},
		},
	}

	summary := DetailedSummary(results)
	assert.Contains(t, summary, "a.md:10")
	assert.Contains(t, summary, "https://old.com")
	assert.Contains(t, summary, "https://new.com")
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestTruncateURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		url      string
		maxLen   int
		expected string
	}{
		{"https://short.com", 20, "https://short.com"},
		{"https://example.com/very/long/path/here", 20, "https://example.c..."},
		{"", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, truncateURL(tt.url, tt.maxLen))
		})
	}
}

func TestCountRefUsages(t *testing.T) {
	t.Parallel()

	links := []parser.Link{
		{FilePath: "a.md", RefName: "ref1"},
		{FilePath: "a.md", RefName: "ref1"},
		{FilePath: "a.md", RefName: "ref2"},
		{FilePath: "b.md", RefName: "ref1"},
	}

	count := countRefUsages(links, "a.md", "ref1")
	assert.Equal(t, 2, count)

	count = countRefUsages(links, "a.md", "ref2")
	assert.Equal(t, 1, count)

	count = countRefUsages(links, "b.md", "ref1")
	assert.Equal(t, 1, count)

	count = countRefUsages(links, "a.md", "nonexistent")
	assert.Equal(t, 0, count)
}

// findAll finds all occurrences of a substring.
func findAll(s, substr string) []int {
	var positions []int
	start := 0
	for {
		pos := indexOf(s[start:], substr)
		if pos == -1 {
			break
		}
		positions = append(positions, start+pos)
		start += pos + len(substr)
	}
	return positions
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// =============================================================================
// Additional Fixer Tests for Coverage
// =============================================================================

func TestFixer_FindFixes_ParserLinksWrongFile(t *testing.T) {
	t.Parallel()

	// Parser links exist but for a different file
	parserLinks := []parser.Link{
		{
			URL:        "https://old.com",
			FilePath:   "other.md", // Different file
			Line:       10,
			Type:       parser.LinkTypeInline,
			RefDefLine: 0,
		},
	}

	results := []checker.Result{
		{
			Link:        checker.Link{URL: "https://old.com", FilePath: "test.md", Line: 10},
			Status:      checker.StatusRedirect,
			FinalURL:    "https://new.com",
			FinalStatus: 200,
		},
	}

	f := New()
	f.SetParserLinks(parserLinks)
	changes := f.FindFixes(results)

	require.Len(t, changes, 1)
	require.Len(t, changes[0].Fixes, 1)
	// Should still create a fix, just without reference info
	assert.Equal(t, parser.LinkTypeInline, changes[0].Fixes[0].LinkType)
}

func TestFixer_FindFixes_InlineLinkWithParserLinks(t *testing.T) {
	t.Parallel()

	parserLinks := []parser.Link{
		{
			URL:        "https://old.com",
			FilePath:   "test.md",
			Line:       10,
			Type:       parser.LinkTypeInline,
			RefDefLine: 0, // Not a reference
		},
	}

	results := []checker.Result{
		{
			Link:        checker.Link{URL: "https://old.com", FilePath: "test.md", Line: 10},
			Status:      checker.StatusRedirect,
			FinalURL:    "https://new.com",
			FinalStatus: 200,
		},
	}

	f := New()
	f.SetParserLinks(parserLinks)
	changes := f.FindFixes(results)

	require.Len(t, changes, 1)
	require.Len(t, changes[0].Fixes, 1)
	assert.Equal(t, parser.LinkTypeInline, changes[0].Fixes[0].LinkType)
	assert.False(t, changes[0].Fixes[0].IsRefDef)
}

func TestFixer_Preview_NoRefUsages(t *testing.T) {
	t.Parallel()

	changes := []FileChanges{
		{
			FilePath:   "test.md",
			TotalFixes: 1,
			Fixes: []Fix{
				{
					FilePath:  "test.md",
					Line:      100,
					OldURL:    "https://old.com",
					NewURL:    "https://new.com",
					IsRefDef:  true,
					RefName:   "myref",
					RefUsages: 0, // No usages
				},
			},
		},
	}

	f := New()
	preview := f.Preview(changes)

	assert.Contains(t, preview, "[myref]")
	assert.NotContains(t, preview, "time(s)") // No usage count shown when 0
}

func TestDetailedSummary_MultipleFilesMultipleChanges(t *testing.T) {
	t.Parallel()

	results := []FixResult{
		{
			FilePath: "a.md",
			Applied:  2,
			ChangedURLs: []URLChange{
				{Line: 10, OldURL: "https://old1.com", NewURL: "https://new1.com"},
				{Line: 20, OldURL: "https://old2.com", NewURL: "https://new2.com"},
			},
		},
		{
			FilePath: "b.md",
			Applied:  1,
			ChangedURLs: []URLChange{
				{Line: 5, OldURL: "https://old3.com", NewURL: "https://new3.com"},
			},
		},
	}

	summary := DetailedSummary(results)
	assert.Contains(t, summary, "Fixed 3 redirect(s) across 2 file(s)")
	assert.Contains(t, summary, "a.md:10")
	assert.Contains(t, summary, "a.md:20")
	assert.Contains(t, summary, "b.md:5")
}

func TestDetailedSummary_LongURLs(t *testing.T) {
	t.Parallel()

	longURL := "https://example.com/" + string(make([]byte, 100))
	results := []FixResult{
		{
			FilePath: "test.md",
			Applied:  1,
			ChangedURLs: []URLChange{
				{Line: 10, OldURL: longURL, NewURL: "https://new.com"},
			},
		},
	}

	summary := DetailedSummary(results)
	// URLs should be truncated
	assert.Contains(t, summary, "...")
}

func TestFixer_ApplyToFile_MultipleDifferentURLs(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")

	content := `# Test
Link 1: https://old1.com
Link 2: https://old2.com
Link 3: https://old3.com
`
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o600))

	changes := FileChanges{
		FilePath:   filePath,
		TotalFixes: 3,
		Fixes: []Fix{
			{FilePath: filePath, Line: 2, OldURL: "https://old1.com", NewURL: "https://new1.com"},
			{FilePath: filePath, Line: 3, OldURL: "https://old2.com", NewURL: "https://new2.com"},
			{FilePath: filePath, Line: 4, OldURL: "https://old3.com", NewURL: "https://new3.com"},
		},
	}

	f := New()
	result, err := f.ApplyToFile(changes)

	require.NoError(t, err)
	assert.Equal(t, 3, result.Applied)
	assert.Len(t, result.ChangedURLs, 3)

	// Verify file content
	newContent, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Contains(t, string(newContent), "https://new1.com")
	assert.Contains(t, string(newContent), "https://new2.com")
	assert.Contains(t, string(newContent), "https://new3.com")
	assert.NotContains(t, string(newContent), "https://old1.com")
}

func TestFixer_FindFixes_MixedResults(t *testing.T) {
	t.Parallel()

	results := []checker.Result{
		{
			Link:        checker.Link{URL: "https://redirect.com", FilePath: "test.md", Line: 1},
			Status:      checker.StatusRedirect,
			FinalURL:    "https://new.com",
			FinalStatus: 200,
		},
		{
			Link:       checker.Link{URL: "https://alive.com", FilePath: "test.md", Line: 5},
			Status:     checker.StatusAlive,
			StatusCode: 200,
		},
		{
			Link:       checker.Link{URL: "https://dead.com", FilePath: "test.md", Line: 10},
			Status:     checker.StatusDead,
			StatusCode: 404,
		},
		{
			Link:   checker.Link{URL: "https://error.com", FilePath: "test.md", Line: 15},
			Status: checker.StatusError,
			Error:  "timeout",
		},
	}

	f := New()
	changes := f.FindFixes(results)

	// Only redirect should be fixable
	require.Len(t, changes, 1)
	require.Len(t, changes[0].Fixes, 1)
	assert.Equal(t, "https://redirect.com", changes[0].Fixes[0].OldURL)
}

func TestFixer_Preview_MultipleFixes(t *testing.T) {
	t.Parallel()

	changes := []FileChanges{
		{
			FilePath:   "a.md",
			TotalFixes: 2,
			Fixes: []Fix{
				{FilePath: "a.md", Line: 10, OldURL: "https://old1.com", NewURL: "https://new1.com"},
				{FilePath: "a.md", Line: 20, OldURL: "https://old2.com", NewURL: "https://new2.com"},
			},
		},
		{
			FilePath:   "b.md",
			TotalFixes: 1,
			Fixes: []Fix{
				{FilePath: "b.md", Line: 5, OldURL: "https://old3.com", NewURL: "https://new3.com"},
			},
		},
	}

	f := New()
	preview := f.Preview(changes)

	assert.Contains(t, preview, "Found 3 fixable redirect(s) across 2 file(s)")
	assert.Contains(t, preview, "a.md (2 fix(es))")
	assert.Contains(t, preview, "b.md (1 fix(es))")
}
