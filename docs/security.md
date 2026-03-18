# Security Guide

This document covers security best practices for deploying and operating the
web crawler in production, including credential management, TLS configuration,
data protection, and GDPR compliance guidance.

## TLS Configuration

The HTTP client enforces **TLS 1.2 minimum** by default (`pkg/client/transport.go`).
No additional configuration is required for standard deployments.

For high-security environments, consider:
- Using TLS 1.3 only by setting `MinVersion: tls.VersionTLS13` in a custom transport
- Enabling certificate pinning for known target domains
- Configuring explicit cipher suite allowlists

## Credential Management

### Environment Variables (Recommended)

Store all credentials in environment variables rather than configuration files:

```bash
# HTTP authentication
export CRAWLER_AUTH_TOKEN="Bearer your-token-here"
export CRAWLER_BASIC_USER="username"
export CRAWLER_BASIC_PASS="password"

# OAuth2 client credentials
export CRAWLER_OAUTH_CLIENT_ID="client-id"
export CRAWLER_OAUTH_CLIENT_SECRET="client-secret"
export CRAWLER_OAUTH_TOKEN_URL="https://auth.example.com/token"

# Proxy credentials
export CRAWLER_PROXY_USER="proxy-user"
export CRAWLER_PROXY_PASS="proxy-password"

# Storage credentials
export CRAWLER_PG_DSN="postgres://user:pass@host:5432/db?sslmode=require"
export CRAWLER_REDIS_URL="redis://:password@host:6379/0"
```

### Using the Credential Type

The `pkg/security.Credential` type prevents accidental credential exposure:

```go
import "github.com/kcenon/web_crawler/pkg/security"

// Load from environment — returns error if unset
token, err := security.CredentialFromEnv("CRAWLER_AUTH_TOKEN")
if err != nil {
    log.Fatal(err)
}

// Safe to log — always prints [REDACTED]
log.Printf("Using token: %s", token)  // Output: Using token: [REDACTED]

// Use the actual value only when needed
req.Header.Set("Authorization", token.Value())
```

### What NOT to Do

- Never hardcode credentials in source code
- Never commit `.env` files to version control
- Never log raw credential values
- Never embed credentials in URLs that might be logged

## Log Sanitization

Enable the log sanitizer in production to automatically redact sensitive data:

```go
import "github.com/kcenon/web_crawler/pkg/observability"

logger := observability.NewLogger(observability.LogConfig{
    Level:    slog.LevelInfo,
    Format:   "json",
    Sanitize: true,  // Enable credential redaction
})
```

The sanitizer automatically redacts:
- **Sensitive keys**: `password`, `token`, `api_key`, `secret`, `credential`,
  `private_key`, `access_key`, and variants
- **Bearer tokens**: `Bearer eyJhbG...` → `Bearer [REDACTED]`
- **Basic auth**: `Basic dXNlcjpw...` → `Basic [REDACTED]`
- **URL credentials**: `http://user:pass@host` → `http://[REDACTED]@host`

## Network Security

### Proxy Configuration

When using proxies, ensure credentials are not embedded in logged URLs:

```go
// Good: Use separate credential fields
proxyConfig := client.ProxyConfig{
    URL:      "http://proxy.example.com:8080",
    Username: proxyUser,    // From env var
    Password: proxyPass,    // From env var
}
```

### robots.txt Compliance

The crawler respects `robots.txt` directives by default via the robots middleware.
This includes:
- Disallow directives for specified paths
- Crawl-delay enforcement
- X-Robots-Tag header support (`noindex`)

Configure in middleware chain:

```go
chain := middleware.NewChain(
    middleware.RobotsMiddleware(middleware.RobotsConfig{
        UserAgent: "MyCrawler/1.0",
        CacheTTL:  24 * time.Hour,
    }),
)
```

## Dependency Security

### Automated Scanning

The project uses multiple layers of dependency security:

1. **GitHub Dependabot** — Automatically creates PRs for vulnerable dependencies
   (configured in `.github/dependabot.yml`)
2. **govulncheck** — Scans Go vulnerability database in CI
   (configured in `.github/workflows/security.yml`)
3. **gosec** — Static application security testing in CI

### Manual Scanning

Run vulnerability checks locally:

```bash
# Check for known vulnerabilities
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# Run security linter
golangci-lint run  # includes gosec
```

## GDPR Compliance Checklist

When crawling websites that may contain personal data (EU/EEA scope), use this
checklist to ensure compliance with the General Data Protection Regulation.

### Before Crawling

- [ ] **Legal basis identified** — Determine the legal basis for processing
  (legitimate interest, consent, etc.) per GDPR Article 6
- [ ] **Purpose limitation** — Document the specific purpose for data collection
- [ ] **Data minimization** — Configure extractors to collect only necessary fields;
  avoid collecting personal data unless required
- [ ] **robots.txt respected** — Verify the target site's robots.txt allows crawling
- [ ] **Terms of Service reviewed** — Check the target site's ToS for crawling
  restrictions
- [ ] **Rate limiting configured** — Avoid excessive load on target servers

### During Operation

- [ ] **Log sanitization enabled** — Use `Sanitize: true` to prevent personal
  data appearing in logs
- [ ] **Access controls** — Restrict access to crawled data storage
- [ ] **Encryption in transit** — Use TLS for all connections (enforced by default)
- [ ] **Encryption at rest** — Enable storage-level encryption for databases
  containing personal data

### Data Retention

- [ ] **Retention period defined** — Set maximum storage duration for crawled data
- [ ] **Deletion process** — Implement automated data purging after retention period
- [ ] **Data subject requests** — Have a process for handling access, rectification,
  and deletion requests (GDPR Articles 15-17)

### Documentation

- [ ] **Processing records** — Maintain records of processing activities
  (GDPR Article 30)
- [ ] **Data Protection Impact Assessment** — Complete DPIA for large-scale crawling
  operations (GDPR Article 35)
- [ ] **Privacy notice** — If crawling identifies individuals, ensure appropriate
  privacy notices are in place

### Technical Measures

| Measure | Implementation |
|---------|---------------|
| TLS minimum version | TLS 1.2 (default) |
| Credential protection | `pkg/security.Credential` type |
| Log redaction | `SanitizingHandler` in observability |
| robots.txt compliance | Robots middleware |
| Rate limiting | Rate limit middleware |
| User-Agent identification | User-Agent middleware |

## Incident Response

If a security issue is discovered:

1. **Assess scope** — Determine what data was affected
2. **Contain** — Stop the affected crawl jobs immediately
3. **Investigate** — Review logs (sanitized) to understand the issue
4. **Remediate** — Fix the vulnerability and deploy the fix
5. **Notify** — If personal data was breached, notify authorities within 72 hours
   per GDPR Article 33

## Reporting Security Issues

Report security vulnerabilities privately via GitHub Security Advisories or
by contacting the maintainers directly. Do not open public issues for
security vulnerabilities.
