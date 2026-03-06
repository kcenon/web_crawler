package middleware

import (
	"context"
	"errors"
	"testing"
)

// echoHandler is a terminal handler that returns a 200 response
// echoing the request URL in the body.
func echoHandler(_ context.Context, req *Request) (*Response, error) {
	return &Response{
		StatusCode: 200,
		Body:       []byte(req.URL),
	}, nil
}

func TestChain_EmptyChain(t *testing.T) {
	c := NewChain(echoHandler)

	resp, err := c.Execute(context.Background(), &Request{URL: "http://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if string(resp.Body) != "http://example.com" {
		t.Errorf("Body = %q, want %q", string(resp.Body), "http://example.com")
	}
}

func TestChain_ExecutionOrder(t *testing.T) {
	var order []string

	mw1 := Func(func(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
		order = append(order, "mw1-before")
		resp, err := next(ctx, req)
		order = append(order, "mw1-after")
		return resp, err
	})

	mw2 := Func(func(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
		order = append(order, "mw2-before")
		resp, err := next(ctx, req)
		order = append(order, "mw2-after")
		return resp, err
	})

	mw3 := Func(func(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
		order = append(order, "mw3-before")
		resp, err := next(ctx, req)
		order = append(order, "mw3-after")
		return resp, err
	})

	c := NewChain(func(ctx context.Context, req *Request) (*Response, error) {
		order = append(order, "handler")
		return echoHandler(ctx, req)
	})
	c.Use(mw1)
	c.Use(mw2)
	c.Use(mw3)

	_, err := c.Execute(context.Background(), &Request{URL: "http://example.com"})
	if err != nil {
		t.Fatal(err)
	}

	// Onion model: mw1 -> mw2 -> mw3 -> handler -> mw3 -> mw2 -> mw1
	expected := []string{
		"mw1-before", "mw2-before", "mw3-before",
		"handler",
		"mw3-after", "mw2-after", "mw1-after",
	}

	if len(order) != len(expected) {
		t.Fatalf("order length = %d, want %d: %v", len(order), len(expected), order)
	}

	for i, want := range expected {
		if order[i] != want {
			t.Errorf("order[%d] = %q, want %q", i, order[i], want)
		}
	}
}

func TestChain_RequestModification(t *testing.T) {
	addHeader := Func(func(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
		if req.Headers == nil {
			req.Headers = make(map[string]string)
		}
		req.Headers["X-Added"] = "by-middleware"
		return next(ctx, req)
	})

	var gotHeader string
	c := NewChain(func(_ context.Context, req *Request) (*Response, error) {
		gotHeader = req.Headers["X-Added"]
		return &Response{StatusCode: 200}, nil
	})
	c.Use(addHeader)

	_, err := c.Execute(context.Background(), &Request{URL: "http://example.com"})
	if err != nil {
		t.Fatal(err)
	}

	if gotHeader != "by-middleware" {
		t.Errorf("header = %q, want %q", gotHeader, "by-middleware")
	}
}

func TestChain_ResponseModification(t *testing.T) {
	modifyResponse := Func(func(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
		resp, err := next(ctx, req)
		if err != nil {
			return nil, err
		}
		if resp.Headers == nil {
			resp.Headers = make(map[string]string)
		}
		resp.Headers["X-Modified"] = "true"
		return resp, nil
	})

	c := NewChain(echoHandler)
	c.Use(modifyResponse)

	resp, err := c.Execute(context.Background(), &Request{URL: "http://example.com"})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Headers["X-Modified"] != "true" {
		t.Errorf("missing X-Modified header")
	}
}

func TestChain_ShortCircuit(t *testing.T) {
	handlerCalled := false
	cachedResponse := &Response{StatusCode: 200, Body: []byte("cached")}

	cacheMiddleware := Func(func(_ context.Context, _ *Request, _ NextFunc) (*Response, error) {
		// Short-circuit: return without calling next.
		return cachedResponse, nil
	})

	c := NewChain(func(_ context.Context, _ *Request) (*Response, error) {
		handlerCalled = true
		return &Response{StatusCode: 200}, nil
	})
	c.Use(cacheMiddleware)

	resp, err := c.Execute(context.Background(), &Request{URL: "http://example.com"})
	if err != nil {
		t.Fatal(err)
	}

	if handlerCalled {
		t.Error("handler should not be called when middleware short-circuits")
	}
	if string(resp.Body) != "cached" {
		t.Errorf("Body = %q, want %q", string(resp.Body), "cached")
	}
}

func TestChain_ErrorPropagation(t *testing.T) {
	errTest := errors.New("test error")

	errorMiddleware := Func(func(_ context.Context, _ *Request, _ NextFunc) (*Response, error) {
		return nil, errTest
	})

	c := NewChain(echoHandler)
	c.Use(errorMiddleware)

	_, err := c.Execute(context.Background(), &Request{URL: "http://example.com"})
	if !errors.Is(err, errTest) {
		t.Errorf("error = %v, want %v", err, errTest)
	}
}

func TestChain_ErrorFromHandler(t *testing.T) {
	errHandler := errors.New("handler error")

	var mwGotError bool
	errorCatcher := Func(func(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
		resp, err := next(ctx, req)
		if err != nil {
			mwGotError = true
		}
		return resp, err
	})

	c := NewChain(func(_ context.Context, _ *Request) (*Response, error) {
		return nil, errHandler
	})
	c.Use(errorCatcher)

	_, err := c.Execute(context.Background(), &Request{URL: "http://example.com"})
	if !errors.Is(err, errHandler) {
		t.Errorf("error = %v, want %v", err, errHandler)
	}
	if !mwGotError {
		t.Error("middleware should see handler error")
	}
}

func TestChain_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	contextChecker := Func(func(ctx context.Context, _ *Request, _ NextFunc) (*Response, error) {
		return nil, ctx.Err()
	})

	c := NewChain(echoHandler)
	c.Use(contextChecker)

	_, err := c.Execute(ctx, &Request{URL: "http://example.com"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

func TestChain_MetaPropagation(t *testing.T) {
	setMeta := Func(func(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
		if req.Meta == nil {
			req.Meta = make(map[string]any)
		}
		req.Meta["key"] = "value"
		return next(ctx, req)
	})

	var gotMeta any
	c := NewChain(func(_ context.Context, req *Request) (*Response, error) {
		gotMeta = req.Meta["key"]
		return &Response{StatusCode: 200}, nil
	})
	c.Use(setMeta)

	_, err := c.Execute(context.Background(), &Request{URL: "http://example.com"})
	if err != nil {
		t.Fatal(err)
	}

	if gotMeta != "value" {
		t.Errorf("meta = %v, want %q", gotMeta, "value")
	}
}

func TestChain_UseFunc(t *testing.T) {
	c := NewChain(echoHandler)

	called := false
	c.UseFunc(func(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
		called = true
		return next(ctx, req)
	})

	_, err := c.Execute(context.Background(), &Request{URL: "http://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("UseFunc middleware not called")
	}
}

func TestChain_Len(t *testing.T) {
	c := NewChain(echoHandler)
	if c.Len() != 0 {
		t.Errorf("initial Len = %d, want 0", c.Len())
	}
	c.UseFunc(func(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
		return next(ctx, req)
	})
	if c.Len() != 1 {
		t.Errorf("Len after Use = %d, want 1", c.Len())
	}
}
