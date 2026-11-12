package payos

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/payOSHQ/payos-lib-golang/v2/internal/apierror"
)

// pathRecorder is a test server that records the raw request URI of every request.
type pathRecorder struct {
	mu   sync.Mutex
	uris []string
	srv  *httptest.Server
}

func newPathRecorder(t *testing.T) *pathRecorder {
	t.Helper()
	rec := &pathRecorder{}
	rec.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.mu.Lock()
		rec.uris = append(rec.uris, r.RequestURI)
		rec.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"101","desc":"not found","data":null}`))
	}))
	t.Cleanup(rec.srv.Close)
	return rec
}

func (rec *pathRecorder) take() []string {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	out := rec.uris
	rec.uris = nil
	return out
}

func newPathTestClient(t *testing.T, baseURL string) *PayOS {
	t.Helper()
	client, err := NewPayOS(&PayOSOptions{
		ClientId:    "client-id",
		ApiKey:      "api-key",
		ChecksumKey: "checksum-key",
		BaseURL:     baseURL,
	})
	if err != nil {
		t.Fatalf("NewPayOS: %v", err)
	}
	return client
}

type idCase struct {
	name    string
	id      string
	encoded string
}

var stringIdCases = []idCase{
	{"normal id", "abc123", "abc123"},
	{"dot segment prefix", "../x", "..%2Fx"},
	{"slash", "a/b", "a%2Fb"},
	{"question mark", "a?b", "a%3Fb"},
	{"hash", "a#b", "a%23b"},
	{"already percent-encoded slash is encoded once", "a%2Fb", "a%252Fb"},
}

func TestIdIsEncodedAsSinglePathSegment(t *testing.T) {
	rec := newPathRecorder(t)
	client := newPathTestClient(t, rec.srv.URL)
	ctx := context.Background()

	methods := []struct {
		name string
		call func(id string) error
		want func(enc string) string
	}{
		{"PaymentRequests.Get", func(id string) error {
			_, err := client.PaymentRequests.Get(ctx, id)
			return err
		}, func(enc string) string { return "/v2/payment-requests/" + enc }},
		{"PaymentRequests.Cancel", func(id string) error {
			_, err := client.PaymentRequests.Cancel(ctx, id, nil)
			return err
		}, func(enc string) string { return "/v2/payment-requests/" + enc + "/cancel" }},
		{"Invoices.Get", func(id string) error {
			_, err := client.PaymentRequests.Invoices.Get(ctx, id)
			return err
		}, func(enc string) string { return "/v2/payment-requests/" + enc + "/invoices" }},
		{"Invoices.Download payment id", func(id string) error {
			_, err := client.PaymentRequests.Invoices.Download(ctx, "inv1", id)
			return err
		}, func(enc string) string { return "/v2/payment-requests/" + enc + "/invoices/inv1/download" }},
		{"Invoices.Download invoice id", func(id string) error {
			_, err := client.PaymentRequests.Invoices.Download(ctx, id, "pl1")
			return err
		}, func(enc string) string { return "/v2/payment-requests/pl1/invoices/" + enc + "/download" }},
		{"Payouts.Get", func(id string) error {
			_, err := client.Payouts.Get(ctx, id)
			return err
		}, func(enc string) string { return "/v1/payouts/" + enc }},
	}

	for _, m := range methods {
		for _, tc := range stringIdCases {
			t.Run(m.name+"/"+tc.name, func(t *testing.T) {
				rec.take()
				_ = m.call(tc.id)
				uris := rec.take()
				if len(uris) != 1 {
					t.Fatalf("expected exactly one request, got %v", uris)
				}
				if want := m.want(tc.encoded); uris[0] != want {
					t.Fatalf("request URI = %q, want %q", uris[0], want)
				}
			})
		}
	}
}

func TestNumericIdIsSentAsDecimalSegment(t *testing.T) {
	rec := newPathRecorder(t)
	client := newPathTestClient(t, rec.srv.URL)
	ctx := context.Background()

	_, _ = client.PaymentRequests.Get(ctx, int64(9007199254740991))
	_, _ = client.PaymentRequests.Cancel(ctx, 123, nil)
	_, _ = client.PaymentRequests.Invoices.Get(ctx, 0)
	_, _ = client.PaymentRequests.Invoices.Download(ctx, "inv1", 456)

	want := []string{
		"/v2/payment-requests/9007199254740991",
		"/v2/payment-requests/123/cancel",
		"/v2/payment-requests/0/invoices",
		"/v2/payment-requests/456/invoices/inv1/download",
	}
	got := rec.take()
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("request URIs = %v, want %v", got, want)
	}
}

func assertPayOSError(t *testing.T, err error, want string) {
	t.Helper()
	var pe *apierror.PayOSError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *apierror.PayOSError, got %T (%v)", err, err)
	}
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestInvalidIdIsRejectedBeforeRequest(t *testing.T) {
	rec := newPathRecorder(t)
	client := newPathTestClient(t, rec.srv.URL)
	ctx := context.Background()

	type call struct {
		name  string
		field string
		run   func(id interface{}) error
	}
	calls := []call{
		{"PaymentRequests.Get", "id", func(id interface{}) error {
			_, err := client.PaymentRequests.Get(ctx, id)
			return err
		}},
		{"PaymentRequests.Cancel", "id", func(id interface{}) error {
			_, err := client.PaymentRequests.Cancel(ctx, id, nil)
			return err
		}},
		{"Invoices.Get", "id", func(id interface{}) error {
			_, err := client.PaymentRequests.Invoices.Get(ctx, id)
			return err
		}},
		{"Invoices.Download", "id", func(id interface{}) error {
			_, err := client.PaymentRequests.Invoices.Download(ctx, "inv1", id)
			return err
		}},
	}
	stringOnly := []call{
		{"Invoices.Download invoiceId", "invoiceId", func(id interface{}) error {
			_, err := client.PaymentRequests.Invoices.Download(ctx, id.(string), "pl1")
			return err
		}},
		{"Payouts.Get", "payoutId", func(id interface{}) error {
			_, err := client.Payouts.Get(ctx, id.(string))
			return err
		}},
	}

	for _, c := range append(append([]call{}, calls...), stringOnly...) {
		t.Run(c.name+"/empty", func(t *testing.T) {
			assertPayOSError(t, c.run(""), "invalid params")
		})
		t.Run(c.name+"/single dot", func(t *testing.T) {
			assertPayOSError(t, c.run("."), c.field+" must not be a dot segment")
		})
		t.Run(c.name+"/double dot", func(t *testing.T) {
			assertPayOSError(t, c.run(".."), c.field+" must not be a dot segment")
		})
	}
	for _, c := range calls {
		t.Run(c.name+"/negative", func(t *testing.T) {
			assertPayOSError(t, c.run(-1), "id must be a non-negative safe integer")
		})
		t.Run(c.name+"/above safe integer", func(t *testing.T) {
			assertPayOSError(t, c.run(int64(9007199254740992)), "id must be a non-negative safe integer")
		})
	}
	if uris := rec.take(); len(uris) != 0 {
		t.Fatalf("expected no requests for rejected ids, got %v", uris)
	}
}

// recordingTransport answers every request locally and records its escaped path.
type recordingTransport struct {
	mu    sync.Mutex
	paths []string
}

func (rt *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.mu.Lock()
	rt.paths = append(rt.paths, req.URL.EscapedPath())
	rt.mu.Unlock()
	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "application/json")
	_, _ = rec.WriteString(`{"code":"101","desc":"not found","data":null}`)
	resp := rec.Result()
	resp.Request = req
	return resp, nil
}

func TestLegacyFunctionsEncodeOrderCode(t *testing.T) {
	rt := &recordingTransport{}
	orig := http.DefaultTransport
	http.DefaultTransport = rt
	origClient := http.DefaultClient.Transport
	http.DefaultClient.Transport = rt
	t.Cleanup(func() {
		http.DefaultTransport = orig
		http.DefaultClient.Transport = origClient
	})
	if err := Key("client-id", "api-key", "checksum-key"); err != nil {
		t.Fatalf("Key: %v", err)
	}

	for _, tc := range stringIdCases {
		t.Run("GetPaymentLinkInformation/"+tc.name, func(t *testing.T) {
			rt.paths = nil
			_, _ = GetPaymentLinkInformation(tc.id)
			if len(rt.paths) != 1 || rt.paths[0] != "/v2/payment-requests/"+tc.encoded {
				t.Fatalf("paths = %v, want [/v2/payment-requests/%s]", rt.paths, tc.encoded)
			}
		})
		t.Run("CancelPaymentLink/"+tc.name, func(t *testing.T) {
			rt.paths = nil
			_, _ = CancelPaymentLink(tc.id, nil)
			if len(rt.paths) != 1 || rt.paths[0] != "/v2/payment-requests/"+tc.encoded+"/cancel" {
				t.Fatalf("paths = %v, want [/v2/payment-requests/%s/cancel]", rt.paths, tc.encoded)
			}
		})
	}

	for _, id := range []string{".", ".."} {
		rt.paths = nil
		_, err := GetPaymentLinkInformation(id)
		assertPayOSError(t, err, "orderCode must not be a dot segment")
		_, err = CancelPaymentLink(id, nil)
		assertPayOSError(t, err, "orderCode must not be a dot segment")
		if len(rt.paths) != 0 {
			t.Fatalf("expected no requests for %q, got %v", id, rt.paths)
		}
	}
	_, err := GetPaymentLinkInformation("")
	assertPayOSError(t, err, "invalid params")
	_, err = CancelPaymentLink("", nil)
	assertPayOSError(t, err, "invalid params")
}
