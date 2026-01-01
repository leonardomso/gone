package checker

import "time"

// Default values for checker options.
// These are tuned for optimal performance while maintaining reliability.
const (
	// DefaultConcurrency is the number of concurrent workers.
	// Higher values speed up checking but use more resources.
	DefaultConcurrency = 50

	// DefaultTimeout is the maximum time to wait for a single HTTP request.
	// Most healthy servers respond within 2-3 seconds.
	DefaultTimeout = 5 * time.Second

	// DefaultMaxRetries is the number of times to retry a failed request.
	// Lower values speed up checking but may miss transient failures.
	DefaultMaxRetries = 1

	// DefaultMaxRedirects is the maximum number of redirects to follow.
	// Most legitimate redirects are 1-2 hops.
	DefaultMaxRedirects = 5

	// DefaultUserAgent is the User-Agent header sent with requests.
	DefaultUserAgent = "gone-link-checker/1.0"
)

// Options configures the behavior of the link checker.
type Options struct {
	// UserAgent is the User-Agent header sent with requests.
	// Some servers block requests without a proper User-Agent.
	UserAgent string

	// Concurrency is the number of concurrent workers checking links.
	// Higher values = faster checking but more resource usage.
	Concurrency int

	// Timeout is the maximum time to wait for a single HTTP request.
	// This includes connection, TLS handshake, and response headers.
	Timeout time.Duration

	// MaxRetries is the number of times to retry a failed request.
	// Only transient errors (timeouts, 5xx, 429) are retried.
	MaxRetries int

	// MaxRedirects is the maximum number of redirects to follow.
	MaxRedirects int
}

// DefaultOptions returns optimized default configuration.
// These defaults prioritize speed while maintaining reasonable reliability.
func DefaultOptions() Options {
	return Options{
		Concurrency:  DefaultConcurrency,
		Timeout:      DefaultTimeout,
		MaxRetries:   DefaultMaxRetries,
		MaxRedirects: DefaultMaxRedirects,
		UserAgent:    DefaultUserAgent,
	}
}

// WithConcurrency sets the number of concurrent workers.
func (o Options) WithConcurrency(n int) Options {
	if n > 0 {
		o.Concurrency = n
	}
	return o
}

// WithTimeout sets the request timeout.
func (o Options) WithTimeout(d time.Duration) Options {
	if d > 0 {
		o.Timeout = d
	}
	return o
}

// WithMaxRetries sets the maximum retry count.
func (o Options) WithMaxRetries(n int) Options {
	if n >= 0 {
		o.MaxRetries = n
	}
	return o
}

// WithMaxRedirects sets the maximum number of redirects to follow.
func (o Options) WithMaxRedirects(n int) Options {
	if n > 0 {
		o.MaxRedirects = n
	}
	return o
}

// WithUserAgent sets the User-Agent header.
func (o Options) WithUserAgent(ua string) Options {
	if ua != "" {
		o.UserAgent = ua
	}
	return o
}

// BrowserUserAgent is a realistic browser User-Agent for bypassing bot detection.
const BrowserUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
