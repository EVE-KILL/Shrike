package ingress

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/felixge/httpsnoop"
	"github.com/rs/zerolog"
)

func accessLog(log zerolog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		upgraded := false
		if isWebSocketUpgrade(r) {
			// CaptureMetrics skips informational statuses, including a successful
			// WebSocket handshake. Observe it while preserving Hijacker.
			w = httpsnoop.Wrap(w, httpsnoop.Hooks{
				WriteHeader: func(write httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
					return func(status int) {
						upgraded = upgraded || status == http.StatusSwitchingProtocols
						write(status)
					}
				},
			})
		}
		metrics := httpsnoop.CaptureMetrics(next, w, r)
		if upgraded {
			metrics.Code = http.StatusSwitchingProtocols
		}
		// Healthy Kubernetes probes are frequent enough to bury useful traffic.
		if r.URL.Path == "/health" && metrics.Code >= 200 && metrics.Code < 400 {
			return
		}
		line := nginxAccessLine(r, metrics.Code, metrics.Written, started, metrics.Duration)
		if metrics.Code >= http.StatusInternalServerError {
			log.Error().Msg(line)
			return
		}
		log.Info().Msg(line)
	})
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("Upgrade")), "websocket") &&
		headerContainsToken(r.Header.Values("Connection"), "upgrade")
}

func headerContainsToken(values []string, want string) bool {
	for _, value := range values {
		for token := range strings.SplitSeq(value, ",") {
			if strings.EqualFold(strings.TrimSpace(token), want) {
				return true
			}
		}
	}
	return false
}

func nginxAccessLine(
	r *http.Request,
	status int,
	size int64,
	started time.Time,
	elapsed time.Duration,
) string {
	request := r.Method + " " + safeRequestURI(r.URL) + " " + r.Proto
	return fmt.Sprintf(
		`%s - - [%s] %s %d %d %s %s %s`,
		clientIP(r),
		started.Format("02/Jan/2006:15:04:05 -0700"),
		strconv.Quote(request),
		status,
		size,
		strconv.Quote(nginxField(r.Referer())),
		strconv.Quote(nginxField(r.UserAgent())),
		strconv.FormatFloat(elapsed.Seconds(), 'f', 3, 64),
	)
}

func nginxField(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func clientIP(r *http.Request) string {
	if ip := validHeaderIP(r.Header.Get("CF-Connecting-IP")); ip != "" {
		return ip
	}
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		first, _, _ := strings.Cut(forwarded, ",")
		if ip := validHeaderIP(first); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}
	if ip := net.ParseIP(strings.TrimSpace(r.RemoteAddr)); ip != nil {
		return ip.String()
	}
	return "-"
}

func validHeaderIP(value string) string {
	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil {
		return ""
	}
	return ip.String()
}

func safeRequestURI(u *url.URL) string {
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	if u.RawQuery == "" {
		return path
	}

	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		// Do not risk logging an unparseable query that may contain a secret.
		return path
	}
	for key := range query {
		if sensitiveQueryKey(key) {
			query[key] = []string{"REDACTED"}
		}
	}
	return path + "?" + query.Encode()
}

func sensitiveQueryKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "access_token", "api_key", "authorization", "client_secret", "code",
		"key", "password", "refresh_token", "secret", "signature", "state",
		"token":
		return true
	default:
		return false
	}
}
