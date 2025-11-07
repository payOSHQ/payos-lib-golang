package payos

import (
	"log"
	"net/http"
	"os"
	"time"
)

// PayOSOptions defines configuration options for the PayOS client
type PayOSOptions struct {
	// ClientId is the PayOS client identifier
	// Defaults to PAYOS_CLIENT_ID environment variable
	ClientId string

	// ApiKey is the PayOS API key for authentication
	// Defaults to PAYOS_API_KEY environment variable
	ApiKey string

	// ChecksumKey is used for signing and verifying requests
	// Defaults to PAYOS_CHECKSUM_KEY environment variable
	ChecksumKey string

	// PartnerCode is an optional partner identifier
	// Defaults to PAYOS_PARTNER_CODE environment variable
	PartnerCode string

	// BaseURL is the base URL for API requests
	// Defaults to https://api-merchant.payos.vn
	BaseURL string

	// HTTPClient is the HTTP client to use for requests
	// If nil, a default client with timeout will be created
	HTTPClient *http.Client

	// MaxRetries is the maximum number of retry attempts for failed requests
	// Defaults to 2
	MaxRetries int

	// Timeout is the maximum duration for a request
	// Defaults to 60 seconds
	Timeout time.Duration

	// Middlewares is a slice of middleware functions to wrap request execution
	// Middlewares are executed in order, with the first middleware being the outermost
	Middlewares []Middleware

	// DebugLogger enables debug logging for requests and responses
	// If set to a non-nil *log.Logger, debug logs will be written to that logger
	// If set to nil and debug logging is desired, pass log.New() with desired output
	// If not set (nil by default), no debug logging will occur
	DebugLogger *log.Logger
}

// NewPayOSOptions creates a new PayOSOptions
func NewPayOSOptions(opts *PayOSOptions) *PayOSOptions {
	if opts == nil {
		opts = &PayOSOptions{}
	}

	return &PayOSOptions{
		ClientId:    getValue(opts.ClientId, os.Getenv("PAYOS_CLIENT_ID"), ""),
		ApiKey:      getValue(opts.ApiKey, os.Getenv("PAYOS_API_KEY"), ""),
		ChecksumKey: getValue(opts.ChecksumKey, os.Getenv("PAYOS_CHECKSUM_KEY"), ""),
		PartnerCode: getValue(opts.PartnerCode, os.Getenv("PAYOS_PARTNER_CODE"), ""),
		BaseURL:     getValue(opts.BaseURL, os.Getenv("PAYOS_BASE_URL"), PayOSBaseUrl),
		HTTPClient:  opts.HTTPClient,
		MaxRetries:  getIntValue(opts.MaxRetries, defaultMaxRetry),
		Timeout:     getTimeoutValue(opts.Timeout, defaultTimeout),
		Middlewares: opts.Middlewares,
		DebugLogger: opts.DebugLogger,
	}
}

// getValue returns the first non-empty string value
func getValue(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// getIntValue returns the first non-zero int value
func getIntValue(values ...int) int {
	for _, v := range values {
		if v != 0 {
			return v
		}
	}
	return 0
}

// getTimeoutValue returns the first non-zero time.Duration value
func getTimeoutValue(values ...time.Duration) time.Duration {
	for _, v := range values {
		if v != 0 {
			return v
		}
	}
	return 0
}
