package payos

import (
	"errors"
	"net/http"
)

const maxRedirects = 10

// sensitiveRedirectHeaders carry credentials that must never reach another origin.
var sensitiveRedirectHeaders = []string{"x-api-key", "x-client-id", "x-partner-code"}

// safeCheckRedirect strips payOS credential headers when a redirect leaves the origin of the
// original request. net/http only removes Authorization/Cookie on cross-domain redirects, so
// custom headers such as x-api-key would otherwise be forwarded to the new host.
func safeCheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return errors.New("stopped after too many redirects")
	}
	if len(via) > 0 {
		orig := via[0].URL
		if req.URL.Scheme != orig.Scheme || req.URL.Host != orig.Host {
			for _, h := range sensitiveRedirectHeaders {
				req.Header.Del(h)
			}
		}
	}
	return nil
}

// withSafeRedirects returns c (or a shallow copy) whose redirect policy removes credentials on
// cross-origin redirects. A caller-supplied CheckRedirect is left untouched.
func withSafeRedirects(c *http.Client) *http.Client {
	if c == nil {
		return &http.Client{CheckRedirect: safeCheckRedirect}
	}
	if c.CheckRedirect != nil {
		return c
	}
	clone := *c
	clone.CheckRedirect = safeCheckRedirect
	return &clone
}
