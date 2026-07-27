package ingress

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/rs/zerolog"
)

// accessLogHandler emits one nginx-style access line for every completed
// request. It wraps the origin handler inside each route, so it covers the Go
// surfaces, the Nuxt reverse proxy, WebSocket upgrades, and the fallback 404.
type accessLogHandler struct {
	log zerolog.Logger
}

func (accessLogHandler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.shrike_access_log",
		New: func() caddy.Module { return new(accessLogHandler) },
	}
}

func (h *accessLogHandler) Provision(caddy.Context) error {
	m := activeManager.Load()
	if m == nil {
		return errors.New("ingress: no active manager; the access logger cannot be used outside a running Manager")
	}
	h.log = m.log
	return nil
}

func (h *accessLogHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
	next caddyhttp.Handler,
) error {
	started := time.Now()
	recorder := caddyhttp.NewResponseRecorder(w, nil, nil)
	err := next.ServeHTTP(recorder, r)

	status := recorder.Status()
	if status == 0 {
		var handlerErr caddyhttp.HandlerError
		if errors.As(err, &handlerErr) && handlerErr.StatusCode != 0 {
			status = handlerErr.StatusCode
		} else {
			status = http.StatusOK
		}
	}

	// Kubernetes probes run frequently enough to bury useful traffic. A
	// healthy probe stays quiet, while any unhealthy response is still logged.
	if r.URL.Path == "/health" && err == nil && status >= 200 && status < 400 {
		return nil
	}

	line := nginxAccessLine(r, status, recorder.Size(), started, time.Since(started))
	if err != nil || status >= http.StatusInternalServerError {
		event := h.log.Error()
		if err != nil {
			event.Err(err)
		}
		event.Msg(line)
	} else {
		h.log.Info().Msg(line)
	}
	return err
}

func nginxAccessLine(
	r *http.Request,
	status int,
	size int,
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

var (
	_ caddy.Provisioner           = (*accessLogHandler)(nil)
	_ caddyhttp.MiddlewareHandler = (*accessLogHandler)(nil)
)
