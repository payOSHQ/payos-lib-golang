package payos

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/payOSHQ/payos-lib-golang/v2/internal/apierror"
	"github.com/payOSHQ/payos-lib-golang/v2/internal/crypto"
)

type webhookVectorFile struct {
	ChecksumKey string          `json:"checksumKey"`
	Cases       []webhookVector `json:"cases"`
}

type webhookVector struct {
	ID     string `json:"id"`
	Expect string `json:"expect"`
	Body   string `json:"body"`
}

func TestWebhookVectors(t *testing.T) {
	raw, err := os.ReadFile("testdata/webhook-vectors.json")
	if err != nil {
		t.Fatalf("read vectors: %v", err)
	}
	var vf webhookVectorFile
	if err := json.Unmarshal(raw, &vf); err != nil {
		t.Fatalf("parse vectors: %v", err)
	}
	if len(vf.Cases) != 26 {
		t.Fatalf("expected 26 vectors, got %d", len(vf.Cases))
	}

	for _, c := range vf.Cases {
		c := c
		t.Run(c.ID, func(t *testing.T) {
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(c.Body), &m); err != nil {
				t.Fatalf("parse body: %v", err)
			}
			got, err := verifyWebhookSignature(m, vf.ChecksumKey)

			switch c.Expect {
			case "accept":
				if err != nil {
					t.Fatalf("expected accept, got %v", err)
				}
				gotMap, ok := got.(map[string]interface{})
				if !ok {
					t.Fatalf("expected map result, got %T", got)
				}
				want := m["data"].(map[string]interface{})["orderCode"]
				if gotMap["orderCode"] != want {
					t.Fatalf("orderCode = %v, want %v", gotMap["orderCode"], want)
				}
			case "reject":
				if err == nil {
					t.Fatal("expected reject, got nil error")
				}
				if strings.HasPrefix(c.ID, "R14_") || strings.HasPrefix(c.ID, "R15_") {
					return
				}
				var we *apierror.WebhookError
				if !errors.As(err, &we) {
					t.Fatalf("expected WebhookError, got %T: %v", err, err)
				}
				if !strings.HasPrefix(we.Message, "Invalid webhook data") {
					t.Fatalf("unexpected message: %q", we.Message)
				}
			default:
				t.Fatalf("unknown expect %q", c.Expect)
			}
		})
	}
}

const typedTestKey = "typed_test_key"

func withChecksumKey(t *testing.T, key string) {
	t.Helper()
	prev := PayOSChecksumKey
	PayOSChecksumKey = key
	t.Cleanup(func() { PayOSChecksumKey = prev })
}

func validTypedWebhookData() *WebhookData {
	name := "CONG TY TNHH A&B"
	return &WebhookData{
		OrderCode:            123,
		Amount:               2000,
		Description:          "VQRIO123",
		AccountNumber:        "0123456789",
		Reference:            "TF230204212323",
		TransactionDateTime:  "2023-02-04 18:25:00",
		Currency:             "VND",
		PaymentLinkId:        "124c33293c43417ab7879e14c8d9eb18",
		Code:                 "00",
		Desc:                 "success",
		CounterAccountName:   &name,
		VirtualAccountNumber: nil,
	}
}

func signedTypedWebhook(t *testing.T, d *WebhookData) WebhookType {
	t.Helper()
	sig, err := crypto.CreateSignatureFromObj(d, typedTestKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return WebhookType{Code: "00", Desc: "success", Data: d, Signature: sig}
}

func assertInvalidWebhookData(t *testing.T, err error) {
	t.Helper()
	var we *apierror.WebhookError
	if !errors.As(err, &we) {
		t.Fatalf("expected WebhookError, got %T: %v", err, err)
	}
	if !strings.HasPrefix(we.Message, "Invalid webhook data") {
		t.Fatalf("unexpected message: %q", we.Message)
	}
}

func TestVerifyPaymentWebhookDataAcceptsValid(t *testing.T) {
	withChecksumKey(t, typedTestKey)
	d := validTypedWebhookData()
	got, err := VerifyPaymentWebhookData(signedTypedWebhook(t, d))
	if err != nil {
		t.Fatalf("expected accept, got %v", err)
	}
	if got.OrderCode != d.OrderCode {
		t.Fatalf("orderCode = %d, want %d", got.OrderCode, d.OrderCode)
	}
}

func TestVerifyPaymentWebhookDataRejectsEquals(t *testing.T) {
	withChecksumKey(t, typedTestKey)

	t.Run("core_field", func(t *testing.T) {
		d := validTypedWebhookData()
		d.Description = "VQRIO123&amount=1"
		_, err := VerifyPaymentWebhookData(signedTypedWebhook(t, d))
		assertInvalidWebhookData(t, err)
	})
	t.Run("optional_pointer_field", func(t *testing.T) {
		d := validTypedWebhookData()
		v := "x&amount=1"
		d.VirtualAccountName = &v
		_, err := VerifyPaymentWebhookData(signedTypedWebhook(t, d))
		assertInvalidWebhookData(t, err)
	})
}

func TestVerifyPaymentWebhookDataRejectsUnsafeIntegers(t *testing.T) {
	withChecksumKey(t, typedTestKey)
	const maxSafe = 1<<53 - 1
	cases := map[string]func(d *WebhookData){
		"order_code_above": func(d *WebhookData) { d.OrderCode = maxSafe + 1 },
		"order_code_below": func(d *WebhookData) { d.OrderCode = -maxSafe - 1 },
		"order_code_min":   func(d *WebhookData) { d.OrderCode = math.MinInt64 },
		"amount_above":     func(d *WebhookData) { d.Amount = maxSafe + 1 },
		"amount_below":     func(d *WebhookData) { d.Amount = -maxSafe - 1 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			d := validTypedWebhookData()
			mutate(d)
			_, err := VerifyPaymentWebhookData(signedTypedWebhook(t, d))
			assertInvalidWebhookData(t, err)
		})
	}
}

func TestVerifyPaymentWebhookDataTypedVectors(t *testing.T) {
	raw, err := os.ReadFile("testdata/webhook-vectors.json")
	if err != nil {
		t.Fatalf("read vectors: %v", err)
	}
	var vf struct {
		ChecksumKey string `json:"checksumKey"`
		Cases       []struct {
			webhookVector
			Typed bool `json:"typed"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &vf); err != nil {
		t.Fatalf("parse vectors: %v", err)
	}
	withChecksumKey(t, vf.ChecksumKey)

	ran := 0
	for _, c := range vf.Cases {
		if !c.Typed {
			continue
		}
		ran++
		c := c
		t.Run(c.ID, func(t *testing.T) {
			var wh WebhookType
			if err := json.Unmarshal([]byte(c.Body), &wh); err != nil {
				if c.Expect == "reject" {
					return
				}
				t.Fatalf("unmarshal accept case: %v", err)
			}
			got, err := VerifyPaymentWebhookData(wh)
			switch c.Expect {
			case "accept":
				if err != nil {
					t.Fatalf("expected accept, got %v", err)
				}
				var raw struct {
					Data struct {
						OrderCode int64 `json:"orderCode"`
					} `json:"data"`
				}
				if err := json.Unmarshal([]byte(c.Body), &raw); err != nil {
					t.Fatalf("parse orderCode: %v", err)
				}
				if got.OrderCode != raw.Data.OrderCode {
					t.Fatalf("orderCode = %d, want %d", got.OrderCode, raw.Data.OrderCode)
				}
			case "reject":
				if err == nil {
					t.Fatal("expected reject, got nil error")
				}
			default:
				t.Fatalf("unknown expect %q", c.Expect)
			}
		})
	}
	if ran != 12 {
		t.Fatalf("expected 12 typed vectors, ran %d", ran)
	}
}
