package frontend

import (
	"fmt"
	"net/url"
	"strconv"
)

type adminCampaign struct {
	ID          string  `json:"campaign_id"`
	Name        string  `json:"name"`
	Status      int     `json:"status"`
	Visibility  int     `json:"visibility"`
	CreatorID   int64   `json:"created_by_character_id"`
	CreatorName *string `json:"creator_name"`
	Paused      bool    `json:"processing_paused"`
	Note        *string `json:"processing_note"`
	Error       *string `json:"last_processing_error"`
	Estimated   *int64  `json:"estimated_killmails"`
	Processed   *int64  `json:"last_processing_killmails"`
	Duration    *int64  `json:"last_processing_duration_ms"`
	Totals      *struct {
		KillCount int64 `json:"killCount"`
	} `json:"totals"`
}
type adminCampaignQuery struct {
	State, Search string
	Page          int
}
type adminCampaignsData struct {
	Rows    []adminCampaign    `json:"campaigns"`
	Page    int                `json:"page"`
	HasMore bool               `json:"hasMore"`
	Query   adminCampaignQuery `json:"-"`
	Failed  bool               `json:"-"`
}

func adminCampaignSelection(query url.Values) adminCampaignQuery {
	result := adminCampaignQuery{State: "all", Search: query.Get("q"), Page: 1}
	switch value := query.Get("state"); value {
	case "paused", "failed", "pending", "active", "archived":
		result.State = value
	}
	if page, err := strconv.ParseInt(query.Get("page"), 10, 32); err == nil && page > 0 {
		result.Page = int(page)
		if result.Page > 10000 {
			result.Page = 10000
		}
	}
	return result
}
func (query adminCampaignQuery) values() url.Values {
	return url.Values{"state": {query.State}, "q": {query.Search}, "page": {strconv.Itoa(query.Page)}}
}
func (campaign adminCampaign) status() (string, string) {
	if campaign.Paused {
		return "Paused", "text-red-400 bg-red-500/10"
	}
	switch campaign.Status {
	case 0:
		return "Pending", "text-yellow-400 bg-yellow-500/10"
	case 2:
		return "Archived", "text-gray-500 bg-white/[0.04]"
	default:
		return "Active", "text-green-400 bg-green-500/10"
	}
}
func (campaign adminCampaign) processed() int64 {
	if campaign.Processed != nil {
		return *campaign.Processed
	}
	if campaign.Totals != nil {
		return campaign.Totals.KillCount
	}
	return 0
}
func (campaign adminCampaign) creator() string {
	if campaign.CreatorName != nil && *campaign.CreatorName != "" {
		return *campaign.CreatorName
	}
	return strconv.FormatInt(campaign.CreatorID, 10)
}
func (campaign adminCampaign) visibility() string {
	if campaign.Visibility == 0 {
		return "Public"
	}
	if campaign.Visibility == 1 {
		return "Private"
	}
	return "Killboard"
}
func adminCampaignDuration(milliseconds *int64) string {
	if milliseconds == nil || *milliseconds == 0 {
		return "—"
	}
	if *milliseconds < 1000 {
		return strconv.FormatInt(*milliseconds, 10) + "ms"
	}
	if *milliseconds < 60000 {
		return fmt.Sprintf("%.1fs", float64(*milliseconds)/1000)
	}
	return fmt.Sprintf("%.1fm", float64(*milliseconds)/60000)
}
func adminCampaignCount(value *int64) string {
	if value == nil {
		return "0"
	}
	return killNumber(*value)
}
