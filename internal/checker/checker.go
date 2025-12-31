// Package checker verifies if URLs are alive by making HTTP requests
package checker

import (
	"net/http"
	"sync"
	"time"

	"gone/internal/parser"
)

// Result represents the outcome of checking a single link
type Result struct {
	Link       parser.Link // The original link info
	StatusCode int         // HTTP status code (0 if network error)
	IsAlive    bool        // true if status is 200
	Error      string      // Error message if request failed
}

// CheckLinks checks multiple URLs concurrently and returns results
// This is where Go's concurrency shines!
func CheckLinks(links []parser.Link) []Result {
	// Create a channel to receive results
	// Channels are Go's way for goroutines to communicate
	// The buffer size (len(links)) prevents blocking
	resultsChan := make(chan Result, len(links))

	// WaitGroup tracks when all goroutines are done
	var wg sync.WaitGroup

	// Create an HTTP client with a timeout
	// We reuse one client for efficiency (connection pooling)
	client := &http.Client{
		Timeout: 10 * time.Second,
		// Don't follow redirects automatically - we want to see 301/302
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Launch a goroutine for each link
	for _, link := range links {
		wg.Add(1) // Increment the WaitGroup counter

		// The 'go' keyword launches a goroutine (lightweight thread)
		// We pass 'link' as a parameter to avoid closure issues
		go func(l parser.Link) {
			defer wg.Done() // Decrement counter when this goroutine finishes

			result := checkSingleLink(client, l)
			resultsChan <- result // Send result to channel
		}(link)
	}

	// Wait for all goroutines to complete, then close the channel
	// We do this in a separate goroutine so we can start reading results immediately
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect all results from the channel
	var results []Result
	for result := range resultsChan {
		results = append(results, result)
	}

	return results
}

// checkSingleLink makes an HTTP request to verify a URL is alive
func checkSingleLink(client *http.Client, link parser.Link) Result {
	result := Result{
		Link: link,
	}

	// Use HEAD request first (faster, no body downloaded)
	req, err := http.NewRequest("HEAD", link.URL, nil)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	// Set a User-Agent to avoid being blocked by some servers
	req.Header.Set("User-Agent", "gone-link-checker/1.0")

	resp, err := client.Do(req)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close() // Always close the response body!

	result.StatusCode = resp.StatusCode
	result.IsAlive = resp.StatusCode == 200

	return result
}

// FilterDeadLinks returns only the links that are not alive
func FilterDeadLinks(results []Result) []Result {
	var dead []Result
	for _, r := range results {
		if !r.IsAlive {
			dead = append(dead, r)
		}
	}
	return dead
}

// CheckLinksAsync checks links and sends results to a channel as they complete
// This is useful for updating a UI in real-time
// The caller is responsible for reading from the returned channel
func CheckLinksAsync(links []parser.Link) <-chan Result {
	// The <-chan syntax means "receive-only channel"
	// This prevents the caller from accidentally sending to it
	resultsChan := make(chan Result, len(links))

	go func() {
		var wg sync.WaitGroup
		client := &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		for _, link := range links {
			wg.Add(1)
			go func(l parser.Link) {
				defer wg.Done()
				result := checkSingleLink(client, l)
				resultsChan <- result
			}(link)
		}

		wg.Wait()
		close(resultsChan)
	}()

	return resultsChan
}
