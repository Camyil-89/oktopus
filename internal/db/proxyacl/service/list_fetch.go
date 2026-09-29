package service

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// До 500 MiB на один remote list (plain text, по строке на значение).
	listFetchMaxBytes = 500 << 20
	// Включая чтение тела ответа на медленных каналах.
	listFetchTimeout = 5 * time.Minute
)

var listFetchHTTPClient = &http.Client{
	Timeout: listFetchTimeout,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	},
}

func validateRemoteSourceURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("source_url must be a valid http or https URL")
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return nil
	default:
		return fmt.Errorf("source_url must use http or https")
	}
}

func fetchListBodyFromURL(sourceURL string) (string, error) {
	if err := validateRemoteSourceURL(sourceURL); err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodGet, strings.TrimSpace(sourceURL), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "oktopus-proxy-acl-list-fetch/1.0")
	req.Header.Set("Accept", "text/plain, text/*, */*")

	resp, err := listFetchHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	limited := io.LimitReader(resp.Body, listFetchMaxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}
	if len(data) > listFetchMaxBytes {
		return "", fmt.Errorf("response too large")
	}
	return normalizeFetchedListBody(string(data)), nil
}

func normalizeFetchedListBody(raw string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	return strings.TrimSpace(raw)
}

// canonicalListBody — единый вид тела list для сравнения и хранения.
func canonicalListBody(body string) string {
	return normalizeFetchedListBody(body)
}
