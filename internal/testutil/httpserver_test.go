package testutil

import (
	"io"
	"net/http"
	"testing"
)

func TestNewHTTPServer_ServesPages(t *testing.T) {
	srv := NewHTTPServer(
		Page{
			Path:        "/hello",
			StatusCode:  200,
			ContentType: "text/html",
			Body:        "<h1>Hello</h1>",
		},
		Page{
			Path:        "/json",
			StatusCode:  200,
			ContentType: "application/json",
			Body:        `{"key":"value"}`,
		},
	)
	defer srv.Close()

	// Test HTML page
	resp, err := http.Get(srv.URL + "/hello")
	AssertNoError(t, err)
	defer resp.Body.Close()
	AssertEqual(t, resp.StatusCode, 200)

	body, _ := io.ReadAll(resp.Body)
	AssertContains(t, string(body), "<h1>Hello</h1>")

	// Test JSON page
	resp2, err := http.Get(srv.URL + "/json")
	AssertNoError(t, err)
	defer resp2.Body.Close()
	AssertEqual(t, resp2.StatusCode, 200)

	body2, _ := io.ReadAll(resp2.Body)
	AssertContains(t, string(body2), `"key":"value"`)
}

func TestNewHTTPServer_CustomStatus(t *testing.T) {
	srv := NewHTTPServer(
		Page{Path: "/error", StatusCode: 500, Body: "Internal Server Error"},
	)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/error")
	AssertNoError(t, err)
	defer resp.Body.Close()
	AssertEqual(t, resp.StatusCode, 500)
}

func TestNewHTTPServer_DefaultValues(t *testing.T) {
	srv := NewHTTPServer(
		Page{Path: "/default", Body: "Hello"},
	)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/default")
	AssertNoError(t, err)
	defer resp.Body.Close()

	AssertEqual(t, resp.StatusCode, 200)
	ct := resp.Header.Get("Content-Type")
	AssertContains(t, ct, "text/html")
}

func TestNewLinkedHTTPServer(t *testing.T) {
	srv := NewLinkedHTTPServer()
	defer srv.Close()

	// Test home page has links
	resp, err := http.Get(srv.URL + "/")
	AssertNoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	AssertContains(t, string(body), "Page 1")
	AssertContains(t, string(body), "Page 2")

	// Test linked page
	resp2, err := http.Get(srv.URL + "/page1")
	AssertNoError(t, err)
	defer resp2.Body.Close()
	AssertEqual(t, resp2.StatusCode, 200)
}

func TestNewHTTPServer_CustomHeaders(t *testing.T) {
	srv := NewHTTPServer(
		Page{
			Path: "/headers",
			Body: "ok",
			Headers: map[string]string{
				"X-Custom": "test-value",
			},
		},
	)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/headers")
	AssertNoError(t, err)
	defer resp.Body.Close()

	AssertEqual(t, resp.Header.Get("X-Custom"), "test-value")
}
