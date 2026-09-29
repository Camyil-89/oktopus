package http_test

import (
	"net/url"
	"testing"

	proxyhttp "oktopus/internal/proxy/http"
)

func TestParseSearchQuery(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "nil url", raw: "", want: ""},
		{name: "google q", raw: "https://www.google.com/search?q=hello+world", want: "hello world"},
		{name: "bing q", raw: "http://bing.com/search?q=go+proxy", want: "go proxy"},
		{name: "query param", raw: "http://example.com/?query=findme", want: "findme"},
		{name: "text param", raw: "http://example.com/?text=abc", want: "abc"},
		{name: "p param", raw: "http://example.com/?p=page", want: "page"},
		{name: "wd param", raw: "http://example.com/?wd=word", want: "word"},
		{name: "priority q over others", raw: "http://example.com/?q=first&query=second", want: "first"},
		{name: "no match", raw: "http://example.com/?foo=bar", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var u *url.URL
			if tt.name != "nil url" {
				parsed, err := url.Parse(tt.raw)
				if err != nil {
					t.Fatal(err)
				}
				u = parsed
			}
			got := proxyhttp.ParseSearchQuery(u)
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestParseSearchRequest(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantEngine string
		wantQuery  string
	}{
		{name: "google search", raw: "https://www.google.com/search?q=hello+world", wantEngine: "google", wantQuery: "hello world"},
		{name: "not search path on google", raw: "https://play.google.com/store/apps/details?id=foo&p=bar", wantEngine: "", wantQuery: ""},
		{name: "generic q on site", raw: "http://example.com/?q=findme", wantEngine: "", wantQuery: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.raw)
			if err != nil {
				t.Fatal(err)
			}
			engine, query := proxyhttp.ParseSearchRequest(u)
			if engine != tt.wantEngine || query != tt.wantQuery {
				t.Fatalf("engine=%q query=%q want engine=%q query=%q", engine, query, tt.wantEngine, tt.wantQuery)
			}
		})
	}
}

func BenchmarkParseSearchQuery(b *testing.B) {
	u, err := url.Parse("https://www.google.com/search?q=performance+testing+proxy")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = proxyhttp.ParseSearchQuery(u)
	}
}
