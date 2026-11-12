package payos

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"

	"github.com/payOSHQ/payos-lib-golang/v2/internal/apierror"
)

var webhookCoreIntKeys = []string{"orderCode", "amount"}

var webhookCoreStringKeys = []string{
	"description", "accountNumber", "reference", "transactionDateTime",
	"currency", "paymentLinkId", "code", "desc",
}

var webhookOptionalStringKeys = []string{
	"counterAccountBankId", "counterAccountBankName", "counterAccountName",
	"counterAccountNumber", "virtualAccountName", "virtualAccountNumber",
}

func invalidWebhookData(reason string) error {
	return apierror.NewWebhookError("Invalid webhook data: " + reason)
}

func isValidKey(k string) bool {
	if k == "" || k == "__proto__" {
		return false
	}
	for i := 0; i < len(k); i++ {
		c := k[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

func isSafeInteger(v interface{}) bool {
	const maxSafe = 1<<53 - 1
	switch x := v.(type) {
	case float64:
		return !math.IsNaN(x) && !math.IsInf(x, 0) && x == math.Trunc(x) && -maxSafe <= x && x <= maxSafe
	case json.Number:
		n, err := strconv.ParseInt(string(x), 10, 64)
		return err == nil && -maxSafe <= n && n <= maxSafe
	case int:
		n := int64(x)
		return -maxSafe <= n && n <= maxSafe
	case int32:
		n := int64(x)
		return -maxSafe <= n && n <= maxSafe
	case int64:
		return -maxSafe <= x && x <= maxSafe
	default:
		return false
	}
}

func validateWebhookData(data interface{}) (map[string]interface{}, error) {
	m, ok := data.(map[string]interface{})
	if !ok {
		return nil, invalidWebhookData("data must be an object")
	}

	for k, v := range m {
		if !isValidKey(k) {
			return nil, invalidWebhookData("invalid key " + strconv.Quote(k))
		}
		if v == nil {
			continue
		}
		if s, isStr := v.(string); isStr {
			if strings.Contains(s, "=") {
				return nil, invalidWebhookData(k + " must not contain '='")
			}
			continue
		}
		if isSafeInteger(v) {
			continue
		}
		return nil, invalidWebhookData(k + " has unsupported type")
	}

	for _, k := range webhookCoreIntKeys {
		v, exists := m[k]
		if !exists || !isSafeInteger(v) {
			return nil, invalidWebhookData(k + " must be a safe integer")
		}
	}
	for _, k := range webhookCoreStringKeys {
		if _, isStr := m[k].(string); !isStr {
			return nil, invalidWebhookData(k + " must be a string")
		}
	}
	for _, k := range webhookOptionalStringKeys {
		v, exists := m[k]
		if !exists || v == nil {
			continue
		}
		if _, isStr := v.(string); !isStr {
			return nil, invalidWebhookData(k + " must be a string or null")
		}
	}

	return m, nil
}

func isSafeInt64(n int64) bool {
	const maxSafe = 1<<53 - 1
	return -maxSafe <= n && n <= maxSafe
}

func validateTypedWebhookData(d *WebhookData) error {
	if !isSafeInt64(d.OrderCode) {
		return invalidWebhookData("orderCode must be a safe integer")
	}
	if !isSafeInt64(int64(d.Amount)) {
		return invalidWebhookData("amount must be a safe integer")
	}

	strs := []struct {
		key string
		val string
	}{
		{"description", d.Description},
		{"accountNumber", d.AccountNumber},
		{"reference", d.Reference},
		{"transactionDateTime", d.TransactionDateTime},
		{"currency", d.Currency},
		{"paymentLinkId", d.PaymentLinkId},
		{"code", d.Code},
		{"desc", d.Desc},
	}
	for _, s := range strs {
		if strings.Contains(s.val, "=") {
			return invalidWebhookData(s.key + " must not contain '='")
		}
	}

	optional := []struct {
		key string
		val *string
	}{
		{"counterAccountBankId", d.CounterAccountBankId},
		{"counterAccountBankName", d.CounterAccountBankName},
		{"counterAccountName", d.CounterAccountName},
		{"counterAccountNumber", d.CounterAccountNumber},
		{"virtualAccountName", d.VirtualAccountName},
		{"virtualAccountNumber", d.VirtualAccountNumber},
	}
	for _, s := range optional {
		if s.val != nil && strings.Contains(*s.val, "=") {
			return invalidWebhookData(s.key + " must not contain '='")
		}
	}
	return nil
}
