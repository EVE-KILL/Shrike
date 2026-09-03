package api

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

func publicOperationResponse(operationID string, status int) *huma.Response {
	switch operationID {
	case "eve-login-start", "eve-login-callback", "eve-login-callback-legacy":
		return &huma.Response{
			Description: http.StatusText(status),
			Headers: map[string]*huma.Param{
				"Location": {
					Description: "Browser destination for the next step in the login flow.",
					Schema:      stringSchema(),
				},
			},
		}
	case "admin-domain-asset-preview", "image-domain-banner-or-logo",
		"image-domain-background", "image-domain-asset-preview",
		"domain-banner-or-logo", "domain-background", "domain-asset-preview":
		binary := &huma.Schema{Type: huma.TypeString, Format: "binary"}
		return &huma.Response{
			Description: http.StatusText(status),
			Content: map[string]*huma.MediaType{
				"image/jpeg": {Schema: binary},
				"image/png":  {Schema: binary},
				"image/webp": {Schema: binary},
				"image/gif":  {Schema: binary},
			},
		}
	}
	return nil
}

func adminAnnouncementSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"id": intSchema(), "tier": intSchema(), "title": stringSchema(),
		"body_md": stringSchema(), "body_html": stringSchema(),
		"color": stringSchema(), "icon": nullable(stringSchema()),
		"link_url": nullable(stringSchema()), "link_label": nullable(stringSchema()),
		"starts_at": timestampSchema(), "expires_at": timestampSchema(),
		"created_by": intSchema(), "created_at": timestampSchema(),
		"updated_at": timestampSchema(), "archived_at": nullable(timestampSchema()),
	})
}

func blogPostSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "slug": stringSchema(), "title": stringSchema(),
		"excerpt": nullable(stringSchema()), "body_md": stringSchema(),
		"body_html": stringSchema(), "cover_image_url": nullable(stringSchema()),
		"status": intSchema(), "author_id": intSchema(),
		"author_name": stringSchema(), "published_at": nullable(timestampSchema()),
		"created_at": timestampSchema(), "updated_at": timestampSchema(),
		"tags":                    arraySchema(stringSchema()),
		"author_corporation_id":   nullable(intSchema()),
		"author_corporation_name": nullable(stringSchema()),
		"author_alliance_id":      nullable(intSchema()),
		"author_alliance_name":    nullable(stringSchema()),
	}, "id", "slug", "title", "excerpt", "body_md", "body_html",
		"cover_image_url", "status", "author_id", "author_name",
		"published_at", "created_at", "updated_at", "tags",
		"author_corporation_id", "author_corporation_name",
		"author_alliance_id", "author_alliance_name")
}

func commentSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "target_type": intSchema(), "target_id": intSchema(),
		"target_slug": nullable(stringSchema()), "domain_id": nullable(intSchema()),
		"parent_id": nullable(intSchema()), "root_id": nullable(intSchema()),
		"depth": intSchema(), "body_md": stringSchema(), "body_html": stringSchema(),
		"character_id": intSchema(), "character_name": stringSchema(),
		"corporation_id": intSchema(), "corporation_name": stringSchema(),
		"alliance_id": nullable(intSchema()), "alliance_name": nullable(stringSchema()),
		"created_at": timestampSchema(), "updated_at": timestampSchema(),
		"edited_at": nullable(timestampSchema()), "deleted_at": nullable(timestampSchema()),
		"deleted_by": nullable(intSchema()), "reports_count": intSchema(),
		"flagged": boolSchema(), "moderation_status": intSchema(),
		"visibility": intSchema(), "reply_count": intSchema(),
	}, "id", "target_type", "target_id", "target_slug", "domain_id",
		"parent_id", "root_id", "depth", "body_md", "body_html",
		"character_id", "character_name", "corporation_id",
		"corporation_name", "alliance_id", "alliance_name", "created_at",
		"updated_at", "edited_at", "deleted_at", "deleted_by",
		"reports_count", "flagged", "moderation_status", "visibility")
}

func commentsFeedSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"comments":   arraySchema(commentSchema()),
		"nextCursor": nullable(intSchema()),
	}, "comments", "nextCursor")
}

func commentsThreadSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"roots": arraySchema(commentSchema()), "replies": arraySchema(commentSchema()),
		"repliesTruncated": boolSchema(), "nextCursor": nullable(intSchema()),
	}, "roots", "replies", "repliesTruncated", "nextCursor")
}

func commentReportSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "comment_id": intSchema(), "reporter_id": intSchema(),
		"reporter_name": stringSchema(), "reason": stringSchema(),
		"message": nullable(stringSchema()), "created_at": timestampSchema(),
		"resolved_at": nullable(timestampSchema()),
		"resolved_by": nullable(intSchema()), "resolution": nullable(stringSchema()),
	}, "id", "comment_id", "reporter_id", "reporter_name", "reason",
		"message", "created_at", "resolved_at", "resolved_by", "resolution")
}

func moderationQueueItemSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "target_kind": intSchema(), "target_id": intSchema(),
		"body": stringSchema(), "body_format": stringSchema(),
		"rendered_html": stringSchema(), "character_id": intSchema(),
		"character_name": stringSchema(), "corporation_id": nullable(intSchema()),
		"corporation_name": nullable(stringSchema()),
		"alliance_id":      nullable(intSchema()), "alliance_name": nullable(stringSchema()),
		"ai_action": stringSchema(), "ai_category": nullable(stringSchema()),
		"ai_max_score": numberSchema(), "ai_scores": mapOfSchema(numberSchema()),
		"ai_source": stringSchema(), "status": intSchema(),
		"submitted_at": timestampSchema(), "reviewed_at": nullable(timestampSchema()),
		"reviewed_by": nullable(intSchema()), "review_notes": nullable(stringSchema()),
		"comment_context": nullable(responseSchema(map[string]*huma.Schema{
			"target_type": intSchema(), "target_id": intSchema(),
			"target_slug": nullable(stringSchema()),
		}, "target_type", "target_id", "target_slug")),
	}, "id", "target_kind", "target_id", "body", "body_format",
		"rendered_html", "character_id", "character_name", "corporation_id",
		"corporation_name", "alliance_id", "alliance_name", "ai_action",
		"ai_category", "ai_max_score", "ai_scores", "ai_source", "status",
		"submitted_at", "reviewed_at", "reviewed_by", "review_notes",
		"comment_context")
}

func moderationQueueResponseSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"items":      arraySchema(moderationQueueItemSchema()),
		"nextCursor": nullable(intSchema()),
		"counts": responseSchema(map[string]*huma.Schema{
			"pending": intSchema(), "pending_comments": intSchema(),
			"pending_bios": intSchema(), "total": intSchema(),
		}, "pending", "pending_comments", "pending_bios", "total"),
	}, "items", "nextCursor", "counts")
}

func klipyResponseSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"items": arraySchema(openJSONObjectSchema(
			"Klipy GIF record passed through from the upstream service.",
		)),
		"page": intSchema(), "per_page": intSchema(), "has_next": boolSchema(),
	}, "items", "page", "per_page", "has_next")
}

func domainEntitySchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"type": stringSchema(), "id": intSchema(), "name": stringSchema(),
	}, "type", "id")
}

func domainWidgetSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"type": stringSchema(), "enabled": boolSchema(),
		"killlistType": stringSchema(), "title": stringSchema(),
		"content": stringSchema(), "campaignId": stringSchema(),
	})
}

func domainAssetSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"id": intSchema(), "domain_id": intSchema(), "type": stringSchema(),
		"status": stringSchema(), "reject_reason": nullable(stringSchema()),
		"created_at": timestampSchema(), "updated_at": timestampSchema(),
	})
}

func domainCampaignSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"campaign_id": stringSchema(), "name": stringSchema(),
		"description": nullable(stringSchema()), "visibility": intSchema(),
		"status": intSchema(), "start_time": timestampSchema(),
		"end_time": timestampSchema(), "created_by_character_id": intSchema(),
		"estimated_killmails": nullable(intSchema()), "public_on_domain": boolSchema(),
	})
}

func domainSchema() *huma.Schema {
	navbarItem := responseSchema(map[string]*huma.Schema{
		"label": stringSchema(), "href": stringSchema(),
		"external": boolSchema(), "icon": stringSchema(),
	}, "label", "href")
	navbarGroup := responseSchema(map[string]*huma.Schema{
		"label": stringSchema(), "items": arraySchema(navbarItem),
	}, "items")
	navbarLink := responseSchema(map[string]*huma.Schema{
		"label": stringSchema(), "href": stringSchema(),
		"external": boolSchema(), "icon": stringSchema(),
		"children": arraySchema(navbarGroup),
	}, "label", "href")
	return recordSchema(map[string]*huma.Schema{
		"id": intSchema(), "subdomain": stringSchema(),
		"custom_hostname": nullable(stringSchema()), "user_id": intSchema(),
		"entities":         arraySchema(domainEntitySchema()),
		"theme":            domainThemeSchema(),
		"navbar_links":     arraySchema(navbarLink),
		"site_name":        nullable(stringSchema()),
		"site_description": nullable(stringSchema()), "active": boolSchema(),
		"created_at": timestampSchema(), "updated_at": timestampSchema(),
		"widgets": responseSchema(map[string]*huma.Schema{
			"top":         arraySchema(domainWidgetSchema()),
			"left":        arraySchema(domainWidgetSchema()),
			"right":       arraySchema(domainWidgetSchema()),
			"columnRatio": stringSchema(),
		}, "top", "left", "right", "columnRatio"),
		"campaign_policy": intSchema(),
		"campaigns":       arraySchema(domainCampaignSchema()),
		"backgrounds":     arraySchema(domainAssetSchema()),
		"bannerAsset":     nullable(domainAssetSchema()),
		"logoAsset":       nullable(domainAssetSchema()),
	})
}

func domainThemeSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"primaryColor": stringSchema(), "accentColor": stringSchema(),
		"bgColor": stringSchema(), "textColor": stringSchema(),
		"bannerUrl": stringSchema(), "logoUrl": stringSchema(),
		"showLogoInBanner": boolSchema(), "showNameInBanner": boolSchema(),
		"showDescriptionInBanner": boolSchema(), "transparentBanner": boolSchema(),
		"contentOpacity": numberSchema(), "defaultThemePreset": stringSchema(),
		"defaultThemeOverrides": mapOfSchema(stringSchema()),
	})
}

func walletCorporationSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"corporation_id": intSchema(), "name": stringSchema(), "ticker": stringSchema(),
	}, "corporation_id", "name", "ticker")
}

func walletBalanceSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"balance": stringSchema(), "totalBalance": stringSchema(),
		"reservedBalance": stringSchema(), "availableBalance": stringSchema(),
	}, "balance", "totalBalance", "reservedBalance", "availableBalance")
}

func walletJournalSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"division": intSchema(), "journal_id": intSchema(),
		"date": timestampSchema(), "ref_type": stringSchema(),
		"description": nullable(stringSchema()), "amount": stringSchema(),
		"balance": stringSchema(), "first_party_id": nullable(intSchema()),
		"second_party_id": nullable(intSchema()), "context_id": nullable(intSchema()),
		"context_id_type": nullable(stringSchema()), "reason": nullable(stringSchema()),
		"tax": nullable(stringSchema()), "tax_receiver_id": nullable(intSchema()),
	})
}

func publicWalletSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"corporation": walletCorporationSchema(), "totalBalance": stringSchema(),
		"lastSynced": nullable(timestampSchema()), "journal": arraySchema(walletJournalSchema()),
		"page": intSchema(), "division": nullable(intSchema()), "hasMore": boolSchema(),
		"pageSize": intSchema(),
	}, "corporation", "totalBalance", "lastSynced", "journal", "page",
		"division", "hasMore", "pageSize")
}

func accountWalletSchema() *huma.Schema {
	transaction := recordSchema(map[string]*huma.Schema{
		"id": intSchema(), "type": intSchema(), "amount": stringSchema(),
		"balance_after": stringSchema(), "description": nullable(stringSchema()),
		"created_at": timestampSchema(),
	})
	reservation := recordSchema(map[string]*huma.Schema{
		"id": intSchema(), "external_reference": stringSchema(),
		"transaction_type": intSchema(), "amount": stringSchema(),
		"description": nullable(stringSchema()), "expires_at": timestampSchema(),
		"created_at": timestampSchema(),
	})
	return responseSchema(map[string]*huma.Schema{
		"character": responseSchema(map[string]*huma.Schema{
			"character_id": intSchema(), "character_name": stringSchema(),
		}, "character_id", "character_name"),
		"corporation": walletCorporationSchema(),
		"balance":     stringSchema(), "totalBalance": stringSchema(),
		"reservedBalance": stringSchema(), "availableBalance": stringSchema(),
		"lastSynced":        nullable(timestampSchema()),
		"depositsEnabledAt": nullable(timestampSchema()),
		"transactions":      arraySchema(transaction), "reservations": arraySchema(reservation),
		"page": intSchema(), "hasMore": boolSchema(), "pageSize": intSchema(),
	}, "character", "corporation", "balance", "totalBalance",
		"reservedBalance", "availableBalance", "lastSynced",
		"depositsEnabledAt", "transactions", "reservations", "page",
		"hasMore", "pageSize")
}

func adminWalletSchema() *huma.Schema {
	authorization := nullable(recordSchema(map[string]*huma.Schema{
		"authorized_character_id":          intSchema(),
		"authorized_character_name":        stringSchema(),
		"authorized_by_admin_character_id": intSchema(),
		"token_expiry":                     nullable(timestampSchema()), "scopes": arraySchema(stringSchema()),
		"disabled": boolSchema(), "last_balance_sync": nullable(timestampSchema()),
		"last_journal_sync": nullable(timestampSchema()),
		"last_error":        nullable(stringSchema()), "created_at": timestampSchema(),
		"updated_at": timestampSchema(),
	}))
	return responseSchema(map[string]*huma.Schema{
		"corporation": walletCorporationSchema(), "authorization": authorization,
		"requiredScopes": arraySchema(stringSchema()),
		"balances": arraySchema(recordSchema(map[string]*huma.Schema{
			"division": intSchema(), "balance": stringSchema(),
			"updated_at": timestampSchema(),
		})),
		"totalBalance": stringSchema(), "journal": arraySchema(walletJournalSchema()),
		"page": intSchema(), "division": nullable(intSchema()),
		"hasMore": boolSchema(), "pageSize": intSchema(),
		"prizeSettlements": arraySchema(recordSchema(map[string]*huma.Schema{
			"campaign_id": stringSchema(), "campaign_name": stringSchema(),
			"pool_status": intSchema(), "funded_total": stringSchema(),
			"finalized_at": nullable(timestampSchema()), "rank": nullable(intSchema()),
			"character_id": nullable(intSchema()), "character_name": nullable(stringSchema()),
			"metric_value":      nullable(stringSchema()),
			"payout_percentage": nullable(intSchema()),
			"payout_amount":     nullable(stringSchema()),
			"claimed_at":        nullable(timestampSchema()), "paid_at": nullable(timestampSchema()),
			"payment_note": nullable(stringSchema()),
		})),
		"walletReferences": arraySchema(recordSchema(map[string]*huma.Schema{
			"corporation_id": intSchema(), "division": intSchema(),
			"journal_id": intSchema(), "reference_type": stringSchema(),
			"reference_id": stringSchema(), "status": intSchema(),
			"amount": stringSchema(), "note": nullable(stringSchema()),
			"created_at": timestampSchema(), "date": timestampSchema(),
			"first_party_id": nullable(intSchema()), "reason": nullable(stringSchema()),
		})),
	}, "corporation", "authorization", "requiredScopes", "balances",
		"totalBalance", "journal", "page", "division", "hasMore",
		"pageSize", "prizeSettlements", "walletReferences")
}

func redditBackgroundsSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"images": arraySchema(responseSchema(map[string]*huma.Schema{
			"url": stringSchema(), "title": stringSchema(),
			"source": stringSchema(), "subreddit": stringSchema(),
		}, "url", "title", "source", "subreddit")),
	}, "images")
}

func marketTreeEntrySchema(children *huma.Schema) *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "name": stringSchema(), "slug": stringSchema(),
		"parent_id": nullable(intSchema()), "has_types": boolSchema(),
		"icon_id": nullable(intSchema()), "children": children,
	}, "id", "name", "slug", "parent_id", "has_types", "icon_id", "children")
}

func marketTreeNodeSchema() *huma.Schema {
	leaf := marketTreeEntrySchema(arraySchema(openJSONObjectSchema(
		"Additional deeply nested market group.",
	)))
	child := marketTreeEntrySchema(arraySchema(leaf))
	return marketTreeEntrySchema(arraySchema(child))
}

func marketGroupItemsSchema() *huma.Schema {
	item := responseSchema(map[string]*huma.Schema{
		"type_id": intSchema(), "name": stringSchema(), "group_id": intSchema(),
		"market_group_id": intSchema(),
		"category_id":     intSchema(), "meta_group_id": nullable(intSchema()),
		"is_ship": boolSchema(), "universe_average": nullable(numberSchema()),
		"universe_volume": nullable(intSchema()), "jita_sell": nullable(numberSchema()),
	}, "type_id", "name", "group_id", "market_group_id", "category_id", "meta_group_id", "is_ship", "universe_average", "universe_volume", "jita_sell")
	return responseSchema(map[string]*huma.Schema{
		"items": arraySchema(item),
	}, "items")
}

func marketItemSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"type_id": intSchema(), "name": stringSchema(),
		"group_id": nullable(intSchema()), "market_group_id": nullable(intSchema()),
	}, "type_id", "name", "group_id", "market_group_id")
}

func marketOrderSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"order_id": intSchema(), "duration": intSchema(), "is_buy_order": boolSchema(),
		"issued": timestampSchema(), "location_id": intSchema(), "min_volume": intSchema(),
		"price": numberSchema(), "order_range": stringSchema(), "system_id": intSchema(),
		"type_id": intSchema(), "volume_remain": intSchema(), "volume_total": intSchema(),
		"region_id": intSchema(), "constellation_id": nullable(intSchema()),
		"snapshot_at": timestampSchema(), "system_name": nullable(stringSchema()),
		"security": nullable(numberSchema()), "region_name": nullable(stringSchema()),
		"location_name": stringSchema(), "expires_at": timestampSchema(),
	}, "order_id", "duration", "is_buy_order", "issued", "location_id", "min_volume",
		"price", "order_range", "system_id", "type_id", "volume_remain", "volume_total",
		"region_id", "constellation_id", "snapshot_at", "system_name", "security",
		"region_name", "location_name", "expires_at")
}

func marketRegionSummarySchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"region_id": intSchema(), "name": nullable(stringSchema()), "order_count": intSchema(),
		"lowest_sell": nullable(numberSchema()), "highest_buy": nullable(numberSchema()),
	}, "region_id", "name", "order_count", "lowest_sell", "highest_buy")
}

func marketItemOrdersSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"item": marketItemSchema(), "snapshot_at": nullable(timestampSchema()),
		"region_id": intSchema(), "security": arraySchema(stringSchema()),
		"sellers": arraySchema(marketOrderSchema()), "buyers": arraySchema(marketOrderSchema()),
		"regions": arraySchema(marketRegionSummarySchema()),
	}, "item", "snapshot_at", "region_id", "security", "sellers", "buyers", "regions")
}

func marketItemHistorySchema() *huma.Schema {
	point := responseSchema(map[string]*huma.Schema{
		"date": dateSchema(), "average": nullable(numberSchema()),
		"highest": nullable(numberSchema()), "lowest": nullable(numberSchema()),
		"order_count": nullable(intSchema()), "volume": nullable(intSchema()),
		"source_updated_at": nullable(timestampSchema()),
	}, "date", "average", "highest", "lowest", "order_count", "volume", "source_updated_at")
	return responseSchema(map[string]*huma.Schema{
		"type_id": intSchema(), "region_id": intSchema(), "days": intSchema(),
		"history": arraySchema(point),
	}, "type_id", "region_id", "days", "history")
}

func directionalScanSchema() *huma.Schema {
	scanType := responseSchema(map[string]*huma.Schema{
		"typeName": stringSchema(), "typeId": nullable(intSchema()), "count": intSchema(),
	}, "typeName", "typeId", "count")
	group := responseSchema(map[string]*huma.Schema{
		"groupId": nullable(intSchema()), "types": arraySchema(scanType),
	}, "groupId", "types")
	category := responseSchema(map[string]*huma.Schema{
		"categoryId": nullable(intSchema()), "groups": mapOfSchema(group),
	}, "categoryId", "groups")
	return responseSchema(map[string]*huma.Schema{
		"grouped": mapOfSchema(category), "totalCount": intSchema(),
		"uniqueTypes": intSchema(),
	}, "grouped", "totalCount", "uniqueTypes")
}

func localScanCorporationSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"name": stringSchema(), "ticker": stringSchema(),
		"characters": arraySchema(responseSchema(map[string]*huma.Schema{
			"characterId": intSchema(), "name": stringSchema(), "kills": intSchema(),
		}, "characterId", "name", "kills")),
	}, "name", "ticker", "characters")
}

func localScanSchema() *huma.Schema {
	alliance := responseSchema(map[string]*huma.Schema{
		"name": stringSchema(), "ticker": stringSchema(),
		"corporations": mapOfSchema(localScanCorporationSchema()),
	}, "name", "ticker", "corporations")
	return responseSchema(map[string]*huma.Schema{
		"alliances":    mapOfSchema(alliance),
		"corporations": mapOfSchema(localScanCorporationSchema()),
		"unresolved":   arraySchema(stringSchema()), "totalCharacters": intSchema(),
		"totalDangerous": intSchema(),
	}, "alliances", "corporations", "unresolved", "totalCharacters",
		"totalDangerous")
}

func sitemapEntrySchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"loc": stringSchema(), "changefreq": stringSchema(),
		"priority": numberSchema(), "lastmod": timestampSchema(),
	}, "loc", "changefreq", "priority")
}

func legacyArchiveKillSchema() *huma.Schema {
	kill := recordSchema(map[string]*huma.Schema{
		"killmail_id": intSchema(), "killmail_time": timestampSchema(),
		"total_value": numberSchema(), "victim_name": nullable(stringSchema()),
		"victim_corp": nullable(stringSchema()), "victim_alliance": nullable(stringSchema()),
		"victim_ship": nullable(stringSchema()), "system_name": nullable(stringSchema()),
		"security": nullable(numberSchema()),
	})
	attacker := recordSchema(map[string]*huma.Schema{
		"id": intSchema(), "killmail_id": intSchema(),
		"name": nullable(stringSchema()), "character_id": nullable(intSchema()),
		"corporation_id": nullable(intSchema()), "alliance_id": nullable(intSchema()),
		"ship_type_id": nullable(intSchema()), "damage_done": intSchema(),
		"final_blow": boolSchema(),
	})
	item := recordSchema(map[string]*huma.Schema{
		"id": intSchema(), "killmail_id": intSchema(), "name": nullable(stringSchema()),
		"type_id": nullable(intSchema()), "quantity_destroyed": nullable(intSchema()),
		"quantity_dropped": nullable(intSchema()), "singleton": intSchema(),
		"flag": intSchema(),
	})
	return responseSchema(map[string]*huma.Schema{
		"kill": kill, "attackers": arraySchema(attacker), "items": arraySchema(item),
	}, "kill", "attackers", "items")
}

func domainStatisticsSchema() *huma.Schema {
	return entriesResponseSchema(&huma.Schema{OneOf: []*huma.Schema{
		mostValuableKillSchema(),
		recordSchema(map[string]*huma.Schema{
			"id": intSchema(), "name": stringSchema(), "type": stringSchema(),
			"count": intSchema(), "value": numberSchema(), "kills": intSchema(),
			"losses": intSchema(), "isk_destroyed": numberSchema(),
			"isk_lost": numberSchema(), "palette": nullable(stringSchema()),
		}),
	}})
}
