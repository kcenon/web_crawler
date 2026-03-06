// Package testutil provides test helpers for the web crawler project.
package testutil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

// Page represents a mock web page served by the test HTTP server.
type Page struct {
	// Path is the URL path (e.g., "/about").
	Path string

	// StatusCode is the HTTP response status code. Default: 200.
	StatusCode int

	// ContentType is the Content-Type header. Default: "text/html".
	ContentType string

	// Body is the response body content.
	Body string

	// Headers are additional response headers.
	Headers map[string]string
}

// NewHTTPServer creates a test HTTP server that serves the given pages.
// Pages are matched by exact path. Unmatched paths return 404.
func NewHTTPServer(pages ...Page) *httptest.Server {
	mux := http.NewServeMux()

	for _, p := range pages {
		page := p // capture range variable
		mux.HandleFunc(page.Path, func(w http.ResponseWriter, _ *http.Request) {
			for k, v := range page.Headers {
				w.Header().Set(k, v)
			}

			ct := page.ContentType
			if ct == "" {
				ct = "text/html"
			}
			w.Header().Set("Content-Type", ct)

			status := page.StatusCode
			if status == 0 {
				status = http.StatusOK
			}
			w.WriteHeader(status)

			fmt.Fprint(w, page.Body)
		})
	}

	return httptest.NewServer(mux)
}

// NewLinkedHTTPServer creates a test server with pages that link to each other,
// simulating a simple website structure for crawl depth testing.
func NewLinkedHTTPServer() *httptest.Server {
	var srv *httptest.Server

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><body>
			<h1>Home</h1>
			<a href="%s/page1">Page 1</a>
			<a href="%s/page2">Page 2</a>
		</body></html>`, srv.URL, srv.URL)
	})

	mux.HandleFunc("/page1", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><body>
			<h1>Page 1</h1>
			<a href="%s/page1/sub">Subpage</a>
		</body></html>`, srv.URL)
	})

	mux.HandleFunc("/page1/sub", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><h1>Subpage</h1></body></html>`)
	})

	mux.HandleFunc("/page2", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><h1>Page 2</h1></body></html>`)
	})

	srv = httptest.NewServer(mux)
	return srv
}
