// Package checker verifies if URLs are alive by making HTTP requests.
// It uses a worker pool pattern for bounded concurrency and includes
// retry logic with exponential backoff for transient failures.
package checker

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Checker performs concurrent link checking with configurable options.
type Checker struct {
	client *http.Client
	opts   Options
}

// New creates a new Checker with the given options.
func New(opts Options) *Checker {
	return &Checker{
		opts:   opts,
		client: newHTTPClient(opts),
	}
}

// newHTTPClient creates an optimized HTTP client with proper timeouts
// and connection pooling. This client does NOT follow redirects automatically.
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
		// Don't follow redirects - we handle them manually to track the chain
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
// URLs are deduplicated - each unique URL is checked once, with duplicate
// occurrences reported as StatusDuplicate.
// The returned channel will be closed when all links have been checked.
// Use the context to cancel ongoing checks.
func (c *Checker) Check(ctx context.Context, links []Link) <-chan Result {
	results := make(chan Result, c.opts.Concurrency)

	go func() {
		defer close(results)

		// Deduplicate: group links by URL
		urlToLinks := map[string][]Link{}
		var urlOrder []string // Preserve order for deterministic output
		for _, link := range links {
			if _, exists := urlToLinks[link.URL]; !exists {
				urlOrder = append(urlOrder, link.URL)
			}
			urlToLinks[link.URL] = append(urlToLinks[link.URL], link)
		}

		// Create job queue with unique URLs only (first occurrence of each)
		uniqueLinks := make([]Link, 0, len(urlOrder))
		for _, u := range urlOrder {
			uniqueLinks = append(uniqueLinks, urlToLinks[u][0])
		}

		// Store primary results for duplicates
		primaryResults := map[string]*Result{}
		var resultsMu sync.Mutex

		// Channel for primary results from workers
		primaryChan := make(chan Result, c.opts.Concurrency)

		// Start worker pool
		var wg sync.WaitGroup
		jobs := make(chan Link, len(uniqueLinks))

		for range c.opts.Concurrency {
			wg.Go(func() {
				for link := range jobs {
					select {
					case <-ctx.Done():
						primaryChan <- Result{
							Link:   link,
							Status: StatusError,
							Error:  "check canceled",
						}
					default:
						result := c.checkWithRetry(ctx, link)
						primaryChan <- result
					}
				}
			})
		}

		// Send jobs to workers
		go func() {
		sendLoop:
			for _, link := range uniqueLinks {
				select {
				case jobs <- link:
				case <-ctx.Done():
					break sendLoop
				}
			}
			close(jobs)
		}()

		// Collect primary results and emit all occurrences
		go func() {
			wg.Wait()
			close(primaryChan)
		}()

		for result := range primaryChan {
			// Store as primary result
			resultsMu.Lock()
			resultCopy := result
			primaryResults[result.Link.URL] = &resultCopy
			resultsMu.Unlock()

			// Get all occurrences of this URL
			occurrences := urlToLinks[result.Link.URL]

			// Emit primary result (first occurrence)
			results <- result

			// Emit duplicate results for additional occurrences
			for i := 1; i < len(occurrences); i++ {
				dupResult := Result{
					Link:        occurrences[i],
					StatusCode:  result.StatusCode,
					Status:      StatusDuplicate,
					DuplicateOf: &resultCopy,
				}
				results <- dupResult
			}
		}
	}()

	return results
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
					Link:   link,
					Status: StatusError,
					Error:  "check canceled during retry",
				}
			}
		}

		result := c.checkSingle(ctx, link)

		// Success or non-retryable - return immediately
		if result.Status == StatusAlive || !isRetryable(result) {
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
	attempt = max(attempt, 1)
	base := time.Second * time.Duration(1<<uint(attempt-1)) //nolint:gosec // attempt is bounded

	// Cap at 30 seconds
	base = min(base, 30*time.Second)

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
	if result.Status == StatusError {
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
	statusCode, err := c.doRequest(ctx, http.MethodHead, link.URL, false)

	// If HEAD fails with 405 or 501, try GET
	if statusCode == http.StatusMethodNotAllowed || statusCode == http.StatusNotImplemented {
		statusCode, err = c.doRequest(ctx, http.MethodGet, link.URL, false)
	}

	if err != nil {
		result.Status = StatusError
		result.Error = err.Error()
		return result
	}

	result.StatusCode = statusCode

	// Determine status based on response code
	switch {
	case statusCode >= 200 && statusCode < 300:
		// 2xx - alive
		result.Status = StatusAlive

	case statusCode >= 300 && statusCode < 400:
		// 3xx - follow redirect chain
		chain, finalURL, finalStatus, err := c.followRedirectChain(ctx, link.URL)
		result.RedirectChain = chain
		result.FinalURL = finalURL
		result.FinalStatus = finalStatus

		switch {
		case err != nil:
			result.Status = StatusDead
			result.Error = err.Error()
		case finalStatus >= 200 && finalStatus < 300:
			result.Status = StatusRedirect // Warning - redirect works
		case finalStatus == 403:
			// Final destination is blocked, try with browser headers
			result.Status = c.handleBlockedFinal(ctx, finalURL, &result)
		default:
			result.Status = StatusDead // Redirect leads to dead page
		}

	case statusCode == 403:
		// 403 - try again with browser-like headers
		result.Status = c.handleBlocked(ctx, link.URL, &result)

	default:
		// 4xx (except 403), 5xx - dead
		result.Status = StatusDead
	}

	return result
}

// handleBlocked tries to access a 403 URL with browser-like headers.
func (c *Checker) handleBlocked(ctx context.Context, urlStr string, result *Result) LinkStatus {
	// Retry with browser-like headers
	statusCode, err := c.doRequest(ctx, http.MethodGet, urlStr, true)
	if err == nil && statusCode >= 200 && statusCode < 300 {
		// It was just blocking our bot UA
		result.StatusCode = statusCode
		return StatusAlive
	}
	// Still blocked
	return StatusBlocked
}

// handleBlockedFinal handles a redirect chain that ends in 403.
func (c *Checker) handleBlockedFinal(ctx context.Context, finalURL string, result *Result) LinkStatus {
	statusCode, err := c.doRequest(ctx, http.MethodGet, finalURL, true)
	if err == nil && statusCode >= 200 && statusCode < 300 {
		result.FinalStatus = statusCode
		return StatusRedirect // Redirect works with browser headers
	}
	return StatusDead // Even with browser headers, it's dead
}

// followRedirectChain follows redirects and returns the chain, final URL, and final status.
//
//nolint:gocritic // Named returns would make this function harder to read
func (c *Checker) followRedirectChain(ctx context.Context, startURL string) ([]Redirect, string, int, error) {
	var chain []Redirect
	currentURL := startURL

	for i := 0; i < c.opts.MaxRedirects; i++ {
		statusCode, location, err := c.doRequestGetLocation(ctx, currentURL)
		if err != nil {
			return chain, currentURL, 0, err
		}

		// Not a redirect - we've reached the final destination
		if statusCode < 300 || statusCode >= 400 {
			return chain, currentURL, statusCode, nil
		}

		// It's a redirect - record it and continue
		chain = append(chain, Redirect{URL: currentURL, StatusCode: statusCode})

		// Resolve relative URLs
		nextURL, err := resolveURL(currentURL, location)
		if err != nil {
			return chain, currentURL, 0, fmt.Errorf("invalid redirect location: %w", err)
		}
		currentURL = nextURL
	}

	return chain, currentURL, 0, errors.New("too many redirects")
}

// resolveURL resolves a potentially relative URL against a base URL.
func resolveURL(baseURL, ref string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		return "", err
	}
	return base.ResolveReference(refURL).String(), nil
}

// doRequest performs an HTTP request and returns the status code.
func (c *Checker) doRequest(ctx context.Context, method, urlStr string, useBrowserHeaders bool) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, urlStr, http.NoBody)
	if err != nil {
		return 0, err
	}

	// Set headers
	if useBrowserHeaders {
		c.setBrowserHeaders(req)
	} else {
		req.Header.Set("User-Agent", c.opts.UserAgent)
		req.Header.Set("Accept", "*/*")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// For GET requests, drain body to allow connection reuse
	if method == http.MethodGet {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024*1024)) // 1MB max
	}

	return resp.StatusCode, nil
}

// doRequestGetLocation performs a request and returns status code and Location header.
//
//nolint:gocritic // Named returns would make this function harder to read
func (c *Checker) doRequestGetLocation(ctx context.Context, urlStr string) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, urlStr, http.NoBody)
	if err != nil {
		return 0, "", err
	}

	req.Header.Set("User-Agent", c.opts.UserAgent)
	req.Header.Set("Accept", "*/*")

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	return resp.StatusCode, resp.Header.Get("Location"), nil
}

// setBrowserHeaders sets headers that mimic a real browser to bypass bot detection.
func (*Checker) setBrowserHeaders(req *http.Request) {
	req.Header.Set("User-Agent", BrowserUserAgent)
	req.Header.Set("Accept",
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Cache-Control", "max-age=0")
}
