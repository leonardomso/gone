// Package checker verifies if URLs are alive by making HTTP requests.
// It uses a worker pool pattern for bounded concurrency and includes
// retry logic with exponential backoff for transient failures.
package checker

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"sync"
	"time"
)

// Checker performs concurrent link checking with configurable options.
type Checker struct {
	opts   Options
	client *http.Client
}

// New creates a new Checker with the given options.
func New(opts Options) *Checker {
	return &Checker{
		opts:   opts,
		client: newHTTPClient(opts),
	}
}

// newHTTPClient creates an optimized HTTP client with proper timeouts
// and connection pooling.
func newHTTPClient(opts Options) *http.Client {
	transport := &http.Transport{
		// Connection pooling - reuse connections for efficiency
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     20,
		IdleConnTimeout:     90 * time.Second,

		// TLS configuration with minimum version for security
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		},

		// Timeout layers for different phases
		DialContext: (&net.Dialer{
			Timeout:   opts.Timeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: opts.Timeout,
		ExpectContinueTimeout: 1 * time.Second,

		// Enable compression
		DisableCompression: false,
	}

	return &http.Client{
		Timeout:   opts.Timeout,
		Transport: transport,
		// Don't follow redirects - we want to see the actual status
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// CheckAll checks all links and returns results after all are complete.
// This is a blocking operation.
func (c *Checker) CheckAll(links []Link) []Result {
	results := make([]Result, 0, len(links))
	for result := range c.Check(context.Background(), links) {
		results = append(results, result)
	}
	return results
}

// Check checks links concurrently using a worker pool and streams results.
// The returned channel will be closed when all links have been checked.
// Use the context to cancel ongoing checks.
func (c *Checker) Check(ctx context.Context, links []Link) <-chan Result {
	results := make(chan Result, c.opts.Concurrency)

	go func() {
		defer close(results)

		// Create job queue
		jobs := make(chan Link, len(links))

		// Start worker pool
		var wg sync.WaitGroup
		for i := 0; i < c.opts.Concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				c.worker(ctx, jobs, results)
			}()
		}

		// Send jobs to workers
	sendLoop:
		for _, link := range links {
			select {
			case jobs <- link:
			case <-ctx.Done():
				// Context canceled, stop sending jobs
				break sendLoop
			}
		}
		close(jobs)

		// Wait for all workers to finish
		wg.Wait()
	}()

	return results
}

// worker processes links from the jobs channel and sends results.
func (c *Checker) worker(ctx context.Context, jobs <-chan Link, results chan<- Result) {
	for link := range jobs {
		select {
		case <-ctx.Done():
			// Context canceled, report as error
			results <- Result{
				Link:  link,
				Error: "check canceled",
			}
		default:
			result := c.checkWithRetry(ctx, link)
			results <- result
		}
	}
}

// checkWithRetry attempts to check a link with exponential backoff retry.
func (c *Checker) checkWithRetry(ctx context.Context, link Link) Result {
	var lastResult Result

	for attempt := 0; attempt <= c.opts.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			delay := backoffDelay(attempt)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return Result{
					Link:  link,
					Error: "check canceled during retry",
				}
			}
		}

		result := c.checkSingle(ctx, link)

		// Success - return immediately
		if result.IsAlive {
			return result
		}

		// Check if error is retryable
		if !isRetryable(result) {
			return result
		}

		lastResult = result
	}

	// All retries exhausted
	if lastResult.Error != "" {
		lastResult.Error = fmt.Sprintf("%s (after %d retries)", lastResult.Error, c.opts.MaxRetries)
	}
	return lastResult
}

// backoffDelay calculates delay for retry with exponential backoff and jitter.
func backoffDelay(attempt int) time.Duration {
	// Base delay: 1s, 2s, 4s, etc.
	// Safe conversion: attempt is small (0-10 range typically)
	if attempt < 1 {
		attempt = 1
	}
	base := time.Second * time.Duration(1<<uint(attempt-1)) //nolint:gosec // attempt is bounded

	// Cap at 30 seconds
	if base > 30*time.Second {
		base = 30 * time.Second
	}

	// Add jitter (0-25% of base) using crypto/rand for security
	maxJitter := int64(base / 4)
	if maxJitter > 0 {
		n, err := rand.Int(rand.Reader, big.NewInt(maxJitter))
		if err == nil {
			return base + time.Duration(n.Int64())
		}
	}

	return base
}

// isRetryable determines if a result should trigger a retry.
func isRetryable(result Result) bool {
	// Retry on network errors (timeout, connection refused, etc.)
	if result.Error != "" {
		return true
	}

	// Retry on server errors (5xx)
	if result.StatusCode >= 500 && result.StatusCode < 600 {
		return true
	}

	// Retry on rate limiting (429 Too Many Requests)
	if result.StatusCode == http.StatusTooManyRequests {
		return true
	}

	return false
}

// checkSingle performs a single check on a link (HEAD with GET fallback).
func (c *Checker) checkSingle(ctx context.Context, link Link) Result {
	result := Result{Link: link}

	// Try HEAD first (faster, no body)
	statusCode, err := c.doRequest(ctx, http.MethodHead, link.URL)

	// If HEAD fails with 405 or 501, try GET
	if statusCode == http.StatusMethodNotAllowed || statusCode == http.StatusNotImplemented {
		statusCode, err = c.doRequest(ctx, http.MethodGet, link.URL)
	}

	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.StatusCode = statusCode
	result.IsAlive = c.isAlive(statusCode)

	return result
}

// doRequest performs an HTTP request and returns the status code.
func (c *Checker) doRequest(ctx context.Context, method, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, http.NoBody)
	if err != nil {
		return 0, err
	}

	// Set headers
	req.Header.Set("User-Agent", c.opts.UserAgent)
	req.Header.Set("Accept", "*/*")

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// For GET requests, drain body to allow connection reuse
	// Limit read to prevent memory issues with large responses
	if method == http.MethodGet {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024*1024)) // 1MB max
	}

	return resp.StatusCode, nil
}

// isAlive determines if a status code indicates the link is alive.
func (c *Checker) isAlive(statusCode int) bool {
	// 2xx are always alive
	if statusCode >= 200 && statusCode < 300 {
		return true
	}

	// 3xx are alive only if FollowRedirects is enabled
	if statusCode >= 300 && statusCode < 400 {
		return c.opts.FollowRedirects
	}

	return false
}
