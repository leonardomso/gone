package checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Options Tests
// =============================================================================

func TestDefaultOptions(t *testing.T) {
	t.Parallel()

	opts := DefaultOptions()

	// Optimized defaults for performance
	assert.Equal(t, DefaultConcurrency, opts.Concurrency)
	assert.Equal(t, DefaultTimeout, opts.Timeout)
	assert.Equal(t, DefaultMaxRetries, opts.MaxRetries)
	assert.Equal(t, DefaultMaxRedirects, opts.MaxRedirects)
	assert.Equal(t, DefaultUserAgent, opts.UserAgent)
}

func TestOptionsWithMethods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		modifier func(Options) Options
		check    func(*testing.T, Options)
	}{
		{
			name:     "WithConcurrency",
			modifier: func(o Options) Options { return o.WithConcurrency(20) },
			check:    func(t *testing.T, o Options) { assert.Equal(t, 20, o.Concurrency) },
		},
		{
			name:     "WithTimeout",
			modifier: func(o Options) Options { return o.WithTimeout(30 * time.Second) },
			check:    func(t *testing.T, o Options) { assert.Equal(t, 30*time.Second, o.Timeout) },
		},
		{
			name:     "WithMaxRetries",
			modifier: func(o Options) Options { return o.WithMaxRetries(5) },
			check:    func(t *testing.T, o Options) { assert.Equal(t, 5, o.MaxRetries) },
		},
		{
			name:     "WithMaxRedirects",
			modifier: func(o Options) Options { return o.WithMaxRedirects(15) },
			check:    func(t *testing.T, o Options) { assert.Equal(t, 15, o.MaxRedirects) },
		},
		{
			name:     "WithUserAgent",
			modifier: func(o Options) Options { return o.WithUserAgent("custom-agent/2.0") },
			check:    func(t *testing.T, o Options) { assert.Equal(t, "custom-agent/2.0", o.UserAgent) },
		},
		{
			name: "ChainedMethods",
			modifier: func(o Options) Options {
				return o.WithConcurrency(5).WithTimeout(5 * time.Second).WithMaxRetries(1)
			},
			check: func(t *testing.T, o Options) {
				assert.Equal(t, 5, o.Concurrency)
				assert.Equal(t, 5*time.Second, o.Timeout)
				assert.Equal(t, 1, o.MaxRetries)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := tt.modifier(DefaultOptions())
			tt.check(t, opts)
		})
	}
}

// =============================================================================
// LinkStatus Tests
// =============================================================================

func TestLinkStatus_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status   LinkStatus
		expected string
	}{
		{StatusAlive, "alive"},
		{StatusRedirect, "redirect"},
		{StatusBlocked, "blocked"},
		{StatusDead, "dead"},
		{StatusError, "error"},
		{StatusDuplicate, "duplicate"},
		{LinkStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestLinkStatus_Label(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status   LinkStatus
		expected string
	}{
		{StatusAlive, "OK"},
		{StatusRedirect, "REDIRECT"},
		{StatusBlocked, "BLOCKED"},
		{StatusDead, "DEAD"},
		{StatusError, "ERROR"},
		{StatusDuplicate, "DUPLICATE"},
		{LinkStatus(99), "???"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.status.Label())
		})
	}
}

func TestLinkStatus_Description(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status      LinkStatus
		containsStr string
	}{
		{StatusAlive, "working"},
		{StatusRedirect, "redirected"},
		{StatusBlocked, "403"},
		{StatusDead, "broken"},
		{StatusError, "Network error"},
		{StatusDuplicate, "multiple times"},
		{LinkStatus(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, tt.status.Description(), tt.containsStr)
		})
	}
}

// =============================================================================
// Result Tests
// =============================================================================

func TestResult_IsAlive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status   LinkStatus
		expected bool
	}{
		{StatusAlive, true},
		{StatusRedirect, false},
		{StatusBlocked, false},
		{StatusDead, false},
		{StatusError, false},
		{StatusDuplicate, false},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			t.Parallel()
			r := Result{Status: tt.status}
			assert.Equal(t, tt.expected, r.IsAlive())
		})
	}
}

func TestResult_IsWarning(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status   LinkStatus
		expected bool
	}{
		{StatusAlive, false},
		{StatusRedirect, true},
		{StatusBlocked, true},
		{StatusDead, false},
		{StatusError, false},
		{StatusDuplicate, false},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			t.Parallel()
			r := Result{Status: tt.status}
			assert.Equal(t, tt.expected, r.IsWarning())
		})
	}
}

func TestResult_IsDead(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status   LinkStatus
		expected bool
	}{
		{StatusAlive, false},
		{StatusRedirect, false},
		{StatusBlocked, false},
		{StatusDead, true},
		{StatusError, true},
		{StatusDuplicate, false},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			t.Parallel()
			r := Result{Status: tt.status}
			assert.Equal(t, tt.expected, r.IsDead())
		})
	}
}

func TestResult_IsDuplicate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status   LinkStatus
		expected bool
	}{
		{StatusAlive, false},
		{StatusRedirect, false},
		{StatusBlocked, false},
		{StatusDead, false},
		{StatusError, false},
		{StatusDuplicate, true},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			t.Parallel()
			r := Result{Status: tt.status}
			assert.Equal(t, tt.expected, r.IsDuplicate())
		})
	}
}

func TestResult_StatusDisplay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		result   Result
		expected string
	}{
		{
			name:     "Alive",
			result:   Result{Status: StatusAlive, StatusCode: 200},
			expected: "[200]",
		},
		{
			name:     "Redirect",
			result:   Result{Status: StatusRedirect},
			expected: "[REDIRECT]",
		},
		{
			name:     "Blocked",
			result:   Result{Status: StatusBlocked},
			expected: "[BLOCKED]",
		},
		{
			name:     "DeadWithCode",
			result:   Result{Status: StatusDead, StatusCode: 404},
			expected: "[404]",
		},
		{
			name:     "DeadNoCode",
			result:   Result{Status: StatusDead, StatusCode: 0},
			expected: "[DEAD]",
		},
		{
			name:     "Error",
			result:   Result{Status: StatusError},
			expected: "[ERROR]",
		},
		{
			name:     "Duplicate",
			result:   Result{Status: StatusDuplicate},
			expected: "[DUPLICATE]",
		},
		{
			name:     "Unknown",
			result:   Result{Status: LinkStatus(99)},
			expected: "[???]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.result.StatusDisplay())
		})
	}
}

// =============================================================================
// Filter Functions Tests
// =============================================================================

func TestFilterByStatus(t *testing.T) {
	t.Parallel()

	results := []Result{
		{Link: Link{URL: "http://a.com"}, Status: StatusAlive},
		{Link: Link{URL: "http://b.com"}, Status: StatusDead},
		{Link: Link{URL: "http://c.com"}, Status: StatusAlive},
		{Link: Link{URL: "http://d.com"}, Status: StatusRedirect},
	}

	alive := FilterByStatus(results, StatusAlive)
	assert.Len(t, alive, 2)
	assert.Equal(t, "http://a.com", alive[0].Link.URL)
	assert.Equal(t, "http://c.com", alive[1].Link.URL)

	dead := FilterByStatus(results, StatusDead)
	assert.Len(t, dead, 1)
	assert.Equal(t, "http://b.com", dead[0].Link.URL)

	redirects := FilterByStatus(results, StatusRedirect)
	assert.Len(t, redirects, 1)

	// Empty result
	errors := FilterByStatus(results, StatusError)
	assert.Empty(t, errors)
}

func TestFilterWarnings(t *testing.T) {
	t.Parallel()

	results := []Result{
		{Status: StatusAlive},
		{Status: StatusRedirect},
		{Status: StatusBlocked},
		{Status: StatusDead},
		{Status: StatusRedirect},
	}

	warnings := FilterWarnings(results)
	assert.Len(t, warnings, 3)
}

func TestFilterDead(t *testing.T) {
	t.Parallel()

	results := []Result{
		{Status: StatusAlive},
		{Status: StatusDead},
		{Status: StatusError},
		{Status: StatusDead},
	}

	dead := FilterDead(results)
	assert.Len(t, dead, 3)
}

func TestFilterAlive(t *testing.T) {
	t.Parallel()

	results := []Result{
		{Status: StatusAlive},
		{Status: StatusDead},
		{Status: StatusAlive},
	}

	alive := FilterAlive(results)
	assert.Len(t, alive, 2)
}

func TestFilterDuplicates(t *testing.T) {
	t.Parallel()

	results := []Result{
		{Status: StatusAlive},
		{Status: StatusDuplicate},
		{Status: StatusDuplicate},
		{Status: StatusDead},
	}

	duplicates := FilterDuplicates(results)
	assert.Len(t, duplicates, 2)
}

// =============================================================================
// Summary Tests
// =============================================================================

func TestSummarize(t *testing.T) {
	t.Parallel()

	results := []Result{
		{Link: Link{URL: "http://a.com"}, Status: StatusAlive},
		{Link: Link{URL: "http://b.com"}, Status: StatusDead},
		{Link: Link{URL: "http://c.com"}, Status: StatusRedirect},
		{Link: Link{URL: "http://d.com"}, Status: StatusBlocked},
		{Link: Link{URL: "http://e.com"}, Status: StatusError},
		{Link: Link{URL: "http://a.com"}, Status: StatusDuplicate}, // Duplicate of a.com
	}

	summary := Summarize(results)

	assert.Equal(t, 6, summary.Total)
	assert.Equal(t, 5, summary.UniqueURLs)
	assert.Equal(t, 1, summary.Alive)
	assert.Equal(t, 1, summary.Dead)
	assert.Equal(t, 1, summary.Redirects)
	assert.Equal(t, 1, summary.Blocked)
	assert.Equal(t, 1, summary.Errors)
	assert.Equal(t, 1, summary.Duplicates)
}

func TestSummary_HasIssues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		summary  Summary
		expected bool
	}{
		{
			name:     "NoIssues",
			summary:  Summary{Alive: 10},
			expected: false,
		},
		{
			name:     "HasRedirects",
			summary:  Summary{Alive: 10, Redirects: 1},
			expected: true,
		},
		{
			name:     "HasBlocked",
			summary:  Summary{Alive: 10, Blocked: 1},
			expected: true,
		},
		{
			name:     "HasDead",
			summary:  Summary{Alive: 10, Dead: 1},
			expected: true,
		},
		{
			name:     "HasErrors",
			summary:  Summary{Alive: 10, Errors: 1},
			expected: true,
		},
		{
			name:     "DuplicatesOnly",
			summary:  Summary{Alive: 10, Duplicates: 5},
			expected: false, // Duplicates are not issues
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.summary.HasIssues())
		})
	}
}

func TestSummary_HasDeadLinks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		summary  Summary
		expected bool
	}{
		{
			name:     "NoDeadLinks",
			summary:  Summary{Alive: 10, Redirects: 5},
			expected: false,
		},
		{
			name:     "HasDead",
			summary:  Summary{Alive: 10, Dead: 1},
			expected: true,
		},
		{
			name:     "HasErrors",
			summary:  Summary{Alive: 10, Errors: 1},
			expected: true,
		},
		{
			name:     "HasBoth",
			summary:  Summary{Dead: 3, Errors: 2},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.summary.HasDeadLinks())
		})
	}
}

func TestSummary_WarningsCount(t *testing.T) {
	t.Parallel()

	summary := Summary{Redirects: 3, Blocked: 2}
	assert.Equal(t, 5, summary.WarningsCount())
}

// =============================================================================
// HTTP Checker Tests (with httptest)
// =============================================================================

func TestChecker_CheckAll_200OK(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0))
	links := []Link{{URL: server.URL, FilePath: "test.md", Line: 1}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, StatusAlive, results[0].Status)
	assert.Equal(t, http.StatusOK, results[0].StatusCode)
}

func TestChecker_CheckAll_404NotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0))
	links := []Link{{URL: server.URL}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, StatusDead, results[0].Status)
	assert.Equal(t, http.StatusNotFound, results[0].StatusCode)
}

func TestChecker_CheckAll_500ServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0))
	links := []Link{{URL: server.URL}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, StatusDead, results[0].Status)
	assert.Equal(t, http.StatusInternalServerError, results[0].StatusCode)
}

func TestChecker_CheckAll_HeadFallbackToGet(t *testing.T) {
	t.Parallel()

	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0))
	links := []Link{{URL: server.URL}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, StatusAlive, results[0].Status)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&requestCount), int32(2)) // HEAD then GET
}

func TestChecker_CheckAll_Redirect301(t *testing.T) {
	t.Parallel()

	finalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer finalServer.Close()

	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, finalServer.URL, http.StatusMovedPermanently)
	}))
	defer redirectServer.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0))
	links := []Link{{URL: redirectServer.URL}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, StatusRedirect, results[0].Status)
	assert.Equal(t, http.StatusMovedPermanently, results[0].StatusCode)
	assert.Equal(t, finalServer.URL, results[0].FinalURL)
	assert.Len(t, results[0].RedirectChain, 1)
}

func TestChecker_CheckAll_Redirect302(t *testing.T) {
	t.Parallel()

	finalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer finalServer.Close()

	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, finalServer.URL, http.StatusFound)
	}))
	defer redirectServer.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0))
	links := []Link{{URL: redirectServer.URL}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, StatusRedirect, results[0].Status)
	assert.Equal(t, http.StatusFound, results[0].StatusCode)
}

func TestChecker_CheckAll_RedirectChain(t *testing.T) {
	t.Parallel()

	// Create 3 servers: A -> B -> C
	serverC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer serverC.Close()

	serverB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, serverC.URL, http.StatusMovedPermanently)
	}))
	defer serverB.Close()

	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, serverB.URL, http.StatusFound)
	}))
	defer serverA.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0))
	links := []Link{{URL: serverA.URL}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, StatusRedirect, results[0].Status)
	assert.Equal(t, serverC.URL, results[0].FinalURL)
	assert.Len(t, results[0].RedirectChain, 2) // A -> B, B -> C
}

func TestChecker_CheckAll_TooManyRedirects(t *testing.T) {
	t.Parallel()

	// Server that always redirects to itself
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, server.URL+"/next", http.StatusFound)
	}))
	defer server.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0).WithMaxRedirects(3))
	links := []Link{{URL: server.URL}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, StatusDead, results[0].Status)
	assert.Contains(t, results[0].Error, "too many redirects")
}

func TestChecker_CheckAll_RedirectToDead(t *testing.T) {
	t.Parallel()

	deadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer deadServer.Close()

	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, deadServer.URL, http.StatusMovedPermanently)
	}))
	defer redirectServer.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0))
	links := []Link{{URL: redirectServer.URL}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, StatusDead, results[0].Status)
}

func TestChecker_CheckAll_403Blocked(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0))
	links := []Link{{URL: server.URL}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, StatusBlocked, results[0].Status)
	assert.Equal(t, http.StatusForbidden, results[0].StatusCode)
}

func TestChecker_CheckAll_403ThenOKWithBrowserHeaders(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		// Check for browser-like user agent
		if strings.Contains(ua, "Mozilla") {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0))
	links := []Link{{URL: server.URL}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, StatusAlive, results[0].Status)
}

func TestChecker_CheckAll_Duplicates(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0))
	links := []Link{
		{URL: server.URL, FilePath: "a.md", Line: 1},
		{URL: server.URL, FilePath: "b.md", Line: 5},  // Duplicate
		{URL: server.URL, FilePath: "c.md", Line: 10}, // Duplicate
	}

	results := checker.CheckAll(links)

	require.Len(t, results, 3)

	// First should be the primary result
	var primary, duplicates int
	for _, r := range results {
		switch r.Status {
		case StatusAlive:
			primary++
		case StatusDuplicate:
			duplicates++
			assert.NotNil(t, r.DuplicateOf)
		}
	}
	assert.Equal(t, 1, primary)
	assert.Equal(t, 2, duplicates)
}

func TestChecker_CheckAll_MultipleDifferentURLs(t *testing.T) {
	t.Parallel()

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server2.Close()

	checker := New(DefaultOptions().WithConcurrency(2).WithMaxRetries(0))
	links := []Link{
		{URL: server1.URL},
		{URL: server2.URL},
	}

	results := checker.CheckAll(links)

	require.Len(t, results, 2)

	resultMap := map[string]Result{}
	for _, r := range results {
		resultMap[r.Link.URL] = r
	}

	assert.Equal(t, StatusAlive, resultMap[server1.URL].Status)
	assert.Equal(t, StatusDead, resultMap[server2.URL].Status)
}

func TestChecker_CheckAll_RetryOn5xx(t *testing.T) {
	t.Parallel()

	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(3))
	links := []Link{{URL: server.URL}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, StatusAlive, results[0].Status)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&attempts), int32(3))
}

func TestChecker_CheckAll_RetryOn429(t *testing.T) {
	t.Parallel()

	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(2))
	links := []Link{{URL: server.URL}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, StatusAlive, results[0].Status)
}

func TestChecker_CheckAll_NoRetryOn404(t *testing.T) {
	t.Parallel()

	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(3))
	links := []Link{{URL: server.URL}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, StatusDead, results[0].Status)
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts)) // No retries for 404
}

func TestChecker_Check_ContextCanceled(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(5 * time.Second) // Slow response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithTimeout(10 * time.Second))
	links := []Link{{URL: server.URL}}

	ctx, cancel := context.WithCancel(context.Background())

	resultChan := checker.Check(ctx, links)

	// Cancel after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	results := make([]Result, 0, 1)
	for r := range resultChan {
		results = append(results, r)
	}

	require.Len(t, results, 1)
	assert.Equal(t, StatusError, results[0].Status)
}

func TestChecker_CheckAll_EmptyLinks(t *testing.T) {
	t.Parallel()

	checker := New(DefaultOptions())
	results := checker.CheckAll(nil)
	assert.Empty(t, results)

	results = checker.CheckAll([]Link{})
	assert.Empty(t, results)
}

func TestChecker_CheckAll_PreservesLinkMetadata(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := New(DefaultOptions().WithConcurrency(1))
	links := []Link{{
		URL:      server.URL,
		FilePath: "README.md",
		Line:     42,
		Text:     "Click here",
	}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, "README.md", results[0].Link.FilePath)
	assert.Equal(t, 42, results[0].Link.Line)
	assert.Equal(t, "Click here", results[0].Link.Text)
}

func TestChecker_CheckAll_Timeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second) // Longer than timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithTimeout(100 * time.Millisecond).WithMaxRetries(0))
	links := []Link{{URL: server.URL}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, StatusError, results[0].Status)
	assert.NotEmpty(t, results[0].Error)
}

func TestChecker_CheckAll_InvalidURL(t *testing.T) {
	t.Parallel()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0))
	links := []Link{{URL: "not-a-valid-url"}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, StatusError, results[0].Status)
}

func TestChecker_CheckAll_ConnectionRefused(t *testing.T) {
	t.Parallel()

	// Port that nothing is listening on
	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0).WithTimeout(1 * time.Second))
	links := []Link{{URL: "http://127.0.0.1:59999"}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, StatusError, results[0].Status)
}

// =============================================================================
// Concurrency Tests
// =============================================================================

func TestChecker_CheckAll_Concurrency(t *testing.T) {
	t.Parallel()

	var activeRequests int32
	var maxConcurrent int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		current := atomic.AddInt32(&activeRequests, 1)
		for {
			old := atomic.LoadInt32(&maxConcurrent)
			if current <= old || atomic.CompareAndSwapInt32(&maxConcurrent, old, current) {
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
		atomic.AddInt32(&activeRequests, -1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create many unique URLs
	links := make([]Link, 20)
	for i := range links {
		links[i] = Link{URL: server.URL + "/" + string(rune('a'+i))}
	}

	checker := New(DefaultOptions().WithConcurrency(5).WithMaxRetries(0))
	results := checker.CheckAll(links)

	assert.Len(t, results, 20)
	assert.LessOrEqual(t, atomic.LoadInt32(&maxConcurrent), int32(5))
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestBackoffDelay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		attempt  int
		minDelay time.Duration
		maxDelay time.Duration
	}{
		{1, 1 * time.Second, 2 * time.Second},    // 1s + up to 25% jitter
		{2, 2 * time.Second, 3 * time.Second},    // 2s + up to 25% jitter
		{3, 4 * time.Second, 5 * time.Second},    // 4s + up to 25% jitter
		{10, 30 * time.Second, 38 * time.Second}, // Capped at 30s + jitter
	}

	for _, tt := range tests {
		delay := backoffDelay(tt.attempt)
		assert.GreaterOrEqual(t, delay, tt.minDelay)
		assert.LessOrEqual(t, delay, tt.maxDelay)
	}
}

func TestIsRetryable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		result   Result
		expected bool
	}{
		{
			name:     "ErrorStatus",
			result:   Result{Status: StatusError},
			expected: true,
		},
		{
			name:     "500Error",
			result:   Result{Status: StatusDead, StatusCode: 500},
			expected: true,
		},
		{
			name:     "503Error",
			result:   Result{Status: StatusDead, StatusCode: 503},
			expected: true,
		},
		{
			name:     "429TooManyRequests",
			result:   Result{Status: StatusDead, StatusCode: 429},
			expected: true,
		},
		{
			name:     "404NotFound",
			result:   Result{Status: StatusDead, StatusCode: 404},
			expected: false,
		},
		{
			name:     "200OK",
			result:   Result{Status: StatusAlive, StatusCode: 200},
			expected: false,
		},
		{
			name:     "403Forbidden",
			result:   Result{Status: StatusBlocked, StatusCode: 403},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, isRetryable(tt.result))
		})
	}
}

func TestResolveURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		baseURL  string
		ref      string
		expected string
		hasError bool
	}{
		{
			name:     "AbsoluteURL",
			baseURL:  "https://example.com/page",
			ref:      "https://other.com/target",
			expected: "https://other.com/target",
		},
		{
			name:     "RelativePath",
			baseURL:  "https://example.com/dir/page",
			ref:      "other.html",
			expected: "https://example.com/dir/other.html",
		},
		{
			name:     "AbsolutePath",
			baseURL:  "https://example.com/dir/page",
			ref:      "/root/file",
			expected: "https://example.com/root/file",
		},
		{
			name:     "ProtocolRelative",
			baseURL:  "https://example.com/page",
			ref:      "//other.com/target",
			expected: "https://other.com/target",
		},
		{
			name:     "InvalidBase",
			baseURL:  "://invalid",
			ref:      "/path",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := resolveURL(tt.baseURL, tt.ref)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// =============================================================================
// Status Code Tests
// =============================================================================

func TestChecker_CheckAll_VariousStatusCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code           int
		expectedStatus LinkStatus
	}{
		{200, StatusAlive},
		{201, StatusAlive},
		{204, StatusAlive},
		{400, StatusDead},
		{401, StatusDead},
		{404, StatusDead},
		{410, StatusDead},
		{500, StatusDead},
		{502, StatusDead},
		{503, StatusDead},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.code), func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.code)
			}))
			defer server.Close()

			checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0))
			links := []Link{{URL: server.URL}}

			results := checker.CheckAll(links)

			require.Len(t, results, 1)
			assert.Equal(t, tt.expectedStatus, results[0].Status)
			assert.Equal(t, tt.code, results[0].StatusCode)
		})
	}
}

// =============================================================================
// Additional Edge Case Tests
// =============================================================================

func TestChecker_CheckAll_RedirectTo403ThenOKWithBrowserHeaders(t *testing.T) {
	t.Parallel()

	// Final server that returns 403 normally but 200 with browser headers
	finalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if strings.Contains(ua, "Mozilla") {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusForbidden)
	}))
	defer finalServer.Close()

	// Redirect server that redirects to the final server
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, finalServer.URL, http.StatusMovedPermanently)
	}))
	defer redirectServer.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0))
	links := []Link{{URL: redirectServer.URL}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	// Should be StatusRedirect since final destination works with browser headers
	assert.Equal(t, StatusRedirect, results[0].Status)
	assert.Equal(t, finalServer.URL, results[0].FinalURL)
}

func TestChecker_CheckAll_RedirectTo403StillBlocked(t *testing.T) {
	t.Parallel()

	// Final server that always returns 403 even with browser headers
	finalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer finalServer.Close()

	// Redirect server
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, finalServer.URL, http.StatusMovedPermanently)
	}))
	defer redirectServer.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0))
	links := []Link{{URL: redirectServer.URL}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	// Should be StatusDead since final destination is blocked even with browser headers
	assert.Equal(t, StatusDead, results[0].Status)
}

func TestChecker_CheckAll_501NotImplemented(t *testing.T) {
	t.Parallel()

	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusNotImplemented) // 501
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0))
	links := []Link{{URL: server.URL}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, StatusAlive, results[0].Status)
	// Should have made at least 2 requests (HEAD then GET fallback)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&requestCount), int32(2))
}

func TestChecker_CheckAll_RedirectWithRelativeLocation(t *testing.T) {
	t.Parallel()

	// Server that redirects with a relative path
	mux := http.NewServeMux()
	mux.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/final", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/final", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0))
	links := []Link{{URL: server.URL + "/start"}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, StatusRedirect, results[0].Status)
	assert.Equal(t, server.URL+"/final", results[0].FinalURL)
	assert.Len(t, results[0].RedirectChain, 1)
}

func TestChecker_CheckAll_RedirectWithInvalidLocation(t *testing.T) {
	t.Parallel()

	// Server that returns a redirect with an invalid Location header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "://invalid-url")
		w.WriteHeader(http.StatusMovedPermanently)
	}))
	defer server.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0))
	links := []Link{{URL: server.URL}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	// The error comes from Go's HTTP client when parsing the Location header
	assert.True(t, results[0].Status == StatusDead || results[0].Status == StatusError)
	assert.NotEmpty(t, results[0].Error)
}

func TestResolveURL_InvalidRef(t *testing.T) {
	t.Parallel()

	// Test with a reference URL that can't be parsed
	_, err := resolveURL("https://example.com", "://invalid")
	assert.Error(t, err)
}

func TestResolveURL_EmptyRef(t *testing.T) {
	t.Parallel()

	result, err := resolveURL("https://example.com/path", "")
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/path", result)
}

func TestResolveURL_QueryStringOnly(t *testing.T) {
	t.Parallel()

	result, err := resolveURL("https://example.com/path", "?query=value")
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/path?query=value", result)
}

func TestResolveURL_FragmentOnly(t *testing.T) {
	t.Parallel()

	result, err := resolveURL("https://example.com/path", "#section")
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/path#section", result)
}

func TestBackoffDelay_ZeroAttempt(t *testing.T) {
	t.Parallel()

	// Edge case: attempt 0 should be treated as attempt 1
	delay := backoffDelay(0)
	assert.GreaterOrEqual(t, delay, 1*time.Second)
	assert.LessOrEqual(t, delay, 2*time.Second)
}

func TestChecker_CheckAll_RedirectToSameHost(t *testing.T) {
	t.Parallel()

	// Server with multiple redirects on same host
	mux := http.NewServeMux()
	mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/b", http.StatusFound)
	})
	mux.HandleFunc("/b", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/c", http.StatusFound)
	})
	mux.HandleFunc("/c", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0))
	links := []Link{{URL: server.URL + "/a"}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, StatusRedirect, results[0].Status)
	assert.Equal(t, server.URL+"/c", results[0].FinalURL)
	assert.Len(t, results[0].RedirectChain, 2)
}

func TestChecker_CheckAll_307TemporaryRedirect(t *testing.T) {
	t.Parallel()

	finalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer finalServer.Close()

	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, finalServer.URL, http.StatusTemporaryRedirect) // 307
	}))
	defer redirectServer.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0))
	links := []Link{{URL: redirectServer.URL}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, StatusRedirect, results[0].Status)
	assert.Equal(t, http.StatusTemporaryRedirect, results[0].StatusCode)
}

func TestChecker_CheckAll_308PermanentRedirect(t *testing.T) {
	t.Parallel()

	finalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer finalServer.Close()

	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, finalServer.URL, http.StatusPermanentRedirect) // 308
	}))
	defer redirectServer.Close()

	checker := New(DefaultOptions().WithConcurrency(1).WithMaxRetries(0))
	links := []Link{{URL: redirectServer.URL}}

	results := checker.CheckAll(links)

	require.Len(t, results, 1)
	assert.Equal(t, StatusRedirect, results[0].Status)
	assert.Equal(t, http.StatusPermanentRedirect, results[0].StatusCode)
}
