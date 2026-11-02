package payos

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

var credentialHeaders = []string{
	"x-api-key",
	"x-client-id",
	"x-partner-code",
	"authorization",
	"cookie",
	"proxy-authorization",
}

// extraCredentialHeaders are sent on top of the headers the client sets itself.
var extraCredentialHeaders = map[string]string{
	"Authorization":       "Bearer token",
	"Cookie":              "session=1",
	"Proxy-Authorization": "Basic cHJveHk=",
}

type redirectRoute struct {
	status   int
	location string
}

type recordedHop struct {
	server string
	method string
	path   string
	header http.Header
	body   string
}

// redirectLab runs two local servers on different origins: "a" (127.0.0.1) and "b" (localhost).
type redirectLab struct {
	mu     sync.Mutex
	routes map[string]redirectRoute // key: server + path
	hops   []recordedHop
	a, b   *httptest.Server
	aURL   string
	bURL   string
}

func newRedirectLab(t *testing.T) *redirectLab {
	t.Helper()
	lab := &redirectLab{routes: map[string]redirectRoute{}}
	handler := func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			lab.mu.Lock()
			lab.hops = append(lab.hops, recordedHop{
				server: name,
				method: r.Method,
				path:   r.URL.Path,
				header: r.Header.Clone(),
				body:   string(body),
			})
			route, ok := lab.routes[name+r.URL.Path]
			lab.mu.Unlock()
			if ok {
				w.Header().Set("Location", route.location)
				w.WriteHeader(route.status)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":"00","desc":"success","data":{"ok":true}}`))
		}
	}
	lab.a = httptest.NewServer(handler("a"))
	lab.b = httptest.NewServer(handler("b"))
	t.Cleanup(lab.a.Close)
	t.Cleanup(lab.b.Close)
	lab.aURL = lab.a.URL
	u, err := url.Parse(lab.b.URL)
	if err != nil {
		t.Fatal(err)
	}
	lab.bURL = "http://localhost:" + u.Port()
	return lab
}

func (lab *redirectLab) route(server, path string, status int, location string) {
	lab.mu.Lock()
	defer lab.mu.Unlock()
	lab.routes[server+path] = redirectRoute{status: status, location: location}
}

func (lab *redirectLab) recorded() []recordedHop {
	lab.mu.Lock()
	defer lab.mu.Unlock()
	return append([]recordedHop(nil), lab.hops...)
}

func (lab *redirectLab) client(t *testing.T, httpClient *http.Client) *Client {
	t.Helper()
	c, err := NewClient(&PayOSOptions{
		ClientId:    "client-id",
		ApiKey:      "api-key",
		ChecksumKey: "checksum-key",
		PartnerCode: "partner-code",
		BaseURL:     lab.aURL,
		HTTPClient:  httpClient,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func assertHeadersPresent(t *testing.T, hop recordedHop) {
	t.Helper()
	for _, h := range credentialHeaders {
		if hop.header.Get(h) == "" {
			t.Errorf("hop %s%s: header %s missing", hop.server, hop.path, h)
		}
	}
}

func assertHeadersAbsent(t *testing.T, hop recordedHop) {
	t.Helper()
	for _, h := range credentialHeaders {
		if v := hop.header.Get(h); v != "" {
			t.Errorf("hop %s%s: header %s = %q, want removed", hop.server, hop.path, h, v)
		}
	}
}

func expectHops(t *testing.T, hops []recordedHop, want ...string) {
	t.Helper()
	got := make([]string, len(hops))
	for i, h := range hops {
		got[i] = h.server + h.path
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("hops = %v, want %v", got, want)
	}
}

func TestCrossOriginRedirectRemovesCredentialHeaders(t *testing.T) {
	for _, status := range []int{http.StatusFound, http.StatusTemporaryRedirect} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			lab := newRedirectLab(t)
			lab.route("a", "/start", status, lab.bURL+"/end")
			c := lab.client(t, nil)
			if _, err := c.Get(context.Background(), "/start", nil, extraCredentialHeaders, nil); err != nil {
				t.Fatalf("Get: %v", err)
			}
			hops := lab.recorded()
			expectHops(t, hops, "a/start", "b/end")
			assertHeadersPresent(t, hops[0])
			assertHeadersAbsent(t, hops[1])
		})
	}
}

func TestCrossOriginRedirectToOtherPortRemovesCredentialHeaders(t *testing.T) {
	lab := newRedirectLab(t)
	// Same host name (127.0.0.1), different port.
	lab.route("a", "/start", http.StatusFound, lab.b.URL+"/end")
	c := lab.client(t, nil)
	if _, err := c.Get(context.Background(), "/start", nil, extraCredentialHeaders, nil); err != nil {
		t.Fatalf("Get: %v", err)
	}
	hops := lab.recorded()
	expectHops(t, hops, "a/start", "b/end")
	assertHeadersAbsent(t, hops[1])
}

func TestSameOriginRedirectKeepsCredentialHeaders(t *testing.T) {
	lab := newRedirectLab(t)
	lab.route("a", "/start", http.StatusFound, "/end")
	c := lab.client(t, nil)
	if _, err := c.Get(context.Background(), "/start", nil, extraCredentialHeaders, nil); err != nil {
		t.Fatalf("Get: %v", err)
	}
	hops := lab.recorded()
	expectHops(t, hops, "a/start", "a/end")
	assertHeadersPresent(t, hops[1])
}

func TestCredentialHeadersStayRemovedAfterReturningToOrigin(t *testing.T) {
	lab := newRedirectLab(t)
	lab.route("a", "/start", http.StatusFound, lab.bURL+"/middle")
	lab.route("b", "/middle", http.StatusFound, lab.aURL+"/end")
	c := lab.client(t, nil)
	if _, err := c.Get(context.Background(), "/start", nil, extraCredentialHeaders, nil); err != nil {
		t.Fatalf("Get: %v", err)
	}
	hops := lab.recorded()
	expectHops(t, hops, "a/start", "b/middle", "a/end")
	assertHeadersAbsent(t, hops[1])
	assertHeadersAbsent(t, hops[2])
}

func TestRedirectSwitchesPostToGetWithoutBody(t *testing.T) {
	for _, status := range []int{http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			lab := newRedirectLab(t)
			lab.route("a", "/start", status, "/end")
			c := lab.client(t, nil)
			if _, err := c.Post(context.Background(), "/start", map[string]string{"k": "v"}, nil, nil); err != nil {
				t.Fatalf("Post: %v", err)
			}
			hops := lab.recorded()
			expectHops(t, hops, "a/start", "a/end")
			if hops[0].method != http.MethodPost || hops[0].body == "" {
				t.Fatalf("first hop = %s %q, want POST with body", hops[0].method, hops[0].body)
			}
			end := hops[1]
			if end.method != http.MethodGet {
				t.Errorf("method = %s, want GET", end.method)
			}
			if end.body != "" {
				t.Errorf("body = %q, want empty", end.body)
			}
			if ct := end.header.Get("Content-Type"); ct != "" {
				t.Errorf("Content-Type = %q, want removed", ct)
			}
		})
	}
}

func TestRedirectKeepsMethodAndBody(t *testing.T) {
	for _, status := range []int{http.StatusTemporaryRedirect, http.StatusPermanentRedirect} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			lab := newRedirectLab(t)
			lab.route("a", "/start", status, "/end")
			c := lab.client(t, nil)
			if _, err := c.Post(context.Background(), "/start", map[string]string{"k": "v"}, nil, nil); err != nil {
				t.Fatalf("Post: %v", err)
			}
			hops := lab.recorded()
			expectHops(t, hops, "a/start", "a/end")
			end := hops[1]
			if end.method != http.MethodPost {
				t.Errorf("method = %s, want POST", end.method)
			}
			if end.body != `{"k":"v"}` {
				t.Errorf("body = %q, want {\"k\":\"v\"}", end.body)
			}
			if ct := end.header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
		})
	}
}

func chainRedirects(lab *redirectLab, n int) {
	for i := 0; i < n; i++ {
		lab.route("a", "/r"+string(rune('0'+i)), http.StatusFound, "/r"+string(rune('0'+i+1)))
	}
}

func TestFiveRedirectsAreFollowed(t *testing.T) {
	lab := newRedirectLab(t)
	chainRedirects(lab, 5)
	c := lab.client(t, nil)
	if _, err := c.Get(context.Background(), "/r0", nil, nil, nil); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if hops := lab.recorded(); len(hops) != 6 {
		t.Fatalf("got %d requests, want 6", len(hops))
	}
}

func TestMoreThanFiveRedirectsFail(t *testing.T) {
	lab := newRedirectLab(t)
	chainRedirects(lab, 6)
	c, err := NewClient(&PayOSOptions{
		ClientId:    "client-id",
		ApiKey:      "api-key",
		ChecksumKey: "checksum-key",
		BaseURL:     lab.aURL,
	})
	if err != nil {
		t.Fatal(err)
	}
	c.maxRetries = 0
	_, err = c.Get(context.Background(), "/r0", nil, nil, nil)
	if err == nil {
		t.Fatal("expected an error for more than 5 redirects")
	}
	if !strings.Contains(err.Error(), "too many redirects (more than 5)") {
		t.Fatalf("error = %v, want too many redirects (more than 5)", err)
	}
	if hops := lab.recorded(); len(hops) != 6 {
		t.Fatalf("got %d requests, want 6", len(hops))
	}
}

func TestCustomHTTPClientWithoutCheckRedirectRemovesCredentialHeaders(t *testing.T) {
	lab := newRedirectLab(t)
	lab.route("a", "/start", http.StatusFound, lab.bURL+"/end")
	custom := &http.Client{}
	c := lab.client(t, custom)
	if _, err := c.Get(context.Background(), "/start", nil, extraCredentialHeaders, nil); err != nil {
		t.Fatalf("Get: %v", err)
	}
	hops := lab.recorded()
	expectHops(t, hops, "a/start", "b/end")
	assertHeadersAbsent(t, hops[1])
	if custom.CheckRedirect != nil {
		t.Fatal("caller's http.Client was modified")
	}
}

func TestCustomCheckRedirectRunsAfterCredentialHeadersAreRemoved(t *testing.T) {
	lab := newRedirectLab(t)
	lab.route("a", "/start", http.StatusFound, lab.bURL+"/end")

	var seen []http.Header
	custom := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		seen = append(seen, req.Header.Clone())
		return nil
	}}
	c := lab.client(t, custom)
	if _, err := c.Get(context.Background(), "/start", nil, extraCredentialHeaders, nil); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(seen) != 1 {
		t.Fatalf("caller CheckRedirect ran %d times, want 1", len(seen))
	}
	for _, h := range credentialHeaders {
		if v := seen[0].Get(h); v != "" {
			t.Errorf("caller CheckRedirect saw %s = %q, want removed", h, v)
		}
	}
	hops := lab.recorded()
	expectHops(t, hops, "a/start", "b/end")
	assertHeadersAbsent(t, hops[1])
}

func TestCustomCheckRedirectCanStopRedirects(t *testing.T) {
	lab := newRedirectLab(t)
	lab.route("a", "/start", http.StatusFound, lab.bURL+"/end")
	stop := errors.New("redirects disabled by caller")
	custom := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return stop
	}}
	c := lab.client(t, custom)
	c.maxRetries = 0
	if _, err := c.Get(context.Background(), "/start", nil, nil, nil); err == nil || !errors.Is(err, stop) {
		t.Fatalf("error = %v, want caller's CheckRedirect error", err)
	}
	expectHops(t, lab.recorded(), "a/start")
}

func TestCustomCheckRedirectStillLimitsRedirectCount(t *testing.T) {
	lab := newRedirectLab(t)
	chainRedirects(lab, 6)
	custom := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return nil }}
	c := lab.client(t, custom)
	c.maxRetries = 0
	_, err := c.Get(context.Background(), "/r0", nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "too many redirects (more than 5)") {
		t.Fatalf("error = %v, want too many redirects (more than 5)", err)
	}
}
