// Package httpfetch safely downloads source images over HTTP.
package httpfetch

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultMaxBytes int64 = 20 << 20

// Fetcher downloads public images while rejecting private network targets.
type Fetcher struct {
	client   *http.Client
	maxBytes int64
}

// New creates a fetcher with redirect, timeout, and SSRF protections.
func New() *Fetcher {
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
			if err != nil {
				return nil, fmt.Errorf("resolve source host: %w", err)
			}
			for _, address := range addresses {
				if !publicIP(net.IP(address.AsSlice())) {
					return nil, fmt.Errorf("source host resolves to a non-public address")
				}
			}
			if len(addresses) == 0 {
				return nil, fmt.Errorf("source host has no addresses")
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(addresses[0].String(), port))
		},
		MaxIdleConns: 20, IdleConnTimeout: 30 * time.Second,
		ResponseHeaderTimeout: 8 * time.Second,
	}
	fetcher := &Fetcher{client: &http.Client{Transport: transport, Timeout: 20 * time.Second}, maxBytes: defaultMaxBytes}
	fetcher.client.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		return validateURL(request.URL)
	}
	return fetcher
}

// Fetch downloads one image into bounded memory.
func (fetcher *Fetcher) Fetch(ctx context.Context, rawURL string) ([]byte, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse source URL: %w", err)
	}
	if err := validateURL(parsed); err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create source request: %w", err)
	}
	request.Header.Set("User-Agent", "ImageFlow/0.1")
	request.Header.Set("Accept", "image/jpeg,image/png")
	response, err := fetcher.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download source image: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("source returned HTTP %d", response.StatusCode)
	}
	if response.ContentLength > fetcher.maxBytes {
		return nil, fmt.Errorf("source image exceeds %d bytes", fetcher.maxBytes)
	}
	contentType := strings.ToLower(strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0]))
	if contentType != "image/jpeg" && contentType != "image/png" {
		return nil, fmt.Errorf("unsupported source content type %q", contentType)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, fetcher.maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read source image: %w", err)
	}
	if int64(len(data)) > fetcher.maxBytes {
		return nil, fmt.Errorf("source image exceeds %d bytes", fetcher.maxBytes)
	}
	return data, nil
}

func validateURL(value *url.URL) error {
	if value.Scheme != "http" && value.Scheme != "https" {
		return fmt.Errorf("source URL must use http or https")
	}
	if value.Hostname() == "" || value.User != nil {
		return fmt.Errorf("source URL has an invalid authority")
	}
	if ip := net.ParseIP(value.Hostname()); ip != nil && !publicIP(ip) {
		return fmt.Errorf("source URL targets a non-public address")
	}
	return nil
}

func publicIP(ip net.IP) bool {
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast()
}
