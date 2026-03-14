package middleware

import (
	"context"
	"fmt"
	"testing"
)

// BenchmarkChain_Execute measures the overhead of executing a middleware chain
// with varying numbers of middleware layers.
//
// Run with:
//
//	go test -bench=BenchmarkChain -benchtime=5s ./pkg/middleware/
func BenchmarkChain_Execute(b *testing.B) {
	req := &Request{URL: "http://example.com/bench"}

	for _, size := range []int{0, 1, 3, 5, 10} {
		size := size
		b.Run(fmt.Sprintf("layers=%d", size), func(b *testing.B) {
			c := NewChain(echoHandler)
			for i := 0; i < size; i++ {
				c.UseFunc(func(ctx context.Context, r *Request, next NextFunc) (*Response, error) {
					return next(ctx, r)
				})
			}

			ctx := context.Background()

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _ = c.Execute(ctx, req)
			}
		})
	}
}
