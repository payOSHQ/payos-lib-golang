package payos

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"

	"github.com/payOSHQ/payos-lib-golang/v2/internal/apierror"
)

// A `&key=` sequence inside a value is what lets the unencoded signed string be re-split.
var responseSmugglingPattern = regexp.MustCompile(`&[A-Za-z0-9_]+=`)

func invalidResponseData(format string, a ...interface{}) error {
	return apierror.NewInvalidSignatureError("Invalid response data: " + fmt.Sprintf(format, a...))
}

func isValidSignedKey(key string) bool {
	if key == "" || key == "__proto__" {
		return false
	}
	for _, r := range key {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_') {
			return false
		}
	}
	return true
}

func validateResponseScalar(path string, value interface{}) error {
	switch v := value.(type) {
	case nil, bool:
	case string:
		if responseSmugglingPattern.MatchString(v) {
			return invalidResponseData("%q must not contain a \"&key=\" sequence", path)
		}
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) || (v == math.Trunc(v) && math.Abs(v) > maxSafeInteger) {
			return invalidResponseData("%q must be a finite number within the safe integer range", path)
		}
	case json.Number:
		f, err := v.Float64()
		if err != nil || math.IsInf(f, 0) || (f == math.Trunc(f) && math.Abs(f) > maxSafeInteger) {
			return invalidResponseData("%q must be a finite number within the safe integer range", path)
		}
	default:
		return invalidResponseData("%q has an unsupported type", path)
	}
	return nil
}

// validateResponseData checks the shape of the `data` of a response signed with the "body"
// signature before the signature is computed. The legacy signature is built from unencoded
// key=value pairs joined by '&' with lists serialised as JSON, so the data must not be able to
// shift delimiters, replace a scalar with an object, or turn a list into a string that swallows
// the remaining fields. Top-level values are scalars or lists; objects are only accepted inside
// lists and may only contain scalars.
func validateResponseData(data interface{}) error {
	record, ok := data.(map[string]interface{})
	if !ok {
		return invalidResponseData("data must be a JSON object")
	}
	for key, value := range record {
		if !isValidSignedKey(key) {
			return invalidResponseData("unsupported key %q", key)
		}
		list, isList := value.([]interface{})
		if !isList {
			if err := validateResponseScalar(key, value); err != nil {
				return err
			}
			continue
		}
		for i, item := range list {
			path := fmt.Sprintf("%s[%d]", key, i)
			entry, isObject := item.(map[string]interface{})
			if !isObject {
				if err := validateResponseScalar(path, item); err != nil {
					return err
				}
				continue
			}
			for field, fieldValue := range entry {
				if !isValidSignedKey(field) {
					return invalidResponseData("unsupported key %q", path+"."+field)
				}
				if err := validateResponseScalar(path+"."+field, fieldValue); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
