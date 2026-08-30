package api

// The legacy handlers keep their permissive json.RawMessage decode targets so
// they preserve the JavaScript coercions the TypeScript services accepted.
// These types are documentation-only: they describe the actual wire shapes
// without changing the runtime parsers.

type killmailSearchDocument struct {
	From  string `json:"from" doc:"Window start as YYYY-MM-DD or an ISO 8601 timestamp."`
	To    string `json:"to" doc:"Window end as YYYY-MM-DD or an ISO 8601 timestamp."`
	Limit jsInt  `json:"limit,omitempty" minimum:"1" maximum:"100" doc:"Maximum killmails to return."`
	After jsInt  `json:"after,omitempty" minimum:"0" doc:"Return killmails after this identifier."`

	SystemIDs        requestList[jsInt] `json:"system_ids,omitempty" maxItems:"15" doc:"Restrict to these solar systems."`
	ConstellationIDs requestList[jsInt] `json:"constellation_ids,omitempty" maxItems:"15" doc:"Restrict to these constellations."`
	RegionIDs        requestList[jsInt] `json:"region_ids,omitempty" maxItems:"15" doc:"Restrict to these regions."`
	CharacterIDs     requestList[jsInt] `json:"character_ids,omitempty" maxItems:"15" doc:"Restrict to killmails involving these characters."`
	CorporationIDs   requestList[jsInt] `json:"corporation_ids,omitempty" maxItems:"15" doc:"Restrict to killmails involving these corporations."`
	AllianceIDs      requestList[jsInt] `json:"alliance_ids,omitempty" maxItems:"15" doc:"Restrict to killmails involving these alliances."`
}

type campaignEntityDocument struct {
	Type string `json:"type" enum:"character,corporation,alliance"`
	ID   jsInt  `json:"id" minimum:"1"`
	Name string `json:"name,omitempty" maxLength:"100"`
}

type campaignSideDocument struct {
	Name     string                              `json:"name,omitempty" maxLength:"50"`
	Entities requestList[campaignEntityDocument] `json:"entities" minItems:"1" maxItems:"15"`
}

type campaignLocationDocument struct {
	SystemIDs        requestList[jsInt] `json:"systemIds,omitempty" maxItems:"10"`
	ConstellationIDs requestList[jsInt] `json:"constellationIds,omitempty" maxItems:"5"`
	RegionIDs        requestList[jsInt] `json:"regionIds,omitempty" maxItems:"5"`
}

type campaignPrizePoolDocument struct {
	Enabled             bool               `json:"enabled"`
	Metric              jsInt              `json:"metric,omitempty" minimum:"0" maximum:"3"`
	WinnerCount         jsInt              `json:"winnerCount,omitempty" minimum:"3" maximum:"10"`
	PayoutPercentages   requestList[jsInt] `json:"payoutPercentages,omitempty" minItems:"3" maxItems:"10"`
	InitialContribution jsFloat            `json:"initialContribution,omitempty" minimum:"0"`
	FundingRequestID    string             `json:"fundingRequestId,omitempty" format:"uuid"`
}

type campaignCreateDocument struct {
	Name            string                              `json:"name" minLength:"3" maxLength:"100"`
	Description     optional[string]                    `json:"description,omitempty" maxLength:"2000"`
	StartTime       string                              `json:"startTime" format:"date-time"`
	EndTime         optional[string]                    `json:"endTime,omitempty" format:"date-time"`
	Visibility      jsInt                               `json:"visibility,omitempty" minimum:"0" maximum:"2" default:"0"`
	Location        campaignLocationDocument            `json:"location,omitempty"`
	Sides           requestList[campaignSideDocument]   `json:"sides,omitempty" maxItems:"4"`
	AllowedEntities requestList[campaignEntityDocument] `json:"allowedEntities,omitempty" maxItems:"10"`
	PrizePool       campaignPrizePoolDocument           `json:"prizePool,omitempty"`
}

type campaignUpdateDocument struct {
	Name             string                              `json:"name,omitempty" minLength:"3" maxLength:"100"`
	Description      optional[string]                    `json:"description,omitempty" maxLength:"2000"`
	EndTime          optional[string]                    `json:"endTime,omitempty" format:"date-time"`
	Visibility       jsInt                               `json:"visibility,omitempty" minimum:"0" maximum:"2"`
	AllowedEntities  requestList[campaignEntityDocument] `json:"allowedEntities,omitempty" maxItems:"10"`
	Sides            requestList[campaignSideDocument]   `json:"sides,omitempty" maxItems:"4"`
	Archived         bool                                `json:"archived,omitempty"`
	ResumeProcessing bool                                `json:"resumeProcessing,omitempty"`
}

type campaignContributeDocument struct {
	RequestID string  `json:"requestId" format:"uuid"`
	Amount    jsFloat `json:"amount" minimum:"0"`
}

type campaignActionDocument struct {
	Action string `json:"action" enum:"pause,resume,reprocess,archive,delete"`
	Reason string `json:"reason,omitempty" maxLength:"500"`
}

type announcementCreateDocument struct {
	Title     string           `json:"title"`
	BodyMD    string           `json:"body_md,omitempty"`
	Tier      int              `json:"tier" enum:"1,2,3"`
	Color     string           `json:"color,omitempty" enum:"info,warning,danger,success" default:"info"`
	Icon      optional[string] `json:"icon,omitempty" maxLength:"512"`
	LinkURL   optional[string] `json:"link_url,omitempty" maxLength:"4096"`
	LinkLabel optional[string] `json:"link_label,omitempty" maxLength:"512"`
	StartsAt  string           `json:"starts_at,omitempty" format:"date-time"`
	ExpiresAt string           `json:"expires_at" format:"date-time"`
}

type announcementUpdateDocument struct {
	Title     string           `json:"title,omitempty"`
	BodyMD    string           `json:"body_md,omitempty"`
	Tier      int              `json:"tier,omitempty" enum:"1,2,3"`
	Color     string           `json:"color,omitempty" enum:"info,warning,danger,success"`
	Icon      optional[string] `json:"icon,omitempty" maxLength:"512"`
	LinkURL   optional[string] `json:"link_url,omitempty" maxLength:"4096"`
	LinkLabel optional[string] `json:"link_label,omitempty" maxLength:"512"`
	StartsAt  string           `json:"starts_at,omitempty" format:"date-time"`
	ExpiresAt string           `json:"expires_at,omitempty" format:"date-time"`
}

type blogCreateDocument struct {
	Title         string              `json:"title"`
	Slug          string              `json:"slug,omitempty"`
	BodyMD        string              `json:"body_md,omitempty"`
	Excerpt       optional[string]    `json:"excerpt,omitempty" maxLength:"500"`
	CoverImageURL optional[string]    `json:"cover_image_url,omitempty" maxLength:"4096"`
	Status        int                 `json:"status,omitempty" enum:"0,1,2" default:"0"`
	PublishedAt   optional[string]    `json:"published_at,omitempty" format:"date-time"`
	Tags          requestList[string] `json:"tags,omitempty" maxItems:"10"`
}

type blogUpdateDocument struct {
	Title         string              `json:"title,omitempty"`
	Slug          string              `json:"slug,omitempty"`
	BodyMD        string              `json:"body_md,omitempty"`
	Excerpt       optional[string]    `json:"excerpt,omitempty" maxLength:"500"`
	CoverImageURL optional[string]    `json:"cover_image_url,omitempty" maxLength:"4096"`
	Status        int                 `json:"status,omitempty" enum:"0,1,2"`
	PublishedAt   optional[string]    `json:"published_at,omitempty" format:"date-time"`
	Tags          requestList[string] `json:"tags,omitempty" maxItems:"10"`
}

type commentCreateDocument struct {
	BodyMD     string          `json:"body_md"`
	ParentID   optional[jsInt] `json:"parent_id,omitempty" minimum:"0"`
	TargetType int             `json:"target_type" minimum:"1" maximum:"10"`
	TargetID   jsInt           `json:"target_id" minimum:"0"`
	TargetSlug string          `json:"target_slug,omitempty" maxLength:"255"`
}

type commentBodyDocument struct {
	BodyMD string `json:"body_md"`
}

type commentReportDocument struct {
	Reason  string           `json:"reason" enum:"spam,harassment,nsfw,offtopic,other"`
	Message optional[string] `json:"message,omitempty" maxLength:"1000"`
}

type moderationActionDocument struct {
	Action string `json:"action" enum:"hide,restore,hidden,published"`
}

type moderationResolutionDocument struct {
	Resolution string `json:"resolution" enum:"dismissed,deleted,warned"`
}

type moderationDecisionDocument struct {
	Decision string           `json:"decision" enum:"approve,reject"`
	Notes    optional[string] `json:"notes,omitempty" maxLength:"1000"`
}

type moderationNotesDocument struct {
	Notes optional[string] `json:"notes,omitempty" maxLength:"1000"`
}

type accountBoardsDocument struct {
	Pinned    requestList[string] `json:"pinned" maxItems:"12"`
	Dismissed requestList[string] `json:"dismissed" maxItems:"12"`
}

type accountPreferencesDocument struct {
	Theme       requestMap[string]    `json:"theme,omitempty"`
	DefaultTabs requestMap[string]    `json:"defaultTabs,omitempty"`
	Boards      accountBoardsDocument `json:"boards,omitempty"`
}

type accountThemeDocument struct {
	Theme requestMap[string] `json:"theme"`
}

type accountDefaultTabsDocument struct {
	DefaultTabs requestMap[string] `json:"defaultTabs"`
}

type accountNotificationReadDocument struct {
	ID jsInt `json:"id" minimum:"0"`
}

type adminSetDiscordDocument struct {
	DiscordUserID optional[string] `json:"discord_user_id,omitempty" pattern:"^[0-9]{15,22}$"`
}

type fittingItemDocument struct {
	SlotGroup    int64  `json:"slot_group" minimum:"1" maximum:"7"`
	Ordinal      int64  `json:"ordinal" minimum:"0" maximum:"15"`
	TypeID       int64  `json:"type_id" minimum:"1"`
	State        int64  `json:"state" minimum:"0" maximum:"3"`
	ChargeTypeID *int64 `json:"charge_type_id,omitempty" minimum:"1"`
	Quantity     *int64 `json:"quantity,omitempty" minimum:"1" maximum:"30000"`
}

type fittingCreateDocument struct {
	ShipTypeID  int64                            `json:"ship_type_id" minimum:"1"`
	Name        string                           `json:"name" minLength:"1" maxLength:"100"`
	Description optional[string]                 `json:"description,omitempty" maxLength:"2000"`
	Visibility  int64                            `json:"visibility" minimum:"0" maximum:"3"`
	Items       requestList[fittingItemDocument] `json:"items" maxItems:"200"`
}

type fittingUpdateDocument struct {
	Name        string                           `json:"name,omitempty" minLength:"1" maxLength:"100"`
	Description optional[string]                 `json:"description,omitempty" maxLength:"2000"`
	Visibility  int64                            `json:"visibility,omitempty" minimum:"0" maximum:"3"`
	Items       requestList[fittingItemDocument] `json:"items,omitempty" maxItems:"200"`
}

type fittingRatingDocument struct {
	Rating int64 `json:"rating" minimum:"1" maximum:"5"`
}

type domainCreateDocument struct {
	Subdomain       string                            `json:"subdomain" minLength:"3" maxLength:"32" pattern:"^[a-z0-9]([a-z0-9-]*[a-z0-9])?$"`
	Entities        requestList[SiteDomainEntity]     `json:"entities" minItems:"1" maxItems:"5"`
	Theme           SiteDomainTheme                   `json:"theme,omitempty"`
	NavbarLinks     requestList[SiteDomainNavbarLink] `json:"navbar_links,omitempty" maxItems:"10"`
	Widgets         SiteDomainWidgets                 `json:"widgets,omitempty"`
	SiteName        optional[string]                  `json:"site_name,omitempty" maxLength:"100"`
	SiteDescription optional[string]                  `json:"site_description,omitempty" maxLength:"500"`
}

type domainUpdateDocument struct {
	Entities          requestList[SiteDomainEntity]     `json:"entities,omitempty" minItems:"1" maxItems:"5"`
	Theme             SiteDomainTheme                   `json:"theme,omitempty"`
	NavbarLinks       requestList[SiteDomainNavbarLink] `json:"navbar_links,omitempty" maxItems:"10"`
	Widgets           SiteDomainWidgets                 `json:"widgets,omitempty"`
	SiteName          optional[string]                  `json:"site_name,omitempty" maxLength:"100"`
	SiteDescription   optional[string]                  `json:"site_description,omitempty" maxLength:"500"`
	Active            bool                              `json:"active,omitempty"`
	CampaignPolicy    int                               `json:"campaign_policy,omitempty" enum:"0,1"`
	CampaignIDs       requestList[string]               `json:"campaign_ids,omitempty" maxItems:"20"`
	CampaignPublicIDs requestList[string]               `json:"campaign_public_ids,omitempty" maxItems:"20"`
}
