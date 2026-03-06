package extractor

import (
	"fmt"
	"regexp"
)

// extractRegex extracts values matching the regular expression pattern.
// If the pattern contains named capture groups, only the first named group
// value is returned per match. Otherwise, the full match (group 0) is returned.
func extractRegex(content []byte, pattern string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("extractor: compile regex %q: %w", pattern, err)
	}

	matches := re.FindAllSubmatch(content, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	// Check for named capture groups.
	names := re.SubexpNames()
	namedIdx := -1
	for i, name := range names {
		if i > 0 && name != "" {
			namedIdx = i
			break
		}
	}

	values := make([]string, 0, len(matches))
	for _, m := range matches {
		if namedIdx >= 0 && namedIdx < len(m) {
			values = append(values, string(m[namedIdx]))
		} else {
			values = append(values, string(m[0]))
		}
	}

	return values, nil
}
