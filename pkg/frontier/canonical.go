package frontier

import (
	"net/url"
	"sort"
	"strings"
)

// Canonicalize normalizes a URL following RFC 3986 conventions:
//   - Lowercases the scheme and host
//   - Removes default ports (80 for http, 443 for https)
//   - Removes fragment
//   - Sorts query parameters
//   - Removes trailing slash on path (except root "/")
//   - Decodes unnecessary percent-encoding
func Canonicalize(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	// Lowercase scheme and host.
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)

	// Remove default ports.
	host := u.Hostname()
	port := u.Port()
	if (u.Scheme == "http" && port == "80") || (u.Scheme == "https" && port == "443") {
		u.Host = host
	}

	// Remove fragment.
	u.Fragment = ""

	// Sort query parameters.
	if u.RawQuery != "" {
		params := u.Query()
		keys := make([]string, 0, len(params))
		for k := range params {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var buf strings.Builder
		for i, k := range keys {
			values := params[k]
			sort.Strings(values)
			for j, v := range values {
				if i > 0 || j > 0 {
					buf.WriteByte('&')
				}
				buf.WriteString(url.QueryEscape(k))
				buf.WriteByte('=')
				buf.WriteString(url.QueryEscape(v))
			}
		}
		u.RawQuery = buf.String()
	}

	// Remove trailing slash (except root path).
	if len(u.Path) > 1 {
		u.Path = strings.TrimRight(u.Path, "/")
	}

	return u.String()
}
