package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkDo(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html>benchmark</html>")
	}))
	defer ts.Close()

	c, err := New(Config{})
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	defer c.Close()

	ctx := context.Background()
	req := &Request{URL: ts.URL}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := c.Do(ctx, req)
		if err != nil {
			b.Fatalf("Do() error = %v", err)
		}
		_ = resp
	}
}

func BenchmarkDo_Parallel(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html>benchmark</html>")
	}))
	defer ts.Close()

	c, err := New(Config{})
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	defer c.Close()

	ctx := context.Background()
	req := &Request{URL: ts.URL}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := c.Do(ctx, req)
			if err != nil {
				b.Errorf("Do() error = %v", err)
				return
			}
			_ = resp
		}
	})
}
