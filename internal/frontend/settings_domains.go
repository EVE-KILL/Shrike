package frontend

type settingsDomainEntity struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}
type settingsDomain struct {
	ID             int64                  `json:"id"`
	Subdomain      string                 `json:"subdomain"`
	Name           string                 `json:"site_name"`
	Description    string                 `json:"site_description"`
	Active         bool                   `json:"active"`
	Entities       []settingsDomainEntity `json:"entities"`
	Theme          map[string]any         `json:"theme"`
	Navbar         []map[string]any       `json:"navbar_links"`
	Widgets        map[string]any         `json:"widgets"`
	CampaignPolicy int                    `json:"campaign_policy"`
	Campaigns      []map[string]any       `json:"campaigns"`
	Backgrounds    []map[string]any       `json:"backgrounds"`
	BannerAsset    map[string]any         `json:"bannerAsset"`
	LogoAsset      map[string]any         `json:"logoAsset"`
}
type settingsDomainsData struct {
	Domains []settingsDomain `json:"domains"`
}

func settingsEntityImage(entity settingsDomainEntity) string {
	return eveEntityImage(entity.Type, entity.ID, 32)
}
func domainDefault(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
