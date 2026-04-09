package codex

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

// HTTPClient wraps a tls-client HTTP client that impersonates a real Chrome
// TLS + HTTP/2 fingerprint. This is required to get past Cloudflare's bot
// management on auth.openai.com — Go's stdlib net/http fingerprint is
// trivially detected and returns 403 regardless of User-Agent headers.
type HTTPClient struct {
	client tls_client.HttpClient
}

// chromeHeaderOrder is the order Chrome sends headers in. tls-client uses the
// fhttp.HeaderOrderKey entry to serialize outgoing headers in this exact order,
// which is another signal Cloudflare checks.
var chromeHeaderOrder = []string{
	"host",
	"connection",
	"content-length",
	"sec-ch-ua",
	"sec-ch-ua-mobile",
	"sec-ch-ua-platform",
	"upgrade-insecure-requests",
	"user-agent",
	"accept",
	"content-type",
	"origin",
	"sec-fetch-site",
	"sec-fetch-mode",
	"sec-fetch-user",
	"sec-fetch-dest",
	"referer",
	"accept-encoding",
	"accept-language",
	"cookie",
	"oai-device-id",
	"openai-sentinel-token",
}

// NewHTTPClient creates a client with cookie jar and optional proxy.
func NewHTTPClient(proxyURL string) *HTTPClient {
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Chrome_146),
		tls_client.WithRandomTLSExtensionOrder(),
		tls_client.WithInsecureSkipVerify(),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
		// Start with redirects enabled; toggled per-call via SetFollowRedirect.
	}
	if proxyURL != "" {
		opts = append(opts, tls_client.WithProxyUrl(proxyURL))
	}

	c, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
	if err != nil {
		// Fall back to a noop client; every request will error but we never
		// panic in construction.
		return &HTTPClient{}
	}
	return &HTTPClient{client: c}
}

// applyHeaders sets headers on a request with Chrome-like ordering.
func applyHeaders(req *fhttp.Request, headers map[string]string) {
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header[fhttp.HeaderOrderKey] = chromeHeaderOrder
}

// do executes the request, optionally following redirects, and returns the
// body plus a net/http.Response shim so callers can keep using standard types.
func (c *HTTPClient) do(req *fhttp.Request, followRedirect bool) ([]byte, *http.Response, error) {
	if c.client == nil {
		return nil, nil, fmt.Errorf("tls-client not initialized")
	}
	c.client.SetFollowRedirect(followRedirect)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body, convertResp(resp), nil
}

// convertResp converts fhttp.Response into a net/http.Response so that
// existing call-site code (which imports net/http) does not need to change.
// Only fields accessed by callers are populated.
func convertResp(fr *fhttp.Response) *http.Response {
	if fr == nil {
		return nil
	}
	h := make(http.Header, len(fr.Header))
	for k, v := range fr.Header {
		h[k] = append([]string(nil), v...)
	}
	out := &http.Response{
		StatusCode: fr.StatusCode,
		Status:     fr.Status,
		Header:     h,
	}
	if fr.Request != nil && fr.Request.URL != nil {
		// fhttp.Request.URL is a standard *net/url.URL.
		out.Request = &http.Request{URL: fr.Request.URL}
	}
	return out
}

// Get performs a GET request with headers, following redirects.
func (c *HTTPClient) Get(rawURL string, headers map[string]string) ([]byte, *http.Response, error) {
	req, err := fhttp.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, nil, err
	}
	applyHeaders(req, headers)
	return c.do(req, true)
}

// GetNoRedirect performs GET without following redirects.
func (c *HTTPClient) GetNoRedirect(rawURL string, headers map[string]string) ([]byte, *http.Response, error) {
	req, err := fhttp.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, nil, err
	}
	applyHeaders(req, headers)
	return c.do(req, false)
}

// Post performs a JSON POST request.
func (c *HTTPClient) Post(rawURL string, headers map[string]string, body []byte) ([]byte, error) {
	req, err := fhttp.NewRequest("POST", rawURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	applyHeaders(req, headers)
	respBody, resp, err := c.do(req, true)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return respBody, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody[:min(len(respBody), 200)]))
	}
	return respBody, nil
}

// PostNoRedirect performs a JSON POST without following redirects.
func (c *HTTPClient) PostNoRedirect(rawURL string, headers map[string]string, body []byte) ([]byte, *http.Response, error) {
	req, err := fhttp.NewRequest("POST", rawURL, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	applyHeaders(req, headers)
	return c.do(req, false)
}

// PostForm performs a form-urlencoded POST.
func (c *HTTPClient) PostForm(rawURL string, headers map[string]string, data url.Values) ([]byte, error) {
	req, err := fhttp.NewRequest("POST", rawURL, bytes.NewReader([]byte(data.Encode())))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	applyHeaders(req, headers)
	respBody, resp, err := c.do(req, true)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return respBody, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody[:min(len(respBody), 200)]))
	}
	return respBody, nil
}

// SetCookie sets a cookie on the jar for a given domain.
func (c *HTTPClient) SetCookie(domain, name, value string) {
	if c.client == nil {
		return
	}
	u, _ := url.Parse("https://" + domain)
	c.client.SetCookies(u, []*fhttp.Cookie{{Name: name, Value: value}})
}

// GetCookie returns a cookie value from the jar.
func (c *HTTPClient) GetCookie(domain, name string) string {
	if c.client == nil {
		return ""
	}
	u, _ := url.Parse("https://" + domain)
	for _, ck := range c.client.GetCookies(u) {
		if ck.Name == name {
			return ck.Value
		}
	}
	return ""
}
