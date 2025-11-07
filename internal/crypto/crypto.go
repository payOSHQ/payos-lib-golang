package crypto

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CreateSignatureFromObj creates a signature from an object by sorting keys
func CreateSignatureFromObj(obj interface{}, key string) (string, error) {
	sortedObj, err := SortObjByKey(obj)
	if err != nil {
		return "", err
	}

	keyBytes := []byte(key)

	hasher := hmac.New(sha256.New, keyBytes)

	hasher.Write([]byte(sortedObj))

	signature := hex.EncodeToString(hasher.Sum(nil))

	return signature, nil
}

// SignatureOptions contains options for signature creation
type SignatureOptions struct {
	EncodeURI  bool
	SortArrays bool
	Algorithm  string // "sha256", "sha1", "sha512", "md5" (default: "sha256")
}

// CreateSignature creates a header signature from JSON data with query string format
func CreateSignature(key string, data interface{}, options *SignatureOptions) (string, error) {
	// Set default options
	if options == nil {
		options = &SignatureOptions{
			EncodeURI:  true,
			SortArrays: false,
			Algorithm:  "sha256",
		}
	}
	if options.Algorithm == "" {
		options.Algorithm = "sha256"
	}

	// Deep sort the data
	sortedData, err := deepSortObject(data, options.SortArrays)
	if err != nil {
		return "", err
	}

	// Convert to map for processing
	dataMap, ok := sortedData.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("data must be an object/map")
	}

	// Build query string
	var pairs []string
	keys := make([]string, 0, len(dataMap))
	for key := range dataMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := dataMap[key]
		var stringValue string

		// Handle different value types
		switch v := value.(type) {
		case []interface{}:
			// Arrays are JSON stringified
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			stringValue = string(jsonBytes)
		case map[string]interface{}:
			// Nested objects are JSON stringified
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			stringValue = string(jsonBytes)
		case nil:
			// Null/undefined values become empty string
			stringValue = ""
		default:
			// Other types (string, number, bool) are converted to string
			stringValue = fmt.Sprint(v)
		}

		// Build key=value pair with optional URL encoding
		if options.EncodeURI {
			// url.QueryEscape encodes space as '+', but we need '%20'
			encodedKey := strings.ReplaceAll(url.QueryEscape(key), "+", "%20")
			encodedValue := strings.ReplaceAll(url.QueryEscape(stringValue), "+", "%20")
			pairs = append(pairs, fmt.Sprintf("%s=%s", encodedKey, encodedValue))
		} else {
			pairs = append(pairs, fmt.Sprintf("%s=%s", key, stringValue))
		}
	}

	queryString := strings.Join(pairs, "&")

	// Create HMAC with specified algorithm
	var hasher hash.Hash
	switch options.Algorithm {
	case "sha256":
		hasher = hmac.New(sha256.New, []byte(key))
	case "sha1":
		hasher = hmac.New(sha1.New, []byte(key))
	case "sha512":
		hasher = hmac.New(sha512.New, []byte(key))
	case "md5":
		hasher = hmac.New(md5.New, []byte(key))
	default:
		return "", fmt.Errorf("unsupported algorithm: %s", options.Algorithm)
	}

	hasher.Write([]byte(queryString))
	signature := hex.EncodeToString(hasher.Sum(nil))

	return signature, nil
}

// CreateSignatureOfPaymentRequest creates a signature specifically for payment requests
func CreateSignatureOfPaymentRequest(data interface{}, key string) (string, error) {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	amountField := v.FieldByName("Amount")
	cancelUrlField := v.FieldByName("CancelUrl")
	descriptionField := v.FieldByName("Description")
	orderCodeField := v.FieldByName("OrderCode")
	returnUrlField := v.FieldByName("ReturnUrl")

	if !amountField.IsValid() || !cancelUrlField.IsValid() || !descriptionField.IsValid() ||
		!orderCodeField.IsValid() || !returnUrlField.IsValid() {
		return "", fmt.Errorf("data must have Amount, CancelUrl, Description, OrderCode, and ReturnUrl fields")
	}

	amount := int(amountField.Int())
	cancelUrl := cancelUrlField.String()
	description := descriptionField.String()
	orderCode := orderCodeField.Int()
	returnUrl := returnUrlField.String()

	dataStr := fmt.Sprintf("amount=%s&cancelUrl=%s&description=%s&orderCode=%s&returnUrl=%s",
		strconv.Itoa(amount), cancelUrl, description, strconv.FormatInt(orderCode, 10), returnUrl)

	hasher := hmac.New(sha256.New, []byte(key))
	hasher.Write([]byte(dataStr))
	signature := hex.EncodeToString(hasher.Sum(nil))

	return signature, nil
}

// SortObjByKey sorts an object's keys and returns a formatted string
func SortObjByKey(obj interface{}) (string, error) {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}

	var sortedPairs []string

	var jsonObj map[string]interface{}
	err = json.Unmarshal(jsonBytes, &jsonObj)
	if err != nil {
		return "", err
	}

	keys := make([]string, 0, len(jsonObj))
	for key := range jsonObj {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := jsonObj[key]
		if value != nil {
			stringValue, err := convertToString(value)
			if err != nil {
				return "", err
			}
			sortedPairs = append(sortedPairs, fmt.Sprintf("%s=%v", key, stringValue))
		} else {
			sortedPairs = append(sortedPairs, fmt.Sprintf("%s=", key))
		}
	}

	sortedObj := strings.Join(sortedPairs, "&")

	return sortedObj, nil
}

func convertToString(value interface{}) (string, error) {
	switch v := value.(type) {
	case int64:
		return strconv.FormatInt(v, 10), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case string:
		return fmt.Sprint(value), nil
	default:
		resultBytes, err := json.Marshal(value)
		return string(resultBytes), err
	}
}

func GenerateUUID() string {
	// Simple UUID v4 generation
	b := make([]byte, 16)
	// This is a simplified version - in production, use crypto/rand
	timestamp := time.Now().UnixNano()
	for i := 0; i < 16; i++ {
		b[i] = byte((timestamp >> (i * 8)) & 0xff)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant bits
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// deepSortObject recursively sorts an object's keys
func deepSortObject(data interface{}, sortArrays bool) (interface{}, error) {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			sorted, err := deepSortObject(value, sortArrays)
			if err != nil {
				return nil, err
			}
			result[key] = sorted
		}
		return result, nil
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			sorted, err := deepSortObject(item, sortArrays)
			if err != nil {
				return nil, err
			}
			result[i] = sorted
		}
		if sortArrays {
			sort.Slice(result, func(i, j int) bool {
				// Sort by JSON string representation
				iStr, _ := json.Marshal(result[i])
				jStr, _ := json.Marshal(result[j])
				return string(iStr) < string(jStr)
			})
		}
		return result, nil
	default:
		// For other types, just marshal and unmarshal to normalize
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		var result interface{}
		err = json.Unmarshal(jsonBytes, &result)
		if err != nil {
			return v, nil // Return original if unmarshal fails
		}
		return result, nil
	}
}
