package payos

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

const maxRedirects = 5

// errTooManyRedirects ends a redirect chain; the client recognises it and does not retry.
var errTooManyRedirects = errors.New("too many redirects (more than 5)")

// leftOriginKey marks a redirect request whose chain has left the original origin.
type leftOriginKey struct{}

// sensitiveRedirectHeaders carry credentials that must never reach another origin.
var sensitiveRedirectHeaders = []string{
	"x-api-key",
	"x-client-id",
	"x-partner-code",
	"authorization",
	"cookie",
	"proxy-authorization",
}

// urlOrigin returns scheme://host:port with the default port made explicit, so that
// https://a and https://a:443 are the same origin.
func urlOrigin(u *url.URL) string {
	scheme := strings.ToLower(u.Scheme)
	port := u.Port()
	if port == "" {
		switch scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		}
	}
	return scheme + "://" + strings.ToLower(u.Hostname()) + ":" + port
}

// safeCheckRedirect strips credential headers once a redirect chain leaves the origin of the
// original request. net/http re-copies the original headers on every hop and only removes
// Authorization/Cookie when the target is not the same domain or a subdomain, so custom headers
// such as x-api-key would otherwise be forwarded to the new host. Once the chain has left the
// origin, credentials stay stripped even if a later hop returns to it. net/http adds cookie-jar
// cookies after this function runs, so the request is also marked for redirectCredentialTransport.
//
// On 301/302/303 net/http re-issues the request without its body (as GET for non-GET methods);
// Content-Type is dropped for those hops as well.
func safeCheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) > maxRedirects {
		return errTooManyRedirects
	}
	if len(via) == 0 {
		return nil
	}
	origin := urlOrigin(via[0].URL)
	leftOrigin := urlOrigin(req.URL) != origin
	for _, prev := range via[1:] {
		if leftOrigin {
			break
		}
		leftOrigin = urlOrigin(prev.URL) != origin
	}
	if leftOrigin {
		for _, h := range sensitiveRedirectHeaders {
			req.Header.Del(h)
		}
		*req = *req.WithContext(context.WithValue(req.Context(), leftOriginKey{}, true))
	}
	if req.Response != nil {
		switch req.Response.StatusCode {
		case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther:
			req.Header.Del("Content-Type")
		}
	}
	return nil
}

// redirectCredentialTransport removes credential headers, including cookies added by a cookie jar,
// from requests marked by safeCheckRedirect. A nil base means http.DefaultTransport.
type redirectCredentialTransport struct {
	base http.RoundTripper
}

func (t *redirectCredentialTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	if left, _ := req.Context().Value(leftOriginKey{}).(bool); left {
		req = req.Clone(req.Context())
		for _, h := range sensitiveRedirectHeaders {
			req.Header.Del(h)
		}
	}
	return base.RoundTrip(req)
}

// withSafeRedirects returns a shallow copy of c (or a new client when c is nil) whose redirect
// policy removes credentials on cross-origin redirects and caps the chain at maxRedirects. A
// caller-supplied CheckRedirect still runs, after the credentials have been removed.
func withSafeRedirects(c *http.Client) *http.Client {
	if c == nil {
		return &http.Client{
			CheckRedirect: safeCheckRedirect,
			Transport:     &redirectCredentialTransport{},
		}
	}
	clone := *c
	clone.Transport = &redirectCredentialTransport{base: c.Transport}
	userCheck := c.CheckRedirect
	if userCheck == nil {
		clone.CheckRedirect = safeCheckRedirect
		return &clone
	}
	clone.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if err := safeCheckRedirect(req, via); err != nil {
			return err
		}
		return userCheck(req, via)
	}
	return &clone
}
