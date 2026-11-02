package payos

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const maxRedirects = 5

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
// origin, credentials stay stripped even if a later hop returns to it.
//
// On 301/302/303 net/http re-issues the request without its body (as GET for non-GET methods);
// Content-Type is dropped for those hops as well.
func safeCheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) > maxRedirects {
		return fmt.Errorf("too many redirects (more than %d)", maxRedirects)
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
	}
	if req.Response != nil {
		switch req.Response.StatusCode {
		case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther:
			req.Header.Del("Content-Type")
		}
	}
	return nil
}

// withSafeRedirects returns a shallow copy of c (or a new client when c is nil) whose redirect
// policy removes credentials on cross-origin redirects and caps the chain at maxRedirects. A
// caller-supplied CheckRedirect still runs, after the credentials have been removed.
func withSafeRedirects(c *http.Client) *http.Client {
	if c == nil {
		return &http.Client{CheckRedirect: safeCheckRedirect}
	}
	clone := *c
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
