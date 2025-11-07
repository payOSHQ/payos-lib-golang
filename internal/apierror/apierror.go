package apierror

import (
	"fmt"
	"net/http"
)

// PayOSError is the base error type for all PayOS errors
type PayOSError struct {
	message string
}

func NewPayOSError(message string) *PayOSError {
	return &PayOSError{
		message: message,
	}
}

func (e *PayOSError) Error() string {
	return e.message
}

// APIError represents an error returned by the PayOS API
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Headers    http.Header
}

func NewAPIError(statusCode int, code, message string, headers http.Header) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
		Headers:    headers,
	}
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (status %d, code %s): %s", e.StatusCode, e.Code, e.Message)
}

// BadRequestError represents a 400 error
type BadRequestError struct {
	*APIError
}

func NewBadRequestError(code, message string, headers http.Header) *BadRequestError {
	return &BadRequestError{
		APIError: NewAPIError(400, code, message, headers),
	}
}

// UnauthorizedError represents a 401 error
type UnauthorizedError struct {
	*APIError
}

func NewUnauthorizedError(code, message string, headers http.Header) *UnauthorizedError {
	return &UnauthorizedError{
		APIError: NewAPIError(401, code, message, headers),
	}
}

// ForbiddenError represents a 403 error
type ForbiddenError struct {
	*APIError
}

func NewForbiddenError(code, message string, headers http.Header) *ForbiddenError {
	return &ForbiddenError{
		APIError: NewAPIError(403, code, message, headers),
	}
}

// NotFoundError represents a 404 error
type NotFoundError struct {
	*APIError
}

func NewNotFoundError(code, message string, headers http.Header) *NotFoundError {
	return &NotFoundError{
		APIError: NewAPIError(404, code, message, headers),
	}
}

// TooManyRequestError represents a 429 error
type TooManyRequestError struct {
	*APIError
}

func NewTooManyRequestError(code, message string, headers http.Header) *TooManyRequestError {
	return &TooManyRequestError{
		APIError: NewAPIError(429, code, message, headers),
	}
}

// InternalServerError represents a 500+ error
type InternalServerError struct {
	*APIError
}

func NewInternalServerError(statusCode int, code, message string, headers http.Header) *InternalServerError {
	return &InternalServerError{
		APIError: NewAPIError(statusCode, code, message, headers),
	}
}

// ConnectionError represents a network connection error
type ConnectionError struct {
	Message string
	Err     error
}

func NewConnectionError(message string, err error) *ConnectionError {
	return &ConnectionError{
		Message: message,
		Err:     err,
	}
}

func (e *ConnectionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("connection error: %s - %v", e.Message, e.Err)
	}
	return fmt.Sprintf("connection error: %s", e.Message)
}

func (e *ConnectionError) Unwrap() error {
	return e.Err
}

// ConnectionTimeoutError represents a request timeout error
type ConnectionTimeoutError struct {
	Message string
}

func NewConnectionTimeoutError(message string) *ConnectionTimeoutError {
	return &ConnectionTimeoutError{
		Message: message,
	}
}

func (e *ConnectionTimeoutError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("connection timeout: %s", e.Message)
	}
	return "connection timeout"
}

// InvalidSignatureError represents a signature verification error
type InvalidSignatureError struct {
	Message string
}

func NewInvalidSignatureError(message string) *InvalidSignatureError {
	return &InvalidSignatureError{
		Message: message,
	}
}

func (e *InvalidSignatureError) Error() string {
	return fmt.Sprintf("invalid signature: %s", e.Message)
}

// WebhookError represents an error processing webhook data
type WebhookError struct {
	Message string
}

func NewWebhookError(message string) *WebhookError {
	return &WebhookError{
		Message: message,
	}
}

func (e *WebhookError) Error() string {
	return fmt.Sprintf("webhook error: %s", e.Message)
}

// GenerateError creates the appropriate error type based on status code
func GenerateError(statusCode int, code, message string, headers http.Header) error {
	switch statusCode {
	case 400:
		return NewBadRequestError(code, message, headers)
	case 401:
		return NewUnauthorizedError(code, message, headers)
	case 403:
		return NewForbiddenError(code, message, headers)
	case 404:
		return NewNotFoundError(code, message, headers)
	case 429:
		return NewTooManyRequestError(code, message, headers)
	default:
		if statusCode >= 500 {
			return NewInternalServerError(statusCode, code, message, headers)
		}
		return NewAPIError(statusCode, code, message, headers)
	}
}
