package checker

import "time"

// Options configures the behavior of the link checker.
type Options struct {

	// UserAgent is the User-Agent header sent with requests.
	// Some servers block requests without a proper User-Agent.
	// Default: "gone-link-checker/1.0"
	UserAgent string
	// Concurrency is the number of concurrent workers checking links.
	// Higher values = faster checking but more resource usage.
	// Default: 10
	Concurrency int

	// Timeout is the maximum time to wait for a single HTTP request.
	// This includes connection, TLS handshake, and response headers.
	// Default: 10s
	Timeout time.Duration

	// MaxRetries is the number of times to retry a failed request.
	// Only transient errors (timeouts, 5xx, 429) are retried.
	// Default: 2
	MaxRetries int

	// MaxRedirects is the maximum number of redirects to follow.
	// Default: 10
	MaxRedirects int
}

// DefaultOptions returns sensible default configuration.
func DefaultOptions() Options {
	return Options{
		Concurrency:  10,
		Timeout:      10 * time.Second,
		MaxRetries:   2,
		MaxRedirects: 10,
		UserAgent:    "gone-link-checker/1.0",
	}
}

// WithConcurrency sets the number of concurrent workers.
func (o Options) WithConcurrency(n int) Options {
	o.Concurrency = n
	return o
}

// WithTimeout sets the request timeout.
func (o Options) WithTimeout(d time.Duration) Options {
	o.Timeout = d
	return o
}

// WithMaxRetries sets the maximum retry count.
func (o Options) WithMaxRetries(n int) Options {
	o.MaxRetries = n
	return o
}

// WithMaxRedirects sets the maximum number of redirects to follow.
func (o Options) WithMaxRedirects(n int) Options {
	o.MaxRedirects = n
	return o
}

// WithUserAgent sets the User-Agent header.
func (o Options) WithUserAgent(ua string) Options {
	o.UserAgent = ua
	return o
}

// BrowserUserAgent is a realistic browser User-Agent for bypassing bot detection.
const BrowserUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
