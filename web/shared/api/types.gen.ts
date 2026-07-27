// Generated from shared/api.openapi.json by @hey-api/openapi-ts.
// Do not edit by hand; run `make gen-api-client`.

export type ClientOptions = {
    baseUrl: `${string}://${string}/api` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | (string & {});
};

export type Announcement = {
    body_html: string;
    body_md: string;
    color: 'info' | 'warning' | 'danger' | 'success';
    created_at: string;
    expires_at: string;
    icon: string | null;
    id: number;
    link_label: string | null;
    link_url: string | null;
    starts_at: string;
    tier: 1 | 2 | 3;
    title: string;
};

export type AnnouncementDismissalResponse = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    ok: boolean;
};

export type AnnouncementsResponse = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    announcements: Array<Announcement>;
};

export type CoalitionSideBody = {
    /**
     * Alliance IDs on this side.
     */
    alliances?: Array<number | string> | null;
    /**
     * Corporation IDs on this side.
     */
    corporations?: Array<number | string> | null;
    /**
     * Display name for this side. Truncated at 120 characters.
     */
    label?: string;
};

export type ConflictBattleGeneratorEntity = {
    id: number;
    type: string;
};

export type ConflictBattleGeneratorSide = {
    entities: Array<ConflictBattleGeneratorEntity> | null;
    name: string;
};

export type ConflictBattleSaveAlliance = {
    alliance_id: number | null;
    corporations: Array<ConflictBattleSaveCorporation> | null;
};

export type ConflictBattleSaveCorporation = {
    corporation_id: number;
    isk_destroyed: number;
    isk_lost: number;
    kills: number;
    losses: number;
};

export type ConflictBattleSaveTeam = {
    alliances: Array<ConflictBattleSaveAlliance> | null;
    total_isk_destroyed: number;
    total_isk_lost: number;
    total_kills: number;
    total_losses: number;
};

export type DismissedAnnouncementIdsResponse = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    dismissedIds: Array<number>;
};

export type ErrorDetail = {
    /**
     * Where the error occurred, e.g. 'body.items[3].tags' or 'path.thing-id'
     */
    location?: string;
    /**
     * Error message text
     */
    message?: string;
    /**
     * The value at the given location
     */
    value?: unknown;
};

export type ErrorModel = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    /**
     * A human-readable explanation specific to this occurrence of the problem.
     */
    detail?: string;
    /**
     * Optional list of individual error details
     */
    errors?: Array<ErrorDetail> | null;
    /**
     * A URI reference that identifies the specific occurrence of the problem.
     */
    instance?: string;
    /**
     * HTTP status code
     */
    status?: number;
    /**
     * A short, human-readable summary of the problem type. This value should not change between occurrences of the error.
     */
    title?: string;
    /**
     * A URI reference to human-readable documentation for the error.
     */
    type?: string;
};

export type FittingItemBody = {
    /**
     * Charge loaded into this module, when it takes one.
     */
    charge_type_id?: number;
    /**
     * Position within the slot family.
     */
    ordinal: number | null;
    /**
     * Stack size. Must be 1 for module slots. Defaults to 1.
     */
    quantity?: number;
    /**
     * Slot family: 1-5 are module slots, 6 drones, 7 cargo.
     */
    slot_group: number | null;
    /**
     * Module state: offline, online, active, overloaded.
     */
    state: number | null;
    /**
     * Inventory type fitted in this position.
     */
    type_id: number | null;
};

export type ImagesOverviewResponse = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    routes: Array<string> | null;
    service: string;
};

export type SiteConfigurationResponse = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    domain: SiteDomainConfiguration | null;
    isDomainHost: boolean;
};

export type SiteDomainConfiguration = {
    backgrounds: Array<string>;
    campaignIds: Array<string>;
    campaignPolicy: 0 | 1;
    customHostname: string | null;
    entities: Array<SiteDomainEntity>;
    entityIds: SiteDomainEntityIds;
    id: number;
    navbarLinks: Array<SiteDomainNavbarLink>;
    publicCampaignIds: Array<string>;
    siteDescription: string | null;
    siteName: string | null;
    subdomain: string;
    theme: SiteDomainTheme;
    userId: number;
    widgets: SiteDomainWidgets;
};

export type SiteDomainEntity = {
    id: number;
    name: string;
    type: 'character' | 'corporation' | 'alliance';
};

export type SiteDomainEntityIds = {
    allianceIds: Array<number>;
    characterIds: Array<number>;
    corporationIds: Array<number>;
};

export type SiteDomainNavbarGroup = {
    items: Array<SiteDomainNavbarItem>;
    label?: string;
};

export type SiteDomainNavbarItem = {
    external?: boolean;
    href: string;
    icon?: string;
    label: string;
};

export type SiteDomainNavbarLink = {
    children?: Array<SiteDomainNavbarGroup>;
    external?: boolean;
    href: string;
    icon?: string;
    label: string;
};

export type SiteDomainTheme = {
    accentColor?: string;
    bannerUrl?: string;
    bgColor?: string;
    contentOpacity?: number;
    defaultThemeOverrides?: {
        [key: string]: string;
    };
    defaultThemePreset?: string;
    logoUrl?: string;
    primaryColor?: string;
    showDescriptionInBanner?: boolean;
    showLogoInBanner?: boolean;
    showNameInBanner?: boolean;
    textColor?: string;
    transparentBanner?: boolean;
};

export type SiteDomainWidget = {
    content?: string;
    enabled: boolean;
    killlistType?: string;
    type: 'mostValuable' | 'killList' | 'topCharacters' | 'topCorporations' | 'topAlliances' | 'topShips' | 'topSystems' | 'topRegions' | 'entityInfo' | 'textBlock' | 'campaigns';
};

export type SiteDomainWidgets = {
    columnRatio?: string;
    left: Array<SiteDomainWidget>;
    right: Array<SiteDomainWidget>;
    top: Array<SiteDomainWidget>;
};

export type AllianceBattlesResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        battle_id?: number;
        duration_minutes?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        end_time?: string;
        entity_isk_destroyed?: number;
        entity_kills?: string;
        entity_losses?: string;
        is_custom?: boolean;
        is_multi_party?: boolean;
        kill_count?: number;
        region_id?: number | null;
        region_name?: string | null;
        solar_system_id?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        start_time?: string;
        system_name?: string | null;
        system_security?: number | null;
        total_isk_destroyed?: number;
        [key: string]: unknown;
    }>;
    pagination: {
        hasMore: boolean;
        limit: number;
        page: number;
    };
};

export type AllianceCorporationsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        alliance_id?: number | null;
        corporation_id?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_founded?: string | null;
        faction_id?: number | null;
        member_count?: number | null;
        name?: string;
        ticker?: string;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type AllianceKillsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        attackers: Array<{
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_done?: number;
            faction_id?: number;
            final_blow?: boolean;
            security_status?: number;
            ship_type_id?: number;
            weapon_type_id?: number;
            [key: string]: unknown;
        }>;
        killmail_hash: string;
        killmail_id: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time: string;
        solar_system_id: number;
        victim: {
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_taken?: number;
            faction_id?: number;
            items?: Array<{
                flag?: number;
                item_type_id?: number;
                quantity_destroyed?: number;
                quantity_dropped?: number;
                singleton?: number;
                [key: string]: unknown;
            }>;
            position?: {
                x: number;
                y: number;
                z: number;
            };
            ship_type_id?: number;
            [key: string]: unknown;
        };
        war_id?: number;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type AllianceLossesResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        attackers: Array<{
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_done?: number;
            faction_id?: number;
            final_blow?: boolean;
            security_status?: number;
            ship_type_id?: number;
            weapon_type_id?: number;
            [key: string]: unknown;
        }>;
        killmail_hash: string;
        killmail_id: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time: string;
        solar_system_id: number;
        victim: {
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_taken?: number;
            faction_id?: number;
            items?: Array<{
                flag?: number;
                item_type_id?: number;
                quantity_destroyed?: number;
                quantity_dropped?: number;
                singleton?: number;
                [key: string]: unknown;
            }>;
            position?: {
                x: number;
                y: number;
                z: number;
            };
            ship_type_id?: number;
            [key: string]: unknown;
        };
        war_id?: number;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type AllianceMembersResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        alliance_id?: number | null;
        character_id?: number;
        corporation_id?: number | null;
        faction_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        last_active?: string | null;
        name?: string;
        security_status?: number | null;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type AllianceResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    alliance: {
        alliance_id?: number;
        corporation_count?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_founded?: string | null;
        faction_id?: number | null;
        member_count?: number | null;
        name?: string;
        ticker?: string;
        [key: string]: unknown;
    };
    recentStats: {
        isk_destroyed: number;
        isk_lost: number;
        kills: number;
        losses: number;
    };
    stats: {
        damage_dealt?: number;
        damage_taken?: number;
        efficiency?: number;
        final_blows?: number;
        isk_destroyed?: number;
        isk_efficiency?: number;
        isk_lost?: number;
        kills?: number;
        losses?: number;
        npc_losses?: number;
        points?: number;
        solo_kills?: number;
        [key: string]: unknown;
    };
};

export type AllianceStatsAlltimeResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    alliance_id?: number;
    character_id?: number;
    corporation_id?: number;
    damage_dealt?: number;
    damage_taken?: number;
    efficiency?: number;
    final_blows?: number;
    isk_destroyed?: number;
    isk_efficiency?: number;
    isk_lost?: number;
    kills?: number;
    losses?: number;
    npc_losses?: number;
    period?: string;
    points?: number;
    solo_kills?: number;
    topMembers?: Array<{
        id?: number;
        kills?: number;
        losses?: number;
        name?: string;
        [key: string]: unknown;
    }>;
    topShips?: Array<{
        kills?: number;
        losses?: number;
        ship_name?: string;
        ship_type_id?: number;
        [key: string]: unknown;
    }>;
    topSystems?: Array<{
        kills?: number;
        losses?: number;
        solar_system_id?: number;
        system_name?: string;
        [key: string]: unknown;
    }>;
    [key: string]: unknown;
};

export type AllianceStatsWeeklyResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    alliance_id?: number;
    character_id?: number;
    corporation_id?: number;
    damage_dealt?: number;
    damage_taken?: number;
    efficiency?: number;
    final_blows?: number;
    isk_destroyed?: number;
    isk_efficiency?: number;
    isk_lost?: number;
    kills?: number;
    losses?: number;
    npc_losses?: number;
    period?: string;
    points?: number;
    solo_kills?: number;
    topMembers?: Array<{
        id?: number;
        kills?: number;
        losses?: number;
        name?: string;
        [key: string]: unknown;
    }>;
    topShips?: Array<{
        kills?: number;
        losses?: number;
        ship_name?: string;
        ship_type_id?: number;
        [key: string]: unknown;
    }>;
    topSystems?: Array<{
        kills?: number;
        losses?: number;
        solar_system_id?: number;
        system_name?: string;
        [key: string]: unknown;
    }>;
    [key: string]: unknown;
};

export type AlliancesBatchStatsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    period: string;
    results: Array<{
        damage_dealt?: number;
        damage_taken?: number;
        efficiency?: number;
        final_blows?: number;
        id?: number;
        isk_destroyed?: number;
        isk_efficiency?: number;
        isk_lost?: number;
        kills?: number;
        losses?: number;
        name?: string;
        npc_losses?: number;
        points?: number;
        solo_kills?: number;
        topShips?: Array<{
            kills?: number;
            losses?: number;
            ship_name?: string;
            ship_type_id?: number;
            [key: string]: unknown;
        }>;
        [key: string]: unknown;
    }>;
};

export type AlliancesCountResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    count: number;
};

export type AlliancesResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        alliance_id?: number;
        corporation_count?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_founded?: string | null;
        faction_id?: number | null;
        member_count?: number | null;
        name?: string;
        ticker?: string;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type BattleResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    battle: {
        battle_id?: number;
        duration_minutes?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        end_time?: string;
        entity_isk_destroyed?: number;
        entity_kills?: string;
        entity_losses?: string;
        is_custom?: boolean;
        is_multi_party?: boolean;
        kill_count?: number;
        region_id?: number | null;
        region_name?: string | null;
        solar_system_id?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        start_time?: string;
        system_name?: string | null;
        system_security?: number | null;
        total_isk_destroyed?: number;
        [key: string]: unknown;
    };
    teams: Array<{
        members?: Array<{
            alliance_id?: number | null;
            corporation_id?: number;
            isk_destroyed?: number;
            isk_lost?: number;
            kills?: number;
            losses?: number;
            [key: string]: unknown;
        }>;
        team_index?: number;
        total_isk_destroyed?: number;
        total_isk_lost?: number;
        total_kills?: number;
        total_losses?: number;
        [key: string]: unknown;
    }>;
};

export type BattlesResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        battle_id?: number;
        duration_minutes?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        end_time?: string;
        entity_isk_destroyed?: number;
        entity_kills?: string;
        entity_losses?: string;
        is_custom?: boolean;
        is_multi_party?: boolean;
        kill_count?: number;
        region_id?: number | null;
        region_name?: string | null;
        solar_system_id?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        start_time?: string;
        system_name?: string | null;
        system_security?: number | null;
        total_isk_destroyed?: number;
        [key: string]: unknown;
    }>;
    pagination: {
        hasMore: boolean;
        limit: number;
        page: number;
    };
};

export type CharacterAnalyzeResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        average_gang_size: number;
        character_id: number;
        cyno_probability: number;
        efficiency: number;
        gang_probability: number;
        last_5_ships: Array<{
            kill_count?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_loss?: string | null;
            ship_name?: string | null;
            ship_type_id?: number;
            [key: string]: unknown;
        }>;
        total_kills: number;
        total_losses: number;
    }>;
};

export type CharacterIntelResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    awox_kills: number;
    bait: string;
    bait_count: number;
    bridge_score: number;
    capital_pilot: boolean;
    character_id: number;
    cyno_deaths: number;
    days: number;
    dominant_style: string;
    fc: {
        likelihood?: number;
        monitor_appearances?: number;
        [key: string]: unknown;
    };
    fleet_partners: Array<{
        [key: string]: unknown;
    }>;
    groups_flown_with: Array<{
        [key: string]: unknown;
    }>;
    is_logi: boolean;
    playstyle: {
        [key: string]: unknown;
    };
    ships_flown: Array<{
        [key: string]: unknown;
    }>;
    ships_lost: Array<{
        [key: string]: unknown;
    }>;
    tags: Array<string>;
    targets: Array<{
        [key: string]: unknown;
    }>;
};

export type CharacterKillsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        attackers: Array<{
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_done?: number;
            faction_id?: number;
            final_blow?: boolean;
            security_status?: number;
            ship_type_id?: number;
            weapon_type_id?: number;
            [key: string]: unknown;
        }>;
        killmail_hash: string;
        killmail_id: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time: string;
        solar_system_id: number;
        victim: {
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_taken?: number;
            faction_id?: number;
            items?: Array<{
                flag?: number;
                item_type_id?: number;
                quantity_destroyed?: number;
                quantity_dropped?: number;
                singleton?: number;
                [key: string]: unknown;
            }>;
            position?: {
                x: number;
                y: number;
                z: number;
            };
            ship_type_id?: number;
            [key: string]: unknown;
        };
        war_id?: number;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type CharacterLossesResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        attackers: Array<{
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_done?: number;
            faction_id?: number;
            final_blow?: boolean;
            security_status?: number;
            ship_type_id?: number;
            weapon_type_id?: number;
            [key: string]: unknown;
        }>;
        killmail_hash: string;
        killmail_id: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time: string;
        solar_system_id: number;
        victim: {
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_taken?: number;
            faction_id?: number;
            items?: Array<{
                flag?: number;
                item_type_id?: number;
                quantity_destroyed?: number;
                quantity_dropped?: number;
                singleton?: number;
                [key: string]: unknown;
            }>;
            position?: {
                x: number;
                y: number;
                z: number;
            };
            ship_type_id?: number;
            [key: string]: unknown;
        };
        war_id?: number;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type CharacterResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    character: {
        alliance_id?: number | null;
        character_id?: number;
        corporation_id?: number | null;
        faction_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        last_active?: string | null;
        name?: string;
        security_status?: number | null;
        [key: string]: unknown;
    };
    corporationHistory: Array<{
        kills?: number;
        losses?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        start_date?: string;
        [key: string]: unknown;
    }>;
    recentStats: {
        isk_destroyed: number;
        isk_lost: number;
        kills: number;
        losses: number;
    };
    stats: {
        damage_dealt?: number;
        damage_taken?: number;
        efficiency?: number;
        final_blows?: number;
        isk_destroyed?: number;
        isk_efficiency?: number;
        isk_lost?: number;
        kills?: number;
        losses?: number;
        npc_losses?: number;
        points?: number;
        solo_kills?: number;
        [key: string]: unknown;
    };
    topShips: Array<{
        kills?: number;
        losses?: number;
        ship_name?: string;
        ship_type_id?: number;
        [key: string]: unknown;
    }>;
    topSystems: Array<{
        kills?: number;
        losses?: number;
        solar_system_id?: number;
        system_name?: string;
        [key: string]: unknown;
    }>;
};

export type CharacterStatsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    alliance_id?: number;
    character_id?: number;
    corporation_id?: number;
    damage_dealt?: number;
    damage_taken?: number;
    efficiency?: number;
    final_blows?: number;
    isk_destroyed?: number;
    isk_efficiency?: number;
    isk_lost?: number;
    kills?: number;
    losses?: number;
    npc_losses?: number;
    period?: string;
    points?: number;
    solo_kills?: number;
    topMembers?: Array<{
        id?: number;
        kills?: number;
        losses?: number;
        name?: string;
        [key: string]: unknown;
    }>;
    topShips?: Array<{
        kills?: number;
        losses?: number;
        ship_name?: string;
        ship_type_id?: number;
        [key: string]: unknown;
    }>;
    topSystems?: Array<{
        kills?: number;
        losses?: number;
        solar_system_id?: number;
        system_name?: string;
        [key: string]: unknown;
    }>;
    [key: string]: unknown;
};

export type CharactersBatchStatsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    period: string;
    results: Array<{
        damage_dealt?: number;
        damage_taken?: number;
        efficiency?: number;
        final_blows?: number;
        id?: number;
        isk_destroyed?: number;
        isk_efficiency?: number;
        isk_lost?: number;
        kills?: number;
        losses?: number;
        name?: string;
        npc_losses?: number;
        points?: number;
        solo_kills?: number;
        topShips?: Array<{
            kills?: number;
            losses?: number;
            ship_name?: string;
            ship_type_id?: number;
            [key: string]: unknown;
        }>;
        [key: string]: unknown;
    }>;
};

export type CharactersCountResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    count: number;
};

export type CharactersResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        alliance_id?: number | null;
        character_id?: number;
        corporation_id?: number | null;
        faction_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        last_active?: string | null;
        name?: string;
        security_status?: number | null;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type CoalitionStatsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    clashed_systems: {
        count: number;
        system_ids: Array<number>;
    };
    daily: Array<{
        date?: string;
        [key: string]: unknown;
    }>;
    from: string;
    mode: string;
    period_days: number;
    sideA: {
        active_regions_count?: number;
        active_systems_count?: number;
        entity_counts?: {
            [key: string]: unknown;
        };
        label?: string;
        overall?: {
            damage_dealt?: number;
            damage_taken?: number;
            efficiency?: number;
            final_blows?: number;
            isk_destroyed?: number;
            isk_efficiency?: number;
            isk_lost?: number;
            kills?: number;
            losses?: number;
            npc_losses?: number;
            points?: number;
            solo_kills?: number;
            [key: string]: unknown;
        };
        top_ships_used?: Array<{
            kills?: number;
            losses?: number;
            ship_name?: string;
            ship_type_id?: number;
            [key: string]: unknown;
        }>;
        vs_opponent?: {
            damage_dealt?: number;
            damage_taken?: number;
            efficiency?: number;
            final_blows?: number;
            isk_destroyed?: number;
            isk_efficiency?: number;
            isk_lost?: number;
            kills?: number;
            losses?: number;
            npc_losses?: number;
            points?: number;
            solo_kills?: number;
            [key: string]: unknown;
        };
        [key: string]: unknown;
    };
    sideB: {
        active_regions_count?: number;
        active_systems_count?: number;
        entity_counts?: {
            [key: string]: unknown;
        };
        label?: string;
        overall?: {
            damage_dealt?: number;
            damage_taken?: number;
            efficiency?: number;
            final_blows?: number;
            isk_destroyed?: number;
            isk_efficiency?: number;
            isk_lost?: number;
            kills?: number;
            losses?: number;
            npc_losses?: number;
            points?: number;
            solo_kills?: number;
            [key: string]: unknown;
        };
        top_ships_used?: Array<{
            kills?: number;
            losses?: number;
            ship_name?: string;
            ship_type_id?: number;
            [key: string]: unknown;
        }>;
        vs_opponent?: {
            damage_dealt?: number;
            damage_taken?: number;
            efficiency?: number;
            final_blows?: number;
            isk_destroyed?: number;
            isk_efficiency?: number;
            isk_lost?: number;
            kills?: number;
            losses?: number;
            npc_losses?: number;
            points?: number;
            solo_kills?: number;
            [key: string]: unknown;
        };
        [key: string]: unknown;
    };
    to: string;
};

export type CorporationBattlesResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        battle_id?: number;
        duration_minutes?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        end_time?: string;
        entity_isk_destroyed?: number;
        entity_kills?: string;
        entity_losses?: string;
        is_custom?: boolean;
        is_multi_party?: boolean;
        kill_count?: number;
        region_id?: number | null;
        region_name?: string | null;
        solar_system_id?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        start_time?: string;
        system_name?: string | null;
        system_security?: number | null;
        total_isk_destroyed?: number;
        [key: string]: unknown;
    }>;
    pagination: {
        hasMore: boolean;
        limit: number;
        page: number;
    };
};

export type CorporationKillsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        attackers: Array<{
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_done?: number;
            faction_id?: number;
            final_blow?: boolean;
            security_status?: number;
            ship_type_id?: number;
            weapon_type_id?: number;
            [key: string]: unknown;
        }>;
        killmail_hash: string;
        killmail_id: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time: string;
        solar_system_id: number;
        victim: {
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_taken?: number;
            faction_id?: number;
            items?: Array<{
                flag?: number;
                item_type_id?: number;
                quantity_destroyed?: number;
                quantity_dropped?: number;
                singleton?: number;
                [key: string]: unknown;
            }>;
            position?: {
                x: number;
                y: number;
                z: number;
            };
            ship_type_id?: number;
            [key: string]: unknown;
        };
        war_id?: number;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type CorporationLossesResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        attackers: Array<{
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_done?: number;
            faction_id?: number;
            final_blow?: boolean;
            security_status?: number;
            ship_type_id?: number;
            weapon_type_id?: number;
            [key: string]: unknown;
        }>;
        killmail_hash: string;
        killmail_id: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time: string;
        solar_system_id: number;
        victim: {
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_taken?: number;
            faction_id?: number;
            items?: Array<{
                flag?: number;
                item_type_id?: number;
                quantity_destroyed?: number;
                quantity_dropped?: number;
                singleton?: number;
                [key: string]: unknown;
            }>;
            position?: {
                x: number;
                y: number;
                z: number;
            };
            ship_type_id?: number;
            [key: string]: unknown;
        };
        war_id?: number;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type CorporationMembersResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        alliance_id?: number | null;
        character_id?: number;
        corporation_id?: number | null;
        faction_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        last_active?: string | null;
        name?: string;
        security_status?: number | null;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type CorporationResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    allianceHistory: Array<{
        kills?: number;
        losses?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        start_date?: string;
        [key: string]: unknown;
    }>;
    corporation: {
        alliance_id?: number | null;
        corporation_id?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_founded?: string | null;
        faction_id?: number | null;
        member_count?: number | null;
        name?: string;
        ticker?: string;
        [key: string]: unknown;
    };
    recentStats: {
        isk_destroyed: number;
        isk_lost: number;
        kills: number;
        losses: number;
    };
    stats: {
        damage_dealt?: number;
        damage_taken?: number;
        efficiency?: number;
        final_blows?: number;
        isk_destroyed?: number;
        isk_efficiency?: number;
        isk_lost?: number;
        kills?: number;
        losses?: number;
        npc_losses?: number;
        points?: number;
        solo_kills?: number;
        [key: string]: unknown;
    };
};

export type CorporationStatsAlltimeResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    alliance_id?: number;
    character_id?: number;
    corporation_id?: number;
    damage_dealt?: number;
    damage_taken?: number;
    efficiency?: number;
    final_blows?: number;
    isk_destroyed?: number;
    isk_efficiency?: number;
    isk_lost?: number;
    kills?: number;
    losses?: number;
    npc_losses?: number;
    period?: string;
    points?: number;
    solo_kills?: number;
    topMembers?: Array<{
        id?: number;
        kills?: number;
        losses?: number;
        name?: string;
        [key: string]: unknown;
    }>;
    topShips?: Array<{
        kills?: number;
        losses?: number;
        ship_name?: string;
        ship_type_id?: number;
        [key: string]: unknown;
    }>;
    topSystems?: Array<{
        kills?: number;
        losses?: number;
        solar_system_id?: number;
        system_name?: string;
        [key: string]: unknown;
    }>;
    [key: string]: unknown;
};

export type CorporationStatsWeeklyResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    alliance_id?: number;
    character_id?: number;
    corporation_id?: number;
    damage_dealt?: number;
    damage_taken?: number;
    efficiency?: number;
    final_blows?: number;
    isk_destroyed?: number;
    isk_efficiency?: number;
    isk_lost?: number;
    kills?: number;
    losses?: number;
    npc_losses?: number;
    period?: string;
    points?: number;
    solo_kills?: number;
    topMembers?: Array<{
        id?: number;
        kills?: number;
        losses?: number;
        name?: string;
        [key: string]: unknown;
    }>;
    topShips?: Array<{
        kills?: number;
        losses?: number;
        ship_name?: string;
        ship_type_id?: number;
        [key: string]: unknown;
    }>;
    topSystems?: Array<{
        kills?: number;
        losses?: number;
        solar_system_id?: number;
        system_name?: string;
        [key: string]: unknown;
    }>;
    [key: string]: unknown;
};

export type CorporationsBatchStatsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    period: string;
    results: Array<{
        damage_dealt?: number;
        damage_taken?: number;
        efficiency?: number;
        final_blows?: number;
        id?: number;
        isk_destroyed?: number;
        isk_efficiency?: number;
        isk_lost?: number;
        kills?: number;
        losses?: number;
        name?: string;
        npc_losses?: number;
        points?: number;
        solo_kills?: number;
        topShips?: Array<{
            kills?: number;
            losses?: number;
            ship_name?: string;
            ship_type_id?: number;
            [key: string]: unknown;
        }>;
        [key: string]: unknown;
    }>;
};

export type CorporationsCountResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    count: number;
};

export type CorporationsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        alliance_id?: number | null;
        corporation_id?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_founded?: string | null;
        faction_id?: number | null;
        member_count?: number | null;
        name?: string;
        ticker?: string;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type FeedIndexResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    description: string;
    endpoints: {
        [key: string]: {
            description?: string;
            example?: string;
            headers?: {
                [key: string]: string;
            };
            params?: {
                [key: string]: string;
            };
            [key: string]: unknown;
        };
    };
    name: string;
    note: string;
    topics: {
        [key: string]: Array<string>;
    };
};

export type FeedPollResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        data: {
            attackers: Array<{
                alliance_id?: number;
                character_id?: number;
                corporation_id?: number;
                damage_done?: number;
                faction_id?: number;
                final_blow?: boolean;
                security_status?: number;
                ship_type_id?: number;
                weapon_type_id?: number;
                [key: string]: unknown;
            }>;
            killmail_hash: string;
            killmail_id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            killmail_time: string;
            solar_system_id: number;
            victim: {
                alliance_id?: number;
                character_id?: number;
                corporation_id?: number;
                damage_taken?: number;
                faction_id?: number;
                items?: Array<{
                    flag?: number;
                    item_type_id?: number;
                    quantity_destroyed?: number;
                    quantity_dropped?: number;
                    singleton?: number;
                    [key: string]: unknown;
                }>;
                position?: {
                    x: number;
                    y: number;
                    z: number;
                };
                ship_type_id?: number;
                [key: string]: unknown;
            };
            war_id?: number;
        };
        killmail_hash: string;
        killmail_id: number;
        seq: number;
    }>;
    hasMore: boolean;
    last: string;
    latest: number | null;
    next: string;
};

export type FeedStatusResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    clients: number;
    latestSeq: number | null;
    status: string;
};

export type GlobalStatsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    entries: Array<{
        efficiency?: number;
        id?: number;
        isk_destroyed?: number;
        isk_lost?: number;
        killmail_hash?: string;
        killmail_id?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time?: string;
        kills?: number;
        losses?: number;
        name?: string;
        sec?: number;
        ticker?: string;
        total_value?: number;
        [key: string]: unknown;
    }>;
};

export type HealthResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    ok: boolean;
    /**
     * UTC timestamp with millisecond precision.
     */
    timestamp: string;
};

export type HistoryDateResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    /**
     * Map of killmail ID to killmail hash.
     */
    data: {
        [key: string]: string;
    };
};

export type HistoryLatestResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    /**
     * Map of killmail ID to killmail hash.
     */
    data: {
        [key: string]: string;
    };
};

export type HistoryResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        count?: number;
        date?: string;
        [key: string]: unknown;
    }>;
};

export type KillmailEsiResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    attackers: Array<{
        alliance_id?: number;
        character_id?: number;
        corporation_id?: number;
        damage_done?: number;
        faction_id?: number;
        final_blow?: boolean;
        security_status?: number;
        ship_type_id?: number;
        weapon_type_id?: number;
        [key: string]: unknown;
    }>;
    killmail_hash: string;
    killmail_id: number;
    /**
     * UTC timestamp with millisecond precision.
     */
    killmail_time: string;
    solar_system_id: number;
    victim: {
        alliance_id?: number;
        character_id?: number;
        corporation_id?: number;
        damage_taken?: number;
        faction_id?: number;
        items?: Array<{
            flag?: number;
            item_type_id?: number;
            quantity_destroyed?: number;
            quantity_dropped?: number;
            singleton?: number;
            [key: string]: unknown;
        }>;
        position?: {
            x: number;
            y: number;
            z: number;
        };
        ship_type_id?: number;
        [key: string]: unknown;
    };
    war_id?: number;
};

export type KillmailFittingResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    cargo?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    drone?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    fighter?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    fighter_bay?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    fleet?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    high?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    killmail_id: number;
    low?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    mid?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    other?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    rig?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    service?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    ship: {
        name: string;
        type_id: number;
    };
    specialized?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    subsystem?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
};

export type KillmailResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    attacker_count?: number;
    attackers: Array<{
        alliance_id?: number | null;
        character_id?: number | null;
        corporation_id?: number | null;
        damage_done?: number;
        final_blow?: boolean;
        ship_type_id?: number | null;
        weapon_type_id?: number | null;
        [key: string]: unknown;
    }>;
    constellation_id?: number | null;
    constellation_name?: string | null;
    destroyed_value?: number;
    dropped_value?: number;
    fitted_value?: number;
    is_npc?: boolean;
    is_solo?: boolean;
    items: Array<{
        item_index?: number;
        price?: number;
        quantity_destroyed?: number;
        quantity_dropped?: number;
        slot?: string;
        total_value?: number;
        type_id?: number;
        type_name?: string | null;
        [key: string]: unknown;
    }>;
    killmail_hash: string;
    killmail_id: number;
    /**
     * UTC timestamp with millisecond precision.
     */
    killmail_time: string;
    location?: {
        distance?: number;
        group_id?: number;
        item_id?: number;
        item_name?: string;
        type_id?: number;
        [key: string]: unknown;
    } | null;
    points?: number;
    position_x?: number | null;
    position_y?: number | null;
    position_z?: number | null;
    region_id?: number | null;
    region_name?: string | null;
    siblings: Array<{
        killmail_id?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time?: string;
        ship_group_id?: number | null;
        ship_name?: string | null;
        ship_type_id?: number | null;
        total_value?: number;
        [key: string]: unknown;
    }>;
    solar_system_id: number;
    solar_system_name?: string | null;
    solar_system_security?: number | null;
    total_damage?: number;
    total_value: number;
    victim: {
        alliance_id?: number | null;
        alliance_name?: string | null;
        character_id?: number | null;
        character_name?: string | null;
        corporation_id?: number | null;
        corporation_name?: string | null;
        corporation_palette?: string | null;
        damage_taken?: number;
        ship_group_id?: number | null;
        ship_group_name?: string | null;
        ship_market_path?: string | null;
        ship_name?: string | null;
        ship_price?: number;
        ship_type_id?: number | null;
        [key: string]: unknown;
    };
};

export type KillmailSearchResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        attacker_count: number;
        final_blow_alliance_id: number | null;
        final_blow_alliance_name: string | null;
        final_blow_character_id: number | null;
        final_blow_character_name: string | null;
        final_blow_corporation_id: number | null;
        final_blow_corporation_name: string | null;
        final_blow_ship_name: string | null;
        final_blow_ship_type_id: number | null;
        is_npc: boolean;
        is_solo: boolean;
        killmail_hash: string;
        killmail_id: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time: string;
        meta_group_id: number | null;
        region_id: number | null;
        region_name: string | null;
        ship_group_name: string | null;
        ship_market_path: string | null;
        ship_name: string | null;
        ship_type_id: number | null;
        solar_system_id: number;
        solar_system_name: string | null;
        solar_system_security: number | null;
        total_value: number;
        victim_alliance_id: number | null;
        victim_alliance_name: string | null;
        victim_character_id: number | null;
        victim_character_name: string | null;
        victim_corporation_id: number | null;
        victim_corporation_name: string | null;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type KillmailsCountResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    count: number;
};

export type KillmailsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        attacker_count: number;
        final_blow_alliance_id: number | null;
        final_blow_alliance_name: string | null;
        final_blow_character_id: number | null;
        final_blow_character_name: string | null;
        final_blow_corporation_id: number | null;
        final_blow_corporation_name: string | null;
        final_blow_ship_name: string | null;
        final_blow_ship_type_id: number | null;
        is_npc: boolean;
        is_solo: boolean;
        killmail_hash: string;
        killmail_id: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time: string;
        meta_group_id: number | null;
        region_id: number | null;
        region_name: string | null;
        ship_group_name: string | null;
        ship_market_path: string | null;
        ship_name: string | null;
        ship_type_id: number | null;
        solar_system_id: number;
        solar_system_name: string | null;
        solar_system_security: number | null;
        total_value: number;
        victim_alliance_id: number | null;
        victim_alliance_name: string | null;
        victim_character_id: number | null;
        victim_character_name: string | null;
        victim_corporation_id: number | null;
        victim_corporation_name: string | null;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type LocationResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    nearest: {
        distance?: number;
        group_id?: number;
        item_id?: number;
        item_name?: string;
        type_id?: number;
        [key: string]: unknown;
    } | null;
    system_id: number;
    x: number;
    y: number;
    z: number;
};

export type ResolveResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    resolved: number;
    results: Array<{
        id?: number | null;
        name?: string;
        resolved_name?: string | null;
        [key: string]: unknown;
    }>;
    type: string;
    unresolved: number;
};

/**
 * bloodline
 *
 * Static data record. Known stable fields are documented; source-table-specific fields are preserved as additional properties.
 */
export type SdeBloodlineResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    alliance_id?: number | null;
    attribute_id?: number | null;
    bloodline_id?: number | null;
    category_id?: number | null;
    corporation_id?: number | null;
    /**
     * UTC timestamp with millisecond precision.
     */
    date_added?: string;
    description?: string | null;
    effect_id?: number | null;
    faction_id?: number | null;
    flag_id?: number | null;
    id?: number;
    item_id?: number | null;
    market_group_id?: number | null;
    material_type_id?: number | null;
    meta_group_id?: number | null;
    name?: string;
    operation?: string;
    published?: boolean;
    race_id?: number | null;
    solar_system_id?: number | null;
    system_id?: number | null;
    type_id?: number | null;
    [key: string]: unknown;
};

export type SdeBloodlinesResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
};

export type SdeCategoriesResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
};

/**
 * category
 *
 * Static data record. Known stable fields are documented; source-table-specific fields are preserved as additional properties.
 */
export type SdeCategoryResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    alliance_id?: number | null;
    attribute_id?: number | null;
    bloodline_id?: number | null;
    category_id?: number | null;
    corporation_id?: number | null;
    /**
     * UTC timestamp with millisecond precision.
     */
    date_added?: string;
    description?: string | null;
    effect_id?: number | null;
    faction_id?: number | null;
    flag_id?: number | null;
    id?: number;
    item_id?: number | null;
    market_group_id?: number | null;
    material_type_id?: number | null;
    meta_group_id?: number | null;
    name?: string;
    operation?: string;
    published?: boolean;
    race_id?: number | null;
    solar_system_id?: number | null;
    system_id?: number | null;
    type_id?: number | null;
    [key: string]: unknown;
};

/**
 * celestial
 *
 * Static data record. Known stable fields are documented; source-table-specific fields are preserved as additional properties.
 */
export type SdeCelestialResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    alliance_id?: number | null;
    attribute_id?: number | null;
    bloodline_id?: number | null;
    category_id?: number | null;
    corporation_id?: number | null;
    /**
     * UTC timestamp with millisecond precision.
     */
    date_added?: string;
    description?: string | null;
    effect_id?: number | null;
    faction_id?: number | null;
    flag_id?: number | null;
    id?: number;
    item_id?: number | null;
    market_group_id?: number | null;
    material_type_id?: number | null;
    meta_group_id?: number | null;
    name?: string;
    operation?: string;
    published?: boolean;
    race_id?: number | null;
    solar_system_id?: number | null;
    system_id?: number | null;
    type_id?: number | null;
    [key: string]: unknown;
};

export type SdeConstellationResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    constellation_id?: number;
    constellation_name?: string;
    faction_id?: number | null;
    region_id?: number;
    region_name?: string | null;
    [key: string]: unknown;
};

export type SdeConstellationsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        constellation_id?: number;
        constellation_name?: string;
        faction_id?: number | null;
        region_id?: number;
        region_name?: string | null;
        [key: string]: unknown;
    }>;
};

export type SdeCustomPricesResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    count: number;
    data: Array<{
        price?: number;
        type_id?: number;
        type_name?: string | null;
        valid_until?: string;
        [key: string]: unknown;
    }>;
};

/**
 * faction
 *
 * Static data record. Known stable fields are documented; source-table-specific fields are preserved as additional properties.
 */
export type SdeFactionResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    alliance_id?: number | null;
    attribute_id?: number | null;
    bloodline_id?: number | null;
    category_id?: number | null;
    corporation_id?: number | null;
    /**
     * UTC timestamp with millisecond precision.
     */
    date_added?: string;
    description?: string | null;
    effect_id?: number | null;
    faction_id?: number | null;
    flag_id?: number | null;
    id?: number;
    item_id?: number | null;
    market_group_id?: number | null;
    material_type_id?: number | null;
    meta_group_id?: number | null;
    name?: string;
    operation?: string;
    published?: boolean;
    race_id?: number | null;
    solar_system_id?: number | null;
    system_id?: number | null;
    type_id?: number | null;
    [key: string]: unknown;
};

export type SdeFactionsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
};

export type SdeFlagsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
};

export type SdeGroupResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    category_id?: number;
    category_name?: string | null;
    group_id?: number;
    icon_id?: number | null;
    name?: string;
    published?: boolean;
    [key: string]: unknown;
};

export type SdeGroupsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        category_id?: number;
        category_name?: string | null;
        group_id?: number;
        icon_id?: number | null;
        name?: string;
        published?: boolean;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

/**
 * market-group
 *
 * Static data record. Known stable fields are documented; source-table-specific fields are preserved as additional properties.
 */
export type SdeMarketGroupResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    alliance_id?: number | null;
    attribute_id?: number | null;
    bloodline_id?: number | null;
    category_id?: number | null;
    corporation_id?: number | null;
    /**
     * UTC timestamp with millisecond precision.
     */
    date_added?: string;
    description?: string | null;
    effect_id?: number | null;
    faction_id?: number | null;
    flag_id?: number | null;
    id?: number;
    item_id?: number | null;
    market_group_id?: number | null;
    material_type_id?: number | null;
    meta_group_id?: number | null;
    name?: string;
    operation?: string;
    published?: boolean;
    race_id?: number | null;
    solar_system_id?: number | null;
    system_id?: number | null;
    type_id?: number | null;
    [key: string]: unknown;
};

export type SdeMarketGroupsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
};

export type SdeMetaGroupsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
};

/**
 * npc-corporation
 *
 * Static data record. Known stable fields are documented; source-table-specific fields are preserved as additional properties.
 */
export type SdeNpcCorporationResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    alliance_id?: number | null;
    attribute_id?: number | null;
    bloodline_id?: number | null;
    category_id?: number | null;
    corporation_id?: number | null;
    /**
     * UTC timestamp with millisecond precision.
     */
    date_added?: string;
    description?: string | null;
    effect_id?: number | null;
    faction_id?: number | null;
    flag_id?: number | null;
    id?: number;
    item_id?: number | null;
    market_group_id?: number | null;
    material_type_id?: number | null;
    meta_group_id?: number | null;
    name?: string;
    operation?: string;
    published?: boolean;
    race_id?: number | null;
    solar_system_id?: number | null;
    system_id?: number | null;
    type_id?: number | null;
    [key: string]: unknown;
};

export type SdeNpcCorporationsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
};

export type SdePricesResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    prices: Array<{
        average?: number;
        date?: string;
        highest?: number;
        lowest?: number;
        order_count?: number;
        region_id?: number;
        type_id?: number;
        volume?: number;
        [key: string]: unknown;
    }>;
    region_id: number;
    type_id: number;
};

/**
 * race
 *
 * Static data record. Known stable fields are documented; source-table-specific fields are preserved as additional properties.
 */
export type SdeRaceResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    alliance_id?: number | null;
    attribute_id?: number | null;
    bloodline_id?: number | null;
    category_id?: number | null;
    corporation_id?: number | null;
    /**
     * UTC timestamp with millisecond precision.
     */
    date_added?: string;
    description?: string | null;
    effect_id?: number | null;
    faction_id?: number | null;
    flag_id?: number | null;
    id?: number;
    item_id?: number | null;
    market_group_id?: number | null;
    material_type_id?: number | null;
    meta_group_id?: number | null;
    name?: string;
    operation?: string;
    published?: boolean;
    race_id?: number | null;
    solar_system_id?: number | null;
    system_id?: number | null;
    type_id?: number | null;
    [key: string]: unknown;
};

export type SdeRacesResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
};

export type SdeRegionKillsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        attacker_count: number;
        final_blow_alliance_id: number | null;
        final_blow_alliance_name: string | null;
        final_blow_character_id: number | null;
        final_blow_character_name: string | null;
        final_blow_corporation_id: number | null;
        final_blow_corporation_name: string | null;
        final_blow_ship_name: string | null;
        final_blow_ship_type_id: number | null;
        is_npc: boolean;
        is_solo: boolean;
        killmail_hash: string;
        killmail_id: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time: string;
        meta_group_id: number | null;
        region_id: number | null;
        region_name: string | null;
        ship_group_name: string | null;
        ship_market_path: string | null;
        ship_name: string | null;
        ship_type_id: number | null;
        solar_system_id: number;
        solar_system_name: string | null;
        solar_system_security: number | null;
        total_value: number;
        victim_alliance_id: number | null;
        victim_alliance_name: string | null;
        victim_character_id: number | null;
        victim_character_name: string | null;
        victim_corporation_id: number | null;
        victim_corporation_name: string | null;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type SdeRegionResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    description?: string | null;
    faction_id?: number | null;
    faction_name?: string | null;
    name?: string;
    region_id?: number;
    [key: string]: unknown;
};

export type SdeRegionsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        description?: string | null;
        faction_id?: number | null;
        faction_name?: string | null;
        name?: string;
        region_id?: number;
        [key: string]: unknown;
    }>;
};

export type SdeSovereigntyHistoryResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    history: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
    system_id: number;
};

export type SdeSovereigntyResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
};

/**
 * sovereignty-system
 *
 * Static data record. Known stable fields are documented; source-table-specific fields are preserved as additional properties.
 */
export type SdeSovereigntySystemResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    alliance_id?: number | null;
    attribute_id?: number | null;
    bloodline_id?: number | null;
    category_id?: number | null;
    corporation_id?: number | null;
    /**
     * UTC timestamp with millisecond precision.
     */
    date_added?: string;
    description?: string | null;
    effect_id?: number | null;
    faction_id?: number | null;
    flag_id?: number | null;
    id?: number;
    item_id?: number | null;
    market_group_id?: number | null;
    material_type_id?: number | null;
    meta_group_id?: number | null;
    name?: string;
    operation?: string;
    published?: boolean;
    race_id?: number | null;
    solar_system_id?: number | null;
    system_id?: number | null;
    type_id?: number | null;
    [key: string]: unknown;
};

/**
 * station-operation
 *
 * Static data record. Known stable fields are documented; source-table-specific fields are preserved as additional properties.
 */
export type SdeStationOperationResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    alliance_id?: number | null;
    attribute_id?: number | null;
    bloodline_id?: number | null;
    category_id?: number | null;
    corporation_id?: number | null;
    /**
     * UTC timestamp with millisecond precision.
     */
    date_added?: string;
    description?: string | null;
    effect_id?: number | null;
    faction_id?: number | null;
    flag_id?: number | null;
    id?: number;
    item_id?: number | null;
    market_group_id?: number | null;
    material_type_id?: number | null;
    meta_group_id?: number | null;
    name?: string;
    operation?: string;
    published?: boolean;
    race_id?: number | null;
    solar_system_id?: number | null;
    system_id?: number | null;
    type_id?: number | null;
    [key: string]: unknown;
};

export type SdeStationOperationsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
};

export type SdeStationResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    corporation_id?: number;
    region_id?: number;
    security?: number;
    solar_system_id?: number;
    station_id?: number;
    station_name?: string;
    type_id?: number;
    [key: string]: unknown;
};

export type SdeStationsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        corporation_id?: number;
        region_id?: number;
        security?: number;
        solar_system_id?: number;
        station_id?: number;
        station_name?: string;
        type_id?: number;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type SdeStructureResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    name?: string;
    owner_id?: number;
    region_id?: number;
    solar_system_id?: number;
    structure_id?: number;
    type_id?: number;
    [key: string]: unknown;
};

export type SdeStructuresResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        name?: string;
        owner_id?: number;
        region_id?: number;
        solar_system_id?: number;
        structure_id?: number;
        type_id?: number;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type SdeSystemCelestialsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    celestials: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
    solar_system_id: number;
};

export type SdeSystemJumpsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    jumps: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
    solar_system_id: number;
};

export type SdeSystemKillsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        attacker_count: number;
        final_blow_alliance_id: number | null;
        final_blow_alliance_name: string | null;
        final_blow_character_id: number | null;
        final_blow_character_name: string | null;
        final_blow_corporation_id: number | null;
        final_blow_corporation_name: string | null;
        final_blow_ship_name: string | null;
        final_blow_ship_type_id: number | null;
        is_npc: boolean;
        is_solo: boolean;
        killmail_hash: string;
        killmail_id: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time: string;
        meta_group_id: number | null;
        region_id: number | null;
        region_name: string | null;
        ship_group_name: string | null;
        ship_market_path: string | null;
        ship_name: string | null;
        ship_type_id: number | null;
        solar_system_id: number;
        solar_system_name: string | null;
        solar_system_security: number | null;
        total_value: number;
        victim_alliance_id: number | null;
        victim_alliance_name: string | null;
        victim_character_id: number | null;
        victim_character_name: string | null;
        victim_corporation_id: number | null;
        victim_corporation_name: string | null;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type SdeSystemResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    constellation_id?: number;
    constellation_name?: string | null;
    region_id?: number;
    region_name?: string | null;
    security?: number;
    security_class?: string | null;
    solar_system_id?: number;
    system_name?: string;
    [key: string]: unknown;
};

export type SdeSystemsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        constellation_id?: number;
        constellation_name?: string | null;
        region_id?: number;
        region_name?: string | null;
        security?: number;
        security_class?: string | null;
        solar_system_id?: number;
        system_name?: string;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type SdeTypeDogmaResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    attributes: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
    effects: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
    type_id: number;
};

export type SdeTypeInsuranceResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    levels: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
    type_id: number;
};

export type SdeTypeMaterialsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    materials: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
    type_id: number;
};

export type SdeTypeResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    base_price?: number | null;
    capacity?: number | null;
    category_id?: number;
    category_name?: string | null;
    description?: string | null;
    group_id?: number;
    group_name?: string | null;
    market_group_id?: number | null;
    mass?: number | null;
    meta_group_id?: number | null;
    name?: string;
    published?: boolean;
    type_id?: number;
    volume?: number | null;
    [key: string]: unknown;
};

export type SdeTypesResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        base_price?: number | null;
        capacity?: number | null;
        category_id?: number;
        category_name?: string | null;
        description?: string | null;
        group_id?: number;
        group_name?: string | null;
        market_group_id?: number | null;
        mass?: number | null;
        meta_group_id?: number | null;
        name?: string;
        published?: boolean;
        type_id?: number;
        volume?: number | null;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type SearchResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    entityCounts: {
        [key: string]: number;
    };
    hits: Array<{
        alliance_id?: number | null;
        alliance_name?: string;
        alliance_ticker?: string;
        corporation_id?: number | null;
        corporation_name?: string;
        corporation_ticker?: string;
        id?: string;
        name?: string;
        ticker?: string | null;
        type?: string;
        [key: string]: unknown;
    }>;
    processingTimeMs: number;
    query: string;
    total: number;
};

export type ShipFittingsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    families: Array<{
        canonical_fit_hash?: string;
        canonical_uses?: number;
        family_hash?: string;
        fit_cost?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        last_used?: string;
        modules?: Array<{
            [key: string]: unknown;
        }>;
        top_alliances?: Array<{
            [key: string]: unknown;
        }>;
        total_uses?: number;
        variant_count?: number;
        [key: string]: unknown;
    }>;
    hull_cost: number;
    is_rare_hull: boolean;
    module_filter: Array<number>;
    ship_type_id: number;
    window_days: number;
};

export type WarResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    stats: {
        top_ships: Array<{
            kills?: number;
            losses?: number;
            ship_name?: string;
            ship_type_id?: number;
            [key: string]: unknown;
        }>;
        total_kills: number;
        total_value: number;
    };
    war: {
        aggressor: {
            id: number;
            isk_destroyed: number;
            name: string;
            ships_killed: number;
            ticker: string;
            type: string;
        };
        allies: Array<{
            id: number;
            name: string;
            type: string;
        }>;
        /**
         * UTC timestamp with millisecond precision.
         */
        declared: string;
        defender: {
            id: number;
            isk_destroyed: number;
            name: string;
            ships_killed: number;
            ticker: string;
            type: string;
        };
        /**
         * UTC timestamp with millisecond precision.
         */
        finished: string | null;
        mutual: boolean;
        open_for_allies: boolean;
        /**
         * UTC timestamp with millisecond precision.
         */
        retracted: string | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        started: string | null;
        war_id: number;
    };
};

export type WarsResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
        aggressor_alliance_id?: number | null;
        aggressor_corporation_id?: number | null;
        aggressor_ships_killed?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        declared?: string;
        defender_alliance_id?: number | null;
        defender_corporation_id?: number | null;
        defender_ships_killed?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        finished?: string | null;
        mutual?: boolean;
        /**
         * UTC timestamp with millisecond precision.
         */
        started?: string | null;
        war_id?: number;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type AnnouncementDismissalResponseWritable = {
    ok: boolean;
};

export type AnnouncementsResponseWritable = {
    announcements: Array<Announcement>;
};

export type DismissedAnnouncementIdsResponseWritable = {
    dismissedIds: Array<number>;
};

export type ErrorModelWritable = {
    /**
     * A human-readable explanation specific to this occurrence of the problem.
     */
    detail?: string;
    /**
     * Optional list of individual error details
     */
    errors?: Array<ErrorDetail> | null;
    /**
     * A URI reference that identifies the specific occurrence of the problem.
     */
    instance?: string;
    /**
     * HTTP status code
     */
    status?: number;
    /**
     * A short, human-readable summary of the problem type. This value should not change between occurrences of the error.
     */
    title?: string;
    /**
     * A URI reference to human-readable documentation for the error.
     */
    type?: string;
};

export type ImagesOverviewResponseWritable = {
    routes: Array<string> | null;
    service: string;
};

export type SiteConfigurationResponseWritable = {
    domain: SiteDomainConfiguration | null;
    isDomainHost: boolean;
};

export type AllianceBattlesResponseWritable = {
    data: Array<{
        battle_id?: number;
        duration_minutes?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        end_time?: string;
        entity_isk_destroyed?: number;
        entity_kills?: string;
        entity_losses?: string;
        is_custom?: boolean;
        is_multi_party?: boolean;
        kill_count?: number;
        region_id?: number | null;
        region_name?: string | null;
        solar_system_id?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        start_time?: string;
        system_name?: string | null;
        system_security?: number | null;
        total_isk_destroyed?: number;
        [key: string]: unknown;
    }>;
    pagination: {
        hasMore: boolean;
        limit: number;
        page: number;
    };
};

export type AllianceCorporationsResponseWritable = {
    data: Array<{
        alliance_id?: number | null;
        corporation_id?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_founded?: string | null;
        faction_id?: number | null;
        member_count?: number | null;
        name?: string;
        ticker?: string;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type AllianceKillsResponseWritable = {
    data: Array<{
        attackers: Array<{
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_done?: number;
            faction_id?: number;
            final_blow?: boolean;
            security_status?: number;
            ship_type_id?: number;
            weapon_type_id?: number;
            [key: string]: unknown;
        }>;
        killmail_hash: string;
        killmail_id: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time: string;
        solar_system_id: number;
        victim: {
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_taken?: number;
            faction_id?: number;
            items?: Array<{
                flag?: number;
                item_type_id?: number;
                quantity_destroyed?: number;
                quantity_dropped?: number;
                singleton?: number;
                [key: string]: unknown;
            }>;
            position?: {
                x: number;
                y: number;
                z: number;
            };
            ship_type_id?: number;
            [key: string]: unknown;
        };
        war_id?: number;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type AllianceLossesResponseWritable = {
    data: Array<{
        attackers: Array<{
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_done?: number;
            faction_id?: number;
            final_blow?: boolean;
            security_status?: number;
            ship_type_id?: number;
            weapon_type_id?: number;
            [key: string]: unknown;
        }>;
        killmail_hash: string;
        killmail_id: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time: string;
        solar_system_id: number;
        victim: {
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_taken?: number;
            faction_id?: number;
            items?: Array<{
                flag?: number;
                item_type_id?: number;
                quantity_destroyed?: number;
                quantity_dropped?: number;
                singleton?: number;
                [key: string]: unknown;
            }>;
            position?: {
                x: number;
                y: number;
                z: number;
            };
            ship_type_id?: number;
            [key: string]: unknown;
        };
        war_id?: number;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type AllianceMembersResponseWritable = {
    data: Array<{
        alliance_id?: number | null;
        character_id?: number;
        corporation_id?: number | null;
        faction_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        last_active?: string | null;
        name?: string;
        security_status?: number | null;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type AllianceResponseWritable = {
    alliance: {
        alliance_id?: number;
        corporation_count?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_founded?: string | null;
        faction_id?: number | null;
        member_count?: number | null;
        name?: string;
        ticker?: string;
        [key: string]: unknown;
    };
    recentStats: {
        isk_destroyed: number;
        isk_lost: number;
        kills: number;
        losses: number;
    };
    stats: {
        damage_dealt?: number;
        damage_taken?: number;
        efficiency?: number;
        final_blows?: number;
        isk_destroyed?: number;
        isk_efficiency?: number;
        isk_lost?: number;
        kills?: number;
        losses?: number;
        npc_losses?: number;
        points?: number;
        solo_kills?: number;
        [key: string]: unknown;
    };
};

export type AllianceStatsAlltimeResponseWritable = {
    alliance_id?: number;
    character_id?: number;
    corporation_id?: number;
    damage_dealt?: number;
    damage_taken?: number;
    efficiency?: number;
    final_blows?: number;
    isk_destroyed?: number;
    isk_efficiency?: number;
    isk_lost?: number;
    kills?: number;
    losses?: number;
    npc_losses?: number;
    period?: string;
    points?: number;
    solo_kills?: number;
    topMembers?: Array<{
        id?: number;
        kills?: number;
        losses?: number;
        name?: string;
        [key: string]: unknown;
    }>;
    topShips?: Array<{
        kills?: number;
        losses?: number;
        ship_name?: string;
        ship_type_id?: number;
        [key: string]: unknown;
    }>;
    topSystems?: Array<{
        kills?: number;
        losses?: number;
        solar_system_id?: number;
        system_name?: string;
        [key: string]: unknown;
    }>;
    [key: string]: unknown;
};

export type AllianceStatsWeeklyResponseWritable = {
    alliance_id?: number;
    character_id?: number;
    corporation_id?: number;
    damage_dealt?: number;
    damage_taken?: number;
    efficiency?: number;
    final_blows?: number;
    isk_destroyed?: number;
    isk_efficiency?: number;
    isk_lost?: number;
    kills?: number;
    losses?: number;
    npc_losses?: number;
    period?: string;
    points?: number;
    solo_kills?: number;
    topMembers?: Array<{
        id?: number;
        kills?: number;
        losses?: number;
        name?: string;
        [key: string]: unknown;
    }>;
    topShips?: Array<{
        kills?: number;
        losses?: number;
        ship_name?: string;
        ship_type_id?: number;
        [key: string]: unknown;
    }>;
    topSystems?: Array<{
        kills?: number;
        losses?: number;
        solar_system_id?: number;
        system_name?: string;
        [key: string]: unknown;
    }>;
    [key: string]: unknown;
};

export type AlliancesBatchStatsResponseWritable = {
    period: string;
    results: Array<{
        damage_dealt?: number;
        damage_taken?: number;
        efficiency?: number;
        final_blows?: number;
        id?: number;
        isk_destroyed?: number;
        isk_efficiency?: number;
        isk_lost?: number;
        kills?: number;
        losses?: number;
        name?: string;
        npc_losses?: number;
        points?: number;
        solo_kills?: number;
        topShips?: Array<{
            kills?: number;
            losses?: number;
            ship_name?: string;
            ship_type_id?: number;
            [key: string]: unknown;
        }>;
        [key: string]: unknown;
    }>;
};

export type AlliancesCountResponseWritable = {
    count: number;
};

export type AlliancesResponseWritable = {
    data: Array<{
        alliance_id?: number;
        corporation_count?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_founded?: string | null;
        faction_id?: number | null;
        member_count?: number | null;
        name?: string;
        ticker?: string;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type BattleResponseWritable = {
    battle: {
        battle_id?: number;
        duration_minutes?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        end_time?: string;
        entity_isk_destroyed?: number;
        entity_kills?: string;
        entity_losses?: string;
        is_custom?: boolean;
        is_multi_party?: boolean;
        kill_count?: number;
        region_id?: number | null;
        region_name?: string | null;
        solar_system_id?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        start_time?: string;
        system_name?: string | null;
        system_security?: number | null;
        total_isk_destroyed?: number;
        [key: string]: unknown;
    };
    teams: Array<{
        members?: Array<{
            alliance_id?: number | null;
            corporation_id?: number;
            isk_destroyed?: number;
            isk_lost?: number;
            kills?: number;
            losses?: number;
            [key: string]: unknown;
        }>;
        team_index?: number;
        total_isk_destroyed?: number;
        total_isk_lost?: number;
        total_kills?: number;
        total_losses?: number;
        [key: string]: unknown;
    }>;
};

export type BattlesResponseWritable = {
    data: Array<{
        battle_id?: number;
        duration_minutes?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        end_time?: string;
        entity_isk_destroyed?: number;
        entity_kills?: string;
        entity_losses?: string;
        is_custom?: boolean;
        is_multi_party?: boolean;
        kill_count?: number;
        region_id?: number | null;
        region_name?: string | null;
        solar_system_id?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        start_time?: string;
        system_name?: string | null;
        system_security?: number | null;
        total_isk_destroyed?: number;
        [key: string]: unknown;
    }>;
    pagination: {
        hasMore: boolean;
        limit: number;
        page: number;
    };
};

export type CharacterAnalyzeResponseWritable = {
    data: Array<{
        average_gang_size: number;
        character_id: number;
        cyno_probability: number;
        efficiency: number;
        gang_probability: number;
        last_5_ships: Array<{
            kill_count?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_loss?: string | null;
            ship_name?: string | null;
            ship_type_id?: number;
            [key: string]: unknown;
        }>;
        total_kills: number;
        total_losses: number;
    }>;
};

export type CharacterIntelResponseWritable = {
    awox_kills: number;
    bait: string;
    bait_count: number;
    bridge_score: number;
    capital_pilot: boolean;
    character_id: number;
    cyno_deaths: number;
    days: number;
    dominant_style: string;
    fc: {
        likelihood?: number;
        monitor_appearances?: number;
        [key: string]: unknown;
    };
    fleet_partners: Array<{
        [key: string]: unknown;
    }>;
    groups_flown_with: Array<{
        [key: string]: unknown;
    }>;
    is_logi: boolean;
    playstyle: {
        [key: string]: unknown;
    };
    ships_flown: Array<{
        [key: string]: unknown;
    }>;
    ships_lost: Array<{
        [key: string]: unknown;
    }>;
    tags: Array<string>;
    targets: Array<{
        [key: string]: unknown;
    }>;
};

export type CharacterKillsResponseWritable = {
    data: Array<{
        attackers: Array<{
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_done?: number;
            faction_id?: number;
            final_blow?: boolean;
            security_status?: number;
            ship_type_id?: number;
            weapon_type_id?: number;
            [key: string]: unknown;
        }>;
        killmail_hash: string;
        killmail_id: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time: string;
        solar_system_id: number;
        victim: {
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_taken?: number;
            faction_id?: number;
            items?: Array<{
                flag?: number;
                item_type_id?: number;
                quantity_destroyed?: number;
                quantity_dropped?: number;
                singleton?: number;
                [key: string]: unknown;
            }>;
            position?: {
                x: number;
                y: number;
                z: number;
            };
            ship_type_id?: number;
            [key: string]: unknown;
        };
        war_id?: number;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type CharacterLossesResponseWritable = {
    data: Array<{
        attackers: Array<{
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_done?: number;
            faction_id?: number;
            final_blow?: boolean;
            security_status?: number;
            ship_type_id?: number;
            weapon_type_id?: number;
            [key: string]: unknown;
        }>;
        killmail_hash: string;
        killmail_id: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time: string;
        solar_system_id: number;
        victim: {
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_taken?: number;
            faction_id?: number;
            items?: Array<{
                flag?: number;
                item_type_id?: number;
                quantity_destroyed?: number;
                quantity_dropped?: number;
                singleton?: number;
                [key: string]: unknown;
            }>;
            position?: {
                x: number;
                y: number;
                z: number;
            };
            ship_type_id?: number;
            [key: string]: unknown;
        };
        war_id?: number;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type CharacterResponseWritable = {
    character: {
        alliance_id?: number | null;
        character_id?: number;
        corporation_id?: number | null;
        faction_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        last_active?: string | null;
        name?: string;
        security_status?: number | null;
        [key: string]: unknown;
    };
    corporationHistory: Array<{
        kills?: number;
        losses?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        start_date?: string;
        [key: string]: unknown;
    }>;
    recentStats: {
        isk_destroyed: number;
        isk_lost: number;
        kills: number;
        losses: number;
    };
    stats: {
        damage_dealt?: number;
        damage_taken?: number;
        efficiency?: number;
        final_blows?: number;
        isk_destroyed?: number;
        isk_efficiency?: number;
        isk_lost?: number;
        kills?: number;
        losses?: number;
        npc_losses?: number;
        points?: number;
        solo_kills?: number;
        [key: string]: unknown;
    };
    topShips: Array<{
        kills?: number;
        losses?: number;
        ship_name?: string;
        ship_type_id?: number;
        [key: string]: unknown;
    }>;
    topSystems: Array<{
        kills?: number;
        losses?: number;
        solar_system_id?: number;
        system_name?: string;
        [key: string]: unknown;
    }>;
};

export type CharacterStatsResponseWritable = {
    alliance_id?: number;
    character_id?: number;
    corporation_id?: number;
    damage_dealt?: number;
    damage_taken?: number;
    efficiency?: number;
    final_blows?: number;
    isk_destroyed?: number;
    isk_efficiency?: number;
    isk_lost?: number;
    kills?: number;
    losses?: number;
    npc_losses?: number;
    period?: string;
    points?: number;
    solo_kills?: number;
    topMembers?: Array<{
        id?: number;
        kills?: number;
        losses?: number;
        name?: string;
        [key: string]: unknown;
    }>;
    topShips?: Array<{
        kills?: number;
        losses?: number;
        ship_name?: string;
        ship_type_id?: number;
        [key: string]: unknown;
    }>;
    topSystems?: Array<{
        kills?: number;
        losses?: number;
        solar_system_id?: number;
        system_name?: string;
        [key: string]: unknown;
    }>;
    [key: string]: unknown;
};

export type CharactersBatchStatsResponseWritable = {
    period: string;
    results: Array<{
        damage_dealt?: number;
        damage_taken?: number;
        efficiency?: number;
        final_blows?: number;
        id?: number;
        isk_destroyed?: number;
        isk_efficiency?: number;
        isk_lost?: number;
        kills?: number;
        losses?: number;
        name?: string;
        npc_losses?: number;
        points?: number;
        solo_kills?: number;
        topShips?: Array<{
            kills?: number;
            losses?: number;
            ship_name?: string;
            ship_type_id?: number;
            [key: string]: unknown;
        }>;
        [key: string]: unknown;
    }>;
};

export type CharactersCountResponseWritable = {
    count: number;
};

export type CharactersResponseWritable = {
    data: Array<{
        alliance_id?: number | null;
        character_id?: number;
        corporation_id?: number | null;
        faction_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        last_active?: string | null;
        name?: string;
        security_status?: number | null;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type CoalitionStatsResponseWritable = {
    clashed_systems: {
        count: number;
        system_ids: Array<number>;
    };
    daily: Array<{
        date?: string;
        [key: string]: unknown;
    }>;
    from: string;
    mode: string;
    period_days: number;
    sideA: {
        active_regions_count?: number;
        active_systems_count?: number;
        entity_counts?: {
            [key: string]: unknown;
        };
        label?: string;
        overall?: {
            damage_dealt?: number;
            damage_taken?: number;
            efficiency?: number;
            final_blows?: number;
            isk_destroyed?: number;
            isk_efficiency?: number;
            isk_lost?: number;
            kills?: number;
            losses?: number;
            npc_losses?: number;
            points?: number;
            solo_kills?: number;
            [key: string]: unknown;
        };
        top_ships_used?: Array<{
            kills?: number;
            losses?: number;
            ship_name?: string;
            ship_type_id?: number;
            [key: string]: unknown;
        }>;
        vs_opponent?: {
            damage_dealt?: number;
            damage_taken?: number;
            efficiency?: number;
            final_blows?: number;
            isk_destroyed?: number;
            isk_efficiency?: number;
            isk_lost?: number;
            kills?: number;
            losses?: number;
            npc_losses?: number;
            points?: number;
            solo_kills?: number;
            [key: string]: unknown;
        };
        [key: string]: unknown;
    };
    sideB: {
        active_regions_count?: number;
        active_systems_count?: number;
        entity_counts?: {
            [key: string]: unknown;
        };
        label?: string;
        overall?: {
            damage_dealt?: number;
            damage_taken?: number;
            efficiency?: number;
            final_blows?: number;
            isk_destroyed?: number;
            isk_efficiency?: number;
            isk_lost?: number;
            kills?: number;
            losses?: number;
            npc_losses?: number;
            points?: number;
            solo_kills?: number;
            [key: string]: unknown;
        };
        top_ships_used?: Array<{
            kills?: number;
            losses?: number;
            ship_name?: string;
            ship_type_id?: number;
            [key: string]: unknown;
        }>;
        vs_opponent?: {
            damage_dealt?: number;
            damage_taken?: number;
            efficiency?: number;
            final_blows?: number;
            isk_destroyed?: number;
            isk_efficiency?: number;
            isk_lost?: number;
            kills?: number;
            losses?: number;
            npc_losses?: number;
            points?: number;
            solo_kills?: number;
            [key: string]: unknown;
        };
        [key: string]: unknown;
    };
    to: string;
};

export type CorporationBattlesResponseWritable = {
    data: Array<{
        battle_id?: number;
        duration_minutes?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        end_time?: string;
        entity_isk_destroyed?: number;
        entity_kills?: string;
        entity_losses?: string;
        is_custom?: boolean;
        is_multi_party?: boolean;
        kill_count?: number;
        region_id?: number | null;
        region_name?: string | null;
        solar_system_id?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        start_time?: string;
        system_name?: string | null;
        system_security?: number | null;
        total_isk_destroyed?: number;
        [key: string]: unknown;
    }>;
    pagination: {
        hasMore: boolean;
        limit: number;
        page: number;
    };
};

export type CorporationKillsResponseWritable = {
    data: Array<{
        attackers: Array<{
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_done?: number;
            faction_id?: number;
            final_blow?: boolean;
            security_status?: number;
            ship_type_id?: number;
            weapon_type_id?: number;
            [key: string]: unknown;
        }>;
        killmail_hash: string;
        killmail_id: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time: string;
        solar_system_id: number;
        victim: {
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_taken?: number;
            faction_id?: number;
            items?: Array<{
                flag?: number;
                item_type_id?: number;
                quantity_destroyed?: number;
                quantity_dropped?: number;
                singleton?: number;
                [key: string]: unknown;
            }>;
            position?: {
                x: number;
                y: number;
                z: number;
            };
            ship_type_id?: number;
            [key: string]: unknown;
        };
        war_id?: number;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type CorporationLossesResponseWritable = {
    data: Array<{
        attackers: Array<{
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_done?: number;
            faction_id?: number;
            final_blow?: boolean;
            security_status?: number;
            ship_type_id?: number;
            weapon_type_id?: number;
            [key: string]: unknown;
        }>;
        killmail_hash: string;
        killmail_id: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time: string;
        solar_system_id: number;
        victim: {
            alliance_id?: number;
            character_id?: number;
            corporation_id?: number;
            damage_taken?: number;
            faction_id?: number;
            items?: Array<{
                flag?: number;
                item_type_id?: number;
                quantity_destroyed?: number;
                quantity_dropped?: number;
                singleton?: number;
                [key: string]: unknown;
            }>;
            position?: {
                x: number;
                y: number;
                z: number;
            };
            ship_type_id?: number;
            [key: string]: unknown;
        };
        war_id?: number;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type CorporationMembersResponseWritable = {
    data: Array<{
        alliance_id?: number | null;
        character_id?: number;
        corporation_id?: number | null;
        faction_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        last_active?: string | null;
        name?: string;
        security_status?: number | null;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type CorporationResponseWritable = {
    allianceHistory: Array<{
        kills?: number;
        losses?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        start_date?: string;
        [key: string]: unknown;
    }>;
    corporation: {
        alliance_id?: number | null;
        corporation_id?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_founded?: string | null;
        faction_id?: number | null;
        member_count?: number | null;
        name?: string;
        ticker?: string;
        [key: string]: unknown;
    };
    recentStats: {
        isk_destroyed: number;
        isk_lost: number;
        kills: number;
        losses: number;
    };
    stats: {
        damage_dealt?: number;
        damage_taken?: number;
        efficiency?: number;
        final_blows?: number;
        isk_destroyed?: number;
        isk_efficiency?: number;
        isk_lost?: number;
        kills?: number;
        losses?: number;
        npc_losses?: number;
        points?: number;
        solo_kills?: number;
        [key: string]: unknown;
    };
};

export type CorporationStatsAlltimeResponseWritable = {
    alliance_id?: number;
    character_id?: number;
    corporation_id?: number;
    damage_dealt?: number;
    damage_taken?: number;
    efficiency?: number;
    final_blows?: number;
    isk_destroyed?: number;
    isk_efficiency?: number;
    isk_lost?: number;
    kills?: number;
    losses?: number;
    npc_losses?: number;
    period?: string;
    points?: number;
    solo_kills?: number;
    topMembers?: Array<{
        id?: number;
        kills?: number;
        losses?: number;
        name?: string;
        [key: string]: unknown;
    }>;
    topShips?: Array<{
        kills?: number;
        losses?: number;
        ship_name?: string;
        ship_type_id?: number;
        [key: string]: unknown;
    }>;
    topSystems?: Array<{
        kills?: number;
        losses?: number;
        solar_system_id?: number;
        system_name?: string;
        [key: string]: unknown;
    }>;
    [key: string]: unknown;
};

export type CorporationStatsWeeklyResponseWritable = {
    alliance_id?: number;
    character_id?: number;
    corporation_id?: number;
    damage_dealt?: number;
    damage_taken?: number;
    efficiency?: number;
    final_blows?: number;
    isk_destroyed?: number;
    isk_efficiency?: number;
    isk_lost?: number;
    kills?: number;
    losses?: number;
    npc_losses?: number;
    period?: string;
    points?: number;
    solo_kills?: number;
    topMembers?: Array<{
        id?: number;
        kills?: number;
        losses?: number;
        name?: string;
        [key: string]: unknown;
    }>;
    topShips?: Array<{
        kills?: number;
        losses?: number;
        ship_name?: string;
        ship_type_id?: number;
        [key: string]: unknown;
    }>;
    topSystems?: Array<{
        kills?: number;
        losses?: number;
        solar_system_id?: number;
        system_name?: string;
        [key: string]: unknown;
    }>;
    [key: string]: unknown;
};

export type CorporationsBatchStatsResponseWritable = {
    period: string;
    results: Array<{
        damage_dealt?: number;
        damage_taken?: number;
        efficiency?: number;
        final_blows?: number;
        id?: number;
        isk_destroyed?: number;
        isk_efficiency?: number;
        isk_lost?: number;
        kills?: number;
        losses?: number;
        name?: string;
        npc_losses?: number;
        points?: number;
        solo_kills?: number;
        topShips?: Array<{
            kills?: number;
            losses?: number;
            ship_name?: string;
            ship_type_id?: number;
            [key: string]: unknown;
        }>;
        [key: string]: unknown;
    }>;
};

export type CorporationsCountResponseWritable = {
    count: number;
};

export type CorporationsResponseWritable = {
    data: Array<{
        alliance_id?: number | null;
        corporation_id?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_founded?: string | null;
        faction_id?: number | null;
        member_count?: number | null;
        name?: string;
        ticker?: string;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type FeedIndexResponseWritable = {
    description: string;
    endpoints: {
        [key: string]: {
            description?: string;
            example?: string;
            headers?: {
                [key: string]: string;
            };
            params?: {
                [key: string]: string;
            };
            [key: string]: unknown;
        };
    };
    name: string;
    note: string;
    topics: {
        [key: string]: Array<string>;
    };
};

export type FeedPollResponseWritable = {
    data: Array<{
        data: {
            attackers: Array<{
                alliance_id?: number;
                character_id?: number;
                corporation_id?: number;
                damage_done?: number;
                faction_id?: number;
                final_blow?: boolean;
                security_status?: number;
                ship_type_id?: number;
                weapon_type_id?: number;
                [key: string]: unknown;
            }>;
            killmail_hash: string;
            killmail_id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            killmail_time: string;
            solar_system_id: number;
            victim: {
                alliance_id?: number;
                character_id?: number;
                corporation_id?: number;
                damage_taken?: number;
                faction_id?: number;
                items?: Array<{
                    flag?: number;
                    item_type_id?: number;
                    quantity_destroyed?: number;
                    quantity_dropped?: number;
                    singleton?: number;
                    [key: string]: unknown;
                }>;
                position?: {
                    x: number;
                    y: number;
                    z: number;
                };
                ship_type_id?: number;
                [key: string]: unknown;
            };
            war_id?: number;
        };
        killmail_hash: string;
        killmail_id: number;
        seq: number;
    }>;
    hasMore: boolean;
    last: string;
    latest: number | null;
    next: string;
};

export type FeedStatusResponseWritable = {
    clients: number;
    latestSeq: number | null;
    status: string;
};

export type GlobalStatsResponseWritable = {
    entries: Array<{
        efficiency?: number;
        id?: number;
        isk_destroyed?: number;
        isk_lost?: number;
        killmail_hash?: string;
        killmail_id?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time?: string;
        kills?: number;
        losses?: number;
        name?: string;
        sec?: number;
        ticker?: string;
        total_value?: number;
        [key: string]: unknown;
    }>;
};

export type HealthResponseWritable = {
    ok: boolean;
    /**
     * UTC timestamp with millisecond precision.
     */
    timestamp: string;
};

export type HistoryDateResponseWritable = {
    /**
     * Map of killmail ID to killmail hash.
     */
    data: {
        [key: string]: string;
    };
};

export type HistoryLatestResponseWritable = {
    /**
     * Map of killmail ID to killmail hash.
     */
    data: {
        [key: string]: string;
    };
};

export type HistoryResponseWritable = {
    data: Array<{
        count?: number;
        date?: string;
        [key: string]: unknown;
    }>;
};

export type KillmailEsiResponseWritable = {
    attackers: Array<{
        alliance_id?: number;
        character_id?: number;
        corporation_id?: number;
        damage_done?: number;
        faction_id?: number;
        final_blow?: boolean;
        security_status?: number;
        ship_type_id?: number;
        weapon_type_id?: number;
        [key: string]: unknown;
    }>;
    killmail_hash: string;
    killmail_id: number;
    /**
     * UTC timestamp with millisecond precision.
     */
    killmail_time: string;
    solar_system_id: number;
    victim: {
        alliance_id?: number;
        character_id?: number;
        corporation_id?: number;
        damage_taken?: number;
        faction_id?: number;
        items?: Array<{
            flag?: number;
            item_type_id?: number;
            quantity_destroyed?: number;
            quantity_dropped?: number;
            singleton?: number;
            [key: string]: unknown;
        }>;
        position?: {
            x: number;
            y: number;
            z: number;
        };
        ship_type_id?: number;
        [key: string]: unknown;
    };
    war_id?: number;
};

export type KillmailFittingResponseWritable = {
    cargo?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    drone?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    fighter?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    fighter_bay?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    fleet?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    high?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    killmail_id: number;
    low?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    mid?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    other?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    rig?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    service?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    ship: {
        name: string;
        type_id: number;
    };
    specialized?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
    subsystem?: Array<{
        name: string;
        quantity: number;
        type_id: number;
    }>;
};

export type KillmailResponseWritable = {
    attacker_count?: number;
    attackers: Array<{
        alliance_id?: number | null;
        character_id?: number | null;
        corporation_id?: number | null;
        damage_done?: number;
        final_blow?: boolean;
        ship_type_id?: number | null;
        weapon_type_id?: number | null;
        [key: string]: unknown;
    }>;
    constellation_id?: number | null;
    constellation_name?: string | null;
    destroyed_value?: number;
    dropped_value?: number;
    fitted_value?: number;
    is_npc?: boolean;
    is_solo?: boolean;
    items: Array<{
        item_index?: number;
        price?: number;
        quantity_destroyed?: number;
        quantity_dropped?: number;
        slot?: string;
        total_value?: number;
        type_id?: number;
        type_name?: string | null;
        [key: string]: unknown;
    }>;
    killmail_hash: string;
    killmail_id: number;
    /**
     * UTC timestamp with millisecond precision.
     */
    killmail_time: string;
    location?: {
        distance?: number;
        group_id?: number;
        item_id?: number;
        item_name?: string;
        type_id?: number;
        [key: string]: unknown;
    } | null;
    points?: number;
    position_x?: number | null;
    position_y?: number | null;
    position_z?: number | null;
    region_id?: number | null;
    region_name?: string | null;
    siblings: Array<{
        killmail_id?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time?: string;
        ship_group_id?: number | null;
        ship_name?: string | null;
        ship_type_id?: number | null;
        total_value?: number;
        [key: string]: unknown;
    }>;
    solar_system_id: number;
    solar_system_name?: string | null;
    solar_system_security?: number | null;
    total_damage?: number;
    total_value: number;
    victim: {
        alliance_id?: number | null;
        alliance_name?: string | null;
        character_id?: number | null;
        character_name?: string | null;
        corporation_id?: number | null;
        corporation_name?: string | null;
        corporation_palette?: string | null;
        damage_taken?: number;
        ship_group_id?: number | null;
        ship_group_name?: string | null;
        ship_market_path?: string | null;
        ship_name?: string | null;
        ship_price?: number;
        ship_type_id?: number | null;
        [key: string]: unknown;
    };
};

export type KillmailSearchResponseWritable = {
    data: Array<{
        attacker_count: number;
        final_blow_alliance_id: number | null;
        final_blow_alliance_name: string | null;
        final_blow_character_id: number | null;
        final_blow_character_name: string | null;
        final_blow_corporation_id: number | null;
        final_blow_corporation_name: string | null;
        final_blow_ship_name: string | null;
        final_blow_ship_type_id: number | null;
        is_npc: boolean;
        is_solo: boolean;
        killmail_hash: string;
        killmail_id: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time: string;
        meta_group_id: number | null;
        region_id: number | null;
        region_name: string | null;
        ship_group_name: string | null;
        ship_market_path: string | null;
        ship_name: string | null;
        ship_type_id: number | null;
        solar_system_id: number;
        solar_system_name: string | null;
        solar_system_security: number | null;
        total_value: number;
        victim_alliance_id: number | null;
        victim_alliance_name: string | null;
        victim_character_id: number | null;
        victim_character_name: string | null;
        victim_corporation_id: number | null;
        victim_corporation_name: string | null;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type KillmailsCountResponseWritable = {
    count: number;
};

export type KillmailsResponseWritable = {
    data: Array<{
        attacker_count: number;
        final_blow_alliance_id: number | null;
        final_blow_alliance_name: string | null;
        final_blow_character_id: number | null;
        final_blow_character_name: string | null;
        final_blow_corporation_id: number | null;
        final_blow_corporation_name: string | null;
        final_blow_ship_name: string | null;
        final_blow_ship_type_id: number | null;
        is_npc: boolean;
        is_solo: boolean;
        killmail_hash: string;
        killmail_id: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time: string;
        meta_group_id: number | null;
        region_id: number | null;
        region_name: string | null;
        ship_group_name: string | null;
        ship_market_path: string | null;
        ship_name: string | null;
        ship_type_id: number | null;
        solar_system_id: number;
        solar_system_name: string | null;
        solar_system_security: number | null;
        total_value: number;
        victim_alliance_id: number | null;
        victim_alliance_name: string | null;
        victim_character_id: number | null;
        victim_character_name: string | null;
        victim_corporation_id: number | null;
        victim_corporation_name: string | null;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type LocationResponseWritable = {
    nearest: {
        distance?: number;
        group_id?: number;
        item_id?: number;
        item_name?: string;
        type_id?: number;
        [key: string]: unknown;
    } | null;
    system_id: number;
    x: number;
    y: number;
    z: number;
};

export type ResolveResponseWritable = {
    resolved: number;
    results: Array<{
        id?: number | null;
        name?: string;
        resolved_name?: string | null;
        [key: string]: unknown;
    }>;
    type: string;
    unresolved: number;
};

/**
 * bloodline
 *
 * Static data record. Known stable fields are documented; source-table-specific fields are preserved as additional properties.
 */
export type SdeBloodlineResponseWritable = {
    alliance_id?: number | null;
    attribute_id?: number | null;
    bloodline_id?: number | null;
    category_id?: number | null;
    corporation_id?: number | null;
    /**
     * UTC timestamp with millisecond precision.
     */
    date_added?: string;
    description?: string | null;
    effect_id?: number | null;
    faction_id?: number | null;
    flag_id?: number | null;
    id?: number;
    item_id?: number | null;
    market_group_id?: number | null;
    material_type_id?: number | null;
    meta_group_id?: number | null;
    name?: string;
    operation?: string;
    published?: boolean;
    race_id?: number | null;
    solar_system_id?: number | null;
    system_id?: number | null;
    type_id?: number | null;
    [key: string]: unknown;
};

export type SdeBloodlinesResponseWritable = {
    data: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
};

export type SdeCategoriesResponseWritable = {
    data: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
};

/**
 * category
 *
 * Static data record. Known stable fields are documented; source-table-specific fields are preserved as additional properties.
 */
export type SdeCategoryResponseWritable = {
    alliance_id?: number | null;
    attribute_id?: number | null;
    bloodline_id?: number | null;
    category_id?: number | null;
    corporation_id?: number | null;
    /**
     * UTC timestamp with millisecond precision.
     */
    date_added?: string;
    description?: string | null;
    effect_id?: number | null;
    faction_id?: number | null;
    flag_id?: number | null;
    id?: number;
    item_id?: number | null;
    market_group_id?: number | null;
    material_type_id?: number | null;
    meta_group_id?: number | null;
    name?: string;
    operation?: string;
    published?: boolean;
    race_id?: number | null;
    solar_system_id?: number | null;
    system_id?: number | null;
    type_id?: number | null;
    [key: string]: unknown;
};

/**
 * celestial
 *
 * Static data record. Known stable fields are documented; source-table-specific fields are preserved as additional properties.
 */
export type SdeCelestialResponseWritable = {
    alliance_id?: number | null;
    attribute_id?: number | null;
    bloodline_id?: number | null;
    category_id?: number | null;
    corporation_id?: number | null;
    /**
     * UTC timestamp with millisecond precision.
     */
    date_added?: string;
    description?: string | null;
    effect_id?: number | null;
    faction_id?: number | null;
    flag_id?: number | null;
    id?: number;
    item_id?: number | null;
    market_group_id?: number | null;
    material_type_id?: number | null;
    meta_group_id?: number | null;
    name?: string;
    operation?: string;
    published?: boolean;
    race_id?: number | null;
    solar_system_id?: number | null;
    system_id?: number | null;
    type_id?: number | null;
    [key: string]: unknown;
};

export type SdeConstellationResponseWritable = {
    constellation_id?: number;
    constellation_name?: string;
    faction_id?: number | null;
    region_id?: number;
    region_name?: string | null;
    [key: string]: unknown;
};

export type SdeConstellationsResponseWritable = {
    data: Array<{
        constellation_id?: number;
        constellation_name?: string;
        faction_id?: number | null;
        region_id?: number;
        region_name?: string | null;
        [key: string]: unknown;
    }>;
};

export type SdeCustomPricesResponseWritable = {
    count: number;
    data: Array<{
        price?: number;
        type_id?: number;
        type_name?: string | null;
        valid_until?: string;
        [key: string]: unknown;
    }>;
};

/**
 * faction
 *
 * Static data record. Known stable fields are documented; source-table-specific fields are preserved as additional properties.
 */
export type SdeFactionResponseWritable = {
    alliance_id?: number | null;
    attribute_id?: number | null;
    bloodline_id?: number | null;
    category_id?: number | null;
    corporation_id?: number | null;
    /**
     * UTC timestamp with millisecond precision.
     */
    date_added?: string;
    description?: string | null;
    effect_id?: number | null;
    faction_id?: number | null;
    flag_id?: number | null;
    id?: number;
    item_id?: number | null;
    market_group_id?: number | null;
    material_type_id?: number | null;
    meta_group_id?: number | null;
    name?: string;
    operation?: string;
    published?: boolean;
    race_id?: number | null;
    solar_system_id?: number | null;
    system_id?: number | null;
    type_id?: number | null;
    [key: string]: unknown;
};

export type SdeFactionsResponseWritable = {
    data: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
};

export type SdeFlagsResponseWritable = {
    data: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
};

export type SdeGroupResponseWritable = {
    category_id?: number;
    category_name?: string | null;
    group_id?: number;
    icon_id?: number | null;
    name?: string;
    published?: boolean;
    [key: string]: unknown;
};

export type SdeGroupsResponseWritable = {
    data: Array<{
        category_id?: number;
        category_name?: string | null;
        group_id?: number;
        icon_id?: number | null;
        name?: string;
        published?: boolean;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

/**
 * market-group
 *
 * Static data record. Known stable fields are documented; source-table-specific fields are preserved as additional properties.
 */
export type SdeMarketGroupResponseWritable = {
    alliance_id?: number | null;
    attribute_id?: number | null;
    bloodline_id?: number | null;
    category_id?: number | null;
    corporation_id?: number | null;
    /**
     * UTC timestamp with millisecond precision.
     */
    date_added?: string;
    description?: string | null;
    effect_id?: number | null;
    faction_id?: number | null;
    flag_id?: number | null;
    id?: number;
    item_id?: number | null;
    market_group_id?: number | null;
    material_type_id?: number | null;
    meta_group_id?: number | null;
    name?: string;
    operation?: string;
    published?: boolean;
    race_id?: number | null;
    solar_system_id?: number | null;
    system_id?: number | null;
    type_id?: number | null;
    [key: string]: unknown;
};

export type SdeMarketGroupsResponseWritable = {
    data: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
};

export type SdeMetaGroupsResponseWritable = {
    data: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
};

/**
 * npc-corporation
 *
 * Static data record. Known stable fields are documented; source-table-specific fields are preserved as additional properties.
 */
export type SdeNpcCorporationResponseWritable = {
    alliance_id?: number | null;
    attribute_id?: number | null;
    bloodline_id?: number | null;
    category_id?: number | null;
    corporation_id?: number | null;
    /**
     * UTC timestamp with millisecond precision.
     */
    date_added?: string;
    description?: string | null;
    effect_id?: number | null;
    faction_id?: number | null;
    flag_id?: number | null;
    id?: number;
    item_id?: number | null;
    market_group_id?: number | null;
    material_type_id?: number | null;
    meta_group_id?: number | null;
    name?: string;
    operation?: string;
    published?: boolean;
    race_id?: number | null;
    solar_system_id?: number | null;
    system_id?: number | null;
    type_id?: number | null;
    [key: string]: unknown;
};

export type SdeNpcCorporationsResponseWritable = {
    data: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
};

export type SdePricesResponseWritable = {
    prices: Array<{
        average?: number;
        date?: string;
        highest?: number;
        lowest?: number;
        order_count?: number;
        region_id?: number;
        type_id?: number;
        volume?: number;
        [key: string]: unknown;
    }>;
    region_id: number;
    type_id: number;
};

/**
 * race
 *
 * Static data record. Known stable fields are documented; source-table-specific fields are preserved as additional properties.
 */
export type SdeRaceResponseWritable = {
    alliance_id?: number | null;
    attribute_id?: number | null;
    bloodline_id?: number | null;
    category_id?: number | null;
    corporation_id?: number | null;
    /**
     * UTC timestamp with millisecond precision.
     */
    date_added?: string;
    description?: string | null;
    effect_id?: number | null;
    faction_id?: number | null;
    flag_id?: number | null;
    id?: number;
    item_id?: number | null;
    market_group_id?: number | null;
    material_type_id?: number | null;
    meta_group_id?: number | null;
    name?: string;
    operation?: string;
    published?: boolean;
    race_id?: number | null;
    solar_system_id?: number | null;
    system_id?: number | null;
    type_id?: number | null;
    [key: string]: unknown;
};

export type SdeRacesResponseWritable = {
    data: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
};

export type SdeRegionKillsResponseWritable = {
    data: Array<{
        attacker_count: number;
        final_blow_alliance_id: number | null;
        final_blow_alliance_name: string | null;
        final_blow_character_id: number | null;
        final_blow_character_name: string | null;
        final_blow_corporation_id: number | null;
        final_blow_corporation_name: string | null;
        final_blow_ship_name: string | null;
        final_blow_ship_type_id: number | null;
        is_npc: boolean;
        is_solo: boolean;
        killmail_hash: string;
        killmail_id: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time: string;
        meta_group_id: number | null;
        region_id: number | null;
        region_name: string | null;
        ship_group_name: string | null;
        ship_market_path: string | null;
        ship_name: string | null;
        ship_type_id: number | null;
        solar_system_id: number;
        solar_system_name: string | null;
        solar_system_security: number | null;
        total_value: number;
        victim_alliance_id: number | null;
        victim_alliance_name: string | null;
        victim_character_id: number | null;
        victim_character_name: string | null;
        victim_corporation_id: number | null;
        victim_corporation_name: string | null;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type SdeRegionResponseWritable = {
    description?: string | null;
    faction_id?: number | null;
    faction_name?: string | null;
    name?: string;
    region_id?: number;
    [key: string]: unknown;
};

export type SdeRegionsResponseWritable = {
    data: Array<{
        description?: string | null;
        faction_id?: number | null;
        faction_name?: string | null;
        name?: string;
        region_id?: number;
        [key: string]: unknown;
    }>;
};

export type SdeSovereigntyHistoryResponseWritable = {
    history: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
    system_id: number;
};

export type SdeSovereigntyResponseWritable = {
    data: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
};

/**
 * sovereignty-system
 *
 * Static data record. Known stable fields are documented; source-table-specific fields are preserved as additional properties.
 */
export type SdeSovereigntySystemResponseWritable = {
    alliance_id?: number | null;
    attribute_id?: number | null;
    bloodline_id?: number | null;
    category_id?: number | null;
    corporation_id?: number | null;
    /**
     * UTC timestamp with millisecond precision.
     */
    date_added?: string;
    description?: string | null;
    effect_id?: number | null;
    faction_id?: number | null;
    flag_id?: number | null;
    id?: number;
    item_id?: number | null;
    market_group_id?: number | null;
    material_type_id?: number | null;
    meta_group_id?: number | null;
    name?: string;
    operation?: string;
    published?: boolean;
    race_id?: number | null;
    solar_system_id?: number | null;
    system_id?: number | null;
    type_id?: number | null;
    [key: string]: unknown;
};

/**
 * station-operation
 *
 * Static data record. Known stable fields are documented; source-table-specific fields are preserved as additional properties.
 */
export type SdeStationOperationResponseWritable = {
    alliance_id?: number | null;
    attribute_id?: number | null;
    bloodline_id?: number | null;
    category_id?: number | null;
    corporation_id?: number | null;
    /**
     * UTC timestamp with millisecond precision.
     */
    date_added?: string;
    description?: string | null;
    effect_id?: number | null;
    faction_id?: number | null;
    flag_id?: number | null;
    id?: number;
    item_id?: number | null;
    market_group_id?: number | null;
    material_type_id?: number | null;
    meta_group_id?: number | null;
    name?: string;
    operation?: string;
    published?: boolean;
    race_id?: number | null;
    solar_system_id?: number | null;
    system_id?: number | null;
    type_id?: number | null;
    [key: string]: unknown;
};

export type SdeStationOperationsResponseWritable = {
    data: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
};

export type SdeStationResponseWritable = {
    corporation_id?: number;
    region_id?: number;
    security?: number;
    solar_system_id?: number;
    station_id?: number;
    station_name?: string;
    type_id?: number;
    [key: string]: unknown;
};

export type SdeStationsResponseWritable = {
    data: Array<{
        corporation_id?: number;
        region_id?: number;
        security?: number;
        solar_system_id?: number;
        station_id?: number;
        station_name?: string;
        type_id?: number;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type SdeStructureResponseWritable = {
    name?: string;
    owner_id?: number;
    region_id?: number;
    solar_system_id?: number;
    structure_id?: number;
    type_id?: number;
    [key: string]: unknown;
};

export type SdeStructuresResponseWritable = {
    data: Array<{
        name?: string;
        owner_id?: number;
        region_id?: number;
        solar_system_id?: number;
        structure_id?: number;
        type_id?: number;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type SdeSystemCelestialsResponseWritable = {
    celestials: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
    solar_system_id: number;
};

export type SdeSystemJumpsResponseWritable = {
    jumps: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
    solar_system_id: number;
};

export type SdeSystemKillsResponseWritable = {
    data: Array<{
        attacker_count: number;
        final_blow_alliance_id: number | null;
        final_blow_alliance_name: string | null;
        final_blow_character_id: number | null;
        final_blow_character_name: string | null;
        final_blow_corporation_id: number | null;
        final_blow_corporation_name: string | null;
        final_blow_ship_name: string | null;
        final_blow_ship_type_id: number | null;
        is_npc: boolean;
        is_solo: boolean;
        killmail_hash: string;
        killmail_id: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        killmail_time: string;
        meta_group_id: number | null;
        region_id: number | null;
        region_name: string | null;
        ship_group_name: string | null;
        ship_market_path: string | null;
        ship_name: string | null;
        ship_type_id: number | null;
        solar_system_id: number;
        solar_system_name: string | null;
        solar_system_security: number | null;
        total_value: number;
        victim_alliance_id: number | null;
        victim_alliance_name: string | null;
        victim_character_id: number | null;
        victim_character_name: string | null;
        victim_corporation_id: number | null;
        victim_corporation_name: string | null;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type SdeSystemResponseWritable = {
    constellation_id?: number;
    constellation_name?: string | null;
    region_id?: number;
    region_name?: string | null;
    security?: number;
    security_class?: string | null;
    solar_system_id?: number;
    system_name?: string;
    [key: string]: unknown;
};

export type SdeSystemsResponseWritable = {
    data: Array<{
        constellation_id?: number;
        constellation_name?: string | null;
        region_id?: number;
        region_name?: string | null;
        security?: number;
        security_class?: string | null;
        solar_system_id?: number;
        system_name?: string;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type SdeTypeDogmaResponseWritable = {
    attributes: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
    effects: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
    type_id: number;
};

export type SdeTypeInsuranceResponseWritable = {
    levels: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
    type_id: number;
};

export type SdeTypeMaterialsResponseWritable = {
    materials: Array<{
        alliance_id?: number | null;
        attribute_id?: number | null;
        bloodline_id?: number | null;
        category_id?: number | null;
        corporation_id?: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        date_added?: string;
        description?: string | null;
        effect_id?: number | null;
        faction_id?: number | null;
        flag_id?: number | null;
        id?: number;
        item_id?: number | null;
        market_group_id?: number | null;
        material_type_id?: number | null;
        meta_group_id?: number | null;
        name?: string;
        operation?: string;
        published?: boolean;
        race_id?: number | null;
        solar_system_id?: number | null;
        system_id?: number | null;
        type_id?: number | null;
        [key: string]: unknown;
    }>;
    type_id: number;
};

export type SdeTypeResponseWritable = {
    base_price?: number | null;
    capacity?: number | null;
    category_id?: number;
    category_name?: string | null;
    description?: string | null;
    group_id?: number;
    group_name?: string | null;
    market_group_id?: number | null;
    mass?: number | null;
    meta_group_id?: number | null;
    name?: string;
    published?: boolean;
    type_id?: number;
    volume?: number | null;
    [key: string]: unknown;
};

export type SdeTypesResponseWritable = {
    data: Array<{
        base_price?: number | null;
        capacity?: number | null;
        category_id?: number;
        category_name?: string | null;
        description?: string | null;
        group_id?: number;
        group_name?: string | null;
        market_group_id?: number | null;
        mass?: number | null;
        meta_group_id?: number | null;
        name?: string;
        published?: boolean;
        type_id?: number;
        volume?: number | null;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type SearchResponseWritable = {
    entityCounts: {
        [key: string]: number;
    };
    hits: Array<{
        alliance_id?: number | null;
        alliance_name?: string;
        alliance_ticker?: string;
        corporation_id?: number | null;
        corporation_name?: string;
        corporation_ticker?: string;
        id?: string;
        name?: string;
        ticker?: string | null;
        type?: string;
        [key: string]: unknown;
    }>;
    processingTimeMs: number;
    query: string;
    total: number;
};

export type ShipFittingsResponseWritable = {
    families: Array<{
        canonical_fit_hash?: string;
        canonical_uses?: number;
        family_hash?: string;
        fit_cost?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        last_used?: string;
        modules?: Array<{
            [key: string]: unknown;
        }>;
        top_alliances?: Array<{
            [key: string]: unknown;
        }>;
        total_uses?: number;
        variant_count?: number;
        [key: string]: unknown;
    }>;
    hull_cost: number;
    is_rare_hull: boolean;
    module_filter: Array<number>;
    ship_type_id: number;
    window_days: number;
};

export type WarResponseWritable = {
    stats: {
        top_ships: Array<{
            kills?: number;
            losses?: number;
            ship_name?: string;
            ship_type_id?: number;
            [key: string]: unknown;
        }>;
        total_kills: number;
        total_value: number;
    };
    war: {
        aggressor: {
            id: number;
            isk_destroyed: number;
            name: string;
            ships_killed: number;
            ticker: string;
            type: string;
        };
        allies: Array<{
            id: number;
            name: string;
            type: string;
        }>;
        /**
         * UTC timestamp with millisecond precision.
         */
        declared: string;
        defender: {
            id: number;
            isk_destroyed: number;
            name: string;
            ships_killed: number;
            ticker: string;
            type: string;
        };
        /**
         * UTC timestamp with millisecond precision.
         */
        finished: string | null;
        mutual: boolean;
        open_for_allies: boolean;
        /**
         * UTC timestamp with millisecond precision.
         */
        retracted: string | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        started: string | null;
        war_id: number;
    };
};

export type WarsResponseWritable = {
    data: Array<{
        aggressor_alliance_id?: number | null;
        aggressor_corporation_id?: number | null;
        aggressor_ships_killed?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        declared?: string;
        defender_alliance_id?: number | null;
        defender_corporation_id?: number | null;
        defender_ships_killed?: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        finished?: string | null;
        mutual?: boolean;
        /**
         * UTC timestamp with millisecond precision.
         */
        started?: string | null;
        war_id?: number;
        [key: string]: unknown;
    }>;
    pagination: {
        cursor: number | null;
        hasMore: boolean;
    };
};

export type SitemapAlliancesCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/__sitemap__/alliances';
};

export type SitemapAlliancesCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type SitemapAlliancesCompatResponse = SitemapAlliancesCompatResponses[keyof SitemapAlliancesCompatResponses];

export type SitemapBattlesCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/__sitemap__/battles';
};

export type SitemapBattlesCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type SitemapBattlesCompatResponse = SitemapBattlesCompatResponses[keyof SitemapBattlesCompatResponses];

export type SitemapCharactersCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/__sitemap__/characters';
};

export type SitemapCharactersCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type SitemapCharactersCompatResponse = SitemapCharactersCompatResponses[keyof SitemapCharactersCompatResponses];

export type SitemapCorporationsCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/__sitemap__/corporations';
};

export type SitemapCorporationsCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type SitemapCorporationsCompatResponse = SitemapCorporationsCompatResponses[keyof SitemapCorporationsCompatResponses];

export type SitemapItemsCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/__sitemap__/items';
};

export type SitemapItemsCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type SitemapItemsCompatResponse = SitemapItemsCompatResponses[keyof SitemapItemsCompatResponses];

export type SitemapKillsCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/__sitemap__/kills';
};

export type SitemapKillsCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type SitemapKillsCompatResponse = SitemapKillsCompatResponses[keyof SitemapKillsCompatResponses];

export type SitemapRegionsCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/__sitemap__/regions';
};

export type SitemapRegionsCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type SitemapRegionsCompatResponse = SitemapRegionsCompatResponses[keyof SitemapRegionsCompatResponses];

export type SitemapShipsCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/__sitemap__/ships';
};

export type SitemapShipsCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type SitemapShipsCompatResponse = SitemapShipsCompatResponses[keyof SitemapShipsCompatResponses];

export type SitemapSystemsCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/__sitemap__/systems';
};

export type SitemapSystemsCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type SitemapSystemsCompatResponse = SitemapSystemsCompatResponses[keyof SitemapSystemsCompatResponses];

export type SitemapWarsCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/__sitemap__/wars';
};

export type SitemapWarsCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type SitemapWarsCompatResponse = SitemapWarsCompatResponses[keyof SitemapWarsCompatResponses];

export type AnnouncementAdminListData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/announcements';
};

export type AnnouncementAdminListResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AnnouncementAdminListResponse = AnnouncementAdminListResponses[keyof AnnouncementAdminListResponses];

export type AnnouncementAdminCreateData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/announcements';
};

export type AnnouncementAdminCreateResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AnnouncementAdminCreateResponse = AnnouncementAdminCreateResponses[keyof AnnouncementAdminCreateResponses];

export type AnnouncementAdminArchiveData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/announcements/{id}';
};

export type AnnouncementAdminArchiveResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AnnouncementAdminArchiveResponse = AnnouncementAdminArchiveResponses[keyof AnnouncementAdminArchiveResponses];

export type AnnouncementAdminDetailData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/announcements/{id}';
};

export type AnnouncementAdminDetailResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AnnouncementAdminDetailResponse = AnnouncementAdminDetailResponses[keyof AnnouncementAdminDetailResponses];

export type AnnouncementAdminUpdateData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/announcements/{id}';
};

export type AnnouncementAdminUpdateResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AnnouncementAdminUpdateResponse = AnnouncementAdminUpdateResponses[keyof AnnouncementAdminUpdateResponses];

export type AnnouncementAdminArchiveCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/announcements/{id}/archive';
};

export type AnnouncementAdminArchiveCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AnnouncementAdminArchiveCompatResponse = AnnouncementAdminArchiveCompatResponses[keyof AnnouncementAdminArchiveCompatResponses];

export type BlogAdminListData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/blog';
};

export type BlogAdminListResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type BlogAdminListResponse = BlogAdminListResponses[keyof BlogAdminListResponses];

export type BlogAdminCreateData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/blog';
};

export type BlogAdminCreateResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type BlogAdminCreateResponse = BlogAdminCreateResponses[keyof BlogAdminCreateResponses];

export type BlogAdminPreviewData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/blog/preview/{slug}';
};

export type BlogAdminPreviewResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type BlogAdminPreviewResponse = BlogAdminPreviewResponses[keyof BlogAdminPreviewResponses];

export type BlogAdminDeleteData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/blog/{id}';
};

export type BlogAdminDeleteResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type BlogAdminDeleteResponse = BlogAdminDeleteResponses[keyof BlogAdminDeleteResponses];

export type BlogAdminDetailData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/blog/{id}';
};

export type BlogAdminDetailResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type BlogAdminDetailResponse = BlogAdminDetailResponses[keyof BlogAdminDetailResponses];

export type BlogAdminUpdateData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/blog/{id}';
};

export type BlogAdminUpdateResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type BlogAdminUpdateResponse = BlogAdminUpdateResponses[keyof BlogAdminUpdateResponses];

export type CampaignPrizePaidLegacyData = {
    body: {
        /**
         * Operator note recorded with the payout.
         */
        note?: string;
    };
    path?: never;
    query?: never;
    url: '/admin/campaign-prizes/{id}/{rank}/paid';
};

export type CampaignPrizePaidLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CampaignPrizePaidLegacyResponse = CampaignPrizePaidLegacyResponses[keyof CampaignPrizePaidLegacyResponses];

export type CampaignAdminListData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/campaigns';
};

export type CampaignAdminListResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CampaignAdminListResponse = CampaignAdminListResponses[keyof CampaignAdminListResponses];

export type CampaignAdminActionLegacyData = {
    body: {
        /**
         * Administrative action to apply to the campaign.
         */
        action: string;
        /**
         * Operator note recorded with the action.
         */
        reason?: string;
    };
    path?: never;
    query?: never;
    url: '/admin/campaigns/{id}/action';
};

export type CampaignAdminActionLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CampaignAdminActionLegacyResponse = CampaignAdminActionLegacyResponses[keyof CampaignAdminActionLegacyResponses];

export type CampaignAdminActionData = {
    body: {
        /**
         * Administrative action to apply to the campaign.
         */
        action: string;
        /**
         * Operator note recorded with the action.
         */
        reason?: string;
    };
    path?: never;
    query?: never;
    url: '/admin/campaigns/{id}/actions';
};

export type CampaignAdminActionResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CampaignAdminActionResponse = CampaignAdminActionResponses[keyof CampaignAdminActionResponses];

export type CampaignPrizePaidData = {
    body: {
        /**
         * Operator note recorded with the payout.
         */
        note?: string;
    };
    path?: never;
    query?: never;
    url: '/admin/campaigns/{id}/prizes/{rank}/payment';
};

export type CampaignPrizePaidResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CampaignPrizePaidResponse = CampaignPrizePaidResponses[keyof CampaignPrizePaidResponses];

export type AdminCommentReportResolutionData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/comment-reports/{id}';
};

export type AdminCommentReportResolutionResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminCommentReportResolutionResponse = AdminCommentReportResolutionResponses[keyof AdminCommentReportResolutionResponses];

export type AdminCommentsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/comments';
};

export type AdminCommentsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminCommentsResponse = AdminCommentsResponses[keyof AdminCommentsResponses];

export type AdminCommentsLiveQueueAliasData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/comments/queue';
};

export type AdminCommentsLiveQueueAliasResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminCommentsLiveQueueAliasResponse = AdminCommentsLiveQueueAliasResponses[keyof AdminCommentsLiveQueueAliasResponses];

export type AdminCommentReportResolutionLiveAliasData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/comments/reports/{id}/resolve';
};

export type AdminCommentReportResolutionLiveAliasResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminCommentReportResolutionLiveAliasResponse = AdminCommentReportResolutionLiveAliasResponses[keyof AdminCommentReportResolutionLiveAliasResponses];

export type AdminCommentModerationData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/comments/{id}';
};

export type AdminCommentModerationResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminCommentModerationResponse = AdminCommentModerationResponses[keyof AdminCommentModerationResponses];

export type AdminCommentHideLiveAliasData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/comments/{id}/hide';
};

export type AdminCommentHideLiveAliasResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminCommentHideLiveAliasResponse = AdminCommentHideLiveAliasResponses[keyof AdminCommentHideLiveAliasResponses];

export type AdminCommentRestoreLiveAliasData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/comments/{id}/restore';
};

export type AdminCommentRestoreLiveAliasResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminCommentRestoreLiveAliasResponse = AdminCommentRestoreLiveAliasResponses[keyof AdminCommentRestoreLiveAliasResponses];

export type AdminDomainsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/domains';
};

export type AdminDomainsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminDomainsResponse = AdminDomainsResponses[keyof AdminDomainsResponses];

export type AdminDomainData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/domains/{id}';
};

export type AdminDomainResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminDomainResponse = AdminDomainResponses[keyof AdminDomainResponses];

export type AdminDomainAssetPreviewData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/domains/{id}/assets/{assetId}/preview';
};

export type AdminDomainAssetPreviewResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminDomainAssetPreviewResponse = AdminDomainAssetPreviewResponses[keyof AdminDomainAssetPreviewResponses];

export type AdminDomainAssetReviewData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/domains/{id}/assets/{assetId}/review';
};

export type AdminDomainAssetReviewResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminDomainAssetReviewResponse = AdminDomainAssetReviewResponses[keyof AdminDomainAssetReviewResponses];

export type AdminDomainToggleData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/domains/{id}/toggle-active';
};

export type AdminDomainToggleResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminDomainToggleResponse = AdminDomainToggleResponses[keyof AdminDomainToggleResponses];

export type AdminEsiOverviewData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/esi';
};

export type AdminEsiOverviewResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminEsiOverviewResponse = AdminEsiOverviewResponses[keyof AdminEsiOverviewResponses];

export type AdminEsiEntitiesData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/esi-entities';
};

export type AdminEsiEntitiesResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminEsiEntitiesResponse = AdminEsiEntitiesResponses[keyof AdminEsiEntitiesResponses];

export type AdminEsiLogsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/esi-logs';
};

export type AdminEsiLogsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminEsiLogsResponse = AdminEsiLogsResponses[keyof AdminEsiLogsResponses];

export type AdminModerationData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/moderation';
};

export type AdminModerationResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminModerationResponse = AdminModerationResponses[keyof AdminModerationResponses];

export type AdminModerationLiveQueueAliasData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/moderation/queue';
};

export type AdminModerationLiveQueueAliasResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminModerationLiveQueueAliasResponse = AdminModerationLiveQueueAliasResponses[keyof AdminModerationLiveQueueAliasResponses];

export type AdminModerationReviewData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/moderation/{id}';
};

export type AdminModerationReviewResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminModerationReviewResponse = AdminModerationReviewResponses[keyof AdminModerationReviewResponses];

export type AdminModerationApproveLiveAliasData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/moderation/{id}/approve';
};

export type AdminModerationApproveLiveAliasResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminModerationApproveLiveAliasResponse = AdminModerationApproveLiveAliasResponses[keyof AdminModerationApproveLiveAliasResponses];

export type AdminModerationRejectLiveAliasData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/moderation/{id}/reject';
};

export type AdminModerationRejectLiveAliasResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminModerationRejectLiveAliasResponse = AdminModerationRejectLiveAliasResponses[keyof AdminModerationRejectLiveAliasResponses];

export type AdminOverviewData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/overview';
};

export type AdminOverviewResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminOverviewResponse = AdminOverviewResponses[keyof AdminOverviewResponses];

export type AdminUsersListData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/users';
};

export type AdminUsersListResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminUsersListResponse = AdminUsersListResponses[keyof AdminUsersListResponses];

export type AdminUsersDetailData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/users/{id}';
};

export type AdminUsersDetailResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminUsersDetailResponse = AdminUsersDetailResponses[keyof AdminUsersDetailResponses];

export type AdminUsersSetDiscordData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/users/{id}/set-discord';
};

export type AdminUsersSetDiscordResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminUsersSetDiscordResponse = AdminUsersSetDiscordResponses[keyof AdminUsersSetDiscordResponses];

export type AdminUsersToggleAdminData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/users/{id}/toggle-admin';
};

export type AdminUsersToggleAdminResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AdminUsersToggleAdminResponse = AdminUsersToggleAdminResponses[keyof AdminUsersToggleAdminResponses];

export type WalletAdminData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/wallet';
};

export type WalletAdminResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type WalletAdminResponse = WalletAdminResponses[keyof WalletAdminResponses];

export type WalletAdminAuthorizeData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/wallet/authorize';
};

export type WalletAdminAuthorizeResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type WalletAdminAuthorizeResponse = WalletAdminAuthorizeResponses[keyof WalletAdminAuthorizeResponses];

export type WalletAdminSyncData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/admin/wallet/sync';
};

export type WalletAdminSyncResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type WalletAdminSyncResponse = WalletAdminSyncResponses[keyof WalletAdminSyncResponses];

export type EntityPageDetailAllianceCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/alliance/{id}';
};

export type EntityPageDetailAllianceCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageDetailAllianceCompatResponse = EntityPageDetailAllianceCompatResponses[keyof EntityPageDetailAllianceCompatResponses];

export type EntityPageCorporationsAllianceCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/alliance/{id}/corporations';
};

export type EntityPageCorporationsAllianceCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageCorporationsAllianceCompatResponse = EntityPageCorporationsAllianceCompatResponses[keyof EntityPageCorporationsAllianceCompatResponses];

export type EntityPageIntelAllianceCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/alliance/{id}/intel';
};

export type EntityPageIntelAllianceCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageIntelAllianceCompatResponse = EntityPageIntelAllianceCompatResponses[keyof EntityPageIntelAllianceCompatResponses];

export type EntityPageMembersAllianceCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/alliance/{id}/members';
};

export type EntityPageMembersAllianceCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageMembersAllianceCompatResponse = EntityPageMembersAllianceCompatResponses[keyof EntityPageMembersAllianceCompatResponses];

export type EntityPageStatsAllianceCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/alliance/{id}/stats';
};

export type EntityPageStatsAllianceCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageStatsAllianceCompatResponse = EntityPageStatsAllianceCompatResponses[keyof EntityPageStatsAllianceCompatResponses];

export type EntityTopAllianceCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/alliance/{id}/top';
};

export type EntityTopAllianceCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityTopAllianceCompatResponse = EntityTopAllianceCompatResponses[keyof EntityTopAllianceCompatResponses];

export type AlliancesData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/alliances';
};

export type AlliancesResponses = {
    /**
     * OK
     */
    200: AlliancesResponse;
};

export type AlliancesResponse2 = AlliancesResponses[keyof AlliancesResponses];

export type AlliancesCountData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/alliances/count';
};

export type AlliancesCountResponses = {
    /**
     * OK
     */
    200: AlliancesCountResponse;
};

export type AlliancesCountResponse2 = AlliancesCountResponses[keyof AlliancesCountResponses];

export type AlliancesBatchStatsData = {
    body: {
        /**
         * Start of the window, for type=custom.
         */
        from?: string;
        /**
         * Entity IDs to resolve, at most 100 per request.
         */
        ids: Array<number> | null;
        /**
         * End of the window, for type=custom.
         */
        to?: string;
        /**
         * Aggregation period. Falls back to the type query parameter, then alltime.
         */
        type?: 'alltime' | 'weekly' | 'monthly' | 'custom';
    };
    path?: never;
    query?: never;
    url: '/alliances/stats';
};

export type AlliancesBatchStatsResponses = {
    /**
     * OK
     */
    200: AlliancesBatchStatsResponse;
};

export type AlliancesBatchStatsResponse2 = AlliancesBatchStatsResponses[keyof AlliancesBatchStatsResponses];

export type AllianceData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/alliances/{id}';
};

export type AllianceResponses = {
    /**
     * OK
     */
    200: AllianceResponse;
};

export type AllianceResponse2 = AllianceResponses[keyof AllianceResponses];

export type AllianceCorporationsData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/alliances/{id}/corporations';
};

export type AllianceCorporationsResponses = {
    /**
     * OK
     */
    200: AllianceCorporationsResponse;
};

export type AllianceCorporationsResponse2 = AllianceCorporationsResponses[keyof AllianceCorporationsResponses];

export type AllianceKillsData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/alliances/{id}/kills';
};

export type AllianceKillsResponses = {
    /**
     * OK
     */
    200: AllianceKillsResponse;
};

export type AllianceKillsResponse2 = AllianceKillsResponses[keyof AllianceKillsResponses];

export type AllianceLossesData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/alliances/{id}/losses';
};

export type AllianceLossesResponses = {
    /**
     * OK
     */
    200: AllianceLossesResponse;
};

export type AllianceLossesResponse2 = AllianceLossesResponses[keyof AllianceLossesResponses];

export type AllianceMembersData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/alliances/{id}/members';
};

export type AllianceMembersResponses = {
    /**
     * OK
     */
    200: AllianceMembersResponse;
};

export type AllianceMembersResponse2 = AllianceMembersResponses[keyof AllianceMembersResponses];

export type AllianceStatsAlltimeData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/alliances/{id}/stats/alltime';
};

export type AllianceStatsAlltimeResponses = {
    /**
     * OK
     */
    200: AllianceStatsAlltimeResponse;
};

export type AllianceStatsAlltimeResponse2 = AllianceStatsAlltimeResponses[keyof AllianceStatsAlltimeResponses];

export type AllianceStatsWeeklyData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/alliances/{id}/stats/weekly';
};

export type AllianceStatsWeeklyResponses = {
    /**
     * OK
     */
    200: AllianceStatsWeeklyResponse;
};

export type AllianceStatsWeeklyResponse2 = AllianceStatsWeeklyResponses[keyof AllianceStatsWeeklyResponses];

export type AnnouncementsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/announcements';
};

export type AnnouncementsErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type AnnouncementsError = AnnouncementsErrors[keyof AnnouncementsErrors];

export type AnnouncementsResponses = {
    /**
     * OK
     */
    200: AnnouncementsResponse;
};

export type AnnouncementsResponse2 = AnnouncementsResponses[keyof AnnouncementsResponses];

export type AnnouncementsActiveCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/announcements/active';
};

export type AnnouncementsActiveCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AnnouncementsActiveCompatResponse = AnnouncementsActiveCompatResponses[keyof AnnouncementsActiveCompatResponses];

export type AnnouncementsDismissedCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/announcements/dismissed';
};

export type AnnouncementsDismissedCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AnnouncementsDismissedCompatResponse = AnnouncementsDismissedCompatResponses[keyof AnnouncementsDismissedCompatResponses];

export type AnnouncementDismissCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/announcements/{id}/dismiss';
};

export type AnnouncementDismissCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AnnouncementDismissCompatResponse = AnnouncementDismissCompatResponses[keyof AnnouncementDismissCompatResponses];

export type EveLoginCallbackLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/auth/callback';
};

export type EveLoginCallbackData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/auth/eve/callback';
};

export type EveLoginStartData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/auth/eve/start';
};

export type AuthLoginLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/auth/login';
};

export type AuthLoginLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AuthLoginLegacyResponse = AuthLoginLegacyResponses[keyof AuthLoginLegacyResponses];

export type AuthLogoutLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/auth/logout';
};

export type AuthLogoutLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AuthLogoutLegacyResponse = AuthLogoutLegacyResponses[keyof AuthLogoutLegacyResponses];

export type AuthMeLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/auth/me';
};

export type AuthMeLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AuthMeLegacyResponse = AuthMeLegacyResponses[keyof AuthMeLegacyResponses];

export type AuthTokenInfoLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/auth/token-info';
};

export type AuthTokenInfoLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AuthTokenInfoLegacyResponse = AuthTokenInfoLegacyResponses[keyof AuthTokenInfoLegacyResponses];

export type BackgroundsRedditData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/backgrounds/reddit';
};

export type BackgroundsRedditResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type BackgroundsRedditResponse = BackgroundsRedditResponses[keyof BackgroundsRedditResponses];

export type BattleGeneratorEntitiesData = {
    body: {
        /**
         * Window end, as an ISO 8601 timestamp.
         */
        endTime: string;
        /**
         * Window start, as an ISO 8601 timestamp.
         */
        startTime: string;
        /**
         * Solar systems to scan for killmails.
         */
        systemIds: Array<number> | null;
    };
    path?: never;
    query?: never;
    url: '/battle/generator/entities';
};

export type BattleGeneratorEntitiesResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type BattleGeneratorEntitiesResponse = BattleGeneratorEntitiesResponses[keyof BattleGeneratorEntitiesResponses];

export type BattleGeneratorPreviewData = {
    body: {
        sides: Array<ConflictBattleGeneratorSide> | null;
    };
    path?: never;
    query?: never;
    url: '/battle/generator/preview';
};

export type BattleGeneratorPreviewResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type BattleGeneratorPreviewResponse = BattleGeneratorPreviewResponses[keyof BattleGeneratorPreviewResponses];

export type BattleGeneratorSaveData = {
    body: {
        battle_id: number;
        duration_minutes: number;
        end_time: string;
        is_multi_party: boolean;
        kill_count: number;
        region_id: number | null;
        solar_system_id: number;
        start_time: string;
        teams: Array<ConflictBattleSaveTeam> | null;
        total_isk_destroyed: number;
    };
    path?: never;
    query?: never;
    url: '/battle/generator/save';
};

export type BattleGeneratorSaveResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type BattleGeneratorSaveResponse = BattleGeneratorSaveResponses[keyof BattleGeneratorSaveResponses];

export type KillmailBattleReportData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/battle/killmail/{id}';
};

export type KillmailBattleReportResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type KillmailBattleReportResponse = KillmailBattleReportResponses[keyof KillmailBattleReportResponses];

export type KillmailBattleCompositionData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/battle/killmail/{id}/composition';
};

export type KillmailBattleCompositionResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type KillmailBattleCompositionResponse = KillmailBattleCompositionResponses[keyof KillmailBattleCompositionResponses];

export type KillmailBattleIntelData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/battle/killmail/{id}/intel';
};

export type KillmailBattleIntelResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type KillmailBattleIntelResponse = KillmailBattleIntelResponses[keyof KillmailBattleIntelResponses];

export type KillmailBattleKilllistData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/battle/killmail/{id}/killlist';
};

export type KillmailBattleKilllistResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type KillmailBattleKilllistResponse = KillmailBattleKilllistResponses[keyof KillmailBattleKilllistResponses];

export type KillmailBattleMostValuableData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/battle/killmail/{id}/most-valuable';
};

export type KillmailBattleMostValuableResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type KillmailBattleMostValuableResponse = KillmailBattleMostValuableResponses[keyof KillmailBattleMostValuableResponses];

export type KillmailBattleTimelineData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/battle/killmail/{id}/timeline';
};

export type KillmailBattleTimelineResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type KillmailBattleTimelineResponse = KillmailBattleTimelineResponses[keyof KillmailBattleTimelineResponses];

export type BattleReportData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/battle/{id}';
};

export type BattleReportResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type BattleReportResponse = BattleReportResponses[keyof BattleReportResponses];

export type BattleReportCompositionData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/battle/{id}/composition';
};

export type BattleReportCompositionResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type BattleReportCompositionResponse = BattleReportCompositionResponses[keyof BattleReportCompositionResponses];

export type BattleReportIntelData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/battle/{id}/intel';
};

export type BattleReportIntelResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type BattleReportIntelResponse = BattleReportIntelResponses[keyof BattleReportIntelResponses];

export type BattleReportKilllistData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/battle/{id}/killlist';
};

export type BattleReportKilllistResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type BattleReportKilllistResponse = BattleReportKilllistResponses[keyof BattleReportKilllistResponses];

export type BattleReportMostValuableData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/battle/{id}/most-valuable';
};

export type BattleReportMostValuableResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type BattleReportMostValuableResponse = BattleReportMostValuableResponses[keyof BattleReportMostValuableResponses];

export type BattleReportTimelineData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/battle/{id}/timeline';
};

export type BattleReportTimelineResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type BattleReportTimelineResponse = BattleReportTimelineResponses[keyof BattleReportTimelineResponses];

export type BattlesData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/battles';
};

export type BattlesResponses = {
    /**
     * OK
     */
    200: BattlesResponse;
};

export type BattlesResponse2 = BattlesResponses[keyof BattlesResponses];

export type AllianceBattlesData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/battles/alliance/{id}';
};

export type AllianceBattlesResponses = {
    /**
     * OK
     */
    200: AllianceBattlesResponse;
};

export type AllianceBattlesResponse2 = AllianceBattlesResponses[keyof AllianceBattlesResponses];

export type CorporationBattlesData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/battles/corporation/{id}';
};

export type CorporationBattlesResponses = {
    /**
     * OK
     */
    200: CorporationBattlesResponse;
};

export type CorporationBattlesResponse2 = CorporationBattlesResponses[keyof CorporationBattlesResponses];

export type BattleData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/battles/{id}';
};

export type BattleResponses = {
    /**
     * OK
     */
    200: BattleResponse;
};

export type BattleResponse2 = BattleResponses[keyof BattleResponses];

export type BlogPostsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/blog';
};

export type BlogPostsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type BlogPostsResponse = BlogPostsResponses[keyof BlogPostsResponses];

export type BlogPostData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/blog/{slug}';
};

export type BlogPostResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type BlogPostResponse = BlogPostResponses[keyof BlogPostResponses];

export type BoardsMineCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/boards/mine';
};

export type BoardsMineCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type BoardsMineCompatResponse = BoardsMineCompatResponses[keyof BoardsMineCompatResponses];

export type CampaignCreateLegacyData = {
    body: {
        /**
         * Entities permitted to view a restricted campaign.
         */
        allowedEntities?: unknown;
        /**
         * Free text shown on the campaign page.
         */
        description?: string;
        /**
         * Campaign end. Omit for an open-ended campaign.
         */
        endTime?: unknown;
        /**
         * Location filter: system, constellation or region identifiers.
         */
        location?: unknown;
        /**
         * Campaign name.
         */
        name: string;
        /**
         * Prize pool definition, including any initial contribution.
         */
        prizePool?: unknown;
        /**
         * Participating sides, each naming its entities.
         */
        sides?: unknown;
        /**
         * Campaign start, as a timestamp or ISO 8601 string.
         */
        startTime: unknown;
        /**
         * Who may see the campaign.
         */
        visibility?: unknown;
    };
    path?: never;
    query?: never;
    url: '/campaign/create';
};

export type CampaignCreateLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CampaignCreateLegacyResponse = CampaignCreateLegacyResponses[keyof CampaignCreateLegacyResponses];

export type CampaignDeleteLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/campaign/{id}';
};

export type CampaignDeleteLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CampaignDeleteLegacyResponse = CampaignDeleteLegacyResponses[keyof CampaignDeleteLegacyResponses];

export type CampaignDetailLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/campaign/{id}';
};

export type CampaignDetailLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CampaignDetailLegacyResponse = CampaignDetailLegacyResponses[keyof CampaignDetailLegacyResponses];

export type CampaignUpdateLegacyData = {
    body: {
        /**
         * Replacement viewer list.
         */
        allowedEntities?: unknown;
        /**
         * Archive or restore the campaign.
         */
        archived?: unknown;
        /**
         * New description. An empty string clears it.
         */
        description?: unknown;
        /**
         * New end time.
         */
        endTime?: unknown;
        /**
         * New campaign name.
         */
        name?: unknown;
        /**
         * Resume killmail processing after an edit.
         */
        resumeProcessing?: unknown;
        /**
         * New visibility.
         */
        visibility?: unknown;
    };
    path?: never;
    query?: never;
    url: '/campaign/{id}';
};

export type CampaignUpdateLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CampaignUpdateLegacyResponse = CampaignUpdateLegacyResponses[keyof CampaignUpdateLegacyResponses];

export type CampaignKilllistLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/campaign/{id}/killlist';
};

export type CampaignKilllistLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CampaignKilllistLegacyResponse = CampaignKilllistLegacyResponses[keyof CampaignKilllistLegacyResponses];

export type CampaignPrizeClaimLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/campaign/{id}/prize/claim';
};

export type CampaignPrizeClaimLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CampaignPrizeClaimLegacyResponse = CampaignPrizeClaimLegacyResponses[keyof CampaignPrizeClaimLegacyResponses];

export type CampaignPrizeContributeLegacyData = {
    body: {
        /**
         * ISK amount to contribute.
         */
        amount: unknown;
        /**
         * Caller-supplied idempotency key for the contribution.
         */
        requestId: string;
    };
    path?: never;
    query?: never;
    url: '/campaign/{id}/prize/contribute';
};

export type CampaignPrizeContributeLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CampaignPrizeContributeLegacyResponse = CampaignPrizeContributeLegacyResponses[keyof CampaignPrizeContributeLegacyResponses];

export type CampaignUpdateBrowserLegacyData = {
    body: {
        /**
         * Replacement viewer list.
         */
        allowedEntities?: unknown;
        /**
         * Archive or restore the campaign.
         */
        archived?: unknown;
        /**
         * New description. An empty string clears it.
         */
        description?: unknown;
        /**
         * New end time.
         */
        endTime?: unknown;
        /**
         * New campaign name.
         */
        name?: unknown;
        /**
         * Resume killmail processing after an edit.
         */
        resumeProcessing?: unknown;
        /**
         * New visibility.
         */
        visibility?: unknown;
    };
    path?: never;
    query?: never;
    url: '/campaign/{id}/update';
};

export type CampaignUpdateBrowserLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CampaignUpdateBrowserLegacyResponse = CampaignUpdateBrowserLegacyResponses[keyof CampaignUpdateBrowserLegacyResponses];

export type CampaignsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/campaigns';
};

export type CampaignsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CampaignsResponse = CampaignsResponses[keyof CampaignsResponses];

export type CampaignCreateData = {
    body: {
        /**
         * Entities permitted to view a restricted campaign.
         */
        allowedEntities?: unknown;
        /**
         * Free text shown on the campaign page.
         */
        description?: string;
        /**
         * Campaign end. Omit for an open-ended campaign.
         */
        endTime?: unknown;
        /**
         * Location filter: system, constellation or region identifiers.
         */
        location?: unknown;
        /**
         * Campaign name.
         */
        name: string;
        /**
         * Prize pool definition, including any initial contribution.
         */
        prizePool?: unknown;
        /**
         * Participating sides, each naming its entities.
         */
        sides?: unknown;
        /**
         * Campaign start, as a timestamp or ISO 8601 string.
         */
        startTime: unknown;
        /**
         * Who may see the campaign.
         */
        visibility?: unknown;
    };
    path?: never;
    query?: never;
    url: '/campaigns';
};

export type CampaignCreateResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CampaignCreateResponse = CampaignCreateResponses[keyof CampaignCreateResponses];

export type CampaignDeleteData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/campaigns/{id}';
};

export type CampaignDeleteResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CampaignDeleteResponse = CampaignDeleteResponses[keyof CampaignDeleteResponses];

export type CampaignDetailData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/campaigns/{id}';
};

export type CampaignDetailResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CampaignDetailResponse = CampaignDetailResponses[keyof CampaignDetailResponses];

export type CampaignUpdateData = {
    body: {
        /**
         * Replacement viewer list.
         */
        allowedEntities?: unknown;
        /**
         * Archive or restore the campaign.
         */
        archived?: unknown;
        /**
         * New description. An empty string clears it.
         */
        description?: unknown;
        /**
         * New end time.
         */
        endTime?: unknown;
        /**
         * New campaign name.
         */
        name?: unknown;
        /**
         * Resume killmail processing after an edit.
         */
        resumeProcessing?: unknown;
        /**
         * New visibility.
         */
        visibility?: unknown;
    };
    path?: never;
    query?: never;
    url: '/campaigns/{id}';
};

export type CampaignUpdateResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CampaignUpdateResponse = CampaignUpdateResponses[keyof CampaignUpdateResponses];

export type CampaignKillmailsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/campaigns/{id}/killmails';
};

export type CampaignKillmailsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CampaignKillmailsResponse = CampaignKillmailsResponses[keyof CampaignKillmailsResponses];

export type CampaignPrizeClaimData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/campaigns/{id}/prizes/claim';
};

export type CampaignPrizeClaimResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CampaignPrizeClaimResponse = CampaignPrizeClaimResponses[keyof CampaignPrizeClaimResponses];

export type CampaignPrizeContributeData = {
    body: {
        /**
         * ISK amount to contribute.
         */
        amount: unknown;
        /**
         * Caller-supplied idempotency key for the contribution.
         */
        requestId: string;
    };
    path?: never;
    query?: never;
    url: '/campaigns/{id}/prizes/contributions';
};

export type CampaignPrizeContributeResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CampaignPrizeContributeResponse = CampaignPrizeContributeResponses[keyof CampaignPrizeContributeResponses];

export type EntityPageDetailCharacterCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/character/{id}';
};

export type EntityPageDetailCharacterCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageDetailCharacterCompatResponse = EntityPageDetailCharacterCompatResponses[keyof EntityPageDetailCharacterCompatResponses];

export type EntityPageAchievementsCharacterCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/character/{id}/achievements';
};

export type EntityPageAchievementsCharacterCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageAchievementsCharacterCompatResponse = EntityPageAchievementsCharacterCompatResponses[keyof EntityPageAchievementsCharacterCompatResponses];

export type EntityPageIntelCharacterCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/character/{id}/intel';
};

export type EntityPageIntelCharacterCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageIntelCharacterCompatResponse = EntityPageIntelCharacterCompatResponses[keyof EntityPageIntelCharacterCompatResponses];

export type EntityPageStatsCharacterCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/character/{id}/stats';
};

export type EntityPageStatsCharacterCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageStatsCharacterCompatResponse = EntityPageStatsCharacterCompatResponses[keyof EntityPageStatsCharacterCompatResponses];

export type EntityTopCharacterCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/character/{id}/top';
};

export type EntityTopCharacterCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityTopCharacterCompatResponse = EntityTopCharacterCompatResponses[keyof EntityTopCharacterCompatResponses];

export type CharactersData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/characters';
};

export type CharactersResponses = {
    /**
     * OK
     */
    200: CharactersResponse;
};

export type CharactersResponse2 = CharactersResponses[keyof CharactersResponses];

export type CharacterAnalyzeData = {
    body: {
        /**
         * Characters to analyze, at most 2500 per request.
         */
        character_ids: Array<number> | null;
    };
    path?: never;
    query?: never;
    url: '/characters/analyze';
};

export type CharacterAnalyzeResponses = {
    /**
     * OK
     */
    200: CharacterAnalyzeResponse;
};

export type CharacterAnalyzeResponse2 = CharacterAnalyzeResponses[keyof CharacterAnalyzeResponses];

export type CharactersCountData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/characters/count';
};

export type CharactersCountResponses = {
    /**
     * OK
     */
    200: CharactersCountResponse;
};

export type CharactersCountResponse2 = CharactersCountResponses[keyof CharactersCountResponses];

export type CharactersBatchStatsData = {
    body: {
        /**
         * Start of the window, for type=custom.
         */
        from?: string;
        /**
         * Entity IDs to resolve, at most 100 per request.
         */
        ids: Array<number> | null;
        /**
         * End of the window, for type=custom.
         */
        to?: string;
        /**
         * Aggregation period. Falls back to the type query parameter, then alltime.
         */
        type?: 'alltime' | 'weekly' | 'monthly' | 'custom';
    };
    path?: never;
    query?: never;
    url: '/characters/stats';
};

export type CharactersBatchStatsResponses = {
    /**
     * OK
     */
    200: CharactersBatchStatsResponse;
};

export type CharactersBatchStatsResponse2 = CharactersBatchStatsResponses[keyof CharactersBatchStatsResponses];

export type CharacterData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/characters/{id}';
};

export type CharacterResponses = {
    /**
     * OK
     */
    200: CharacterResponse;
};

export type CharacterResponse2 = CharacterResponses[keyof CharacterResponses];

export type CharacterIntelData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/characters/{id}/intel';
};

export type CharacterIntelResponses = {
    /**
     * OK
     */
    200: CharacterIntelResponse;
};

export type CharacterIntelResponse2 = CharacterIntelResponses[keyof CharacterIntelResponses];

export type CharacterKillsData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/characters/{id}/kills';
};

export type CharacterKillsResponses = {
    /**
     * OK
     */
    200: CharacterKillsResponse;
};

export type CharacterKillsResponse2 = CharacterKillsResponses[keyof CharacterKillsResponses];

export type CharacterLossesData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/characters/{id}/losses';
};

export type CharacterLossesResponses = {
    /**
     * OK
     */
    200: CharacterLossesResponse;
};

export type CharacterLossesResponse2 = CharacterLossesResponses[keyof CharacterLossesResponses];

export type CharacterStatsData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/characters/{id}/stats';
};

export type CharacterStatsResponses = {
    /**
     * OK
     */
    200: CharacterStatsResponse;
};

export type CharacterStatsResponse2 = CharacterStatsResponses[keyof CharacterStatsResponses];

export type CoalitionStatsData = {
    body: {
        /**
         * Restrict to a single day. Takes precedence over days.
         */
        date?: string;
        /**
         * Lookback window ending today, clamped to 1-90. Defaults to 30.
         */
        days?: number | string;
        /**
         * First coalition. Needs at least one alliance or corporation.
         */
        sideA: CoalitionSideBody;
        /**
         * Second coalition. Needs at least one alliance or corporation.
         */
        sideB: CoalitionSideBody;
    };
    path?: never;
    query?: never;
    url: '/coalitions/stats';
};

export type CoalitionStatsResponses = {
    /**
     * OK
     */
    200: CoalitionStatsResponse;
};

export type CoalitionStatsResponse2 = CoalitionStatsResponses[keyof CoalitionStatsResponses];

export type CommentsFeedData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/comments';
};

export type CommentsFeedResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CommentsFeedResponse = CommentsFeedResponses[keyof CommentsFeedResponses];

export type CommentsCreateData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/comments';
};

export type CommentsCreateResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CommentsCreateResponse = CommentsCreateResponses[keyof CommentsCreateResponses];

export type CommentsKlipySearchData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/comments/klipy/search';
};

export type CommentsKlipySearchResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CommentsKlipySearchResponse = CommentsKlipySearchResponses[keyof CommentsKlipySearchResponses];

export type CommentsKlipyTrendingData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/comments/klipy/trending';
};

export type CommentsKlipyTrendingResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CommentsKlipyTrendingResponse = CommentsKlipyTrendingResponses[keyof CommentsKlipyTrendingResponses];

export type CommentsPreviewData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/comments/preview';
};

export type CommentsPreviewResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CommentsPreviewResponse = CommentsPreviewResponses[keyof CommentsPreviewResponses];

export type CommentsThreadData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/comments/thread';
};

export type CommentsThreadResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CommentsThreadResponse = CommentsThreadResponses[keyof CommentsThreadResponses];

export type CommentDeleteData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/comments/{id}';
};

export type CommentDeleteResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CommentDeleteResponse = CommentDeleteResponses[keyof CommentDeleteResponses];

export type CommentDetailData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/comments/{id}';
};

export type CommentDetailResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CommentDetailResponse = CommentDetailResponses[keyof CommentDetailResponses];

export type CommentEditData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/comments/{id}';
};

export type CommentEditResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CommentEditResponse = CommentEditResponses[keyof CommentEditResponses];

export type CommentReportData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/comments/{id}/report';
};

export type CommentReportResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type CommentReportResponse = CommentReportResponses[keyof CommentReportResponses];

export type ConflictBattlesData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/conflicts/battles';
};

export type ConflictBattlesResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type ConflictBattlesResponse = ConflictBattlesResponses[keyof ConflictBattlesResponses];

export type ConflictWarsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/conflicts/wars';
};

export type ConflictWarsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type ConflictWarsResponse = ConflictWarsResponses[keyof ConflictWarsResponses];

export type ConstellationCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/constellation/{id}';
};

export type ConstellationCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type ConstellationCompatResponse = ConstellationCompatResponses[keyof ConstellationCompatResponses];

export type ConstellationKilllistCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/constellation/{id}/killlist';
};

export type ConstellationKilllistCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type ConstellationKilllistCompatResponse = ConstellationKilllistCompatResponses[keyof ConstellationKilllistCompatResponses];

export type ConstellationMostValuableCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/constellation/{id}/most-valuable';
};

export type ConstellationMostValuableCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type ConstellationMostValuableCompatResponse = ConstellationMostValuableCompatResponses[keyof ConstellationMostValuableCompatResponses];

export type EntityPageDetailCorporationCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/corporation/{id}';
};

export type EntityPageDetailCorporationCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageDetailCorporationCompatResponse = EntityPageDetailCorporationCompatResponses[keyof EntityPageDetailCorporationCompatResponses];

export type EntityPageIntelCorporationCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/corporation/{id}/intel';
};

export type EntityPageIntelCorporationCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageIntelCorporationCompatResponse = EntityPageIntelCorporationCompatResponses[keyof EntityPageIntelCorporationCompatResponses];

export type EntityPageMembersCorporationCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/corporation/{id}/members';
};

export type EntityPageMembersCorporationCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageMembersCorporationCompatResponse = EntityPageMembersCorporationCompatResponses[keyof EntityPageMembersCorporationCompatResponses];

export type EntityPageStatsCorporationCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/corporation/{id}/stats';
};

export type EntityPageStatsCorporationCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageStatsCorporationCompatResponse = EntityPageStatsCorporationCompatResponses[keyof EntityPageStatsCorporationCompatResponses];

export type EntityTopCorporationCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/corporation/{id}/top';
};

export type EntityTopCorporationCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityTopCorporationCompatResponse = EntityTopCorporationCompatResponses[keyof EntityTopCorporationCompatResponses];

export type CorporationsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/corporations';
};

export type CorporationsResponses = {
    /**
     * OK
     */
    200: CorporationsResponse;
};

export type CorporationsResponse2 = CorporationsResponses[keyof CorporationsResponses];

export type CorporationsCountData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/corporations/count';
};

export type CorporationsCountResponses = {
    /**
     * OK
     */
    200: CorporationsCountResponse;
};

export type CorporationsCountResponse2 = CorporationsCountResponses[keyof CorporationsCountResponses];

export type CorporationsBatchStatsData = {
    body: {
        /**
         * Start of the window, for type=custom.
         */
        from?: string;
        /**
         * Entity IDs to resolve, at most 100 per request.
         */
        ids: Array<number> | null;
        /**
         * End of the window, for type=custom.
         */
        to?: string;
        /**
         * Aggregation period. Falls back to the type query parameter, then alltime.
         */
        type?: 'alltime' | 'weekly' | 'monthly' | 'custom';
    };
    path?: never;
    query?: never;
    url: '/corporations/stats';
};

export type CorporationsBatchStatsResponses = {
    /**
     * OK
     */
    200: CorporationsBatchStatsResponse;
};

export type CorporationsBatchStatsResponse2 = CorporationsBatchStatsResponses[keyof CorporationsBatchStatsResponses];

export type CorporationData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/corporations/{id}';
};

export type CorporationResponses = {
    /**
     * OK
     */
    200: CorporationResponse;
};

export type CorporationResponse2 = CorporationResponses[keyof CorporationResponses];

export type CorporationKillsData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/corporations/{id}/kills';
};

export type CorporationKillsResponses = {
    /**
     * OK
     */
    200: CorporationKillsResponse;
};

export type CorporationKillsResponse2 = CorporationKillsResponses[keyof CorporationKillsResponses];

export type CorporationLossesData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/corporations/{id}/losses';
};

export type CorporationLossesResponses = {
    /**
     * OK
     */
    200: CorporationLossesResponse;
};

export type CorporationLossesResponse2 = CorporationLossesResponses[keyof CorporationLossesResponses];

export type CorporationMembersData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/corporations/{id}/members';
};

export type CorporationMembersResponses = {
    /**
     * OK
     */
    200: CorporationMembersResponse;
};

export type CorporationMembersResponse2 = CorporationMembersResponses[keyof CorporationMembersResponses];

export type CorporationStatsAlltimeData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/corporations/{id}/stats/alltime';
};

export type CorporationStatsAlltimeResponses = {
    /**
     * OK
     */
    200: CorporationStatsAlltimeResponse;
};

export type CorporationStatsAlltimeResponse2 = CorporationStatsAlltimeResponses[keyof CorporationStatsAlltimeResponses];

export type CorporationStatsWeeklyData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/corporations/{id}/stats/weekly';
};

export type CorporationStatsWeeklyResponses = {
    /**
     * OK
     */
    200: CorporationStatsWeeklyResponse;
};

export type CorporationStatsWeeklyResponse2 = CorporationStatsWeeklyResponses[keyof CorporationStatsWeeklyResponses];

export type DomainConstellationKilllistData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/custom/constellation/{id}/killlist';
};

export type DomainConstellationKilllistResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainConstellationKilllistResponse = DomainConstellationKilllistResponses[keyof DomainConstellationKilllistResponses];

export type DomainKilllistData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/custom/killlist';
};

export type DomainKilllistResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainKilllistResponse = DomainKilllistResponses[keyof DomainKilllistResponses];

export type DomainKillsMostValuableData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/custom/kills/most-valuable';
};

export type DomainKillsMostValuableResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainKillsMostValuableResponse = DomainKillsMostValuableResponses[keyof DomainKillsMostValuableResponses];

export type DomainKillsTopData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/custom/kills/top';
};

export type DomainKillsTopResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainKillsTopResponse = DomainKillsTopResponses[keyof DomainKillsTopResponses];

export type DomainRegionKilllistData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/custom/region/{id}/killlist';
};

export type DomainRegionKilllistResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainRegionKilllistResponse = DomainRegionKilllistResponses[keyof DomainRegionKilllistResponses];

export type DomainStatisticsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/custom/stats';
};

export type DomainStatisticsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainStatisticsResponse = DomainStatisticsResponses[keyof DomainStatisticsResponses];

export type DomainSystemKilllistData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/custom/system/{id}/killlist';
};

export type DomainSystemKilllistResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainSystemKilllistResponse = DomainSystemKilllistResponses[keyof DomainSystemKilllistResponses];

export type DomainBannerOrLogoData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/domains/asset/{id}/{type}';
};

export type DomainBannerOrLogoResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainBannerOrLogoResponse = DomainBannerOrLogoResponses[keyof DomainBannerOrLogoResponses];

export type DomainBackgroundData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/domains/bg/{assetId}';
};

export type DomainBackgroundResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainBackgroundResponse = DomainBackgroundResponses[keyof DomainBackgroundResponses];

export type DomainAssetPreviewData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/domains/preview/{assetId}';
};

export type DomainAssetPreviewResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainAssetPreviewResponse = DomainAssetPreviewResponses[keyof DomainAssetPreviewResponses];

export type EntityResolveData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/entities/resolve';
};

export type EntityResolveResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityResolveResponse = EntityResolveResponses[keyof EntityResolveResponses];

export type EntityPageDetailData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/entities/{type}/{id}';
};

export type EntityPageDetailResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageDetailResponse = EntityPageDetailResponses[keyof EntityPageDetailResponses];

export type EntityPageAchievementsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/entities/{type}/{id}/achievements';
};

export type EntityPageAchievementsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageAchievementsResponse = EntityPageAchievementsResponses[keyof EntityPageAchievementsResponses];

export type EntityPageCorporationsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/entities/{type}/{id}/corporations';
};

export type EntityPageCorporationsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageCorporationsResponse = EntityPageCorporationsResponses[keyof EntityPageCorporationsResponses];

export type EntityPageIntelData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/entities/{type}/{id}/intel';
};

export type EntityPageIntelResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageIntelResponse = EntityPageIntelResponses[keyof EntityPageIntelResponses];

export type EntityPageKilllistData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/entities/{type}/{id}/killlist';
};

export type EntityPageKilllistResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageKilllistResponse = EntityPageKilllistResponses[keyof EntityPageKilllistResponses];

export type EntityPageMembersData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/entities/{type}/{id}/members';
};

export type EntityPageMembersResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageMembersResponse = EntityPageMembersResponses[keyof EntityPageMembersResponses];

export type EntityPageMostValuableData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/entities/{type}/{id}/most-valuable';
};

export type EntityPageMostValuableResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageMostValuableResponse = EntityPageMostValuableResponses[keyof EntityPageMostValuableResponses];

export type EntityPageShipClassesData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/entities/{type}/{id}/ship-classes';
};

export type EntityPageShipClassesResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageShipClassesResponse = EntityPageShipClassesResponses[keyof EntityPageShipClassesResponses];

export type EntityPageStatsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/entities/{type}/{id}/stats';
};

export type EntityPageStatsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageStatsResponse = EntityPageStatsResponses[keyof EntityPageStatsResponses];

export type EntityPageTopListsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/entities/{type}/{id}/top-lists';
};

export type EntityPageTopListsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageTopListsResponse = EntityPageTopListsResponses[keyof EntityPageTopListsResponses];

export type EntityResolveCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/entity/resolve';
};

export type EntityResolveCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityResolveCompatResponse = EntityResolveCompatResponses[keyof EntityResolveCompatResponses];

export type EntityPageKilllistGenericCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/entity/{type}/{id}/killlist';
};

export type EntityPageKilllistGenericCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageKilllistGenericCompatResponse = EntityPageKilllistGenericCompatResponses[keyof EntityPageKilllistGenericCompatResponses];

export type EntityPageMostValuableGenericCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/entity/{type}/{id}/most-valuable';
};

export type EntityPageMostValuableGenericCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageMostValuableGenericCompatResponse = EntityPageMostValuableGenericCompatResponses[keyof EntityPageMostValuableGenericCompatResponses];

export type EntityPageShipClassesGenericCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/entity/{type}/{id}/ship-classes';
};

export type EntityPageShipClassesGenericCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageShipClassesGenericCompatResponse = EntityPageShipClassesGenericCompatResponses[keyof EntityPageShipClassesGenericCompatResponses];

export type EntityPageTopListsGenericCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/entity/{type}/{id}/top-lists';
};

export type EntityPageTopListsGenericCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageTopListsGenericCompatResponse = EntityPageTopListsGenericCompatResponses[keyof EntityPageTopListsGenericCompatResponses];

export type FactionWarDashboardDetailData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/faction-war/{matchup}';
};

export type FactionWarDashboardDetailResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FactionWarDashboardDetailResponse = FactionWarDashboardDetailResponses[keyof FactionWarDashboardDetailResponses];

export type FactionWarDashboardData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/faction-war/{matchup}/dashboard';
};

export type FactionWarDashboardResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FactionWarDashboardResponse = FactionWarDashboardResponses[keyof FactionWarDashboardResponses];

export type FactionWarIntelData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/faction-war/{matchup}/intel';
};

export type FactionWarIntelResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FactionWarIntelResponse = FactionWarIntelResponses[keyof FactionWarIntelResponses];

export type FactionWarMembersData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/faction-war/{matchup}/members';
};

export type FactionWarMembersResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FactionWarMembersResponse = FactionWarMembersResponses[keyof FactionWarMembersResponses];

export type FactionWarOverviewData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/faction-war/{matchup}/overview';
};

export type FactionWarOverviewResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FactionWarOverviewResponse = FactionWarOverviewResponses[keyof FactionWarOverviewResponses];

export type FactionWarSystemsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/faction-war/{matchup}/systems';
};

export type FactionWarSystemsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FactionWarSystemsResponse = FactionWarSystemsResponses[keyof FactionWarSystemsResponses];

export type FactionWarsDashboardData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/faction-wars';
};

export type FactionWarsDashboardResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FactionWarsDashboardResponse = FactionWarsDashboardResponses[keyof FactionWarsDashboardResponses];

export type EntityPageDetailFactionCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/faction/{id}';
};

export type EntityPageDetailFactionCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type EntityPageDetailFactionCompatResponse = EntityPageDetailFactionCompatResponses[keyof EntityPageDetailFactionCompatResponses];

export type FeedIndexData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/feed';
};

export type FeedIndexResponses = {
    /**
     * OK
     */
    200: FeedIndexResponse;
};

export type FeedIndexResponse2 = FeedIndexResponses[keyof FeedIndexResponses];

export type FeedPollData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/feed/poll';
};

export type FeedPollResponses = {
    /**
     * OK
     */
    200: FeedPollResponse;
};

export type FeedPollResponse2 = FeedPollResponses[keyof FeedPollResponses];

export type FeedStatusData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/feed/status';
};

export type FeedStatusResponses = {
    /**
     * OK
     */
    200: FeedStatusResponse;
};

export type FeedStatusResponse2 = FeedStatusResponses[keyof FeedStatusResponses];

export type FeedStreamData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/feed/stream';
};

export type FeedStreamResponses = {
    /**
     * Server-sent events
     */
    200: string;
};

export type FeedStreamResponse = FeedStreamResponses[keyof FeedStreamResponses];

export type FittingCreateLegacyData = {
    body: {
        /**
         * Free text shown with the fitting. Null clears it.
         */
        description?: string | null;
        /**
         * Fitted modules, charges, drones and cargo.
         */
        items: Array<FittingItemBody> | null;
        /**
         * Display name for the fitting.
         */
        name: string | null;
        /**
         * Hull the fitting is for.
         */
        ship_type_id: number | null;
        /**
         * Who may see the fitting.
         */
        visibility: number | null;
    };
    path?: never;
    query?: never;
    url: '/fit';
};

export type FittingCreateLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingCreateLegacyResponse = FittingCreateLegacyResponses[keyof FittingCreateLegacyResponses];

export type FittingDeleteLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fit/{fit_id}';
};

export type FittingDeleteLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingDeleteLegacyResponse = FittingDeleteLegacyResponses[keyof FittingDeleteLegacyResponses];

export type FittingDetailLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fit/{fit_id}';
};

export type FittingDetailLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingDetailLegacyResponse = FittingDetailLegacyResponses[keyof FittingDetailLegacyResponses];

export type FittingUpdateLegacyData = {
    body: {
        /**
         * New description. Null clears it.
         */
        description?: string | null;
        /**
         * Replacement item list. Absent leaves the stored items alone.
         */
        items?: Array<FittingItemBody> | null;
        /**
         * New display name.
         */
        name?: string | null;
        /**
         * New visibility.
         */
        visibility?: number | null;
    };
    path?: never;
    query?: never;
    url: '/fit/{fit_id}';
};

export type FittingUpdateLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingUpdateLegacyResponse = FittingUpdateLegacyResponses[keyof FittingUpdateLegacyResponses];

export type FittingRatingDeleteLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fit/{fit_id}/rating';
};

export type FittingRatingDeleteLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingRatingDeleteLegacyResponse = FittingRatingDeleteLegacyResponses[keyof FittingRatingDeleteLegacyResponses];

export type FittingRatingPutLegacyData = {
    body: {
        /**
         * Rating from 1 to 5.
         */
        rating: number | null;
    };
    path?: never;
    query?: never;
    url: '/fit/{fit_id}/rating';
};

export type FittingRatingPutLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingRatingPutLegacyResponse = FittingRatingPutLegacyResponses[keyof FittingRatingPutLegacyResponses];

export type FittingsCommunityLatestLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fits/community-latest';
};

export type FittingsCommunityLatestLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingsCommunityLatestLegacyResponse = FittingsCommunityLatestLegacyResponses[keyof FittingsCommunityLatestLegacyResponses];

export type FittingsTrendingLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fits/flavors-of-the-week';
};

export type FittingsTrendingLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingsTrendingLegacyResponse = FittingsTrendingLegacyResponses[keyof FittingsTrendingLegacyResponses];

export type FittingsPopularShipsLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fits/popular-ships';
};

export type FittingsPopularShipsLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingsPopularShipsLegacyResponse = FittingsPopularShipsLegacyResponses[keyof FittingsPopularShipsLegacyResponses];

export type FittingsStatsLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fits/quick-stats';
};

export type FittingsStatsLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingsStatsLegacyResponse = FittingsStatsLegacyResponses[keyof FittingsStatsLegacyResponses];

export type FittingsRolesLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fits/roles';
};

export type FittingsRolesLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingsRolesLegacyResponse = FittingsRolesLegacyResponses[keyof FittingsRolesLegacyResponses];

export type FittingsSearchLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fits/search';
};

export type FittingsSearchLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingsSearchLegacyResponse = FittingsSearchLegacyResponses[keyof FittingsSearchLegacyResponses];

export type FittingsAllianceDoctrinesLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fits/top-alliance-doctrines';
};

export type FittingsAllianceDoctrinesLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingsAllianceDoctrinesLegacyResponse = FittingsAllianceDoctrinesLegacyResponses[keyof FittingsAllianceDoctrinesLegacyResponses];

export type FittingsCommunityTopRatedLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fits/top-rated';
};

export type FittingsCommunityTopRatedLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingsCommunityTopRatedLegacyResponse = FittingsCommunityTopRatedLegacyResponses[keyof FittingsCommunityTopRatedLegacyResponses];

export type FittingCreateData = {
    body: {
        /**
         * Free text shown with the fitting. Null clears it.
         */
        description?: string | null;
        /**
         * Fitted modules, charges, drones and cargo.
         */
        items: Array<FittingItemBody> | null;
        /**
         * Display name for the fitting.
         */
        name: string | null;
        /**
         * Hull the fitting is for.
         */
        ship_type_id: number | null;
        /**
         * Who may see the fitting.
         */
        visibility: number | null;
    };
    path?: never;
    query?: never;
    url: '/fittings';
};

export type FittingCreateResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingCreateResponse = FittingCreateResponses[keyof FittingCreateResponses];

export type FittingsCommunityLatestData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fittings/community/latest';
};

export type FittingsCommunityLatestResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingsCommunityLatestResponse = FittingsCommunityLatestResponses[keyof FittingsCommunityLatestResponses];

export type FittingsCommunityTopRatedData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fittings/community/top-rated';
};

export type FittingsCommunityTopRatedResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingsCommunityTopRatedResponse = FittingsCommunityTopRatedResponses[keyof FittingsCommunityTopRatedResponses];

export type FittingsAllianceDoctrinesData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fittings/doctrines/alliances';
};

export type FittingsAllianceDoctrinesResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingsAllianceDoctrinesResponse = FittingsAllianceDoctrinesResponses[keyof FittingsAllianceDoctrinesResponses];

export type FittingsRolesData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fittings/roles';
};

export type FittingsRolesResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingsRolesResponse = FittingsRolesResponses[keyof FittingsRolesResponses];

export type FittingsSearchData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fittings/search';
};

export type FittingsSearchResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingsSearchResponse = FittingsSearchResponses[keyof FittingsSearchResponses];

export type FittingsPopularShipsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fittings/ships/popular';
};

export type FittingsPopularShipsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingsPopularShipsResponse = FittingsPopularShipsResponses[keyof FittingsPopularShipsResponses];

export type FittingsShipFamiliesData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fittings/ships/{id}/families';
};

export type FittingsShipFamiliesResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingsShipFamiliesResponse = FittingsShipFamiliesResponses[keyof FittingsShipFamiliesResponses];

export type FittingsShipMetadataData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fittings/ships/{id}/metadata';
};

export type FittingsShipMetadataResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingsShipMetadataResponse = FittingsShipMetadataResponses[keyof FittingsShipMetadataResponses];

export type FittingsStatsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fittings/stats';
};

export type FittingsStatsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingsStatsResponse = FittingsStatsResponses[keyof FittingsStatsResponses];

export type FittingsTrendingData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fittings/trending';
};

export type FittingsTrendingResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingsTrendingResponse = FittingsTrendingResponses[keyof FittingsTrendingResponses];

export type FittingDeleteData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fittings/{id}';
};

export type FittingDeleteResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingDeleteResponse = FittingDeleteResponses[keyof FittingDeleteResponses];

export type FittingDetailData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fittings/{id}';
};

export type FittingDetailResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingDetailResponse = FittingDetailResponses[keyof FittingDetailResponses];

export type FittingUpdateData = {
    body: {
        /**
         * New description. Null clears it.
         */
        description?: string | null;
        /**
         * Replacement item list. Absent leaves the stored items alone.
         */
        items?: Array<FittingItemBody> | null;
        /**
         * New display name.
         */
        name?: string | null;
        /**
         * New visibility.
         */
        visibility?: number | null;
    };
    path?: never;
    query?: never;
    url: '/fittings/{id}';
};

export type FittingUpdateResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingUpdateResponse = FittingUpdateResponses[keyof FittingUpdateResponses];

export type FittingRatingDeleteData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/fittings/{id}/rating';
};

export type FittingRatingDeleteResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingRatingDeleteResponse = FittingRatingDeleteResponses[keyof FittingRatingDeleteResponses];

export type FittingRatingPutData = {
    body: {
        /**
         * Rating from 1 to 5.
         */
        rating: number | null;
    };
    path?: never;
    query?: never;
    url: '/fittings/{id}/rating';
};

export type FittingRatingPutResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingRatingPutResponse = FittingRatingPutResponses[keyof FittingRatingPutResponses];

export type GraphData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/graph';
};

export type GraphResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type GraphResponse = GraphResponses[keyof GraphResponses];

export type HealthData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/health';
};

export type HealthResponses = {
    /**
     * OK
     */
    200: HealthResponse;
};

export type HealthResponse2 = HealthResponses[keyof HealthResponses];

export type HistoryData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/history';
};

export type HistoryResponses = {
    /**
     * OK
     */
    200: HistoryResponse;
};

export type HistoryResponse2 = HistoryResponses[keyof HistoryResponses];

export type HistoryLatestData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/history/latest';
};

export type HistoryLatestResponses = {
    /**
     * OK
     */
    200: HistoryLatestResponse;
};

export type HistoryLatestResponse2 = HistoryLatestResponses[keyof HistoryLatestResponses];

export type HistoryDateData = {
    body?: never;
    path: {
        date: string;
    };
    query?: never;
    url: '/history/{date}';
};

export type HistoryDateResponses = {
    /**
     * OK
     */
    200: HistoryDateResponse;
};

export type HistoryDateResponse2 = HistoryDateResponses[keyof HistoryDateResponses];

export type ImagesOverviewData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/images';
};

export type ImagesOverviewErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type ImagesOverviewError = ImagesOverviewErrors[keyof ImagesOverviewErrors];

export type ImagesOverviewResponses = {
    /**
     * OK
     */
    200: ImagesOverviewResponse;
};

export type ImagesOverviewResponse2 = ImagesOverviewResponses[keyof ImagesOverviewResponses];

export type ImageAllianceData = {
    body?: never;
    path: {
        id: number;
    };
    query?: {
        /**
         * Maximum width and height in pixels. Images are never upscaled.
         */
        size?: 8 | 16 | 32 | 64 | 128 | 256 | 512 | 1024;
        /**
         * Output format. Auto uses WebP when the request Accept header supports it.
         */
        format?: 'auto' | 'source' | 'webp';
        /**
         * Deprecated alias for format=webp.
         *
         * @deprecated
         */
        imagetype?: 'webp';
    };
    url: '/images/alliances/{id}';
};

export type ImageAllianceErrors = {
    /**
     * Invalid request
     */
    400: unknown;
    /**
     * Image not found
     */
    404: unknown;
    /**
     * Image origin unavailable
     */
    502: unknown;
    /**
     * Image storage unavailable
     */
    503: unknown;
};

export type ImageAllianceResponses = {
    /**
     * Image
     */
    200: Blob | File;
};

export type ImageAllianceResponse = ImageAllianceResponses[keyof ImageAllianceResponses];

export type ImageAllianceVariantData = {
    body?: never;
    path: {
        id: number;
    };
    query?: {
        /**
         * Maximum width and height in pixels. Images are never upscaled.
         */
        size?: 8 | 16 | 32 | 64 | 128 | 256 | 512 | 1024;
        /**
         * Output format. Auto uses WebP when the request Accept header supports it.
         */
        format?: 'auto' | 'source' | 'webp';
        /**
         * Deprecated alias for format=webp.
         *
         * @deprecated
         */
        imagetype?: 'webp';
    };
    url: '/images/alliances/{id}/{variant}';
};

export type ImageAllianceVariantErrors = {
    /**
     * Invalid request
     */
    400: unknown;
    /**
     * Image not found
     */
    404: unknown;
    /**
     * Image origin unavailable
     */
    502: unknown;
    /**
     * Image storage unavailable
     */
    503: unknown;
};

export type ImageAllianceVariantResponses = {
    /**
     * Image
     */
    200: Blob | File;
};

export type ImageAllianceVariantResponse = ImageAllianceVariantResponses[keyof ImageAllianceVariantResponses];

export type ImageCharacterData = {
    body?: never;
    path: {
        id: number;
    };
    query?: {
        /**
         * Maximum width and height in pixels. Images are never upscaled.
         */
        size?: 8 | 16 | 32 | 64 | 128 | 256 | 512 | 1024;
        /**
         * Output format. Auto uses WebP when the request Accept header supports it.
         */
        format?: 'auto' | 'source' | 'webp';
        /**
         * Deprecated alias for format=webp.
         *
         * @deprecated
         */
        imagetype?: 'webp';
    };
    url: '/images/characters/{id}';
};

export type ImageCharacterErrors = {
    /**
     * Invalid request
     */
    400: unknown;
    /**
     * Image not found
     */
    404: unknown;
    /**
     * Image origin unavailable
     */
    502: unknown;
    /**
     * Image storage unavailable
     */
    503: unknown;
};

export type ImageCharacterResponses = {
    /**
     * Image
     */
    200: Blob | File;
};

export type ImageCharacterResponse = ImageCharacterResponses[keyof ImageCharacterResponses];

export type ImageCharacterVariantData = {
    body?: never;
    path: {
        id: number;
    };
    query?: {
        /**
         * Maximum width and height in pixels. Images are never upscaled.
         */
        size?: 8 | 16 | 32 | 64 | 128 | 256 | 512 | 1024;
        /**
         * Output format. Auto uses WebP when the request Accept header supports it.
         */
        format?: 'auto' | 'source' | 'webp';
        /**
         * Deprecated alias for format=webp.
         *
         * @deprecated
         */
        imagetype?: 'webp';
    };
    url: '/images/characters/{id}/{variant}';
};

export type ImageCharacterVariantErrors = {
    /**
     * Invalid request
     */
    400: unknown;
    /**
     * Image not found
     */
    404: unknown;
    /**
     * Image origin unavailable
     */
    502: unknown;
    /**
     * Image storage unavailable
     */
    503: unknown;
};

export type ImageCharacterVariantResponses = {
    /**
     * Image
     */
    200: Blob | File;
};

export type ImageCharacterVariantResponse = ImageCharacterVariantResponses[keyof ImageCharacterVariantResponses];

export type ImageConstellationData = {
    body?: never;
    path: {
        id: string;
    };
    query?: {
        /**
         * Maximum width and height in pixels. Images are never upscaled.
         */
        size?: 32 | 64 | 128;
        /**
         * Output format. Auto uses WebP when the request Accept header supports it.
         */
        format?: 'auto' | 'source' | 'webp';
        /**
         * Deprecated alias for format=webp.
         *
         * @deprecated
         */
        imagetype?: 'webp';
    };
    url: '/images/constellations/{id}';
};

export type ImageConstellationErrors = {
    /**
     * Invalid request
     */
    400: unknown;
    /**
     * Image not found
     */
    404: unknown;
    /**
     * Image origin unavailable
     */
    502: unknown;
    /**
     * Image storage unavailable
     */
    503: unknown;
};

export type ImageConstellationResponses = {
    /**
     * Image
     */
    200: Blob | File;
};

export type ImageConstellationResponse = ImageConstellationResponses[keyof ImageConstellationResponses];

export type ImageCorporationData = {
    body?: never;
    path: {
        id: number;
    };
    query?: {
        /**
         * Maximum width and height in pixels. Images are never upscaled.
         */
        size?: 8 | 16 | 32 | 64 | 128 | 256 | 512 | 1024;
        /**
         * Output format. Auto uses WebP when the request Accept header supports it.
         */
        format?: 'auto' | 'source' | 'webp';
        /**
         * Deprecated alias for format=webp.
         *
         * @deprecated
         */
        imagetype?: 'webp';
    };
    url: '/images/corporations/{id}';
};

export type ImageCorporationErrors = {
    /**
     * Invalid request
     */
    400: unknown;
    /**
     * Image not found
     */
    404: unknown;
    /**
     * Image origin unavailable
     */
    502: unknown;
    /**
     * Image storage unavailable
     */
    503: unknown;
};

export type ImageCorporationResponses = {
    /**
     * Image
     */
    200: Blob | File;
};

export type ImageCorporationResponse = ImageCorporationResponses[keyof ImageCorporationResponses];

export type ImageCorporationVariantData = {
    body?: never;
    path: {
        id: number;
    };
    query?: {
        /**
         * Maximum width and height in pixels. Images are never upscaled.
         */
        size?: 8 | 16 | 32 | 64 | 128 | 256 | 512 | 1024;
        /**
         * Output format. Auto uses WebP when the request Accept header supports it.
         */
        format?: 'auto' | 'source' | 'webp';
        /**
         * Deprecated alias for format=webp.
         *
         * @deprecated
         */
        imagetype?: 'webp';
    };
    url: '/images/corporations/{id}/{variant}';
};

export type ImageCorporationVariantErrors = {
    /**
     * Invalid request
     */
    400: unknown;
    /**
     * Image not found
     */
    404: unknown;
    /**
     * Image origin unavailable
     */
    502: unknown;
    /**
     * Image storage unavailable
     */
    503: unknown;
};

export type ImageCorporationVariantResponses = {
    /**
     * Image
     */
    200: Blob | File;
};

export type ImageCorporationVariantResponse = ImageCorporationVariantResponses[keyof ImageCorporationVariantResponses];

export type ImageDomainBackgroundData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/images/domains/background/{assetId}';
};

export type ImageDomainBackgroundResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type ImageDomainBackgroundResponse = ImageDomainBackgroundResponses[keyof ImageDomainBackgroundResponses];

export type ImageDomainAssetPreviewData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/images/domains/preview/{assetId}';
};

export type ImageDomainAssetPreviewResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type ImageDomainAssetPreviewResponse = ImageDomainAssetPreviewResponses[keyof ImageDomainAssetPreviewResponses];

export type ImageDomainBannerOrLogoData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/images/domains/{id}/{type}';
};

export type ImageDomainBannerOrLogoResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type ImageDomainBannerOrLogoResponse = ImageDomainBannerOrLogoResponses[keyof ImageDomainBannerOrLogoResponses];

export type ImageKillmailSocialData = {
    body?: never;
    path: {
        id: number;
    };
    query?: never;
    url: '/images/killmail/{id}/social.png';
};

export type ImageKillmailSocialErrors = {
    /**
     * Invalid request
     */
    400: unknown;
    /**
     * Image not found
     */
    404: unknown;
    /**
     * Image origin unavailable
     */
    502: unknown;
    /**
     * Image storage unavailable
     */
    503: unknown;
};

export type ImageKillmailSocialResponses = {
    /**
     * Image
     */
    200: Blob | File;
};

export type ImageKillmailSocialResponse = ImageKillmailSocialResponses[keyof ImageKillmailSocialResponses];

export type ImageOldCharacterData = {
    body?: never;
    path: {
        id: number;
    };
    query?: {
        /**
         * Maximum width and height in pixels. Legacy portraits are never upscaled beyond their 256px source.
         */
        size?: 8 | 16 | 32 | 64 | 128 | 256;
        /**
         * Output format. Auto uses WebP when the request Accept header supports it.
         */
        format?: 'auto' | 'source' | 'webp';
        /**
         * Deprecated alias for format=webp.
         *
         * @deprecated
         */
        imagetype?: 'webp';
    };
    url: '/images/oldcharacters/{id}';
};

export type ImageOldCharacterErrors = {
    /**
     * Invalid request
     */
    400: unknown;
    /**
     * Image not found
     */
    404: unknown;
    /**
     * Image origin unavailable
     */
    502: unknown;
    /**
     * Image storage unavailable
     */
    503: unknown;
};

export type ImageOldCharacterResponses = {
    /**
     * Image
     */
    200: Blob | File;
};

export type ImageOldCharacterResponse = ImageOldCharacterResponses[keyof ImageOldCharacterResponses];

export type ImageRegionData = {
    body?: never;
    path: {
        id: string;
    };
    query?: {
        /**
         * Maximum width and height in pixels. Images are never upscaled.
         */
        size?: 32 | 64 | 128;
        /**
         * Output format. Auto uses WebP when the request Accept header supports it.
         */
        format?: 'auto' | 'source' | 'webp';
        /**
         * Deprecated alias for format=webp.
         *
         * @deprecated
         */
        imagetype?: 'webp';
    };
    url: '/images/regions/{id}';
};

export type ImageRegionErrors = {
    /**
     * Invalid request
     */
    400: unknown;
    /**
     * Image not found
     */
    404: unknown;
    /**
     * Image origin unavailable
     */
    502: unknown;
    /**
     * Image storage unavailable
     */
    503: unknown;
};

export type ImageRegionResponses = {
    /**
     * Image
     */
    200: Blob | File;
};

export type ImageRegionResponse = ImageRegionResponses[keyof ImageRegionResponses];

export type ImageSystemData = {
    body?: never;
    path: {
        id: string;
    };
    query?: {
        /**
         * Maximum width and height in pixels. Images are never upscaled.
         */
        size?: 32 | 64 | 128;
        /**
         * Output format. Auto uses WebP when the request Accept header supports it.
         */
        format?: 'auto' | 'source' | 'webp';
        /**
         * Deprecated alias for format=webp.
         *
         * @deprecated
         */
        imagetype?: 'webp';
    };
    url: '/images/systems/{id}';
};

export type ImageSystemErrors = {
    /**
     * Invalid request
     */
    400: unknown;
    /**
     * Image not found
     */
    404: unknown;
    /**
     * Image origin unavailable
     */
    502: unknown;
    /**
     * Image storage unavailable
     */
    503: unknown;
};

export type ImageSystemResponses = {
    /**
     * Image
     */
    200: Blob | File;
};

export type ImageSystemResponse = ImageSystemResponses[keyof ImageSystemResponses];

export type ImageTypeData = {
    body?: never;
    path: {
        id: number;
        variant: string;
    };
    query?: {
        /**
         * Maximum width and height in pixels. Images are never upscaled.
         */
        size?: 8 | 16 | 32 | 64 | 128 | 256 | 512 | 1024;
        /**
         * Output format. Auto uses WebP when the request Accept header supports it.
         */
        format?: 'auto' | 'source' | 'webp';
        /**
         * Deprecated alias for format=webp.
         *
         * @deprecated
         */
        imagetype?: 'webp';
    };
    url: '/images/types/{id}/{variant}';
};

export type ImageTypeErrors = {
    /**
     * Invalid request
     */
    400: unknown;
    /**
     * Image not found
     */
    404: unknown;
    /**
     * Image origin unavailable
     */
    502: unknown;
    /**
     * Image storage unavailable
     */
    503: unknown;
};

export type ImageTypeResponses = {
    /**
     * Image
     */
    200: Blob | File;
};

export type ImageTypeResponse = ImageTypeResponses[keyof ImageTypeResponses];

export type ImageUiData = {
    body?: never;
    path: {
        name: string;
    };
    query?: {
        /**
         * Maximum width and height in pixels. Images are never upscaled.
         */
        size?: 32 | 64 | 128;
        /**
         * Output format. Auto uses WebP when the request Accept header supports it.
         */
        format?: 'auto' | 'source' | 'webp';
        /**
         * Deprecated alias for format=webp.
         *
         * @deprecated
         */
        imagetype?: 'webp';
    };
    url: '/images/ui/{name}';
};

export type ImageUiErrors = {
    /**
     * Invalid request
     */
    400: unknown;
    /**
     * Image not found
     */
    404: unknown;
    /**
     * Image origin unavailable
     */
    502: unknown;
    /**
     * Image storage unavailable
     */
    503: unknown;
};

export type ImageUiResponses = {
    /**
     * Image
     */
    200: Blob | File;
};

export type ImageUiResponse = ImageUiResponses[keyof ImageUiResponses];

export type TypeCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/item/{id}';
};

export type TypeCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type TypeCompatResponse = TypeCompatResponses[keyof TypeCompatResponses];

export type FittingsShipMetadataLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/item/{id}/fit-meta';
};

export type FittingsShipMetadataLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingsShipMetadataLegacyResponse = FittingsShipMetadataLegacyResponses[keyof FittingsShipMetadataLegacyResponses];

export type FittingsShipFamiliesLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/item/{id}/fittings';
};

export type FittingsShipFamiliesLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type FittingsShipFamiliesLegacyResponse = FittingsShipFamiliesLegacyResponses[keyof FittingsShipFamiliesLegacyResponses];

export type ItemKilllistCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/item/{id}/killlist';
};

export type ItemKilllistCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type ItemKilllistCompatResponse = ItemKilllistCompatResponses[keyof ItemKilllistCompatResponses];

export type KilllistData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/killlist';
};

export type KilllistResponses = {
    /**
     * OK
     */
    200: {
        cursor: number | null;
        hasMore: boolean;
        kills: Array<{
            attacker_count: number;
            final_blow_alliance_id: number | null;
            final_blow_alliance_name: string | null;
            final_blow_character_id: number | null;
            final_blow_character_name: string | null;
            final_blow_corporation_id: number | null;
            final_blow_corporation_name: string | null;
            final_blow_ship_name: string | null;
            final_blow_ship_type_id: number | null;
            is_npc: boolean;
            is_solo: boolean;
            killmail_hash: string;
            killmail_id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            killmail_time: string;
            meta_group_id: number | null;
            region_id: number | null;
            region_name: string | null;
            ship_group_name: string | null;
            ship_market_path: string | null;
            ship_name: string | null;
            ship_type_id: number | null;
            solar_system_id: number;
            solar_system_name: string | null;
            solar_system_security: number | null;
            total_value: number;
            victim_alliance_id: number | null;
            victim_alliance_name: string | null;
            victim_character_id: number | null;
            victim_character_name: string | null;
            victim_corporation_id: number | null;
            victim_corporation_name: string | null;
            [key: string]: unknown;
        }>;
        totalPages?: number;
    };
};

export type KilllistResponse = KilllistResponses[keyof KilllistResponses];

export type KilllistAdvancedData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/killlist/advanced';
};

export type KilllistAdvancedResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type KilllistAdvancedResponse = KilllistAdvancedResponses[keyof KilllistAdvancedResponses];

export type KillmailSubmitData = {
    body: {
        /**
         * ESI killmail links. Joined with newlines and parsed the same way as text.
         */
        links?: Array<string> | null;
        /**
         * Free text containing ESI killmail links, one per line. Takes precedence over links.
         */
        text?: string;
    };
    path?: never;
    query?: never;
    url: '/killmail/post';
};

export type KillmailSubmitResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type KillmailSubmitResponse = KillmailSubmitResponses[keyof KillmailSubmitResponses];

export type KillmailDetailLegacyData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/killmail/{id}';
};

export type KillmailDetailLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type KillmailDetailLegacyResponse = KillmailDetailLegacyResponses[keyof KillmailDetailLegacyResponses];

export type KillmailExistsLegacyData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/killmail/{id}/exists';
};

export type KillmailExistsLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type KillmailExistsLegacyResponse = KillmailExistsLegacyResponses[keyof KillmailExistsLegacyResponses];

export type KillmailEditorFitLegacyData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/killmail/{id}/fit';
};

export type KillmailEditorFitLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type KillmailEditorFitLegacyResponse = KillmailEditorFitLegacyResponses[keyof KillmailEditorFitLegacyResponses];

export type KillmailSiblingsLegacyData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/killmail/{id}/siblings';
};

export type KillmailSiblingsLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type KillmailSiblingsLegacyResponse = KillmailSiblingsLegacyResponses[keyof KillmailSiblingsLegacyResponses];

export type KillmailsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/killmails';
};

export type KillmailsResponses = {
    /**
     * OK
     */
    200: KillmailsResponse;
};

export type KillmailsResponse2 = KillmailsResponses[keyof KillmailsResponses];

export type KillmailsCountData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/killmails/count';
};

export type KillmailsCountResponses = {
    /**
     * OK
     */
    200: KillmailsCountResponse;
};

export type KillmailsCountResponse2 = KillmailsCountResponses[keyof KillmailsCountResponses];

export type KillmailSearchData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/killmails/search';
};

export type KillmailSearchResponses = {
    /**
     * OK
     */
    200: KillmailSearchResponse;
};

export type KillmailSearchResponse2 = KillmailSearchResponses[keyof KillmailSearchResponses];

export type KillmailData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/killmails/{id}';
};

export type KillmailResponses = {
    /**
     * OK
     */
    200: KillmailResponse;
};

export type KillmailResponse2 = KillmailResponses[keyof KillmailResponses];

export type KillmailEditorFitData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/killmails/{id}/editor-fit';
};

export type KillmailEditorFitResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type KillmailEditorFitResponse = KillmailEditorFitResponses[keyof KillmailEditorFitResponses];

export type KillmailEftData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/killmails/{id}/eft';
};

export type KillmailEftResponses = {
    /**
     * EFT fitting
     */
    200: string;
};

export type KillmailEftResponse = KillmailEftResponses[keyof KillmailEftResponses];

export type KillmailEsiData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/killmails/{id}/esi';
};

export type KillmailEsiResponses = {
    /**
     * OK
     */
    200: KillmailEsiResponse;
};

export type KillmailEsiResponse2 = KillmailEsiResponses[keyof KillmailEsiResponses];

export type KillmailExistsData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/killmails/{id}/exists';
};

export type KillmailExistsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type KillmailExistsResponse = KillmailExistsResponses[keyof KillmailExistsResponses];

export type KillmailFittingData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/killmails/{id}/fitting';
};

export type KillmailFittingResponses = {
    /**
     * OK
     */
    200: KillmailFittingResponse;
};

export type KillmailFittingResponse2 = KillmailFittingResponses[keyof KillmailFittingResponses];

export type KillmailSiblingsData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/killmails/{id}/siblings';
};

export type KillmailSiblingsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type KillmailSiblingsResponse = KillmailSiblingsResponses[keyof KillmailSiblingsResponses];

export type KillsMostValuableData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/kills/most-valuable';
};

export type KillsMostValuableResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type KillsMostValuableResponse = KillsMostValuableResponses[keyof KillsMostValuableResponses];

export type KillsTopData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/kills/top';
};

export type KillsTopResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type KillsTopResponse = KillsTopResponses[keyof KillsTopResponses];

export type LegacyArchiveAutocompleteData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/legacy/autocomplete';
};

export type LegacyArchiveAutocompleteResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type LegacyArchiveAutocompleteResponse = LegacyArchiveAutocompleteResponses[keyof LegacyArchiveAutocompleteResponses];

export type LegacyArchiveKillData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/legacy/kill/{id}';
};

export type LegacyArchiveKillResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type LegacyArchiveKillResponse = LegacyArchiveKillResponses[keyof LegacyArchiveKillResponses];

export type LegacyArchiveKillsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/legacy/kills';
};

export type LegacyArchiveKillsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type LegacyArchiveKillsResponse = LegacyArchiveKillsResponses[keyof LegacyArchiveKillsResponses];

export type LegacyArchiveStatsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/legacy/stats';
};

export type LegacyArchiveStatsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type LegacyArchiveStatsResponse = LegacyArchiveStatsResponses[keyof LegacyArchiveStatsResponses];

export type LegacyArchiveTopData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/legacy/top';
};

export type LegacyArchiveTopResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type LegacyArchiveTopResponse = LegacyArchiveTopResponses[keyof LegacyArchiveTopResponses];

export type LocationData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/location';
};

export type LocationResponses = {
    /**
     * OK
     */
    200: LocationResponse;
};

export type LocationResponse2 = LocationResponses[keyof LocationResponses];

export type MapRegionData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/map/region/{id}';
};

export type MapRegionResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type MapRegionResponse = MapRegionResponses[keyof MapRegionResponses];

export type MapRegionsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/map/regions';
};

export type MapRegionsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type MapRegionsResponse = MapRegionsResponses[keyof MapRegionsResponses];

export type MapScopeData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/map/scope';
};

export type MapScopeResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type MapScopeResponse = MapScopeResponses[keyof MapScopeResponses];

export type MarketGroupItemsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/market/groups/{id}/items';
};

export type MarketGroupItemsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type MarketGroupItemsResponse = MarketGroupItemsResponses[keyof MarketGroupItemsResponses];

export type MarketTreeData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/market/tree';
};

export type MarketTreeResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type MarketTreeResponse = MarketTreeResponses[keyof MarketTreeResponses];

export type ShipMatchupData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/matchup';
};

export type ShipMatchupResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type ShipMatchupResponse = ShipMatchupResponses[keyof ShipMatchupResponses];

export type MeData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me';
};

export type MeResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type MeResponse = MeResponses[keyof MeResponses];

export type AccountDismissedAnnouncementsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/announcements/dismissed';
};

export type AccountDismissedAnnouncementsErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type AccountDismissedAnnouncementsError = AccountDismissedAnnouncementsErrors[keyof AccountDismissedAnnouncementsErrors];

export type AccountDismissedAnnouncementsResponses = {
    /**
     * OK
     */
    200: DismissedAnnouncementIdsResponse;
};

export type AccountDismissedAnnouncementsResponse = AccountDismissedAnnouncementsResponses[keyof AccountDismissedAnnouncementsResponses];

export type AccountAnnouncementDismissalData = {
    body?: never;
    path: {
        id: number;
    };
    query?: never;
    url: '/me/announcements/{id}/dismissal';
};

export type AccountAnnouncementDismissalErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type AccountAnnouncementDismissalError = AccountAnnouncementDismissalErrors[keyof AccountAnnouncementDismissalErrors];

export type AccountAnnouncementDismissalResponses = {
    /**
     * OK
     */
    200: AnnouncementDismissalResponse;
};

export type AccountAnnouncementDismissalResponse = AccountAnnouncementDismissalResponses[keyof AccountAnnouncementDismissalResponses];

export type AccountBoardsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/boards';
};

export type AccountBoardsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AccountBoardsResponse = AccountBoardsResponses[keyof AccountBoardsResponses];

export type MyCommentsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/comments';
};

export type MyCommentsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type MyCommentsResponse = MyCommentsResponses[keyof MyCommentsResponses];

export type MyCommentDeleteData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/comments/{id}';
};

export type MyCommentDeleteResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type MyCommentDeleteResponse = MyCommentDeleteResponses[keyof MyCommentDeleteResponses];

export type AccountDescriptionsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/descriptions';
};

export type AccountDescriptionsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AccountDescriptionsResponse = AccountDescriptionsResponses[keyof AccountDescriptionsResponses];

export type AccountDescriptionUpdateData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/descriptions';
};

export type AccountDescriptionUpdateResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AccountDescriptionUpdateResponse = AccountDescriptionUpdateResponses[keyof AccountDescriptionUpdateResponses];

export type DomainsMineData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/domains';
};

export type DomainsMineResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainsMineResponse = DomainsMineResponses[keyof DomainsMineResponses];

export type DomainCreateData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/domains';
};

export type DomainCreateResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainCreateResponse = DomainCreateResponses[keyof DomainCreateResponses];

export type DomainSubdomainCheckData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/domains/check-subdomain';
};

export type DomainSubdomainCheckResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainSubdomainCheckResponse = DomainSubdomainCheckResponses[keyof DomainSubdomainCheckResponses];

export type DomainDeleteData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/domains/{id}';
};

export type DomainDeleteResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainDeleteResponse = DomainDeleteResponses[keyof DomainDeleteResponses];

export type DomainUpdateData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/domains/{id}';
};

export type DomainUpdateResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainUpdateResponse = DomainUpdateResponses[keyof DomainUpdateResponses];

export type DomainAssetsDeleteTypeData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/domains/{id}/assets';
};

export type DomainAssetsDeleteTypeResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainAssetsDeleteTypeResponse = DomainAssetsDeleteTypeResponses[keyof DomainAssetsDeleteTypeResponses];

export type DomainAssetUploadData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/domains/{id}/assets';
};

export type DomainAssetUploadResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainAssetUploadResponse = DomainAssetUploadResponses[keyof DomainAssetUploadResponses];

export type DomainAssetDeleteData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/domains/{id}/assets/{assetId}';
};

export type DomainAssetDeleteResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainAssetDeleteResponse = DomainAssetDeleteResponses[keyof DomainAssetDeleteResponses];

export type DomainCampaignSearchData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/domains/{id}/campaigns';
};

export type DomainCampaignSearchResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainCampaignSearchResponse = DomainCampaignSearchResponses[keyof DomainCampaignSearchResponses];

export type AccountEsiData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/esi';
};

export type AccountEsiResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AccountEsiResponse = AccountEsiResponses[keyof AccountEsiResponses];

export type AccountEsiLogsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/esi/logs';
};

export type AccountEsiLogsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AccountEsiLogsResponse = AccountEsiLogsResponses[keyof AccountEsiLogsResponses];

export type AccountNotificationReadCursorData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/notifications/read-cursor';
};

export type AccountNotificationReadCursorResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AccountNotificationReadCursorResponse = AccountNotificationReadCursorResponses[keyof AccountNotificationReadCursorResponses];

export type AccountNotificationRepliesData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/notifications/replies';
};

export type AccountNotificationRepliesResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AccountNotificationRepliesResponse = AccountNotificationRepliesResponses[keyof AccountNotificationRepliesResponses];

export type AccountOverviewData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/overview';
};

export type AccountOverviewResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AccountOverviewResponse = AccountOverviewResponses[keyof AccountOverviewResponses];

export type AccountPreferencesData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/preferences';
};

export type AccountPreferencesResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AccountPreferencesResponse = AccountPreferencesResponses[keyof AccountPreferencesResponses];

export type AccountPreferencesUpdateData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/preferences';
};

export type AccountPreferencesUpdateResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type AccountPreferencesUpdateResponse = AccountPreferencesUpdateResponses[keyof AccountPreferencesUpdateResponses];

export type SessionDeleteData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/session';
};

export type SessionDeleteResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type SessionDeleteResponse = SessionDeleteResponses[keyof SessionDeleteResponses];

export type OtherSessionsRevokeData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/sessions';
};

export type OtherSessionsRevokeResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type OtherSessionsRevokeResponse = OtherSessionsRevokeResponses[keyof OtherSessionsRevokeResponses];

export type SessionsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/sessions';
};

export type SessionsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type SessionsResponse = SessionsResponses[keyof SessionsResponses];

export type SessionRevokeData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/sessions/{id}';
};

export type SessionRevokeResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type SessionRevokeResponse = SessionRevokeResponses[keyof SessionRevokeResponses];

export type MeSettingsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/settings';
};

export type MeSettingsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type MeSettingsResponse = MeSettingsResponses[keyof MeSettingsResponses];

export type WalletAccountData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/wallet';
};

export type WalletAccountResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type WalletAccountResponse = WalletAccountResponses[keyof WalletAccountResponses];

export type WalletAccountBalanceData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/me/wallet/balance';
};

export type WalletAccountBalanceResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type WalletAccountBalanceResponse = WalletAccountBalanceResponses[keyof WalletAccountBalanceResponses];

export type NotificationMarkReadCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/notifications/mark-read';
};

export type NotificationMarkReadCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type NotificationMarkReadCompatResponse = NotificationMarkReadCompatResponses[keyof NotificationMarkReadCompatResponses];

export type NotificationRepliesCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/notifications/replies';
};

export type NotificationRepliesCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type NotificationRepliesCompatResponse = NotificationRepliesCompatResponses[keyof NotificationRepliesCompatResponses];

export type BulkPricesData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/prices/bulk';
};

export type BulkPricesResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type BulkPricesResponse = BulkPricesResponses[keyof BulkPricesResponses];

export type RegionCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/region/{id}';
};

export type RegionCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type RegionCompatResponse = RegionCompatResponses[keyof RegionCompatResponses];

export type RegionKilllistCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/region/{id}/killlist';
};

export type RegionKilllistCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type RegionKilllistCompatResponse = RegionKilllistCompatResponses[keyof RegionKilllistCompatResponses];

export type RegionMostValuableCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/region/{id}/most-valuable';
};

export type RegionMostValuableCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type RegionMostValuableCompatResponse = RegionMostValuableCompatResponses[keyof RegionMostValuableCompatResponses];

export type ResolveData = {
    body: {
        /**
         * Exact entity names to resolve. Matching is case-sensitive and exact; use /search for fuzzy lookup.
         */
        names: Array<string> | null;
        /**
         * Which entity table to resolve against.
         */
        type?: 'character' | 'corporation' | 'alliance';
    };
    path?: never;
    query?: never;
    url: '/resolve';
};

export type ResolveResponses = {
    /**
     * OK
     */
    200: ResolveResponse;
};

export type ResolveResponse2 = ResolveResponses[keyof ResolveResponses];

export type DscanSaveData = {
    body: {
        /**
         * Raw directional scan text. Used by the dscan routes.
         */
        dscan?: string;
        /**
         * Character names. Used by the local scan routes.
         */
        names?: Array<string>;
        /**
         * Analyzed scan output to store alongside the input.
         */
        result: unknown;
    };
    path?: never;
    query?: never;
    url: '/scans/dscan';
};

export type DscanSaveResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DscanSaveResponse = DscanSaveResponses[keyof DscanSaveResponses];

export type DscanAnalyzeData = {
    body: {
        /**
         * Raw directional scan text, one result per line, tab separated.
         */
        dscan: string;
    };
    path?: never;
    query?: never;
    url: '/scans/dscan/analyze';
};

export type DscanAnalyzeResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DscanAnalyzeResponse = DscanAnalyzeResponses[keyof DscanAnalyzeResponses];

export type DscanGetData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/scans/dscan/{hash}';
};

export type DscanGetResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DscanGetResponse = DscanGetResponses[keyof DscanGetResponses];

export type LocalscanSaveData = {
    body: {
        /**
         * Raw directional scan text. Used by the dscan routes.
         */
        dscan?: string;
        /**
         * Character names. Used by the local scan routes.
         */
        names?: Array<string>;
        /**
         * Analyzed scan output to store alongside the input.
         */
        result: unknown;
    };
    path?: never;
    query?: never;
    url: '/scans/local';
};

export type LocalscanSaveResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type LocalscanSaveResponse = LocalscanSaveResponses[keyof LocalscanSaveResponses];

export type LocalscanAnalyzeData = {
    body: Array<string> | null;
    path?: never;
    query?: never;
    url: '/scans/local/analyze';
};

export type LocalscanAnalyzeResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type LocalscanAnalyzeResponse = LocalscanAnalyzeResponses[keyof LocalscanAnalyzeResponses];

export type LocalscanGetData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/scans/local/{hash}';
};

export type LocalscanGetResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type LocalscanGetResponse = LocalscanGetResponses[keyof LocalscanGetResponses];

export type SdeBloodlinesData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/sde/bloodlines';
};

export type SdeBloodlinesResponses = {
    /**
     * OK
     */
    200: SdeBloodlinesResponse;
};

export type SdeBloodlinesResponse2 = SdeBloodlinesResponses[keyof SdeBloodlinesResponses];

export type SdeBloodlineData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/bloodlines/{id}';
};

export type SdeBloodlineResponses = {
    /**
     * OK
     */
    200: SdeBloodlineResponse;
};

export type SdeBloodlineResponse2 = SdeBloodlineResponses[keyof SdeBloodlineResponses];

export type SdeCategoriesData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/sde/categories';
};

export type SdeCategoriesResponses = {
    /**
     * OK
     */
    200: SdeCategoriesResponse;
};

export type SdeCategoriesResponse2 = SdeCategoriesResponses[keyof SdeCategoriesResponses];

export type SdeCategoryData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/categories/{id}';
};

export type SdeCategoryResponses = {
    /**
     * OK
     */
    200: SdeCategoryResponse;
};

export type SdeCategoryResponse2 = SdeCategoryResponses[keyof SdeCategoryResponses];

export type SdeCelestialData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/celestials/{id}';
};

export type SdeCelestialResponses = {
    /**
     * OK
     */
    200: SdeCelestialResponse;
};

export type SdeCelestialResponse2 = SdeCelestialResponses[keyof SdeCelestialResponses];

export type SdeConstellationsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/sde/constellations';
};

export type SdeConstellationsResponses = {
    /**
     * OK
     */
    200: SdeConstellationsResponse;
};

export type SdeConstellationsResponse2 = SdeConstellationsResponses[keyof SdeConstellationsResponses];

export type SdeConstellationData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/constellations/{id}';
};

export type SdeConstellationResponses = {
    /**
     * OK
     */
    200: SdeConstellationResponse;
};

export type SdeConstellationResponse2 = SdeConstellationResponses[keyof SdeConstellationResponses];

export type SdeCustomPricesData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/sde/custom-prices';
};

export type SdeCustomPricesResponses = {
    /**
     * OK
     */
    200: SdeCustomPricesResponse;
};

export type SdeCustomPricesResponse2 = SdeCustomPricesResponses[keyof SdeCustomPricesResponses];

export type SdeFactionsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/sde/factions';
};

export type SdeFactionsResponses = {
    /**
     * OK
     */
    200: SdeFactionsResponse;
};

export type SdeFactionsResponse2 = SdeFactionsResponses[keyof SdeFactionsResponses];

export type SdeFactionData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/factions/{id}';
};

export type SdeFactionResponses = {
    /**
     * OK
     */
    200: SdeFactionResponse;
};

export type SdeFactionResponse2 = SdeFactionResponses[keyof SdeFactionResponses];

export type SdeFlagsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/sde/flags';
};

export type SdeFlagsResponses = {
    /**
     * OK
     */
    200: SdeFlagsResponse;
};

export type SdeFlagsResponse2 = SdeFlagsResponses[keyof SdeFlagsResponses];

export type SdeGroupsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/sde/groups';
};

export type SdeGroupsResponses = {
    /**
     * OK
     */
    200: SdeGroupsResponse;
};

export type SdeGroupsResponse2 = SdeGroupsResponses[keyof SdeGroupsResponses];

export type SdeGroupData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/groups/{id}';
};

export type SdeGroupResponses = {
    /**
     * OK
     */
    200: SdeGroupResponse;
};

export type SdeGroupResponse2 = SdeGroupResponses[keyof SdeGroupResponses];

export type SdeMarketGroupsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/sde/market-groups';
};

export type SdeMarketGroupsResponses = {
    /**
     * OK
     */
    200: SdeMarketGroupsResponse;
};

export type SdeMarketGroupsResponse2 = SdeMarketGroupsResponses[keyof SdeMarketGroupsResponses];

export type SdeMarketGroupData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/market-groups/{id}';
};

export type SdeMarketGroupResponses = {
    /**
     * OK
     */
    200: SdeMarketGroupResponse;
};

export type SdeMarketGroupResponse2 = SdeMarketGroupResponses[keyof SdeMarketGroupResponses];

export type SdeMetaGroupsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/sde/meta-groups';
};

export type SdeMetaGroupsResponses = {
    /**
     * OK
     */
    200: SdeMetaGroupsResponse;
};

export type SdeMetaGroupsResponse2 = SdeMetaGroupsResponses[keyof SdeMetaGroupsResponses];

export type SdeNpcCorporationsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/sde/npc-corporations';
};

export type SdeNpcCorporationsResponses = {
    /**
     * OK
     */
    200: SdeNpcCorporationsResponse;
};

export type SdeNpcCorporationsResponse2 = SdeNpcCorporationsResponses[keyof SdeNpcCorporationsResponses];

export type SdeNpcCorporationData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/npc-corporations/{id}';
};

export type SdeNpcCorporationResponses = {
    /**
     * OK
     */
    200: SdeNpcCorporationResponse;
};

export type SdeNpcCorporationResponse2 = SdeNpcCorporationResponses[keyof SdeNpcCorporationResponses];

export type SdePricesData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/prices/{id}';
};

export type SdePricesResponses = {
    /**
     * OK
     */
    200: SdePricesResponse;
};

export type SdePricesResponse2 = SdePricesResponses[keyof SdePricesResponses];

export type SdeRacesData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/sde/races';
};

export type SdeRacesResponses = {
    /**
     * OK
     */
    200: SdeRacesResponse;
};

export type SdeRacesResponse2 = SdeRacesResponses[keyof SdeRacesResponses];

export type SdeRaceData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/races/{id}';
};

export type SdeRaceResponses = {
    /**
     * OK
     */
    200: SdeRaceResponse;
};

export type SdeRaceResponse2 = SdeRaceResponses[keyof SdeRaceResponses];

export type SdeRegionsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/sde/regions';
};

export type SdeRegionsResponses = {
    /**
     * OK
     */
    200: SdeRegionsResponse;
};

export type SdeRegionsResponse2 = SdeRegionsResponses[keyof SdeRegionsResponses];

export type SdeRegionData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/regions/{id}';
};

export type SdeRegionResponses = {
    /**
     * OK
     */
    200: SdeRegionResponse;
};

export type SdeRegionResponse2 = SdeRegionResponses[keyof SdeRegionResponses];

export type SdeRegionKillsData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/regions/{id}/kills';
};

export type SdeRegionKillsResponses = {
    /**
     * OK
     */
    200: SdeRegionKillsResponse;
};

export type SdeRegionKillsResponse2 = SdeRegionKillsResponses[keyof SdeRegionKillsResponses];

export type SdeSovereigntyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/sde/sovereignty';
};

export type SdeSovereigntyResponses = {
    /**
     * OK
     */
    200: SdeSovereigntyResponse;
};

export type SdeSovereigntyResponse2 = SdeSovereigntyResponses[keyof SdeSovereigntyResponses];

export type SdeSovereigntySystemData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/sovereignty/{id}';
};

export type SdeSovereigntySystemResponses = {
    /**
     * OK
     */
    200: SdeSovereigntySystemResponse;
};

export type SdeSovereigntySystemResponse2 = SdeSovereigntySystemResponses[keyof SdeSovereigntySystemResponses];

export type SdeSovereigntyHistoryData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/sovereignty/{id}/history';
};

export type SdeSovereigntyHistoryResponses = {
    /**
     * OK
     */
    200: SdeSovereigntyHistoryResponse;
};

export type SdeSovereigntyHistoryResponse2 = SdeSovereigntyHistoryResponses[keyof SdeSovereigntyHistoryResponses];

export type SdeStationOperationsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/sde/station-operations';
};

export type SdeStationOperationsResponses = {
    /**
     * OK
     */
    200: SdeStationOperationsResponse;
};

export type SdeStationOperationsResponse2 = SdeStationOperationsResponses[keyof SdeStationOperationsResponses];

export type SdeStationOperationData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/station-operations/{id}';
};

export type SdeStationOperationResponses = {
    /**
     * OK
     */
    200: SdeStationOperationResponse;
};

export type SdeStationOperationResponse2 = SdeStationOperationResponses[keyof SdeStationOperationResponses];

export type SdeStationsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/sde/stations';
};

export type SdeStationsResponses = {
    /**
     * OK
     */
    200: SdeStationsResponse;
};

export type SdeStationsResponse2 = SdeStationsResponses[keyof SdeStationsResponses];

export type SdeStationData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/stations/{id}';
};

export type SdeStationResponses = {
    /**
     * OK
     */
    200: SdeStationResponse;
};

export type SdeStationResponse2 = SdeStationResponses[keyof SdeStationResponses];

export type SdeStructuresData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/sde/structures';
};

export type SdeStructuresResponses = {
    /**
     * OK
     */
    200: SdeStructuresResponse;
};

export type SdeStructuresResponse2 = SdeStructuresResponses[keyof SdeStructuresResponses];

export type SdeStructureData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/structures/{id}';
};

export type SdeStructureResponses = {
    /**
     * OK
     */
    200: SdeStructureResponse;
};

export type SdeStructureResponse2 = SdeStructureResponses[keyof SdeStructureResponses];

export type SdeSystemsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/sde/systems';
};

export type SdeSystemsResponses = {
    /**
     * OK
     */
    200: SdeSystemsResponse;
};

export type SdeSystemsResponse2 = SdeSystemsResponses[keyof SdeSystemsResponses];

export type SdeSystemData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/systems/{id}';
};

export type SdeSystemResponses = {
    /**
     * OK
     */
    200: SdeSystemResponse;
};

export type SdeSystemResponse2 = SdeSystemResponses[keyof SdeSystemResponses];

export type SdeSystemCelestialsData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/systems/{id}/celestials';
};

export type SdeSystemCelestialsResponses = {
    /**
     * OK
     */
    200: SdeSystemCelestialsResponse;
};

export type SdeSystemCelestialsResponse2 = SdeSystemCelestialsResponses[keyof SdeSystemCelestialsResponses];

export type SdeSystemJumpsData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/systems/{id}/jumps';
};

export type SdeSystemJumpsResponses = {
    /**
     * OK
     */
    200: SdeSystemJumpsResponse;
};

export type SdeSystemJumpsResponse2 = SdeSystemJumpsResponses[keyof SdeSystemJumpsResponses];

export type SdeSystemKillsData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/systems/{id}/kills';
};

export type SdeSystemKillsResponses = {
    /**
     * OK
     */
    200: SdeSystemKillsResponse;
};

export type SdeSystemKillsResponse2 = SdeSystemKillsResponses[keyof SdeSystemKillsResponses];

export type SdeTypesData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/sde/types';
};

export type SdeTypesResponses = {
    /**
     * OK
     */
    200: SdeTypesResponse;
};

export type SdeTypesResponse2 = SdeTypesResponses[keyof SdeTypesResponses];

export type SdeTypeData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/types/{id}';
};

export type SdeTypeResponses = {
    /**
     * OK
     */
    200: SdeTypeResponse;
};

export type SdeTypeResponse2 = SdeTypeResponses[keyof SdeTypeResponses];

export type SdeTypeDogmaData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/types/{id}/dogma';
};

export type SdeTypeDogmaResponses = {
    /**
     * OK
     */
    200: SdeTypeDogmaResponse;
};

export type SdeTypeDogmaResponse2 = SdeTypeDogmaResponses[keyof SdeTypeDogmaResponses];

export type SdeTypeInsuranceData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/types/{id}/insurance';
};

export type SdeTypeInsuranceResponses = {
    /**
     * OK
     */
    200: SdeTypeInsuranceResponse;
};

export type SdeTypeInsuranceResponse2 = SdeTypeInsuranceResponses[keyof SdeTypeInsuranceResponses];

export type SdeTypeMaterialsData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/sde/types/{id}/materials';
};

export type SdeTypeMaterialsResponses = {
    /**
     * OK
     */
    200: SdeTypeMaterialsResponse;
};

export type SdeTypeMaterialsResponse2 = SdeTypeMaterialsResponses[keyof SdeTypeMaterialsResponses];

export type SearchData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/search';
};

export type SearchResponses = {
    /**
     * OK
     */
    200: SearchResponse;
};

export type SearchResponse2 = SearchResponses[keyof SearchResponses];

export type ShipKilllistCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/ship/{id}/killlist';
};

export type ShipKilllistCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type ShipKilllistCompatResponse = ShipKilllistCompatResponses[keyof ShipKilllistCompatResponses];

export type ShipFittingsData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/ships/{id}/fittings';
};

export type ShipFittingsResponses = {
    /**
     * OK
     */
    200: ShipFittingsResponse;
};

export type ShipFittingsResponse2 = ShipFittingsResponses[keyof ShipFittingsResponses];

export type SiteConfigurationData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/site';
};

export type SiteConfigurationErrors = {
    /**
     * Internal Server Error
     */
    500: ErrorModel;
    /**
     * Service Unavailable
     */
    503: ErrorModel;
};

export type SiteConfigurationError = SiteConfigurationErrors[keyof SiteConfigurationErrors];

export type SiteConfigurationResponses = {
    /**
     * OK
     */
    200: SiteConfigurationResponse;
};

export type SiteConfigurationResponse2 = SiteConfigurationResponses[keyof SiteConfigurationResponses];

export type SitemapData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/sitemap/{kind}';
};

export type SitemapResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type SitemapResponse = SitemapResponses[keyof SitemapResponses];

export type GlobalStatsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/stats';
};

export type GlobalStatsResponses = {
    /**
     * OK
     */
    200: GlobalStatsResponse;
};

export type GlobalStatsResponse2 = GlobalStatsResponses[keyof GlobalStatsResponses];

export type StatsRankingsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/stats/rankings';
};

export type StatsRankingsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type StatsRankingsResponse = StatsRankingsResponses[keyof StatsRankingsResponses];

export type SystemCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/system/{id}';
};

export type SystemCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type SystemCompatResponse = SystemCompatResponses[keyof SystemCompatResponses];

export type SystemKilllistCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/system/{id}/killlist';
};

export type SystemKilllistCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type SystemKilllistCompatResponse = SystemKilllistCompatResponses[keyof SystemKilllistCompatResponses];

export type SystemMostValuableCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/system/{id}/most-valuable';
};

export type SystemMostValuableCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type SystemMostValuableCompatResponse = SystemMostValuableCompatResponses[keyof SystemMostValuableCompatResponses];

export type DscanAnalyzeLegacyData = {
    body: {
        /**
         * Raw directional scan text, one result per line, tab separated.
         */
        dscan: string;
    };
    path?: never;
    query?: never;
    url: '/tools/dscan';
};

export type DscanAnalyzeLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DscanAnalyzeLegacyResponse = DscanAnalyzeLegacyResponses[keyof DscanAnalyzeLegacyResponses];

export type DscanSaveLegacyData = {
    body: {
        /**
         * Raw directional scan text. Used by the dscan routes.
         */
        dscan?: string;
        /**
         * Character names. Used by the local scan routes.
         */
        names?: Array<string>;
        /**
         * Analyzed scan output to store alongside the input.
         */
        result: unknown;
    };
    path?: never;
    query?: never;
    url: '/tools/dscan/save';
};

export type DscanSaveLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DscanSaveLegacyResponse = DscanSaveLegacyResponses[keyof DscanSaveLegacyResponses];

export type DscanGetLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/tools/dscan/{hash}';
};

export type DscanGetLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DscanGetLegacyResponse = DscanGetLegacyResponses[keyof DscanGetLegacyResponses];

export type LocalscanAnalyzeLegacyData = {
    body: Array<string> | null;
    path?: never;
    query?: never;
    url: '/tools/localscan';
};

export type LocalscanAnalyzeLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type LocalscanAnalyzeLegacyResponse = LocalscanAnalyzeLegacyResponses[keyof LocalscanAnalyzeLegacyResponses];

export type LocalscanSaveLegacyData = {
    body: {
        /**
         * Raw directional scan text. Used by the dscan routes.
         */
        dscan?: string;
        /**
         * Character names. Used by the local scan routes.
         */
        names?: Array<string>;
        /**
         * Analyzed scan output to store alongside the input.
         */
        result: unknown;
    };
    path?: never;
    query?: never;
    url: '/tools/localscan/save';
};

export type LocalscanSaveLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type LocalscanSaveLegacyResponse = LocalscanSaveLegacyResponses[keyof LocalscanSaveLegacyResponses];

export type LocalscanGetLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/tools/localscan/{hash}';
};

export type LocalscanGetLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type LocalscanGetLegacyResponse = LocalscanGetLegacyResponses[keyof LocalscanGetLegacyResponses];

export type UniverseConstellationData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/universe/constellations/{id}';
};

export type UniverseConstellationResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type UniverseConstellationResponse = UniverseConstellationResponses[keyof UniverseConstellationResponses];

export type UniverseConstellationKillmailsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/universe/constellations/{id}/killmails';
};

export type UniverseConstellationKillmailsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type UniverseConstellationKillmailsResponse = UniverseConstellationKillmailsResponses[keyof UniverseConstellationKillmailsResponses];

export type UniverseConstellationMostValuableData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/universe/constellations/{id}/most-valuable';
};

export type UniverseConstellationMostValuableResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type UniverseConstellationMostValuableResponse = UniverseConstellationMostValuableResponses[keyof UniverseConstellationMostValuableResponses];

export type UniverseRegionData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/universe/regions/{id}';
};

export type UniverseRegionResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type UniverseRegionResponse = UniverseRegionResponses[keyof UniverseRegionResponses];

export type UniverseRegionKillmailsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/universe/regions/{id}/killmails';
};

export type UniverseRegionKillmailsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type UniverseRegionKillmailsResponse = UniverseRegionKillmailsResponses[keyof UniverseRegionKillmailsResponses];

export type UniverseRegionMostValuableData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/universe/regions/{id}/most-valuable';
};

export type UniverseRegionMostValuableResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type UniverseRegionMostValuableResponse = UniverseRegionMostValuableResponses[keyof UniverseRegionMostValuableResponses];

export type UniverseSystemData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/universe/systems/{id}';
};

export type UniverseSystemResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type UniverseSystemResponse = UniverseSystemResponses[keyof UniverseSystemResponses];

export type UniverseSystemKillmailsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/universe/systems/{id}/killmails';
};

export type UniverseSystemKillmailsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type UniverseSystemKillmailsResponse = UniverseSystemKillmailsResponses[keyof UniverseSystemKillmailsResponses];

export type UniverseSystemMostValuableData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/universe/systems/{id}/most-valuable';
};

export type UniverseSystemMostValuableResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type UniverseSystemMostValuableResponse = UniverseSystemMostValuableResponses[keyof UniverseSystemMostValuableResponses];

export type UniverseTypeData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/universe/types/{id}';
};

export type UniverseTypeResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type UniverseTypeResponse = UniverseTypeResponses[keyof UniverseTypeResponses];

export type UniverseTypeKillmailsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/universe/types/{id}/killmails';
};

export type UniverseTypeKillmailsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type UniverseTypeKillmailsResponse = UniverseTypeKillmailsResponses[keyof UniverseTypeKillmailsResponses];

export type UserBoardsUpdateCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/boards';
};

export type UserBoardsUpdateCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type UserBoardsUpdateCompatResponse = UserBoardsUpdateCompatResponses[keyof UserBoardsUpdateCompatResponses];

export type MyCommentsLiveAliasData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/comments';
};

export type MyCommentsLiveAliasResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type MyCommentsLiveAliasResponse = MyCommentsLiveAliasResponses[keyof MyCommentsLiveAliasResponses];

export type MyCommentDeleteLiveAliasData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/comments/{id}';
};

export type MyCommentDeleteLiveAliasResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type MyCommentDeleteLiveAliasResponse = MyCommentDeleteLiveAliasResponses[keyof MyCommentDeleteLiveAliasResponses];

export type UserDescriptionUpdateCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/descriptions';
};

export type UserDescriptionUpdateCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type UserDescriptionUpdateCompatResponse = UserDescriptionUpdateCompatResponses[keyof UserDescriptionUpdateCompatResponses];

export type DomainsMineCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/domains';
};

export type DomainsMineCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainsMineCompatResponse = DomainsMineCompatResponses[keyof DomainsMineCompatResponses];

export type DomainCreateCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/domains';
};

export type DomainCreateCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainCreateCompatResponse = DomainCreateCompatResponses[keyof DomainCreateCompatResponses];

export type DomainSubdomainCheckCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/domains/check-subdomain';
};

export type DomainSubdomainCheckCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainSubdomainCheckCompatResponse = DomainSubdomainCheckCompatResponses[keyof DomainSubdomainCheckCompatResponses];

export type DomainDeleteCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/domains/{id}';
};

export type DomainDeleteCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainDeleteCompatResponse = DomainDeleteCompatResponses[keyof DomainDeleteCompatResponses];

export type DomainUpdatePatchCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/domains/{id}';
};

export type DomainUpdatePatchCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainUpdatePatchCompatResponse = DomainUpdatePatchCompatResponses[keyof DomainUpdatePatchCompatResponses];

export type DomainUpdatePutCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/domains/{id}';
};

export type DomainUpdatePutCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainUpdatePutCompatResponse = DomainUpdatePutCompatResponses[keyof DomainUpdatePutCompatResponses];

export type DomainAssetDeleteCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/domains/{id}/assets/{assetId}';
};

export type DomainAssetDeleteCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainAssetDeleteCompatResponse = DomainAssetDeleteCompatResponses[keyof DomainAssetDeleteCompatResponses];

export type DomainCampaignSearchCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/domains/{id}/campaigns/search';
};

export type DomainCampaignSearchCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainCampaignSearchCompatResponse = DomainCampaignSearchCompatResponses[keyof DomainCampaignSearchCompatResponses];

export type DomainAssetsDeleteTypeCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/domains/{id}/upload';
};

export type DomainAssetsDeleteTypeCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainAssetsDeleteTypeCompatResponse = DomainAssetsDeleteTypeCompatResponses[keyof DomainAssetsDeleteTypeCompatResponses];

export type DomainAssetUploadCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/domains/{id}/upload';
};

export type DomainAssetUploadCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type DomainAssetUploadCompatResponse = DomainAssetUploadCompatResponses[keyof DomainAssetUploadCompatResponses];

export type UserEsiCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/esi';
};

export type UserEsiCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type UserEsiCompatResponse = UserEsiCompatResponses[keyof UserEsiCompatResponses];

export type UserEsiLogsCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/esi-logs';
};

export type UserEsiLogsCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type UserEsiLogsCompatResponse = UserEsiLogsCompatResponses[keyof UserEsiLogsCompatResponses];

export type UserManageableEntitiesCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/manageable-entities';
};

export type UserManageableEntitiesCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type UserManageableEntitiesCompatResponse = UserManageableEntitiesCompatResponses[keyof UserManageableEntitiesCompatResponses];

export type UserOverviewCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/overview';
};

export type UserOverviewCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type UserOverviewCompatResponse = UserOverviewCompatResponses[keyof UserOverviewCompatResponses];

export type UserPreferencesCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/preferences';
};

export type UserPreferencesCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type UserPreferencesCompatResponse = UserPreferencesCompatResponses[keyof UserPreferencesCompatResponses];

export type UserPreferencesUpdateCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/preferences';
};

export type UserPreferencesUpdateCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type UserPreferencesUpdateCompatResponse = UserPreferencesUpdateCompatResponses[keyof UserPreferencesUpdateCompatResponses];

export type UserSessionsLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/sessions';
};

export type UserSessionsLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type UserSessionsLegacyResponse = UserSessionsLegacyResponses[keyof UserSessionsLegacyResponses];

export type OtherSessionsRevokeLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/sessions/revoke-others';
};

export type OtherSessionsRevokeLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type OtherSessionsRevokeLegacyResponse = OtherSessionsRevokeLegacyResponses[keyof OtherSessionsRevokeLegacyResponses];

export type UserSessionRevokeLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/sessions/{id}/revoke';
};

export type UserSessionRevokeLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type UserSessionRevokeLegacyResponse = UserSessionRevokeLegacyResponses[keyof UserSessionRevokeLegacyResponses];

export type UserThemeUpdateCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/theme';
};

export type UserThemeUpdateCompatResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type UserThemeUpdateCompatResponse = UserThemeUpdateCompatResponses[keyof UserThemeUpdateCompatResponses];

export type WalletAccountLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/wallet';
};

export type WalletAccountLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type WalletAccountLegacyResponse = WalletAccountLegacyResponses[keyof WalletAccountLegacyResponses];

export type WalletAccountBalanceLegacyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/user/wallet/balance';
};

export type WalletAccountBalanceLegacyResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type WalletAccountBalanceLegacyResponse = WalletAccountBalanceLegacyResponses[keyof WalletAccountBalanceLegacyResponses];

export type WalletPublicData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/wallet';
};

export type WalletPublicResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type WalletPublicResponse = WalletPublicResponses[keyof WalletPublicResponses];

export type WarDashboardDetailData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/war/{id}';
};

export type WarDashboardDetailResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type WarDashboardDetailResponse = WarDashboardDetailResponses[keyof WarDashboardDetailResponses];

export type WarDashboardData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/war/{id}/dashboard';
};

export type WarDashboardResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type WarDashboardResponse = WarDashboardResponses[keyof WarDashboardResponses];

export type WarIntelData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/war/{id}/intel';
};

export type WarIntelResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type WarIntelResponse = WarIntelResponses[keyof WarIntelResponses];

export type WarKilllistData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/war/{id}/killlist';
};

export type WarKilllistResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type WarKilllistResponse = WarKilllistResponses[keyof WarKilllistResponses];

export type WarMembersData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/war/{id}/members';
};

export type WarMembersResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type WarMembersResponse = WarMembersResponses[keyof WarMembersResponses];

export type WarLeaderboardsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/war/{id}/stats';
};

export type WarLeaderboardsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type WarLeaderboardsResponse = WarLeaderboardsResponses[keyof WarLeaderboardsResponses];

export type WarsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/wars';
};

export type WarsResponses = {
    /**
     * OK
     */
    200: WarsResponse;
};

export type WarsResponse2 = WarsResponses[keyof WarsResponses];

export type WarsEligibleData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/wars/eligible';
};

export type WarsEligibleResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type WarsEligibleResponse = WarsEligibleResponses[keyof WarsEligibleResponses];

export type WarsOverviewStatsData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/wars/stats';
};

export type WarsOverviewStatsResponses = {
    /**
     * OK
     */
    200: {
        [key: string]: unknown;
    };
};

export type WarsOverviewStatsResponse = WarsOverviewStatsResponses[keyof WarsOverviewStatsResponses];

export type WarData = {
    body?: never;
    path: {
        id: string;
    };
    query?: never;
    url: '/wars/{id}';
};

export type WarResponses = {
    /**
     * OK
     */
    200: WarResponse;
};

export type WarResponse2 = WarResponses[keyof WarResponses];
