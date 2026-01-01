package checker

import "testing"

// BenchmarkSummarize measures the performance of result summarization.
// This is called after all links are checked to generate statistics.
func BenchmarkSummarize(b *testing.B) {
	results := createTestResults(1000)

	b.ResetTimer()
	for b.Loop() {
		Summarize(results)
	}
}

// BenchmarkSummarize_Small benchmarks with fewer results.
func BenchmarkSummarize_Small(b *testing.B) {
	results := createTestResults(100)

	b.ResetTimer()
	for b.Loop() {
		Summarize(results)
	}
}

// BenchmarkSummarize_Large benchmarks with many results.
func BenchmarkSummarize_Large(b *testing.B) {
	results := createTestResults(10000)

	b.ResetTimer()
	for b.Loop() {
		Summarize(results)
	}
}

// BenchmarkFilterByStatus measures filter performance.
func BenchmarkFilterByStatus(b *testing.B) {
	results := createTestResults(1000)

	b.ResetTimer()
	for b.Loop() {
		FilterByStatus(results, StatusAlive)
	}
}

// BenchmarkFilterWarnings measures warning filter performance.
func BenchmarkFilterWarnings(b *testing.B) {
	results := createTestResults(1000)

	b.ResetTimer()
	for b.Loop() {
		FilterWarnings(results)
	}
}

// BenchmarkFilterDead measures dead link filter performance.
func BenchmarkFilterDead(b *testing.B) {
	results := createTestResults(1000)

	b.ResetTimer()
	for b.Loop() {
		FilterDead(results)
	}
}

// createTestResults creates test results with varied statuses.
func createTestResults(n int) []Result {
	results := make([]Result, n)
	statuses := []LinkStatus{
		StatusAlive, StatusAlive, StatusAlive, StatusAlive, // 40% alive
		StatusRedirect, StatusBlocked, // 20% warnings
		StatusDead, StatusError, // 20% dead
		StatusDuplicate, StatusDuplicate, // 20% duplicates
	}

	for i := range results {
		results[i] = Result{
			Link:       Link{URL: "http://example.com/page" + string(rune('0'+i%100))},
			Status:     statuses[i%len(statuses)],
			StatusCode: 200,
		}
	}
	return results
}
