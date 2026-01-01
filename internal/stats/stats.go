// Package stats provides performance tracking and statistics for link checking operations.
// It captures timing information for each phase of execution, memory usage,
// and throughput metrics to help identify bottlenecks and optimize performance.
package stats

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// Stats holds performance metrics for a link checking session.
type Stats struct {
	// Timing for each phase
	ScanStart  time.Time
	ScanEnd    time.Time
	ParseStart time.Time
	ParseEnd   time.Time
	CheckStart time.Time
	CheckEnd   time.Time

	// Counts
	FilesScanned int
	LinksFound   int
	UniqueURLs   int
	Duplicates   int
	Ignored      int

	// Memory stats (captured at end)
	HeapAlloc    uint64
	TotalAlloc   uint64
	NumGC        uint32
	NumGoroutine int
}

// New creates a new Stats instance.
func New() *Stats {
	return &Stats{}
}

// StartScan marks the beginning of the file scanning phase.
func (s *Stats) StartScan() {
	s.ScanStart = time.Now()
}

// EndScan marks the end of the file scanning phase.
func (s *Stats) EndScan(filesFound int) {
	s.ScanEnd = time.Now()
	s.FilesScanned = filesFound
}

// StartParse marks the beginning of the link parsing phase.
func (s *Stats) StartParse() {
	s.ParseStart = time.Now()
}

// EndParse marks the end of the link parsing phase.
func (s *Stats) EndParse(linksFound, uniqueURLs, duplicates, ignored int) {
	s.ParseEnd = time.Now()
	s.LinksFound = linksFound
	s.UniqueURLs = uniqueURLs
	s.Duplicates = duplicates
	s.Ignored = ignored
}

// StartCheck marks the beginning of the URL checking phase.
func (s *Stats) StartCheck() {
	s.CheckStart = time.Now()
}

// EndCheck marks the end of the URL checking phase and captures memory stats.
func (s *Stats) EndCheck() {
	s.CheckEnd = time.Now()
	s.captureMemoryStats()
}

// captureMemoryStats reads current memory statistics from runtime.
func (s *Stats) captureMemoryStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	s.HeapAlloc = m.HeapAlloc
	s.TotalAlloc = m.TotalAlloc
	s.NumGC = m.NumGC
	s.NumGoroutine = runtime.NumGoroutine()
}

// ScanDuration returns the time spent scanning for files.
func (s *Stats) ScanDuration() time.Duration {
	if s.ScanEnd.IsZero() {
		return 0
	}
	return s.ScanEnd.Sub(s.ScanStart)
}

// ParseDuration returns the time spent parsing links from files.
func (s *Stats) ParseDuration() time.Duration {
	if s.ParseEnd.IsZero() {
		return 0
	}
	return s.ParseEnd.Sub(s.ParseStart)
}

// CheckDuration returns the time spent checking URLs.
func (s *Stats) CheckDuration() time.Duration {
	if s.CheckEnd.IsZero() {
		return 0
	}
	return s.CheckEnd.Sub(s.CheckStart)
}

// TotalDuration returns the total time from scan start to check end.
func (s *Stats) TotalDuration() time.Duration {
	if s.CheckEnd.IsZero() {
		return 0
	}
	return s.CheckEnd.Sub(s.ScanStart)
}

// URLsPerSecond returns the throughput of URL checking.
func (s *Stats) URLsPerSecond() float64 {
	checkDur := s.CheckDuration()
	if checkDur == 0 || s.UniqueURLs == 0 {
		return 0
	}
	return float64(s.UniqueURLs) / checkDur.Seconds()
}

// AvgResponseTime returns the average time per URL check.
func (s *Stats) AvgResponseTime() time.Duration {
	checkDur := s.CheckDuration()
	if s.UniqueURLs == 0 {
		return 0
	}
	return checkDur / time.Duration(s.UniqueURLs)
}

// FormatDuration formats a duration for display.
func FormatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%dm%.1fs", int(d.Minutes()), d.Seconds()-float64(int(d.Minutes())*60))
}

// FormatBytes formats bytes for human-readable display.
func FormatBytes(bytes uint64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)

	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/gb)
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/mb)
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/kb)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// String returns a formatted string representation of the stats.
func (s *Stats) String() string {
	var b strings.Builder

	total := s.TotalDuration()

	b.WriteString("\n=== Performance Statistics ===\n\n")

	// Timing breakdown
	b.WriteString("Timing:\n")
	b.WriteString(fmt.Sprintf("  Scan files:    %8s", FormatDuration(s.ScanDuration())))
	if total > 0 {
		b.WriteString(fmt.Sprintf("  (%4.1f%%)", float64(s.ScanDuration())/float64(total)*100))
	}
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("  Parse links:   %8s", FormatDuration(s.ParseDuration())))
	if total > 0 {
		b.WriteString(fmt.Sprintf("  (%4.1f%%)", float64(s.ParseDuration())/float64(total)*100))
	}
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("  Check URLs:    %8s", FormatDuration(s.CheckDuration())))
	if total > 0 {
		b.WriteString(fmt.Sprintf("  (%4.1f%%)", float64(s.CheckDuration())/float64(total)*100))
	}
	b.WriteString("\n")

	b.WriteString("  ─────────────────────────\n")
	b.WriteString(fmt.Sprintf("  Total:         %8s\n", FormatDuration(total)))

	// Throughput
	b.WriteString("\nThroughput:\n")
	b.WriteString(fmt.Sprintf("  Files scanned:     %5d\n", s.FilesScanned))
	b.WriteString(fmt.Sprintf("  Links found:       %5d\n", s.LinksFound))
	b.WriteString(fmt.Sprintf("  Unique URLs:       %5d\n", s.UniqueURLs))
	if s.Duplicates > 0 {
		b.WriteString(fmt.Sprintf("  Duplicates:        %5d\n", s.Duplicates))
	}
	if s.Ignored > 0 {
		b.WriteString(fmt.Sprintf("  Ignored:           %5d\n", s.Ignored))
	}
	b.WriteString(fmt.Sprintf("  URLs/second:       %5.1f\n", s.URLsPerSecond()))
	b.WriteString(fmt.Sprintf("  Avg response:    %7s\n", FormatDuration(s.AvgResponseTime())))

	// Memory
	b.WriteString("\nMemory:\n")
	b.WriteString(fmt.Sprintf("  Heap in use:   %8s\n", FormatBytes(s.HeapAlloc)))
	b.WriteString(fmt.Sprintf("  Total alloc:   %8s\n", FormatBytes(s.TotalAlloc)))
	b.WriteString(fmt.Sprintf("  GC cycles:     %8d\n", s.NumGC))
	b.WriteString(fmt.Sprintf("  Goroutines:    %8d\n", s.NumGoroutine))

	return b.String()
}

// ToJSON returns a map suitable for JSON serialization.
func (s *Stats) ToJSON() map[string]any {
	return map[string]any{
		"timing": map[string]any{
			"scan_ms":  s.ScanDuration().Milliseconds(),
			"parse_ms": s.ParseDuration().Milliseconds(),
			"check_ms": s.CheckDuration().Milliseconds(),
			"total_ms": s.TotalDuration().Milliseconds(),
		},
		"throughput": map[string]any{
			"files_scanned":   s.FilesScanned,
			"links_found":     s.LinksFound,
			"unique_urls":     s.UniqueURLs,
			"duplicates":      s.Duplicates,
			"ignored":         s.Ignored,
			"urls_per_second": s.URLsPerSecond(),
			"avg_response_ms": s.AvgResponseTime().Milliseconds(),
		},
		"memory": map[string]any{
			"heap_bytes":  s.HeapAlloc,
			"total_bytes": s.TotalAlloc,
			"gc_cycles":   s.NumGC,
			"goroutines":  s.NumGoroutine,
		},
	}
}
