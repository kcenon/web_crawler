package extractor

import "errors"

// ExtractionType identifies the extraction strategy.
type ExtractionType int

const (
	TypeCSS   ExtractionType = iota // CSS selector (GoQuery)
	TypeXPath                       // XPath expression
	TypeJSON                        // JSON path (gjson)
	TypeRegex                       // Regular expression
)

// ExtractionRule defines a single extraction rule.
type ExtractionRule struct {
	Name      string         // Identifier for the extracted field.
	Type      ExtractionType // Strategy to use.
	Selector  string         // CSS selector, XPath, JSON path, or regex pattern.
	Attribute string         // For CSS: "text", "html", or attribute name (e.g. "href").
}

// ExtractionResult holds the result of applying an extraction rule.
type ExtractionResult struct {
	Name   string   // From ExtractionRule.Name.
	Values []string // Extracted values; one per matched element.
}

// Extractor applies extraction rules to content.
type Extractor interface {
	Extract(content []byte, rules []ExtractionRule) ([]ExtractionResult, error)
}

var (
	ErrEmptySelector = errors.New("extractor: empty selector")
	ErrInvalidType   = errors.New("extractor: invalid extraction type")
)

// Engine is the default Extractor implementation that dispatches
// to the appropriate strategy based on ExtractionType.
type Engine struct{}

// New creates a new extraction Engine.
func New() *Engine {
	return &Engine{}
}

// Extract applies each rule to the content and collects results.
func (e *Engine) Extract(content []byte, rules []ExtractionRule) ([]ExtractionResult, error) {
	results := make([]ExtractionResult, 0, len(rules))

	for _, rule := range rules {
		if rule.Selector == "" {
			return nil, ErrEmptySelector
		}

		var values []string
		var err error

		switch rule.Type {
		case TypeCSS:
			values, err = extractCSS(content, rule.Selector, rule.Attribute)
		case TypeXPath:
			values, err = extractXPath(content, rule.Selector)
		case TypeJSON:
			values, err = extractJSON(content, rule.Selector)
		case TypeRegex:
			values, err = extractRegex(content, rule.Selector)
		default:
			return nil, ErrInvalidType
		}

		if err != nil {
			return nil, err
		}

		results = append(results, ExtractionResult{
			Name:   rule.Name,
			Values: values,
		})
	}

	return results, nil
}
