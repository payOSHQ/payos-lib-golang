package payos

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/payOSHQ/payos-lib-golang/internal"
	"github.com/payOSHQ/payos-lib-golang/internal/apierror"
	"github.com/payOSHQ/payos-lib-golang/internal/crypto"
)

const (
	PayOSBaseUrl    = "https://api-merchant.payos.vn"
	defaultTimeout  = 60 * time.Second
	defaultMaxRetry = 2
)

// Middleware defines a function that wraps HTTP request execution
// It receives the next handler and returns a new handler
type Middleware func(next RequestHandler) RequestHandler

// RequestHandler is a function that executes an HTTP request
type RequestHandler func(ctx context.Context, req *http.Request) (*http.Response, error)

// Client is the PayOS API client
type Client struct {
	clientId    string
	apiKey      string
	checksumKey string
	partnerCode string
	baseURL     string
	httpClient  *http.Client
	maxRetries  int
	timeout     time.Duration
	middlewares []Middleware
}

// NewClient creates a new PayOS client with the provided options
func NewClient(opts *PayOSOptions) (*Client, error) {
	opts = NewPayOSOptions(opts)

	// Validate required fields
	if opts.ClientId == "" {
		return nil, apierror.NewPayOSError("The PAYOS_CLIENT_ID environment variable is missing or empty; either provide it, or instantiate the PayOS client with a ClientId option.")
	}
	if opts.ApiKey == "" {
		return nil, apierror.NewPayOSError("The PAYOS_API_KEY environment variable is missing or empty; either provide it, or instantiate the PayOS client with a ApiKey option.")
	}
	if opts.ChecksumKey == "" {
		return nil, apierror.NewPayOSError("The PAYOS_CHECKSUM_KEY environment variable is missing or empty; either provide it, or instantiate the PayOS client with a ChecksumKey option.")
	}

	// Create HTTP client if not provided
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: opts.Timeout,
		}
	}

	// Prepare middlewares with debug logger if enabled
	middlewares := opts.Middlewares
	if opts.DebugLogger != nil {
		// Prepend debug logger middleware as the first (outermost) middleware
		debugMiddleware := createDebugLoggerMiddleware(opts.DebugLogger)
		middlewares = append([]Middleware{debugMiddleware}, middlewares...)
	}

	return &Client{
		clientId:    opts.ClientId,
		apiKey:      opts.ApiKey,
		checksumKey: opts.ChecksumKey,
		partnerCode: opts.PartnerCode,
		baseURL:     opts.BaseURL,
		httpClient:  httpClient,
		maxRetries:  opts.MaxRetries,
		timeout:     opts.Timeout,
		middlewares: middlewares,
	}, nil
}

// SignatureOpts contains signature options for requests and responses
type SignatureOpts struct {
	// Request signature type: "create-payment-link", "body", or "header"
	Request string
	// Response signature type: "body" or "header"
	Response string
}

// RequestOptions contains options for making API requests
type RequestOptions struct {
	Method        string
	Path          string
	Query         map[string]interface{}
	Body          interface{}
	Headers       map[string]string
	SignatureOpts *SignatureOpts
}

func (c *Client) getUserAgent() string {
	return fmt.Sprintf("PayOS/Go %s", internal.PackageVersion)
}

// buildURL constructs the full URL with query parameters
func (c *Client) buildURL(path string, query map[string]interface{}) (string, error) {
	baseURL, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}

	endpoint, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	fullURL := baseURL.ResolveReference(endpoint)

	if len(query) > 0 {
		q := fullURL.Query()
		for key, val := range query {
			if val == nil {
				continue
			}
			switch v := val.(type) {
			case string:
				q.Add(key, v)
			case int, int64, float64:
				q.Add(key, fmt.Sprintf("%v", v))
			default:
				jsonVal, _ := json.Marshal(v)
				q.Add(key, string(jsonVal))
			}
		}
		fullURL.RawQuery = q.Encode()
	}

	return fullURL.String(), nil
}

// buildHeaders constructs HTTP headers for the request
func (c *Client) buildHeaders(additional map[string]string) http.Header {
	headers := http.Header{}
	headers.Set("x-client-id", c.clientId)
	headers.Set("x-api-key", c.apiKey)
	headers.Set("Content-Type", "application/json")
	headers.Set("User-Agent", c.getUserAgent())

	if c.partnerCode != "" {
		headers.Set("x-partner-code", c.partnerCode)
	}

	for key, value := range additional {
		headers.Set(key, value)
	}

	return headers
}

// buildMiddlewareChain builds the middleware chain with the base HTTP handler
func (c *Client) buildMiddlewareChain() RequestHandler {
	// Base handler that actually executes the HTTP request
	baseHandler := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		return c.httpClient.Do(req)
	}

	// Wrap base handler with middlewares in reverse order
	// so the first middleware in the slice is the outermost
	handler := baseHandler
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		handler = c.middlewares[i](handler)
	}

	return handler
}

// createDebugLoggerMiddleware creates a middleware for debug logging
func createDebugLoggerMiddleware(logger *log.Logger) Middleware {
	// Use default logger if nil is passed
	if logger == nil {
		logger = log.New(os.Stderr, "[PayOS] ", log.LstdFlags)
	}

	return func(next RequestHandler) RequestHandler {
		return func(ctx context.Context, req *http.Request) (*http.Response, error) {
			start := time.Now()

			// Dump request
			requestDump, err := httputil.DumpRequestOut(req, true)
			if err != nil {
				logger.Printf("Failed to dump request: %v", err)
			} else {
				logger.Printf("Request:\n%s", string(requestDump))
			}

			// Execute request
			resp, err := next(ctx, req)

			duration := time.Since(start)

			if err != nil {
				logger.Printf("Request failed after %v: %v", duration, err)
				return resp, err
			}

			// Dump response
			responseDump, dumpErr := httputil.DumpResponse(resp, true)
			if dumpErr != nil {
				logger.Printf("Response: %d %s in %v (failed to dump body: %v)",
					resp.StatusCode, http.StatusText(resp.StatusCode), duration, dumpErr)
			} else {
				logger.Printf("Response (in %v):\n%s", duration, string(responseDump))
			}

			return resp, nil
		}
	}
}

// shouldRetry determines if a request should be retried based on the response
func (c *Client) shouldRetry(statusCode int) bool {
	return statusCode == 408 || statusCode == 429 || statusCode >= 500
}

// calculateBackoff calculates the backoff duration for retry attempts
func (c *Client) calculateBackoff(attempt int, headers http.Header) time.Duration {
	// Check for Retry-After header
	if retryAfter := headers.Get("Retry-After"); retryAfter != "" {
		if seconds, err := strconv.ParseFloat(retryAfter, 64); err == nil {
			return time.Duration(seconds * float64(time.Second))
		}
		if retryTime, err := time.Parse(time.RFC1123, retryAfter); err == nil {
			duration := time.Until(retryTime)
			if duration > 0 && duration < 60*time.Second {
				return duration
			}
		}
	}

	// Check for X-RateLimit-Reset header
	if rateLimitReset := headers.Get("X-RateLimit-Reset"); rateLimitReset != "" {
		if timestamp, err := strconv.ParseFloat(rateLimitReset, 64); err == nil {
			duration := time.Until(time.Unix(int64(timestamp), 0))
			if duration > 0 && duration < 60*time.Second {
				return duration
			}
		}
	}

	// Exponential backoff with jitter
	const (
		initRetryDelay = 0.5  // seconds
		maxRetryDelay  = 10.0 // seconds
	)

	sleepSeconds := math.Min(initRetryDelay*math.Pow(2, float64(attempt)), maxRetryDelay)
	jitter := 1 - rand.Float64()*0.25 // 75% to 100%
	return time.Duration(sleepSeconds * jitter * float64(time.Second))
}

// Request makes an HTTP request with retry logic
func (c *Client) Request(ctx context.Context, opts *RequestOptions) (any, error) {
	if opts == nil {
		return nil, errors.New("request options cannot be nil")
	}

	var lastErr error
	maxAttempts := c.maxRetries + 1

	for attempt := 0; attempt < maxAttempts; attempt++ {
		result, err := c.executeRequest(ctx, opts, attempt)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if we should retry
		if attempt >= c.maxRetries {
			break
		}

		// Determine if error is retryable
		shouldRetry := false
		var backoffHeaders http.Header

		// Check for specific retryable error types
		var tooManyReqErr *apierror.TooManyRequestError
		var internalServerErr *apierror.InternalServerError
		var apiErr *apierror.APIError

		if errors.As(err, &tooManyReqErr) {
			shouldRetry = true
			backoffHeaders = tooManyReqErr.Headers
		} else if errors.As(err, &internalServerErr) {
			shouldRetry = true
			backoffHeaders = internalServerErr.Headers
		} else if errors.As(err, &apiErr) {
			shouldRetry = c.shouldRetry(apiErr.StatusCode)
			backoffHeaders = apiErr.Headers
		} else {
			// Network errors are retryable
			var connErr *apierror.ConnectionError
			var timeoutErr *apierror.ConnectionTimeoutError
			if errors.As(err, &connErr) || errors.As(err, &timeoutErr) {
				shouldRetry = true
			}
		}

		if !shouldRetry {
			break
		}

		// Calculate backoff
		backoff := c.calculateBackoff(attempt, backoffHeaders)

		// Wait before retry
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			// Continue to next attempt
		}
	}

	return nil, lastErr
}

// executeRequest performs a single HTTP request
func (c *Client) executeRequest(ctx context.Context, opts *RequestOptions, attempt int) (any, error) {
	// Build URL
	fullURL, err := c.buildURL(opts.Path, opts.Query)
	if err != nil {
		return nil, apierror.NewConnectionError("failed to build URL", err)
	}

	// Prepare request body
	var bodyReader io.Reader
	var bodyData interface{} = opts.Body

	// Handle signature for request
	if opts.SignatureOpts != nil && opts.SignatureOpts.Request != "" && opts.Body != nil {
		switch opts.SignatureOpts.Request {
		case "create-payment-link":
			signature, err := crypto.CreateSignatureOfPaymentRequest(opts.Body, c.checksumKey)
			if err != nil {
				return nil, apierror.NewInvalidSignatureError("failed to create payment signature")
			}
			// Add signature to the body
			// Since CreatePaymentLinkRequest is an alias, we can use it directly
			if req, ok := opts.Body.(CreatePaymentLinkRequest); ok {
				req.Signature = &signature
				bodyData = req
			} else if bodyMap, ok := opts.Body.(map[string]interface{}); ok {
				bodyMap["signature"] = signature
				bodyData = bodyMap
			}
		case "body":
			signature, err := crypto.CreateSignatureFromObj(opts.Body, c.checksumKey)
			if err != nil {
				return nil, apierror.NewInvalidSignatureError("failed to create body signature")
			}
			// Add signature to body
			if bodyMap, ok := opts.Body.(map[string]interface{}); ok {
				bodyMap["signature"] = signature
				bodyData = bodyMap
			}
		case "header":
			signature, err := crypto.CreateSignature(c.checksumKey, opts.Body, nil)
			if err != nil {
				return nil, apierror.NewInvalidSignatureError("failed to create header signature")
			}
			if opts.Headers == nil {
				opts.Headers = make(map[string]string)
			}
			opts.Headers["x-signature"] = signature
		}
	}

	// Serialize body
	if bodyData != nil {
		bodyBytes, err := json.Marshal(bodyData)
		if err != nil {
			return nil, apierror.NewPayOSError("failed to marshal request body")
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, opts.Method, fullURL, bodyReader)
	if err != nil {
		return nil, apierror.NewConnectionError("failed to create request", err)
	}

	// Set headers
	req.Header = c.buildHeaders(opts.Headers)

	// Build and execute middleware chain
	handler := c.buildMiddlewareChain()
	resp, err := handler(ctx, req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, apierror.NewConnectionTimeoutError("request cancelled or timed out")
		}
		return nil, apierror.NewConnectionError("request failed", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apierror.NewConnectionError("failed to read response body", err)
	}

	// Only parse response for 200 status code
	if resp.StatusCode != http.StatusOK {
		// Try to parse error response
		var apiResp PayOSResponseType
		var errCode, errDesc string
		if err := json.Unmarshal(respBody, &apiResp); err == nil {
			errCode = apiResp.Code
			errDesc = apiResp.Desc
		} else {
			errDesc = string(respBody)
		}
		return nil, apierror.GenerateError(resp.StatusCode, errCode, errDesc, resp.Header)
	}

	// Parse response
	var apiResp PayOSResponseType
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, apierror.NewPayOSError("failed to parse response")
	}

	// Check response status
	if apiResp.Code != "00" || apiResp.Data == nil {
		return nil, apierror.GenerateError(resp.StatusCode, apiResp.Code, apiResp.Desc, resp.Header)
	}

	// Verify response signature if required
	if opts.SignatureOpts != nil && opts.SignatureOpts.Response != "" {
		var receivedSignature string
		var expectedSignature string
		var err error

		switch opts.SignatureOpts.Response {
		case "body":
			if apiResp.Signature == nil {
				return nil, apierror.NewInvalidSignatureError("response signature not found in body")
			}
			receivedSignature = *apiResp.Signature
			expectedSignature, err = crypto.CreateSignatureFromObj(apiResp.Data, c.checksumKey)
			if err != nil {
				return nil, apierror.NewInvalidSignatureError("failed to create response signature")
			}
		case "header":
			receivedSignature = resp.Header.Get("x-signature")
			if receivedSignature == "" {
				return nil, apierror.NewInvalidSignatureError("response signature not found in header")
			}
			expectedSignature, err = crypto.CreateSignature(c.checksumKey, apiResp.Data, nil)
			if err != nil {
				return nil, apierror.NewInvalidSignatureError("failed to create response signature")
			}
		default:
			return nil, apierror.NewInvalidSignatureError("invalid signature response type")
		}

		if receivedSignature != expectedSignature {
			return nil, apierror.NewInvalidSignatureError("data integrity check failed")
		}
	}

	return apiResp.Data, nil
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, path string, query map[string]interface{}, headers map[string]string, signatureOpts *SignatureOpts) (interface{}, error) {
	return c.Request(ctx, &RequestOptions{
		Method:        "GET",
		Path:          path,
		Query:         query,
		Headers:       headers,
		SignatureOpts: signatureOpts,
	})
}

// Post performs a POST request
func (c *Client) Post(ctx context.Context, path string, body interface{}, signatureOpts *SignatureOpts, headers map[string]string) (interface{}, error) {
	return c.Request(ctx, &RequestOptions{
		Method:        "POST",
		Path:          path,
		Body:          body,
		SignatureOpts: signatureOpts,
		Headers:       headers,
	})
}

// Put performs a PUT request
func (c *Client) Put(ctx context.Context, path string, body interface{}, signatureOpts *SignatureOpts, headers map[string]string) (interface{}, error) {
	return c.Request(ctx, &RequestOptions{
		Method:        "PUT",
		Path:          path,
		Body:          body,
		SignatureOpts: signatureOpts,
		Headers:       headers,
	})
}

// Delete performs a DELETE request
func (c *Client) Delete(ctx context.Context, path string, query map[string]interface{}, headers map[string]string, signatureOpts *SignatureOpts) (interface{}, error) {
	return c.Request(ctx, &RequestOptions{
		Method:        "DELETE",
		Path:          path,
		Query:         query,
		Headers:       headers,
		SignatureOpts: signatureOpts,
	})
}

// DownloadFile downloads a file from the API
func (c *Client) DownloadFile(ctx context.Context, path string) (*FileDownloadResponse, error) {
	// Build URL
	fullURL, err := c.buildURL(path, nil)
	if err != nil {
		return nil, apierror.NewConnectionError("failed to build URL", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, apierror.NewConnectionError("failed to create request", err)
	}

	// Set headers
	req.Header = c.buildHeaders(nil)

	// Execute request with retry logic
	var lastErr error
	maxAttempts := c.maxRetries + 1

	for attempt := 0; attempt < maxAttempts; attempt++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = apierror.NewConnectionError("request failed", err)
			if attempt < c.maxRetries {
				time.Sleep(c.calculateBackoff(attempt, nil))
				continue
			}
			return nil, lastErr
		}

		// Check if response is successful
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// Check if it's a JSON error response
			contentType := resp.Header.Get("Content-Type")
			if contentType != "" && bytes.Contains([]byte(contentType), []byte("application/json")) {
				defer resp.Body.Close()
				respBody, _ := io.ReadAll(resp.Body)
				var apiResp PayOSResponseType
				if err := json.Unmarshal(respBody, &apiResp); err == nil {
					return nil, apierror.GenerateError(resp.StatusCode, apiResp.Code, apiResp.Desc, resp.Header)
				}
			}

			// Read file data
			defer resp.Body.Close()
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, apierror.NewConnectionError("failed to read file data", err)
			}

			// Extract filename from Content-Disposition header
			var filename *string
			contentDisposition := resp.Header.Get("Content-Disposition")
			if contentDisposition != "" {
				// Simple filename extraction - in production, use more robust parsing
				if idx := strings.Index(contentDisposition, "filename="); idx >= 0 {
					fnameStr := contentDisposition[idx+9:]
					fnameStr = strings.Trim(fnameStr, "\"")
					filename = &fnameStr
				}
			}

			// Get content type
			contentTypeVal := resp.Header.Get("Content-Type")
			if contentTypeVal == "" {
				contentTypeVal = "application/octet-stream"
			}

			// Get content length
			var size *int64
			if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
				if length, err := strconv.ParseInt(contentLength, 10, 64); err == nil {
					size = &length
				}
			}

			return &FileDownloadResponse{
				Filename:    filename,
				ContentType: contentTypeVal,
				Size:        size,
				Data:        data,
			}, nil
		}

		// Handle error responses
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		var apiResp PayOSResponseType
		if err := json.Unmarshal(respBody, &apiResp); err == nil {
			lastErr = apierror.GenerateError(resp.StatusCode, apiResp.Code, apiResp.Desc, resp.Header)
		} else {
			lastErr = apierror.GenerateError(resp.StatusCode, "", string(respBody), resp.Header)
		}

		// Check if we should retry
		if c.shouldRetry(resp.StatusCode) && attempt < c.maxRetries {
			time.Sleep(c.calculateBackoff(attempt, resp.Header))
			continue
		}

		return nil, lastErr
	}

	return nil, lastErr
}
