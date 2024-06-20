package payos

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

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

func CreateSignatureOfPaymentRequest(data CheckoutRequestType, key string) (string, error) {
	dataStr := fmt.Sprintf("amount=%s&cancelUrl=%s&description=%s&orderCode=%s&returnUrl=%s",
		strconv.Itoa(data.Amount), data.CancelUrl, data.Description, strconv.FormatInt(data.OrderCode, 10), data.ReturnUrl)

	hasher := hmac.New(sha256.New, []byte(key))
	hasher.Write([]byte(dataStr))
	signature := hex.EncodeToString(hasher.Sum(nil))

	return signature, nil
}

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
