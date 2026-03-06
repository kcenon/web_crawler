package extractor

import (
	"testing"
)

const testHTML = `<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body>
  <h1>Main Title</h1>
  <div class="content">
    <p>First paragraph</p>
    <p>Second paragraph</p>
  </div>
  <ul id="links">
    <li><a href="https://example.com/1">Link One</a></li>
    <li><a href="https://example.com/2">Link Two</a></li>
    <li><a href="https://example.com/3">Link Three</a></li>
  </ul>
  <span data-value="42">Span Text</span>
</body>
</html>`

const testJSON = `{
  "name": "web_crawler",
  "version": "0.1.0",
  "authors": ["Alice", "Bob", "Charlie"],
  "config": {
    "maxDepth": 3,
    "workers": 10
  },
  "items": [
    {"id": 1, "title": "First"},
    {"id": 2, "title": "Second"}
  ]
}`

// --- CSS Selector Tests ---

func TestCSS_ExtractText(t *testing.T) {
	e := New()
	results, err := e.Extract([]byte(testHTML), []ExtractionRule{
		{Name: "title", Type: TypeCSS, Selector: "h1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || len(results[0].Values) != 1 {
		t.Fatalf("expected 1 result with 1 value, got %d results", len(results))
	}
	if results[0].Values[0] != "Main Title" {
		t.Errorf("title = %q, want %q", results[0].Values[0], "Main Title")
	}
}

func TestCSS_ExtractMultipleElements(t *testing.T) {
	e := New()
	results, err := e.Extract([]byte(testHTML), []ExtractionRule{
		{Name: "paragraphs", Type: TypeCSS, Selector: ".content p"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results[0].Values) != 2 {
		t.Fatalf("got %d values, want 2", len(results[0].Values))
	}
	if results[0].Values[0] != "First paragraph" {
		t.Errorf("first = %q, want %q", results[0].Values[0], "First paragraph")
	}
}

func TestCSS_ExtractAttribute(t *testing.T) {
	e := New()
	results, err := e.Extract([]byte(testHTML), []ExtractionRule{
		{Name: "links", Type: TypeCSS, Selector: "#links a", Attribute: "href"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results[0].Values) != 3 {
		t.Fatalf("got %d hrefs, want 3", len(results[0].Values))
	}
	if results[0].Values[0] != "https://example.com/1" {
		t.Errorf("first href = %q", results[0].Values[0])
	}
}

func TestCSS_ExtractHTML(t *testing.T) {
	e := New()
	results, err := e.Extract([]byte(testHTML), []ExtractionRule{
		{Name: "title_html", Type: TypeCSS, Selector: "h1", Attribute: "html"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results[0].Values) != 1 {
		t.Fatalf("got %d values, want 1", len(results[0].Values))
	}
	if results[0].Values[0] != "Main Title" {
		t.Errorf("html = %q, want %q", results[0].Values[0], "Main Title")
	}
}

func TestCSS_ExtractDataAttribute(t *testing.T) {
	e := New()
	results, err := e.Extract([]byte(testHTML), []ExtractionRule{
		{Name: "data", Type: TypeCSS, Selector: "span[data-value]", Attribute: "data-value"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results[0].Values) != 1 || results[0].Values[0] != "42" {
		t.Errorf("data-value = %v, want [42]", results[0].Values)
	}
}

func TestCSS_NoMatch(t *testing.T) {
	e := New()
	results, err := e.Extract([]byte(testHTML), []ExtractionRule{
		{Name: "missing", Type: TypeCSS, Selector: ".nonexistent"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results[0].Values) != 0 {
		t.Errorf("expected 0 values for non-matching selector, got %d", len(results[0].Values))
	}
}

// --- XPath Tests ---

func TestXPath_ExtractText(t *testing.T) {
	e := New()
	results, err := e.Extract([]byte(testHTML), []ExtractionRule{
		{Name: "title", Type: TypeXPath, Selector: "//h1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results[0].Values) != 1 || results[0].Values[0] != "Main Title" {
		t.Errorf("XPath title = %v, want [Main Title]", results[0].Values)
	}
}

func TestXPath_ExtractMultiple(t *testing.T) {
	e := New()
	results, err := e.Extract([]byte(testHTML), []ExtractionRule{
		{Name: "links", Type: TypeXPath, Selector: "//ul[@id='links']//a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results[0].Values) != 3 {
		t.Errorf("XPath links count = %d, want 3", len(results[0].Values))
	}
}

func TestXPath_NoMatch(t *testing.T) {
	e := New()
	results, err := e.Extract([]byte(testHTML), []ExtractionRule{
		{Name: "missing", Type: TypeXPath, Selector: "//nonexistent"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results[0].Values) != 0 {
		t.Errorf("expected 0 values, got %d", len(results[0].Values))
	}
}

// --- JSON Extraction Tests ---

func TestJSON_SimpleValue(t *testing.T) {
	e := New()
	results, err := e.Extract([]byte(testJSON), []ExtractionRule{
		{Name: "name", Type: TypeJSON, Selector: "name"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Values[0] != "web_crawler" {
		t.Errorf("name = %q, want %q", results[0].Values[0], "web_crawler")
	}
}

func TestJSON_NestedValue(t *testing.T) {
	e := New()
	results, err := e.Extract([]byte(testJSON), []ExtractionRule{
		{Name: "maxDepth", Type: TypeJSON, Selector: "config.maxDepth"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Values[0] != "3" {
		t.Errorf("maxDepth = %q, want %q", results[0].Values[0], "3")
	}
}

func TestJSON_Array(t *testing.T) {
	e := New()
	results, err := e.Extract([]byte(testJSON), []ExtractionRule{
		{Name: "authors", Type: TypeJSON, Selector: "authors"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results[0].Values) != 3 {
		t.Fatalf("authors count = %d, want 3", len(results[0].Values))
	}
	if results[0].Values[0] != "Alice" {
		t.Errorf("first author = %q, want %q", results[0].Values[0], "Alice")
	}
}

func TestJSON_ArrayQuery(t *testing.T) {
	e := New()
	results, err := e.Extract([]byte(testJSON), []ExtractionRule{
		{Name: "titles", Type: TypeJSON, Selector: "items.#.title"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results[0].Values) != 2 {
		t.Fatalf("titles count = %d, want 2", len(results[0].Values))
	}
	if results[0].Values[0] != "First" {
		t.Errorf("first title = %q, want %q", results[0].Values[0], "First")
	}
}

func TestJSON_NonexistentPath(t *testing.T) {
	e := New()
	results, err := e.Extract([]byte(testJSON), []ExtractionRule{
		{Name: "missing", Type: TypeJSON, Selector: "nonexistent.path"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Values != nil {
		t.Errorf("expected nil values for missing path, got %v", results[0].Values)
	}
}

// --- Regex Tests ---

func TestRegex_BasicMatch(t *testing.T) {
	content := []byte("Price: $19.99, Sale: $9.99")
	e := New()
	results, err := e.Extract(content, []ExtractionRule{
		{Name: "prices", Type: TypeRegex, Selector: `\$\d+\.\d+`},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results[0].Values) != 2 {
		t.Fatalf("got %d matches, want 2", len(results[0].Values))
	}
	if results[0].Values[0] != "$19.99" {
		t.Errorf("first = %q, want %q", results[0].Values[0], "$19.99")
	}
}

func TestRegex_NamedCaptureGroup(t *testing.T) {
	content := []byte("email: alice@example.com, bob@test.com")
	e := New()
	results, err := e.Extract(content, []ExtractionRule{
		{Name: "emails", Type: TypeRegex, Selector: `(?P<email>\w+@\w+\.\w+)`},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results[0].Values) != 2 {
		t.Fatalf("got %d matches, want 2", len(results[0].Values))
	}
	if results[0].Values[0] != "alice@example.com" {
		t.Errorf("first email = %q", results[0].Values[0])
	}
}

func TestRegex_NoMatch(t *testing.T) {
	e := New()
	results, err := e.Extract([]byte("no numbers here"), []ExtractionRule{
		{Name: "nums", Type: TypeRegex, Selector: `\d+`},
	})
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Values != nil {
		t.Errorf("expected nil values, got %v", results[0].Values)
	}
}

func TestRegex_InvalidPattern(t *testing.T) {
	e := New()
	_, err := e.Extract([]byte("test"), []ExtractionRule{
		{Name: "bad", Type: TypeRegex, Selector: `[invalid`},
	})
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

// --- Engine Tests ---

func TestEngine_MultipleRules(t *testing.T) {
	e := New()
	results, err := e.Extract([]byte(testHTML), []ExtractionRule{
		{Name: "title", Type: TypeCSS, Selector: "h1"},
		{Name: "links", Type: TypeCSS, Selector: "#links a", Attribute: "href"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].Name != "title" || results[1].Name != "links" {
		t.Errorf("names = [%q, %q], want [title, links]", results[0].Name, results[1].Name)
	}
}

func TestEngine_EmptySelector(t *testing.T) {
	e := New()
	_, err := e.Extract([]byte(testHTML), []ExtractionRule{
		{Name: "bad", Type: TypeCSS, Selector: ""},
	})
	if err != ErrEmptySelector {
		t.Errorf("error = %v, want ErrEmptySelector", err)
	}
}

func TestEngine_InvalidType(t *testing.T) {
	e := New()
	_, err := e.Extract([]byte(testHTML), []ExtractionRule{
		{Name: "bad", Type: ExtractionType(99), Selector: "h1"},
	})
	if err != ErrInvalidType {
		t.Errorf("error = %v, want ErrInvalidType", err)
	}
}

func TestEngine_EmptyRules(t *testing.T) {
	e := New()
	results, err := e.Extract([]byte(testHTML), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}
