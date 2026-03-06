package middleware

import "context"

// Request represents an HTTP request flowing through the middleware chain.
type Request struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    []byte
	Meta    map[string]any // Arbitrary metadata passed between middleware.
}

// Response represents an HTTP response flowing back through the middleware chain.
type Response struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
	Meta       map[string]any
}

// NextFunc is a function that invokes the next middleware in the chain
// or the final handler if no more middleware remain.
type NextFunc func(ctx context.Context, req *Request) (*Response, error)

// Middleware processes a request and optionally delegates to the next handler.
// It follows the "onion model": each middleware wraps the inner handler,
// seeing the request on the way in and the response on the way out.
type Middleware interface {
	// ProcessRequest handles the request and calls next to continue the chain.
	// A middleware may modify the request before calling next, modify the
	// response after, or short-circuit by returning without calling next.
	ProcessRequest(ctx context.Context, req *Request, next NextFunc) (*Response, error)
}

// Func is an adapter to allow use of ordinary functions as Middleware.
type Func func(ctx context.Context, req *Request, next NextFunc) (*Response, error)

// ProcessRequest implements Middleware.
func (f Func) ProcessRequest(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
	return f(ctx, req, next)
}

// Chain is an ordered collection of middleware that processes requests
// in the order they were added (first-in, first-to-process-request).
type Chain struct {
	middlewares []Middleware
	handler     NextFunc // Terminal handler that performs the actual request.
}

// NewChain creates a new middleware chain with the given terminal handler.
// The handler is called after all middleware have processed the request.
func NewChain(handler NextFunc) *Chain {
	return &Chain{handler: handler}
}

// Use appends a middleware to the chain. Middleware are executed in the
// order they are added.
func (c *Chain) Use(mw Middleware) {
	c.middlewares = append(c.middlewares, mw)
}

// UseFunc is a convenience method for adding a function as Middleware.
func (c *Chain) UseFunc(fn func(ctx context.Context, req *Request, next NextFunc) (*Response, error)) {
	c.Use(Func(fn))
}

// Execute runs the request through the middleware chain and terminal handler.
// It constructs a nested call chain where each middleware wraps the next.
func (c *Chain) Execute(ctx context.Context, req *Request) (*Response, error) {
	if len(c.middlewares) == 0 {
		return c.handler(ctx, req)
	}

	// Build the chain from the inside out.
	// The innermost function is the terminal handler.
	next := c.handler
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		mw := c.middlewares[i]
		innerNext := next // capture for closure
		next = func(ctx context.Context, req *Request) (*Response, error) {
			return mw.ProcessRequest(ctx, req, innerNext)
		}
	}

	return next(ctx, req)
}

// Len returns the number of middleware in the chain.
func (c *Chain) Len() int {
	return len(c.middlewares)
}
