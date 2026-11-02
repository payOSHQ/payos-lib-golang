package payos

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/payOSHQ/payos-lib-golang/v2/internal/apierror"
	"github.com/payOSHQ/payos-lib-golang/v2/internal/crypto"
)

const responseChecksumKey = "checksum-key"

// signedResponseServer answers every request with dataJSON signed using the SDK's own helper.
func signedResponseServer(t *testing.T, dataJSON string) *PayOS {
	t.Helper()
	var data interface{}
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	signature, err := crypto.CreateSignatureFromObj(data, responseChecksumKey)
	if err != nil {
		signature = "unsigned"
	}
	body := `{"code":"00","desc":"success","data":` + dataJSON + `,"signature":"` + signature + `"}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	client, err := NewPayOS(&PayOSOptions{
		ClientId:    "client-id",
		ApiKey:      "api-key",
		ChecksumKey: responseChecksumKey,
		BaseURL:     srv.URL,
	})
	if err != nil {
		t.Fatalf("NewPayOS: %v", err)
	}
	return client
}

const paymentLinkData = `{
	"id": "f2b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8",
	"orderCode": 1730000000123,
	"amount": 10000,
	"amountPaid": 10000,
	"amountRemaining": 0,
	"status": "PAID",
	"createdAt": "2026-11-02T10:00:00+07:00",
	"transactions": [{
		"reference": "FT26306123456",
		"amount": 10000,
		"accountNumber": "0123456789",
		"description": "CSABC123 thanh toan don hang",
		"transactionDateTime": "2026-11-02T10:01:00+07:00",
		"virtualAccountName": null,
		"virtualAccountNumber": null,
		"counterAccountBankId": "970422",
		"counterAccountBankName": null,
		"counterAccountName": "NGUYEN VAN A",
		"counterAccountNumber": "9876543210"
	}],
	"cancellationReason": null,
	"canceledAt": null
}`

func TestSignedPaymentLinkResponseIsAccepted(t *testing.T) {
	client := signedResponseServer(t, paymentLinkData)
	ctx := context.Background()

	link, err := client.PaymentRequests.Get(ctx, "f2b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if link.OrderCode != 1730000000123 || link.Status != "PAID" || len(link.Transactions) != 1 {
		t.Fatalf("unexpected payment link: %+v", link)
	}
	if _, err := client.PaymentRequests.Cancel(ctx, 1730000000123, nil); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
}

func TestSignedInvoicesResponseIsAccepted(t *testing.T) {
	client := signedResponseServer(t, `{"invoices":[{
		"invoiceId": "6a1b2c3d",
		"invoiceNumber": "0000123",
		"issuedTimestamp": 1730516400,
		"issuedDatetime": "2026-11-02T10:00:00+07:00",
		"transactionId": "FT26306123456",
		"reservationCode": "AB12CD34",
		"codeOfTax": "00ABC123DEF456"
	}]}`)

	info, err := client.PaymentRequests.Invoices.Get(context.Background(), 1730000000123)
	if err != nil {
		t.Fatalf("Invoices.Get: %v", err)
	}
	if len(info.Invoices) != 1 || info.Invoices[0].InvoiceId != "6a1b2c3d" {
		t.Fatalf("unexpected invoices: %+v", info)
	}
}

func TestSignedResponseDataIsValidatedBeforeSignature(t *testing.T) {
	cases := []struct {
		name string
		data string
		want string
	}{
		{"ampersand-key sequence in string", `{"description":"order&amount=1","amount":10000}`,
			`"description" must not contain a "&key=" sequence`},
		{"ampersand-key sequence in list item field", `{"transactions":[{"description":"x&reference=y"}]}`,
			`"transactions[0].description" must not contain a "&key=" sequence`},
		{"key with hyphen", `{"order-code":1}`, `unsupported key "order-code"`},
		{"key with equals sign", `{"a=b":1}`, `unsupported key "a=b"`},
		{"proto key", `{"__proto__":1}`, `unsupported key "__proto__"`},
		{"unsupported key inside list item", `{"transactions":[{"a&b":1}]}`, `unsupported key "transactions[0].a&b"`},
		{"nested object at top level", `{"meta":{"a":1}}`, `"meta" has an unsupported type`},
		{"list inside list item object", `{"transactions":[{"refs":[1]}]}`, `"transactions[0].refs" has an unsupported type`},
		{"object inside list item object", `{"transactions":[{"x":{"a":1}}]}`, `"transactions[0].x" has an unsupported type`},
		{"list inside list", `{"items":[[1]]}`, `"items[0]" has an unsupported type`},
		{"integer above safe range", `{"orderCode":9007199254740993}`,
			`"orderCode" must be a finite number within the safe integer range`},
		{"negative integer below safe range", `{"amount":-9007199254740993}`,
			`"amount" must be a finite number within the safe integer range`},
		{"unsafe integer in list item", `{"transactions":[{"amount":1e20}]}`,
			`"transactions[0].amount" must be a finite number within the safe integer range`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := signedResponseServer(t, tc.data)
			_, err := client.PaymentRequests.Get(context.Background(), "abc")
			var sigErr *apierror.InvalidSignatureError
			if !errors.As(err, &sigErr) {
				t.Fatalf("expected *apierror.InvalidSignatureError, got %T (%v)", err, err)
			}
			want := "Invalid response data: " + tc.want
			if sigErr.Message != want {
				t.Fatalf("message = %q, want %q", sigErr.Message, want)
			}
			if !strings.HasPrefix(err.Error(), "invalid signature: ") {
				t.Fatalf("Error() = %q", err.Error())
			}
		})
	}
}

func TestSafeIntegerBoundaryIsAccepted(t *testing.T) {
	client := signedResponseServer(t, `{"orderCode":9007199254740991,"amount":-9007199254740991,"ratio":0.5}`)
	if _, err := client.PaymentRequests.Get(context.Background(), "abc"); err != nil {
		t.Fatalf("Get: %v", err)
	}
}
