package cmd

import (
	"testing"

	"github.com/leonardomso/gone/internal/checker"
	"github.com/leonardomso/gone/internal/parser"
)

// BenchmarkFilterResultsWarnings measures warning filter performance.
func BenchmarkFilterResultsWarnings(b *testing.B) {
	results := createBenchResults(1000)

	b.ResetTimer()
	for b.Loop() {
		FilterResultsWarnings(results)
	}
}

// BenchmarkFilterResultsDead measures dead filter performance.
func BenchmarkFilterResultsDead(b *testing.B) {
	results := createBenchResults(1000)

	b.ResetTimer()
	for b.Loop() {
		FilterResultsDead(results)
	}
}

// BenchmarkFilterResultsDuplicates measures duplicate filter performance.
func BenchmarkFilterResultsDuplicates(b *testing.B) {
	results := createBenchResults(1000)

	b.ResetTimer()
	for b.Loop() {
		FilterResultsDuplicates(results)
	}
}

// BenchmarkFilterResultsAlive measures alive filter performance.
func BenchmarkFilterResultsAlive(b *testing.B) {
	results := createBenchResults(1000)

	b.ResetTimer()
	for b.Loop() {
		FilterResultsAlive(results)
	}
}

// BenchmarkCountUniqueURLs measures URL counting performance.
func BenchmarkCountUniqueURLs(b *testing.B) {
	links := make([]checker.Link, 1000)
	for i := range links {
		links[i] = checker.Link{
			URL: "http://example.com/page" + string(rune('0'+i%100)),
		}
	}

	b.ResetTimer()
	for b.Loop() {
		CountUniqueURLs(links)
	}
}

// BenchmarkConvertParserLinks measures link conversion performance.
func BenchmarkConvertParserLinks(b *testing.B) {
	parserLinks := createBenchParserLinks(1000)

	b.ResetTimer()
	for b.Loop() {
		ConvertParserLinks(parserLinks)
	}
}

// createBenchResults creates test results with varied statuses.
func createBenchResults(n int) []checker.Result {
	results := make([]checker.Result, n)
	statuses := []checker.LinkStatus{
		checker.StatusAlive, checker.StatusAlive, checker.StatusAlive, checker.StatusAlive,
		checker.StatusRedirect, checker.StatusBlocked,
		checker.StatusDead, checker.StatusError,
		checker.StatusDuplicate, checker.StatusDuplicate,
	}

	for i := range results {
		results[i] = checker.Result{
			Link:       checker.Link{URL: "http://example.com/page" + string(rune('0'+i%100))},
			Status:     statuses[i%len(statuses)],
			StatusCode: 200,
		}
	}
	return results
}

// createBenchParserLinks creates test parser links.
func createBenchParserLinks(n int) []parser.Link {
	links := make([]parser.Link, n)
	for i := range links {
		links[i] = parser.Link{
			URL:      "http://example.com/page" + string(rune('0'+i%100)),
			FilePath: "test.md",
			Line:     i + 1,
			Text:     "Link text",
		}
	}
	return links
}
