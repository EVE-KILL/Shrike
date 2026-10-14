package telemetry

import (
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/eve-kill/shrike/internal/config"
	"github.com/getsentry/sentry-go"
	"github.com/getsentry/sentry-go/attribute"
)

type redactor struct{ secrets []string }

var (
	credentialURL   = regexp.MustCompile(`(?:https?|postgres(?:ql)?|bolt|rediss?)://[^\s<>"']+`)
	bearerValue     = regexp.MustCompile(`(?i)(?:bearer|basic)\s+[^\s,"']+`)
	credentialValue = regexp.MustCompile(`(?i)(?:access_token|refresh_token|client_secret|password|api_key|authorization)\s*[=:]\s*[^\s,&"']+`)
	jwtValue        = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`)
)

func newRedactor(cfg *config.Config) *redactor {
	r := &redactor{}
	for _, value := range []string{cfg.RedisPassword, cfg.EVEClientSecret, cfg.OpenAIAPIKey, cfg.KlipyAPIKey, cfg.B2AppKey, cfg.SentryDSN} {
		if len(value) >= 4 {
			r.secrets = append(r.secrets, value)
		}
	}
	for _, raw := range []string{cfg.DatabaseURL, cfg.DatabaseReadURL, cfg.MemgraphURL} {
		if u, err := url.Parse(raw); err == nil && u.User != nil {
			if password, ok := u.User.Password(); ok && len(password) >= 4 {
				r.secrets = append(r.secrets, password)
			}
		}
	}
	return r
}

func (r *redactor) text(value string) string {
	for _, secret := range r.secrets {
		value = strings.ReplaceAll(value, secret, "[Filtered]")
	}
	value = credentialURL.ReplaceAllStringFunc(value, cleanURL)
	value = bearerValue.ReplaceAllString(value, "Bearer [Filtered]")
	value = credentialValue.ReplaceAllString(value, "credential=[Filtered]")
	return jwtValue.ReplaceAllString(value, "[Filtered]")
}

func cleanURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "[Filtered URL]"
	}
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

func sensitiveField(key string) bool {
	key = strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	compact := strings.ReplaceAll(key, "_", "")
	for _, term := range []string{"ownerhash", "killmailhash", "apikey"} {
		if strings.Contains(compact, term) {
			return true
		}
	}
	for _, term := range []string{"password", "secret", "token", "authorization", "cookie", "api_key", "owner_hash", "killmail_hash"} {
		if strings.Contains(key, term) {
			return true
		}
	}
	leaf := key
	if index := strings.LastIndexByte(key, '.'); index >= 0 {
		leaf = key[index+1:]
	}
	switch leaf {
	case "args", "encoded_args", "encodedargs", "metadata", "payload", "body", "request_body", "response_body", "input", "output", "headers", "query", "query_string", "query_params", "params", "ip", "client_ip", "email", "user":
		return true
	}
	return strings.HasPrefix(key, "_sentry_")
}

func (r *redactor) data(value any) any {
	switch value := value.(type) {
	case string:
		return r.text(value)
	case map[string]any:
		for key, item := range value {
			if sensitiveField(key) {
				delete(value, key)
			} else {
				value[key] = r.data(item)
			}
		}
	case []any:
		for i, item := range value {
			value[i] = r.data(item)
		}
	}
	return value
}

func (r *redactor) attributes(values map[string]attribute.Value) {
	for key, value := range values {
		if sensitiveField(key) {
			delete(values, key)
			continue
		}
		if text, ok := value.AsInterface().(string); ok {
			values[key] = attribute.StringValue(r.text(text))
		}
	}
}

func (r *redactor) breadcrumb(b *sentry.Breadcrumb, _ *sentry.BreadcrumbHint) *sentry.Breadcrumb {
	b.Message = r.text(b.Message)
	r.data(b.Data)
	return b
}

func (r *redactor) event(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
	event.Message = r.text(event.Message)
	event.User = sentry.User{}
	if req := event.Request; req != nil {
		req.URL = cleanURL(req.URL)
		if _, pattern, ok := strings.Cut(event.Transaction, " "); ok && strings.HasPrefix(pattern, "/") {
			if requestURL, err := url.Parse(req.URL); err == nil {
				requestURL.Path, requestURL.RawPath = pattern, ""
				req.URL = requestURL.String()
			}
		}
		req.Headers = nil
		req.Cookies = ""
		req.Data = ""
		req.QueryString = ""
		req.Env = nil
	}
	for key, value := range event.Tags {
		if sensitiveField(key) {
			delete(event.Tags, key)
		} else {
			event.Tags[key] = r.text(value)
		}
	}
	for key, value := range event.Contexts {
		r.data(value)
		event.Contexts[key] = value
	}
	for _, exception := range event.Exception {
		scrubStack(exception.Stacktrace, r)
	}
	for i := range event.Exception {
		event.Exception[i].Value = r.text(event.Exception[i].Value)
	}
	for _, thread := range event.Threads {
		scrubStack(thread.Stacktrace, r)
	}
	for _, b := range event.Breadcrumbs {
		r.breadcrumb(b, nil)
	}
	for _, span := range event.Spans {
		span.Description = r.text(span.Description)
		r.data(span.Data)
	}
	return event
}

func scrubStack(stack *sentry.Stacktrace, r *redactor) {
	if stack == nil {
		return
	}
	for i := range stack.Frames {
		stack.Frames[i].Vars = nil
		if strings.HasPrefix(stack.Frames[i].Module, "github.com/eve-kill/shrike/internal/telemetry") || strings.HasPrefix(stack.Frames[i].Module, "github.com/eve-kill/shrike/internal/logging") {
			stack.Frames[i].InApp = false
		}
	}
}

// NormalizeSQL preserves the query shape, never bind arguments or literals.
// This also handles SQL/Cypher comments and PostgreSQL dollar-quoted bodies.
func NormalizeSQL(query string) string {
	var out strings.Builder
	for i := 0; i < len(query) && out.Len() < 4096; {
		switch {
		case strings.HasPrefix(query[i:], "--"):
			end := strings.IndexByte(query[i:], '\n')
			if end < 0 {
				i = len(query)
			} else {
				i += end + 1
			}
			out.WriteByte(' ')
		case strings.HasPrefix(query[i:], "/*"):
			i += 2
			depth := 1
			for i < len(query) && depth > 0 {
				if strings.HasPrefix(query[i:], "/*") {
					depth++
					i += 2
				} else if strings.HasPrefix(query[i:], "*/") {
					depth--
					i += 2
				} else {
					i++
				}
			}
			out.WriteByte(' ')
		case query[i] == '\'':
			out.WriteByte('?')
			i++
			for i < len(query) {
				if query[i] == '\\' && i+1 < len(query) {
					i += 2
					continue
				}
				if query[i] == '\'' {
					i++
					if i < len(query) && query[i] == '\'' {
						i++
						continue
					}
					break
				}
				i++
			}
		case query[i] == '$':
			end := i + 1
			for end < len(query) && (query[end] == '_' || query[end] >= 'a' && query[end] <= 'z' || query[end] >= 'A' && query[end] <= 'Z' || end > i+1 && query[end] >= '0' && query[end] <= '9') {
				end++
			}
			if end < len(query) && query[end] == '$' {
				tag := query[i : end+1]
				out.WriteByte('?')
				i = end + 1
				if close := strings.Index(query[i:], tag); close >= 0 {
					i += close + len(tag)
				} else {
					i = len(query)
				}
			} else {
				out.WriteByte(query[i])
				i++
				for i < len(query) && query[i] >= '0' && query[i] <= '9' {
					out.WriteByte(query[i])
					i++
				}
			}
		case query[i] >= '0' && query[i] <= '9' && (i == 0 || !identifierByte(query[i-1])):
			out.WriteByte('?')
			i++
			for i < len(query) && (identifierByte(query[i]) || query[i] == '.') {
				i++
			}
		default:
			out.WriteByte(query[i])
			i++
		}
	}
	return strings.Join(strings.Fields(out.String()), " ")
}

func identifierByte(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || ch >= '0' && ch <= '9' || ch == '_'
}
