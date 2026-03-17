package plugin

import (
	"errors"
	"sort"
	"sync"
	"testing"
)

// --- Registration & Retrieval ----------------------------------------

func TestRegistry_Storage(t *testing.T) {
	r := NewRegistry()
	p := &stubStorage{name: "postgres"}

	if err := r.RegisterStorage("postgres", p); err != nil {
		t.Fatal(err)
	}

	got, err := r.GetStorage("postgres")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name() != "postgres" {
		t.Errorf("Name() = %q, want %q", got.Name(), "postgres")
	}
}

func TestRegistry_Parser(t *testing.T) {
	r := NewRegistry()
	p := &stubParser{name: "html", contentType: "text/html"}

	if err := r.RegisterParser("html", p); err != nil {
		t.Fatal(err)
	}

	got, err := r.GetParser("html")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name() != "html" {
		t.Errorf("Name() = %q, want %q", got.Name(), "html")
	}
}

func TestRegistry_Notifier(t *testing.T) {
	r := NewRegistry()
	p := &stubNotifier{name: "webhook"}

	if err := r.RegisterNotifier("webhook", p); err != nil {
		t.Fatal(err)
	}

	got, err := r.GetNotifier("webhook")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name() != "webhook" {
		t.Errorf("Name() = %q, want %q", got.Name(), "webhook")
	}
}

func TestRegistry_Exporter(t *testing.T) {
	r := NewRegistry()
	p := &stubExporter{name: "s3"}

	if err := r.RegisterExporter("s3", p); err != nil {
		t.Fatal(err)
	}

	got, err := r.GetExporter("s3")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name() != "s3" {
		t.Errorf("Name() = %q, want %q", got.Name(), "s3")
	}
}

// --- Duplicate Registration ------------------------------------------

func TestRegistry_DuplicateStorage(t *testing.T) {
	r := NewRegistry()
	p := &stubStorage{name: "pg"}

	if err := r.RegisterStorage("pg", p); err != nil {
		t.Fatal(err)
	}

	err := r.RegisterStorage("pg", p)
	var dup *ErrDuplicatePlugin
	if !errors.As(err, &dup) {
		t.Fatalf("expected ErrDuplicatePlugin, got %v", err)
	}
	if dup.Category != "storage" || dup.Name != "pg" {
		t.Errorf("unexpected duplicate error: %v", dup)
	}
}

func TestRegistry_DuplicateParser(t *testing.T) {
	r := NewRegistry()
	p := &stubParser{name: "html", contentType: "text/html"}

	_ = r.RegisterParser("html", p)

	err := r.RegisterParser("html", p)
	var dup *ErrDuplicatePlugin
	if !errors.As(err, &dup) {
		t.Fatalf("expected ErrDuplicatePlugin, got %v", err)
	}
}

func TestRegistry_DuplicateNotifier(t *testing.T) {
	r := NewRegistry()
	p := &stubNotifier{name: "wh"}

	_ = r.RegisterNotifier("wh", p)

	err := r.RegisterNotifier("wh", p)
	var dup *ErrDuplicatePlugin
	if !errors.As(err, &dup) {
		t.Fatalf("expected ErrDuplicatePlugin, got %v", err)
	}
}

func TestRegistry_DuplicateExporter(t *testing.T) {
	r := NewRegistry()
	p := &stubExporter{name: "s3"}

	_ = r.RegisterExporter("s3", p)

	err := r.RegisterExporter("s3", p)
	var dup *ErrDuplicatePlugin
	if !errors.As(err, &dup) {
		t.Fatalf("expected ErrDuplicatePlugin, got %v", err)
	}
}

// --- Nil Registration ------------------------------------------------

func TestRegistry_NilStorage(t *testing.T) {
	r := NewRegistry()
	err := r.RegisterStorage("bad", nil)
	var nilErr *ErrNilPlugin
	if !errors.As(err, &nilErr) {
		t.Fatalf("expected ErrNilPlugin, got %v", err)
	}
}

func TestRegistry_NilParser(t *testing.T) {
	r := NewRegistry()
	err := r.RegisterParser("bad", nil)
	var nilErr *ErrNilPlugin
	if !errors.As(err, &nilErr) {
		t.Fatalf("expected ErrNilPlugin, got %v", err)
	}
}

func TestRegistry_NilNotifier(t *testing.T) {
	r := NewRegistry()
	err := r.RegisterNotifier("bad", nil)
	var nilErr *ErrNilPlugin
	if !errors.As(err, &nilErr) {
		t.Fatalf("expected ErrNilPlugin, got %v", err)
	}
}

func TestRegistry_NilExporter(t *testing.T) {
	r := NewRegistry()
	err := r.RegisterExporter("bad", nil)
	var nilErr *ErrNilPlugin
	if !errors.As(err, &nilErr) {
		t.Fatalf("expected ErrNilPlugin, got %v", err)
	}
}

// --- Not Found -------------------------------------------------------

func TestRegistry_NotFoundStorage(t *testing.T) {
	r := NewRegistry()
	_, err := r.GetStorage("nope")
	var nf *ErrPluginNotFound
	if !errors.As(err, &nf) {
		t.Fatalf("expected ErrPluginNotFound, got %v", err)
	}
}

func TestRegistry_NotFoundParser(t *testing.T) {
	r := NewRegistry()
	_, err := r.GetParser("nope")
	var nf *ErrPluginNotFound
	if !errors.As(err, &nf) {
		t.Fatalf("expected ErrPluginNotFound, got %v", err)
	}
}

func TestRegistry_NotFoundNotifier(t *testing.T) {
	r := NewRegistry()
	_, err := r.GetNotifier("nope")
	var nf *ErrPluginNotFound
	if !errors.As(err, &nf) {
		t.Fatalf("expected ErrPluginNotFound, got %v", err)
	}
}

func TestRegistry_NotFoundExporter(t *testing.T) {
	r := NewRegistry()
	_, err := r.GetExporter("nope")
	var nf *ErrPluginNotFound
	if !errors.As(err, &nf) {
		t.Fatalf("expected ErrPluginNotFound, got %v", err)
	}
}

// --- List ------------------------------------------------------------

func TestRegistry_ListStorage(t *testing.T) {
	r := NewRegistry()
	_ = r.RegisterStorage("file", &stubStorage{name: "file"})
	_ = r.RegisterStorage("pg", &stubStorage{name: "pg"})

	names := r.ListStorage()
	sort.Strings(names)

	if len(names) != 2 {
		t.Fatalf("len = %d, want 2", len(names))
	}
	if names[0] != "file" || names[1] != "pg" {
		t.Errorf("names = %v, want [file pg]", names)
	}
}

func TestRegistry_ListParsers(t *testing.T) {
	r := NewRegistry()
	_ = r.RegisterParser("html", &stubParser{name: "html"})

	names := r.ListParsers()
	if len(names) != 1 || names[0] != "html" {
		t.Errorf("names = %v, want [html]", names)
	}
}

func TestRegistry_ListEmpty(t *testing.T) {
	r := NewRegistry()
	if len(r.ListStorage()) != 0 {
		t.Error("expected empty storage list")
	}
	if len(r.ListParsers()) != 0 {
		t.Error("expected empty parser list")
	}
	if len(r.ListNotifiers()) != 0 {
		t.Error("expected empty notifier list")
	}
	if len(r.ListExporters()) != 0 {
		t.Error("expected empty exporter list")
	}
}

// --- CloseAll --------------------------------------------------------

func TestRegistry_CloseAll(t *testing.T) {
	r := NewRegistry()
	s := &stubStorage{name: "mem"}
	_ = r.RegisterStorage("mem", s)
	_ = r.RegisterParser("html", &stubParser{name: "html"})
	_ = r.RegisterNotifier("log", &stubNotifier{name: "log"})
	_ = r.RegisterExporter("s3", &stubExporter{name: "s3"})

	errs := r.CloseAll()
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if !s.closed {
		t.Error("expected storage plugin to be closed")
	}
}

// --- Concurrent Access -----------------------------------------------

func TestRegistry_ConcurrentRegister(t *testing.T) {
	r := NewRegistry()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := "storage-" + itoa(i)
			_ = r.RegisterStorage(name, &stubStorage{name: name})
		}(i)
	}
	wg.Wait()

	names := r.ListStorage()
	if len(names) != 100 {
		t.Errorf("registered %d storage plugins, want 100", len(names))
	}
}

func TestRegistry_ConcurrentReadWrite(t *testing.T) {
	r := NewRegistry()
	_ = r.RegisterStorage("base", &stubStorage{name: "base"})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _ = r.GetStorage("base")
		}()
		go func(i int) {
			defer wg.Done()
			_ = r.ListStorage()
		}(i)
	}
	wg.Wait()
}

// itoa converts an int to a string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
