package extractor

import (
	"github.com/tidwall/gjson"
)

// extractJSON extracts values from JSON content using a gjson path expression.
// For array results, each element is returned as a separate value.
func extractJSON(content []byte, path string) ([]string, error) {
	result := gjson.GetBytes(content, path)

	if !result.Exists() {
		return nil, nil
	}

	if result.IsArray() {
		arr := result.Array()
		values := make([]string, 0, len(arr))
		for _, item := range arr {
			values = append(values, item.String())
		}
		return values, nil
	}

	return []string{result.String()}, nil
}
