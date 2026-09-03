package api

import "github.com/danielgtaylor/huma/v2"

// applicationOperationResponseSchema documents the successful payloads that
// were historically implemented by Nuxt server routes. The compatibility
// handlers stay low-level so their validation and error shapes do not change;
// this catalogue is therefore the single source of truth for their OpenAPI
// output types.
func applicationOperationResponseSchema(operationID string) *huma.Schema {
	switch operationID {
	// Authentication and sessions.
	case "auth-login-legacy":
		return responseSchema(map[string]*huma.Schema{
			"url": stringSchema(),
		}, "url")
	case "me", "auth-me-legacy":
		return responseSchema(map[string]*huma.Schema{
			"user": nullable(accountUserSchema()),
		}, "user")
	case "me-settings":
		return accountSettingsSchema()
	case "auth-token-info-legacy":
		return responseSchema(map[string]*huma.Schema{
			"scopes":       arraySchema(stringSchema()),
			"token_expiry": nullable(timestampSchema()),
		}, "scopes", "token_expiry")
	case "session-delete", "auth-logout-legacy":
		return successResponseSchema()
	case "sessions", "user-sessions-legacy":
		return responseSchema(map[string]*huma.Schema{
			"sessions": arraySchema(accountSessionSchema()),
		}, "sessions")
	case "session-revoke", "user-session-revoke-legacy":
		return responseSchema(map[string]*huma.Schema{
			"revoked": boolSchema(),
			"current": boolSchema(),
		}, "revoked", "current")
	case "other-sessions-revoke", "other-sessions-revoke-legacy":
		return responseSchema(map[string]*huma.Schema{
			"revoked": intSchema(),
		}, "revoked")

	// Account preferences, boards, descriptions, ESI, and notifications.
	case "account-preferences", "user-preferences-compat":
		return accountPreferencesSchema()
	case "account-preferences-update", "user-preferences-update-compat":
		return responseSchema(map[string]*huma.Schema{
			"preferences": accountPreferencesSchema(),
		}, "preferences")
	case "user-theme-update-compat":
		return responseSchema(map[string]*huma.Schema{
			"theme": openJSONObjectSchema("User-selected theme settings."),
		}, "theme")
	case "user-boards-update-compat":
		return accountBoardStateSchema()
	case "account-boards", "boards-mine-compat":
		return accountBoardsSchema()
	case "account-overview", "user-overview-compat":
		return accountOverviewSchema()
	case "user-manageable-entities-compat":
		return manageableEntitiesSchema()
	case "account-descriptions":
		return manageableEntitiesSchema()
	case "account-description-update", "user-description-update-compat":
		return responseSchema(map[string]*huma.Schema{
			"ok":        boolSchema(),
			"status":    stringSchema(),
			"entity":    stringSchema(),
			"entity_id": intSchema(),
			"queue_id":  intSchema(),
		}, "ok", "status", "entity", "entity_id")
	case "account-esi", "user-esi-compat", "admin-esi-overview":
		return esiMetricsSchema()
	case "account-esi-logs", "user-esi-logs-compat", "admin-esi-logs":
		return esiLogsSchema()
	case "announcements-active-compat":
		return responseSchema(map[string]*huma.Schema{
			"announcements": arraySchema(announcementSchema()),
		}, "announcements")
	case "announcements-dismissed-compat":
		return responseSchema(map[string]*huma.Schema{
			"dismissedIds": arraySchema(intSchema()),
		}, "dismissedIds")
	case "announcement-dismiss-compat":
		return responseSchema(map[string]*huma.Schema{"ok": boolSchema()}, "ok")
	case "account-notification-replies", "notification-replies-compat":
		return notificationRepliesSchema()
	case "account-notification-read-cursor", "notification-mark-read-compat":
		return responseSchema(map[string]*huma.Schema{
			"lastSeenNotificationId": intSchema(),
		}, "lastSeenNotificationId")

	// Administration.
	case "admin-overview":
		return adminOverviewSchema()
	case "admin-users-list":
		return adminUsersListSchema()
	case "admin-users-detail":
		return adminUserDetailSchema()
	case "admin-users-set-discord":
		return responseSchema(map[string]*huma.Schema{
			"character_id":    intSchema(),
			"discord_user_id": nullable(stringSchema()),
		}, "character_id", "discord_user_id")
	case "admin-users-toggle-admin":
		return responseSchema(map[string]*huma.Schema{
			"character_id": intSchema(),
			"is_admin":     boolSchema(),
		}, "character_id", "is_admin")
	case "admin-esi-entities":
		return responseSchema(map[string]*huma.Schema{
			"results": arraySchema(responseSchema(map[string]*huma.Schema{
				"id": intSchema(), "name": stringSchema(), "type": stringSchema(),
			}, "id", "name", "type")),
		}, "results")
	case "admin-river-overview":
		return responseSchema(map[string]*huma.Schema{
			"queues": arraySchema(riverQueueSchema()),
		}, "queues")
	case "admin-river-jobs":
		return responseSchema(map[string]*huma.Schema{
			"jobs":           arraySchema(riverJobSchema()),
			"next_before_id": intSchema(),
		}, "jobs", "next_before_id")
	case "admin-river-job":
		return responseSchema(map[string]*huma.Schema{"job": riverJobSchema()}, "job")
	case "admin-river-job-action":
		return responseSchema(map[string]*huma.Schema{
			"job": riverJobSchema(), "action": stringSchema(),
		}, "job", "action")
	case "admin-river-queue-action":
		return responseSchema(map[string]*huma.Schema{
			"queue": stringSchema(), "action": stringSchema(),
		}, "queue", "action")
	case "admin-river-queue-clear":
		return responseSchema(map[string]*huma.Schema{
			"queue": stringSchema(), "deleted": intSchema(),
		}, "queue", "deleted")

	// Killboard and killmail compatibility endpoints.
	case "killlist-advanced",
		"universe-region-killmails", "region-killlist-compat",
		"universe-constellation-killmails", "constellation-killlist-compat",
		"universe-system-killmails", "system-killlist-compat",
		"universe-type-killmails", "item-killlist-compat",
		"ship-killlist-compat",
		"entity-page-killlist", "entity-page-killlist-generic-compat":
		return killlistFrontendResponseSchema()
	case "kills-top":
		return entriesResponseSchema(topKillEntrySchema())
	case "kills-most-valuable":
		return entriesResponseSchema(mostValuableKillSchema())
	case "graph":
		return graphResponseSchema()
	case "killmail-detail-legacy":
		return killmailDetailSchema()
	case "killmail-exists", "killmail-exists-legacy":
		return responseSchema(map[string]*huma.Schema{
			"exists": boolSchema(),
		}, "exists")
	case "killmail-editor-fit", "killmail-editor-fit-legacy":
		return killmailEditorFitSchema()
	case "killmail-siblings", "killmail-siblings-legacy":
		return responseSchema(map[string]*huma.Schema{
			"siblings": arraySchema(killmailSiblingSchema()),
		}, "siblings")
	case "killmail-submit":
		return killmailSubmissionResponseSchema()
	case "coalitions":
		return coalitionDirectoryListResponseSchema()
	case "coalition", "coalition-create", "coalition-update":
		return coalitionDirectoryDetailResponseSchema()

	// Universe page aggregates.
	case "universe-region", "region-compat":
		return universeRegionResponseSchema()
	case "universe-constellation", "constellation-compat":
		return universeConstellationResponseSchema()
	case "universe-system", "system-compat":
		return universeSystemResponseSchema()
	case "universe-type", "type-compat":
		return universeTypeResponseSchema()
	case "universe-group", "group-compat":
		return universeGroupResponseSchema()
	case "universe-region-most-valuable", "region-most-valuable-compat",
		"universe-constellation-most-valuable", "constellation-most-valuable-compat",
		"universe-system-most-valuable", "system-most-valuable-compat":
		return entriesResponseSchema(mostValuableKillSchema())
	case "ship-matchup":
		return shipMatchupResponseSchema()

	// Consolidated entity pages and their singular aliases.
	case "entity-resolve", "entity-resolve-compat":
		return responseSchema(map[string]*huma.Schema{
			"type": stringSchema(), "id": intSchema(), "name": stringSchema(),
		}, "type", "id", "name")
	case "entity-page-detail", "entity-page-detail-character-compat",
		"entity-page-detail-corporation-compat",
		"entity-page-detail-alliance-compat",
		"entity-page-detail-faction-compat":
		return entityPageDetailResponseSchema()
	case "entity-page-stats", "entity-page-stats-character-compat",
		"entity-page-stats-corporation-compat",
		"entity-page-stats-alliance-compat":
		return entityDashboardStatsSchema()
	case "entity-page-intel", "entity-page-intel-character-compat",
		"entity-page-intel-corporation-compat",
		"entity-page-intel-alliance-compat":
		return entityPageIntelSchema()
	case "entity-page-achievements", "entity-page-achievements-character-compat":
		return responseSchema(map[string]*huma.Schema{
			"achievements": arraySchema(entityAchievementSchema()),
		}, "achievements")
	case "entity-page-members", "entity-page-members-corporation-compat",
		"entity-page-members-alliance-compat":
		return entityPageMembersSchema()
	case "entity-page-corporations",
		"entity-page-corporations-alliance-compat":
		return entityPageCorporationsSchema()
	case "entity-page-most-valuable",
		"entity-page-most-valuable-generic-compat":
		return entriesResponseSchema(mostValuableKillSchema())
	case "entity-page-ship-classes",
		"entity-page-ship-classes-generic-compat":
		return responseSchema(map[string]*huma.Schema{
			"groups": arraySchema(responseSchema(map[string]*huma.Schema{
				"group_id": intSchema(), "group_name": stringSchema(),
				"losses": intSchema(), "isk_lost": numberSchema(),
			}, "group_id", "group_name", "losses", "isk_lost")),
		}, "groups")
	case "entity-page-top-lists", "entity-page-top-lists-generic-compat":
		return entityPageTopListsSchema()
	case "entity-top-character-compat", "entity-top-corporation-compat",
		"entity-top-alliance-compat":
		return entityLegacyTopSchema()

	// Map and cross-section rankings.
	case "map-regions":
		return mapRegionsResponseSchema()
	case "map-scope":
		return mapScopeResponseSchema()
	case "map-region":
		return mapRegionResponseSchema()
	case "map-sovereignty":
		return mapSovereigntyResponseSchema()
	case "map-aiid":
		return mapAIIDResponseSchema()
	case "stats-rankings":
		return entriesResponseSchema(statsRankingEntrySchema())

	// Saved fittings and the public fitting catalogue.
	case "fitting-create", "fitting-create-legacy":
		return fittingCreateResponseSchema()
	case "fitting-detail", "fitting-detail-legacy":
		return fittingDetailResponseSchema(true)
	case "fitting-update", "fitting-update-legacy":
		return fittingDetailResponseSchema(false)
	case "fitting-delete", "fitting-delete-legacy":
		return responseSchema(map[string]*huma.Schema{"ok": boolSchema()}, "ok")
	case "fitting-rating-put", "fitting-rating-put-legacy":
		return responseSchema(map[string]*huma.Schema{
			"rating": intSchema(), "aggregate": fittingRatingAggregateSchema(),
		}, "rating", "aggregate")
	case "fitting-rating-delete", "fitting-rating-delete-legacy":
		return responseSchema(map[string]*huma.Schema{
			"deleted": boolSchema(), "aggregate": fittingRatingAggregateSchema(),
		}, "deleted", "aggregate")
	case "fittings-community-latest", "fittings-community-latest-legacy",
		"fittings-community-top-rated", "fittings-community-top-rated-legacy":
		return responseSchema(map[string]*huma.Schema{
			"fits": arraySchema(communityFittingSchema()),
		}, "fits")
	case "fittings-trending", "fittings-trending-legacy":
		return responseSchema(map[string]*huma.Schema{
			"window_days":  intSchema(),
			"ranking_mode": stringSchema(),
			"families":     arraySchema(fittingFamilySchema(true, false)),
		}, "window_days", "ranking_mode", "families")
	case "fittings-popular-ships", "fittings-popular-ships-legacy":
		return responseSchema(map[string]*huma.Schema{
			"window_days": intSchema(),
			"ships":       arraySchema(popularFittingShipSchema()),
		}, "window_days", "ships")
	case "fittings-stats", "fittings-stats-legacy":
		return responseSchema(map[string]*huma.Schema{
			"fittings_known":     intSchema(),
			"killmails_analyzed": intSchema(),
			"community_fits":     intSchema(),
			"ratings_cast":       intSchema(),
		}, "fittings_known", "killmails_analyzed", "community_fits",
			"ratings_cast")
	case "fittings-roles", "fittings-roles-legacy":
		return responseSchema(map[string]*huma.Schema{
			"roles": arraySchema(fittingRoleSchema()),
		}, "roles")
	case "fittings-search", "fittings-search-legacy":
		return fittingSearchResponseSchema()
	case "fittings-search-availability", "fittings-search-availability-legacy":
		return responseSchema(map[string]*huma.Schema{
			"ship_type_id": intSchema(), "window_days": intSchema(),
			"role_counts": mapOfSchema(intSchema()), "type_counts": mapOfSchema(intSchema()),
		}, "ship_type_id", "window_days", "role_counts", "type_counts")
	case "fittings-alliance-doctrines",
		"fittings-alliance-doctrines-legacy":
		return responseSchema(map[string]*huma.Schema{
			"window_days": intSchema(),
			"entity_type": stringSchema(),
			"doctrines":   arraySchema(fittingDoctrineSchema()),
		}, "window_days", "entity_type", "doctrines")
	case "fittings-ship-families", "fittings-ship-families-legacy":
		return responseSchema(map[string]*huma.Schema{
			"ship_type_id": intSchema(), "window_days": intSchema(),
			"is_rare_hull": boolSchema(), "hull_cost": nullable(numberSchema()),
			"families": arraySchema(fittingFamilySchema(false, true)),
		}, "ship_type_id", "window_days", "is_rare_hull", "families")
	case "fittings-ship-metadata", "fittings-ship-metadata-legacy":
		return fittingShipMetadataSchema()
	case "fittings-ship-distributions", "fittings-ship-distributions-legacy",
		"fittings-search-distributions", "fittings-search-distributions-legacy":
		bucket := responseSchema(map[string]*huma.Schema{
			"bucket": intSchema(), "lower_bound": numberSchema(), "upper_bound": numberSchema(),
			"fit_count": intSchema(), "observation_count": intSchema(),
		}, "bucket", "lower_bound", "upper_bound", "fit_count", "observation_count")
		metric := responseSchema(map[string]*huma.Schema{
			"metric": stringSchema(), "fit_count": intSchema(), "observation_count": intSchema(),
			"minimum": numberSchema(), "maximum": numberSchema(), "p10": numberSchema(), "p25": numberSchema(),
			"median": numberSchema(), "p75": numberSchema(), "p90": numberSchema(),
			"lower_bound": numberSchema(), "upper_bound": numberSchema(), "calculated_at": stringSchema(),
			"buckets": arraySchema(bucket),
		}, "metric", "fit_count", "observation_count", "minimum", "maximum", "p10", "p25", "median", "p75", "p90", "lower_bound", "upper_bound", "calculated_at", "buckets")
		return responseSchema(map[string]*huma.Schema{
			"ship_type_id": intSchema(), "window_days": intSchema(), "metrics": arraySchema(metric),
		}, "ship_type_id", "window_days", "metrics")

	// Battle reports, wars, and faction warfare dashboards.
	case "battle-generator-entities":
		return battleGeneratorEntitiesSchema()
	case "battle-generator-preview", "battle-report":
		return battleDetailFrontendSchema()
	case "battle-generator-save":
		return responseSchema(map[string]*huma.Schema{
			"battle_id": intSchema(),
		}, "battle_id")
	case "killmail-battle-report":
		return &huma.Schema{OneOf: []*huma.Schema{
			responseSchema(map[string]*huma.Schema{
				"redirect": stringSchema(), "battle_id": intSchema(),
			}, "redirect", "battle_id"),
			battleDetailFrontendSchema(),
		}}
	case "killmail-battle-killlist", "battle-report-killlist",
		"killmail-battle-timeline", "battle-report-timeline":
		return responseSchema(map[string]*huma.Schema{
			"kills": arraySchema(killlistRowSchema()),
		}, "kills")
	case "killmail-battle-most-valuable", "battle-report-most-valuable":
		return entriesResponseSchema(battleMostValuableKillSchema())
	case "killmail-battle-composition", "battle-report-composition":
		return battleCompositionSchema()
	case "killmail-battle-intel", "battle-report-intel":
		return battleIntelSchema()
	case "conflict-battles":
		return conflictBattlesSchema()
	case "conflict-wars":
		return conflictWarsSchema()
	case "wars-overview-stats":
		return warOverviewStatsSchema()
	case "wars-eligible":
		return warEligibleSchema()
	case "war-dashboard-detail":
		return warDashboardDetailSchema(false)
	case "war-dashboard":
		return warDashboardDetailSchema(true)
	case "war-leaderboards":
		return warLeaderboardsSchema()
	case "war-members":
		return conflictMembersSchema("war_id")
	case "war-intel":
		return conflictIntelSchema("war_id")
	case "war-killlist":
		return killlistFrontendResponseSchema()
	case "faction-wars-dashboard":
		return factionWarsDashboardSchema()
	case "faction-war-dashboard-detail":
		return factionWarBaseSchema(false)
	case "faction-war-dashboard":
		return factionWarBaseSchema(true)
	case "faction-war-overview":
		return factionWarOverviewSchema()
	case "faction-war-systems":
		return factionWarSystemsSchema()
	case "faction-war-members":
		return conflictMembersSchema("matchup", "days")
	case "faction-war-intel":
		return conflictIntelSchema("matchup", "days")

	// Content, comments, moderation, and account-owned domains.
	case "announcement-admin-list":
		return responseSchema(map[string]*huma.Schema{
			"announcements": arraySchema(adminAnnouncementSchema()),
		}, "announcements")
	case "announcement-admin-detail", "announcement-admin-create",
		"announcement-admin-update", "announcement-admin-archive",
		"announcement-admin-archive-compat":
		return responseSchema(map[string]*huma.Schema{
			"announcement": adminAnnouncementSchema(),
		}, "announcement")
	case "blog-posts":
		return responseSchema(map[string]*huma.Schema{
			"posts":      arraySchema(blogPostSchema()),
			"nextCursor": nullable(stringSchema()),
		}, "posts", "nextCursor")
	case "blog-post", "blog-admin-detail", "blog-admin-preview",
		"blog-admin-create", "blog-admin-update":
		return responseSchema(map[string]*huma.Schema{
			"post": blogPostSchema(),
		}, "post")
	case "blog-admin-list":
		return responseSchema(map[string]*huma.Schema{
			"posts": arraySchema(blogPostSchema()),
		}, "posts")
	case "blog-admin-delete":
		return responseSchema(map[string]*huma.Schema{
			"ok": boolSchema(), "id": intSchema(),
		}, "ok", "id")
	case "comments-feed", "my-comments", "my-comments-live-alias":
		return commentsFeedSchema()
	case "comments-thread":
		return commentsThreadSchema()
	case "comments-create", "comment-detail", "comment-edit":
		return responseSchema(map[string]*huma.Schema{
			"comment": commentSchema(),
		}, "comment")
	case "comments-preview":
		return responseSchema(map[string]*huma.Schema{
			"html": stringSchema(), "error": stringSchema(),
		}, "html")
	case "comment-delete", "my-comment-delete",
		"my-comment-delete-live-alias":
		return responseSchema(map[string]*huma.Schema{"ok": boolSchema()}, "ok")
	case "comment-report":
		return responseSchema(map[string]*huma.Schema{
			"ok": boolSchema(), "reports_count": intSchema(), "flagged": boolSchema(),
		}, "ok", "reports_count", "flagged")
	case "comments-klipy-search", "comments-klipy-trending":
		return klipyResponseSchema()
	case "admin-comments", "admin-comments-live-queue-alias":
		return responseSchema(map[string]*huma.Schema{
			"comments": arraySchema(commentSchema()),
		}, "comments")
	case "admin-comment-moderation", "admin-comment-hide-live-alias",
		"admin-comment-restore-live-alias":
		return responseSchema(map[string]*huma.Schema{
			"ok": boolSchema(), "comment": commentSchema(),
		}, "ok", "comment")
	case "admin-comment-report-resolution",
		"admin-comment-report-resolution-live-alias":
		return responseSchema(map[string]*huma.Schema{
			"ok": boolSchema(), "report": commentReportSchema(),
		}, "ok", "report")
	case "admin-moderation", "admin-moderation-live-queue-alias":
		return moderationQueueResponseSchema()
	case "admin-moderation-review", "admin-moderation-approve-live-alias",
		"admin-moderation-reject-live-alias":
		return responseSchema(map[string]*huma.Schema{
			"ok": boolSchema(), "id": intSchema(), "status": stringSchema(),
		}, "ok", "id", "status")
	case "domains-mine", "domains-mine-compat":
		return responseSchema(map[string]*huma.Schema{
			"domains": arraySchema(domainSchema()),
		}, "domains")
	case "domain-create", "domain-create-compat", "domain-update",
		"domain-update-patch-compat", "domain-update-put-compat",
		"admin-domain-toggle":
		return responseSchema(map[string]*huma.Schema{
			"domain": domainSchema(),
		}, "domain")
	case "domain-subdomain-check", "domain-subdomain-check-compat":
		return responseSchema(map[string]*huma.Schema{
			"available": boolSchema(), "reason": stringSchema(),
		}, "available")
	case "domain-delete", "domain-delete-compat":
		return responseSchema(map[string]*huma.Schema{"deleted": boolSchema()}, "deleted")
	case "domain-asset-upload", "domain-asset-upload-compat":
		return responseSchema(map[string]*huma.Schema{
			"assetId": intSchema(), "status": stringSchema(), "message": stringSchema(),
		}, "assetId", "status", "message")
	case "domain-assets-delete-type", "domain-assets-delete-type-compat",
		"domain-asset-delete", "domain-asset-delete-compat":
		return responseSchema(map[string]*huma.Schema{"success": boolSchema()}, "success")
	case "domain-campaign-search", "domain-campaign-search-compat":
		return responseSchema(map[string]*huma.Schema{
			"campaigns": arraySchema(domainCampaignSchema()),
		}, "campaigns")
	case "admin-domains":
		return responseSchema(map[string]*huma.Schema{
			"domains": arraySchema(domainSchema()),
		}, "domains")
	case "admin-domain":
		return responseSchema(map[string]*huma.Schema{
			"domain": domainSchema(), "assets": arraySchema(domainAssetSchema()),
		}, "domain", "assets")
	case "admin-domain-asset-review":
		return responseSchema(map[string]*huma.Schema{
			"success": boolSchema(), "status": stringSchema(),
		}, "success", "status")

	// Wallet, markets, tools, sitemaps, and legacy archive compatibility.
	case "wallet-public":
		return publicWalletSchema()
	case "wallet-account", "wallet-account-legacy":
		return accountWalletSchema()
	case "wallet-account-balance", "wallet-account-balance-legacy":
		return walletBalanceSchema()
	case "wallet-admin":
		return adminWalletSchema()
	case "wallet-admin-sync":
		return responseSchema(map[string]*huma.Schema{"queued": boolSchema()}, "queued")
	case "wallet-admin-authorize":
		return responseSchema(map[string]*huma.Schema{"url": stringSchema()}, "url")
	case "backgrounds-reddit":
		return redditBackgroundsSchema()
	case "market-tree":
		return responseSchema(map[string]*huma.Schema{
			"groups": arraySchema(marketTreeNodeSchema()),
		}, "groups")
	case "market-group-items":
		return marketGroupItemsSchema()
	case "bulk-prices":
		return responseSchema(map[string]*huma.Schema{
			"prices": mapOfSchema(numberSchema()),
		}, "prices")
	case "market-item-orders":
		return marketItemOrdersSchema()
	case "market-item-history":
		return marketItemHistorySchema()
	case "dscan-analyze", "dscan-analyze-legacy",
		"dscan-get", "dscan-get-legacy":
		return directionalScanSchema()
	case "localscan-analyze", "localscan-analyze-legacy",
		"localscan-get", "localscan-get-legacy":
		return localScanSchema()
	case "dscan-save", "dscan-save-legacy",
		"localscan-save", "localscan-save-legacy":
		return responseSchema(map[string]*huma.Schema{"hash": stringSchema()}, "hash")
	case "sitemap", "sitemap-alliances-compat", "sitemap-battles-compat",
		"sitemap-characters-compat", "sitemap-coalitions-compat", "sitemap-corporations-compat",
		"sitemap-items-compat", "sitemap-kills-compat",
		"sitemap-regions-compat", "sitemap-ships-compat",
		"sitemap-systems-compat", "sitemap-wars-compat":
		return arraySchema(sitemapEntrySchema())
	case "legacy-archive-autocomplete":
		return arraySchema(responseSchema(map[string]*huma.Schema{
			"name": stringSchema(), "id": nullable(intSchema()),
		}, "name", "id"))
	case "legacy-archive-kills":
		return responseSchema(map[string]*huma.Schema{
			"kills": arraySchema(killlistRowSchema()), "hasMore": boolSchema(),
			"cursor": nullable(intSchema()),
		}, "kills", "hasMore", "cursor")
	case "legacy-archive-stats":
		return responseSchema(map[string]*huma.Schema{
			"killmails": intSchema(), "characters": intSchema(),
			"corporations": intSchema(), "alliances": intSchema(),
		}, "killmails", "characters", "corporations", "alliances")
	case "legacy-archive-top":
		return entriesResponseSchema(topKillEntrySchema())
	case "legacy-archive-kill":
		return legacyArchiveKillSchema()
	case "domain-killlist", "domain-region-killlist",
		"domain-constellation-killlist", "domain-system-killlist":
		return killlistFrontendResponseSchema()
	case "domain-kills-most-valuable":
		return entriesResponseSchema(mostValuableKillSchema())
	case "domain-kills-top":
		return entriesResponseSchema(topKillEntrySchema())
	case "domain-statistics":
		return domainStatisticsSchema()

	// Campaign lists, detail pages, mutations, prizes, and administration.
	case "campaigns":
		return campaignListResponseSchema()
	case "campaign-detail", "campaign-detail-legacy":
		return campaignDetailResponseSchema()
	case "campaign-killmails", "campaign-killlist-legacy":
		return responseSchema(map[string]*huma.Schema{
			"kills": arraySchema(killlistRowSchema()), "hasMore": boolSchema(),
			"cursor": nullable(intSchema()),
		}, "kills", "hasMore", "cursor")
	case "campaign-create", "campaign-create-legacy":
		return responseSchema(map[string]*huma.Schema{
			"campaign_id": stringSchema(), "estimated_killmails": intSchema(),
			"initial_contribution": stringSchema(), "replayed": boolSchema(),
		}, "campaign_id", "estimated_killmails", "initial_contribution",
			"replayed")
	case "campaign-update", "campaign-update-legacy",
		"campaign-update-browser-legacy":
		return responseSchema(map[string]*huma.Schema{
			"updated": boolSchema(), "recompute": boolSchema(),
		}, "updated")
	case "campaign-delete", "campaign-delete-legacy":
		return responseSchema(map[string]*huma.Schema{"deleted": boolSchema()}, "deleted")
	case "campaign-prize-contribute", "campaign-prize-contribute-legacy":
		return responseSchema(map[string]*huma.Schema{
			"contributed": stringSchema(), "replayed": boolSchema(),
			"balance": stringSchema(),
		}, "contributed", "replayed", "balance")
	case "campaign-prize-claim", "campaign-prize-claim-legacy":
		return responseSchema(map[string]*huma.Schema{
			"claimed": boolSchema(), "rank": intSchema(),
		}, "claimed", "rank")
	case "campaign-admin-list":
		return campaignAdminListSchema()
	case "campaign-admin-action", "campaign-admin-action-legacy":
		return responseSchema(map[string]*huma.Schema{
			"ok": boolSchema(), "action": stringSchema(), "dispatched": boolSchema(),
		}, "ok", "action", "dispatched")
	case "campaign-prize-paid", "campaign-prize-paid-legacy":
		return responseSchema(map[string]*huma.Schema{"paid": boolSchema()}, "paid")
	}
	return nil
}

func successResponseSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{"success": boolSchema()}, "success")
}

func coalitionDirectorySummarySchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"coalition_id": intSchema(), "slug": stringSchema(), "name": stringSchema(),
		"description": stringSchema(), "source_url": nullable(stringSchema()),
		"revision": intSchema(), "created_at": timestampSchema(), "updated_at": timestampSchema(),
		"created_by_character_id":   nullable(intSchema()),
		"created_by_character_name": nullable(stringSchema()),
		"updated_by_character_id":   nullable(intSchema()),
		"updated_by_character_name": nullable(stringSchema()),
		"alliance_count":            intSchema(), "member_count": intSchema(), "system_count": intSchema(),
		"kills": intSchema(), "losses": intSchema(),
		"isk_destroyed": numberSchema(), "isk_lost": numberSchema(),
	}, "coalition_id", "slug", "name", "description", "source_url", "revision",
		"created_at", "updated_at", "alliance_count", "member_count", "system_count",
		"kills", "losses", "isk_destroyed", "isk_lost")
}

func coalitionDirectoryListResponseSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"coalitions":        arraySchema(coalitionDirectorySummarySchema()),
		"stats_window_days": intSchema(),
	}, "coalitions", "stats_window_days")
}

func coalitionDirectoryDetailResponseSchema() *huma.Schema {
	alliance := responseSchema(map[string]*huma.Schema{
		"alliance_id": intSchema(), "name": stringSchema(), "ticker": stringSchema(),
		"member_count": intSchema(), "corporation_count": intSchema(), "system_count": intSchema(),
		"added_at": timestampSchema(), "added_by_character_id": nullable(intSchema()),
		"added_by_character_name": nullable(stringSchema()),
	}, "alliance_id", "name", "ticker", "member_count", "corporation_count", "system_count", "added_at")
	edit := responseSchema(map[string]*huma.Schema{
		"edit_id": intSchema(), "editor_character_id": nullable(intSchema()),
		"editor_character_name": stringSchema(), "action": stringSchema(), "summary": stringSchema(),
		"changes":    openJSONObjectSchema("Before and after snapshots for this edit."),
		"created_at": timestampSchema(),
	}, "edit_id", "editor_character_id", "editor_character_name", "action", "summary", "changes", "created_at")
	return responseSchema(map[string]*huma.Schema{
		"coalition":         coalitionDirectorySummarySchema(),
		"alliances":         arraySchema(alliance),
		"edits":             arraySchema(edit),
		"stats_window_days": intSchema(),
	}, "coalition", "alliances", "edits", "stats_window_days")
}

func openJSONObjectSchema(description string) *huma.Schema {
	return &huma.Schema{
		Type:                 huma.TypeObject,
		Description:          description,
		AdditionalProperties: true,
	}
}

func riverJobSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "state": stringSchema(), "attempt": intSchema(),
		"max_attempts": intSchema(), "priority": intSchema(), "queue": stringSchema(),
		"kind": stringSchema(), "created_at": timestampSchema(),
		"scheduled_at": timestampSchema(), "attempted_at": nullable(timestampSchema()),
		"finalized_at": nullable(timestampSchema()), "attempted_by": arraySchema(stringSchema()),
		"tags": arraySchema(stringSchema()), "args": openJSONObjectSchema("Job input arguments."),
		"errors":   arraySchema(openJSONObjectSchema("A failed attempt.")),
		"metadata": openJSONObjectSchema("River and application job metadata."),
		"output":   nullable(openJSONObjectSchema("Durable worker output, when recorded.")),
	}, "id", "state", "attempt", "max_attempts", "priority", "queue", "kind",
		"created_at", "scheduled_at", "attempted_by", "tags", "args", "errors", "metadata")
}

func riverQueueSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"name": stringSchema(), "depth": openJSONObjectSchema("Counts by River job state."),
		"cron": boolSchema(), "concurrency": intSchema(), "description": stringSchema(),
		"paused_at": nullable(timestampSchema()), "worker_updated_at": nullable(timestampSchema()),
		"worker_active": boolSchema(),
	})
}

func accountUserSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"characterId":            intSchema(),
		"characterName":          stringSchema(),
		"isAdmin":                boolSchema(),
		"corporationId":          nullable(intSchema()),
		"corporationName":        nullable(stringSchema()),
		"allianceId":             nullable(intSchema()),
		"allianceName":           nullable(stringSchema()),
		"lastSeenNotificationId": intSchema(),
		"characterOwnerHash":     stringSchema(),
		"settings":               accountPreferencesSchema(),
	}, "characterId", "characterName", "isAdmin", "corporationId",
		"corporationName", "allianceId", "allianceName",
		"lastSeenNotificationId")
}

func accountPreferencesSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"theme":       openJSONObjectSchema("User-selected theme settings."),
		"defaultTabs": openJSONObjectSchema("Default tab keyed by page type."),
		"boards":      accountBoardStateSchema(),
	})
}

func accountBoardStateSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"pinned":    arraySchema(stringSchema()),
		"dismissed": arraySchema(stringSchema()),
	}, "pinned", "dismissed")
}

func accountBoardsSchema() *huma.Schema {
	board := responseSchema(map[string]*huma.Schema{
		"key":     stringSchema(),
		"host":    stringSchema(),
		"url":     stringSchema(),
		"name":    stringSchema(),
		"tracked": boolSchema(),
		"pinned":  boolSchema(),
	}, "key", "host", "url", "name", "tracked", "pinned")
	current := responseSchema(map[string]*huma.Schema{
		"key": stringSchema(), "name": stringSchema(), "listed": boolSchema(),
	}, "key", "name", "listed")
	return responseSchema(map[string]*huma.Schema{
		"boards":        arraySchema(board),
		"current":       nullable(current),
		"authenticated": boolSchema(),
		"atCapacity":    boolSchema(),
	}, "boards", "current", "authenticated", "atCapacity")
}

func accountOverviewSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"account": responseSchema(map[string]*huma.Schema{
			"characterId":   intSchema(),
			"characterName": stringSchema(),
			"isAdmin":       boolSchema(),
			"lastLogin":     nullable(timestampSchema()),
			"createdAt":     nullable(timestampSchema()),
		}, "characterId", "characterName", "isAdmin", "lastLogin", "createdAt"),
		"esiStats": esiStatsSchema(),
		"esiToken": nullable(responseSchema(map[string]*huma.Schema{
			"scopeCount":  intSchema(),
			"tokenExpiry": nullable(timestampSchema()),
			"lastFetched": nullable(timestampSchema()),
		}, "scopeCount", "tokenExpiry", "lastFetched")),
	}, "account", "esiStats", "esiToken")
}

func accountSettingsSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"account": responseSchema(map[string]*huma.Schema{
			"characterId":        intSchema(),
			"characterName":      stringSchema(),
			"characterOwnerHash": nullable(stringSchema()),
			"isAdmin":            boolSchema(),
			"lastLogin":          nullable(timestampSchema()),
			"createdAt":          nullable(timestampSchema()),
		}, "characterId", "characterName", "characterOwnerHash", "isAdmin",
			"lastLogin", "createdAt"),
		"preferences": accountPreferencesSchema(),
		"esiToken": nullable(responseSchema(map[string]*huma.Schema{
			"scopes":          arraySchema(stringSchema()),
			"effectiveScopes": arraySchema(stringSchema()),
			"revokedScopes":   arraySchema(stringSchema()),
			"scopeCount":      intSchema(),
			"tokenExpiry":     nullable(timestampSchema()),
			"lastFetched":     nullable(timestampSchema()),
			"disabled":        boolSchema(),
		}, "scopes", "effectiveScopes", "revokedScopes", "scopeCount",
			"tokenExpiry", "lastFetched", "disabled")),
	}, "account", "preferences", "esiToken")
}

func accountSessionSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"id":              intSchema(),
		"current":         boolSchema(),
		"ipAddress":       nullable(stringSchema()),
		"countryCode":     nullable(stringSchema()),
		"browser":         stringSchema(),
		"operatingSystem": stringSchema(),
		"device":          stringSchema(),
		"createdAt":       timestampSchema(),
		"lastSeenAt":      timestampSchema(),
		"expiresAt":       timestampSchema(),
	}, "id", "current", "ipAddress", "countryCode", "browser",
		"operatingSystem", "device", "createdAt", "lastSeenAt", "expiresAt")
}

func manageableEntitiesSchema() *huma.Schema {
	pending := responseSchema(map[string]*huma.Schema{
		"body": stringSchema(), "body_format": stringSchema(),
		"submitted_at": timestampSchema(),
	}, "body", "body_format", "submitted_at")
	base := map[string]*huma.Schema{
		"id":                        intSchema(),
		"name":                      stringSchema(),
		"ticker":                    stringSchema(),
		"esi_description":           nullable(stringSchema()),
		"custom_description":        nullable(stringSchema()),
		"custom_description_format": stringSchema(),
		"canEdit":                   boolSchema(),
		"pending_submission":        nullable(pending),
		"ceo_id":                    nullable(intSchema()),
		"ceo_name":                  nullable(stringSchema()),
		"executor_corporation_id":   nullable(intSchema()),
		"executor_ceo_id":           nullable(intSchema()),
		"executor_ceo_name":         nullable(stringSchema()),
	}
	entity := responseSchema(base, "id", "name", "custom_description",
		"custom_description_format", "canEdit", "pending_submission")
	return responseSchema(map[string]*huma.Schema{
		"character":   entity,
		"corporation": nullable(entity),
		"alliance":    nullable(entity),
	}, "character", "corporation", "alliance")
}

func esiStatsSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"total_requests":  intSchema(),
		"total_errors":    intSchema(),
		"total_new_items": intSchema(),
		"last_request":    nullable(timestampSchema()),
		"requests_24h":    intSchema(),
		"errors_24h":      intSchema(),
		"new_items_24h":   intSchema(),
	})
}

func esiMetricsSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"volumeByHour": arraySchema(responseSchema(map[string]*huma.Schema{
			"hour":      stringSchema(),
			"total":     intSchema(),
			"errors":    intSchema(),
			"new_items": intSchema(),
		}, "hour", "total", "errors", "new_items")),
		"rateLimit": responseSchema(map[string]*huma.Schema{
			"request_count": intSchema(),
		}, "request_count"),
		"responseTime": responseSchema(map[string]*huma.Schema{
			"avg_ms": nullable(intSchema()), "p95_ms": nullable(intSchema()),
		}, "avg_ms", "p95_ms"),
	}, "volumeByHour", "rateLimit", "responseTime")
}

func esiLogRowSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"id":                  intSchema(),
		"endpoint":            stringSchema(),
		"method":              stringSchema(),
		"status_code":         nullable(intSchema()),
		"success":             boolSchema(),
		"error_message":       nullable(stringSchema()),
		"items_returned":      nullable(intSchema()),
		"new_items":           nullable(intSchema()),
		"source":              stringSchema(),
		"request_duration_ms": nullable(intSchema()),
		"created_at":          timestampSchema(),
		"endpoint_type":       stringSchema(),
		"endpoint_action":     stringSchema(),
	}, "id", "endpoint", "method", "status_code", "success",
		"error_message", "items_returned", "new_items", "source",
		"request_duration_ms", "created_at", "endpoint_type",
		"endpoint_action")
}

func esiLogsSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"rows":    arraySchema(esiLogRowSchema()),
		"newRows": boolSchema(),
		"total":   intSchema(),
		"page":    intSchema(),
		"limit":   intSchema(),
		"pages":   intSchema(),
		"sources": arraySchema(stringSchema()),
	}, "rows")
}

func announcementSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"id":         intSchema(),
		"tier":       intSchema(),
		"title":      stringSchema(),
		"body_md":    stringSchema(),
		"body_html":  stringSchema(),
		"color":      stringSchema(),
		"icon":       nullable(stringSchema()),
		"link_url":   nullable(stringSchema()),
		"link_label": nullable(stringSchema()),
		"starts_at":  timestampSchema(),
		"expires_at": timestampSchema(),
		"created_at": timestampSchema(),
	}, "id", "tier", "title", "body_md", "body_html", "color", "icon",
		"link_url", "link_label", "starts_at", "expires_at", "created_at")
}

func notificationRepliesSchema() *huma.Schema {
	reply := responseSchema(map[string]*huma.Schema{
		"id":                intSchema(),
		"target_type":       intSchema(),
		"target_id":         intSchema(),
		"parent_id":         nullable(intSchema()),
		"root_id":           nullable(intSchema()),
		"body_html":         stringSchema(),
		"created_at":        timestampSchema(),
		"character_id":      intSchema(),
		"character_name":    stringSchema(),
		"corporation_id":    intSchema(),
		"corporation_name":  stringSchema(),
		"alliance_id":       nullable(intSchema()),
		"alliance_name":     nullable(stringSchema()),
		"parent_comment_id": intSchema(),
		"parent_snippet":    stringSchema(),
	}, "id", "target_type", "target_id", "parent_id", "root_id",
		"body_html", "created_at", "character_id", "character_name",
		"corporation_id", "corporation_name", "alliance_id", "alliance_name",
		"parent_comment_id", "parent_snippet")
	return responseSchema(map[string]*huma.Schema{
		"replies":   arraySchema(reply),
		"highestId": intSchema(),
	}, "replies", "highestId")
}

func adminOverviewSchema() *huma.Schema {
	counter := responseSchema(map[string]*huma.Schema{
		"total": intSchema(), "recent7d": intSchema(),
		"last24h": intSchema(), "last7d": intSchema(),
	})
	return responseSchema(map[string]*huma.Schema{
		"users":     counter,
		"killmails": counter,
		"comments":  counter,
		"moderation": responseSchema(map[string]*huma.Schema{
			"pending": intSchema(), "flagged": intSchema(),
		}),
		"esi": responseSchema(map[string]*huma.Schema{
			"total": intSchema(), "errors": intSchema(),
			"errorRate": numberSchema(),
		}),
	}, "users", "killmails", "comments", "moderation", "esi")
}

func adminUserSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"character_id":         intSchema(),
		"character_name":       stringSchema(),
		"character_owner_hash": nullable(stringSchema()),
		"discord_user_id":      nullable(stringSchema()),
		"is_admin":             boolSchema(),
		"last_login":           nullable(timestampSchema()),
		"created_at":           nullable(timestampSchema()),
		"updated_at":           nullable(timestampSchema()),
	})
}

func adminUsersListSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"users": arraySchema(adminUserSchema()),
		"total": intSchema(),
		"page":  intSchema(),
		"limit": intSchema(),
		"pages": intSchema(),
	}, "users", "total", "page", "limit", "pages")
}

func adminUserDetailSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"user": adminUserSchema(),
		"config": arraySchema(recordSchema(map[string]*huma.Schema{
			"key": stringSchema(), "value": stringSchema(),
		})),
		"esiToken": nullable(recordSchema(map[string]*huma.Schema{
			"scopes":       arraySchema(stringSchema()),
			"token_expiry": nullable(timestampSchema()),
		})),
		"esiStats": esiStatsSchema(),
	}, "user", "config", "esiToken", "esiStats")
}

func entriesResponseSchema(item *huma.Schema) *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"entries": arraySchema(item),
	}, "entries")
}

func topKillEntrySchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"id":        intSchema(),
		"name":      stringSchema(),
		"count":     intSchema(),
		"type":      stringSchema(),
		"region_id": nullable(intSchema()),
		"palette":   nullable(stringSchema()),
	}, "id", "name", "count", "type")
}

func mostValuableKillSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"killmail_id":             intSchema(),
		"killmail_hash":           stringSchema(),
		"ship_type_id":            intSchema(),
		"ship_name":               stringSchema(),
		"total_value":             numberSchema(),
		"victim_character_id":     nullable(intSchema()),
		"victim_character_name":   nullable(stringSchema()),
		"victim_corporation_name": nullable(stringSchema()),
		"victim_alliance_name":    nullable(stringSchema()),
	}, "killmail_id", "killmail_hash", "ship_type_id", "ship_name",
		"total_value", "victim_character_id", "victim_character_name",
		"victim_corporation_name", "victim_alliance_name")
}

func killmailEditorFitSchema() *huma.Schema {
	module := responseSchema(map[string]*huma.Schema{
		"slot_group":     intSchema(),
		"ordinal":        intSchema(),
		"type_id":        intSchema(),
		"name":           nullable(stringSchema()),
		"charge_type_id": nullable(intSchema()),
	}, "slot_group", "ordinal", "type_id", "name", "charge_type_id")
	drone := responseSchema(map[string]*huma.Schema{
		"type_id": intSchema(), "name": nullable(stringSchema()),
		"quantity": intSchema(),
	}, "type_id", "name", "quantity")
	return responseSchema(map[string]*huma.Schema{
		"shipTypeId": intSchema(),
		"name":       stringSchema(),
		"modules":    arraySchema(module),
		"drones":     arraySchema(drone),
	}, "shipTypeId", "name", "modules", "drones")
}

func killmailSiblingSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"killmail_id":   intSchema(),
		"ship_type_id":  nullable(intSchema()),
		"ship_group_id": nullable(intSchema()),
		"ship_name":     nullable(stringSchema()),
		"total_value":   numberSchema(),
		"killmail_time": timestampSchema(),
	}, "killmail_id", "ship_type_id", "ship_group_id", "ship_name",
		"total_value", "killmail_time")
}

func killmailSubmissionResponseSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"accepted":    intSchema(),
		"rejected":    intSchema(),
		"existing":    intSchema(),
		"total":       intSchema(),
		"killmails":   arraySchema(intSchema()),
		"existingIds": arraySchema(intSchema()),
		"message":     stringSchema(),
	}, "accepted", "rejected", "existing", "total", "killmails",
		"existingIds")
}

func universeStatsSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"kills":       intSchema(),
		"total_value": numberSchema(),
		"npc_kills":   intSchema(),
		"pod_kills":   intSchema(),
	}, "kills", "total_value", "npc_kills", "pod_kills")
}

func universeSovereigntyDistributionSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"alliance_id":   nullable(intSchema()),
		"alliance_name": nullable(stringSchema()),
		"faction_id":    nullable(intSchema()),
		"faction_name":  nullable(stringSchema()),
		"system_count":  intSchema(),
	}, "alliance_id", "alliance_name", "faction_id", "faction_name",
		"system_count")
}

func universeRegionResponseSchema() *huma.Schema {
	region := responseSchema(map[string]*huma.Schema{
		"region_id":           intSchema(),
		"name":                stringSchema(),
		"description":         nullable(stringSchema()),
		"faction_id":          nullable(intSchema()),
		"faction_name":        nullable(stringSchema()),
		"constellation_count": intSchema(),
		"system_count":        intSchema(),
	}, "region_id", "name", "description", "faction_id", "faction_name",
		"constellation_count", "system_count")
	constellation := responseSchema(map[string]*huma.Schema{
		"constellation_id":   intSchema(),
		"constellation_name": stringSchema(),
		"system_count":       intSchema(),
		"alliance_id":        nullable(intSchema()),
		"alliance_name":      nullable(stringSchema()),
		"faction_id":         nullable(intSchema()),
		"faction_name":       nullable(stringSchema()),
	}, "constellation_id", "constellation_name", "system_count",
		"alliance_id", "alliance_name", "faction_id", "faction_name")
	topSystem := responseSchema(map[string]*huma.Schema{
		"solar_system_id": intSchema(), "system_name": stringSchema(),
		"security": numberSchema(), "kills": intSchema(),
		"total_value": numberSchema(),
	}, "solar_system_id", "system_name", "security", "kills", "total_value")
	return responseSchema(map[string]*huma.Schema{
		"region":          region,
		"constellations":  arraySchema(constellation),
		"stats":           universeStatsSchema(),
		"sovDistribution": arraySchema(universeSovereigntyDistributionSchema()),
		"topSystems":      arraySchema(topSystem),
	}, "region", "constellations", "stats", "sovDistribution", "topSystems")
}

func universeConstellationResponseSchema() *huma.Schema {
	constellation := responseSchema(map[string]*huma.Schema{
		"constellation_id":   intSchema(),
		"constellation_name": stringSchema(),
		"region_id":          intSchema(),
		"region_name":        nullable(stringSchema()),
		"faction_id":         nullable(intSchema()),
	}, "constellation_id", "constellation_name", "region_id", "region_name",
		"faction_id")
	system := responseSchema(map[string]*huma.Schema{
		"solar_system_id": intSchema(), "system_name": stringSchema(),
		"security": numberSchema(), "alliance_id": nullable(intSchema()),
		"alliance_name": nullable(stringSchema()),
		"faction_id":    nullable(intSchema()), "faction_name": nullable(stringSchema()),
	}, "solar_system_id", "system_name", "security", "alliance_id",
		"alliance_name", "faction_id", "faction_name")
	return responseSchema(map[string]*huma.Schema{
		"constellation":   constellation,
		"systems":         arraySchema(system),
		"sovDistribution": arraySchema(universeSovereigntyDistributionSchema()),
		"stats":           universeStatsSchema(),
	}, "constellation", "systems", "sovDistribution", "stats")
}

func universeSystemSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"solar_system_id":    intSchema(),
		"system_name":        stringSchema(),
		"security":           numberSchema(),
		"security_class":     nullable(stringSchema()),
		"faction_id":         nullable(intSchema()),
		"sun_type_id":        nullable(intSchema()),
		"sun_type_name":      nullable(stringSchema()),
		"border":             boolSchema(),
		"fringe":             boolSchema(),
		"corridor":           boolSchema(),
		"hub":                boolSchema(),
		"international":      boolSchema(),
		"regional":           boolSchema(),
		"constellation_id":   intSchema(),
		"constellation_name": stringSchema(),
		"region_id":          intSchema(),
		"region_name":        stringSchema(),
	}, "solar_system_id", "system_name", "security", "security_class",
		"faction_id", "sun_type_id", "sun_type_name", "border", "fringe",
		"corridor", "hub", "international", "regional", "constellation_id",
		"constellation_name", "region_id", "region_name")
}

func universeActivityEntrySchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"ship_kills": intSchema(), "npc_kills": intSchema(),
		"pod_kills": intSchema(), "ship_jumps": intSchema(),
		"timestamp": timestampSchema(),
	}, "ship_kills", "npc_kills", "pod_kills", "ship_jumps")
}

func universeSystemResponseSchema() *huma.Schema {
	celestial := responseSchema(map[string]*huma.Schema{
		"item_id": intSchema(), "item_name": stringSchema(),
		"type_id": intSchema(), "group_id": intSchema(),
		"type_name": nullable(stringSchema()), "category": stringSchema(),
	}, "item_id", "item_name", "type_id", "group_id", "type_name", "category")
	station := responseSchema(map[string]*huma.Schema{
		"station_id": intSchema(), "station_name": stringSchema(),
		"type_id": intSchema(), "corporation_id": intSchema(),
		"corporation_name": nullable(stringSchema()),
		"operation_name":   nullable(stringSchema()),
	}, "station_id", "station_name", "type_id", "corporation_id",
		"corporation_name", "operation_name")
	connection := responseSchema(map[string]*huma.Schema{
		"to_solar_system_id": intSchema(), "system_name": stringSchema(),
		"security": numberSchema(), "region_id": intSchema(),
		"is_regional": boolSchema(),
	}, "to_solar_system_id", "system_name", "security", "region_id",
		"is_regional")
	sovereignty := responseSchema(map[string]*huma.Schema{
		"alliance_id": nullable(intSchema()), "alliance_name": nullable(stringSchema()),
		"corporation_id":   nullable(intSchema()),
		"corporation_name": nullable(stringSchema()),
		"faction_id":       nullable(intSchema()), "faction_name": nullable(stringSchema()),
	}, "alliance_id", "alliance_name", "corporation_id", "corporation_name",
		"faction_id", "faction_name")
	history := responseSchema(map[string]*huma.Schema{
		"alliance_id": nullable(intSchema()), "alliance_name": nullable(stringSchema()),
		"corporation_id":   nullable(intSchema()),
		"corporation_name": nullable(stringSchema()),
		"faction_id":       nullable(intSchema()), "faction_name": nullable(stringSchema()),
		"date": timestampSchema(),
	}, "alliance_id", "alliance_name", "corporation_id", "corporation_name",
		"faction_id", "faction_name", "date")
	structure := responseSchema(map[string]*huma.Schema{
		"structure_id": intSchema(), "name": stringSchema(),
		"owner_id": intSchema(), "owner_name": nullable(stringSchema()),
		"type_id": intSchema(), "type_name": nullable(stringSchema()),
		"is_market": boolSchema(), "last_seen": nullable(timestampSchema()),
	}, "structure_id", "name", "owner_id", "owner_name", "type_id",
		"type_name", "is_market", "last_seen")
	activity := responseSchema(map[string]*huma.Schema{
		"latest":      nullable(universeActivityEntrySchema()),
		"summary_24h": nullable(universeActivityEntrySchema()),
		"history":     arraySchema(universeActivityEntrySchema()),
	}, "latest", "summary_24h", "history")
	return responseSchema(map[string]*huma.Schema{
		"system":             universeSystemSchema(),
		"celestials":         mapOfSchema(intSchema()),
		"celestialList":      arraySchema(celestial),
		"stations":           arraySchema(station),
		"connections":        arraySchema(connection),
		"sovereignty":        nullable(sovereignty),
		"stats":              universeStatsSchema(),
		"sovereigntyHistory": arraySchema(history),
		"structures":         arraySchema(structure),
		"activity":           activity,
	}, "system", "celestials", "celestialList", "stations", "connections",
		"sovereignty", "stats", "sovereigntyHistory", "structures", "activity")
}

func mapOfSchema(value *huma.Schema) *huma.Schema {
	return &huma.Schema{
		Type:                 huma.TypeObject,
		AdditionalProperties: value,
	}
}

func universePriceSummarySchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"latest":         numberSchema(),
		"latest_date":    dateSchema(),
		"average_90d":    numberSchema(),
		"highest_90d":    numberSchema(),
		"lowest_90d":     nullable(numberSchema()),
		"avg_volume_90d": numberSchema(),
	}, "latest", "latest_date", "average_90d", "highest_90d", "lowest_90d")
}

func universeTypeResponseSchema() *huma.Schema {
	item := responseSchema(map[string]*huma.Schema{
		"type_id": intSchema(), "name": stringSchema(),
		"description": nullable(stringSchema()),
		"group_id":    intSchema(), "category_id": intSchema(),
		"mass": nullable(numberSchema()), "volume": nullable(numberSchema()),
		"capacity": nullable(numberSchema()), "portion_size": nullable(intSchema()),
		"packaged_volume": nullable(numberSchema()), "radius": nullable(numberSchema()),
		"meta_group_id": nullable(intSchema()), "market_group_id": nullable(intSchema()),
		"race_id": nullable(intSchema()), "faction_id": nullable(intSchema()),
		"base_price": nullable(numberSchema()), "published": boolSchema(),
		"group_name": nullable(stringSchema()), "category_name": nullable(stringSchema()),
		"meta_group_name": nullable(stringSchema()), "tech_level": nullable(numberSchema()),
		"meta_level": nullable(numberSchema()), "is_ship": boolSchema(),
	}, "type_id", "name", "description", "group_id", "category_id", "mass",
		"volume", "capacity", "portion_size", "packaged_volume", "radius",
		"meta_group_id", "market_group_id", "race_id", "faction_id",
		"base_price", "published", "group_name", "category_name",
		"meta_group_name", "tech_level", "meta_level", "is_ship")
	attribute := responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "name": stringSchema(), "value": numberSchema(),
	}, "id", "value")
	skill := responseSchema(map[string]*huma.Schema{
		"type_id": intSchema(), "name": nullable(stringSchema()),
		"level": numberSchema(),
	}, "type_id", "name", "level")
	material := responseSchema(map[string]*huma.Schema{
		"type_id": intSchema(), "name": nullable(stringSchema()),
		"quantity": intSchema(),
	}, "type_id", "name", "quantity")
	breadcrumb := responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "name": stringSchema(), "slug": stringSchema(),
	}, "id", "name", "slug")
	price := responseSchema(map[string]*huma.Schema{
		"date": dateSchema(), "average": numberSchema(), "highest": numberSchema(),
		"lowest": numberSchema(), "volume": intSchema(),
	}, "date", "average", "highest", "lowest", "volume")
	insurance := recordSchema(map[string]*huma.Schema{
		"name": stringSchema(), "cost": numberSchema(), "payout": numberSchema(),
	})
	variation := responseSchema(map[string]*huma.Schema{
		"type_id": intSchema(), "name": stringSchema(),
		"meta_group_id": nullable(intSchema()),
	}, "type_id", "name", "meta_group_id")
	return responseSchema(map[string]*huma.Schema{
		"item":             item,
		"shipAttributes":   nullable(mapOfSchema(arraySchema(attribute))),
		"attributes":       arraySchema(attribute),
		"requiredSkills":   arraySchema(skill),
		"materials":        arraySchema(material),
		"marketBreadcrumb": arraySchema(breadcrumb),
		"variations":       arraySchema(variation),
		"pricing": responseSchema(map[string]*huma.Schema{
			"summary":       nullable(universePriceSummarySchema()),
			"history":       arraySchema(price),
			"insurance":     arraySchema(insurance),
			"customSummary": nullable(universePriceSummarySchema()),
			"customHistory": arraySchema(responseSchema(map[string]*huma.Schema{
				"date": dateSchema(), "price": numberSchema(),
			}, "date", "price")),
		}, "summary", "history", "insurance", "customSummary", "customHistory"),
	}, "item", "shipAttributes", "attributes", "requiredSkills", "materials",
		"marketBreadcrumb", "variations", "pricing")
}

func universeGroupResponseSchema() *huma.Schema {
	group := responseSchema(map[string]*huma.Schema{
		"group_id": intSchema(), "name": nullable(stringSchema()),
		"category_id": nullable(intSchema()), "category_name": nullable(stringSchema()),
		"published": nullable(boolSchema()), "icon_id": nullable(intSchema()),
		"category_published": nullable(boolSchema()),
		"type_count":         intSchema(), "published_type_count": intSchema(),
	}, "group_id", "name", "category_id", "category_name", "published",
		"icon_id", "category_published", "type_count", "published_type_count")
	typeItem := responseSchema(map[string]*huma.Schema{
		"type_id": intSchema(), "name": nullable(stringSchema()),
		"description": nullable(stringSchema()), "published": nullable(boolSchema()),
		"meta_group_id": nullable(intSchema()), "meta_group_name": nullable(stringSchema()),
		"volume": nullable(numberSchema()), "mass": nullable(numberSchema()),
		"base_price": nullable(numberSchema()),
	}, "type_id", "name", "description", "published", "meta_group_id",
		"meta_group_name", "volume", "mass", "base_price")
	return responseSchema(map[string]*huma.Schema{
		"group": group,
		"types": arraySchema(typeItem),
	}, "group", "types")
}

func shipMatchupResponseSchema() *huma.Schema {
	module := responseSchema(map[string]*huma.Schema{
		"slot_group": intSchema(), "type_id": intSchema(),
		"name": nullable(stringSchema()),
	}, "slot_group", "type_id", "name")
	fit := responseSchema(map[string]*huma.Schema{
		"family_hash": stringSchema(), "uses": intSchema(), "pct": numberSchema(),
		"modules": arraySchema(module),
	}, "family_hash", "uses", "pct", "modules")
	return responseSchema(map[string]*huma.Schema{
		"attacker_ship_type_id": intSchema(),
		"victim_ship_type_id":   intSchema(),
		"window_days":           intSchema(),
		"min_sample":            intSchema(),
		"mirror":                boolSchema(),
		"attacker_wins":         intSchema(),
		"victim_wins":           intSchema(),
		"sample":                intSchema(),
		"attacker_win_rate":     numberSchema(),
		"enough":                boolSchema(),
		"top_fits":              arraySchema(fit),
	}, "attacker_ship_type_id", "victim_ship_type_id", "window_days",
		"min_sample", "mirror", "attacker_wins", "victim_wins", "sample",
		"attacker_win_rate", "enough", "top_fits")
}

func entityPageDetailResponseSchema() *huma.Schema {
	return &huma.Schema{OneOf: []*huma.Schema{
		entityCharacterDetailSchema(), entityCorporationDetailSchema(),
		entityAllianceDetailSchema(), entityFactionDetailSchema(),
	}}
}

func entityCharacterDetailSchema() *huma.Schema {
	character := recordSchema(map[string]*huma.Schema{
		"character_id": intSchema(), "name": stringSchema(),
		"description":               nullable(stringSchema()),
		"custom_description":        nullable(stringSchema()),
		"custom_description_format": nullable(stringSchema()),
		"custom_description_html":   nullable(stringSchema()),
		"birthday":                  nullable(timestampSchema()), "gender": nullable(stringSchema()),
		"security_status": numberSchema(), "title": nullable(stringSchema()),
		"last_active":        nullable(timestampSchema()),
		"corporation_id":     nullable(intSchema()),
		"corporation_name":   nullable(stringSchema()),
		"corporation_ticker": nullable(stringSchema()),
		"palette":            nullable(stringSchema()), "alliance_id": nullable(intSchema()),
		"alliance_name":   nullable(stringSchema()),
		"alliance_ticker": nullable(stringSchema()),
		"faction_id":      nullable(intSchema()), "faction_name": nullable(stringSchema()),
		"race_name": nullable(stringSchema()), "bloodline_name": nullable(stringSchema()),
	})
	history := responseSchema(map[string]*huma.Schema{
		"corporation_id": intSchema(), "corporation_name": stringSchema(),
		"corporation_ticker": stringSchema(), "start_date": timestampSchema(),
		"kills": intSchema(), "losses": intSchema(),
	}, "corporation_id", "corporation_name", "corporation_ticker",
		"start_date", "kills", "losses")
	return responseSchema(map[string]*huma.Schema{
		"character": character, "stats": scalarStatsSchema(),
		"recentStats": recentStatsSchema(), "topShips": arraySchema(topShipSchema()),
		"corporationHistoryQueued": boolSchema(),
		"corporationHistory":       arraySchema(history),
	}, "character", "stats", "recentStats", "topShips",
		"corporationHistoryQueued", "corporationHistory")
}

func entityCorporationDetailSchema() *huma.Schema {
	corporation := recordSchema(map[string]*huma.Schema{
		"corporation_id": intSchema(), "name": stringSchema(),
		"ticker": stringSchema(), "description": nullable(stringSchema()),
		"custom_description":        nullable(stringSchema()),
		"custom_description_format": nullable(stringSchema()),
		"custom_description_html":   nullable(stringSchema()),
		"date_founded":              nullable(timestampSchema()), "url": nullable(stringSchema()),
		"member_count": intSchema(), "tax_rate": numberSchema(),
		"lp_tax_rate": nullable(numberSchema()), "war_eligible": boolSchema(),
		"friendly_fire": nullable(boolSchema()), "state": nullable(stringSchema()),
		"type": nullable(stringSchema()), "palette": nullable(stringSchema()),
		"ceo_id": nullable(intSchema()), "ceo_name": nullable(stringSchema()),
		"creator_id": nullable(intSchema()), "creator_name": nullable(stringSchema()),
		"alliance_id": nullable(intSchema()), "alliance_name": nullable(stringSchema()),
		"alliance_ticker": nullable(stringSchema()),
		"faction_id":      nullable(intSchema()), "faction_name": nullable(stringSchema()),
	})
	history := responseSchema(map[string]*huma.Schema{
		"alliance_id": nullable(intSchema()), "alliance_name": nullable(stringSchema()),
		"alliance_ticker": nullable(stringSchema()), "start_date": timestampSchema(),
	}, "alliance_id", "alliance_name", "alliance_ticker", "start_date")
	return responseSchema(map[string]*huma.Schema{
		"corporation": corporation, "stats": scalarStatsSchema(),
		"recentStats": recentStatsSchema(), "allianceHistory": arraySchema(history),
	}, "corporation", "stats", "recentStats", "allianceHistory")
}

func entityAllianceDetailSchema() *huma.Schema {
	alliance := recordSchema(map[string]*huma.Schema{
		"alliance_id": intSchema(), "name": stringSchema(), "ticker": stringSchema(),
		"custom_description":        nullable(stringSchema()),
		"custom_description_format": nullable(stringSchema()),
		"custom_description_html":   nullable(stringSchema()),
		"date_founded":              nullable(timestampSchema()), "corporation_count": intSchema(),
		"member_count": intSchema(), "creator_id": nullable(intSchema()),
		"creator_name":            nullable(stringSchema()),
		"executor_corporation_id": nullable(intSchema()),
		"executor_name":           nullable(stringSchema()),
		"executor_ticker":         nullable(stringSchema()), "palette": nullable(stringSchema()),
		"faction_id": nullable(intSchema()), "faction_name": nullable(stringSchema()),
	})
	return responseSchema(map[string]*huma.Schema{
		"alliance": alliance, "stats": scalarStatsSchema(),
		"recentStats": recentStatsSchema(),
	}, "alliance", "stats", "recentStats")
}

func entityFactionDetailSchema() *huma.Schema {
	faction := recordSchema(map[string]*huma.Schema{
		"faction_id": intSchema(), "name": stringSchema(),
		"description":            nullable(stringSchema()),
		"corporation_id":         nullable(intSchema()),
		"militia_corporation_id": nullable(intSchema()),
		"solar_system_id":        nullable(intSchema()),
		"station_count":          intSchema(), "station_system_count": intSchema(),
	})
	stats := responseSchema(map[string]*huma.Schema{
		"losses": intSchema(), "isk_lost": numberSchema(),
	}, "losses", "isk_lost")
	return responseSchema(map[string]*huma.Schema{
		"faction": faction, "stats": stats, "recentStats": stats,
	}, "faction", "stats", "recentStats")
}

func entityDashboardStatsSchema() *huma.Schema {
	topShip := responseSchema(map[string]*huma.Schema{
		"ship_type_id": intSchema(), "ship_name": stringSchema(),
		"count": intSchema(),
	}, "ship_type_id", "ship_name", "count")
	topEntity := responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "name": stringSchema(), "count": intSchema(),
		"isk_value": numberSchema(),
	}, "id", "name", "count", "isk_value")
	schema := responseSchema(map[string]*huma.Schema{
		"stats":              scalarStatsSchema(),
		"topShipsUsed":       arraySchema(topShip),
		"topShipsLost":       arraySchema(topShip),
		"diesToCorporations": arraySchema(topEntity),
		"diesToAlliances":    arraySchema(topEntity),
		"heatMap":            mapOfSchema(intSchema()),
		"activity": responseSchema(map[string]*huma.Schema{
			"kills":  arraySchema(arraySchema(intSchema())),
			"losses": arraySchema(arraySchema(intSchema())),
		}, "kills", "losses"),
		"fliesWithCorporations": arraySchema(topEntity),
		"fliesWithAlliances":    arraySchema(topEntity),
	}, "stats", "topShipsUsed", "topShipsLost", "diesToCorporations",
		"diesToAlliances")
	return schema
}

func organizationIntelSchema() *huma.Schema {
	peer := responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "name": stringSchema(),
		"shared_enemy_kills": intSchema(), "mutual_kills": intSchema(),
		"kills_given": intSchema(), "kills_taken": intSchema(), "total": intSchema(),
	})
	hunting := responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "name": stringSchema(), "active_characters": intSchema(),
	}, "id", "name", "active_characters")
	census := responseSchema(map[string]*huma.Schema{
		"total": intSchema(), "fcs": intSchema(), "logis": intSchema(),
		"caps": intSchema(), "supers": intSchema(), "droppers": intSchema(),
		"corps": arraySchema(recordSchema(map[string]*huma.Schema{
			"id": intSchema(), "name": stringSchema(), "total": intSchema(),
		})),
	}, "total", "fcs", "logis", "caps", "supers", "droppers")
	change := responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "name": stringSchema(),
		"current_corp": nullable(responseSchema(map[string]*huma.Schema{
			"id": intSchema(), "name": nullable(stringSchema()),
		}, "id", "name")),
		"previous_corp": nullable(responseSchema(map[string]*huma.Schema{
			"id": intSchema(), "name": nullable(stringSchema()),
		}, "id", "name")),
		"left_at": nullable(timestampSchema()), "joined_at": nullable(timestampSchema()),
	}, "id", "name")
	return responseSchema(map[string]*huma.Schema{
		"allies": arraySchema(peer), "enemies": arraySchema(peer),
		"huntingGrounds": arraySchema(hunting), "census": census,
		"recentDepartures": arraySchema(change), "recentJoins": arraySchema(change),
		"activeMembers": responseSchema(map[string]*huma.Schema{
			"days_7": intSchema(), "days_30": intSchema(), "days_90": intSchema(),
		}, "days_7", "days_30", "days_90"),
	}, "allies", "enemies", "huntingGrounds", "census",
		"recentDepartures", "recentJoins", "activeMembers")
}

func entityPageIntelSchema() *huma.Schema {
	return &huma.Schema{OneOf: []*huma.Schema{
		characterIntelSchema(), organizationIntelSchema(),
	}}
}

func entityAchievementSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"achievement_id": stringSchema(), "current_count": intSchema(),
		"threshold": intSchema(), "completion_tiers": intSchema(),
		"is_completed": boolSchema(), "points": intSchema(),
		"completed_at": nullable(timestampSchema()),
		"name":         stringSchema(), "description": stringSchema(),
		"category": stringSchema(), "rarity": stringSchema(), "type": stringSchema(),
		"points_modifier": stringSchema(), "level": intSchema(), "max_level": intSchema(),
		"level_thresholds": arraySchema(intSchema()), "next_threshold": intSchema(),
	}, "achievement_id", "current_count", "threshold", "completion_tiers",
		"is_completed", "points", "completed_at", "name", "description",
		"category", "rarity", "type", "points_modifier", "level", "max_level",
		"level_thresholds", "next_threshold")
}

func entityPageMembersSchema() *huma.Schema {
	member := responseSchema(map[string]*huma.Schema{
		"character_id": intSchema(), "name": stringSchema(),
		"security_status": numberSchema(), "last_active": nullable(timestampSchema()),
		"kills_90d": intSchema(), "losses_90d": intSchema(),
		"is_fc": boolSchema(), "is_logi": boolSchema(),
		"is_capital_pilot": boolSchema(), "corporation_id": nullable(intSchema()),
	}, "character_id", "name", "security_status", "last_active",
		"kills_90d", "losses_90d", "is_fc", "is_logi", "is_capital_pilot")
	return responseSchema(map[string]*huma.Schema{
		"members": arraySchema(member), "total": intSchema(),
		"page": intSchema(), "limit": intSchema(),
	}, "members", "total", "page", "limit")
}

func entityPageCorporationsSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"corporations": arraySchema(responseSchema(map[string]*huma.Schema{
			"corporation_id": intSchema(), "name": stringSchema(),
			"ticker": stringSchema(), "member_count": intSchema(),
			"palette": nullable(stringSchema()),
		}, "corporation_id", "name", "ticker", "member_count", "palette")),
		"total": intSchema(),
	}, "corporations", "total")
}

func entityPageTopListsSchema() *huma.Schema {
	item := responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "name": stringSchema(), "count": intSchema(),
		"isk_value": numberSchema(), "palette": nullable(stringSchema()),
	}, "id", "name", "count", "isk_value", "palette")
	group := responseSchema(map[string]*huma.Schema{
		"characters": arraySchema(item), "corporations": arraySchema(item),
		"alliances": arraySchema(item),
	}, "characters", "corporations", "alliances")
	return responseSchema(map[string]*huma.Schema{
		"killed": group, "killedBy": group,
	}, "killed", "killedBy")
}

func entityLegacyTopSchema() *huma.Schema {
	item := responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "name": stringSchema(), "count": numberSchema(),
		"palette": nullable(stringSchema()),
	}, "id", "name", "count")
	properties := map[string]*huma.Schema{}
	for _, key := range []string{
		"charactersByKills", "charactersByPoints", "charactersByIsk",
		"soloKillers", "corporationsByKills", "shipsUsed", "systems",
		"constellations", "regions", "killedCorporations", "killedAlliances",
		"killedByCorporations", "killedByAlliances", "achievementPoints",
		"recentMembers",
	} {
		properties[key] = arraySchema(item)
	}
	return responseSchema(properties)
}

func mapRegionSummarySchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"region_id": intSchema(), "name": stringSchema(), "system_count": intSchema(),
	}, "region_id", "name")
}

func mapRegionsResponseSchema() *huma.Schema {
	properties := map[string]*huma.Schema{}
	for _, key := range []string{
		"kspace", "pochven", "zarzakh", "wormhole", "abyssal", "proving",
	} {
		properties[key] = arraySchema(mapRegionSummarySchema())
	}
	return responseSchema(properties, "kspace", "pochven", "zarzakh",
		"wormhole", "abyssal", "proving")
}

func mapSystemSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"solar_system_id": intSchema(), "system_name": stringSchema(),
		"x": numberSchema(), "y": numberSchema(), "z": numberSchema(),
		"x2d": numberSchema(), "z2d": numberSchema(), "security": numberSchema(),
		"region_id": intSchema(), "constellation_id": intSchema(),
		"distance": intSchema(), "is_anchor": boolSchema(),
	}, "solar_system_id", "system_name", "x", "y", "z", "x2d", "z2d",
		"security", "region_id", "constellation_id")
}

func mapJumpSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"from_solar_system_id": intSchema(), "to_solar_system_id": intSchema(),
	}, "from_solar_system_id", "to_solar_system_id")
}

func mapExternalJumpSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"internal_system_id": intSchema(), "external_system_id": intSchema(),
		"external_system_name": stringSchema(), "external_region_id": intSchema(),
		"external_region_name": stringSchema(), "external_security": numberSchema(),
		"external_x": numberSchema(), "external_z": numberSchema(),
		"external_x2d": numberSchema(), "external_z2d": numberSchema(),
	}, "internal_system_id", "external_system_id", "external_system_name",
		"external_region_id", "external_region_name", "external_security",
		"external_x", "external_z", "external_x2d", "external_z2d")
}

func mapActivitySchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"system_id": intSchema(), "ship_kills": intSchema(),
		"npc_kills": intSchema(), "pod_kills": intSchema(),
		"ship_jumps": intSchema(),
	}, "system_id", "ship_kills", "npc_kills", "pod_kills", "ship_jumps")
}

func mapSovereigntyClaimSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"system_id": intSchema(), "alliance_id": intSchema(),
		"date_added": timestampSchema(), "alliance_name": stringSchema(),
		"alliance_ticker": stringSchema(), "member_count": intSchema(),
	}, "system_id", "alliance_id", "date_added", "alliance_name",
		"alliance_ticker", "member_count")
}

func mapSovereigntyChangeSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"system_id": intSchema(), "alliance_id": nullable(intSchema()),
		"date_added": timestampSchema(), "alliance_name": nullable(stringSchema()),
		"alliance_ticker": nullable(stringSchema()),
	}, "system_id", "alliance_id", "date_added", "alliance_name",
		"alliance_ticker")
}

func mapSovereigntyResponseSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"scope":          stringSchema(),
		"activity_hours": intSchema(),
		"snapshot_at":    timestampSchema(),
		"regions":        arraySchema(mapRegionSummarySchema()),
		"systems":        arraySchema(mapSystemSchema()),
		"jumps":          arraySchema(mapJumpSchema()),
		"externalJumps":  arraySchema(mapExternalJumpSchema()),
		"activity":       arraySchema(mapActivitySchema()),
		"sovereignty":    arraySchema(mapSovereigntyClaimSchema()),
		"changes":        arraySchema(mapSovereigntyChangeSchema()),
	}, "scope", "activity_hours", "snapshot_at", "regions", "systems",
		"jumps", "externalJumps", "activity", "sovereignty", "changes")
}

func mapScopeResponseSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"scope":          stringSchema(),
		"activity_hours": intSchema(),
		"regions":        arraySchema(mapRegionSummarySchema()),
		"systems":        arraySchema(mapSystemSchema()),
		"jumps":          arraySchema(mapJumpSchema()),
		"externalJumps":  arraySchema(mapExternalJumpSchema()),
		"activity":       arraySchema(mapActivitySchema()),
	}, "scope", "activity_hours", "regions", "systems", "jumps", "externalJumps", "activity")
}

func mapAIIDResponseSchema() *huma.Schema {
	anchorSchema := responseSchema(map[string]*huma.Schema{
		"solar_system_id": intSchema(), "system_name": stringSchema(),
		"region_id": intSchema(),
	}, "solar_system_id", "system_name", "region_id")
	return responseSchema(map[string]*huma.Schema{
		"scope":          stringSchema(),
		"activity_hours": intSchema(),
		"jump_radius":    intSchema(),
		"anchors":        arraySchema(anchorSchema),
		"regions":        arraySchema(mapRegionSummarySchema()),
		"systems":        arraySchema(mapSystemSchema()),
		"jumps":          arraySchema(mapJumpSchema()),
		"externalJumps":  arraySchema(mapExternalJumpSchema()),
		"activity":       arraySchema(mapActivitySchema()),
		"kills":          arraySchema(killlistRowSchema()),
	}, "scope", "activity_hours", "jump_radius", "anchors", "regions",
		"systems", "jumps", "externalJumps", "activity", "kills")
}

func mapRegionResponseSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"region":         mapRegionSummarySchema(),
		"activity_hours": intSchema(),
		"systems":        arraySchema(mapSystemSchema()),
		"constellations": arraySchema(responseSchema(map[string]*huma.Schema{
			"constellation_id": intSchema(), "constellation_name": stringSchema(),
		}, "constellation_id", "constellation_name")),
		"jumps":         arraySchema(mapJumpSchema()),
		"externalJumps": arraySchema(mapExternalJumpSchema()),
		"activity":      arraySchema(mapActivitySchema()),
		"celestials": arraySchema(responseSchema(map[string]*huma.Schema{
			"system_id": intSchema(), "group_id": intSchema(),
			"x": numberSchema(), "z": numberSchema(),
		}, "system_id", "group_id", "x", "z")),
	}, "region", "activity_hours", "systems", "constellations", "jumps", "externalJumps",
		"activity", "celestials")
}

func statsRankingEntrySchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"id": intSchema(), "name": stringSchema(), "type": stringSchema(),
		"member_count": intSchema(), "delta_1d": intSchema(),
		"delta_7d": intSchema(), "delta_30d": intSchema(),
		"security_status": numberSchema(), "weighted_score": numberSchema(),
		"growth": numberSchema(), "date_founded": timestampSchema(),
		"achievement_points": intSchema(), "completed_count": intSchema(),
	})
}

func graphResponseSchema() *huma.Schema {
	mode := func(properties map[string]*huma.Schema, required ...string) *huma.Schema {
		properties["mode"] = stringSchema()
		return responseSchema(properties, append(required, "mode")...)
	}
	entity := responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "name": stringSchema(),
		"corp_name":     nullable(stringSchema()),
		"alliance_name": nullable(stringSchema()),
	}, "id", "name")
	path := mode(map[string]*huma.Schema{
		"path": nullable(responseSchema(map[string]*huma.Schema{
			"nodes": arraySchema(entity),
			"edges": arraySchema(responseSchema(map[string]*huma.Schema{
				"weight": intSchema(),
			}, "weight")),
			"hops": intSchema(),
		}, "nodes", "edges", "hops")),
		"error": stringSchema(),
	}, "path")
	coalitions := mode(map[string]*huma.Schema{
		"coalitions": arraySchema(responseSchema(map[string]*huma.Schema{
			"id": intSchema(),
			"alliances": arraySchema(responseSchema(map[string]*huma.Schema{
				"id": intSchema(), "name": stringSchema(), "connections": intSchema(),
			}, "id", "name")),
		}, "id", "alliances")),
	}, "coalitions")
	rivalries := mode(map[string]*huma.Schema{
		"items": arraySchema(responseSchema(map[string]*huma.Schema{
			"entity_a": entity, "entity_b": entity,
			"mutual_kills": intSchema(), "total_isk": numberSchema(),
		}, "entity_a", "entity_b", "mutual_kills", "total_isk")),
		"entityType": stringSchema(),
	}, "items", "entityType")
	intel := mode(map[string]*huma.Schema{
		"allies": arraySchema(recordSchema(map[string]*huma.Schema{
			"id": intSchema(), "name": stringSchema(),
		})),
		"enemies": arraySchema(recordSchema(map[string]*huma.Schema{
			"id": intSchema(), "name": stringSchema(),
		})),
		"entityType": stringSchema(),
	}, "allies", "enemies")
	systems := mode(map[string]*huma.Schema{
		"systems": arraySchema(recordSchema(map[string]*huma.Schema{
			"id": intSchema(), "name": stringSchema(),
			"active_characters": intSchema(), "latest_activity": timestampSchema(),
			"alliances": intSchema(), "characters": intSchema(),
		})),
		"entityType": stringSchema(),
	}, "systems")
	migration := mode(map[string]*huma.Schema{
		"departed": arraySchema(recordSchema(map[string]*huma.Schema{
			"id": intSchema(), "name": stringSchema(), "left_at": timestampSchema(),
		})),
		"joined": arraySchema(recordSchema(map[string]*huma.Schema{
			"id": intSchema(), "name": stringSchema(), "joined_at": timestampSchema(),
		})),
		"corp_name": nullable(stringSchema()),
	}, "departed", "joined")
	spies := mode(map[string]*huma.Schema{
		"suspects": arraySchema(recordSchema(map[string]*huma.Schema{
			"id": intSchema(), "name": stringSchema(), "total_flights": intSchema(),
		})),
		"entityType": stringSchema(),
	}, "suspects")
	census := mode(map[string]*huma.Schema{
		"corps": arraySchema(recordSchema(map[string]*huma.Schema{
			"id": intSchema(), "name": stringSchema(), "total": intSchema(),
			"fcs": intSchema(), "logis": intSchema(), "caps": intSchema(),
			"supers": intSchema(), "droppers": intSchema(),
		})),
		"totals": recordSchema(map[string]*huma.Schema{
			"total": intSchema(), "fcs": intSchema(), "logis": intSchema(),
			"caps": intSchema(), "supers": intSchema(), "droppers": intSchema(),
		}),
	}, "corps", "totals")
	return &huma.Schema{OneOf: []*huma.Schema{
		path, coalitions, rivalries, intel, systems, migration, spies, census,
	}}
}

func fittingItemSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"slot_group":     intSchema(),
		"ordinal":        intSchema(),
		"type_id":        intSchema(),
		"state":          intSchema(),
		"charge_type_id": nullable(intSchema()),
		"quantity":       intSchema(),
	}, "slot_group", "ordinal", "type_id", "state", "charge_type_id",
		"quantity")
}

func savedFittingSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"fit_id":             stringSchema(),
		"owner_character_id": intSchema(),
		"ship_type_id":       intSchema(),
		"name":               stringSchema(),
		"description":        nullable(stringSchema()),
		"visibility":         intSchema(),
		"rating_avg":         nullable(numberSchema()),
		"rating_count":       intSchema(),
		"created_at":         timestampSchema(),
		"updated_at":         timestampSchema(),
	})
}

func fittingCreateResponseSchema() *huma.Schema {
	schema := savedFittingSchema()
	schema.Properties["items"] = arraySchema(fittingItemSchema())
	schema.Required = []string{
		"fit_id", "owner_character_id", "ship_type_id", "name",
		"description", "visibility", "items",
	}
	return schema
}

func fittingDetailResponseSchema(includeViewerRating bool) *huma.Schema {
	properties := map[string]*huma.Schema{
		"fit":   savedFittingSchema(),
		"items": arraySchema(fittingItemSchema()),
	}
	required := []string{"fit", "items"}
	if includeViewerRating {
		properties["viewer_rating"] = nullable(intSchema())
		required = append(required, "viewer_rating")
	}
	return responseSchema(properties, required...)
}

func fittingRatingAggregateSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"rating_avg": nullable(numberSchema()), "rating_count": intSchema(),
	}, "rating_avg", "rating_count")
}

func communityFittingSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"fit_id": stringSchema(), "name": stringSchema(),
		"description": nullable(stringSchema()), "ship_type_id": intSchema(),
		"ship_name": nullable(stringSchema()), "owner_character_id": intSchema(),
		"owner_name": nullable(stringSchema()), "rating_avg": nullable(numberSchema()),
		"rating_count": intSchema(), "created_at": timestampSchema(),
		"updated_at": timestampSchema(), "module_count": intSchema(),
	}, "fit_id", "name", "description", "ship_type_id", "ship_name",
		"owner_character_id", "owner_name", "rating_avg", "rating_count",
		"created_at", "updated_at", "module_count")
}

func fittingCatalogueModuleSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"slot_group": intSchema(), "ordinal": intSchema(), "type_id": intSchema(),
		"name": nullable(stringSchema()), "charge_type_id": nullable(intSchema()),
		"charge_name": nullable(stringSchema()),
	}, "slot_group", "ordinal", "type_id", "name", "charge_type_id",
		"charge_name")
}

func fittingCatalogueDroneSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"type_id": intSchema(), "name": nullable(stringSchema()),
		"quantity": intSchema(),
	}, "type_id", "name", "quantity")
}

func fittingFamilyContextSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"security_distribution": arraySchema(responseSchema(map[string]*huma.Schema{
			"name": stringSchema(), "count": intSchema(), "pct": numberSchema(),
		}, "name", "count", "pct")),
		"top_region": nullable(responseSchema(map[string]*huma.Schema{
			"region_id": intSchema(), "name": nullable(stringSchema()),
			"count": intSchema(), "pct": numberSchema(),
		}, "region_id", "name", "count", "pct")),
		"median_attackers":  numberSchema(),
		"median_loss_value": numberSchema(),
	}, "security_distribution", "top_region", "median_attackers", "median_loss_value")
}

func fittingFamilySchema(includeShip, includeContext bool) *huma.Schema {
	properties := map[string]*huma.Schema{
		"family_hash":        stringSchema(),
		"canonical_fit_hash": stringSchema(),
		"total_uses":         intSchema(),
		"canonical_uses":     intSchema(),
		"variant_count":      intSchema(),
		"last_used":          timestampSchema(),
		"fit_cost":           numberSchema(),
		"hull_cost":          nullable(numberSchema()),
		"modules":            arraySchema(fittingCatalogueModuleSchema()),
		"drones":             arraySchema(fittingCatalogueDroneSchema()),
		"top_alliances": arraySchema(responseSchema(map[string]*huma.Schema{
			"alliance_id": nullable(intSchema()), "name": nullable(stringSchema()),
			"uses": intSchema(), "pct_of_alliance_losses": numberSchema(),
		}, "alliance_id", "name", "uses", "pct_of_alliance_losses")),
	}
	required := []string{
		"family_hash", "canonical_fit_hash", "total_uses", "canonical_uses",
		"variant_count", "last_used", "fit_cost", "modules", "drones",
	}
	if includeContext {
		properties["context"] = fittingFamilyContextSchema()
		properties["stats"] = responseSchema(map[string]*huma.Schema{
			"ehp": nullable(numberSchema()), "dps": nullable(numberSchema()),
			"alpha": nullable(numberSchema()), "speed": nullable(numberSchema()),
			"align": nullable(numberSchema()), "repair": nullable(numberSchema()),
			"npc_profile": stringSchema(), "npc_ehp": nullable(numberSchema()),
		}, "ehp", "dps", "alpha", "speed", "align", "repair", "npc_profile", "npc_ehp")
		required = append(required, "context", "stats")
	}
	if includeShip {
		properties["ship_type_id"] = intSchema()
		properties["ship_name"] = nullable(stringSchema())
		properties["ranking_count"] = intSchema()
		required = append(required, "ship_type_id", "ship_name", "hull_cost", "ranking_count")
	}
	return responseSchema(properties, required...)
}

func popularFittingShipSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"ship_type_id": intSchema(), "ship_name": nullable(stringSchema()),
		"group_id": nullable(intSchema()), "total_uses": intSchema(),
		"fit_count": intSchema(), "last_used": timestampSchema(),
	}, "ship_type_id", "ship_name", "group_id", "total_uses", "fit_count",
		"last_used")
}

func fittingRoleSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"id": stringSchema(), "label": stringSchema(), "icon": stringSchema(),
		"category": stringSchema(), "typeCount": intSchema(),
		"description": stringSchema(),
	}, "id", "label", "icon", "category", "typeCount")
}

func fittingSearchFilterSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"kind": stringSchema(), "op": stringSchema(), "count": intSchema(),
		"role_id": stringSchema(), "type_id": intSchema(),
		"type_name": stringSchema(),
	}, "kind", "op", "count")
}

func fittingStatsSchema() *huma.Schema {
	fields := map[string]*huma.Schema{
		"skill_level": intSchema(), "dps_with_reload": nullable(numberSchema()),
		"dps_without_reload": nullable(numberSchema()), "alpha": nullable(numberSchema()),
		"ehp": nullable(numberSchema()), "shield_ehp": nullable(numberSchema()),
		"armor_ehp": nullable(numberSchema()), "hull_ehp": nullable(numberSchema()),
		"shield_boost": nullable(numberSchema()), "shield_effective_boost": nullable(numberSchema()),
		"armor_repair": nullable(numberSchema()), "armor_effective_repair": nullable(numberSchema()),
		"hull_repair": nullable(numberSchema()), "hull_effective_repair": nullable(numberSchema()),
		"passive_shield": nullable(numberSchema()), "passive_shield_effective": nullable(numberSchema()),
		"remote_shield": nullable(numberSchema()), "remote_armor": nullable(numberSchema()),
		"remote_hull": nullable(numberSchema()), "remote_cap": nullable(numberSchema()),
		"neut": nullable(numberSchema()), "nos": nullable(numberSchema()),
		"cap_stable": boolSchema(), "cap_depletes_in": nullable(numberSchema()),
		"cap_capacity": nullable(numberSchema()), "cap_peak_delta": nullable(numberSchema()),
		"max_velocity": nullable(numberSchema()), "align_time": nullable(numberSchema()),
		"signature_radius": nullable(numberSchema()), "max_target_range": nullable(numberSchema()),
		"scan_resolution": nullable(numberSchema()), "engine_version": stringSchema(),
		"sde_version": stringSchema(), "calculated_at": timestampSchema(),
	}
	return responseSchema(fields)
}

func fittingSearchFitSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"fit_hash": stringSchema(), "family_hash": stringSchema(),
		"ship_type_id": intSchema(), "ship_name": nullable(stringSchema()),
		"total_uses": intSchema(), "last_used": timestampSchema(),
		"family_total_uses": intSchema(), "variant_count": intSchema(),
		"fit_cost": numberSchema(), "hull_cost": nullable(numberSchema()),
		"modules": arraySchema(fittingCatalogueModuleSchema()),
		"drones":  arraySchema(fittingCatalogueDroneSchema()),
		"context": fittingFamilyContextSchema(),
		"stats":   nullable(fittingStatsSchema()),
	}, "fit_hash", "family_hash", "ship_type_id", "ship_name", "total_uses",
		"last_used", "family_total_uses", "variant_count", "fit_cost",
		"hull_cost", "modules", "drones", "context")
}

func fittingSearchResponseSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"ship_type_id":    intSchema(),
		"ship_name":       nullable(stringSchema()),
		"window_days":     intSchema(),
		"total":           intSchema(),
		"has_more":        boolSchema(),
		"offset":          intSchema(),
		"limit":           intSchema(),
		"filters_applied": arraySchema(fittingSearchFilterSchema()),
		"fits":            arraySchema(fittingSearchFitSchema()),
	}, "ship_type_id", "window_days", "total", "has_more", "offset", "limit",
		"filters_applied", "fits")
}

func fittingDoctrineSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"entity_id": intSchema(), "entity_name": nullable(stringSchema()),
		"total_losses": intSchema(), "family_hash": stringSchema(),
		"ship_type_id": intSchema(), "ship_name": nullable(stringSchema()),
		"canonical_fit_hash": stringSchema(), "doctrine_uses": intSchema(),
		"doctrine_share": numberSchema(), "last_used": timestampSchema(),
		"fit_cost": numberSchema(), "hull_cost": nullable(numberSchema()),
		"modules": arraySchema(fittingCatalogueModuleSchema()),
		"drones":  arraySchema(fittingCatalogueDroneSchema()),
	}, "entity_id", "entity_name", "total_losses", "family_hash",
		"ship_type_id", "ship_name", "canonical_fit_hash", "doctrine_uses",
		"doctrine_share", "last_used", "fit_cost", "hull_cost", "modules",
		"drones")
}

func fittingShipMetadataSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"ship_type_id": intSchema(), "window_days": intSchema(),
		"total_kills": intSchema(),
		"groups": arraySchema(responseSchema(map[string]*huma.Schema{
			"group_id": intSchema(), "name": stringSchema(),
			"kill_count": intSchema(), "pct": numberSchema(),
		}, "group_id", "name", "kill_count", "pct")),
	}, "ship_type_id", "window_days", "total_kills", "groups")
}

func battleGeneratorEntitiesSchema() *huma.Schema {
	entity := responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "name": stringSchema(), "type": stringSchema(),
		"alliance_id":   nullable(intSchema()),
		"alliance_name": nullable(stringSchema()),
	}, "id", "name", "type")
	return responseSchema(map[string]*huma.Schema{
		"alliances":    arraySchema(entity),
		"corporations": arraySchema(entity),
		"killCount":    intSchema(),
	}, "alliances", "corporations", "killCount")
}

func battleTeamSchema() *huma.Schema {
	corporation := responseSchema(map[string]*huma.Schema{
		"corporation_id": intSchema(), "corporation_name": nullable(stringSchema()),
		"kills": intSchema(), "losses": intSchema(),
		"isk_destroyed": numberSchema(), "isk_lost": numberSchema(),
	}, "corporation_id", "corporation_name", "kills", "losses",
		"isk_destroyed", "isk_lost")
	alliance := responseSchema(map[string]*huma.Schema{
		"alliance_id": nullable(intSchema()), "alliance_name": nullable(stringSchema()),
		"kills": intSchema(), "losses": intSchema(),
		"isk_destroyed": numberSchema(), "isk_lost": numberSchema(),
		"corporations": arraySchema(corporation),
	}, "alliance_id", "alliance_name", "kills", "losses",
		"isk_destroyed", "isk_lost", "corporations")
	return responseSchema(map[string]*huma.Schema{
		"team_index": intSchema(), "total_kills": intSchema(),
		"total_losses": intSchema(), "total_isk_destroyed": numberSchema(),
		"total_isk_lost": numberSchema(), "alliance_count": intSchema(),
		"corp_count": intSchema(), "alliances": arraySchema(alliance),
		"dominant_corp_palette": nullable(stringSchema()),
	}, "team_index", "total_kills", "total_losses", "total_isk_destroyed",
		"total_isk_lost", "alliance_count", "corp_count", "alliances",
		"dominant_corp_palette")
}

func battleDetailFrontendSchema() *huma.Schema {
	teamEntities := responseSchema(map[string]*huma.Schema{
		"corps": arraySchema(intSchema()), "alliances": arraySchema(intSchema()),
	}, "corps", "alliances")
	unsided := responseSchema(map[string]*huma.Schema{
		"kills": intSchema(), "losses": intSchema(),
		"isk_destroyed": numberSchema(), "isk_lost": numberSchema(),
		"alliance_count": intSchema(), "corp_count": intSchema(),
		"alliances": battleTeamSchema().Properties["alliances"],
	}, "kills", "losses", "isk_destroyed", "isk_lost", "alliance_count",
		"corp_count", "alliances")
	return responseSchema(map[string]*huma.Schema{
		"battle_id":             nullable(intSchema()),
		"solar_system_id":       intSchema(),
		"solar_system_name":     nullable(stringSchema()),
		"solar_system_security": nullable(numberSchema()),
		"region_id":             nullable(intSchema()), "region_name": nullable(stringSchema()),
		"start_time": timestampSchema(), "end_time": timestampSchema(),
		"duration_minutes": intSchema(), "kill_count": intSchema(),
		"total_isk_destroyed": numberSchema(), "is_multi_party": boolSchema(),
		"is_custom": boolSchema(), "characters_involved": intSchema(),
		"corporations_involved": intSchema(), "alliances_involved": intSchema(),
		"total_damage": intSchema(), "teams": arraySchema(battleTeamSchema()),
		"team_entities": arraySchema(teamEntities),
		"unsided":       nullable(unsided), "killmail_id": intSchema(),
	}, "solar_system_id", "solar_system_name", "solar_system_security",
		"region_id", "region_name", "start_time", "end_time",
		"duration_minutes", "kill_count", "total_isk_destroyed",
		"is_multi_party", "is_custom", "characters_involved",
		"corporations_involved", "alliances_involved", "total_damage",
		"teams", "team_entities")
}

func battleMostValuableKillSchema() *huma.Schema {
	schema := mostValuableKillSchema()
	schema.Properties["killmail_time"] = timestampSchema()
	return schema
}

func battleCompositionSchema() *huma.Schema {
	pilot := responseSchema(map[string]*huma.Schema{
		"character_id": intSchema(), "character_name": nullable(stringSchema()),
		"corporation_id":   nullable(intSchema()),
		"corporation_name": nullable(stringSchema()),
		"alliance_id":      nullable(intSchema()), "alliance_name": nullable(stringSchema()),
		"ship_type_id": intSchema(), "ship_name": nullable(stringSchema()),
		"ship_group_id": nullable(intSchema()), "ship_group_name": nullable(stringSchema()),
		"isk_lost": numberSchema(), "damage_done": intSchema(),
		"damage_taken": intSchema(), "deaths": intSchema(),
		"team_index": intSchema(), "rank": intSchema(),
	}, "character_id", "character_name", "corporation_id",
		"corporation_name", "alliance_id", "alliance_name", "ship_type_id",
		"ship_name", "ship_group_id", "ship_group_name", "isk_lost",
		"damage_done", "damage_taken", "deaths", "team_index", "rank")
	aggregate := responseSchema(map[string]*huma.Schema{
		"key": stringSchema(), "name": nullable(stringSchema()),
		"ship_type_id": intSchema(), "ship_group_id": intSchema(),
		"count": intSchema(), "losses": intSchema(), "isk_lost": numberSchema(),
		"damage_done": intSchema(), "damage_taken": intSchema(), "rank": intSchema(),
	}, "key", "name", "count", "losses", "isk_lost", "damage_done",
		"damage_taken", "rank")
	team := responseSchema(map[string]*huma.Schema{
		"team_index": intSchema(), "individuals": arraySchema(pilot),
		"by_ship": arraySchema(aggregate), "by_group": arraySchema(aggregate),
	}, "team_index", "individuals", "by_ship", "by_group")
	return responseSchema(map[string]*huma.Schema{
		"teams": arraySchema(team), "team_count": intSchema(),
	}, "teams", "team_count")
}

func battleIntelSchema() *huma.Schema {
	pilot := responseSchema(map[string]*huma.Schema{
		"character_id": intSchema(), "character_name": nullable(stringSchema()),
		"corporation_id":   nullable(intSchema()),
		"corporation_name": nullable(stringSchema()),
		"alliance_id":      nullable(intSchema()), "alliance_name": nullable(stringSchema()),
		"ship_type_id": intSchema(), "ship_name": nullable(stringSchema()),
		"ship_group_id": nullable(intSchema()), "ship_group_name": nullable(stringSchema()),
		"damage_done": intSchema(), "died": boolSchema(), "confirmed": boolSchema(),
	}, "character_id", "character_name", "corporation_id",
		"corporation_name", "alliance_id", "alliance_name", "ship_type_id",
		"ship_name", "ship_group_id", "ship_group_name", "damage_done", "died")
	team := responseSchema(map[string]*huma.Schema{
		"team_index": intSchema(), "fcs": arraySchema(pilot),
		"logistics": arraySchema(pilot), "capitals": arraySchema(pilot),
	}, "team_index", "fcs", "logistics", "capitals")
	return responseSchema(map[string]*huma.Schema{
		"teams": arraySchema(team),
	}, "teams")
}

func conflictBattlesSchema() *huma.Schema {
	battle := recordSchema(map[string]*huma.Schema{
		"battle_id": intSchema(), "solar_system_id": intSchema(),
		"solar_system_name":     nullable(stringSchema()),
		"solar_system_security": nullable(numberSchema()),
		"region_id":             nullable(intSchema()), "region_name": nullable(stringSchema()),
		"start_time": timestampSchema(), "end_time": timestampSchema(),
		"kill_count": intSchema(), "total_value": numberSchema(),
		"is_custom": boolSchema(),
	})
	year := responseSchema(map[string]*huma.Schema{
		"year": intSchema(), "count": intSchema(),
	}, "year", "count")
	return responseSchema(map[string]*huma.Schema{
		"battles": arraySchema(battle), "years": arraySchema(year),
		"page": intSchema(), "limit": intSchema(),
	}, "battles", "years", "page", "limit")
}

func conflictWarEntitySchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "name": stringSchema(), "ticker": stringSchema(),
		"type": stringSchema(),
	}, "id", "type")
}

func conflictWarSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"war_id": intSchema(), "declared": timestampSchema(),
		"started": nullable(timestampSchema()), "finished": nullable(timestampSchema()),
		"retracted": nullable(timestampSchema()), "mutual": boolSchema(),
		"open_for_allies": boolSchema(), "aggressor": conflictWarEntitySchema(),
		"defender": conflictWarEntitySchema(),
	})
}

func conflictWarsSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"wars": arraySchema(conflictWarSchema()),
		"page": intSchema(), "limit": intSchema(),
	}, "wars", "page", "limit")
}

func warOverviewStatsSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"activeWars": intSchema(), "finishedWars": intSchema(),
		"upcomingWars": intSchema(), "eligibleCorps": intSchema(),
		"eligibleAlliances": intSchema(),
	}, "activeWars", "finishedWars", "upcomingWars", "eligibleCorps",
		"eligibleAlliances")
}

func warEligibleSchema() *huma.Schema {
	entry := recordSchema(map[string]*huma.Schema{
		"id": intSchema(), "name": stringSchema(), "ticker": stringSchema(),
		"member_count": intSchema(), "corporation_count": intSchema(),
		"alliance_id": nullable(intSchema()), "alliance_name": nullable(stringSchema()),
		"kills": intSchema(), "losses": intSchema(),
		"isk_destroyed": numberSchema(), "isk_lost": numberSchema(),
	})
	return responseSchema(map[string]*huma.Schema{
		"entries": arraySchema(entry), "type": stringSchema(), "limit": intSchema(),
	}, "entries", "type", "limit")
}

func conflictTopShipSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"ship_type_id": intSchema(), "ship_name": stringSchema(), "count": intSchema(),
	}, "ship_type_id", "ship_name", "count")
}

func warLeaderboardBoardSchema() *huma.Schema {
	entry := responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "name": stringSchema(), "count": intSchema(),
		"isk_value": numberSchema(),
	}, "id", "name", "count", "isk_value")
	return responseSchema(map[string]*huma.Schema{
		"topCharacters":   arraySchema(entry),
		"topCorporations": arraySchema(entry),
		"topAlliances":    arraySchema(entry),
	}, "topCharacters", "topCorporations", "topAlliances")
}

func warLeaderboardsSchema() *huma.Schema {
	side := responseSchema(map[string]*huma.Schema{
		"corporations": arraySchema(intSchema()),
		"alliances":    arraySchema(intSchema()),
	}, "corporations", "alliances")
	return responseSchema(map[string]*huma.Schema{
		"combined":  warLeaderboardBoardSchema(),
		"aggressor": warLeaderboardBoardSchema(),
		"defender":  warLeaderboardBoardSchema(),
		"topShips":  arraySchema(conflictTopShipSchema()),
		"sides": responseSchema(map[string]*huma.Schema{
			"aggressor": side, "defender": side,
		}, "aggressor", "defender"),
	}, "combined", "aggressor", "defender", "topShips", "sides")
}

func warDashboardDetailSchema(includeLeaderboards bool) *huma.Schema {
	ally := responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "name": stringSchema(), "type": stringSchema(),
		"kills": intSchema(), "losses": intSchema(),
		"isk_destroyed": numberSchema(), "isk_lost": numberSchema(),
	}, "id", "name", "type", "kills", "losses", "isk_destroyed", "isk_lost")
	war := conflictWarSchema()
	war.Properties["allies"] = arraySchema(ally)
	stats := responseSchema(map[string]*huma.Schema{
		"total_kills": intSchema(), "total_value": numberSchema(),
		"top_ships": arraySchema(conflictTopShipSchema()),
	}, "total_kills", "total_value", "top_ships")
	properties := map[string]*huma.Schema{"war": war, "stats": stats}
	required := []string{"war", "stats"}
	if includeLeaderboards {
		properties["leaderboards"] = warLeaderboardsSchema()
		required = append(required, "leaderboards")
	}
	return responseSchema(properties, required...)
}

func conflictMemberSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"character_id": intSchema(), "character_name": stringSchema(),
		"side": stringSchema(), "corporation_id": nullable(intSchema()),
		"corporation_name":   nullable(stringSchema()),
		"corporation_ticker": nullable(stringSchema()),
		"alliance_id":        nullable(intSchema()), "alliance_name": nullable(stringSchema()),
		"alliance_ticker": nullable(stringSchema()), "kills": intSchema(),
		"losses": intSchema(), "isk_destroyed": numberSchema(),
		"isk_lost": numberSchema(), "top_ship_type_id": nullable(intSchema()),
		"top_ship_name": nullable(stringSchema()), "top_ship_count": intSchema(),
	}, "character_id", "character_name", "side", "corporation_id",
		"corporation_name", "corporation_ticker", "alliance_id",
		"alliance_name", "alliance_ticker", "kills", "losses",
		"isk_destroyed", "isk_lost", "top_ship_type_id", "top_ship_name",
		"top_ship_count")
}

func conflictMembersSchema(extra ...string) *huma.Schema {
	properties := map[string]*huma.Schema{
		"side": stringSchema(), "count": intSchema(), "limit": intSchema(),
		"members": arraySchema(conflictMemberSchema()),
	}
	required := []string{"side", "count", "limit", "members"}
	for _, name := range extra {
		if name == "matchup" {
			properties[name] = stringSchema()
		} else {
			properties[name] = intSchema()
		}
		required = append(required, name)
	}
	return responseSchema(properties, required...)
}

func conflictIntelSchema(extra ...string) *huma.Schema {
	properties := map[string]*huma.Schema{
		"summary": responseSchema(map[string]*huma.Schema{
			"kills": intSchema(), "isk_destroyed": numberSchema(),
			"systems": intSchema(), "constellations": intSchema(),
			"regions": intSchema(),
		}, "kills", "isk_destroyed", "systems", "constellations", "regions"),
		"top_systems": arraySchema(recordSchema(map[string]*huma.Schema{
			"system_id": intSchema(), "system_name": nullable(stringSchema()),
			"security": nullable(numberSchema()), "region_id": nullable(intSchema()),
			"region_name": nullable(stringSchema()), "kills": intSchema(),
			"isk_destroyed": numberSchema(),
		})),
		"top_constellations": arraySchema(recordSchema(map[string]*huma.Schema{
			"constellation_id":   intSchema(),
			"constellation_name": nullable(stringSchema()),
			"region_id":          nullable(intSchema()), "region_name": nullable(stringSchema()),
			"kills": intSchema(), "isk_destroyed": numberSchema(),
		})),
		"top_regions": arraySchema(recordSchema(map[string]*huma.Schema{
			"region_id": intSchema(), "region_name": nullable(stringSchema()),
			"kills": intSchema(), "isk_destroyed": numberSchema(),
		})),
		"ships_destroyed": arraySchema(conflictIntelShipSchema(true)),
		"ships_used":      arraySchema(conflictIntelShipSchema(false)),
		"ship_groups_destroyed": arraySchema(recordSchema(map[string]*huma.Schema{
			"group_id": intSchema(), "group_name": nullable(stringSchema()),
			"count": intSchema(), "isk_destroyed": numberSchema(),
		})),
		"security_breakdown": arraySchema(responseSchema(map[string]*huma.Schema{
			"sec_class": stringSchema(), "kills": intSchema(),
			"isk_destroyed": numberSchema(),
		}, "sec_class", "kills", "isk_destroyed")),
	}
	required := []string{
		"summary", "top_systems", "top_constellations", "top_regions",
		"ships_destroyed", "ships_used", "ship_groups_destroyed",
		"security_breakdown",
	}
	for _, name := range extra {
		if name == "matchup" {
			properties[name] = stringSchema()
		} else {
			properties[name] = intSchema()
		}
		required = append(required, name)
	}
	return responseSchema(properties, required...)
}

func conflictIntelShipSchema(destroyed bool) *huma.Schema {
	properties := map[string]*huma.Schema{
		"ship_type_id": intSchema(), "ship_name": nullable(stringSchema()),
		"group_id": nullable(intSchema()), "group_name": nullable(stringSchema()),
		"count": intSchema(),
	}
	required := []string{"ship_type_id", "ship_name", "group_id", "group_name", "count"}
	if destroyed {
		properties["isk_destroyed"] = numberSchema()
		required = append(required, "isk_destroyed")
	}
	return responseSchema(properties, required...)
}

func factionWarListingSideSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"faction_id": intSchema(), "name": stringSchema(), "kills": intSchema(),
		"isk_destroyed": numberSchema(), "losses": intSchema(),
		"isk_lost": numberSchema(), "pilots": intSchema(),
		"systems_controlled": intSchema(),
	}, "faction_id", "name", "kills", "isk_destroyed", "losses",
		"isk_lost", "pilots", "systems_controlled")
}

func factionWarsDashboardSchema() *huma.Schema {
	matchup := mapOfSchema(factionWarListingSideSchema())
	return &huma.Schema{
		Type: huma.TypeObject,
		Properties: map[string]*huma.Schema{
			"amarr-minmatar":   matchup,
			"caldari-gallente": matchup,
		},
		Required:             []string{"amarr-minmatar", "caldari-gallente"},
		AdditionalProperties: false,
	}
}

func factionWarCombatSideSchema() *huma.Schema {
	leader := responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "name": nullable(stringSchema()), "kills": intSchema(),
	}, "id", "name", "kills")
	return responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "name": stringSchema(), "corpId": intSchema(),
		"kills": intSchema(), "isk_destroyed": numberSchema(),
		"losses": intSchema(), "isk_lost": numberSchema(),
		"topCharacters":   arraySchema(leader),
		"topCorporations": arraySchema(leader),
		"topAlliances":    arraySchema(leader),
	}, "id", "name", "corpId", "kills", "isk_destroyed", "losses",
		"isk_lost", "topCharacters", "topCorporations", "topAlliances")
}

func factionWarBaseSchema(includeOverview bool) *huma.Schema {
	properties := map[string]*huma.Schema{
		"matchup": stringSchema(), "days": intSchema(),
		"side1": factionWarCombatSideSchema(),
		"side2": factionWarCombatSideSchema(),
		"topShips": arraySchema(responseSchema(map[string]*huma.Schema{
			"ship_type_id": intSchema(), "ship_name": stringSchema(),
			"kills": intSchema(),
		}, "ship_type_id", "ship_name", "kills")),
	}
	required := []string{"matchup", "days", "side1", "side2", "topShips"}
	if includeOverview {
		properties["overview"] = factionWarOverviewSchema()
		required = append(required, "overview")
	}
	return responseSchema(properties, required...)
}

func factionWarOverviewSchema() *huma.Schema {
	stats := nullable(responseSchema(map[string]*huma.Schema{
		"pilots": intSchema(), "systems_controlled": intSchema(),
		"kills_yesterday": intSchema(), "kills_last_week": intSchema(),
		"kills_total": intSchema(), "vp_yesterday": intSchema(),
		"vp_last_week": intSchema(), "vp_total": intSchema(),
	}, "pilots", "systems_controlled", "kills_yesterday", "kills_last_week",
		"kills_total", "vp_yesterday", "vp_last_week", "vp_total"))
	breakdown := responseSchema(map[string]*huma.Schema{
		"total": intSchema(), "uncontested": intSchema(),
		"contested": intSchema(), "vulnerable": intSchema(),
		"captured": intSchema(), "total_vp": intSchema(),
		"total_threshold": intSchema(),
	}, "total", "uncontested", "contested", "vulnerable", "captured",
		"total_vp", "total_threshold")
	leader := responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "name": nullable(stringSchema()), "kills": intSchema(),
		"corporation_name": nullable(stringSchema()),
	}, "id", "name", "kills")
	leaderSides := responseSchema(map[string]*huma.Schema{
		"side1": arraySchema(leader), "side2": arraySchema(leader),
	}, "side1", "side2")
	flip := responseSchema(map[string]*huma.Schema{
		"solar_system_id": intSchema(), "system_name": nullable(stringSchema()),
		"old_faction_id": intSchema(), "old_faction_name": nullable(stringSchema()),
		"new_faction_id": intSchema(), "new_faction_name": nullable(stringSchema()),
		"flipped_at": timestampSchema(),
	}, "solar_system_id", "system_name", "old_faction_id",
		"old_faction_name", "new_faction_id", "new_faction_name", "flipped_at")
	return responseSchema(map[string]*huma.Schema{
		"factionStats": responseSchema(map[string]*huma.Schema{
			"side1": stats, "side2": stats,
		}, "side1", "side2"),
		"warzone": responseSchema(map[string]*huma.Schema{
			"total_systems": intSchema(), "side1": breakdown, "side2": breakdown,
		}, "total_systems", "side1", "side2"),
		"flipDays": arraySchema(responseSchema(map[string]*huma.Schema{
			"day": stringSchema(), "items": arraySchema(flip),
		}, "day", "items")),
		"leaderboards": responseSchema(map[string]*huma.Schema{
			"characters": leaderSides, "corporations": leaderSides,
		}, "characters", "corporations"),
	}, "factionStats", "warzone", "flipDays", "leaderboards")
}

func factionWarSystemsSchema() *huma.Schema {
	system := responseSchema(map[string]*huma.Schema{
		"solar_system_id": intSchema(), "system_name": nullable(stringSchema()),
		"x": numberSchema(), "y": numberSchema(), "security": numberSchema(),
		"region_id": intSchema(), "region_name": nullable(stringSchema()),
		"constellation_id":   intSchema(),
		"constellation_name": nullable(stringSchema()),
		"owner_faction_id":   intSchema(), "occupier_faction_id": intSchema(),
		"contested": nullable(stringSchema()), "victory_points": intSchema(),
		"victory_points_threshold": intSchema(), "kills_24h": intSchema(),
		"isk_24h": numberSchema(),
	}, "solar_system_id", "system_name", "x", "y", "security",
		"region_id", "region_name", "constellation_id", "constellation_name",
		"owner_faction_id", "occupier_faction_id", "contested",
		"victory_points", "victory_points_threshold", "kills_24h", "isk_24h")
	celestial := responseSchema(map[string]*huma.Schema{
		"system_id": intSchema(), "group_id": intSchema(),
		"x": numberSchema(), "z": numberSchema(),
	}, "system_id", "group_id", "x", "z")
	return responseSchema(map[string]*huma.Schema{
		"systems":    arraySchema(system),
		"jumps":      arraySchema(arraySchema(intSchema())),
		"celestials": arraySchema(celestial),
	}, "systems", "jumps", "celestials")
}
