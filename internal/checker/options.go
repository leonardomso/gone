package checker

import "time"

// Options configures the behavior of the link checker.
type Options struct {
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

	// FollowRedirects determines if 3xx responses are considered alive.
	// If false, redirects are reported as dead links.
	// Default: false
	FollowRedirects bool

	// UserAgent is the User-Agent header sent with requests.
	// Some servers block requests without a proper User-Agent.
	// Default: "gone-link-checker/1.0"
	UserAgent string
}

// DefaultOptions returns sensible default configuration.
func DefaultOptions() Options {
	return Options{
		Concurrency:     10,
		Timeout:         10 * time.Second,
		MaxRetries:      2,
		FollowRedirects: false,
		UserAgent:       "gone-link-checker/1.0",
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

// WithFollowRedirects sets whether redirects are considered alive.
func (o Options) WithFollowRedirects(follow bool) Options {
	o.FollowRedirects = follow
	return o
}

// WithUserAgent sets the User-Agent header.
func (o Options) WithUserAgent(ua string) Options {
	o.UserAgent = ua
	return o
}
