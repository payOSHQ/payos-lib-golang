package payos

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/payOSHQ/payos-lib-golang/v2/internal/apierror"
)

// maxSafeInteger is the largest integer a JSON number (IEEE-754 double) represents exactly.
const maxSafeInteger = 1<<53 - 1

// encodePathSegment returns value as a single, percent-encoded URL path segment.
//
// Identifiers are interpolated into paths such as /v2/payment-requests/{id}/cancel. Without
// encoding, a value like "../../v1/payouts-account/balance" or "x?" re-targets the authenticated
// request to another endpoint (path traversal / query injection), so strings are percent-encoded
// and "." / ".." segments are rejected. Integers must be non-negative and within the safe range.
func encodePathSegment(name string, value interface{}) (string, error) {
	switch v := value.(type) {
	case int:
		return encodeIntSegment(name, int64(v))
	case int64:
		return encodeIntSegment(name, v)
	case string:
		if v == "" {
			return "", apierror.NewPayOSError("invalid params")
		}
		if v == "." || v == ".." {
			return "", apierror.NewPayOSError(fmt.Sprintf("%s must not be a dot segment", name))
		}
		return url.PathEscape(v), nil
	default:
		return "", apierror.NewPayOSError(fmt.Sprintf("%s must be string or number", name))
	}
}

func encodeIntSegment(name string, v int64) (string, error) {
	if v < 0 || v > maxSafeInteger {
		return "", apierror.NewPayOSError(fmt.Sprintf("%s must be a non-negative safe integer", name))
	}
	return strconv.FormatInt(v, 10), nil
}
