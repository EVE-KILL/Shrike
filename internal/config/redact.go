package config

import (
	"net/url"
	"strings"
)

// RedactURL strips credentials from a connection string so it can be printed.
// On a parse failure it returns a fixed placeholder rather than the original:
// an unparseable DSN is exactly the case where a password is most likely to be
// sitting in a weird position, and leaking it into a terminal someone screen-
// shares is worse than an unhelpful log line.
func RedactURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "<unparseable>"
	}
	if u.User != nil {
		if _, hasPassword := u.User.Password(); hasPassword {
			// The mask must be URL-safe: url.String() percent-encodes reserved
			// characters in userinfo, so a "****" mask would render as the
			// unreadable "%2A%2A%2A%2A".
			u.User = url.UserPassword(u.User.Username(), "REDACTED")
		}
	}
	return u.String()
}

// RedactSecret masks a bare secret, keeping only enough to confirm which value
// is loaded. Short secrets are masked wholesale — revealing a prefix of a
// 6-character password gives away most of it.
func RedactSecret(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 8 {
		return strings.Repeat("*", len(s))
	}
	return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
}
