package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	debugProxyMaxBody   = 2 << 20 // 2 MiB
	debugProxyDefTimeout = 30 * time.Second
	debugProxyMaxTimeout = 120 * time.Second
)

var allowedDebugMethods = map[string]struct{}{
	http.MethodGet:     {},
	http.MethodPost:    {},
	http.MethodPut:     {},
	http.MethodPatch:   {},
	http.MethodDelete:  {},
	http.MethodHead:    {},
	http.MethodOptions: {},
}

type DebugProxyInput struct {
	Method     string            `json:"method" binding:"required"`
	URL        string            `json:"url" binding:"required"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	BodyBase64 string            `json:"body_base64"`
	TimeoutMs  int               `json:"timeout_ms"`
}

type DebugProxyResult struct {
	Status     int                 `json:"status"`
	StatusText string              `json:"status_text"`
	Headers    map[string][]string `json:"headers"`
	Body       string              `json:"body"`
	DurationMs int64               `json:"duration_ms"`
	Size       int                 `json:"size"`
	Truncated  bool                `json:"truncated"`
}

type DebugProxyService struct{}

func NewDebugProxyService() *DebugProxyService {
	return &DebugProxyService{}
}

func (s *DebugProxyService) Proxy(ctx context.Context, in DebugProxyInput) (*DebugProxyResult, error) {
	method := strings.ToUpper(strings.TrimSpace(in.Method))
	if _, ok := allowedDebugMethods[method]; !ok {
		return nil, fmt.Errorf("%w: unsupported method %s", ErrBadRequest, in.Method)
	}

	rawURL := strings.TrimSpace(in.URL)
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("%w: invalid url", ErrBadRequest)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("%w: only http/https urls are allowed", ErrBadRequest)
	}
	if err := validateDebugHost(u.Hostname()); err != nil {
		return nil, err
	}

	timeout := debugProxyDefTimeout
	if in.TimeoutMs > 0 {
		timeout = time.Duration(in.TimeoutMs) * time.Millisecond
		if timeout > debugProxyMaxTimeout {
			timeout = debugProxyMaxTimeout
		}
	}

	var bodyReader io.Reader
	if method != http.MethodGet && method != http.MethodHead {
		if strings.TrimSpace(in.BodyBase64) != "" {
			raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(in.BodyBase64))
			if err != nil {
				return nil, fmt.Errorf("%w: invalid body_base64", ErrBadRequest)
			}
			if len(raw) > 0 {
				bodyReader = bytes.NewReader(raw)
			}
		} else if in.Body != "" {
			bodyReader = strings.NewReader(in.Body)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("%w: build request failed: %v", ErrBadRequest, err)
	}
	for k, v := range in.Headers {
		name := strings.TrimSpace(k)
		if name == "" {
			continue
		}
		lower := strings.ToLower(name)
		if lower == "content-length" || lower == "connection" || lower == "transfer-encoding" ||
			lower == "keep-alive" || lower == "upgrade" || lower == "proxy-connection" {
			continue
		}
		if lower == "host" {
			req.Host = strings.TrimSpace(v)
			req.Header.Set("Host", strings.TrimSpace(v))
			continue
		}
		req.Header.Set(name, v)
	}

	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start).Milliseconds()
	if err != nil {
		return nil, fmt.Errorf("%w: upstream request failed: %v", ErrBadRequest, err)
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, debugProxyMaxBody+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %v", err)
	}
	truncated := len(raw) > debugProxyMaxBody
	if truncated {
		raw = raw[:debugProxyMaxBody]
	}

	headers := make(map[string][]string, len(resp.Header))
	for k, vals := range resp.Header {
		cp := make([]string, len(vals))
		copy(cp, vals)
		headers[k] = cp
	}

	return &DebugProxyResult{
		Status:     resp.StatusCode,
		StatusText: resp.Status,
		Headers:    headers,
		Body:       string(raw),
		DurationMs: duration,
		Size:       len(raw),
		Truncated:  truncated,
	}, nil
}

func validateDebugHost(host string) error {
	h := strings.TrimSpace(host)
	if h == "" {
		return fmt.Errorf("%w: empty host", ErrBadRequest)
	}
	// Block obvious metadata / link-local cloud endpoints while allowing private LAN
	// gateways that operators commonly debug against.
	if strings.EqualFold(h, "metadata.google.internal") {
		return fmt.Errorf("%w: host is not allowed", ErrBadRequest)
	}
	ip := net.ParseIP(h)
	if ip != nil && ip.IsLinkLocalUnicast() {
		return fmt.Errorf("%w: link-local addresses are not allowed", ErrBadRequest)
	}
	return nil
}
