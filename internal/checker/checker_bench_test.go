package checker

import (
	"context"
	"testing"
)

// BenchmarkBackoffDelay measures jitter calculation performance.
// This benchmarks the random number generation used for retry backoff.
func BenchmarkBackoffDelay(b *testing.B) {
	for b.Loop() {
		backoffDelay(2)
	}
}

// BenchmarkDeduplicateLinks measures URL deduplication performance.
// This is a hot path when processing many links.
func BenchmarkDeduplicateLinks(b *testing.B) {
	// Create test data with ~10% duplicates
	links := make([]Link, 1000)
	for i := range links {
		links[i] = Link{
			URL:      "http://example.com/page" + string(rune('0'+i%100)),
			FilePath: "test.md",
			Line:     i + 1,
		}
	}

	b.ResetTimer()
	for b.Loop() {
		// Simulate the deduplication logic from Check()
		urlToLinks := make(map[string][]Link, len(links))
		urlOrder := make([]string, 0, len(links))
		for _, link := range links {
			if _, exists := urlToLinks[link.URL]; !exists {
				urlOrder = append(urlOrder, link.URL)
			}
			urlToLinks[link.URL] = append(urlToLinks[link.URL], link)
		}
		_ = urlOrder
		_ = urlToLinks
	}
}

// BenchmarkDeduplicateLinks_Small benchmarks with fewer links.
func BenchmarkDeduplicateLinks_Small(b *testing.B) {
	links := make([]Link, 50)
	for i := range links {
		links[i] = Link{
			URL:      "http://example.com/page" + string(rune('0'+i%10)),
			FilePath: "test.md",
			Line:     i + 1,
		}
	}

	b.ResetTimer()
	for b.Loop() {
		urlToLinks := make(map[string][]Link, len(links))
		urlOrder := make([]string, 0, len(links))
		for _, link := range links {
			if _, exists := urlToLinks[link.URL]; !exists {
				urlOrder = append(urlOrder, link.URL)
			}
			urlToLinks[link.URL] = append(urlToLinks[link.URL], link)
		}
		_ = urlOrder
		_ = urlToLinks
	}
}

// BenchmarkCheckAll benchmarks the full check pipeline with a mock.
// Note: This doesn't make real HTTP requests.
func BenchmarkCheckAll_Setup(b *testing.B) {
	links := make([]Link, 100)
	for i := range links {
		links[i] = Link{
			URL:      "http://example.com/page" + string(rune('0'+i%50)),
			FilePath: "test.md",
			Line:     i + 1,
		}
	}

	checker := New(DefaultOptions())
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately to avoid actual HTTP requests

	b.ResetTimer()
	for b.Loop() {
		// This will exercise the setup/deduplication code path
		ch := checker.Check(ctx, links)
		//nolint:revive // Draining channel intentionally empty
		for range ch {
		}
	}
}
