package api

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

const loginHost = "eve-kill.com"

// Only registered, active first-party boards may be return destinations. A
// separate host parameter keeps normalizeReturnTo's strict path validation
// intact and never accepts an arbitrary absolute redirect URL.
func (s *authService) validateLoginReturnHost(ctx context.Context, host string) error {
	if host == loginHost || host == "www."+loginHost {
		return nil
	}
	subdomain := strings.TrimSuffix(host, "."+loginHost)
	if subdomain == host || !domainSubdomainPattern.MatchString(subdomain) {
		return apiError(http.StatusBadRequest, "Invalid login return host")
	}
	if s.domainDB == nil {
		return apiError(http.StatusServiceUnavailable, "Domain storage is not configured")
	}
	var active bool
	err := s.domainDB.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM custom_domains WHERE active IS TRUE AND subdomain = $1
		)`, subdomain).Scan(&active)
	if err != nil {
		return err
	}
	if !active {
		return apiError(http.StatusBadRequest, "Unknown or inactive login return host")
	}
	return nil
}

func (s *authService) subsiteLoginDestination(
	ctx context.Context, req *legacyRequest, path string,
) (returnTo string, loginHop string, err error) {
	returnHost := req.Query.Get("returnHost")
	if !s.production {
		if returnHost != "" {
			return "", "", apiError(http.StatusBadRequest, "Invalid login return host")
		}
		return path, "", nil
	}
	host := strings.ToLower(strings.TrimSuffix(legacyRequestHost(req), ":443"))
	if host != loginHost && strings.HasSuffix(host, "."+loginHost) {
		// Derive the destination from the actual request host, never a forwarded
		// header or caller-supplied returnHost on the subsite.
		if err := s.validateLoginReturnHost(ctx, host); err != nil {
			return "", "", err
		}
		query := url.Values{"returnTo": {path}, "returnHost": {host}}
		for _, key := range []string{"charKm", "corpKm", "delay"} {
			if value := req.Query.Get(key); value != "" {
				query.Set(key, value)
			}
		}
		return "", "https://" + loginHost + "/auth/eve/start?" + query.Encode(), nil
	}
	if returnHost == "" {
		return path, "", nil
	}
	if host != loginHost {
		return "", "", apiError(http.StatusBadRequest, "Invalid login return host")
	}
	if err := s.validateLoginReturnHost(ctx, returnHost); err != nil {
		return "", "", err
	}
	return "https://" + returnHost + path, "", nil
}
