package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	s := New()

	require.NotNil(t, s)
	assert.True(t, s.ScanStart.IsZero())
	assert.True(t, s.ScanEnd.IsZero())
	assert.True(t, s.ParseStart.IsZero())
	assert.True(t, s.ParseEnd.IsZero())
	assert.True(t, s.CheckStart.IsZero())
	assert.True(t, s.CheckEnd.IsZero())
	assert.Equal(t, 0, s.FilesScanned)
	assert.Equal(t, 0, s.LinksFound)
	assert.Equal(t, 0, s.UniqueURLs)
	assert.Equal(t, 0, s.Duplicates)
	assert.Equal(t, 0, s.Ignored)
}

func TestScanPhase(t *testing.T) {
	t.Parallel()

	t.Run("StartScan", func(t *testing.T) {
		t.Parallel()
		s := New()
		s.StartScan()

		assert.False(t, s.ScanStart.IsZero())
		assert.True(t, s.ScanEnd.IsZero())
	})

	t.Run("EndScan", func(t *testing.T) {
		t.Parallel()
		s := New()
		s.StartScan()
		time.Sleep(10 * time.Millisecond)
		s.EndScan(25)

		assert.False(t, s.ScanEnd.IsZero())
		assert.Equal(t, 25, s.FilesScanned)
	})

	t.Run("ScanDuration", func(t *testing.T) {
		t.Parallel()
		s := New()

		// Duration is 0 before ending
		assert.Equal(t, time.Duration(0), s.ScanDuration())

		s.StartScan()
		time.Sleep(10 * time.Millisecond)
		s.EndScan(10)

		duration := s.ScanDuration()
		assert.True(t, duration >= 10*time.Millisecond)
	})
}

func TestParsePhase(t *testing.T) {
	t.Parallel()

	t.Run("StartParse", func(t *testing.T) {
		t.Parallel()
		s := New()
		s.StartParse()

		assert.False(t, s.ParseStart.IsZero())
		assert.True(t, s.ParseEnd.IsZero())
	})

	t.Run("EndParse", func(t *testing.T) {
		t.Parallel()
		s := New()
		s.StartParse()
		time.Sleep(10 * time.Millisecond)
		s.EndParse(100, 80, 15, 5)

		assert.False(t, s.ParseEnd.IsZero())
		assert.Equal(t, 100, s.LinksFound)
		assert.Equal(t, 80, s.UniqueURLs)
		assert.Equal(t, 15, s.Duplicates)
		assert.Equal(t, 5, s.Ignored)
	})

	t.Run("ParseDuration", func(t *testing.T) {
		t.Parallel()
		s := New()

		// Duration is 0 before ending
		assert.Equal(t, time.Duration(0), s.ParseDuration())

		s.StartParse()
		time.Sleep(10 * time.Millisecond)
		s.EndParse(100, 80, 15, 5)

		duration := s.ParseDuration()
		assert.True(t, duration >= 10*time.Millisecond)
	})
}

func TestCheckPhase(t *testing.T) {
	t.Parallel()

	t.Run("StartCheck", func(t *testing.T) {
		t.Parallel()
		s := New()
		s.StartCheck()

		assert.False(t, s.CheckStart.IsZero())
		assert.True(t, s.CheckEnd.IsZero())
	})

	t.Run("EndCheck", func(t *testing.T) {
		t.Parallel()
		s := New()
		s.StartCheck()
		time.Sleep(10 * time.Millisecond)
		s.EndCheck()

		assert.False(t, s.CheckEnd.IsZero())
		// Memory stats should be populated
		assert.True(t, s.HeapAlloc > 0)
		assert.True(t, s.TotalAlloc > 0)
		assert.True(t, s.NumGoroutine > 0)
	})

	t.Run("CheckDuration", func(t *testing.T) {
		t.Parallel()
		s := New()

		// Duration is 0 before ending
		assert.Equal(t, time.Duration(0), s.CheckDuration())

		s.StartCheck()
		time.Sleep(10 * time.Millisecond)
		s.EndCheck()

		duration := s.CheckDuration()
		assert.True(t, duration >= 10*time.Millisecond)
	})
}

func TestTotalDuration(t *testing.T) {
	t.Parallel()

	t.Run("ReturnsZeroWhenIncomplete", func(t *testing.T) {
		t.Parallel()
		s := New()
		s.StartScan()
		s.EndScan(10)
		s.StartParse()
		s.EndParse(100, 80, 15, 5)
		s.StartCheck()
		// CheckEnd not set

		assert.Equal(t, time.Duration(0), s.TotalDuration())
	})

	t.Run("ReturnsFullDuration", func(t *testing.T) {
		t.Parallel()
		s := New()
		s.StartScan()
		time.Sleep(5 * time.Millisecond)
		s.EndScan(10)
		s.StartParse()
		time.Sleep(5 * time.Millisecond)
		s.EndParse(100, 80, 15, 5)
		s.StartCheck()
		time.Sleep(5 * time.Millisecond)
		s.EndCheck()

		duration := s.TotalDuration()
		assert.True(t, duration >= 15*time.Millisecond)
	})
}

func TestURLsPerSecond(t *testing.T) {
	t.Parallel()

	t.Run("ReturnsZeroWhenNoURLs", func(t *testing.T) {
		t.Parallel()
		s := New()
		s.StartCheck()
		time.Sleep(10 * time.Millisecond)
		s.EndCheck()
		s.UniqueURLs = 0

		assert.Equal(t, 0.0, s.URLsPerSecond())
	})

	t.Run("ReturnsZeroWhenNoDuration", func(t *testing.T) {
		t.Parallel()
		s := New()
		s.UniqueURLs = 100
		// CheckStart and CheckEnd are zero

		assert.Equal(t, 0.0, s.URLsPerSecond())
	})

	t.Run("CalculatesCorrectly", func(t *testing.T) {
		t.Parallel()
		s := New()
		s.UniqueURLs = 100
		// Set times directly to avoid timing variations
		s.CheckStart = time.Now()
		s.CheckEnd = s.CheckStart.Add(2 * time.Second)

		urlsPerSec := s.URLsPerSecond()
		assert.InDelta(t, 50.0, urlsPerSec, 0.1)
	})
}

func TestAvgResponseTime(t *testing.T) {
	t.Parallel()

	t.Run("ReturnsZeroWhenNoURLs", func(t *testing.T) {
		t.Parallel()
		s := New()
		s.StartCheck()
		time.Sleep(10 * time.Millisecond)
		s.EndCheck()
		s.UniqueURLs = 0

		assert.Equal(t, time.Duration(0), s.AvgResponseTime())
	})

	t.Run("CalculatesCorrectly", func(t *testing.T) {
		t.Parallel()
		s := New()
		s.UniqueURLs = 100
		// Set times directly to avoid timing variations
		s.CheckStart = time.Now()
		s.CheckEnd = s.CheckStart.Add(2 * time.Second)

		avgTime := s.AvgResponseTime()
		assert.Equal(t, 20*time.Millisecond, avgTime)
	})
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "Zero",
			duration: 0,
			expected: "0µs",
		},
		{
			name:     "Microseconds",
			duration: 500 * time.Microsecond,
			expected: "500µs",
		},
		{
			name:     "Milliseconds",
			duration: 500 * time.Millisecond,
			expected: "500ms",
		},
		{
			name:     "JustUnderSecond",
			duration: 999 * time.Millisecond,
			expected: "999ms",
		},
		{
			name:     "Seconds",
			duration: 2500 * time.Millisecond,
			expected: "2.5s",
		},
		{
			name:     "JustUnderMinute",
			duration: 59*time.Second + 500*time.Millisecond,
			expected: "59.5s",
		},
		{
			name:     "Minutes",
			duration: 65 * time.Second,
			expected: "1m5.0s",
		},
		{
			name:     "MultipleMinutes",
			duration: 125 * time.Second,
			expected: "2m5.0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := FormatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bytes    uint64
		expected string
	}{
		{
			name:     "Zero",
			bytes:    0,
			expected: "0 B",
		},
		{
			name:     "Bytes",
			bytes:    500,
			expected: "500 B",
		},
		{
			name:     "JustUnderKB",
			bytes:    1023,
			expected: "1023 B",
		},
		{
			name:     "Kilobytes",
			bytes:    1536,
			expected: "1.5 KB",
		},
		{
			name:     "Megabytes",
			bytes:    1572864,
			expected: "1.5 MB",
		},
		{
			name:     "Gigabytes",
			bytes:    1610612736,
			expected: "1.5 GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := FormatBytes(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestString(t *testing.T) {
	t.Parallel()

	t.Run("ContainsAllSections", func(t *testing.T) {
		t.Parallel()
		s := New()
		s.StartScan()
		s.EndScan(25)
		s.StartParse()
		s.EndParse(100, 80, 15, 5)
		s.StartCheck()
		s.EndCheck()

		output := s.String()

		assert.Contains(t, output, "Performance Statistics")
		assert.Contains(t, output, "Timing:")
		assert.Contains(t, output, "Scan files:")
		assert.Contains(t, output, "Parse links:")
		assert.Contains(t, output, "Check URLs:")
		assert.Contains(t, output, "Total:")
		assert.Contains(t, output, "Throughput:")
		assert.Contains(t, output, "Files scanned:")
		assert.Contains(t, output, "Links found:")
		assert.Contains(t, output, "Unique URLs:")
		assert.Contains(t, output, "URLs/second:")
		assert.Contains(t, output, "Avg response:")
		assert.Contains(t, output, "Memory:")
		assert.Contains(t, output, "Heap in use:")
		assert.Contains(t, output, "Total alloc:")
		assert.Contains(t, output, "GC cycles:")
		assert.Contains(t, output, "Goroutines:")
	})

	t.Run("IncludesDuplicatesWhenPresent", func(t *testing.T) {
		t.Parallel()
		s := New()
		s.Duplicates = 10

		output := s.String()
		assert.Contains(t, output, "Duplicates:")
	})

	t.Run("ExcludesDuplicatesWhenZero", func(t *testing.T) {
		t.Parallel()
		s := New()
		s.Duplicates = 0

		output := s.String()
		assert.NotContains(t, output, "Duplicates:")
	})

	t.Run("IncludesIgnoredWhenPresent", func(t *testing.T) {
		t.Parallel()
		s := New()
		s.Ignored = 5

		output := s.String()
		assert.Contains(t, output, "Ignored:")
	})

	t.Run("ExcludesIgnoredWhenZero", func(t *testing.T) {
		t.Parallel()
		s := New()
		s.Ignored = 0

		output := s.String()
		assert.NotContains(t, output, "Ignored:")
	})
}

func TestToJSON(t *testing.T) {
	t.Parallel()

	t.Run("HasCorrectStructure", func(t *testing.T) {
		t.Parallel()
		s := New()
		s.StartScan()
		s.EndScan(25)
		s.StartParse()
		s.EndParse(100, 80, 15, 5)
		s.StartCheck()
		s.EndCheck()

		result := s.ToJSON()

		// Check top-level keys
		assert.Contains(t, result, "timing")
		assert.Contains(t, result, "throughput")
		assert.Contains(t, result, "memory")

		// Check timing keys
		timing, ok := result["timing"].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, timing, "scan_ms")
		assert.Contains(t, timing, "parse_ms")
		assert.Contains(t, timing, "check_ms")
		assert.Contains(t, timing, "total_ms")

		// Check throughput keys
		throughput, ok := result["throughput"].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, throughput, "files_scanned")
		assert.Contains(t, throughput, "links_found")
		assert.Contains(t, throughput, "unique_urls")
		assert.Contains(t, throughput, "duplicates")
		assert.Contains(t, throughput, "ignored")
		assert.Contains(t, throughput, "urls_per_second")
		assert.Contains(t, throughput, "avg_response_ms")

		// Check memory keys
		memory, ok := result["memory"].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, memory, "heap_bytes")
		assert.Contains(t, memory, "total_bytes")
		assert.Contains(t, memory, "gc_cycles")
		assert.Contains(t, memory, "goroutines")
	})

	t.Run("ValuesMatchFields", func(t *testing.T) {
		t.Parallel()
		s := New()
		s.FilesScanned = 25
		s.LinksFound = 100
		s.UniqueURLs = 80
		s.Duplicates = 15
		s.Ignored = 5

		result := s.ToJSON()

		throughput, ok := result["throughput"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 25, throughput["files_scanned"])
		assert.Equal(t, 100, throughput["links_found"])
		assert.Equal(t, 80, throughput["unique_urls"])
		assert.Equal(t, 15, throughput["duplicates"])
		assert.Equal(t, 5, throughput["ignored"])
	})
}
