package frontend

import (
	"strconv"
	"time"
)

type settingsESIData struct {
	Token struct {
		Scopes []string   `json:"scopes"`
		Expiry *time.Time `json:"token_expiry"`
	}
	Metrics struct {
		RateLimit struct {
			Count *float64 `json:"request_count"`
		} `json:"rateLimit"`
		ResponseTime struct {
			Average *float64 `json:"avg_ms"`
			P95     *float64 `json:"p95_ms"`
		} `json:"responseTime"`
		Volume []map[string]any `json:"volumeByHour"`
	}
	Logs                               map[string]any
	TokenError, MetricsError, LogError bool
}

func esiMetric(number *float64) string {
	if number == nil {
		return "—"
	}
	return strconv.FormatFloat(*number, 'f', -1, 64)
}
func esiCount(number *float64) string {
	if number == nil {
		return "—"
	}
	return fullNumber(*number)
}
func esiTokenStatus(expiry *time.Time, now time.Time) (string, string) {
	if expiry == nil {
		return "None", "text-red-400"
	}
	if expiry.After(now) {
		return "Active", "text-green-400"
	}
	return "Refreshing", "text-yellow-400"
}
