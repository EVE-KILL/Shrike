// Generated from shared/api.openapi.json by @hey-api/openapi-ts.
// Do not edit by hand; run `make gen-api-client`.

export type ClientOptions = {
    baseUrl: `${string}://${string}/api` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | `${string}://${string}` | (string & {});
};

export type AccountBoardsDocument = {
    dismissed: Array<string>;
    pinned: Array<string>;
};

export type AllianceNode = {
    alliance_id: number;
    name: string | null;
    ticker: string | null;
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

export type BattleAllianceSummary = {
    alliance_id: number;
    isk_destroyed: number;
    kills: number;
    losses: number;
    name: string | null;
    ticker: string | null;
};

export type BattleMember = {
    alliance_id: number | null;
    alliance_name?: string;
    alliance_ticker?: string;
    corporation_count?: number;
    corporation_id?: number;
    corporation_name?: string;
    corporation_ticker?: string;
    isk_destroyed: number;
    isk_lost: number;
    kills: number;
    losses: number;
};

export type BattleReportInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    battle_id: number;
    format?: 'json' | 'summary';
    level?: 'alliance' | 'corp';
};

export type BattleReportOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    battle_id: number;
    duration_minutes?: number;
    end_time?: string;
    is_custom?: boolean;
    is_multi_party?: boolean;
    kill_count?: number;
    level?: string;
    start_time?: string;
    summary?: string;
    system?: BattleSystemSummary;
    teams?: Array<BattleTeam> | null;
    total_isk_destroyed?: number;
    url: string;
};

export type BattleSystemSummary = {
    id: number;
    name: string | null;
    region_id: number | null;
    region_name: string | null;
};

export type BattleTeam = {
    members: Array<BattleMember> | null;
    team_index: number;
    total_isk_destroyed: number;
    total_isk_lost: number;
    total_kills: number;
    total_losses: number;
};

export type BattleTeamSummary = {
    alliance_count: number;
    team_index: number;
    top_alliances: Array<BattleAllianceSummary> | null;
};

export type BattleTopAlliance = {
    alliance_id: number;
    name: string | null;
    ticker: string | null;
};

export type CampaignLocationDocument = {
    /**
     * An integer. A numeric string is accepted for compatibility.
     */
    constellationIds?: Array<number | string>;
    /**
     * An integer. A numeric string is accepted for compatibility.
     */
    regionIds?: Array<number | string>;
    /**
     * An integer. A numeric string is accepted for compatibility.
     */
    systemIds?: Array<number | string>;
};

export type CampaignPrizePoolDocument = {
    enabled: boolean;
    fundingRequestId?: string;
    /**
     * A number. A numeric string is accepted for compatibility.
     */
    initialContribution?: number | string;
    /**
     * An integer. A numeric string is accepted for compatibility.
     */
    metric?: number | string;
    /**
     * An integer. A numeric string is accepted for compatibility.
     */
    payoutPercentages?: Array<number | string>;
    /**
     * An integer. A numeric string is accepted for compatibility.
     */
    winnerCount?: number | string;
};

export type CapDiff = {
    a: string | null;
    b: string | null;
};

export type CharacterHistoryInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    /**
     * Character name or id.
     */
    entity: string | number;
    /**
     * ISO datetime lower bound. Default all history.
     */
    since?: string;
};

export type CharacterHistoryOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    character: CharacterRef;
    notes?: string;
    observation_count: number;
    period_count: number;
    periods: Array<MembershipPeriod> | null;
};

export type CharacterIntelInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    /**
     * Character name or id.
     */
    entity: string | number;
    limit?: number;
    /**
     * Must be character if specified.
     */
    type?: 'character';
};

export type CharacterIntelOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    characters: Array<IntelCharacter> | null;
    count: number;
    entity: Entity;
    window_days: number;
};

export type CharacterRef = {
    id: number;
    name: string;
    url: string;
};

export type CoalitionEdge = {
    a: AllianceNode;
    allied_battles: number;
    b: AllianceNode;
    enemy_battles: number;
    total_battles: number;
};

export type CoalitionFocus = {
    alliance_id: number;
    name: string | null;
};

export type CoalitionGraphInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    /**
     * Alliance name or id for an ego graph.
     */
    focus_alliance?: string | number;
    limit_edges?: number;
    min_alliance_battles?: number;
    min_edge_weight?: number;
    /**
     * ISO datetime lower bound. Default 30 days ago.
     */
    since?: string;
    /**
     * ISO datetime upper bound. Default now.
     */
    until?: string;
};

export type CoalitionGraphOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    allied_edges: Array<CoalitionEdge> | null;
    edge_count: number;
    enemy_edges: Array<CoalitionEdge> | null;
    focus?: CoalitionFocus;
    mixed_edges: Array<CoalitionEdge> | null;
    node_count: number;
    nodes: Array<AllianceNode> | null;
    window: TimeWindow;
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

export type CompareInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    /**
     * First character name or id.
     */
    a: string | number;
    /**
     * Second character name or id.
     */
    b: string | number;
};

export type CompareOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    a: Entity;
    a_killed_b: HeadToHead;
    b: Entity;
    b_killed_a: HeadToHead;
    shared_systems: Array<SharedSystem> | null;
    shared_wingmates: number;
    window_days: number;
};

export type ConflictBattleGeneratorEntity = {
    id: number;
    type: string;
};

export type ConflictBattleGeneratorSide = {
    entities: Array<ConflictBattleGeneratorEntity>;
    name: string;
};

export type ConflictBattleSaveAlliance = {
    alliance_id: number | null;
    corporations: Array<ConflictBattleSaveCorporation>;
};

export type ConflictBattleSaveCorporation = {
    corporation_id: number;
    isk_destroyed: number;
    isk_lost: number;
    kills: number;
    losses: number;
};

export type ConflictBattleSaveTeam = {
    alliances: Array<ConflictBattleSaveAlliance>;
    total_isk_destroyed: number;
    total_isk_lost: number;
    total_kills: number;
    total_losses: number;
};

export type ContestedSystem = {
    a_kills_b: number;
    b_kills_a: number;
    solar_system_id: number;
    system_name: string | null;
    total_isk: number;
    total_kills: number;
};

export type DismissedAnnouncementIdsResponse = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    dismissedIds: Array<number>;
};

export type DoctrineCluster = {
    avg_isk_per_loss: number;
    example_killmail: DoctrineExample;
    family_hash: string;
    first_loss?: string;
    isk_lost: number;
    last_loss?: string;
    losses: number;
    ship: DoctrineShip;
    signature?: string;
};

export type DoctrineDetectInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    entity: string | number;
    include_rookie_ships?: boolean;
    limit?: number;
    min_cluster_size?: number;
    since?: string;
    type?: 'character' | 'corporation' | 'alliance';
    until?: string;
};

export type DoctrineDetectOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    clusters: Array<DoctrineCluster> | null;
    count: number;
    entity: Entity;
    notes?: string;
    window: TimeWindow;
};

export type DoctrineExample = {
    killmail_id: number;
    modules?: Array<string> | null;
    url: string;
};

export type DoctrineShip = {
    group?: string;
    name: string | null;
    type_id: number;
};

export type DogmaDroneInput = {
    quantity?: number;
    type_id: number;
};

export type DogmaEvalInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    drones?: Array<DogmaDroneInput> | null;
    eft?: string;
    fit_hash?: string;
    killmail_id?: number;
    modules?: Array<DogmaModuleInput> | null;
    ship_type_id?: number;
    skills?: 'all_v' | 'none';
};

export type DogmaEvalOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    dps_note?: string;
    drone_count: number;
    fit_hash?: string;
    fitting: FittingDisplay;
    killmail_id?: number;
    module_count: number;
    ship: DogmaShip;
    skills: string;
    source: string;
    stats: HullDisplay;
};

export type DogmaFitInput = {
    drones?: Array<DogmaDroneInput> | null;
    eft?: string;
    fit_hash?: string;
    killmail_id?: number;
    modules?: Array<DogmaModuleInput> | null;
    ship_type_id?: number;
};

export type DogmaModuleInput = {
    charge_type_id?: number;
    index?: number;
    slot: 'high' | 'med' | 'low' | 'rig' | 'subsystem';
    type_id: number;
};

export type DogmaShip = {
    name: string | null;
    type_id: number;
};

export type DossierArchetypeLastSeen = {
    last_blops_seen: string | null;
    last_capital_kill: string | null;
    last_fc_seen: string | null;
    last_logi_seen: string | null;
    last_super_kill: string | null;
};

export type DossierInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    /**
     * Character name or id.
     */
    entity: string | number;
    format?: 'json' | 'summary';
    type?: 'character';
};

export type DossierOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    archetype_last_seen?: DossierArchetypeLastSeen;
    archetype_tags?: Array<string> | null;
    entity: Entity;
    lifetime?: EntityLifetime;
    playstyle_90d?: DossierPlaystyle;
    summary?: string;
    top_ships?: Array<DossierTopShip> | null;
    top_systems?: Array<DossierTopSystem> | null;
    top_wingmates?: Array<DossierWingmate> | null;
};

export type DossierPlaystyle = {
    avg_fleet_size: number;
    blob_pct: number;
    dominant: string;
    fleet_pct: number;
    mid_gang_pct: number;
    small_gang_pct: number;
    solo_pct: number;
    total_kills_90d: number;
};

export type DossierTopShip = {
    kills: number;
    name: string | null;
    type_id: number;
};

export type DossierTopSystem = {
    kills: number;
    name: string | null;
    system_id: number;
};

export type DossierWingmate = {
    character_id: number;
    name: string | null;
    shared_kills: number;
    url: string;
};

export type Entity = {
    id: number;
    name: string;
    ticker: string | null;
    type: string;
    url: string;
};

export type EntityBreakdown = {
    id: number;
    isk_destroyed: number;
    isk_lost: number;
    kills: number;
    losses: number;
    name: string | null;
};

export type EntityKillsInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    before?: number;
    entity: string | number;
    from?: string;
    limit?: number;
    role?: 'kills' | 'losses' | 'all';
    to?: string;
    type?: 'character' | 'corporation' | 'alliance' | 'ship' | 'system' | 'region' | 'constellation' | 'faction';
};

export type EntityKillsOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    count?: number;
    entity: Entity;
    kills: Array<KillmailSummary> | null;
    next_before: number | null;
};

export type EntityLifetime = {
    final_blows: number;
    first_seen_year: string | null;
    isk_destroyed: number;
    isk_efficiency: number;
    isk_lost: number;
    kills: number;
    last_seen_year: string | null;
    losses: number;
    npc_losses: number;
    points: number;
    solo_kills: number;
    solo_losses: number;
};

export type EntityOverviewInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    entity: string | number;
    format?: 'json' | 'summary';
    type?: 'character' | 'corporation' | 'alliance' | 'ship' | 'system' | 'constellation' | 'region';
};

export type EntityOverviewOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    entity: Entity;
    lifetime?: EntityLifetime;
    summary?: string;
    top_prey?: Array<EntityBreakdown> | null;
    top_regions?: Array<EntityBreakdown> | null;
    top_ships_flown?: Array<EntityBreakdown> | null;
    top_ships_lost?: Array<EntityBreakdown> | null;
    top_systems?: Array<EntityBreakdown> | null;
    top_tormentors?: Array<EntityBreakdown> | null;
};

export type EntityTimelineInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    bucket?: 'day' | 'month' | 'year';
    entity: string | number;
    since?: string;
    type?: 'character' | 'corporation' | 'alliance' | 'ship' | 'system' | 'constellation' | 'region';
    until?: string;
    /**
     * Opponent character, corporation, or alliance.
     */
    vs?: string | number;
};

export type EntityTimelineOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    bucket: string;
    buckets: Array<TimelineBucket> | null;
    count: number;
    entity: Entity;
    vs?: Entity;
    window: TimelineWindow;
};

export type EntityTopInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    dimension: 'ship_flown' | 'ship_lost' | 'system' | 'constellation' | 'region' | 'dies_to_corporation' | 'dies_to_alliance' | 'killed_corporation' | 'killed_alliance';
    entity: string | number;
    limit?: number;
    since?: string;
    sort_by?: 'kills' | 'losses' | 'isk_destroyed' | 'isk_lost';
    type?: 'character' | 'corporation' | 'alliance';
    until?: string;
    /**
     * Opponent character, corporation, or alliance.
     */
    vs?: string | number;
};

export type EntityTopOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    count: number;
    dimension: string;
    entity: Entity;
    rows: Array<EntityBreakdown> | null;
    sort_by: string;
    vs?: Entity;
    warnings?: Array<string> | null;
    window: 'lifetime' | {
        since: string;
        until: string;
    };
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

export type ExpensiveLoss = {
    killmail_id: number;
    system: LossSystem;
    time: string | null;
    total_value: number;
    url: string;
    victim: KillmailVictim;
    victim_ship: LossShip;
};

export type ExpensiveLossesInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    days?: number;
    limit?: number;
    min_value?: number;
    region_id?: number;
    ship_type_id?: number;
    system_id?: number;
    victim_alliance_id?: number;
    victim_character_id?: number;
    /**
     * Character, corporation, or alliance present as attacker.
     */
    vs?: string | number;
};

export type ExpensiveLossesOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    count: number;
    kills: Array<ExpensiveLoss> | null;
    window_days: number;
};

export type FindBattlesInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    limit?: number;
    min_isk?: number;
    min_kills?: number;
    opposing?: boolean;
    participants?: Array<string | number> | null;
    region_id?: number;
    since?: string;
    sort?: 'isk' | 'kills' | 'recent' | 'intensity';
    system_id?: number;
    until?: string;
};

export type FindBattlesOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    battles: Array<FoundBattle> | null;
    count: number;
    opposing_required?: boolean;
    participants_resolved?: Array<ResolvedParticipant> | null;
};

export type FitCompareInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    a: DogmaFitInput;
    b: DogmaFitInput;
    skills?: 'all_v' | 'none';
};

export type FitCompareOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    a: FitComparisonSide;
    b: FitComparisonSide;
    diff: FitDiff;
    skills: string;
};

export type FitComparisonSide = {
    dps_note?: string;
    ship: DogmaShip;
    stats: HullDisplay;
};

export type FitDiff = {
    align_time_s: NumericDiff;
    alpha: NumericDiff;
    cap: CapDiff;
    dps: NumericDiff;
    ehp: NumericDiff;
    ehp_armor: NumericDiff;
    ehp_hull: NumericDiff;
    ehp_shield: NumericDiff;
    max_velocity_ms: NumericDiff;
    signature_radius_m: NumericDiff;
};

export type FittingDisplay = {
    calibration: number | null;
    cpu: number | null;
    powergrid: number | null;
};

export type FittingItem = {
    charge?: FittingType;
    name: string | null;
    quantity: number;
    type_id: number;
};

export type FittingSlot = {
    items: Array<FittingItem> | null;
    slot: string;
};

export type FittingType = {
    name: string | null;
    type_id: number;
};

export type FittingVictim = {
    character_id: number | null;
    character_name: string | null;
};

export type FliesWithOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    count: number;
    entity: Entity;
    partners: Array<IntelPartner> | null;
    window_days: number;
};

export type ForensicsEvidence = {
    actual_ehp?: number;
    align_time_s?: number;
    attacker_count?: number;
    cap_stable?: boolean;
    cap_time_s?: number;
    family_hash?: string;
    hull_class?: string;
    top_family_count?: number;
    typical_ehp?: number;
    typical_threshold?: number;
};

export type ForensicsFinding = {
    code: string;
    evidence: ForensicsEvidence;
    message: string;
    severity: 'info' | 'warn' | 'critical';
};

export type ForensicsShip = {
    group: string | null;
    name: string | null;
    type_id: number;
};

export type ForensicsSystem = {
    id: number;
    name: string | null;
    security: number | null;
};

export type FoundBattle = {
    alliances_involved: number | null;
    battle_id: number;
    corporations_involved: number | null;
    duration_minutes: number;
    end_time: string | null;
    intensity_isk_per_minute: number | null;
    is_multi_party: boolean;
    kill_count: number;
    start_time: string | null;
    system: BattleSystemSummary;
    top_alliance_by_isk: BattleTopAlliance;
    total_isk_destroyed: number;
    url: string;
};

export type GlobalPulseAlliance = {
    alliance_id: number;
    kills: number;
    name: string | null;
    systems_active: number;
    ticker: string | null;
    url: string;
};

export type GlobalPulseCorporation = {
    alliance_id: number | null;
    alliance_name: string | null;
    corporation_id: number;
    kills: number;
    name: string | null;
    ticker: string | null;
    url: string;
};

export type GlobalPulseInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    /**
     * Lookback window in hours (max 24).
     */
    hours?: number;
    /**
     * How many rows in each list.
     */
    top_n?: number;
};

export type GlobalPulseOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    hottest_systems: Array<GlobalPulseSystem> | null;
    most_active_alliances: Array<GlobalPulseAlliance> | null;
    most_active_corporations: Array<GlobalPulseCorporation> | null;
    since: string;
    totals: GlobalPulseTotals;
    window_hours: number;
};

export type GlobalPulseSystem = {
    isk_destroyed: number;
    kills: number;
    latest_kill: string | null;
    name: string | null;
    region_name: string | null;
    security: number | null;
    solar_system_id: number;
    url: string;
};

export type GlobalPulseTotals = {
    isk_destroyed: number;
    kills: number;
    solo_kills: number;
    systems_active: number;
};

export type HeadToHead = {
    count: number;
    final_blows?: number;
    isk_destroyed?: number;
    last_seen?: string;
};

export type HullCapacitor = {
    capacity: number | null;
    recharge_s: number | null;
};

export type HullCargo = {
    special_bay: number | null;
    standard: number | null;
};

export type HullDifference = {
    armor_hp?: ValueDifference;
    cpu?: ValueDifference;
    hull_hp?: ValueDifference;
    max_target_range_m?: ValueDifference;
    max_velocity_ms?: ValueDifference;
    powergrid?: ValueDifference;
    shield_hp?: ValueDifference;
    signature_radius_m?: ValueDifference;
};

export type HullDisplay = {
    align_time_s: number | null;
    alpha: number | null;
    cap_capacity_gj: number | null;
    cap_peak_delta_gj_s: number | null;
    cap_stable: boolean;
    cap_time_s: number | null;
    dps_with_reload: number | null;
    dps_without_reload: number | null;
    ehp: number | null;
    ehp_armor: number | null;
    ehp_hull: number | null;
    ehp_shield: number | null;
    max_locked_targets: number | null;
    max_target_range_km: number | null;
    max_velocity_ms: number | null;
    scan_resolution_mm: number | null;
    signature_radius_m: number | null;
};

export type HullDrones = {
    bandwidth: number | null;
    bay: number | null;
};

export type HullFitting = {
    calibration: number | null;
    cpu: number | null;
    powergrid: number | null;
};

export type HullHp = {
    armor: number | null;
    hull: number | null;
    shield: number | null;
};

export type HullMobility = {
    agility: number | null;
    mass: number | null;
    max_velocity_ms: number | null;
    signature_radius_m: number | null;
};

export type HullResists = {
    armor: ResistLayer;
    hull: ResistLayer;
    shield: ResistLayer;
};

export type HullSensors = {
    gravimetric_strength: number | null;
    ladar_strength: number | null;
    magnetometric_strength: number | null;
    max_locked_targets: number | null;
    max_target_range_m: number | null;
    radar_strength: number | null;
    scan_resolution_mm: number | null;
};

export type HullSlots = {
    high: number | null;
    launcher_hardpoints: number | null;
    low: number | null;
    med: number | null;
    rig: number | null;
    subsystem: number | null;
    turret_hardpoints: number | null;
};

export type HullStats = {
    align_time: number | null;
    alpha: number | null;
    armor_ehp: number | null;
    calibration: number | null;
    cap_capacity: number | null;
    cap_depletes_in: number | null;
    cap_peak_delta: number | null;
    cpu_output: number | null;
    dps_with_reload: number | null;
    dps_without_reload: number | null;
    ehp: number | null;
    hull_ehp: number | null;
    mass: number | null;
    max_locked_targets: number | null;
    max_target_range: number | null;
    max_velocity: number | null;
    pg_output: number | null;
    scan_resolution: number | null;
    shield_ehp: number | null;
    signature_radius: number | null;
};

export type HuntsInOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    count: number;
    entity: Entity;
    systems: Array<IntelSystem> | null;
    window_days: number;
};

export type IdName = {
    id: number | null;
    name: string | null;
};

export type IdNameTicker = {
    id: number;
    name: string | null;
    ticker: string | null;
};

export type ImagesOverviewResponse = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    routes: Array<string> | null;
    service: string;
};

export type IntelCharacter = {
    alliance_id: number | null;
    alliance_name: string | null;
    character_id: number;
    character_name: string | null;
    corporation_id: number | null;
    corporation_name: string | null;
    count: number;
    final_blows: number;
    isk_destroyed: number;
    last_seen: string | null;
    url: string;
};

export type IntelPartner = {
    alliance_id: number | null;
    alliance_name: string | null;
    character_id: number;
    character_name: string | null;
    corporation_id: number | null;
    corporation_name: string | null;
    first_seen: string | null;
    last_seen: string | null;
    shared_kills: number;
    url: string;
};

export type IntelSystem = {
    kills: number;
    last_seen: string | null;
    region_id: number | null;
    region_name: string | null;
    security: number | null;
    system_id: number;
    system_name: string | null;
    url: string;
};

export type ItemFitting = {
    calibration: number | null;
    cpu: number | null;
    powergrid: number | null;
};

export type ItemInfoInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    item: string | number;
};

export type ItemInfoOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    category: IdName;
    current_jita_price: number | null;
    description: string | null;
    fitting: ItemFitting;
    group: IdName;
    market_note?: string;
    meta_group: IdName;
    name: string;
    physical: ItemPhysical;
    type_id: number;
    url: string;
    variants: Array<ItemVariant> | null;
    variation_parent_type_id: number | null;
};

export type ItemPhysical = {
    capacity: number | null;
    mass: number | null;
    volume: number | null;
};

export type ItemVariant = {
    meta_group: string | null;
    name: string;
    type_id: number;
};

export type KillmailAttacker = {
    alliance_id: number | null;
    alliance_name: string | null;
    alliance_ticker: string | null;
    character_id: number | null;
    character_name: string | null;
    corporation_id: number | null;
    corporation_name: string | null;
    corporation_ticker: string | null;
    damage_done: number;
    faction_id: number | null;
    faction_name: string | null;
    final_blow: boolean;
    security_status: number | null;
    ship_name: string | null;
    ship_type_id: number | null;
    weapon_name: string | null;
    weapon_type_id: number | null;
};

export type KillmailFittingInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    format?: 'json' | 'eft';
    killmail_id: number;
};

export type KillmailFittingOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    drones?: Array<FittingItem> | null;
    eft?: string;
    family_hash: string;
    first_seen_at?: string;
    fit_hash: string;
    kill_time?: string;
    killmail_id: number;
    ship: FittingType;
    slots?: Array<FittingSlot> | null;
    total_value?: number;
    url: string;
    victim?: FittingVictim;
};

export type KillmailForensicsOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    attacker_count: number;
    dogma_stats: HullStats;
    finding_count: number;
    findings: Array<ForensicsFinding> | null;
    fit_hash: string | null;
    kill_time: string | null;
    killmail_id: number;
    system: ForensicsSystem;
    total_value: number;
    url: string;
    victim_ship: ForensicsShip;
};

export type KillmailInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    killmail_id: number;
};

export type KillmailLocation = {
    constellation_id: number | null;
    constellation_name: string | null;
    id: number;
    name: string | null;
    region_id: number | null;
    region_name: string | null;
    security: number | null;
};

export type KillmailOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    attacker_count: number;
    attackers: Array<KillmailAttacker> | null;
    destroyed_value: number;
    dropped_value: number;
    final_blow: KillmailAttacker;
    fitted_value: number;
    hash: string;
    is_npc: boolean;
    is_solo: boolean;
    killmail_id: number;
    points: number;
    system: KillmailLocation;
    time: string;
    total_value: number;
    url: string;
    victim: KillmailVictim;
    war_id: number | null;
};

export type KillmailParticipant = {
    alliance_id: number | null;
    alliance_name: string | null;
    alliance_ticker: string | null;
    character_id: number | null;
    character_name: string | null;
    corporation_id: number | null;
    corporation_name: string | null;
    corporation_ticker: string | null;
    ship_name: string | null;
    ship_type_id: number | null;
};

export type KillmailStoryFacts = {
    attacker_count: number;
    final_blow: KillmailAttacker;
    is_npc: boolean;
    is_solo: boolean;
    system: KillmailLocation;
    time: string;
    total_value: number;
    victim: string | null;
    victim_affiliation: string | null;
    victim_ship: string | null;
};

export type KillmailStoryOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    facts: KillmailStoryFacts;
    killmail_id: number;
    story: string;
    url: string;
};

export type KillmailSummary = {
    attacker_count?: number;
    final_blow: KillmailParticipant;
    is_npc?: boolean;
    is_solo?: boolean;
    killmail_id: number;
    system?: KillmailSystem;
    time?: string;
    total_value?: number;
    url: string;
    victim?: KillmailParticipant;
};

export type KillmailSystem = {
    id: number;
    name: string | null;
    region_id: number | null;
    region_name: string | null;
    security: number | null;
};

export type KillmailVictim = {
    alliance_id: number | null;
    alliance_name: string | null;
    alliance_ticker: string | null;
    character_id: number | null;
    character_name: string | null;
    corporation_id: number | null;
    corporation_name: string | null;
    corporation_ticker: string | null;
    damage_taken: number;
    faction_id: number | null;
    faction_name: string | null;
    ship_group_id: number | null;
    ship_group_name: string | null;
    ship_name: string | null;
    ship_type_id: number | null;
};

export type KillsWithBreakdown = {
    isk_destroyed: number;
    kills: number;
    month?: string;
    name?: string;
    region_id?: number;
    system_id?: number;
    type_id?: number;
};

export type KillsWithFilters = {
    entity_ship: NamedShip;
    from: string | null;
    partner_ship: NamedShip;
    region: Entity;
    system: Entity;
    to: string | null;
    victim_entity: Entity;
    victim_ship: NamedShip;
};

export type KillsWithInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    entity: string | number;
    entity_ship?: string | number;
    from?: string;
    group_by?: 'none' | 'victim_ship' | 'system' | 'region' | 'month' | 'partner_ship' | 'entity_ship';
    limit?: number;
    partner: string | number;
    partner_ship?: string | number;
    region?: string | number;
    system?: string | number;
    to?: string;
    type?: 'character';
    victim_entity?: string | number;
    victim_ship?: string | number;
};

export type KillsWithOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    breakdown?: Array<KillsWithBreakdown> | null;
    entity: Entity;
    filters: KillsWithFilters;
    group_by?: string;
    partner: Entity;
    totals: KillsWithTotals;
};

export type KillsWithTotals = {
    isk_destroyed: number;
    kills: number;
};

export type LossShip = {
    name: string | null;
    type_id: number | null;
};

export type LossSystem = {
    id: number;
    name: string | null;
    region_id: number | null;
    region_name: string | null;
};

export type MeInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    /**
     * Your EVE character name as typed in-game.
     */
    me: string;
};

export type MeIntelInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    limit?: number;
    me: string;
};

export type MeKillsInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    before?: number;
    from?: string;
    limit?: number;
    me: string;
    role?: 'kills' | 'losses' | 'all';
    to?: string;
};

export type MeKillsWithInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    entity_ship?: string | number;
    from?: string;
    group_by?: 'none' | 'victim_ship' | 'system' | 'region' | 'month' | 'partner_ship' | 'entity_ship';
    limit?: number;
    me: string;
    partner: string | number;
    partner_ship?: string | number;
    region?: string | number;
    system?: string | number;
    to?: string;
    victim_entity?: string | number;
    victim_ship?: string | number;
};

export type MeShipsUsedInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    from?: string;
    group_by?: 'none' | 'ship' | 'victim_ship' | 'system' | 'region' | 'month';
    limit?: number;
    me: string;
    region?: string | number;
    role?: 'kills' | 'losses' | 'all';
    ship?: string | number;
    system?: string | number;
    to?: string;
};

export type MeTimelineInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    bucket?: 'day' | 'month' | 'year';
    me: string;
    since?: string;
    until?: string;
    vs?: string | number;
};

export type MembershipPeriod = {
    alliance: IdNameTicker;
    corporation: IdNameTicker;
    duration_days: number;
    first_seen: string;
    last_seen: string;
    observation_count: number;
};

export type MetaPulseInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    include_rookie_ships?: boolean;
    limit?: number;
    min_cluster_size?: number;
    region_id?: number;
    ship_category?: 'all' | 'frigate' | 'destroyer' | 'cruiser' | 'battlecruiser' | 'battleship' | 'capital' | 'supercap' | 'subcap';
    since?: string;
    until?: string;
};

export type MetaPulseOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    clusters: Array<DoctrineCluster> | null;
    count: number;
    region_id: number | null;
    ship_category: string;
    window: TimeWindow;
};

export type NamedShip = {
    name: string;
    type_id: number;
};

export type NumericDiff = {
    a: number | null;
    b: number | null;
    delta: number | null;
    delta_pct: number | null;
};

export type OrganizationActivity = {
    id: number;
    kills: number;
    name: string | null;
    ticker: string | null;
};

export type PilotActivity = {
    by_day_of_week: {
        [key: string]: number;
    };
    by_hour_utc: {
        [key: string]: number;
    };
    description: string;
    peak_hour_event_count: number;
    peak_hour_utc: number;
    semantics: string;
};

export type PilotEfficiencyInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    /**
     * Character name or id.
     */
    entity: string | number;
    /**
     * Restrict to events involving this ship type.
     */
    ship_type_id?: number;
    /**
     * ISO datetime lower bound. Default 90 days ago.
     */
    since?: string;
    /**
     * ISO datetime upper bound. Default now.
     */
    until?: string;
};

export type PilotEfficiencyOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    activity: PilotActivity;
    character: CharacterRef;
    ship_filter?: ShipFilter;
    totals: PilotEfficiencyTotals;
    window: TimeWindow;
};

export type PilotEfficiencyTotals = {
    avg_gang_on_kills: number;
    avg_gang_on_losses: number;
    final_blows: number;
    isk_destroyed: number;
    isk_efficiency_pct: number;
    isk_lost: number;
    isk_ratio: number | null;
    kill_loss_ratio: number | null;
    kills: number;
    losses: number;
    solo_kills: number;
    solo_rate_pct: number;
};

export type RequiredIdName = {
    id: number;
    name: string | null;
};

export type ResistLayer = {
    em: number | null;
    explosive: number | null;
    kinetic: number | null;
    thermal: number | null;
};

export type ResolvedParticipant = {
    id: number;
    name: string;
    type: string;
};

export type RouteDangerInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    avoid?: Array<string | number> | null;
    /**
     * Starting solar system name or id.
     */
    from: string | number;
    hours?: number;
    prefer?: 'shortest' | 'safest' | 'lowsec_ok';
    round_trip?: boolean;
    /**
     * Destination solar system name or id.
     */
    to: string | number;
};

export type RouteDangerOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    avg_danger: number;
    avoided_systems: number;
    crosses_lowsec: boolean;
    crosses_nullsec: boolean;
    from: SystemRef;
    hops: Array<RouteHop> | null;
    jumps: number;
    prefer: string;
    return_leg?: RouteLeg;
    to: SystemRef;
    total_kills_on_route: number;
    window_hours: number;
    worst_hop: WorstRouteHop;
};

export type RouteHop = {
    danger: number;
    kills_window: number;
    region_id: number | null;
    region_name: string | null;
    sec_band: string;
    security: number | null;
    step: number;
    system_id: number;
    system_name: string | null;
    url: string;
};

export type RouteLeg = {
    avg_danger: number;
    crosses_lowsec: boolean;
    crosses_nullsec: boolean;
    hops: Array<RouteHop> | null;
    jumps: number;
    total_kills_on_route: number;
};

export type SearchHit = {
    alliance_id?: number;
    corporation_id?: number;
    id: number;
    name: string;
    ticker: string | null;
    type: string;
    url: string;
};

export type SearchInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    limit?: number;
    /**
     * Name or ticker to search for.
     */
    query: string;
    type?: 'character' | 'corporation' | 'alliance' | 'ship' | 'item' | 'system' | 'region' | 'constellation' | 'faction';
};

export type SearchOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    count: number;
    hits: Array<SearchHit> | null;
    query: string;
};

export type SharedSystem = {
    a_kills_seen: number;
    b_kills_seen: number;
    region_name: string | null;
    system_id: number;
    system_name: string | null;
};

export type ShipCompareInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    a: string | number;
    b: string | number;
};

export type ShipCompareOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    a: ShipInfoOutput;
    b: ShipInfoOutput;
    diff: HullDifference;
};

export type ShipFilter = {
    type_id: number;
};

export type ShipInfoInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    ship: string | number;
};

export type ShipInfoOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    base_hp: HullHp;
    capacitor: HullCapacitor;
    cargo: HullCargo;
    category: string | null;
    current_jita_price: number | null;
    description: string | null;
    drones: HullDrones;
    fitting: HullFitting;
    group: string | null;
    market_note?: string;
    meta_group: string | null;
    mobility: HullMobility;
    name: string;
    race: string | null;
    resist_profile: HullResists;
    sensors: HullSensors;
    slots: HullSlots;
    type_id: number;
    url: string;
};

export type ShipLossActivity = {
    isk_lost: number;
    losses: number;
    name: string | null;
    type_id: number | null;
};

export type ShipsUsedBreakdown = {
    isk_destroyed: number;
    isk_lost: number;
    kills: number;
    losses: number;
    month?: string;
    name?: string;
    region_id?: number;
    system_id?: number;
    type_id?: number;
};

export type ShipsUsedFilters = {
    from: string | null;
    region: Entity;
    role: string;
    ship: NamedShip;
    system: Entity;
    to: string | null;
};

export type ShipsUsedInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    entity: string | number;
    from?: string;
    group_by?: 'none' | 'ship' | 'victim_ship' | 'system' | 'region' | 'month';
    limit?: number;
    region?: string | number;
    role?: 'kills' | 'losses' | 'all';
    ship?: string | number;
    system?: string | number;
    to?: string;
    type?: 'character' | 'corporation' | 'alliance';
};

export type ShipsUsedOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    breakdown?: Array<ShipsUsedBreakdown> | null;
    entity: Entity;
    filters: ShipsUsedFilters;
    group_by?: string;
    totals: ShipsUsedTotals;
};

export type ShipsUsedTotals = {
    isk_destroyed: number;
    isk_lost: number;
    kills: number;
    losses: number;
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

export type SystemFlags = {
    border: boolean;
    corridor: boolean;
    fringe: boolean;
    hub: boolean;
    international: boolean;
    regional: boolean;
};

export type SystemInfoInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    system: string | number;
};

export type SystemInfoOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    constellation: RequiredIdName;
    faction: RequiredIdName;
    flags: SystemFlags;
    is_pipe_tip: boolean;
    name: string;
    neighbor_count: number;
    neighbors: Array<SystemNeighbor> | null;
    region: RequiredIdName;
    security: number | null;
    security_band: string | null;
    security_class: string | null;
    solar_system_id: number;
    station_count: number;
    stations: Array<SystemStation> | null;
    url: string;
};

export type SystemNeighbor = {
    name: string;
    region_name: string | null;
    security: number | null;
    solar_system_id: number;
};

export type SystemPulseInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    hours?: number;
    /**
     * ISO datetime lower bound.
     */
    since?: string;
    /**
     * System name or id.
     */
    system: string | number;
    top_n?: number;
    /**
     * ISO datetime upper bound.
     */
    until?: string;
};

export type SystemPulseOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    heat_score: number;
    system: SystemRef;
    top_attacker_alliances: Array<OrganizationActivity> | null;
    top_attacker_corps: Array<OrganizationActivity> | null;
    top_victim_ships: Array<ShipLossActivity> | null;
    totals: SystemPulseTotals;
    window: TimeWindow;
    window_hours: number | null;
};

export type SystemPulseTotals = {
    attackers_total: number;
    isk_destroyed: number;
    kills: number;
    pvp_kills: number;
    solo_kills: number;
};

export type SystemRef = {
    id: number;
    name: string;
    url: string;
};

export type SystemStation = {
    name: string;
    station_id: number;
};

export type TimeWindow = {
    since: string;
    until: string;
};

export type TimelineBucket = {
    final_blows?: number;
    isk_destroyed: number;
    isk_lost: number;
    kills: number;
    losses: number;
    period_start: string;
    points?: number;
    solo_kills?: number;
    solo_losses?: number;
};

export type TimelineWindow = {
    since: string | null;
    until: string | null;
};

export type ValueDifference = {
    a: number;
    b: number;
    delta: number;
    delta_pct: number | null;
};

export type WarBattleSummary = {
    battle_id: number;
    duration_minutes: number;
    end_time: string | null;
    is_multi_party: boolean;
    kill_count: number;
    start_time: string | null;
    system: BattleSystemSummary;
    teams: Array<BattleTeamSummary> | null;
    total_isk_destroyed: number;
    url: string;
};

export type WarReportInput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    a: string | number;
    b: string | number;
    since?: string;
    top_battles?: number;
    top_systems?: number;
    until?: string;
};

export type WarReportOutput = {
    /**
     * A URL to the JSON Schema for this object.
     */
    readonly $schema?: string;
    a: Entity;
    b: Entity;
    recent_battles: Array<WarBattleSummary> | null;
    timeline_daily: Array<WarTimelineDay> | null;
    top_contested_systems: Array<ContestedSystem> | null;
    totals: WarTotals;
    window: TimeWindow;
};

export type WarTimelineDay = {
    a_isk_on_b: number;
    a_kills_b: number;
    b_isk_on_a: number;
    b_kills_a: number;
    period_start: string;
};

export type WarTotals = {
    a_isk_destroyed: number;
    a_isk_share: number;
    a_kills_b: number;
    b_isk_destroyed: number;
    b_kills_a: number;
    leader: string;
    total_isk: number;
    total_kills: number;
};

export type WorstRouteHop = {
    danger: number;
    kills_window: number;
    sec_band: string;
    system_id: number;
    system_name: string | null;
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

export type CharacterIntelBatchResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
    data: Array<{
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
            likelihood?: string;
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
    }>;
    days: number;
    not_found: Array<number>;
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
        likelihood?: string;
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
        ship_group_id: number | null;
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
        victim_faction_id: number | null;
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
        ship_group_id: number | null;
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
        victim_faction_id: number | null;
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

export type ReadyResponse = {
    /**
     * A URL to the JSON Schema for this response.
     */
    readonly $schema: string;
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
        ship_group_id: number | null;
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
        victim_faction_id: number | null;
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
        ship_group_id: number | null;
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
        victim_faction_id: number | null;
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

export type BattleReportInputWritable = {
    battle_id: number;
    format?: 'json' | 'summary';
    level?: 'alliance' | 'corp';
};

export type BattleReportOutputWritable = {
    battle_id: number;
    duration_minutes?: number;
    end_time?: string;
    is_custom?: boolean;
    is_multi_party?: boolean;
    kill_count?: number;
    level?: string;
    start_time?: string;
    summary?: string;
    system?: BattleSystemSummary;
    teams?: Array<BattleTeam> | null;
    total_isk_destroyed?: number;
    url: string;
};

export type CharacterHistoryInputWritable = {
    /**
     * Character name or id.
     */
    entity: string | number;
    /**
     * ISO datetime lower bound. Default all history.
     */
    since?: string;
};

export type CharacterHistoryOutputWritable = {
    character: CharacterRef;
    notes?: string;
    observation_count: number;
    period_count: number;
    periods: Array<MembershipPeriod> | null;
};

export type CharacterIntelInputWritable = {
    /**
     * Character name or id.
     */
    entity: string | number;
    limit?: number;
    /**
     * Must be character if specified.
     */
    type?: 'character';
};

export type CharacterIntelOutputWritable = {
    characters: Array<IntelCharacter> | null;
    count: number;
    entity: Entity;
    window_days: number;
};

export type CoalitionGraphInputWritable = {
    /**
     * Alliance name or id for an ego graph.
     */
    focus_alliance?: string | number;
    limit_edges?: number;
    min_alliance_battles?: number;
    min_edge_weight?: number;
    /**
     * ISO datetime lower bound. Default 30 days ago.
     */
    since?: string;
    /**
     * ISO datetime upper bound. Default now.
     */
    until?: string;
};

export type CoalitionGraphOutputWritable = {
    allied_edges: Array<CoalitionEdge> | null;
    edge_count: number;
    enemy_edges: Array<CoalitionEdge> | null;
    focus?: CoalitionFocus;
    mixed_edges: Array<CoalitionEdge> | null;
    node_count: number;
    nodes: Array<AllianceNode> | null;
    window: TimeWindow;
};

export type CompareInputWritable = {
    /**
     * First character name or id.
     */
    a: string | number;
    /**
     * Second character name or id.
     */
    b: string | number;
};

export type CompareOutputWritable = {
    a: Entity;
    a_killed_b: HeadToHead;
    b: Entity;
    b_killed_a: HeadToHead;
    shared_systems: Array<SharedSystem> | null;
    shared_wingmates: number;
    window_days: number;
};

export type DismissedAnnouncementIdsResponseWritable = {
    dismissedIds: Array<number>;
};

export type DoctrineDetectInputWritable = {
    entity: string | number;
    include_rookie_ships?: boolean;
    limit?: number;
    min_cluster_size?: number;
    since?: string;
    type?: 'character' | 'corporation' | 'alliance';
    until?: string;
};

export type DoctrineDetectOutputWritable = {
    clusters: Array<DoctrineCluster> | null;
    count: number;
    entity: Entity;
    notes?: string;
    window: TimeWindow;
};

export type DogmaEvalInputWritable = {
    drones?: Array<DogmaDroneInput> | null;
    eft?: string;
    fit_hash?: string;
    killmail_id?: number;
    modules?: Array<DogmaModuleInput> | null;
    ship_type_id?: number;
    skills?: 'all_v' | 'none';
};

export type DogmaEvalOutputWritable = {
    dps_note?: string;
    drone_count: number;
    fit_hash?: string;
    fitting: FittingDisplay;
    killmail_id?: number;
    module_count: number;
    ship: DogmaShip;
    skills: string;
    source: string;
    stats: HullDisplay;
};

export type DossierInputWritable = {
    /**
     * Character name or id.
     */
    entity: string | number;
    format?: 'json' | 'summary';
    type?: 'character';
};

export type DossierOutputWritable = {
    archetype_last_seen?: DossierArchetypeLastSeen;
    archetype_tags?: Array<string> | null;
    entity: Entity;
    lifetime?: EntityLifetime;
    playstyle_90d?: DossierPlaystyle;
    summary?: string;
    top_ships?: Array<DossierTopShip> | null;
    top_systems?: Array<DossierTopSystem> | null;
    top_wingmates?: Array<DossierWingmate> | null;
};

export type EntityKillsInputWritable = {
    before?: number;
    entity: string | number;
    from?: string;
    limit?: number;
    role?: 'kills' | 'losses' | 'all';
    to?: string;
    type?: 'character' | 'corporation' | 'alliance' | 'ship' | 'system' | 'region' | 'constellation' | 'faction';
};

export type EntityKillsOutputWritable = {
    count?: number;
    entity: Entity;
    kills: Array<KillmailSummary> | null;
    next_before: number | null;
};

export type EntityOverviewInputWritable = {
    entity: string | number;
    format?: 'json' | 'summary';
    type?: 'character' | 'corporation' | 'alliance' | 'ship' | 'system' | 'constellation' | 'region';
};

export type EntityOverviewOutputWritable = {
    entity: Entity;
    lifetime?: EntityLifetime;
    summary?: string;
    top_prey?: Array<EntityBreakdown> | null;
    top_regions?: Array<EntityBreakdown> | null;
    top_ships_flown?: Array<EntityBreakdown> | null;
    top_ships_lost?: Array<EntityBreakdown> | null;
    top_systems?: Array<EntityBreakdown> | null;
    top_tormentors?: Array<EntityBreakdown> | null;
};

export type EntityTimelineInputWritable = {
    bucket?: 'day' | 'month' | 'year';
    entity: string | number;
    since?: string;
    type?: 'character' | 'corporation' | 'alliance' | 'ship' | 'system' | 'constellation' | 'region';
    until?: string;
    /**
     * Opponent character, corporation, or alliance.
     */
    vs?: string | number;
};

export type EntityTimelineOutputWritable = {
    bucket: string;
    buckets: Array<TimelineBucket> | null;
    count: number;
    entity: Entity;
    vs?: Entity;
    window: TimelineWindow;
};

export type EntityTopInputWritable = {
    dimension: 'ship_flown' | 'ship_lost' | 'system' | 'constellation' | 'region' | 'dies_to_corporation' | 'dies_to_alliance' | 'killed_corporation' | 'killed_alliance';
    entity: string | number;
    limit?: number;
    since?: string;
    sort_by?: 'kills' | 'losses' | 'isk_destroyed' | 'isk_lost';
    type?: 'character' | 'corporation' | 'alliance';
    until?: string;
    /**
     * Opponent character, corporation, or alliance.
     */
    vs?: string | number;
};

export type EntityTopOutputWritable = {
    count: number;
    dimension: string;
    entity: Entity;
    rows: Array<EntityBreakdown> | null;
    sort_by: string;
    vs?: Entity;
    warnings?: Array<string> | null;
    window: 'lifetime' | {
        since: string;
        until: string;
    };
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

export type ExpensiveLossesInputWritable = {
    days?: number;
    limit?: number;
    min_value?: number;
    region_id?: number;
    ship_type_id?: number;
    system_id?: number;
    victim_alliance_id?: number;
    victim_character_id?: number;
    /**
     * Character, corporation, or alliance present as attacker.
     */
    vs?: string | number;
};

export type ExpensiveLossesOutputWritable = {
    count: number;
    kills: Array<ExpensiveLoss> | null;
    window_days: number;
};

export type FindBattlesInputWritable = {
    limit?: number;
    min_isk?: number;
    min_kills?: number;
    opposing?: boolean;
    participants?: Array<string | number> | null;
    region_id?: number;
    since?: string;
    sort?: 'isk' | 'kills' | 'recent' | 'intensity';
    system_id?: number;
    until?: string;
};

export type FindBattlesOutputWritable = {
    battles: Array<FoundBattle> | null;
    count: number;
    opposing_required?: boolean;
    participants_resolved?: Array<ResolvedParticipant> | null;
};

export type FitCompareInputWritable = {
    a: DogmaFitInput;
    b: DogmaFitInput;
    skills?: 'all_v' | 'none';
};

export type FitCompareOutputWritable = {
    a: FitComparisonSide;
    b: FitComparisonSide;
    diff: FitDiff;
    skills: string;
};

export type FliesWithOutputWritable = {
    count: number;
    entity: Entity;
    partners: Array<IntelPartner> | null;
    window_days: number;
};

export type GlobalPulseInputWritable = {
    /**
     * Lookback window in hours (max 24).
     */
    hours?: number;
    /**
     * How many rows in each list.
     */
    top_n?: number;
};

export type GlobalPulseOutputWritable = {
    hottest_systems: Array<GlobalPulseSystem> | null;
    most_active_alliances: Array<GlobalPulseAlliance> | null;
    most_active_corporations: Array<GlobalPulseCorporation> | null;
    since: string;
    totals: GlobalPulseTotals;
    window_hours: number;
};

export type HuntsInOutputWritable = {
    count: number;
    entity: Entity;
    systems: Array<IntelSystem> | null;
    window_days: number;
};

export type ImagesOverviewResponseWritable = {
    routes: Array<string> | null;
    service: string;
};

export type ItemInfoInputWritable = {
    item: string | number;
};

export type ItemInfoOutputWritable = {
    category: IdName;
    current_jita_price: number | null;
    description: string | null;
    fitting: ItemFitting;
    group: IdName;
    market_note?: string;
    meta_group: IdName;
    name: string;
    physical: ItemPhysical;
    type_id: number;
    url: string;
    variants: Array<ItemVariant> | null;
    variation_parent_type_id: number | null;
};

export type KillmailFittingInputWritable = {
    format?: 'json' | 'eft';
    killmail_id: number;
};

export type KillmailFittingOutputWritable = {
    drones?: Array<FittingItem> | null;
    eft?: string;
    family_hash: string;
    first_seen_at?: string;
    fit_hash: string;
    kill_time?: string;
    killmail_id: number;
    ship: FittingType;
    slots?: Array<FittingSlot> | null;
    total_value?: number;
    url: string;
    victim?: FittingVictim;
};

export type KillmailForensicsOutputWritable = {
    attacker_count: number;
    dogma_stats: HullStats;
    finding_count: number;
    findings: Array<ForensicsFinding> | null;
    fit_hash: string | null;
    kill_time: string | null;
    killmail_id: number;
    system: ForensicsSystem;
    total_value: number;
    url: string;
    victim_ship: ForensicsShip;
};

export type KillmailInputWritable = {
    killmail_id: number;
};

export type KillmailOutputWritable = {
    attacker_count: number;
    attackers: Array<KillmailAttacker> | null;
    destroyed_value: number;
    dropped_value: number;
    final_blow: KillmailAttacker;
    fitted_value: number;
    hash: string;
    is_npc: boolean;
    is_solo: boolean;
    killmail_id: number;
    points: number;
    system: KillmailLocation;
    time: string;
    total_value: number;
    url: string;
    victim: KillmailVictim;
    war_id: number | null;
};

export type KillmailStoryOutputWritable = {
    facts: KillmailStoryFacts;
    killmail_id: number;
    story: string;
    url: string;
};

export type KillsWithInputWritable = {
    entity: string | number;
    entity_ship?: string | number;
    from?: string;
    group_by?: 'none' | 'victim_ship' | 'system' | 'region' | 'month' | 'partner_ship' | 'entity_ship';
    limit?: number;
    partner: string | number;
    partner_ship?: string | number;
    region?: string | number;
    system?: string | number;
    to?: string;
    type?: 'character';
    victim_entity?: string | number;
    victim_ship?: string | number;
};

export type KillsWithOutputWritable = {
    breakdown?: Array<KillsWithBreakdown> | null;
    entity: Entity;
    filters: KillsWithFilters;
    group_by?: string;
    partner: Entity;
    totals: KillsWithTotals;
};

export type MeInputWritable = {
    /**
     * Your EVE character name as typed in-game.
     */
    me: string;
};

export type MeIntelInputWritable = {
    limit?: number;
    me: string;
};

export type MeKillsInputWritable = {
    before?: number;
    from?: string;
    limit?: number;
    me: string;
    role?: 'kills' | 'losses' | 'all';
    to?: string;
};

export type MeKillsWithInputWritable = {
    entity_ship?: string | number;
    from?: string;
    group_by?: 'none' | 'victim_ship' | 'system' | 'region' | 'month' | 'partner_ship' | 'entity_ship';
    limit?: number;
    me: string;
    partner: string | number;
    partner_ship?: string | number;
    region?: string | number;
    system?: string | number;
    to?: string;
    victim_entity?: string | number;
    victim_ship?: string | number;
};

export type MeShipsUsedInputWritable = {
    from?: string;
    group_by?: 'none' | 'ship' | 'victim_ship' | 'system' | 'region' | 'month';
    limit?: number;
    me: string;
    region?: string | number;
    role?: 'kills' | 'losses' | 'all';
    ship?: string | number;
    system?: string | number;
    to?: string;
};

export type MeTimelineInputWritable = {
    bucket?: 'day' | 'month' | 'year';
    me: string;
    since?: string;
    until?: string;
    vs?: string | number;
};

export type MetaPulseInputWritable = {
    include_rookie_ships?: boolean;
    limit?: number;
    min_cluster_size?: number;
    region_id?: number;
    ship_category?: 'all' | 'frigate' | 'destroyer' | 'cruiser' | 'battlecruiser' | 'battleship' | 'capital' | 'supercap' | 'subcap';
    since?: string;
    until?: string;
};

export type MetaPulseOutputWritable = {
    clusters: Array<DoctrineCluster> | null;
    count: number;
    region_id: number | null;
    ship_category: string;
    window: TimeWindow;
};

export type PilotEfficiencyInputWritable = {
    /**
     * Character name or id.
     */
    entity: string | number;
    /**
     * Restrict to events involving this ship type.
     */
    ship_type_id?: number;
    /**
     * ISO datetime lower bound. Default 90 days ago.
     */
    since?: string;
    /**
     * ISO datetime upper bound. Default now.
     */
    until?: string;
};

export type PilotEfficiencyOutputWritable = {
    activity: PilotActivity;
    character: CharacterRef;
    ship_filter?: ShipFilter;
    totals: PilotEfficiencyTotals;
    window: TimeWindow;
};

export type RouteDangerInputWritable = {
    avoid?: Array<string | number> | null;
    /**
     * Starting solar system name or id.
     */
    from: string | number;
    hours?: number;
    prefer?: 'shortest' | 'safest' | 'lowsec_ok';
    round_trip?: boolean;
    /**
     * Destination solar system name or id.
     */
    to: string | number;
};

export type RouteDangerOutputWritable = {
    avg_danger: number;
    avoided_systems: number;
    crosses_lowsec: boolean;
    crosses_nullsec: boolean;
    from: SystemRef;
    hops: Array<RouteHop> | null;
    jumps: number;
    prefer: string;
    return_leg?: RouteLeg;
    to: SystemRef;
    total_kills_on_route: number;
    window_hours: number;
    worst_hop: WorstRouteHop;
};

export type SearchInputWritable = {
    limit?: number;
    /**
     * Name or ticker to search for.
     */
    query: string;
    type?: 'character' | 'corporation' | 'alliance' | 'ship' | 'item' | 'system' | 'region' | 'constellation' | 'faction';
};

export type SearchOutputWritable = {
    count: number;
    hits: Array<SearchHit> | null;
    query: string;
};

export type ShipCompareInputWritable = {
    a: string | number;
    b: string | number;
};

export type ShipCompareOutputWritable = {
    a: ShipInfoOutputWritable;
    b: ShipInfoOutputWritable;
    diff: HullDifference;
};

export type ShipInfoInputWritable = {
    ship: string | number;
};

export type ShipInfoOutputWritable = {
    base_hp: HullHp;
    capacitor: HullCapacitor;
    cargo: HullCargo;
    category: string | null;
    current_jita_price: number | null;
    description: string | null;
    drones: HullDrones;
    fitting: HullFitting;
    group: string | null;
    market_note?: string;
    meta_group: string | null;
    mobility: HullMobility;
    name: string;
    race: string | null;
    resist_profile: HullResists;
    sensors: HullSensors;
    slots: HullSlots;
    type_id: number;
    url: string;
};

export type ShipsUsedInputWritable = {
    entity: string | number;
    from?: string;
    group_by?: 'none' | 'ship' | 'victim_ship' | 'system' | 'region' | 'month';
    limit?: number;
    region?: string | number;
    role?: 'kills' | 'losses' | 'all';
    ship?: string | number;
    system?: string | number;
    to?: string;
    type?: 'character' | 'corporation' | 'alliance';
};

export type ShipsUsedOutputWritable = {
    breakdown?: Array<ShipsUsedBreakdown> | null;
    entity: Entity;
    filters: ShipsUsedFilters;
    group_by?: string;
    totals: ShipsUsedTotals;
};

export type SiteConfigurationResponseWritable = {
    domain: SiteDomainConfiguration | null;
    isDomainHost: boolean;
};

export type SystemInfoInputWritable = {
    system: string | number;
};

export type SystemInfoOutputWritable = {
    constellation: RequiredIdName;
    faction: RequiredIdName;
    flags: SystemFlags;
    is_pipe_tip: boolean;
    name: string;
    neighbor_count: number;
    neighbors: Array<SystemNeighbor> | null;
    region: RequiredIdName;
    security: number | null;
    security_band: string | null;
    security_class: string | null;
    solar_system_id: number;
    station_count: number;
    stations: Array<SystemStation> | null;
    url: string;
};

export type SystemPulseInputWritable = {
    hours?: number;
    /**
     * ISO datetime lower bound.
     */
    since?: string;
    /**
     * System name or id.
     */
    system: string | number;
    top_n?: number;
    /**
     * ISO datetime upper bound.
     */
    until?: string;
};

export type SystemPulseOutputWritable = {
    heat_score: number;
    system: SystemRef;
    top_attacker_alliances: Array<OrganizationActivity> | null;
    top_attacker_corps: Array<OrganizationActivity> | null;
    top_victim_ships: Array<ShipLossActivity> | null;
    totals: SystemPulseTotals;
    window: TimeWindow;
    window_hours: number | null;
};

export type WarReportInputWritable = {
    a: string | number;
    b: string | number;
    since?: string;
    top_battles?: number;
    top_systems?: number;
    until?: string;
};

export type WarReportOutputWritable = {
    a: Entity;
    b: Entity;
    recent_battles: Array<WarBattleSummary> | null;
    timeline_daily: Array<WarTimelineDay> | null;
    top_contested_systems: Array<ContestedSystem> | null;
    totals: WarTotals;
    window: TimeWindow;
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

export type CharacterIntelBatchResponseWritable = {
    data: Array<{
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
            likelihood?: string;
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
    }>;
    days: number;
    not_found: Array<number>;
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
        likelihood?: string;
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
        ship_group_id: number | null;
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
        victim_faction_id: number | null;
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
        ship_group_id: number | null;
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
        victim_faction_id: number | null;
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
        ship_group_id: number | null;
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
        victim_faction_id: number | null;
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
        ship_group_id: number | null;
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
        victim_faction_id: number | null;
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
    200: Array<{
        changefreq: string;
        /**
         * UTC timestamp with millisecond precision.
         */
        lastmod?: string;
        loc: string;
        priority: number;
    }>;
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
    200: Array<{
        changefreq: string;
        /**
         * UTC timestamp with millisecond precision.
         */
        lastmod?: string;
        loc: string;
        priority: number;
    }>;
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
    200: Array<{
        changefreq: string;
        /**
         * UTC timestamp with millisecond precision.
         */
        lastmod?: string;
        loc: string;
        priority: number;
    }>;
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
    200: Array<{
        changefreq: string;
        /**
         * UTC timestamp with millisecond precision.
         */
        lastmod?: string;
        loc: string;
        priority: number;
    }>;
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
    200: Array<{
        changefreq: string;
        /**
         * UTC timestamp with millisecond precision.
         */
        lastmod?: string;
        loc: string;
        priority: number;
    }>;
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
    200: Array<{
        changefreq: string;
        /**
         * UTC timestamp with millisecond precision.
         */
        lastmod?: string;
        loc: string;
        priority: number;
    }>;
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
    200: Array<{
        changefreq: string;
        /**
         * UTC timestamp with millisecond precision.
         */
        lastmod?: string;
        loc: string;
        priority: number;
    }>;
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
    200: Array<{
        changefreq: string;
        /**
         * UTC timestamp with millisecond precision.
         */
        lastmod?: string;
        loc: string;
        priority: number;
    }>;
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
    200: Array<{
        changefreq: string;
        /**
         * UTC timestamp with millisecond precision.
         */
        lastmod?: string;
        loc: string;
        priority: number;
    }>;
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
    200: Array<{
        changefreq: string;
        /**
         * UTC timestamp with millisecond precision.
         */
        lastmod?: string;
        loc: string;
        priority: number;
    }>;
};

export type SitemapWarsCompatResponse = SitemapWarsCompatResponses[keyof SitemapWarsCompatResponses];

export type AnnouncementAdminListData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Restrict to one lifecycle state.
         */
        status?: 'active' | 'scheduled' | 'expired' | 'archived';
        /**
         * Restrict to one tier.
         */
        tier?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/admin/announcements';
};

export type AnnouncementAdminListResponses = {
    /**
     * OK
     */
    200: {
        announcements: Array<{
            /**
             * UTC timestamp with millisecond precision.
             */
            archived_at?: string | null;
            body_html?: string;
            body_md?: string;
            color?: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            created_by?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            expires_at?: string;
            icon?: string | null;
            id?: number;
            link_label?: string | null;
            link_url?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            starts_at?: string;
            tier?: number;
            title?: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            [key: string]: unknown;
        }>;
    };
};

export type AnnouncementAdminListResponse = AnnouncementAdminListResponses[keyof AnnouncementAdminListResponses];

export type AnnouncementAdminCreateData = {
    body: {
        body_md?: string;
        color?: 'info' | 'warning' | 'danger' | 'success';
        expires_at: string;
        icon?: string | null;
        link_label?: string | null;
        link_url?: string | null;
        starts_at?: string;
        tier: 1 | 2 | 3;
        title: string;
    };
    path?: never;
    query?: never;
    url: '/admin/announcements';
};

export type AnnouncementAdminCreateResponses = {
    /**
     * OK
     */
    200: {
        announcement: {
            /**
             * UTC timestamp with millisecond precision.
             */
            archived_at?: string | null;
            body_html?: string;
            body_md?: string;
            color?: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            created_by?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            expires_at?: string;
            icon?: string | null;
            id?: number;
            link_label?: string | null;
            link_url?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            starts_at?: string;
            tier?: number;
            title?: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            [key: string]: unknown;
        };
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
        announcement: {
            /**
             * UTC timestamp with millisecond precision.
             */
            archived_at?: string | null;
            body_html?: string;
            body_md?: string;
            color?: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            created_by?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            expires_at?: string;
            icon?: string | null;
            id?: number;
            link_label?: string | null;
            link_url?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            starts_at?: string;
            tier?: number;
            title?: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            [key: string]: unknown;
        };
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
        announcement: {
            /**
             * UTC timestamp with millisecond precision.
             */
            archived_at?: string | null;
            body_html?: string;
            body_md?: string;
            color?: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            created_by?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            expires_at?: string;
            icon?: string | null;
            id?: number;
            link_label?: string | null;
            link_url?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            starts_at?: string;
            tier?: number;
            title?: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            [key: string]: unknown;
        };
    };
};

export type AnnouncementAdminDetailResponse = AnnouncementAdminDetailResponses[keyof AnnouncementAdminDetailResponses];

export type AnnouncementAdminUpdateData = {
    body: {
        body_md?: string;
        color?: 'info' | 'warning' | 'danger' | 'success';
        expires_at?: string;
        icon?: string | null;
        link_label?: string | null;
        link_url?: string | null;
        starts_at?: string;
        tier?: 1 | 2 | 3;
        title?: string;
    };
    path?: never;
    query?: never;
    url: '/admin/announcements/{id}';
};

export type AnnouncementAdminUpdateResponses = {
    /**
     * OK
     */
    200: {
        announcement: {
            /**
             * UTC timestamp with millisecond precision.
             */
            archived_at?: string | null;
            body_html?: string;
            body_md?: string;
            color?: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            created_by?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            expires_at?: string;
            icon?: string | null;
            id?: number;
            link_label?: string | null;
            link_url?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            starts_at?: string;
            tier?: number;
            title?: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            [key: string]: unknown;
        };
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
        announcement: {
            /**
             * UTC timestamp with millisecond precision.
             */
            archived_at?: string | null;
            body_html?: string;
            body_md?: string;
            color?: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            created_by?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            expires_at?: string;
            icon?: string | null;
            id?: number;
            link_label?: string | null;
            link_url?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            starts_at?: string;
            tier?: number;
            title?: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            [key: string]: unknown;
        };
    };
};

export type AnnouncementAdminArchiveCompatResponse = AnnouncementAdminArchiveCompatResponses[keyof AnnouncementAdminArchiveCompatResponses];

export type BlogAdminListData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Restrict to one lifecycle state.
         */
        status?: 'draft' | 'published' | 'archived';
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/admin/blog';
};

export type BlogAdminListResponses = {
    /**
     * OK
     */
    200: {
        posts: Array<{
            author_alliance_id: number | null;
            author_alliance_name: string | null;
            author_corporation_id: number | null;
            author_corporation_name: string | null;
            author_id: number;
            author_name: string;
            body_html: string;
            body_md: string;
            cover_image_url: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            excerpt: string | null;
            id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            published_at: string | null;
            slug: string;
            status: number;
            tags: Array<string>;
            title: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
        }>;
    };
};

export type BlogAdminListResponse = BlogAdminListResponses[keyof BlogAdminListResponses];

export type BlogAdminCreateData = {
    body: {
        body_md?: string;
        cover_image_url?: string | null;
        excerpt?: string | null;
        published_at?: string | null;
        slug?: string;
        status?: 0 | 1 | 2;
        tags?: Array<string>;
        title: string;
    };
    path?: never;
    query?: never;
    url: '/admin/blog';
};

export type BlogAdminCreateResponses = {
    /**
     * OK
     */
    200: {
        post: {
            author_alliance_id: number | null;
            author_alliance_name: string | null;
            author_corporation_id: number | null;
            author_corporation_name: string | null;
            author_id: number;
            author_name: string;
            body_html: string;
            body_md: string;
            cover_image_url: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            excerpt: string | null;
            id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            published_at: string | null;
            slug: string;
            status: number;
            tags: Array<string>;
            title: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
        };
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
        post: {
            author_alliance_id: number | null;
            author_alliance_name: string | null;
            author_corporation_id: number | null;
            author_corporation_name: string | null;
            author_id: number;
            author_name: string;
            body_html: string;
            body_md: string;
            cover_image_url: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            excerpt: string | null;
            id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            published_at: string | null;
            slug: string;
            status: number;
            tags: Array<string>;
            title: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
        };
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
        id: number;
        ok: boolean;
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
        post: {
            author_alliance_id: number | null;
            author_alliance_name: string | null;
            author_corporation_id: number | null;
            author_corporation_name: string | null;
            author_id: number;
            author_name: string;
            body_html: string;
            body_md: string;
            cover_image_url: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            excerpt: string | null;
            id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            published_at: string | null;
            slug: string;
            status: number;
            tags: Array<string>;
            title: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
        };
    };
};

export type BlogAdminDetailResponse = BlogAdminDetailResponses[keyof BlogAdminDetailResponses];

export type BlogAdminUpdateData = {
    body: {
        body_md?: string;
        cover_image_url?: string | null;
        excerpt?: string | null;
        published_at?: string | null;
        slug?: string;
        status?: 0 | 1 | 2;
        tags?: Array<string>;
        title?: string;
    };
    path?: never;
    query?: never;
    url: '/admin/blog/{id}';
};

export type BlogAdminUpdateResponses = {
    /**
     * OK
     */
    200: {
        post: {
            author_alliance_id: number | null;
            author_alliance_name: string | null;
            author_corporation_id: number | null;
            author_corporation_name: string | null;
            author_id: number;
            author_name: string;
            body_html: string;
            body_md: string;
            cover_image_url: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            excerpt: string | null;
            id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            published_at: string | null;
            slug: string;
            status: number;
            tags: Array<string>;
            title: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
        };
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
        paid: boolean;
    };
};

export type CampaignPrizePaidLegacyResponse = CampaignPrizePaidLegacyResponses[keyof CampaignPrizePaidLegacyResponses];

export type CampaignAdminListData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Restrict to one processing state.
         */
        state?: 'pending' | 'active' | 'archived' | 'paused' | 'failed';
        /**
         * Campaign name, ID, or creator search.
         */
        q?: string;
        /**
         * Page number, counted from 1.
         */
        page?: number;
    };
    url: '/admin/campaigns';
};

export type CampaignAdminListResponses = {
    /**
     * OK
     */
    200: {
        campaigns: Array<{
            campaign_id: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            created_by_character_id: number;
            creator_name: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            end_time: string | null;
            estimated_killmails: number;
            last_processing_duration_ms: number | null;
            last_processing_error: string | null;
            last_processing_killmails: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_processing_started_at: string | null;
            name: string;
            processing_note: string | null;
            processing_paused: boolean;
            /**
             * UTC timestamp with millisecond precision.
             */
            start_time: string;
            status: number;
            totals: {
                alliancesInvolved: number;
                charactersInvolved: number;
                corporationsInvolved: number;
                iskDestroyed: number;
                killCount: number;
            } | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
            visibility: number;
        }>;
        hasMore: boolean;
        page: number;
    };
};

export type CampaignAdminListResponse = CampaignAdminListResponses[keyof CampaignAdminListResponses];

export type CampaignAdminActionLegacyData = {
    body: {
        action: 'pause' | 'resume' | 'reprocess' | 'archive' | 'delete';
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
        action: string;
        dispatched: boolean;
        ok: boolean;
    };
};

export type CampaignAdminActionLegacyResponse = CampaignAdminActionLegacyResponses[keyof CampaignAdminActionLegacyResponses];

export type CampaignAdminActionData = {
    body: {
        action: 'pause' | 'resume' | 'reprocess' | 'archive' | 'delete';
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
        action: string;
        dispatched: boolean;
        ok: boolean;
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
        paid: boolean;
    };
};

export type CampaignPrizePaidResponse = CampaignPrizePaidResponses[keyof CampaignPrizePaidResponses];

export type AdminCommentReportResolutionData = {
    body: {
        resolution: 'dismissed' | 'deleted' | 'warned';
    };
    path?: never;
    query?: never;
    url: '/admin/comment-reports/{id}';
};

export type AdminCommentReportResolutionResponses = {
    /**
     * OK
     */
    200: {
        ok: boolean;
        report: {
            comment_id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            id: number;
            message: string | null;
            reason: string;
            reporter_id: number;
            reporter_name: string;
            resolution: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            resolved_at: string | null;
            resolved_by: number | null;
        };
    };
};

export type AdminCommentReportResolutionResponse = AdminCommentReportResolutionResponses[keyof AdminCommentReportResolutionResponses];

export type AdminCommentsData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Which comments to queue. Unknown values behave like `all`.
         */
        filter?: 'flagged' | 'reported' | 'all';
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/admin/comments';
};

export type AdminCommentsResponses = {
    /**
     * OK
     */
    200: {
        comments: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            body_html: string;
            body_md: string;
            character_id: number;
            character_name: string;
            corporation_id: number;
            corporation_name: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            deleted_at: string | null;
            deleted_by: number | null;
            depth: number;
            domain_id: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            edited_at: string | null;
            flagged: boolean;
            id: number;
            moderation_status: number;
            parent_id: number | null;
            reply_count?: number;
            reports_count: number;
            root_id: number | null;
            target_id: number;
            target_slug: string | null;
            target_type: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
            visibility: number;
        }>;
    };
};

export type AdminCommentsResponse = AdminCommentsResponses[keyof AdminCommentsResponses];

export type AdminCommentsLiveQueueAliasData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Which comments to queue. Unknown values behave like `all`.
         */
        filter?: 'flagged' | 'reported' | 'all';
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/admin/comments/queue';
};

export type AdminCommentsLiveQueueAliasResponses = {
    /**
     * OK
     */
    200: {
        comments: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            body_html: string;
            body_md: string;
            character_id: number;
            character_name: string;
            corporation_id: number;
            corporation_name: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            deleted_at: string | null;
            deleted_by: number | null;
            depth: number;
            domain_id: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            edited_at: string | null;
            flagged: boolean;
            id: number;
            moderation_status: number;
            parent_id: number | null;
            reply_count?: number;
            reports_count: number;
            root_id: number | null;
            target_id: number;
            target_slug: string | null;
            target_type: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
            visibility: number;
        }>;
    };
};

export type AdminCommentsLiveQueueAliasResponse = AdminCommentsLiveQueueAliasResponses[keyof AdminCommentsLiveQueueAliasResponses];

export type AdminCommentReportResolutionLiveAliasData = {
    body: {
        resolution: 'dismissed' | 'deleted' | 'warned';
    };
    path?: never;
    query?: never;
    url: '/admin/comments/reports/{id}/resolve';
};

export type AdminCommentReportResolutionLiveAliasResponses = {
    /**
     * OK
     */
    200: {
        ok: boolean;
        report: {
            comment_id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            id: number;
            message: string | null;
            reason: string;
            reporter_id: number;
            reporter_name: string;
            resolution: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            resolved_at: string | null;
            resolved_by: number | null;
        };
    };
};

export type AdminCommentReportResolutionLiveAliasResponse = AdminCommentReportResolutionLiveAliasResponses[keyof AdminCommentReportResolutionLiveAliasResponses];

export type AdminCommentModerationData = {
    body: {
        action: 'hide' | 'restore' | 'hidden' | 'published';
    };
    path?: never;
    query?: never;
    url: '/admin/comments/{id}';
};

export type AdminCommentModerationResponses = {
    /**
     * OK
     */
    200: {
        comment: {
            alliance_id: number | null;
            alliance_name: string | null;
            body_html: string;
            body_md: string;
            character_id: number;
            character_name: string;
            corporation_id: number;
            corporation_name: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            deleted_at: string | null;
            deleted_by: number | null;
            depth: number;
            domain_id: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            edited_at: string | null;
            flagged: boolean;
            id: number;
            moderation_status: number;
            parent_id: number | null;
            reply_count?: number;
            reports_count: number;
            root_id: number | null;
            target_id: number;
            target_slug: string | null;
            target_type: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
            visibility: number;
        };
        ok: boolean;
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
        comment: {
            alliance_id: number | null;
            alliance_name: string | null;
            body_html: string;
            body_md: string;
            character_id: number;
            character_name: string;
            corporation_id: number;
            corporation_name: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            deleted_at: string | null;
            deleted_by: number | null;
            depth: number;
            domain_id: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            edited_at: string | null;
            flagged: boolean;
            id: number;
            moderation_status: number;
            parent_id: number | null;
            reply_count?: number;
            reports_count: number;
            root_id: number | null;
            target_id: number;
            target_slug: string | null;
            target_type: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
            visibility: number;
        };
        ok: boolean;
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
        comment: {
            alliance_id: number | null;
            alliance_name: string | null;
            body_html: string;
            body_md: string;
            character_id: number;
            character_name: string;
            corporation_id: number;
            corporation_name: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            deleted_at: string | null;
            deleted_by: number | null;
            depth: number;
            domain_id: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            edited_at: string | null;
            flagged: boolean;
            id: number;
            moderation_status: number;
            parent_id: number | null;
            reply_count?: number;
            reports_count: number;
            root_id: number | null;
            target_id: number;
            target_slug: string | null;
            target_type: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
            visibility: number;
        };
        ok: boolean;
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
        domains: Array<{
            active?: boolean;
            backgrounds?: Array<{
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            }>;
            bannerAsset?: {
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            } | null;
            campaign_policy?: number;
            campaigns?: Array<{
                campaign_id?: string;
                created_by_character_id?: number;
                description?: string | null;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                end_time?: string;
                estimated_killmails?: number | null;
                name?: string;
                public_on_domain?: boolean;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                start_time?: string;
                status?: number;
                visibility?: number;
                [key: string]: unknown;
            }>;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            custom_hostname?: string | null;
            entities?: Array<{
                id: number;
                name?: string;
                type: string;
            }>;
            id?: number;
            logoAsset?: {
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            } | null;
            navbar_links?: Array<{
                children?: Array<{
                    items: Array<{
                        external?: boolean;
                        href: string;
                        icon?: string;
                        label: string;
                    }>;
                    label?: string;
                }>;
                external?: boolean;
                href: string;
                icon?: string;
                label: string;
            }>;
            site_description?: string | null;
            site_name?: string | null;
            subdomain?: string;
            theme?: {
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
                [key: string]: unknown;
            };
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            user_id?: number;
            widgets?: {
                columnRatio: string;
                left: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
                right: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
                top: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
            };
            [key: string]: unknown;
        }>;
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
        assets: Array<{
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            domain_id?: number;
            id?: number;
            reject_reason?: string | null;
            status?: string;
            type?: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            [key: string]: unknown;
        }>;
        domain: {
            active?: boolean;
            backgrounds?: Array<{
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            }>;
            bannerAsset?: {
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            } | null;
            campaign_policy?: number;
            campaigns?: Array<{
                campaign_id?: string;
                created_by_character_id?: number;
                description?: string | null;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                end_time?: string;
                estimated_killmails?: number | null;
                name?: string;
                public_on_domain?: boolean;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                start_time?: string;
                status?: number;
                visibility?: number;
                [key: string]: unknown;
            }>;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            custom_hostname?: string | null;
            entities?: Array<{
                id: number;
                name?: string;
                type: string;
            }>;
            id?: number;
            logoAsset?: {
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            } | null;
            navbar_links?: Array<{
                children?: Array<{
                    items: Array<{
                        external?: boolean;
                        href: string;
                        icon?: string;
                        label: string;
                    }>;
                    label?: string;
                }>;
                external?: boolean;
                href: string;
                icon?: string;
                label: string;
            }>;
            site_description?: string | null;
            site_name?: string | null;
            subdomain?: string;
            theme?: {
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
                [key: string]: unknown;
            };
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            user_id?: number;
            widgets?: {
                columnRatio: string;
                left: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
                right: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
                top: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
            };
            [key: string]: unknown;
        };
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
    200: Blob | File;
};

export type AdminDomainAssetPreviewResponse = AdminDomainAssetPreviewResponses[keyof AdminDomainAssetPreviewResponses];

export type AdminDomainAssetReviewData = {
    body: {
        /**
         * Review outcome for the uploaded asset.
         */
        action: 'approve' | 'reject';
        /**
         * Operator note recorded with the decision.
         */
        reason?: string;
    };
    path?: never;
    query?: never;
    url: '/admin/domains/{id}/assets/{assetId}/review';
};

export type AdminDomainAssetReviewResponses = {
    /**
     * OK
     */
    200: {
        status: string;
        success: boolean;
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
        domain: {
            active?: boolean;
            backgrounds?: Array<{
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            }>;
            bannerAsset?: {
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            } | null;
            campaign_policy?: number;
            campaigns?: Array<{
                campaign_id?: string;
                created_by_character_id?: number;
                description?: string | null;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                end_time?: string;
                estimated_killmails?: number | null;
                name?: string;
                public_on_domain?: boolean;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                start_time?: string;
                status?: number;
                visibility?: number;
                [key: string]: unknown;
            }>;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            custom_hostname?: string | null;
            entities?: Array<{
                id: number;
                name?: string;
                type: string;
            }>;
            id?: number;
            logoAsset?: {
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            } | null;
            navbar_links?: Array<{
                children?: Array<{
                    items: Array<{
                        external?: boolean;
                        href: string;
                        icon?: string;
                        label: string;
                    }>;
                    label?: string;
                }>;
                external?: boolean;
                href: string;
                icon?: string;
                label: string;
            }>;
            site_description?: string | null;
            site_name?: string | null;
            subdomain?: string;
            theme?: {
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
                [key: string]: unknown;
            };
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            user_id?: number;
            widgets?: {
                columnRatio: string;
                left: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
                right: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
                top: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
            };
            [key: string]: unknown;
        };
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
        rateLimit: {
            request_count: number;
        };
        responseTime: {
            avg_ms: number | null;
            p95_ms: number | null;
        };
        volumeByHour: Array<{
            errors: number;
            hour: string;
            new_items: number;
            total: number;
        }>;
    };
};

export type AdminEsiOverviewResponse = AdminEsiOverviewResponses[keyof AdminEsiOverviewResponses];

export type AdminEsiEntitiesData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Entity name or ID search.
         */
        q?: string;
    };
    url: '/admin/esi-entities';
};

export type AdminEsiEntitiesResponses = {
    /**
     * OK
     */
    200: {
        results: Array<{
            id: number;
            name: string;
            type: string;
        }>;
    };
};

export type AdminEsiEntitiesResponse = AdminEsiEntitiesResponses[keyof AdminEsiEntitiesResponses];

export type AdminEsiLogsData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Page number, counted from 1.
         */
        page?: number;
        /**
         * Match the recorded request source exactly.
         */
        source?: string;
        /**
         * Restrict to successful or failed requests.
         */
        status?: 'success' | 'error';
        /**
         * Restrict to one ESI endpoint family, for example `killmails`.
         */
        endpoint_type?: string;
        /**
         * Return log rows below this log ID.
         */
        after_id?: number;
        /**
         * Match an endpoint or error message.
         */
        search?: string;
        /**
         * Restrict to one character.
         */
        character_id?: number;
        /**
         * Restrict to one corporation.
         */
        corporation_id?: number;
        /**
         * Only requests that returned new items.
         */
        has_new?: boolean;
    };
    url: '/admin/esi-logs';
};

export type AdminEsiLogsResponses = {
    /**
     * OK
     */
    200: {
        limit?: number;
        newRows?: boolean;
        page?: number;
        pages?: number;
        rows: Array<{
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            endpoint: string;
            endpoint_action: string;
            endpoint_type: string;
            error_message: string | null;
            id: number;
            items_returned: number | null;
            method: string;
            new_items: number | null;
            request_duration_ms: number | null;
            source: string;
            status_code: number | null;
            success: boolean;
        }>;
        sources?: Array<string>;
        total?: number;
    };
};

export type AdminEsiLogsResponse = AdminEsiLogsResponses[keyof AdminEsiLogsResponses];

export type AdminModerationData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Restrict to one queue.
         */
        kind?: 'all' | 'comments' | 'bios' | 'bio_character' | 'bio_corporation' | 'bio_alliance';
        /**
         * Restrict to one review state.
         */
        status?: 'all' | 'pending' | 'auto_approved' | 'auto_rejected' | 'approved' | 'rejected';
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Identifier cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        cursor?: number;
    };
    url: '/admin/moderation';
};

export type AdminModerationResponses = {
    /**
     * OK
     */
    200: {
        counts: {
            pending: number;
            pending_bios: number;
            pending_comments: number;
            total: number;
        };
        items: Array<{
            ai_action: string;
            ai_category: string | null;
            ai_max_score: number;
            ai_scores: {
                [key: string]: number;
            };
            ai_source: string;
            alliance_id: number | null;
            alliance_name: string | null;
            body: string;
            body_format: string;
            character_id: number;
            character_name: string;
            comment_context: {
                target_id: number;
                target_slug: string | null;
                target_type: number;
            } | null;
            corporation_id: number | null;
            corporation_name: string | null;
            id: number;
            rendered_html: string;
            review_notes: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            reviewed_at: string | null;
            reviewed_by: number | null;
            status: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            submitted_at: string;
            target_id: number;
            target_kind: number;
        }>;
        nextCursor: number | null;
    };
};

export type AdminModerationResponse = AdminModerationResponses[keyof AdminModerationResponses];

export type AdminModerationLiveQueueAliasData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Restrict to one queue.
         */
        kind?: 'all' | 'comments' | 'bios' | 'bio_character' | 'bio_corporation' | 'bio_alliance';
        /**
         * Restrict to one review state.
         */
        status?: 'all' | 'pending' | 'auto_approved' | 'auto_rejected' | 'approved' | 'rejected';
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Identifier cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        cursor?: number;
    };
    url: '/admin/moderation/queue';
};

export type AdminModerationLiveQueueAliasResponses = {
    /**
     * OK
     */
    200: {
        counts: {
            pending: number;
            pending_bios: number;
            pending_comments: number;
            total: number;
        };
        items: Array<{
            ai_action: string;
            ai_category: string | null;
            ai_max_score: number;
            ai_scores: {
                [key: string]: number;
            };
            ai_source: string;
            alliance_id: number | null;
            alliance_name: string | null;
            body: string;
            body_format: string;
            character_id: number;
            character_name: string;
            comment_context: {
                target_id: number;
                target_slug: string | null;
                target_type: number;
            } | null;
            corporation_id: number | null;
            corporation_name: string | null;
            id: number;
            rendered_html: string;
            review_notes: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            reviewed_at: string | null;
            reviewed_by: number | null;
            status: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            submitted_at: string;
            target_id: number;
            target_kind: number;
        }>;
        nextCursor: number | null;
    };
};

export type AdminModerationLiveQueueAliasResponse = AdminModerationLiveQueueAliasResponses[keyof AdminModerationLiveQueueAliasResponses];

export type AdminModerationReviewData = {
    body: {
        decision: 'approve' | 'reject';
        notes?: string | null;
    };
    path?: never;
    query?: never;
    url: '/admin/moderation/{id}';
};

export type AdminModerationReviewResponses = {
    /**
     * OK
     */
    200: {
        id: number;
        ok: boolean;
        status: string;
    };
};

export type AdminModerationReviewResponse = AdminModerationReviewResponses[keyof AdminModerationReviewResponses];

export type AdminModerationApproveLiveAliasData = {
    body?: {
        notes?: string | null;
    };
    path?: never;
    query?: never;
    url: '/admin/moderation/{id}/approve';
};

export type AdminModerationApproveLiveAliasResponses = {
    /**
     * OK
     */
    200: {
        id: number;
        ok: boolean;
        status: string;
    };
};

export type AdminModerationApproveLiveAliasResponse = AdminModerationApproveLiveAliasResponses[keyof AdminModerationApproveLiveAliasResponses];

export type AdminModerationRejectLiveAliasData = {
    body?: {
        notes?: string | null;
    };
    path?: never;
    query?: never;
    url: '/admin/moderation/{id}/reject';
};

export type AdminModerationRejectLiveAliasResponses = {
    /**
     * OK
     */
    200: {
        id: number;
        ok: boolean;
        status: string;
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
        comments: {
            last24h?: number;
            last7d?: number;
            recent7d?: number;
            total?: number;
        };
        esi: {
            errorRate?: number;
            errors?: number;
            total?: number;
        };
        killmails: {
            last24h?: number;
            last7d?: number;
            recent7d?: number;
            total?: number;
        };
        moderation: {
            flagged?: number;
            pending?: number;
        };
        users: {
            last24h?: number;
            last7d?: number;
            recent7d?: number;
            total?: number;
        };
    };
};

export type AdminOverviewResponse = AdminOverviewResponses[keyof AdminOverviewResponses];

export type AdminUsersListData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Match a character name or ID.
         */
        search?: string;
        /**
         * Ordering for the user rows.
         */
        sort?: 'last_login' | 'created_at' | 'character_name';
        /**
         * Sort direction.
         */
        dir?: 'asc' | 'desc';
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Page number, counted from 1.
         */
        page?: number;
    };
    url: '/admin/users';
};

export type AdminUsersListResponses = {
    /**
     * OK
     */
    200: {
        limit: number;
        page: number;
        pages: number;
        total: number;
        users: Array<{
            character_id?: number;
            character_name?: string;
            character_owner_hash?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string | null;
            discord_user_id?: string | null;
            is_admin?: boolean;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_login?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string | null;
            [key: string]: unknown;
        }>;
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
        config: Array<{
            key?: string;
            value?: string;
            [key: string]: unknown;
        }>;
        esiStats: {
            errors_24h?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_request?: string | null;
            new_items_24h?: number;
            requests_24h?: number;
            total_errors?: number;
            total_new_items?: number;
            total_requests?: number;
        };
        esiToken: {
            scopes?: Array<string>;
            /**
             * UTC timestamp with millisecond precision.
             */
            token_expiry?: string | null;
            [key: string]: unknown;
        } | null;
        user: {
            character_id?: number;
            character_name?: string;
            character_owner_hash?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string | null;
            discord_user_id?: string | null;
            is_admin?: boolean;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_login?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string | null;
            [key: string]: unknown;
        };
    };
};

export type AdminUsersDetailResponse = AdminUsersDetailResponses[keyof AdminUsersDetailResponses];

export type AdminUsersSetDiscordData = {
    body: {
        discord_user_id?: string | null;
    };
    path?: never;
    query?: never;
    url: '/admin/users/{id}/set-discord';
};

export type AdminUsersSetDiscordResponses = {
    /**
     * OK
     */
    200: {
        character_id: number;
        discord_user_id: string | null;
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
        character_id: number;
        is_admin: boolean;
    };
};

export type AdminUsersToggleAdminResponse = AdminUsersToggleAdminResponses[keyof AdminUsersToggleAdminResponses];

export type WalletAdminData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Page number, counted from 1.
         */
        page?: number;
        /**
         * Corporation wallet division.
         */
        division?: number;
    };
    url: '/admin/wallet';
};

export type WalletAdminResponses = {
    /**
     * OK
     */
    200: {
        authorization: {
            authorized_by_admin_character_id?: number;
            authorized_character_id?: number;
            authorized_character_name?: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            disabled?: boolean;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_balance_sync?: string | null;
            last_error?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_journal_sync?: string | null;
            scopes?: Array<string>;
            /**
             * UTC timestamp with millisecond precision.
             */
            token_expiry?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            [key: string]: unknown;
        } | null;
        balances: Array<{
            balance?: string;
            division?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            [key: string]: unknown;
        }>;
        corporation: {
            corporation_id: number;
            name: string;
            ticker: string;
        };
        division: number | null;
        hasMore: boolean;
        journal: Array<{
            amount?: string;
            balance?: string;
            context_id?: number | null;
            context_id_type?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            date?: string;
            description?: string | null;
            division?: number;
            first_party_id?: number | null;
            journal_id?: number;
            reason?: string | null;
            ref_type?: string;
            second_party_id?: number | null;
            tax?: string | null;
            tax_receiver_id?: number | null;
            [key: string]: unknown;
        }>;
        page: number;
        pageSize: number;
        prizeSettlements: Array<{
            campaign_id?: string;
            campaign_name?: string;
            character_id?: number | null;
            character_name?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            claimed_at?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            finalized_at?: string | null;
            funded_total?: string;
            metric_value?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            paid_at?: string | null;
            payment_note?: string | null;
            payout_amount?: string | null;
            payout_percentage?: number | null;
            pool_status?: number;
            rank?: number | null;
            [key: string]: unknown;
        }>;
        requiredScopes: Array<string>;
        totalBalance: string;
        walletReferences: Array<{
            amount?: string;
            corporation_id?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            date?: string;
            division?: number;
            first_party_id?: number | null;
            journal_id?: number;
            note?: string | null;
            reason?: string | null;
            reference_id?: string;
            reference_type?: string;
            status?: number;
            [key: string]: unknown;
        }>;
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
        url: string;
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
        queued: boolean;
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
        character: {
            alliance_id?: number | null;
            alliance_name?: string | null;
            alliance_ticker?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            birthday?: string | null;
            bloodline_name?: string | null;
            character_id?: number;
            corporation_id?: number | null;
            corporation_name?: string | null;
            corporation_ticker?: string | null;
            custom_description?: string | null;
            custom_description_format?: string | null;
            custom_description_html?: string | null;
            description?: string | null;
            faction_id?: number | null;
            faction_name?: string | null;
            gender?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_active?: string | null;
            name?: string;
            palette?: string | null;
            race_name?: string | null;
            security_status?: number;
            title?: string | null;
            [key: string]: unknown;
        };
        corporationHistory: Array<{
            corporation_id: number;
            corporation_name: string;
            corporation_ticker: string;
            kills: number;
            losses: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            start_date: string;
        }>;
        corporationHistoryQueued: boolean;
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
    } | {
        allianceHistory: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            alliance_ticker: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            start_date: string;
        }>;
        corporation: {
            alliance_id?: number | null;
            alliance_name?: string | null;
            alliance_ticker?: string | null;
            ceo_id?: number | null;
            ceo_name?: string | null;
            corporation_id?: number;
            creator_id?: number | null;
            creator_name?: string | null;
            custom_description?: string | null;
            custom_description_format?: string | null;
            custom_description_html?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            date_founded?: string | null;
            description?: string | null;
            faction_id?: number | null;
            faction_name?: string | null;
            friendly_fire?: boolean | null;
            lp_tax_rate?: number | null;
            member_count?: number;
            name?: string;
            palette?: string | null;
            state?: string | null;
            tax_rate?: number;
            ticker?: string;
            type?: string | null;
            url?: string | null;
            war_eligible?: boolean;
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
    } | {
        alliance: {
            alliance_id?: number;
            corporation_count?: number;
            creator_id?: number | null;
            creator_name?: string | null;
            custom_description?: string | null;
            custom_description_format?: string | null;
            custom_description_html?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            date_founded?: string | null;
            executor_corporation_id?: number | null;
            executor_name?: string | null;
            executor_ticker?: string | null;
            faction_id?: number | null;
            faction_name?: string | null;
            member_count?: number;
            name?: string;
            palette?: string | null;
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
    } | {
        faction: {
            corporation_id?: number | null;
            description?: string | null;
            faction_id?: number;
            militia_corporation_id?: number | null;
            name?: string;
            solar_system_id?: number | null;
            station_count?: number;
            station_system_count?: number;
            [key: string]: unknown;
        };
        recentStats: {
            isk_lost: number;
            losses: number;
        };
        stats: {
            isk_lost: number;
            losses: number;
        };
    };
};

export type EntityPageDetailAllianceCompatResponse = EntityPageDetailAllianceCompatResponses[keyof EntityPageDetailAllianceCompatResponses];

export type EntityPageCorporationsAllianceCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Ordering for the corporation rows.
         */
        sort?: 'member_count' | 'name';
        /**
         * Sort direction.
         */
        dir?: 'asc' | 'desc';
    };
    url: '/alliance/{id}/corporations';
};

export type EntityPageCorporationsAllianceCompatResponses = {
    /**
     * OK
     */
    200: {
        corporations: Array<{
            corporation_id: number;
            member_count: number;
            name: string;
            palette: string | null;
            ticker: string;
        }>;
        total: number;
    };
};

export type EntityPageCorporationsAllianceCompatResponse = EntityPageCorporationsAllianceCompatResponses[keyof EntityPageCorporationsAllianceCompatResponses];

export type EntityPageIntelAllianceCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Trailing window in days.
         */
        days?: number;
    };
    url: '/alliance/{id}/intel';
};

export type EntityPageIntelAllianceCompatResponses = {
    /**
     * OK
     */
    200: {
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
            likelihood?: string;
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
    } | {
        activeMembers: {
            days_30: number;
            days_7: number;
            days_90: number;
        };
        allies: Array<{
            id?: number;
            kills_given?: number;
            kills_taken?: number;
            mutual_kills?: number;
            name?: string;
            shared_enemy_kills?: number;
            total?: number;
        }>;
        census: {
            caps: number;
            corps?: Array<{
                id?: number;
                name?: string;
                total?: number;
                [key: string]: unknown;
            }>;
            droppers: number;
            fcs: number;
            logis: number;
            supers: number;
            total: number;
        };
        enemies: Array<{
            id?: number;
            kills_given?: number;
            kills_taken?: number;
            mutual_kills?: number;
            name?: string;
            shared_enemy_kills?: number;
            total?: number;
        }>;
        huntingGrounds: Array<{
            active_characters: number;
            id: number;
            name: string;
        }>;
        recentDepartures: Array<{
            current_corp?: {
                id: number;
                name: string | null;
            } | null;
            id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            joined_at?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            left_at?: string | null;
            name: string;
            previous_corp?: {
                id: number;
                name: string | null;
            } | null;
        }>;
        recentJoins: Array<{
            current_corp?: {
                id: number;
                name: string | null;
            } | null;
            id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            joined_at?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            left_at?: string | null;
            name: string;
            previous_corp?: {
                id: number;
                name: string | null;
            } | null;
        }>;
    };
};

export type EntityPageIntelAllianceCompatResponse = EntityPageIntelAllianceCompatResponses[keyof EntityPageIntelAllianceCompatResponses];

export type EntityPageMembersAllianceCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Ordering for the member rows.
         */
        sort?: 'name' | 'last_active' | 'security_status';
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Page number, counted from 1.
         */
        page?: number;
        /**
         * Restrict an alliance's members to one corporation.
         */
        corporation_id?: number;
        /**
         * Only members active within this many days. 0 disables the filter.
         */
        activity?: number;
    };
    url: '/alliance/{id}/members';
};

export type EntityPageMembersAllianceCompatResponses = {
    /**
     * OK
     */
    200: {
        limit: number;
        members: Array<{
            character_id: number;
            corporation_id?: number | null;
            is_capital_pilot: boolean;
            is_fc: boolean;
            is_logi: boolean;
            kills_90d: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_active: string | null;
            losses_90d: number;
            name: string;
            security_status: number;
        }>;
        page: number;
        total: number;
    };
};

export type EntityPageMembersAllianceCompatResponse = EntityPageMembersAllianceCompatResponses[keyof EntityPageMembersAllianceCompatResponses];

export type EntityPageStatsAllianceCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Trailing window in days. 0 covers the whole record.
         */
        days?: number;
    };
    url: '/alliance/{id}/stats';
};

export type EntityPageStatsAllianceCompatResponses = {
    /**
     * OK
     */
    200: {
        activity?: {
            kills: Array<Array<number>>;
            losses: Array<Array<number>>;
        };
        diesToAlliances: Array<{
            count: number;
            id: number;
            isk_value: number;
            name: string;
        }>;
        diesToCorporations: Array<{
            count: number;
            id: number;
            isk_value: number;
            name: string;
        }>;
        fliesWithAlliances?: Array<{
            count: number;
            id: number;
            isk_value: number;
            name: string;
        }>;
        fliesWithCorporations?: Array<{
            count: number;
            id: number;
            isk_value: number;
            name: string;
        }>;
        heatMap?: {
            [key: string]: number;
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
        topShipsLost: Array<{
            count: number;
            ship_name: string;
            ship_type_id: number;
        }>;
        topShipsUsed: Array<{
            count: number;
            ship_name: string;
            ship_type_id: number;
        }>;
    };
};

export type EntityPageStatsAllianceCompatResponse = EntityPageStatsAllianceCompatResponses[keyof EntityPageStatsAllianceCompatResponses];

export type EntityTopAllianceCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Which half of the leaderboard set to build. `right` also accepts `days=alltime`.
         */
        slice?: 'left' | 'right';
        /**
         * Window in days, between 1/24 and 365. Send `alltime` with `slice=right` for the unbounded set. Default 7.
         */
        days?: string;
    };
    url: '/alliance/{id}/top';
};

export type EntityTopAllianceCompatResponses = {
    /**
     * OK
     */
    200: {
        achievementPoints?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        charactersByIsk?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        charactersByKills?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        charactersByPoints?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        constellations?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        corporationsByKills?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        killedAlliances?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        killedByAlliances?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        killedByCorporations?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        killedCorporations?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        recentMembers?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        regions?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        shipsUsed?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        soloKillers?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        systems?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
    };
};

export type EntityTopAllianceCompatResponse = EntityTopAllianceCompatResponses[keyof EntityTopAllianceCompatResponses];

export type AlliancesData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Case-insensitive name prefix to match.
         */
        name?: string;
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
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
         * Start of the window, for type=range.
         */
        from?: string;
        /**
         * Entity IDs to resolve, at most 100 per request.
         */
        ids: Array<number>;
        /**
         * End of the window, for type=range.
         */
        to?: string;
        /**
         * Aggregation period. Falls back to the type query parameter, then alltime.
         */
        type?: 'alltime' | 'weekly' | 'range';
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
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
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
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
        /**
         * Descending cursor, walking newest to oldest. Pass the previous response's pagination cursor to fetch the next page. Mutually exclusive with `after`, which it overrides.
         */
        before?: number;
    };
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
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
        /**
         * Descending cursor, walking newest to oldest. Pass the previous response's pagination cursor to fetch the next page. Mutually exclusive with `after`, which it overrides.
         */
        before?: number;
    };
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
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
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
        announcements: Array<{
            body_html: string;
            body_md: string;
            color: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            expires_at: string;
            icon: string | null;
            id: number;
            link_label: string | null;
            link_url: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            starts_at: string;
            tier: number;
            title: string;
        }>;
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
        dismissedIds: Array<number>;
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
        ok: boolean;
    };
};

export type AnnouncementDismissCompatResponse = AnnouncementDismissCompatResponses[keyof AnnouncementDismissCompatResponses];

export type EveLoginCallbackLegacyData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Authorization code returned by EVE SSO.
         */
        code?: string;
        /**
         * Signed state issued when the flow started.
         */
        state?: string;
        /**
         * Error returned by EVE SSO, when it refuses.
         */
        error?: string;
    };
    url: '/auth/callback';
};

export type EveLoginCallbackData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Authorization code returned by EVE SSO.
         */
        code?: string;
        /**
         * Signed state issued when the flow started.
         */
        state?: string;
        /**
         * Error returned by EVE SSO, when it refuses.
         */
        error?: string;
    };
    url: '/auth/eve/callback';
};

export type EveLoginStartData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Same-origin path to return to after login. `redirect` is accepted as an alias.
         */
        returnTo?: string;
        /**
         * Alias for `returnTo`.
         */
        redirect?: string;
        /**
         * Delay applied before the redirect, in seconds.
         */
        delay?: string;
        /**
         * Send `0` to drop the character killmail scope from the authorization request.
         */
        charKm?: '0' | '1';
        /**
         * Send `0` to drop the corporation killmail scope from the authorization request.
         */
        corpKm?: '0' | '1';
    };
    url: '/auth/eve/start';
};

export type AuthLoginLegacyData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Same-origin path to return to after login. `redirect` is accepted as an alias.
         */
        returnTo?: string;
        /**
         * Alias for `returnTo`.
         */
        redirect?: string;
        /**
         * Delay applied before the redirect, in seconds.
         */
        delay?: string;
        /**
         * Send `0` to drop the character killmail scope from the authorization request.
         */
        charKm?: '0' | '1';
        /**
         * Send `0` to drop the corporation killmail scope from the authorization request.
         */
        corpKm?: '0' | '1';
    };
    url: '/auth/login';
};

export type AuthLoginLegacyResponses = {
    /**
     * OK
     */
    200: {
        url: string;
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
        success: boolean;
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
        user: {
            allianceId: number | null;
            allianceName: string | null;
            characterId: number;
            characterName: string;
            characterOwnerHash?: string;
            corporationId: number | null;
            corporationName: string | null;
            isAdmin: boolean;
            lastSeenNotificationId: number;
            settings?: {
                boards?: {
                    dismissed: Array<string>;
                    pinned: Array<string>;
                };
                /**
                 * Default tab keyed by page type.
                 */
                defaultTabs?: {
                    [key: string]: unknown;
                };
                /**
                 * User-selected theme settings.
                 */
                theme?: {
                    [key: string]: unknown;
                };
            };
        } | null;
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
        scopes: Array<string>;
        /**
         * UTC timestamp with millisecond precision.
         */
        token_expiry: string | null;
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
        images: Array<{
            source: string;
            subreddit: string;
            title: string;
            url: string;
        }>;
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
        systemIds: Array<number>;
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
        alliances: Array<{
            alliance_id?: number | null;
            alliance_name?: string | null;
            id: number;
            name: string;
            type: string;
        }>;
        corporations: Array<{
            alliance_id?: number | null;
            alliance_name?: string | null;
            id: number;
            name: string;
            type: string;
        }>;
        killCount: number;
    };
};

export type BattleGeneratorEntitiesResponse = BattleGeneratorEntitiesResponses[keyof BattleGeneratorEntitiesResponses];

export type BattleGeneratorPreviewData = {
    body: {
        sides: Array<ConflictBattleGeneratorSide>;
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
        alliances_involved: number;
        battle_id?: number | null;
        characters_involved: number;
        corporations_involved: number;
        duration_minutes: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        end_time: string;
        is_custom: boolean;
        is_multi_party: boolean;
        kill_count: number;
        killmail_id?: number;
        region_id: number | null;
        region_name: string | null;
        solar_system_id: number;
        solar_system_name: string | null;
        solar_system_security: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        start_time: string;
        team_entities: Array<{
            alliances: Array<number>;
            corps: Array<number>;
        }>;
        teams: Array<{
            alliance_count: number;
            alliances: Array<{
                alliance_id: number | null;
                alliance_name: string | null;
                corporations: Array<{
                    corporation_id: number;
                    corporation_name: string | null;
                    isk_destroyed: number;
                    isk_lost: number;
                    kills: number;
                    losses: number;
                }>;
                isk_destroyed: number;
                isk_lost: number;
                kills: number;
                losses: number;
            }>;
            corp_count: number;
            dominant_corp_palette: string | null;
            team_index: number;
            total_isk_destroyed: number;
            total_isk_lost: number;
            total_kills: number;
            total_losses: number;
        }>;
        total_damage: number;
        total_isk_destroyed: number;
        unsided?: {
            alliance_count: number;
            alliances: Array<{
                alliance_id: number | null;
                alliance_name: string | null;
                corporations: Array<{
                    corporation_id: number;
                    corporation_name: string | null;
                    isk_destroyed: number;
                    isk_lost: number;
                    kills: number;
                    losses: number;
                }>;
                isk_destroyed: number;
                isk_lost: number;
                kills: number;
                losses: number;
            }>;
            corp_count: number;
            isk_destroyed: number;
            isk_lost: number;
            kills: number;
            losses: number;
        } | null;
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
        teams: [
            ConflictBattleSaveTeam,
            ConflictBattleSaveTeam
        ];
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
        battle_id: number;
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
        battle_id: number;
        redirect: string;
    } | {
        alliances_involved: number;
        battle_id?: number | null;
        characters_involved: number;
        corporations_involved: number;
        duration_minutes: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        end_time: string;
        is_custom: boolean;
        is_multi_party: boolean;
        kill_count: number;
        killmail_id?: number;
        region_id: number | null;
        region_name: string | null;
        solar_system_id: number;
        solar_system_name: string | null;
        solar_system_security: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        start_time: string;
        team_entities: Array<{
            alliances: Array<number>;
            corps: Array<number>;
        }>;
        teams: Array<{
            alliance_count: number;
            alliances: Array<{
                alliance_id: number | null;
                alliance_name: string | null;
                corporations: Array<{
                    corporation_id: number;
                    corporation_name: string | null;
                    isk_destroyed: number;
                    isk_lost: number;
                    kills: number;
                    losses: number;
                }>;
                isk_destroyed: number;
                isk_lost: number;
                kills: number;
                losses: number;
            }>;
            corp_count: number;
            dominant_corp_palette: string | null;
            team_index: number;
            total_isk_destroyed: number;
            total_isk_lost: number;
            total_kills: number;
            total_losses: number;
        }>;
        total_damage: number;
        total_isk_destroyed: number;
        unsided?: {
            alliance_count: number;
            alliances: Array<{
                alliance_id: number | null;
                alliance_name: string | null;
                corporations: Array<{
                    corporation_id: number;
                    corporation_name: string | null;
                    isk_destroyed: number;
                    isk_lost: number;
                    kills: number;
                    losses: number;
                }>;
                isk_destroyed: number;
                isk_lost: number;
                kills: number;
                losses: number;
            }>;
            corp_count: number;
            isk_destroyed: number;
            isk_lost: number;
            kills: number;
            losses: number;
        } | null;
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
        team_count: number;
        teams: Array<{
            by_group: Array<{
                count: number;
                damage_done: number;
                damage_taken: number;
                isk_lost: number;
                key: string;
                losses: number;
                name: string | null;
                rank: number;
                ship_group_id?: number;
                ship_type_id?: number;
            }>;
            by_ship: Array<{
                count: number;
                damage_done: number;
                damage_taken: number;
                isk_lost: number;
                key: string;
                losses: number;
                name: string | null;
                rank: number;
                ship_group_id?: number;
                ship_type_id?: number;
            }>;
            individuals: Array<{
                alliance_id: number | null;
                alliance_name: string | null;
                character_id: number;
                character_name: string | null;
                corporation_id: number | null;
                corporation_name: string | null;
                damage_done: number;
                damage_taken: number;
                deaths: number;
                isk_lost: number;
                rank: number;
                ship_group_id: number | null;
                ship_group_name: string | null;
                ship_name: string | null;
                ship_type_id: number;
                team_index: number;
            }>;
            team_index: number;
        }>;
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
        teams: Array<{
            capitals: Array<{
                alliance_id: number | null;
                alliance_name: string | null;
                character_id: number;
                character_name: string | null;
                confirmed?: boolean;
                corporation_id: number | null;
                corporation_name: string | null;
                damage_done: number;
                died: boolean;
                ship_group_id: number | null;
                ship_group_name: string | null;
                ship_name: string | null;
                ship_type_id: number;
            }>;
            fcs: Array<{
                alliance_id: number | null;
                alliance_name: string | null;
                character_id: number;
                character_name: string | null;
                confirmed?: boolean;
                corporation_id: number | null;
                corporation_name: string | null;
                damage_done: number;
                died: boolean;
                ship_group_id: number | null;
                ship_group_name: string | null;
                ship_name: string | null;
                ship_type_id: number;
            }>;
            logistics: Array<{
                alliance_id: number | null;
                alliance_name: string | null;
                character_id: number;
                character_name: string | null;
                confirmed?: boolean;
                corporation_id: number | null;
                corporation_name: string | null;
                damage_done: number;
                died: boolean;
                ship_group_id: number | null;
                ship_group_name: string | null;
                ship_name: string | null;
                ship_type_id: number;
            }>;
            team_index: number;
        }>;
    };
};

export type KillmailBattleIntelResponse = KillmailBattleIntelResponses[keyof KillmailBattleIntelResponses];

export type KillmailBattleKilllistData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
        /**
         * Page number for offset paging. Leave at 0 to page by cursor.
         */
        page?: number;
    };
    url: '/battle/killmail/{id}/killlist';
};

export type KillmailBattleKilllistResponses = {
    /**
     * OK
     */
    200: {
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
    };
};

export type KillmailBattleKilllistResponse = KillmailBattleKilllistResponses[keyof KillmailBattleKilllistResponses];

export type KillmailBattleMostValuableData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Restrict the losses to one category of hull.
         */
        dataType?: 'most_valuable_kills' | 'most_valuable_ships' | 'most_valuable_structures';
        /**
         * Restrict to one team index in the battle.
         */
        team?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/battle/killmail/{id}/most-valuable';
};

export type KillmailBattleMostValuableResponses = {
    /**
     * OK
     */
    200: {
        entries: Array<{
            killmail_hash: string;
            killmail_id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            killmail_time?: string;
            ship_name: string;
            ship_type_id: number;
            total_value: number;
            victim_alliance_name: string | null;
            victim_character_id: number | null;
            victim_character_name: string | null;
            victim_corporation_name: string | null;
        }>;
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
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
        alliances_involved: number;
        battle_id?: number | null;
        characters_involved: number;
        corporations_involved: number;
        duration_minutes: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        end_time: string;
        is_custom: boolean;
        is_multi_party: boolean;
        kill_count: number;
        killmail_id?: number;
        region_id: number | null;
        region_name: string | null;
        solar_system_id: number;
        solar_system_name: string | null;
        solar_system_security: number | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        start_time: string;
        team_entities: Array<{
            alliances: Array<number>;
            corps: Array<number>;
        }>;
        teams: Array<{
            alliance_count: number;
            alliances: Array<{
                alliance_id: number | null;
                alliance_name: string | null;
                corporations: Array<{
                    corporation_id: number;
                    corporation_name: string | null;
                    isk_destroyed: number;
                    isk_lost: number;
                    kills: number;
                    losses: number;
                }>;
                isk_destroyed: number;
                isk_lost: number;
                kills: number;
                losses: number;
            }>;
            corp_count: number;
            dominant_corp_palette: string | null;
            team_index: number;
            total_isk_destroyed: number;
            total_isk_lost: number;
            total_kills: number;
            total_losses: number;
        }>;
        total_damage: number;
        total_isk_destroyed: number;
        unsided?: {
            alliance_count: number;
            alliances: Array<{
                alliance_id: number | null;
                alliance_name: string | null;
                corporations: Array<{
                    corporation_id: number;
                    corporation_name: string | null;
                    isk_destroyed: number;
                    isk_lost: number;
                    kills: number;
                    losses: number;
                }>;
                isk_destroyed: number;
                isk_lost: number;
                kills: number;
                losses: number;
            }>;
            corp_count: number;
            isk_destroyed: number;
            isk_lost: number;
            kills: number;
            losses: number;
        } | null;
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
        team_count: number;
        teams: Array<{
            by_group: Array<{
                count: number;
                damage_done: number;
                damage_taken: number;
                isk_lost: number;
                key: string;
                losses: number;
                name: string | null;
                rank: number;
                ship_group_id?: number;
                ship_type_id?: number;
            }>;
            by_ship: Array<{
                count: number;
                damage_done: number;
                damage_taken: number;
                isk_lost: number;
                key: string;
                losses: number;
                name: string | null;
                rank: number;
                ship_group_id?: number;
                ship_type_id?: number;
            }>;
            individuals: Array<{
                alliance_id: number | null;
                alliance_name: string | null;
                character_id: number;
                character_name: string | null;
                corporation_id: number | null;
                corporation_name: string | null;
                damage_done: number;
                damage_taken: number;
                deaths: number;
                isk_lost: number;
                rank: number;
                ship_group_id: number | null;
                ship_group_name: string | null;
                ship_name: string | null;
                ship_type_id: number;
                team_index: number;
            }>;
            team_index: number;
        }>;
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
        teams: Array<{
            capitals: Array<{
                alliance_id: number | null;
                alliance_name: string | null;
                character_id: number;
                character_name: string | null;
                confirmed?: boolean;
                corporation_id: number | null;
                corporation_name: string | null;
                damage_done: number;
                died: boolean;
                ship_group_id: number | null;
                ship_group_name: string | null;
                ship_name: string | null;
                ship_type_id: number;
            }>;
            fcs: Array<{
                alliance_id: number | null;
                alliance_name: string | null;
                character_id: number;
                character_name: string | null;
                confirmed?: boolean;
                corporation_id: number | null;
                corporation_name: string | null;
                damage_done: number;
                died: boolean;
                ship_group_id: number | null;
                ship_group_name: string | null;
                ship_name: string | null;
                ship_type_id: number;
            }>;
            logistics: Array<{
                alliance_id: number | null;
                alliance_name: string | null;
                character_id: number;
                character_name: string | null;
                confirmed?: boolean;
                corporation_id: number | null;
                corporation_name: string | null;
                damage_done: number;
                died: boolean;
                ship_group_id: number | null;
                ship_group_name: string | null;
                ship_name: string | null;
                ship_type_id: number;
            }>;
            team_index: number;
        }>;
    };
};

export type BattleReportIntelResponse = BattleReportIntelResponses[keyof BattleReportIntelResponses];

export type BattleReportKilllistData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
        /**
         * Page number for offset paging. Leave at 0 to page by cursor.
         */
        page?: number;
    };
    url: '/battle/{id}/killlist';
};

export type BattleReportKilllistResponses = {
    /**
     * OK
     */
    200: {
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
    };
};

export type BattleReportKilllistResponse = BattleReportKilllistResponses[keyof BattleReportKilllistResponses];

export type BattleReportMostValuableData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Restrict the losses to one category of hull.
         */
        dataType?: 'most_valuable_kills' | 'most_valuable_ships' | 'most_valuable_structures';
        /**
         * Restrict to one team index in the battle.
         */
        team?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/battle/{id}/most-valuable';
};

export type BattleReportMostValuableResponses = {
    /**
     * OK
     */
    200: {
        entries: Array<{
            killmail_hash: string;
            killmail_id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            killmail_time?: string;
            ship_name: string;
            ship_type_id: number;
            total_value: number;
            victim_alliance_name: string | null;
            victim_character_id: number | null;
            victim_character_name: string | null;
            victim_corporation_name: string | null;
        }>;
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
    };
};

export type BattleReportTimelineResponse = BattleReportTimelineResponses[keyof BattleReportTimelineResponses];

export type BattlesData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Ordering for the battle rows.
         */
        sort?: 'battle_id' | 'total_isk_destroyed' | 'kill_count' | 'start_time';
        /**
         * Sort direction.
         */
        order?: 'asc' | 'desc';
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Page number, counted from 1.
         */
        page?: number;
        /**
         * Only battles starting after this.
         */
        start_after?: string;
        /**
         * Only battles starting before this.
         */
        start_before?: string;
    };
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
    query?: {
        /**
         * Ordering for the battle rows.
         */
        sort?: 'battle_id' | 'total_isk_destroyed' | 'kill_count' | 'start_time';
        /**
         * Sort direction.
         */
        order?: 'asc' | 'desc';
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Page number, counted from 1.
         */
        page?: number;
        /**
         * Only battles starting after this.
         */
        start_after?: string;
        /**
         * Only battles starting before this.
         */
        start_before?: string;
    };
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
    query?: {
        /**
         * Ordering for the battle rows.
         */
        sort?: 'battle_id' | 'total_isk_destroyed' | 'kill_count' | 'start_time';
        /**
         * Sort direction.
         */
        order?: 'asc' | 'desc';
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Page number, counted from 1.
         */
        page?: number;
        /**
         * Only battles starting after this.
         */
        start_after?: string;
        /**
         * Only battles starting before this.
         */
        start_before?: string;
    };
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
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Identifier cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        cursor?: number;
        /**
         * Restrict to one tag.
         */
        tag?: string;
    };
    url: '/blog';
};

export type BlogPostsResponses = {
    /**
     * OK
     */
    200: {
        nextCursor: string | null;
        posts: Array<{
            author_alliance_id: number | null;
            author_alliance_name: string | null;
            author_corporation_id: number | null;
            author_corporation_name: string | null;
            author_id: number;
            author_name: string;
            body_html: string;
            body_md: string;
            cover_image_url: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            excerpt: string | null;
            id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            published_at: string | null;
            slug: string;
            status: number;
            tags: Array<string>;
            title: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
        }>;
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
        post: {
            author_alliance_id: number | null;
            author_alliance_name: string | null;
            author_corporation_id: number | null;
            author_corporation_name: string | null;
            author_id: number;
            author_name: string;
            body_html: string;
            body_md: string;
            cover_image_url: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            excerpt: string | null;
            id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            published_at: string | null;
            slug: string;
            status: number;
            tags: Array<string>;
            title: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
        };
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
        atCapacity: boolean;
        authenticated: boolean;
        boards: Array<{
            host: string;
            key: string;
            name: string;
            pinned: boolean;
            tracked: boolean;
            url: string;
        }>;
        current: {
            key: string;
            listed: boolean;
            name: string;
        } | null;
    };
};

export type BoardsMineCompatResponse = BoardsMineCompatResponses[keyof BoardsMineCompatResponses];

export type CampaignCreateLegacyData = {
    body: {
        allowedEntities?: Array<{
            /**
             * An integer. A numeric string is accepted for compatibility.
             */
            id: number | string;
            name?: string;
            type: 'character' | 'corporation' | 'alliance';
        }>;
        description?: string | null;
        endTime?: string | null;
        location?: CampaignLocationDocument;
        name: string;
        prizePool?: CampaignPrizePoolDocument;
        sides?: Array<{
            entities: Array<{
                /**
                 * An integer. A numeric string is accepted for compatibility.
                 */
                id: number | string;
                name?: string;
                type: 'character' | 'corporation' | 'alliance';
            }>;
            name?: string;
        }>;
        startTime: string;
        /**
         * An integer. A numeric string is accepted for compatibility.
         */
        visibility?: number | string;
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
        campaign_id: string;
        estimated_killmails: number;
        initial_contribution: string;
        replayed: boolean;
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
        deleted: boolean;
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
        allowed_entities: Array<{
            id: number;
            name?: string;
            type: string;
        }>;
        campaign_id: string;
        /**
         * UTC timestamp with millisecond precision.
         */
        created_at: string;
        creator: {
            character_id: number;
            name: string | null;
        };
        daily: {
            granularity: string;
            rows: Array<{
                isk_destroyed: number;
                isk_lost: number;
                kills: number;
                losses: number;
                period: string;
                side_index: number;
            }>;
        };
        description: string | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        end_time: string | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        last_activity_at: string | null;
        location: {
            constellationIds?: Array<number>;
            regionIds?: Array<number>;
            systemIds?: Array<number>;
        } | null;
        location_details: {
            constellations: Array<{
                id: number;
                name: string;
            }>;
            regions: Array<{
                id: number;
                name: string;
            }>;
            systems: Array<{
                id: number;
                name: string;
            }>;
        };
        mode: string;
        name: string;
        prize_pool: {
            contribution_count: number;
            contributions: Array<{
                amount: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                contributed_at: string;
                contributor_id: number | null;
                contributor_name: string;
                contributor_type: string;
                id: string;
                source: string;
            }>;
            discord_url: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            finalized_at: string | null;
            funded_total: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            funding_closes_at: string | null;
            funding_reference: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_wallet_sync: string | null;
            metric: number;
            metric_label: string;
            payout_percentages: Array<number>;
            projected_lead_percent: number | null;
            results: Array<{
                can_claim: boolean;
                character_id: number;
                character_name: string;
                claimed: boolean;
                metric_value: string;
                paid: boolean;
                payout_amount: number;
                payout_percentage: number;
                rank: number;
                secondary_value: string;
            }>;
            /**
             * UTC timestamp with millisecond precision.
             */
            rules_locked_at: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            settles_at: string | null;
            status: number;
            winner_count: number;
        } | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        processed_through: string | null;
        processing: {
            estimated_killmails: number | null;
            last_duration_ms: number | null;
            last_error: string | null;
            last_killmails: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_started_at: string | null;
            note: string | null;
            paused: boolean;
        };
        sides: Array<{
            entities: Array<{
                entity_id: number;
                entity_type: number;
                isk_destroyed?: number;
                isk_lost?: number;
                kills?: number;
                losses?: number;
                name?: string | null;
            }>;
            isk_destroyed: number;
            isk_lost: number;
            kills: number;
            losses: number;
            name: string;
            palette: string | null;
            side_index: number;
        }>;
        /**
         * UTC timestamp with millisecond precision.
         */
        start_time: string;
        stats: {
            intelBySide: {
                [key: string]: {
                    capitals: Array<{
                        allianceName: string | null;
                        characterId: number;
                        corporationName: string | null;
                        damage: number;
                        died: boolean;
                        name: string | null;
                        shipGroupName: string | null;
                        shipName: string | null;
                        shipTypeId: number | null;
                    }>;
                    capitalsCount: number;
                    fcs: Array<{
                        allianceName: string | null;
                        characterId: number;
                        corporationName: string | null;
                        damage: number;
                        died: boolean;
                        name: string | null;
                        shipGroupName: string | null;
                        shipName: string | null;
                        shipTypeId: number | null;
                    }>;
                    logistics: Array<{
                        allianceName: string | null;
                        characterId: number;
                        corporationName: string | null;
                        damage: number;
                        died: boolean;
                        name: string | null;
                        shipGroupName: string | null;
                        shipName: string | null;
                        shipTypeId: number | null;
                    }>;
                    logisticsCount: number;
                } | null;
            };
            mostValuable: Array<{
                killmailId: number;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                killmailTime: string;
                shipName: string | null;
                shipTypeId: number | null;
                value: number;
                victimCharacterId: number | null;
                victimCharacterName: string | null;
                victimCorporationId: number | null;
                victimCorporationName: string | null;
                victimSide: number | null;
            }>;
            shipClassesBySide: {
                [key: string]: Array<{
                    groupId: number;
                    iskLost: number;
                    losses: number;
                    name: string | null;
                }>;
            };
            shipClassesOverall: Array<{
                groupId: number;
                iskLost: number;
                losses: number;
                name: string | null;
            }>;
            topKillersBySide: {
                [key: string]: Array<{
                    allianceId?: number | null;
                    allianceName?: string | null;
                    allianceTicker?: string | null;
                    characterId: number;
                    corporationId?: number | null;
                    corporationName?: string | null;
                    corporationTicker?: string | null;
                    iskDestroyed?: number;
                    iskLost?: number;
                    kills?: number;
                    losses?: number;
                    name: string | null;
                }>;
            };
            topKillersOverall: Array<{
                allianceId?: number | null;
                allianceName?: string | null;
                allianceTicker?: string | null;
                characterId: number;
                corporationId?: number | null;
                corporationName?: string | null;
                corporationTicker?: string | null;
                iskDestroyed?: number;
                iskLost?: number;
                kills?: number;
                losses?: number;
                name: string | null;
            }>;
            topSystems: Array<{
                iskDestroyed: number;
                kills: number;
                name: string | null;
                regionId: number | null;
                regionName: string | null;
                systemId: number;
            }>;
            topVictimsBySide: {
                [key: string]: Array<{
                    allianceId?: number | null;
                    allianceName?: string | null;
                    allianceTicker?: string | null;
                    characterId: number;
                    corporationId?: number | null;
                    corporationName?: string | null;
                    corporationTicker?: string | null;
                    iskDestroyed?: number;
                    iskLost?: number;
                    kills?: number;
                    losses?: number;
                    name: string | null;
                }>;
            };
            topVictimsOverall: Array<{
                allianceId?: number | null;
                allianceName?: string | null;
                allianceTicker?: string | null;
                characterId: number;
                corporationId?: number | null;
                corporationName?: string | null;
                corporationTicker?: string | null;
                iskDestroyed?: number;
                iskLost?: number;
                kills?: number;
                losses?: number;
                name: string | null;
            }>;
            totals: {
                alliancesInvolved: number;
                charactersInvolved: number;
                corporationsInvolved: number;
                iskDestroyed: number;
                killCount: number;
            };
        } | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        stats_updated_at: string | null;
        status: number;
        visibility: number;
    };
};

export type CampaignDetailLegacyResponse = CampaignDetailLegacyResponses[keyof CampaignDetailLegacyResponses];

export type CampaignUpdateLegacyData = {
    body: {
        allowedEntities?: Array<{
            /**
             * An integer. A numeric string is accepted for compatibility.
             */
            id: number | string;
            name?: string;
            type: 'character' | 'corporation' | 'alliance';
        }>;
        archived?: boolean;
        description?: string | null;
        endTime?: string | null;
        name?: string;
        resumeProcessing?: boolean;
        sides?: Array<{
            entities: Array<{
                /**
                 * An integer. A numeric string is accepted for compatibility.
                 */
                id: number | string;
                name?: string;
                type: 'character' | 'corporation' | 'alliance';
            }>;
            name?: string;
        }>;
        /**
         * An integer. A numeric string is accepted for compatibility.
         */
        visibility?: number | string;
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
        recompute?: boolean;
        updated: boolean;
    };
};

export type CampaignUpdateLegacyResponse = CampaignUpdateLegacyResponses[keyof CampaignUpdateLegacyResponses];

export type CampaignKilllistLegacyData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Restrict to one campaign side index. Must match a side the campaign defines.
         */
        side?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
    url: '/campaign/{id}/killlist';
};

export type CampaignKilllistLegacyResponses = {
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
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
        claimed: boolean;
        rank: number;
    };
};

export type CampaignPrizeClaimLegacyResponse = CampaignPrizeClaimLegacyResponses[keyof CampaignPrizeClaimLegacyResponses];

export type CampaignPrizeContributeLegacyData = {
    body: {
        /**
         * A number. A numeric string is accepted for compatibility.
         */
        amount: number | string;
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
        balance: string;
        contributed: string;
        replayed: boolean;
    };
};

export type CampaignPrizeContributeLegacyResponse = CampaignPrizeContributeLegacyResponses[keyof CampaignPrizeContributeLegacyResponses];

export type CampaignUpdateBrowserLegacyData = {
    body: {
        allowedEntities?: Array<{
            /**
             * An integer. A numeric string is accepted for compatibility.
             */
            id: number | string;
            name?: string;
            type: 'character' | 'corporation' | 'alliance';
        }>;
        archived?: boolean;
        description?: string | null;
        endTime?: string | null;
        name?: string;
        resumeProcessing?: boolean;
        sides?: Array<{
            entities: Array<{
                /**
                 * An integer. A numeric string is accepted for compatibility.
                 */
                id: number | string;
                name?: string;
                type: 'character' | 'corporation' | 'alliance';
            }>;
            name?: string;
        }>;
        /**
         * An integer. A numeric string is accepted for compatibility.
         */
        visibility?: number | string;
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
        recompute?: boolean;
        updated: boolean;
    };
};

export type CampaignUpdateBrowserLegacyResponse = CampaignUpdateBrowserLegacyResponses[keyof CampaignUpdateBrowserLegacyResponses];

export type CampaignsData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Which campaigns to list. Defaults to `all` when the request is scoped to an entity.
         */
        status?: 'active' | 'archived' | 'private' | 'all';
        /**
         * Restrict to two-sided conflicts or area campaigns.
         */
        mode?: 'conflict' | 'area';
        /**
         * Scope the list to one entity. Requires `entityId`.
         */
        entityType?: 'character' | 'corporation' | 'alliance';
        /**
         * Entity ID to scope to. Requires `entityType`.
         */
        entityId?: number;
        /**
         * Name search, truncated to 100 characters.
         */
        q?: string;
        /**
         * Page number, counted from 1.
         */
        page?: number;
    };
    url: '/campaigns';
};

export type CampaignsResponses = {
    /**
     * OK
     */
    200: {
        campaigns: Array<{
            campaign_id: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            creator: {
                character_id: number;
                name: string | null;
            };
            description: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            end_time: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_activity_at: string | null;
            location: {
                constellationIds?: Array<number>;
                regionIds?: Array<number>;
                systemIds?: Array<number>;
            } | null;
            mode: string;
            name: string;
            sides: Array<{
                entities: Array<{
                    entity_id: number;
                    entity_type: number;
                    isk_destroyed?: number;
                    isk_lost?: number;
                    kills?: number;
                    losses?: number;
                    name?: string | null;
                }>;
                isk_destroyed: number;
                isk_lost: number;
                kills: number;
                losses: number;
                name: string;
                palette: string | null;
                side_index: number;
            }>;
            /**
             * UTC timestamp with millisecond precision.
             */
            start_time: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            stats_updated_at: string | null;
            status: number;
            totals: {
                alliancesInvolved: number;
                charactersInvolved: number;
                corporationsInvolved: number;
                iskDestroyed: number;
                killCount: number;
            } | null;
            visibility: number;
        }>;
        counts: {
            active: number;
            archived: number;
            private: number;
        };
        hasMore: boolean;
        page: number;
        total: number;
    };
};

export type CampaignsResponse = CampaignsResponses[keyof CampaignsResponses];

export type CampaignCreateData = {
    body: {
        allowedEntities?: Array<{
            /**
             * An integer. A numeric string is accepted for compatibility.
             */
            id: number | string;
            name?: string;
            type: 'character' | 'corporation' | 'alliance';
        }>;
        description?: string | null;
        endTime?: string | null;
        location?: CampaignLocationDocument;
        name: string;
        prizePool?: CampaignPrizePoolDocument;
        sides?: Array<{
            entities: Array<{
                /**
                 * An integer. A numeric string is accepted for compatibility.
                 */
                id: number | string;
                name?: string;
                type: 'character' | 'corporation' | 'alliance';
            }>;
            name?: string;
        }>;
        startTime: string;
        /**
         * An integer. A numeric string is accepted for compatibility.
         */
        visibility?: number | string;
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
        campaign_id: string;
        estimated_killmails: number;
        initial_contribution: string;
        replayed: boolean;
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
        deleted: boolean;
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
        allowed_entities: Array<{
            id: number;
            name?: string;
            type: string;
        }>;
        campaign_id: string;
        /**
         * UTC timestamp with millisecond precision.
         */
        created_at: string;
        creator: {
            character_id: number;
            name: string | null;
        };
        daily: {
            granularity: string;
            rows: Array<{
                isk_destroyed: number;
                isk_lost: number;
                kills: number;
                losses: number;
                period: string;
                side_index: number;
            }>;
        };
        description: string | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        end_time: string | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        last_activity_at: string | null;
        location: {
            constellationIds?: Array<number>;
            regionIds?: Array<number>;
            systemIds?: Array<number>;
        } | null;
        location_details: {
            constellations: Array<{
                id: number;
                name: string;
            }>;
            regions: Array<{
                id: number;
                name: string;
            }>;
            systems: Array<{
                id: number;
                name: string;
            }>;
        };
        mode: string;
        name: string;
        prize_pool: {
            contribution_count: number;
            contributions: Array<{
                amount: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                contributed_at: string;
                contributor_id: number | null;
                contributor_name: string;
                contributor_type: string;
                id: string;
                source: string;
            }>;
            discord_url: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            finalized_at: string | null;
            funded_total: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            funding_closes_at: string | null;
            funding_reference: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_wallet_sync: string | null;
            metric: number;
            metric_label: string;
            payout_percentages: Array<number>;
            projected_lead_percent: number | null;
            results: Array<{
                can_claim: boolean;
                character_id: number;
                character_name: string;
                claimed: boolean;
                metric_value: string;
                paid: boolean;
                payout_amount: number;
                payout_percentage: number;
                rank: number;
                secondary_value: string;
            }>;
            /**
             * UTC timestamp with millisecond precision.
             */
            rules_locked_at: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            settles_at: string | null;
            status: number;
            winner_count: number;
        } | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        processed_through: string | null;
        processing: {
            estimated_killmails: number | null;
            last_duration_ms: number | null;
            last_error: string | null;
            last_killmails: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_started_at: string | null;
            note: string | null;
            paused: boolean;
        };
        sides: Array<{
            entities: Array<{
                entity_id: number;
                entity_type: number;
                isk_destroyed?: number;
                isk_lost?: number;
                kills?: number;
                losses?: number;
                name?: string | null;
            }>;
            isk_destroyed: number;
            isk_lost: number;
            kills: number;
            losses: number;
            name: string;
            palette: string | null;
            side_index: number;
        }>;
        /**
         * UTC timestamp with millisecond precision.
         */
        start_time: string;
        stats: {
            intelBySide: {
                [key: string]: {
                    capitals: Array<{
                        allianceName: string | null;
                        characterId: number;
                        corporationName: string | null;
                        damage: number;
                        died: boolean;
                        name: string | null;
                        shipGroupName: string | null;
                        shipName: string | null;
                        shipTypeId: number | null;
                    }>;
                    capitalsCount: number;
                    fcs: Array<{
                        allianceName: string | null;
                        characterId: number;
                        corporationName: string | null;
                        damage: number;
                        died: boolean;
                        name: string | null;
                        shipGroupName: string | null;
                        shipName: string | null;
                        shipTypeId: number | null;
                    }>;
                    logistics: Array<{
                        allianceName: string | null;
                        characterId: number;
                        corporationName: string | null;
                        damage: number;
                        died: boolean;
                        name: string | null;
                        shipGroupName: string | null;
                        shipName: string | null;
                        shipTypeId: number | null;
                    }>;
                    logisticsCount: number;
                } | null;
            };
            mostValuable: Array<{
                killmailId: number;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                killmailTime: string;
                shipName: string | null;
                shipTypeId: number | null;
                value: number;
                victimCharacterId: number | null;
                victimCharacterName: string | null;
                victimCorporationId: number | null;
                victimCorporationName: string | null;
                victimSide: number | null;
            }>;
            shipClassesBySide: {
                [key: string]: Array<{
                    groupId: number;
                    iskLost: number;
                    losses: number;
                    name: string | null;
                }>;
            };
            shipClassesOverall: Array<{
                groupId: number;
                iskLost: number;
                losses: number;
                name: string | null;
            }>;
            topKillersBySide: {
                [key: string]: Array<{
                    allianceId?: number | null;
                    allianceName?: string | null;
                    allianceTicker?: string | null;
                    characterId: number;
                    corporationId?: number | null;
                    corporationName?: string | null;
                    corporationTicker?: string | null;
                    iskDestroyed?: number;
                    iskLost?: number;
                    kills?: number;
                    losses?: number;
                    name: string | null;
                }>;
            };
            topKillersOverall: Array<{
                allianceId?: number | null;
                allianceName?: string | null;
                allianceTicker?: string | null;
                characterId: number;
                corporationId?: number | null;
                corporationName?: string | null;
                corporationTicker?: string | null;
                iskDestroyed?: number;
                iskLost?: number;
                kills?: number;
                losses?: number;
                name: string | null;
            }>;
            topSystems: Array<{
                iskDestroyed: number;
                kills: number;
                name: string | null;
                regionId: number | null;
                regionName: string | null;
                systemId: number;
            }>;
            topVictimsBySide: {
                [key: string]: Array<{
                    allianceId?: number | null;
                    allianceName?: string | null;
                    allianceTicker?: string | null;
                    characterId: number;
                    corporationId?: number | null;
                    corporationName?: string | null;
                    corporationTicker?: string | null;
                    iskDestroyed?: number;
                    iskLost?: number;
                    kills?: number;
                    losses?: number;
                    name: string | null;
                }>;
            };
            topVictimsOverall: Array<{
                allianceId?: number | null;
                allianceName?: string | null;
                allianceTicker?: string | null;
                characterId: number;
                corporationId?: number | null;
                corporationName?: string | null;
                corporationTicker?: string | null;
                iskDestroyed?: number;
                iskLost?: number;
                kills?: number;
                losses?: number;
                name: string | null;
            }>;
            totals: {
                alliancesInvolved: number;
                charactersInvolved: number;
                corporationsInvolved: number;
                iskDestroyed: number;
                killCount: number;
            };
        } | null;
        /**
         * UTC timestamp with millisecond precision.
         */
        stats_updated_at: string | null;
        status: number;
        visibility: number;
    };
};

export type CampaignDetailResponse = CampaignDetailResponses[keyof CampaignDetailResponses];

export type CampaignUpdateData = {
    body: {
        allowedEntities?: Array<{
            /**
             * An integer. A numeric string is accepted for compatibility.
             */
            id: number | string;
            name?: string;
            type: 'character' | 'corporation' | 'alliance';
        }>;
        archived?: boolean;
        description?: string | null;
        endTime?: string | null;
        name?: string;
        resumeProcessing?: boolean;
        sides?: Array<{
            entities: Array<{
                /**
                 * An integer. A numeric string is accepted for compatibility.
                 */
                id: number | string;
                name?: string;
                type: 'character' | 'corporation' | 'alliance';
            }>;
            name?: string;
        }>;
        /**
         * An integer. A numeric string is accepted for compatibility.
         */
        visibility?: number | string;
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
        recompute?: boolean;
        updated: boolean;
    };
};

export type CampaignUpdateResponse = CampaignUpdateResponses[keyof CampaignUpdateResponses];

export type CampaignKillmailsData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Restrict to one campaign side index. Must match a side the campaign defines.
         */
        side?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
    url: '/campaigns/{id}/killmails';
};

export type CampaignKillmailsResponses = {
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
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
        claimed: boolean;
        rank: number;
    };
};

export type CampaignPrizeClaimResponse = CampaignPrizeClaimResponses[keyof CampaignPrizeClaimResponses];

export type CampaignPrizeContributeData = {
    body: {
        /**
         * A number. A numeric string is accepted for compatibility.
         */
        amount: number | string;
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
        balance: string;
        contributed: string;
        replayed: boolean;
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
        character: {
            alliance_id?: number | null;
            alliance_name?: string | null;
            alliance_ticker?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            birthday?: string | null;
            bloodline_name?: string | null;
            character_id?: number;
            corporation_id?: number | null;
            corporation_name?: string | null;
            corporation_ticker?: string | null;
            custom_description?: string | null;
            custom_description_format?: string | null;
            custom_description_html?: string | null;
            description?: string | null;
            faction_id?: number | null;
            faction_name?: string | null;
            gender?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_active?: string | null;
            name?: string;
            palette?: string | null;
            race_name?: string | null;
            security_status?: number;
            title?: string | null;
            [key: string]: unknown;
        };
        corporationHistory: Array<{
            corporation_id: number;
            corporation_name: string;
            corporation_ticker: string;
            kills: number;
            losses: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            start_date: string;
        }>;
        corporationHistoryQueued: boolean;
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
    } | {
        allianceHistory: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            alliance_ticker: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            start_date: string;
        }>;
        corporation: {
            alliance_id?: number | null;
            alliance_name?: string | null;
            alliance_ticker?: string | null;
            ceo_id?: number | null;
            ceo_name?: string | null;
            corporation_id?: number;
            creator_id?: number | null;
            creator_name?: string | null;
            custom_description?: string | null;
            custom_description_format?: string | null;
            custom_description_html?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            date_founded?: string | null;
            description?: string | null;
            faction_id?: number | null;
            faction_name?: string | null;
            friendly_fire?: boolean | null;
            lp_tax_rate?: number | null;
            member_count?: number;
            name?: string;
            palette?: string | null;
            state?: string | null;
            tax_rate?: number;
            ticker?: string;
            type?: string | null;
            url?: string | null;
            war_eligible?: boolean;
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
    } | {
        alliance: {
            alliance_id?: number;
            corporation_count?: number;
            creator_id?: number | null;
            creator_name?: string | null;
            custom_description?: string | null;
            custom_description_format?: string | null;
            custom_description_html?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            date_founded?: string | null;
            executor_corporation_id?: number | null;
            executor_name?: string | null;
            executor_ticker?: string | null;
            faction_id?: number | null;
            faction_name?: string | null;
            member_count?: number;
            name?: string;
            palette?: string | null;
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
    } | {
        faction: {
            corporation_id?: number | null;
            description?: string | null;
            faction_id?: number;
            militia_corporation_id?: number | null;
            name?: string;
            solar_system_id?: number | null;
            station_count?: number;
            station_system_count?: number;
            [key: string]: unknown;
        };
        recentStats: {
            isk_lost: number;
            losses: number;
        };
        stats: {
            isk_lost: number;
            losses: number;
        };
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
        achievements: Array<{
            achievement_id: string;
            category: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            completed_at: string | null;
            completion_tiers: number;
            current_count: number;
            description: string;
            is_completed: boolean;
            level: number;
            level_thresholds: Array<number>;
            max_level: number;
            name: string;
            next_threshold: number;
            points: number;
            points_modifier: string;
            rarity: string;
            threshold: number;
            type: string;
        }>;
    };
};

export type EntityPageAchievementsCharacterCompatResponse = EntityPageAchievementsCharacterCompatResponses[keyof EntityPageAchievementsCharacterCompatResponses];

export type EntityPageIntelCharacterCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Trailing window in days.
         */
        days?: number;
    };
    url: '/character/{id}/intel';
};

export type EntityPageIntelCharacterCompatResponses = {
    /**
     * OK
     */
    200: {
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
            likelihood?: string;
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
    } | {
        activeMembers: {
            days_30: number;
            days_7: number;
            days_90: number;
        };
        allies: Array<{
            id?: number;
            kills_given?: number;
            kills_taken?: number;
            mutual_kills?: number;
            name?: string;
            shared_enemy_kills?: number;
            total?: number;
        }>;
        census: {
            caps: number;
            corps?: Array<{
                id?: number;
                name?: string;
                total?: number;
                [key: string]: unknown;
            }>;
            droppers: number;
            fcs: number;
            logis: number;
            supers: number;
            total: number;
        };
        enemies: Array<{
            id?: number;
            kills_given?: number;
            kills_taken?: number;
            mutual_kills?: number;
            name?: string;
            shared_enemy_kills?: number;
            total?: number;
        }>;
        huntingGrounds: Array<{
            active_characters: number;
            id: number;
            name: string;
        }>;
        recentDepartures: Array<{
            current_corp?: {
                id: number;
                name: string | null;
            } | null;
            id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            joined_at?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            left_at?: string | null;
            name: string;
            previous_corp?: {
                id: number;
                name: string | null;
            } | null;
        }>;
        recentJoins: Array<{
            current_corp?: {
                id: number;
                name: string | null;
            } | null;
            id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            joined_at?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            left_at?: string | null;
            name: string;
            previous_corp?: {
                id: number;
                name: string | null;
            } | null;
        }>;
    };
};

export type EntityPageIntelCharacterCompatResponse = EntityPageIntelCharacterCompatResponses[keyof EntityPageIntelCharacterCompatResponses];

export type EntityPageStatsCharacterCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Trailing window in days. 0 covers the whole record.
         */
        days?: number;
    };
    url: '/character/{id}/stats';
};

export type EntityPageStatsCharacterCompatResponses = {
    /**
     * OK
     */
    200: {
        activity?: {
            kills: Array<Array<number>>;
            losses: Array<Array<number>>;
        };
        diesToAlliances: Array<{
            count: number;
            id: number;
            isk_value: number;
            name: string;
        }>;
        diesToCorporations: Array<{
            count: number;
            id: number;
            isk_value: number;
            name: string;
        }>;
        fliesWithAlliances?: Array<{
            count: number;
            id: number;
            isk_value: number;
            name: string;
        }>;
        fliesWithCorporations?: Array<{
            count: number;
            id: number;
            isk_value: number;
            name: string;
        }>;
        heatMap?: {
            [key: string]: number;
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
        topShipsLost: Array<{
            count: number;
            ship_name: string;
            ship_type_id: number;
        }>;
        topShipsUsed: Array<{
            count: number;
            ship_name: string;
            ship_type_id: number;
        }>;
    };
};

export type EntityPageStatsCharacterCompatResponse = EntityPageStatsCharacterCompatResponses[keyof EntityPageStatsCharacterCompatResponses];

export type EntityTopCharacterCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Which half of the leaderboard set to build. `right` also accepts `days=alltime`.
         */
        slice?: 'left' | 'right';
        /**
         * Window in days, between 1/24 and 365. Send `alltime` with `slice=right` for the unbounded set. Default 7.
         */
        days?: string;
    };
    url: '/character/{id}/top';
};

export type EntityTopCharacterCompatResponses = {
    /**
     * OK
     */
    200: {
        achievementPoints?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        charactersByIsk?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        charactersByKills?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        charactersByPoints?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        constellations?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        corporationsByKills?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        killedAlliances?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        killedByAlliances?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        killedByCorporations?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        killedCorporations?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        recentMembers?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        regions?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        shipsUsed?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        soloKillers?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        systems?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
    };
};

export type EntityTopCharacterCompatResponse = EntityTopCharacterCompatResponses[keyof EntityTopCharacterCompatResponses];

export type CharactersData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Case-insensitive name prefix to match.
         */
        name?: string;
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
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
        character_ids: Array<number>;
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

export type CharacterIntelBatchData = {
    body: {
        /**
         * Characters to inspect, at most 100 unique IDs.
         */
        character_ids: Array<number>;
        /**
         * Look-back window in days, from 1 through 90.
         */
        days?: number;
    };
    path?: never;
    query?: never;
    url: '/characters/intel';
};

export type CharacterIntelBatchResponses = {
    /**
     * OK
     */
    200: CharacterIntelBatchResponse;
};

export type CharacterIntelBatchResponse2 = CharacterIntelBatchResponses[keyof CharacterIntelBatchResponses];

export type CharactersBatchStatsData = {
    body: {
        /**
         * Start of the window, for type=range.
         */
        from?: string;
        /**
         * Entity IDs to resolve, at most 100 per request.
         */
        ids: Array<number>;
        /**
         * End of the window, for type=range.
         */
        to?: string;
        /**
         * Aggregation period. Falls back to the type query parameter, then alltime.
         */
        type?: 'alltime' | 'weekly' | 'range';
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
    query?: {
        /**
         * Size of the trailing window, in days.
         */
        days?: number;
    };
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
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
        /**
         * Descending cursor, walking newest to oldest. Pass the previous response's pagination cursor to fetch the next page. Mutually exclusive with `after`, which it overrides.
         */
        before?: number;
    };
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
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
        /**
         * Descending cursor, walking newest to oldest. Pass the previous response's pagination cursor to fetch the next page. Mutually exclusive with `after`, which it overrides.
         */
        before?: number;
    };
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
    query?: {
        /**
         * Reporting period.
         */
        type?: 'alltime' | 'weekly' | 'range';
        /**
         * Start date as YYYY-MM-DD. Required when `type` is `range`.
         */
        from?: string;
        /**
         * End date as YYYY-MM-DD. Required when `type` is `range`.
         */
        to?: string;
    };
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
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Identifier cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        cursor?: number;
        /**
         * Numeric target kind: 1 killmail, 2 character, 3 corporation, 4 alliance, 5 system, 6 page, 7 battle, 8 fit, 9 blog, 10 campaign.
         */
        target_type?: number;
        /**
         * Restrict to one author.
         */
        character_id?: number;
        /**
         * Restrict to one author corporation.
         */
        corporation_id?: number;
        /**
         * Restrict to one author alliance.
         */
        alliance_id?: number;
        /**
         * Body search. Applied once the text is at least two characters.
         */
        q?: string;
    };
    url: '/comments';
};

export type CommentsFeedResponses = {
    /**
     * OK
     */
    200: {
        comments: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            body_html: string;
            body_md: string;
            character_id: number;
            character_name: string;
            corporation_id: number;
            corporation_name: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            deleted_at: string | null;
            deleted_by: number | null;
            depth: number;
            domain_id: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            edited_at: string | null;
            flagged: boolean;
            id: number;
            moderation_status: number;
            parent_id: number | null;
            reply_count?: number;
            reports_count: number;
            root_id: number | null;
            target_id: number;
            target_slug: string | null;
            target_type: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
            visibility: number;
        }>;
        nextCursor: number | null;
    };
};

export type CommentsFeedResponse = CommentsFeedResponses[keyof CommentsFeedResponses];

export type CommentsCreateData = {
    body: {
        body_md: string;
        /**
         * An integer. A numeric string is accepted for compatibility.
         */
        parent_id?: number | string | null;
        /**
         * An integer. A numeric string is accepted for compatibility.
         */
        target_id: number | string;
        target_slug?: string;
        target_type: number;
    };
    path?: never;
    query?: never;
    url: '/comments';
};

export type CommentsCreateResponses = {
    /**
     * OK
     */
    200: {
        comment: {
            alliance_id: number | null;
            alliance_name: string | null;
            body_html: string;
            body_md: string;
            character_id: number;
            character_name: string;
            corporation_id: number;
            corporation_name: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            deleted_at: string | null;
            deleted_by: number | null;
            depth: number;
            domain_id: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            edited_at: string | null;
            flagged: boolean;
            id: number;
            moderation_status: number;
            parent_id: number | null;
            reply_count?: number;
            reports_count: number;
            root_id: number | null;
            target_id: number;
            target_slug: string | null;
            target_type: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
            visibility: number;
        };
    };
};

export type CommentsCreateResponse = CommentsCreateResponses[keyof CommentsCreateResponses];

export type CommentsKlipySearchData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Search text.
         */
        q?: string;
        /**
         * Page number, counted from 1.
         */
        page?: number;
        /**
         * Results per page.
         */
        per_page?: number;
    };
    url: '/comments/klipy/search';
};

export type CommentsKlipySearchResponses = {
    /**
     * OK
     */
    200: {
        has_next: boolean;
        items: Array<{
            [key: string]: unknown;
        }>;
        page: number;
        per_page: number;
    };
};

export type CommentsKlipySearchResponse = CommentsKlipySearchResponses[keyof CommentsKlipySearchResponses];

export type CommentsKlipyTrendingData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Search text.
         */
        q?: string;
        /**
         * Page number, counted from 1.
         */
        page?: number;
        /**
         * Results per page.
         */
        per_page?: number;
    };
    url: '/comments/klipy/trending';
};

export type CommentsKlipyTrendingResponses = {
    /**
     * OK
     */
    200: {
        has_next: boolean;
        items: Array<{
            [key: string]: unknown;
        }>;
        page: number;
        per_page: number;
    };
};

export type CommentsKlipyTrendingResponse = CommentsKlipyTrendingResponses[keyof CommentsKlipyTrendingResponses];

export type CommentsPreviewData = {
    body: {
        body_md: string;
    };
    path?: never;
    query?: never;
    url: '/comments/preview';
};

export type CommentsPreviewResponses = {
    /**
     * OK
     */
    200: {
        error?: string;
        html: string;
    };
};

export type CommentsPreviewResponse = CommentsPreviewResponses[keyof CommentsPreviewResponses];

export type CommentsThreadData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Identifier cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        cursor?: number;
        /**
         * Numeric target kind: 1 killmail, 2 character, 3 corporation, 4 alliance, 5 system, 6 page, 7 battle, 8 fit, 9 blog, 10 campaign.
         */
        target_type?: number;
        /**
         * Numeric ID of the commented object.
         */
        target_id?: number;
        /**
         * Slug of the commented object.
         */
        target_slug?: string;
    };
    url: '/comments/thread';
};

export type CommentsThreadResponses = {
    /**
     * OK
     */
    200: {
        nextCursor: number | null;
        replies: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            body_html: string;
            body_md: string;
            character_id: number;
            character_name: string;
            corporation_id: number;
            corporation_name: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            deleted_at: string | null;
            deleted_by: number | null;
            depth: number;
            domain_id: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            edited_at: string | null;
            flagged: boolean;
            id: number;
            moderation_status: number;
            parent_id: number | null;
            reply_count?: number;
            reports_count: number;
            root_id: number | null;
            target_id: number;
            target_slug: string | null;
            target_type: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
            visibility: number;
        }>;
        repliesTruncated: boolean;
        roots: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            body_html: string;
            body_md: string;
            character_id: number;
            character_name: string;
            corporation_id: number;
            corporation_name: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            deleted_at: string | null;
            deleted_by: number | null;
            depth: number;
            domain_id: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            edited_at: string | null;
            flagged: boolean;
            id: number;
            moderation_status: number;
            parent_id: number | null;
            reply_count?: number;
            reports_count: number;
            root_id: number | null;
            target_id: number;
            target_slug: string | null;
            target_type: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
            visibility: number;
        }>;
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
        ok: boolean;
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
        comment: {
            alliance_id: number | null;
            alliance_name: string | null;
            body_html: string;
            body_md: string;
            character_id: number;
            character_name: string;
            corporation_id: number;
            corporation_name: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            deleted_at: string | null;
            deleted_by: number | null;
            depth: number;
            domain_id: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            edited_at: string | null;
            flagged: boolean;
            id: number;
            moderation_status: number;
            parent_id: number | null;
            reply_count?: number;
            reports_count: number;
            root_id: number | null;
            target_id: number;
            target_slug: string | null;
            target_type: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
            visibility: number;
        };
    };
};

export type CommentDetailResponse = CommentDetailResponses[keyof CommentDetailResponses];

export type CommentEditData = {
    body: {
        body_md: string;
    };
    path?: never;
    query?: never;
    url: '/comments/{id}';
};

export type CommentEditResponses = {
    /**
     * OK
     */
    200: {
        comment: {
            alliance_id: number | null;
            alliance_name: string | null;
            body_html: string;
            body_md: string;
            character_id: number;
            character_name: string;
            corporation_id: number;
            corporation_name: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            deleted_at: string | null;
            deleted_by: number | null;
            depth: number;
            domain_id: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            edited_at: string | null;
            flagged: boolean;
            id: number;
            moderation_status: number;
            parent_id: number | null;
            reply_count?: number;
            reports_count: number;
            root_id: number | null;
            target_id: number;
            target_slug: string | null;
            target_type: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
            visibility: number;
        };
    };
};

export type CommentEditResponse = CommentEditResponses[keyof CommentEditResponses];

export type CommentReportData = {
    body: {
        message?: string | null;
        reason: 'spam' | 'harassment' | 'nsfw' | 'offtopic' | 'other';
    };
    path?: never;
    query?: never;
    url: '/comments/{id}/report';
};

export type CommentReportResponses = {
    /**
     * OK
     */
    200: {
        flagged: boolean;
        ok: boolean;
        reports_count: number;
    };
};

export type CommentReportResponse = CommentReportResponses[keyof CommentReportResponses];

export type ConflictBattlesData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Page number, counted from 1.
         */
        page?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Restrict to battles that started in this year.
         */
        year?: number;
        /**
         * Minimum killmail count in the battle.
         */
        minKills?: number;
        /**
         * Minimum ISK destroyed in the battle.
         */
        minIsk?: number;
        /**
         * Restrict to battles inside the custom domain's own scope.
         */
        custom?: boolean;
        /**
         * Restrict to one alliance.
         */
        allianceId?: number;
        /**
         * Restrict to one corporation.
         */
        corporationId?: number;
        /**
         * Restrict to one character.
         */
        characterId?: number;
        /**
         * Restrict to one constellation.
         */
        constellationId?: number;
        /**
         * Restrict to one region.
         */
        regionId?: number;
        /**
         * Restrict to one solar system.
         */
        systemId?: number;
    };
    url: '/conflicts/battles';
};

export type ConflictBattlesResponses = {
    /**
     * OK
     */
    200: {
        battles: Array<{
            battle_id?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            end_time?: string;
            is_custom?: boolean;
            kill_count?: number;
            region_id?: number | null;
            region_name?: string | null;
            solar_system_id?: number;
            solar_system_name?: string | null;
            solar_system_security?: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            start_time?: string;
            total_value?: number;
            [key: string]: unknown;
        }>;
        limit: number;
        page: number;
        years: Array<{
            count: number;
            year: number;
        }>;
    };
};

export type ConflictBattlesResponse = ConflictBattlesResponses[keyof ConflictBattlesResponses];

export type ConflictWarsData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Page number, counted from 1.
         */
        page?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Only wars that have not started yet.
         */
        upcoming?: boolean;
        /**
         * Only wars that have finished.
         */
        finished?: boolean;
        /**
         * Only wars that are currently running.
         */
        ongoing?: boolean;
        /**
         * Only wars both parties agreed to.
         */
        mutual?: boolean;
        /**
         * Only wars with recorded activity.
         */
        hasActivity?: boolean;
        /**
         * Only wars with at least one killmail.
         */
        hasKills?: boolean;
        /**
         * Only wars that have allies.
         */
        hasAllies?: boolean;
        /**
         * Ordering. Ignored while `upcoming` is set, which always orders by start date.
         */
        sort?: 'kills' | 'isk';
        /**
         * Restrict to one alliance.
         */
        allianceId?: number;
        /**
         * Restrict to one corporation.
         */
        corporationId?: number;
        /**
         * Restrict to one character.
         */
        characterId?: number;
    };
    url: '/conflicts/wars';
};

export type ConflictWarsResponses = {
    /**
     * OK
     */
    200: {
        limit: number;
        page: number;
        wars: Array<{
            aggressor?: {
                id: number;
                name?: string;
                ticker?: string;
                type: string;
            };
            /**
             * UTC timestamp with millisecond precision.
             */
            declared?: string;
            defender?: {
                id: number;
                name?: string;
                ticker?: string;
                type: string;
            };
            /**
             * UTC timestamp with millisecond precision.
             */
            finished?: string | null;
            mutual?: boolean;
            open_for_allies?: boolean;
            /**
             * UTC timestamp with millisecond precision.
             */
            retracted?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            started?: string | null;
            war_id?: number;
            [key: string]: unknown;
        }>;
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
        constellation: {
            constellation_id: number;
            constellation_name: string;
            faction_id: number | null;
            region_id: number;
            region_name: string | null;
        };
        sovDistribution: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            faction_id: number | null;
            faction_name: string | null;
            system_count: number;
        }>;
        stats: {
            kills: number;
            npc_kills: number;
            pod_kills: number;
            total_value: number;
        };
        systems: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            faction_id: number | null;
            faction_name: string | null;
            security: number;
            solar_system_id: number;
            system_name: string;
        }>;
    };
};

export type ConstellationCompatResponse = ConstellationCompatResponses[keyof ConstellationCompatResponses];

export type ConstellationKilllistCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
        /**
         * Page number for offset paging. Leave at 0 to page by cursor.
         */
        page?: number;
    };
    url: '/constellation/{id}/killlist';
};

export type ConstellationKilllistCompatResponses = {
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
        totalPages?: number;
    };
};

export type ConstellationKilllistCompatResponse = ConstellationKilllistCompatResponses[keyof ConstellationKilllistCompatResponses];

export type ConstellationMostValuableCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Restrict the losses to one category of hull.
         */
        dataType?: 'most_valuable_kills' | 'most_valuable_ships' | 'most_valuable_structures';
        /**
         * Size of the trailing window, in days.
         */
        days?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/constellation/{id}/most-valuable';
};

export type ConstellationMostValuableCompatResponses = {
    /**
     * OK
     */
    200: {
        entries: Array<{
            killmail_hash: string;
            killmail_id: number;
            ship_name: string;
            ship_type_id: number;
            total_value: number;
            victim_alliance_name: string | null;
            victim_character_id: number | null;
            victim_character_name: string | null;
            victim_corporation_name: string | null;
        }>;
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
        character: {
            alliance_id?: number | null;
            alliance_name?: string | null;
            alliance_ticker?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            birthday?: string | null;
            bloodline_name?: string | null;
            character_id?: number;
            corporation_id?: number | null;
            corporation_name?: string | null;
            corporation_ticker?: string | null;
            custom_description?: string | null;
            custom_description_format?: string | null;
            custom_description_html?: string | null;
            description?: string | null;
            faction_id?: number | null;
            faction_name?: string | null;
            gender?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_active?: string | null;
            name?: string;
            palette?: string | null;
            race_name?: string | null;
            security_status?: number;
            title?: string | null;
            [key: string]: unknown;
        };
        corporationHistory: Array<{
            corporation_id: number;
            corporation_name: string;
            corporation_ticker: string;
            kills: number;
            losses: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            start_date: string;
        }>;
        corporationHistoryQueued: boolean;
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
    } | {
        allianceHistory: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            alliance_ticker: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            start_date: string;
        }>;
        corporation: {
            alliance_id?: number | null;
            alliance_name?: string | null;
            alliance_ticker?: string | null;
            ceo_id?: number | null;
            ceo_name?: string | null;
            corporation_id?: number;
            creator_id?: number | null;
            creator_name?: string | null;
            custom_description?: string | null;
            custom_description_format?: string | null;
            custom_description_html?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            date_founded?: string | null;
            description?: string | null;
            faction_id?: number | null;
            faction_name?: string | null;
            friendly_fire?: boolean | null;
            lp_tax_rate?: number | null;
            member_count?: number;
            name?: string;
            palette?: string | null;
            state?: string | null;
            tax_rate?: number;
            ticker?: string;
            type?: string | null;
            url?: string | null;
            war_eligible?: boolean;
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
    } | {
        alliance: {
            alliance_id?: number;
            corporation_count?: number;
            creator_id?: number | null;
            creator_name?: string | null;
            custom_description?: string | null;
            custom_description_format?: string | null;
            custom_description_html?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            date_founded?: string | null;
            executor_corporation_id?: number | null;
            executor_name?: string | null;
            executor_ticker?: string | null;
            faction_id?: number | null;
            faction_name?: string | null;
            member_count?: number;
            name?: string;
            palette?: string | null;
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
    } | {
        faction: {
            corporation_id?: number | null;
            description?: string | null;
            faction_id?: number;
            militia_corporation_id?: number | null;
            name?: string;
            solar_system_id?: number | null;
            station_count?: number;
            station_system_count?: number;
            [key: string]: unknown;
        };
        recentStats: {
            isk_lost: number;
            losses: number;
        };
        stats: {
            isk_lost: number;
            losses: number;
        };
    };
};

export type EntityPageDetailCorporationCompatResponse = EntityPageDetailCorporationCompatResponses[keyof EntityPageDetailCorporationCompatResponses];

export type EntityPageIntelCorporationCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Trailing window in days.
         */
        days?: number;
    };
    url: '/corporation/{id}/intel';
};

export type EntityPageIntelCorporationCompatResponses = {
    /**
     * OK
     */
    200: {
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
            likelihood?: string;
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
    } | {
        activeMembers: {
            days_30: number;
            days_7: number;
            days_90: number;
        };
        allies: Array<{
            id?: number;
            kills_given?: number;
            kills_taken?: number;
            mutual_kills?: number;
            name?: string;
            shared_enemy_kills?: number;
            total?: number;
        }>;
        census: {
            caps: number;
            corps?: Array<{
                id?: number;
                name?: string;
                total?: number;
                [key: string]: unknown;
            }>;
            droppers: number;
            fcs: number;
            logis: number;
            supers: number;
            total: number;
        };
        enemies: Array<{
            id?: number;
            kills_given?: number;
            kills_taken?: number;
            mutual_kills?: number;
            name?: string;
            shared_enemy_kills?: number;
            total?: number;
        }>;
        huntingGrounds: Array<{
            active_characters: number;
            id: number;
            name: string;
        }>;
        recentDepartures: Array<{
            current_corp?: {
                id: number;
                name: string | null;
            } | null;
            id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            joined_at?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            left_at?: string | null;
            name: string;
            previous_corp?: {
                id: number;
                name: string | null;
            } | null;
        }>;
        recentJoins: Array<{
            current_corp?: {
                id: number;
                name: string | null;
            } | null;
            id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            joined_at?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            left_at?: string | null;
            name: string;
            previous_corp?: {
                id: number;
                name: string | null;
            } | null;
        }>;
    };
};

export type EntityPageIntelCorporationCompatResponse = EntityPageIntelCorporationCompatResponses[keyof EntityPageIntelCorporationCompatResponses];

export type EntityPageMembersCorporationCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Ordering for the member rows.
         */
        sort?: 'name' | 'last_active' | 'security_status';
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Page number, counted from 1.
         */
        page?: number;
        /**
         * Restrict an alliance's members to one corporation.
         */
        corporation_id?: number;
        /**
         * Only members active within this many days. 0 disables the filter.
         */
        activity?: number;
    };
    url: '/corporation/{id}/members';
};

export type EntityPageMembersCorporationCompatResponses = {
    /**
     * OK
     */
    200: {
        limit: number;
        members: Array<{
            character_id: number;
            corporation_id?: number | null;
            is_capital_pilot: boolean;
            is_fc: boolean;
            is_logi: boolean;
            kills_90d: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_active: string | null;
            losses_90d: number;
            name: string;
            security_status: number;
        }>;
        page: number;
        total: number;
    };
};

export type EntityPageMembersCorporationCompatResponse = EntityPageMembersCorporationCompatResponses[keyof EntityPageMembersCorporationCompatResponses];

export type EntityPageStatsCorporationCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Trailing window in days. 0 covers the whole record.
         */
        days?: number;
    };
    url: '/corporation/{id}/stats';
};

export type EntityPageStatsCorporationCompatResponses = {
    /**
     * OK
     */
    200: {
        activity?: {
            kills: Array<Array<number>>;
            losses: Array<Array<number>>;
        };
        diesToAlliances: Array<{
            count: number;
            id: number;
            isk_value: number;
            name: string;
        }>;
        diesToCorporations: Array<{
            count: number;
            id: number;
            isk_value: number;
            name: string;
        }>;
        fliesWithAlliances?: Array<{
            count: number;
            id: number;
            isk_value: number;
            name: string;
        }>;
        fliesWithCorporations?: Array<{
            count: number;
            id: number;
            isk_value: number;
            name: string;
        }>;
        heatMap?: {
            [key: string]: number;
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
        topShipsLost: Array<{
            count: number;
            ship_name: string;
            ship_type_id: number;
        }>;
        topShipsUsed: Array<{
            count: number;
            ship_name: string;
            ship_type_id: number;
        }>;
    };
};

export type EntityPageStatsCorporationCompatResponse = EntityPageStatsCorporationCompatResponses[keyof EntityPageStatsCorporationCompatResponses];

export type EntityTopCorporationCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Which half of the leaderboard set to build. `right` also accepts `days=alltime`.
         */
        slice?: 'left' | 'right';
        /**
         * Window in days, between 1/24 and 365. Send `alltime` with `slice=right` for the unbounded set. Default 7.
         */
        days?: string;
    };
    url: '/corporation/{id}/top';
};

export type EntityTopCorporationCompatResponses = {
    /**
     * OK
     */
    200: {
        achievementPoints?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        charactersByIsk?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        charactersByKills?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        charactersByPoints?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        constellations?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        corporationsByKills?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        killedAlliances?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        killedByAlliances?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        killedByCorporations?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        killedCorporations?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        recentMembers?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        regions?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        shipsUsed?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        soloKillers?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
        systems?: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
        }>;
    };
};

export type EntityTopCorporationCompatResponse = EntityTopCorporationCompatResponses[keyof EntityTopCorporationCompatResponses];

export type CorporationsData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Case-insensitive name prefix to match.
         */
        name?: string;
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
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
         * Start of the window, for type=range.
         */
        from?: string;
        /**
         * Entity IDs to resolve, at most 100 per request.
         */
        ids: Array<number>;
        /**
         * End of the window, for type=range.
         */
        to?: string;
        /**
         * Aggregation period. Falls back to the type query parameter, then alltime.
         */
        type?: 'alltime' | 'weekly' | 'range';
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
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
        /**
         * Descending cursor, walking newest to oldest. Pass the previous response's pagination cursor to fetch the next page. Mutually exclusive with `after`, which it overrides.
         */
        before?: number;
    };
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
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
        /**
         * Descending cursor, walking newest to oldest. Pass the previous response's pagination cursor to fetch the next page. Mutually exclusive with `after`, which it overrides.
         */
        before?: number;
    };
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
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
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
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
    url: '/custom/constellation/{id}/killlist';
};

export type DomainConstellationKilllistResponses = {
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
        totalPages?: number;
    };
};

export type DomainConstellationKilllistResponse = DomainConstellationKilllistResponses[keyof DomainConstellationKilllistResponses];

export type DomainKilllistData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Killmail category: space, ship class, tech level, or value band.
         */
        type?: '10b' | '5b' | 'abyssal' | 'battlecruisers' | 'battleships' | 'big' | 'capitals' | 'citadels' | 'cruisers' | 'destroyers' | 'faction' | 'freighters' | 'frigates' | 'highsec' | 'jove' | 'latest' | 'lowsec' | 'npc' | 'nullsec' | 'pochven' | 'solo' | 'supercarriers' | 't1' | 't2' | 't3' | 'titans' | 'wspace';
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
    url: '/custom/killlist';
};

export type DomainKilllistResponses = {
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
        totalPages?: number;
    };
};

export type DomainKilllistResponse = DomainKilllistResponses[keyof DomainKilllistResponses];

export type DomainKillsMostValuableData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Killmail category: space, ship class, tech level, or value band.
         */
        type?: '10b' | '5b' | 'abyssal' | 'battlecruisers' | 'battleships' | 'big' | 'capitals' | 'citadels' | 'cruisers' | 'destroyers' | 'faction' | 'freighters' | 'frigates' | 'highsec' | 'jove' | 'latest' | 'lowsec' | 'npc' | 'nullsec' | 'pochven' | 'solo' | 'supercarriers' | 't1' | 't2' | 't3' | 'titans' | 'wspace';
        /**
         * Size of the trailing window, in days.
         */
        days?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/custom/kills/most-valuable';
};

export type DomainKillsMostValuableResponses = {
    /**
     * OK
     */
    200: {
        entries: Array<{
            killmail_hash: string;
            killmail_id: number;
            ship_name: string;
            ship_type_id: number;
            total_value: number;
            victim_alliance_name: string | null;
            victim_character_id: number | null;
            victim_character_name: string | null;
            victim_corporation_name: string | null;
        }>;
    };
};

export type DomainKillsMostValuableResponse = DomainKillsMostValuableResponses[keyof DomainKillsMostValuableResponses];

export type DomainKillsTopData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Killmail category: space, ship class, tech level, or value band.
         */
        type?: '10b' | '5b' | 'abyssal' | 'battlecruisers' | 'battleships' | 'big' | 'capitals' | 'citadels' | 'cruisers' | 'destroyers' | 'faction' | 'freighters' | 'frigates' | 'highsec' | 'jove' | 'latest' | 'lowsec' | 'npc' | 'nullsec' | 'pochven' | 'solo' | 'supercarriers' | 't1' | 't2' | 't3' | 'titans' | 'wspace';
        /**
         * Which leaderboard to build.
         */
        dataType?: 'characters' | 'corporations' | 'alliances' | 'ships' | 'systems' | 'regions';
        /**
         * Size of the trailing window, in days.
         */
        days?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/custom/kills/top';
};

export type DomainKillsTopResponses = {
    /**
     * OK
     */
    200: {
        entries: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
            region_id?: number | null;
            type: string;
        }>;
    };
};

export type DomainKillsTopResponse = DomainKillsTopResponses[keyof DomainKillsTopResponses];

export type DomainRegionKilllistData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
    url: '/custom/region/{id}/killlist';
};

export type DomainRegionKilllistResponses = {
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
        totalPages?: number;
    };
};

export type DomainRegionKilllistResponse = DomainRegionKilllistResponses[keyof DomainRegionKilllistResponses];

export type DomainStatisticsData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Which leaderboard to build.
         */
        dataType?: 'characters' | 'corporations' | 'alliances' | 'ships' | 'systems' | 'regions';
        /**
         * Size of the trailing window, in days.
         */
        days?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/custom/stats';
};

export type DomainStatisticsResponses = {
    /**
     * OK
     */
    200: {
        entries: Array<{
            killmail_hash: string;
            killmail_id: number;
            ship_name: string;
            ship_type_id: number;
            total_value: number;
            victim_alliance_name: string | null;
            victim_character_id: number | null;
            victim_character_name: string | null;
            victim_corporation_name: string | null;
        } | {
            count?: number;
            id?: number;
            isk_destroyed?: number;
            isk_lost?: number;
            kills?: number;
            losses?: number;
            name?: string;
            palette?: string | null;
            type?: string;
            value?: number;
            [key: string]: unknown;
        }>;
    };
};

export type DomainStatisticsResponse = DomainStatisticsResponses[keyof DomainStatisticsResponses];

export type DomainSystemKilllistData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
    url: '/custom/system/{id}/killlist';
};

export type DomainSystemKilllistResponses = {
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
        totalPages?: number;
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
    200: Blob | File;
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
    200: Blob | File;
};

export type DomainBackgroundResponse = DomainBackgroundResponses[keyof DomainBackgroundResponses];

export type DomainAssetPreviewData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Signed preview token for an unapproved asset.
         */
        token?: string;
    };
    url: '/domains/preview/{assetId}';
};

export type DomainAssetPreviewResponses = {
    /**
     * OK
     */
    200: Blob | File;
};

export type DomainAssetPreviewResponse = DomainAssetPreviewResponses[keyof DomainAssetPreviewResponses];

export type EntityResolveData = {
    body?: never;
    path?: never;
    query: {
        /**
         * Entity kind to resolve. Required.
         */
        type: 'character' | 'corporation' | 'alliance' | 'faction';
        /**
         * Entity ID to resolve. Required.
         */
        id: number;
    };
    url: '/entities/resolve';
};

export type EntityResolveResponses = {
    /**
     * OK
     */
    200: {
        id: number;
        name: string;
        type: string;
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
        character: {
            alliance_id?: number | null;
            alliance_name?: string | null;
            alliance_ticker?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            birthday?: string | null;
            bloodline_name?: string | null;
            character_id?: number;
            corporation_id?: number | null;
            corporation_name?: string | null;
            corporation_ticker?: string | null;
            custom_description?: string | null;
            custom_description_format?: string | null;
            custom_description_html?: string | null;
            description?: string | null;
            faction_id?: number | null;
            faction_name?: string | null;
            gender?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_active?: string | null;
            name?: string;
            palette?: string | null;
            race_name?: string | null;
            security_status?: number;
            title?: string | null;
            [key: string]: unknown;
        };
        corporationHistory: Array<{
            corporation_id: number;
            corporation_name: string;
            corporation_ticker: string;
            kills: number;
            losses: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            start_date: string;
        }>;
        corporationHistoryQueued: boolean;
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
    } | {
        allianceHistory: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            alliance_ticker: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            start_date: string;
        }>;
        corporation: {
            alliance_id?: number | null;
            alliance_name?: string | null;
            alliance_ticker?: string | null;
            ceo_id?: number | null;
            ceo_name?: string | null;
            corporation_id?: number;
            creator_id?: number | null;
            creator_name?: string | null;
            custom_description?: string | null;
            custom_description_format?: string | null;
            custom_description_html?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            date_founded?: string | null;
            description?: string | null;
            faction_id?: number | null;
            faction_name?: string | null;
            friendly_fire?: boolean | null;
            lp_tax_rate?: number | null;
            member_count?: number;
            name?: string;
            palette?: string | null;
            state?: string | null;
            tax_rate?: number;
            ticker?: string;
            type?: string | null;
            url?: string | null;
            war_eligible?: boolean;
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
    } | {
        alliance: {
            alliance_id?: number;
            corporation_count?: number;
            creator_id?: number | null;
            creator_name?: string | null;
            custom_description?: string | null;
            custom_description_format?: string | null;
            custom_description_html?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            date_founded?: string | null;
            executor_corporation_id?: number | null;
            executor_name?: string | null;
            executor_ticker?: string | null;
            faction_id?: number | null;
            faction_name?: string | null;
            member_count?: number;
            name?: string;
            palette?: string | null;
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
    } | {
        faction: {
            corporation_id?: number | null;
            description?: string | null;
            faction_id?: number;
            militia_corporation_id?: number | null;
            name?: string;
            solar_system_id?: number | null;
            station_count?: number;
            station_system_count?: number;
            [key: string]: unknown;
        };
        recentStats: {
            isk_lost: number;
            losses: number;
        };
        stats: {
            isk_lost: number;
            losses: number;
        };
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
        achievements: Array<{
            achievement_id: string;
            category: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            completed_at: string | null;
            completion_tiers: number;
            current_count: number;
            description: string;
            is_completed: boolean;
            level: number;
            level_thresholds: Array<number>;
            max_level: number;
            name: string;
            next_threshold: number;
            points: number;
            points_modifier: string;
            rarity: string;
            threshold: number;
            type: string;
        }>;
    };
};

export type EntityPageAchievementsResponse = EntityPageAchievementsResponses[keyof EntityPageAchievementsResponses];

export type EntityPageCorporationsData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Ordering for the corporation rows.
         */
        sort?: 'member_count' | 'name';
        /**
         * Sort direction.
         */
        dir?: 'asc' | 'desc';
    };
    url: '/entities/{type}/{id}/corporations';
};

export type EntityPageCorporationsResponses = {
    /**
     * OK
     */
    200: {
        corporations: Array<{
            corporation_id: number;
            member_count: number;
            name: string;
            palette: string | null;
            ticker: string;
        }>;
        total: number;
    };
};

export type EntityPageCorporationsResponse = EntityPageCorporationsResponses[keyof EntityPageCorporationsResponses];

export type EntityPageIntelData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Trailing window in days.
         */
        days?: number;
    };
    url: '/entities/{type}/{id}/intel';
};

export type EntityPageIntelResponses = {
    /**
     * OK
     */
    200: {
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
            likelihood?: string;
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
    } | {
        activeMembers: {
            days_30: number;
            days_7: number;
            days_90: number;
        };
        allies: Array<{
            id?: number;
            kills_given?: number;
            kills_taken?: number;
            mutual_kills?: number;
            name?: string;
            shared_enemy_kills?: number;
            total?: number;
        }>;
        census: {
            caps: number;
            corps?: Array<{
                id?: number;
                name?: string;
                total?: number;
                [key: string]: unknown;
            }>;
            droppers: number;
            fcs: number;
            logis: number;
            supers: number;
            total: number;
        };
        enemies: Array<{
            id?: number;
            kills_given?: number;
            kills_taken?: number;
            mutual_kills?: number;
            name?: string;
            shared_enemy_kills?: number;
            total?: number;
        }>;
        huntingGrounds: Array<{
            active_characters: number;
            id: number;
            name: string;
        }>;
        recentDepartures: Array<{
            current_corp?: {
                id: number;
                name: string | null;
            } | null;
            id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            joined_at?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            left_at?: string | null;
            name: string;
            previous_corp?: {
                id: number;
                name: string | null;
            } | null;
        }>;
        recentJoins: Array<{
            current_corp?: {
                id: number;
                name: string | null;
            } | null;
            id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            joined_at?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            left_at?: string | null;
            name: string;
            previous_corp?: {
                id: number;
                name: string | null;
            } | null;
        }>;
    };
};

export type EntityPageIntelResponse = EntityPageIntelResponses[keyof EntityPageIntelResponses];

export type EntityPageKilllistData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Whose killmails to return.
         */
        role?: 'kills' | 'losses' | 'combined';
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
        /**
         * Page number for offset paging. Leave at 0 to page by cursor.
         */
        page?: number;
        /**
         * Restrict to one victim ship group ID.
         */
        ship_group?: number;
    };
    url: '/entities/{type}/{id}/killlist';
};

export type EntityPageKilllistResponses = {
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
        totalPages?: number;
    };
};

export type EntityPageKilllistResponse = EntityPageKilllistResponses[keyof EntityPageKilllistResponses];

export type EntityPageMembersData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Ordering for the member rows.
         */
        sort?: 'name' | 'last_active' | 'security_status';
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Page number, counted from 1.
         */
        page?: number;
        /**
         * Restrict an alliance's members to one corporation.
         */
        corporation_id?: number;
        /**
         * Only members active within this many days. 0 disables the filter.
         */
        activity?: number;
    };
    url: '/entities/{type}/{id}/members';
};

export type EntityPageMembersResponses = {
    /**
     * OK
     */
    200: {
        limit: number;
        members: Array<{
            character_id: number;
            corporation_id?: number | null;
            is_capital_pilot: boolean;
            is_fc: boolean;
            is_logi: boolean;
            kills_90d: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_active: string | null;
            losses_90d: number;
            name: string;
            security_status: number;
        }>;
        page: number;
        total: number;
    };
};

export type EntityPageMembersResponse = EntityPageMembersResponses[keyof EntityPageMembersResponses];

export type EntityPageMostValuableData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Restrict the losses to one category of hull.
         */
        dataType?: 'most_valuable_kills' | 'most_valuable_ships' | 'most_valuable_structures';
        /**
         * Size of the trailing window, in days.
         */
        days?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/entities/{type}/{id}/most-valuable';
};

export type EntityPageMostValuableResponses = {
    /**
     * OK
     */
    200: {
        entries: Array<{
            killmail_hash: string;
            killmail_id: number;
            ship_name: string;
            ship_type_id: number;
            total_value: number;
            victim_alliance_name: string | null;
            victim_character_id: number | null;
            victim_character_name: string | null;
            victim_corporation_name: string | null;
        }>;
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
        groups: Array<{
            group_id: number;
            group_name: string;
            isk_lost: number;
            losses: number;
        }>;
    };
};

export type EntityPageShipClassesResponse = EntityPageShipClassesResponses[keyof EntityPageShipClassesResponses];

export type EntityPageStatsData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Trailing window in days. 0 covers the whole record.
         */
        days?: number;
    };
    url: '/entities/{type}/{id}/stats';
};

export type EntityPageStatsResponses = {
    /**
     * OK
     */
    200: {
        activity?: {
            kills: Array<Array<number>>;
            losses: Array<Array<number>>;
        };
        diesToAlliances: Array<{
            count: number;
            id: number;
            isk_value: number;
            name: string;
        }>;
        diesToCorporations: Array<{
            count: number;
            id: number;
            isk_value: number;
            name: string;
        }>;
        fliesWithAlliances?: Array<{
            count: number;
            id: number;
            isk_value: number;
            name: string;
        }>;
        fliesWithCorporations?: Array<{
            count: number;
            id: number;
            isk_value: number;
            name: string;
        }>;
        heatMap?: {
            [key: string]: number;
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
        topShipsLost: Array<{
            count: number;
            ship_name: string;
            ship_type_id: number;
        }>;
        topShipsUsed: Array<{
            count: number;
            ship_name: string;
            ship_type_id: number;
        }>;
    };
};

export type EntityPageStatsResponse = EntityPageStatsResponses[keyof EntityPageStatsResponses];

export type EntityPageTopListsData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Trailing window in days. 0 covers the whole record.
         */
        days?: number;
    };
    url: '/entities/{type}/{id}/top-lists';
};

export type EntityPageTopListsResponses = {
    /**
     * OK
     */
    200: {
        killed: {
            alliances: Array<{
                count: number;
                id: number;
                isk_value: number;
                name: string;
                palette: string | null;
            }>;
            characters: Array<{
                count: number;
                id: number;
                isk_value: number;
                name: string;
                palette: string | null;
            }>;
            corporations: Array<{
                count: number;
                id: number;
                isk_value: number;
                name: string;
                palette: string | null;
            }>;
        };
        killedBy: {
            alliances: Array<{
                count: number;
                id: number;
                isk_value: number;
                name: string;
                palette: string | null;
            }>;
            characters: Array<{
                count: number;
                id: number;
                isk_value: number;
                name: string;
                palette: string | null;
            }>;
            corporations: Array<{
                count: number;
                id: number;
                isk_value: number;
                name: string;
                palette: string | null;
            }>;
        };
    };
};

export type EntityPageTopListsResponse = EntityPageTopListsResponses[keyof EntityPageTopListsResponses];

export type EntityResolveCompatData = {
    body?: never;
    path?: never;
    query: {
        /**
         * Entity kind to resolve. Required.
         */
        type: 'character' | 'corporation' | 'alliance' | 'faction';
        /**
         * Entity ID to resolve. Required.
         */
        id: number;
    };
    url: '/entity/resolve';
};

export type EntityResolveCompatResponses = {
    /**
     * OK
     */
    200: {
        id: number;
        name: string;
        type: string;
    };
};

export type EntityResolveCompatResponse = EntityResolveCompatResponses[keyof EntityResolveCompatResponses];

export type EntityPageKilllistGenericCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Whose killmails to return.
         */
        role?: 'kills' | 'losses' | 'combined';
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
        /**
         * Page number for offset paging. Leave at 0 to page by cursor.
         */
        page?: number;
        /**
         * Restrict to one victim ship group ID.
         */
        ship_group?: number;
    };
    url: '/entity/{type}/{id}/killlist';
};

export type EntityPageKilllistGenericCompatResponses = {
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
        totalPages?: number;
    };
};

export type EntityPageKilllistGenericCompatResponse = EntityPageKilllistGenericCompatResponses[keyof EntityPageKilllistGenericCompatResponses];

export type EntityPageMostValuableGenericCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Restrict the losses to one category of hull.
         */
        dataType?: 'most_valuable_kills' | 'most_valuable_ships' | 'most_valuable_structures';
        /**
         * Size of the trailing window, in days.
         */
        days?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/entity/{type}/{id}/most-valuable';
};

export type EntityPageMostValuableGenericCompatResponses = {
    /**
     * OK
     */
    200: {
        entries: Array<{
            killmail_hash: string;
            killmail_id: number;
            ship_name: string;
            ship_type_id: number;
            total_value: number;
            victim_alliance_name: string | null;
            victim_character_id: number | null;
            victim_character_name: string | null;
            victim_corporation_name: string | null;
        }>;
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
        groups: Array<{
            group_id: number;
            group_name: string;
            isk_lost: number;
            losses: number;
        }>;
    };
};

export type EntityPageShipClassesGenericCompatResponse = EntityPageShipClassesGenericCompatResponses[keyof EntityPageShipClassesGenericCompatResponses];

export type EntityPageTopListsGenericCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Trailing window in days. 0 covers the whole record.
         */
        days?: number;
    };
    url: '/entity/{type}/{id}/top-lists';
};

export type EntityPageTopListsGenericCompatResponses = {
    /**
     * OK
     */
    200: {
        killed: {
            alliances: Array<{
                count: number;
                id: number;
                isk_value: number;
                name: string;
                palette: string | null;
            }>;
            characters: Array<{
                count: number;
                id: number;
                isk_value: number;
                name: string;
                palette: string | null;
            }>;
            corporations: Array<{
                count: number;
                id: number;
                isk_value: number;
                name: string;
                palette: string | null;
            }>;
        };
        killedBy: {
            alliances: Array<{
                count: number;
                id: number;
                isk_value: number;
                name: string;
                palette: string | null;
            }>;
            characters: Array<{
                count: number;
                id: number;
                isk_value: number;
                name: string;
                palette: string | null;
            }>;
            corporations: Array<{
                count: number;
                id: number;
                isk_value: number;
                name: string;
                palette: string | null;
            }>;
        };
    };
};

export type EntityPageTopListsGenericCompatResponse = EntityPageTopListsGenericCompatResponses[keyof EntityPageTopListsGenericCompatResponses];

export type FactionWarDashboardDetailData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Size of the trailing window, in days.
         */
        days?: number;
    };
    url: '/faction-war/{matchup}';
};

export type FactionWarDashboardDetailResponses = {
    /**
     * OK
     */
    200: {
        days: number;
        matchup: string;
        side1: {
            corpId: number;
            id: number;
            isk_destroyed: number;
            isk_lost: number;
            kills: number;
            losses: number;
            name: string;
            topAlliances: Array<{
                id: number;
                kills: number;
                name: string | null;
            }>;
            topCharacters: Array<{
                id: number;
                kills: number;
                name: string | null;
            }>;
            topCorporations: Array<{
                id: number;
                kills: number;
                name: string | null;
            }>;
        };
        side2: {
            corpId: number;
            id: number;
            isk_destroyed: number;
            isk_lost: number;
            kills: number;
            losses: number;
            name: string;
            topAlliances: Array<{
                id: number;
                kills: number;
                name: string | null;
            }>;
            topCharacters: Array<{
                id: number;
                kills: number;
                name: string | null;
            }>;
            topCorporations: Array<{
                id: number;
                kills: number;
                name: string | null;
            }>;
        };
        topShips: Array<{
            kills: number;
            ship_name: string;
            ship_type_id: number;
        }>;
    };
};

export type FactionWarDashboardDetailResponse = FactionWarDashboardDetailResponses[keyof FactionWarDashboardDetailResponses];

export type FactionWarDashboardData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Size of the trailing window, in days.
         */
        days?: number;
        /**
         * Reporting window.
         */
        period?: 'yesterday' | 'last_week' | 'active_total';
    };
    url: '/faction-war/{matchup}/dashboard';
};

export type FactionWarDashboardResponses = {
    /**
     * OK
     */
    200: {
        days: number;
        matchup: string;
        overview: {
            factionStats: {
                side1: {
                    kills_last_week: number;
                    kills_total: number;
                    kills_yesterday: number;
                    pilots: number;
                    systems_controlled: number;
                    vp_last_week: number;
                    vp_total: number;
                    vp_yesterday: number;
                } | null;
                side2: {
                    kills_last_week: number;
                    kills_total: number;
                    kills_yesterday: number;
                    pilots: number;
                    systems_controlled: number;
                    vp_last_week: number;
                    vp_total: number;
                    vp_yesterday: number;
                } | null;
            };
            flipDays: Array<{
                day: string;
                items: Array<{
                    /**
                     * UTC timestamp with millisecond precision.
                     */
                    flipped_at: string;
                    new_faction_id: number;
                    new_faction_name: string | null;
                    old_faction_id: number;
                    old_faction_name: string | null;
                    solar_system_id: number;
                    system_name: string | null;
                }>;
            }>;
            leaderboards: {
                characters: {
                    side1: Array<{
                        corporation_name?: string | null;
                        id: number;
                        kills: number;
                        name: string | null;
                    }>;
                    side2: Array<{
                        corporation_name?: string | null;
                        id: number;
                        kills: number;
                        name: string | null;
                    }>;
                };
                corporations: {
                    side1: Array<{
                        corporation_name?: string | null;
                        id: number;
                        kills: number;
                        name: string | null;
                    }>;
                    side2: Array<{
                        corporation_name?: string | null;
                        id: number;
                        kills: number;
                        name: string | null;
                    }>;
                };
            };
            warzone: {
                side1: {
                    captured: number;
                    contested: number;
                    total: number;
                    total_threshold: number;
                    total_vp: number;
                    uncontested: number;
                    vulnerable: number;
                };
                side2: {
                    captured: number;
                    contested: number;
                    total: number;
                    total_threshold: number;
                    total_vp: number;
                    uncontested: number;
                    vulnerable: number;
                };
                total_systems: number;
            };
        };
        side1: {
            corpId: number;
            id: number;
            isk_destroyed: number;
            isk_lost: number;
            kills: number;
            losses: number;
            name: string;
            topAlliances: Array<{
                id: number;
                kills: number;
                name: string | null;
            }>;
            topCharacters: Array<{
                id: number;
                kills: number;
                name: string | null;
            }>;
            topCorporations: Array<{
                id: number;
                kills: number;
                name: string | null;
            }>;
        };
        side2: {
            corpId: number;
            id: number;
            isk_destroyed: number;
            isk_lost: number;
            kills: number;
            losses: number;
            name: string;
            topAlliances: Array<{
                id: number;
                kills: number;
                name: string | null;
            }>;
            topCharacters: Array<{
                id: number;
                kills: number;
                name: string | null;
            }>;
            topCorporations: Array<{
                id: number;
                kills: number;
                name: string | null;
            }>;
        };
        topShips: Array<{
            kills: number;
            ship_name: string;
            ship_type_id: number;
        }>;
    };
};

export type FactionWarDashboardResponse = FactionWarDashboardResponses[keyof FactionWarDashboardResponses];

export type FactionWarIntelData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Size of the trailing window, in days.
         */
        days?: number;
    };
    url: '/faction-war/{matchup}/intel';
};

export type FactionWarIntelResponses = {
    /**
     * OK
     */
    200: {
        days: number;
        matchup: string;
        security_breakdown: Array<{
            isk_destroyed: number;
            kills: number;
            sec_class: string;
        }>;
        ship_groups_destroyed: Array<{
            count?: number;
            group_id?: number;
            group_name?: string | null;
            isk_destroyed?: number;
            [key: string]: unknown;
        }>;
        ships_destroyed: Array<{
            count: number;
            group_id: number | null;
            group_name: string | null;
            isk_destroyed: number;
            ship_name: string | null;
            ship_type_id: number;
        }>;
        ships_used: Array<{
            count: number;
            group_id: number | null;
            group_name: string | null;
            ship_name: string | null;
            ship_type_id: number;
        }>;
        summary: {
            constellations: number;
            isk_destroyed: number;
            kills: number;
            regions: number;
            systems: number;
        };
        top_constellations: Array<{
            constellation_id?: number;
            constellation_name?: string | null;
            isk_destroyed?: number;
            kills?: number;
            region_id?: number | null;
            region_name?: string | null;
            [key: string]: unknown;
        }>;
        top_regions: Array<{
            isk_destroyed?: number;
            kills?: number;
            region_id?: number;
            region_name?: string | null;
            [key: string]: unknown;
        }>;
        top_systems: Array<{
            isk_destroyed?: number;
            kills?: number;
            region_id?: number | null;
            region_name?: string | null;
            security?: number | null;
            system_id?: number;
            system_name?: string | null;
            [key: string]: unknown;
        }>;
    };
};

export type FactionWarIntelResponse = FactionWarIntelResponses[keyof FactionWarIntelResponses];

export type FactionWarMembersData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Which side of the war to report on.
         */
        side?: 'aggressor' | 'defender' | 'combined';
        /**
         * Ordering for the member rows.
         */
        sort?: 'kills' | 'losses' | 'isk' | 'activity';
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Restrict to one corporation.
         */
        corporationId?: number;
        /**
         * Restrict to one alliance.
         */
        allianceId?: number;
        /**
         * Size of the trailing window, in days.
         */
        days?: number;
    };
    url: '/faction-war/{matchup}/members';
};

export type FactionWarMembersResponses = {
    /**
     * OK
     */
    200: {
        count: number;
        days: number;
        limit: number;
        matchup: string;
        members: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            alliance_ticker: string | null;
            character_id: number;
            character_name: string;
            corporation_id: number | null;
            corporation_name: string | null;
            corporation_ticker: string | null;
            isk_destroyed: number;
            isk_lost: number;
            kills: number;
            losses: number;
            side: string;
            top_ship_count: number;
            top_ship_name: string | null;
            top_ship_type_id: number | null;
        }>;
        side: string;
    };
};

export type FactionWarMembersResponse = FactionWarMembersResponses[keyof FactionWarMembersResponses];

export type FactionWarOverviewData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Reporting window.
         */
        period?: 'yesterday' | 'last_week' | 'active_total';
    };
    url: '/faction-war/{matchup}/overview';
};

export type FactionWarOverviewResponses = {
    /**
     * OK
     */
    200: {
        factionStats: {
            side1: {
                kills_last_week: number;
                kills_total: number;
                kills_yesterday: number;
                pilots: number;
                systems_controlled: number;
                vp_last_week: number;
                vp_total: number;
                vp_yesterday: number;
            } | null;
            side2: {
                kills_last_week: number;
                kills_total: number;
                kills_yesterday: number;
                pilots: number;
                systems_controlled: number;
                vp_last_week: number;
                vp_total: number;
                vp_yesterday: number;
            } | null;
        };
        flipDays: Array<{
            day: string;
            items: Array<{
                /**
                 * UTC timestamp with millisecond precision.
                 */
                flipped_at: string;
                new_faction_id: number;
                new_faction_name: string | null;
                old_faction_id: number;
                old_faction_name: string | null;
                solar_system_id: number;
                system_name: string | null;
            }>;
        }>;
        leaderboards: {
            characters: {
                side1: Array<{
                    corporation_name?: string | null;
                    id: number;
                    kills: number;
                    name: string | null;
                }>;
                side2: Array<{
                    corporation_name?: string | null;
                    id: number;
                    kills: number;
                    name: string | null;
                }>;
            };
            corporations: {
                side1: Array<{
                    corporation_name?: string | null;
                    id: number;
                    kills: number;
                    name: string | null;
                }>;
                side2: Array<{
                    corporation_name?: string | null;
                    id: number;
                    kills: number;
                    name: string | null;
                }>;
            };
        };
        warzone: {
            side1: {
                captured: number;
                contested: number;
                total: number;
                total_threshold: number;
                total_vp: number;
                uncontested: number;
                vulnerable: number;
            };
            side2: {
                captured: number;
                contested: number;
                total: number;
                total_threshold: number;
                total_vp: number;
                uncontested: number;
                vulnerable: number;
            };
            total_systems: number;
        };
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
        celestials: Array<{
            group_id: number;
            system_id: number;
            x: number;
            z: number;
        }>;
        jumps: Array<Array<number>>;
        systems: Array<{
            constellation_id: number;
            constellation_name: string | null;
            contested: string | null;
            isk_24h: number;
            kills_24h: number;
            occupier_faction_id: number;
            owner_faction_id: number;
            region_id: number;
            region_name: string | null;
            security: number;
            solar_system_id: number;
            system_name: string | null;
            victory_points: number;
            victory_points_threshold: number;
            x: number;
            y: number;
        }>;
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
        'amarr-minmatar': {
            [key: string]: {
                faction_id: number;
                isk_destroyed: number;
                isk_lost: number;
                kills: number;
                losses: number;
                name: string;
                pilots: number;
                systems_controlled: number;
            };
        };
        'caldari-gallente': {
            [key: string]: {
                faction_id: number;
                isk_destroyed: number;
                isk_lost: number;
                kills: number;
                losses: number;
                name: string;
                pilots: number;
                systems_controlled: number;
            };
        };
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
        character: {
            alliance_id?: number | null;
            alliance_name?: string | null;
            alliance_ticker?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            birthday?: string | null;
            bloodline_name?: string | null;
            character_id?: number;
            corporation_id?: number | null;
            corporation_name?: string | null;
            corporation_ticker?: string | null;
            custom_description?: string | null;
            custom_description_format?: string | null;
            custom_description_html?: string | null;
            description?: string | null;
            faction_id?: number | null;
            faction_name?: string | null;
            gender?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_active?: string | null;
            name?: string;
            palette?: string | null;
            race_name?: string | null;
            security_status?: number;
            title?: string | null;
            [key: string]: unknown;
        };
        corporationHistory: Array<{
            corporation_id: number;
            corporation_name: string;
            corporation_ticker: string;
            kills: number;
            losses: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            start_date: string;
        }>;
        corporationHistoryQueued: boolean;
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
    } | {
        allianceHistory: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            alliance_ticker: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            start_date: string;
        }>;
        corporation: {
            alliance_id?: number | null;
            alliance_name?: string | null;
            alliance_ticker?: string | null;
            ceo_id?: number | null;
            ceo_name?: string | null;
            corporation_id?: number;
            creator_id?: number | null;
            creator_name?: string | null;
            custom_description?: string | null;
            custom_description_format?: string | null;
            custom_description_html?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            date_founded?: string | null;
            description?: string | null;
            faction_id?: number | null;
            faction_name?: string | null;
            friendly_fire?: boolean | null;
            lp_tax_rate?: number | null;
            member_count?: number;
            name?: string;
            palette?: string | null;
            state?: string | null;
            tax_rate?: number;
            ticker?: string;
            type?: string | null;
            url?: string | null;
            war_eligible?: boolean;
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
    } | {
        alliance: {
            alliance_id?: number;
            corporation_count?: number;
            creator_id?: number | null;
            creator_name?: string | null;
            custom_description?: string | null;
            custom_description_format?: string | null;
            custom_description_html?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            date_founded?: string | null;
            executor_corporation_id?: number | null;
            executor_name?: string | null;
            executor_ticker?: string | null;
            faction_id?: number | null;
            faction_name?: string | null;
            member_count?: number;
            name?: string;
            palette?: string | null;
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
    } | {
        faction: {
            corporation_id?: number | null;
            description?: string | null;
            faction_id?: number;
            militia_corporation_id?: number | null;
            name?: string;
            solar_system_id?: number | null;
            station_count?: number;
            station_system_count?: number;
            [key: string]: unknown;
        };
        recentStats: {
            isk_lost: number;
            losses: number;
        };
        stats: {
            isk_lost: number;
            losses: number;
        };
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
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
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
        description?: string | null;
        items: Array<{
            charge_type_id?: number;
            ordinal: number;
            quantity?: number;
            slot_group: number;
            state: number;
            type_id: number;
        }>;
        name: string;
        ship_type_id: number;
        visibility: number;
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
        /**
         * UTC timestamp with millisecond precision.
         */
        created_at?: string;
        description: string | null;
        fit_id: string;
        items: Array<{
            charge_type_id: number | null;
            ordinal: number;
            quantity: number;
            slot_group: number;
            state: number;
            type_id: number;
        }>;
        name: string;
        owner_character_id: number;
        rating_avg?: number | null;
        rating_count?: number;
        ship_type_id: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        updated_at?: string;
        visibility: number;
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
        ok: boolean;
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
        fit: {
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            description?: string | null;
            fit_id?: string;
            name?: string;
            owner_character_id?: number;
            rating_avg?: number | null;
            rating_count?: number;
            ship_type_id?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            visibility?: number;
            [key: string]: unknown;
        };
        items: Array<{
            charge_type_id: number | null;
            ordinal: number;
            quantity: number;
            slot_group: number;
            state: number;
            type_id: number;
        }>;
        viewer_rating: number | null;
    };
};

export type FittingDetailLegacyResponse = FittingDetailLegacyResponses[keyof FittingDetailLegacyResponses];

export type FittingUpdateLegacyData = {
    body: {
        description?: string | null;
        items?: Array<{
            charge_type_id?: number;
            ordinal: number;
            quantity?: number;
            slot_group: number;
            state: number;
            type_id: number;
        }>;
        name?: string;
        visibility?: number;
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
        fit: {
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            description?: string | null;
            fit_id?: string;
            name?: string;
            owner_character_id?: number;
            rating_avg?: number | null;
            rating_count?: number;
            ship_type_id?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            visibility?: number;
            [key: string]: unknown;
        };
        items: Array<{
            charge_type_id: number | null;
            ordinal: number;
            quantity: number;
            slot_group: number;
            state: number;
            type_id: number;
        }>;
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
        aggregate: {
            rating_avg: number | null;
            rating_count: number;
        };
        deleted: boolean;
    };
};

export type FittingRatingDeleteLegacyResponse = FittingRatingDeleteLegacyResponses[keyof FittingRatingDeleteLegacyResponses];

export type FittingRatingPutLegacyData = {
    body: {
        rating: number;
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
        aggregate: {
            rating_avg: number | null;
            rating_count: number;
        };
        rating: number;
    };
};

export type FittingRatingPutLegacyResponse = FittingRatingPutLegacyResponses[keyof FittingRatingPutLegacyResponses];

export type FittingsCommunityLatestLegacyData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/fits/community-latest';
};

export type FittingsCommunityLatestLegacyResponses = {
    /**
     * OK
     */
    200: {
        fits: Array<{
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            description: string | null;
            fit_id: string;
            module_count: number;
            name: string;
            owner_character_id: number;
            owner_name: string | null;
            rating_avg: number | null;
            rating_count: number;
            ship_name: string | null;
            ship_type_id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
        }>;
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
        families: Array<{
            canonical_fit_hash: string;
            canonical_uses: number;
            drones: Array<{
                name: string | null;
                quantity: number;
                type_id: number;
            }>;
            family_hash: string;
            fit_cost: number;
            hull_cost: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_used: string;
            modules: Array<{
                charge_name: string | null;
                charge_type_id: number | null;
                name: string | null;
                ordinal: number;
                slot_group: number;
                type_id: number;
            }>;
            ship_name: string | null;
            ship_type_id: number;
            top_alliances?: Array<{
                alliance_id: number | null;
                name: string | null;
                pct_of_alliance_losses: number;
                uses: number;
            }>;
            total_uses: number;
            variant_count: number;
        }>;
        window_days: number;
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
        ships: Array<{
            fit_count: number;
            group_id: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_used: string;
            ship_name: string | null;
            ship_type_id: number;
            total_uses: number;
        }>;
        window_days: number;
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
        community_fits: number;
        fittings_known: number;
        killmails_analyzed: number;
        ratings_cast: number;
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
        roles: Array<{
            category: string;
            description?: string;
            icon: string;
            id: string;
            label: string;
            typeCount: number;
        }>;
    };
};

export type FittingsRolesLegacyResponse = FittingsRolesLegacyResponses[keyof FittingsRolesLegacyResponses];

export type FittingsSearchLegacyData = {
    body?: never;
    path?: never;
    query: {
        /**
         * Hull type ID to search fittings for. Required.
         */
        ship: number;
        /**
         * JSON array of at most 8 module or role filters, each `{"kind":"module"|"role",...}`.
         */
        filters?: string;
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Rows to skip before the page.
         */
        offset?: number;
    };
    url: '/fits/search';
};

export type FittingsSearchLegacyResponses = {
    /**
     * OK
     */
    200: {
        filters_applied: Array<{
            count: number;
            kind: string;
            op: string;
            role_id?: string;
            type_id?: number;
            type_name?: string;
        }>;
        fits: Array<{
            drones: Array<{
                name: string | null;
                quantity: number;
                type_id: number;
            }>;
            family_hash: string;
            fit_cost: number;
            fit_hash: string;
            hull_cost: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_used: string;
            modules: Array<{
                charge_name: string | null;
                charge_type_id: number | null;
                name: string | null;
                ordinal: number;
                slot_group: number;
                type_id: number;
            }>;
            ship_name: string | null;
            ship_type_id: number;
            total_uses: number;
        }>;
        has_more: boolean;
        limit: number;
        offset: number;
        ship_name?: string | null;
        ship_type_id: number;
        total: number;
        window_days: number;
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
        doctrines: Array<{
            alliance_id: number;
            alliance_name: string | null;
            canonical_fit_hash: string;
            doctrine_share: number;
            doctrine_uses: number;
            drones: Array<{
                name: string | null;
                quantity: number;
                type_id: number;
            }>;
            family_hash: string;
            fit_cost: number;
            hull_cost: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_used: string;
            modules: Array<{
                charge_name: string | null;
                charge_type_id: number | null;
                name: string | null;
                ordinal: number;
                slot_group: number;
                type_id: number;
            }>;
            ship_name: string | null;
            ship_type_id: number;
            total_losses: number;
        }>;
        window_days: number;
    };
};

export type FittingsAllianceDoctrinesLegacyResponse = FittingsAllianceDoctrinesLegacyResponses[keyof FittingsAllianceDoctrinesLegacyResponses];

export type FittingsCommunityTopRatedLegacyData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/fits/top-rated';
};

export type FittingsCommunityTopRatedLegacyResponses = {
    /**
     * OK
     */
    200: {
        fits: Array<{
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            description: string | null;
            fit_id: string;
            module_count: number;
            name: string;
            owner_character_id: number;
            owner_name: string | null;
            rating_avg: number | null;
            rating_count: number;
            ship_name: string | null;
            ship_type_id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
        }>;
    };
};

export type FittingsCommunityTopRatedLegacyResponse = FittingsCommunityTopRatedLegacyResponses[keyof FittingsCommunityTopRatedLegacyResponses];

export type FittingCreateData = {
    body: {
        description?: string | null;
        items: Array<{
            charge_type_id?: number;
            ordinal: number;
            quantity?: number;
            slot_group: number;
            state: number;
            type_id: number;
        }>;
        name: string;
        ship_type_id: number;
        visibility: number;
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
        /**
         * UTC timestamp with millisecond precision.
         */
        created_at?: string;
        description: string | null;
        fit_id: string;
        items: Array<{
            charge_type_id: number | null;
            ordinal: number;
            quantity: number;
            slot_group: number;
            state: number;
            type_id: number;
        }>;
        name: string;
        owner_character_id: number;
        rating_avg?: number | null;
        rating_count?: number;
        ship_type_id: number;
        /**
         * UTC timestamp with millisecond precision.
         */
        updated_at?: string;
        visibility: number;
        [key: string]: unknown;
    };
};

export type FittingCreateResponse = FittingCreateResponses[keyof FittingCreateResponses];

export type FittingsCommunityLatestData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/fittings/community/latest';
};

export type FittingsCommunityLatestResponses = {
    /**
     * OK
     */
    200: {
        fits: Array<{
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            description: string | null;
            fit_id: string;
            module_count: number;
            name: string;
            owner_character_id: number;
            owner_name: string | null;
            rating_avg: number | null;
            rating_count: number;
            ship_name: string | null;
            ship_type_id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
        }>;
    };
};

export type FittingsCommunityLatestResponse = FittingsCommunityLatestResponses[keyof FittingsCommunityLatestResponses];

export type FittingsCommunityTopRatedData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/fittings/community/top-rated';
};

export type FittingsCommunityTopRatedResponses = {
    /**
     * OK
     */
    200: {
        fits: Array<{
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            description: string | null;
            fit_id: string;
            module_count: number;
            name: string;
            owner_character_id: number;
            owner_name: string | null;
            rating_avg: number | null;
            rating_count: number;
            ship_name: string | null;
            ship_type_id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
        }>;
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
        doctrines: Array<{
            alliance_id: number;
            alliance_name: string | null;
            canonical_fit_hash: string;
            doctrine_share: number;
            doctrine_uses: number;
            drones: Array<{
                name: string | null;
                quantity: number;
                type_id: number;
            }>;
            family_hash: string;
            fit_cost: number;
            hull_cost: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_used: string;
            modules: Array<{
                charge_name: string | null;
                charge_type_id: number | null;
                name: string | null;
                ordinal: number;
                slot_group: number;
                type_id: number;
            }>;
            ship_name: string | null;
            ship_type_id: number;
            total_losses: number;
        }>;
        window_days: number;
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
        roles: Array<{
            category: string;
            description?: string;
            icon: string;
            id: string;
            label: string;
            typeCount: number;
        }>;
    };
};

export type FittingsRolesResponse = FittingsRolesResponses[keyof FittingsRolesResponses];

export type FittingsSearchData = {
    body?: never;
    path?: never;
    query: {
        /**
         * Hull type ID to search fittings for. Required.
         */
        ship: number;
        /**
         * JSON array of at most 8 module or role filters, each `{"kind":"module"|"role",...}`.
         */
        filters?: string;
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Rows to skip before the page.
         */
        offset?: number;
    };
    url: '/fittings/search';
};

export type FittingsSearchResponses = {
    /**
     * OK
     */
    200: {
        filters_applied: Array<{
            count: number;
            kind: string;
            op: string;
            role_id?: string;
            type_id?: number;
            type_name?: string;
        }>;
        fits: Array<{
            drones: Array<{
                name: string | null;
                quantity: number;
                type_id: number;
            }>;
            family_hash: string;
            fit_cost: number;
            fit_hash: string;
            hull_cost: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_used: string;
            modules: Array<{
                charge_name: string | null;
                charge_type_id: number | null;
                name: string | null;
                ordinal: number;
                slot_group: number;
                type_id: number;
            }>;
            ship_name: string | null;
            ship_type_id: number;
            total_uses: number;
        }>;
        has_more: boolean;
        limit: number;
        offset: number;
        ship_name?: string | null;
        ship_type_id: number;
        total: number;
        window_days: number;
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
        ships: Array<{
            fit_count: number;
            group_id: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_used: string;
            ship_name: string | null;
            ship_type_id: number;
            total_uses: number;
        }>;
        window_days: number;
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
        families: Array<{
            canonical_fit_hash: string;
            canonical_uses: number;
            drones: Array<{
                name: string | null;
                quantity: number;
                type_id: number;
            }>;
            family_hash: string;
            fit_cost: number;
            hull_cost?: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_used: string;
            modules: Array<{
                charge_name: string | null;
                charge_type_id: number | null;
                name: string | null;
                ordinal: number;
                slot_group: number;
                type_id: number;
            }>;
            top_alliances?: Array<{
                alliance_id: number | null;
                name: string | null;
                pct_of_alliance_losses: number;
                uses: number;
            }>;
            total_uses: number;
            variant_count: number;
        }>;
        hull_cost?: number | null;
        is_rare_hull: boolean;
        ship_type_id: number;
        window_days: number;
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
        groups: Array<{
            group_id: number;
            kill_count: number;
            name: string;
            pct: number;
        }>;
        ship_type_id: number;
        total_kills: number;
        window_days: number;
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
        community_fits: number;
        fittings_known: number;
        killmails_analyzed: number;
        ratings_cast: number;
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
        families: Array<{
            canonical_fit_hash: string;
            canonical_uses: number;
            drones: Array<{
                name: string | null;
                quantity: number;
                type_id: number;
            }>;
            family_hash: string;
            fit_cost: number;
            hull_cost: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_used: string;
            modules: Array<{
                charge_name: string | null;
                charge_type_id: number | null;
                name: string | null;
                ordinal: number;
                slot_group: number;
                type_id: number;
            }>;
            ship_name: string | null;
            ship_type_id: number;
            top_alliances?: Array<{
                alliance_id: number | null;
                name: string | null;
                pct_of_alliance_losses: number;
                uses: number;
            }>;
            total_uses: number;
            variant_count: number;
        }>;
        window_days: number;
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
        ok: boolean;
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
        fit: {
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            description?: string | null;
            fit_id?: string;
            name?: string;
            owner_character_id?: number;
            rating_avg?: number | null;
            rating_count?: number;
            ship_type_id?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            visibility?: number;
            [key: string]: unknown;
        };
        items: Array<{
            charge_type_id: number | null;
            ordinal: number;
            quantity: number;
            slot_group: number;
            state: number;
            type_id: number;
        }>;
        viewer_rating: number | null;
    };
};

export type FittingDetailResponse = FittingDetailResponses[keyof FittingDetailResponses];

export type FittingUpdateData = {
    body: {
        description?: string | null;
        items?: Array<{
            charge_type_id?: number;
            ordinal: number;
            quantity?: number;
            slot_group: number;
            state: number;
            type_id: number;
        }>;
        name?: string;
        visibility?: number;
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
        fit: {
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            description?: string | null;
            fit_id?: string;
            name?: string;
            owner_character_id?: number;
            rating_avg?: number | null;
            rating_count?: number;
            ship_type_id?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            visibility?: number;
            [key: string]: unknown;
        };
        items: Array<{
            charge_type_id: number | null;
            ordinal: number;
            quantity: number;
            slot_group: number;
            state: number;
            type_id: number;
        }>;
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
        aggregate: {
            rating_avg: number | null;
            rating_count: number;
        };
        deleted: boolean;
    };
};

export type FittingRatingDeleteResponse = FittingRatingDeleteResponses[keyof FittingRatingDeleteResponses];

export type FittingRatingPutData = {
    body: {
        rating: number;
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
        aggregate: {
            rating_avg: number | null;
            rating_count: number;
        };
        rating: number;
    };
};

export type FittingRatingPutResponse = FittingRatingPutResponses[keyof FittingRatingPutResponses];

export type GraphData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Which graph query to run.
         */
        mode?: 'path_finder' | 'coalitions' | 'rivalries' | 'entity_intel' | 'hunting_grounds' | 'hot_zones' | 'migration' | 'spy_check' | 'census';
        /**
         * Entity kind for the modes that take one. Anything other than `corporation` is read as `alliance`.
         */
        entityType?: 'character' | 'corporation' | 'alliance';
        /**
         * Entity ID for the entity-scoped modes.
         */
        entityId?: number;
        /**
         * Starting character for `path_finder`.
         */
        fromId?: number;
        /**
         * Destination character for `path_finder`.
         */
        toId?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/graph';
};

export type GraphResponses = {
    /**
     * OK
     */
    200: {
        error?: string;
        mode: string;
        path: {
            edges: Array<{
                weight: number;
            }>;
            hops: number;
            nodes: Array<{
                alliance_name?: string | null;
                corp_name?: string | null;
                id: number;
                name: string;
            }>;
        } | null;
    } | {
        coalitions: Array<{
            alliances: Array<{
                connections?: number;
                id: number;
                name: string;
            }>;
            id: number;
        }>;
        mode: string;
    } | {
        entityType: string;
        items: Array<{
            entity_a: {
                alliance_name?: string | null;
                corp_name?: string | null;
                id: number;
                name: string;
            };
            entity_b: {
                alliance_name?: string | null;
                corp_name?: string | null;
                id: number;
                name: string;
            };
            mutual_kills: number;
            total_isk: number;
        }>;
        mode: string;
    } | {
        allies: Array<{
            id?: number;
            name?: string;
            [key: string]: unknown;
        }>;
        enemies: Array<{
            id?: number;
            name?: string;
            [key: string]: unknown;
        }>;
        entityType?: string;
        mode: string;
    } | {
        entityType?: string;
        mode: string;
        systems: Array<{
            active_characters?: number;
            alliances?: number;
            characters?: number;
            id?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            latest_activity?: string;
            name?: string;
            [key: string]: unknown;
        }>;
    } | {
        corp_name?: string | null;
        departed: Array<{
            id?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            left_at?: string;
            name?: string;
            [key: string]: unknown;
        }>;
        joined: Array<{
            id?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            joined_at?: string;
            name?: string;
            [key: string]: unknown;
        }>;
        mode: string;
    } | {
        entityType?: string;
        mode: string;
        suspects: Array<{
            id?: number;
            name?: string;
            total_flights?: number;
            [key: string]: unknown;
        }>;
    } | {
        corps: Array<{
            caps?: number;
            droppers?: number;
            fcs?: number;
            id?: number;
            logis?: number;
            name?: string;
            supers?: number;
            total?: number;
            [key: string]: unknown;
        }>;
        mode: string;
        totals: {
            caps?: number;
            droppers?: number;
            fcs?: number;
            logis?: number;
            supers?: number;
            total?: number;
            [key: string]: unknown;
        };
    };
};

export type GraphResponse = GraphResponses[keyof GraphResponses];

export type GroupCompatData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/group/{id}';
};

export type GroupCompatResponses = {
    /**
     * OK
     */
    200: {
        group: {
            category_id: number | null;
            category_name: string | null;
            category_published: boolean | null;
            group_id: number;
            icon_id: number | null;
            name: string | null;
            published: boolean | null;
            published_type_count: number;
            type_count: number;
        };
        types: Array<{
            base_price: number | null;
            description: string | null;
            mass: number | null;
            meta_group_id: number | null;
            meta_group_name: string | null;
            name: string | null;
            published: boolean | null;
            type_id: number;
            volume: number | null;
        }>;
    };
};

export type GroupCompatResponse = GroupCompatResponses[keyof GroupCompatResponses];

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
    query?: {
        /**
         * Restrict the daily counts to one year. Omit for every day on record.
         */
        year?: number;
    };
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
        size?: 32 | 64 | 128 | 512 | 1024;
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
    200: Blob | File;
};

export type ImageDomainBackgroundResponse = ImageDomainBackgroundResponses[keyof ImageDomainBackgroundResponses];

export type ImageDomainAssetPreviewData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Signed preview token for an unapproved asset.
         */
        token?: string;
    };
    url: '/images/domains/preview/{assetId}';
};

export type ImageDomainAssetPreviewResponses = {
    /**
     * OK
     */
    200: Blob | File;
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
    200: Blob | File;
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
        size?: 32 | 64 | 128 | 512 | 1024;
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
        size?: 32 | 64 | 128 | 512 | 1024;
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
        size?: 32 | 64 | 128 | 512 | 1024;
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
        attributes: Array<{
            id: number;
            name?: string;
            value: number;
        }>;
        item: {
            base_price: number | null;
            capacity: number | null;
            category_id: number;
            category_name: string | null;
            description: string | null;
            faction_id: number | null;
            group_id: number;
            group_name: string | null;
            is_ship: boolean;
            market_group_id: number | null;
            mass: number | null;
            meta_group_id: number | null;
            meta_group_name: string | null;
            meta_level: number | null;
            name: string;
            packaged_volume: number | null;
            portion_size: number | null;
            published: boolean;
            race_id: number | null;
            radius: number | null;
            tech_level: number | null;
            type_id: number;
            volume: number | null;
        };
        marketBreadcrumb: Array<{
            id: number;
            name: string;
            slug: string;
        }>;
        materials: Array<{
            name: string | null;
            quantity: number;
            type_id: number;
        }>;
        pricing: {
            customHistory: Array<{
                date: string;
                price: number;
            }>;
            customSummary: {
                average_90d: number;
                avg_volume_90d?: number;
                highest_90d: number;
                latest: number;
                latest_date: string;
                lowest_90d: number | null;
            } | null;
            history: Array<{
                average: number;
                date: string;
                highest: number;
                lowest: number;
                volume: number;
            }>;
            insurance: Array<{
                cost?: number;
                name?: string;
                payout?: number;
                [key: string]: unknown;
            }>;
            summary: {
                average_90d: number;
                avg_volume_90d?: number;
                highest_90d: number;
                latest: number;
                latest_date: string;
                lowest_90d: number | null;
            } | null;
        };
        requiredSkills: Array<{
            level: number;
            name: string | null;
            type_id: number;
        }>;
        shipAttributes: {
            [key: string]: Array<{
                id: number;
                name?: string;
                value: number;
            }>;
        } | null;
        variations: Array<{
            meta_group_id: number | null;
            name: string;
            type_id: number;
        }>;
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
        groups: Array<{
            group_id: number;
            kill_count: number;
            name: string;
            pct: number;
        }>;
        ship_type_id: number;
        total_kills: number;
        window_days: number;
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
        families: Array<{
            canonical_fit_hash: string;
            canonical_uses: number;
            drones: Array<{
                name: string | null;
                quantity: number;
                type_id: number;
            }>;
            family_hash: string;
            fit_cost: number;
            hull_cost?: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_used: string;
            modules: Array<{
                charge_name: string | null;
                charge_type_id: number | null;
                name: string | null;
                ordinal: number;
                slot_group: number;
                type_id: number;
            }>;
            top_alliances?: Array<{
                alliance_id: number | null;
                name: string | null;
                pct_of_alliance_losses: number;
                uses: number;
            }>;
            total_uses: number;
            variant_count: number;
        }>;
        hull_cost?: number | null;
        is_rare_hull: boolean;
        ship_type_id: number;
        window_days: number;
    };
};

export type FittingsShipFamiliesLegacyResponse = FittingsShipFamiliesLegacyResponses[keyof FittingsShipFamiliesLegacyResponses];

export type ItemKilllistCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
    url: '/item/{id}/killlist';
};

export type ItemKilllistCompatResponses = {
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
        totalPages?: number;
    };
};

export type ItemKilllistCompatResponse = ItemKilllistCompatResponses[keyof ItemKilllistCompatResponses];

export type KilllistData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Killmail category: space, ship class, tech level, or value band.
         */
        type?: '10b' | '5b' | 'abyssal' | 'battlecruisers' | 'battleships' | 'big' | 'capitals' | 'citadels' | 'cruisers' | 'destroyers' | 'faction' | 'freighters' | 'frigates' | 'highsec' | 'jove' | 'latest' | 'lowsec' | 'npc' | 'nullsec' | 'pochven' | 'solo' | 'supercarriers' | 't1' | 't2' | 't3' | 'titans' | 'wspace';
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
        /**
         * Page number for offset paging. Leave at 0 to page by cursor.
         */
        page?: number;
        /**
         * Comma-separated faction IDs to restrict the victim to, for example `500001,500002`.
         */
        victimFactions?: string;
    };
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
        totalPages?: number;
    };
};

export type KilllistResponse = KilllistResponses[keyof KilllistResponses];

export type KilllistAdvancedData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * JSON filter tree: `entities`, `items`, `location`, `timeRange`, `attackerCount`, `attackerType`, `iskValue`, `iskMin`, `iskMax`, `shipCategory`, `techLevel`, and `sort`. Each ID-bearing list holds at most 15 entries.
         */
        filters?: string;
        /**
         * Return killmails or the fittings behind them.
         */
        view?: 'kills' | 'fits';
        /**
         * Group the `fits` view by exact fit or by fit family.
         */
        dedup?: 'none' | 'exact' | 'family';
        /**
         * 64-character hex fit hash to drill into.
         */
        fitHash?: string;
        /**
         * 64-character hex fit family hash to drill into.
         */
        familyHash?: string;
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
    url: '/killlist/advanced';
};

export type KilllistAdvancedResponses = {
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
        totalPages?: number;
    };
};

export type KilllistAdvancedResponse = KilllistAdvancedResponses[keyof KilllistAdvancedResponses];

export type KillmailSubmitData = {
    body: {
        /**
         * ESI killmail links. Joined with newlines and parsed the same way as text.
         */
        links?: Array<string>;
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
        accepted: number;
        existing: number;
        existingIds: Array<number>;
        killmails: Array<number>;
        message?: string;
        rejected: number;
        total: number;
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
        exists: boolean;
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
        drones: Array<{
            name: string | null;
            quantity: number;
            type_id: number;
        }>;
        modules: Array<{
            charge_type_id: number | null;
            name: string | null;
            ordinal: number;
            slot_group: number;
            type_id: number;
        }>;
        name: string;
        shipTypeId: number;
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
        siblings: Array<{
            killmail_id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            killmail_time: string;
            ship_group_id: number | null;
            ship_name: string | null;
            ship_type_id: number | null;
            total_value: number;
        }>;
    };
};

export type KillmailSiblingsLegacyResponse = KillmailSiblingsLegacyResponses[keyof KillmailSiblingsLegacyResponses];

export type KillmailsData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Killmail category: space, ship class, tech level, or value band.
         */
        type?: '10b' | '5b' | 'abyssal' | 'battlecruisers' | 'battleships' | 'big' | 'capitals' | 'citadels' | 'cruisers' | 'destroyers' | 'faction' | 'freighters' | 'frigates' | 'highsec' | 'jove' | 'latest' | 'lowsec' | 'npc' | 'nullsec' | 'pochven' | 'solo' | 'supercarriers' | 't1' | 't2' | 't3' | 'titans' | 'wspace';
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
        /**
         * Descending cursor, walking newest to oldest. Pass the previous response's pagination cursor to fetch the next page. Mutually exclusive with `after`, which it overrides.
         */
        before?: number;
        /**
         * Comma-separated faction IDs to restrict the victim to, for example `500001,500002`.
         */
        victimFactions?: string;
    };
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
    body: {
        /**
         * Return killmails after this identifier.
         */
        after?: number | string;
        /**
         * An integer. A numeric string is accepted for compatibility.
         */
        alliance_ids?: Array<number | string>;
        /**
         * An integer. A numeric string is accepted for compatibility.
         */
        character_ids?: Array<number | string>;
        /**
         * An integer. A numeric string is accepted for compatibility.
         */
        constellation_ids?: Array<number | string>;
        /**
         * An integer. A numeric string is accepted for compatibility.
         */
        corporation_ids?: Array<number | string>;
        /**
         * Window start as YYYY-MM-DD or an ISO 8601 timestamp.
         */
        from: string;
        /**
         * Maximum killmails to return.
         */
        limit?: number | string;
        /**
         * An integer. A numeric string is accepted for compatibility.
         */
        region_ids?: Array<number | string>;
        /**
         * An integer. A numeric string is accepted for compatibility.
         */
        system_ids?: Array<number | string>;
        /**
         * Window end as YYYY-MM-DD or an ISO 8601 timestamp.
         */
        to: string;
    };
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
        drones: Array<{
            name: string | null;
            quantity: number;
            type_id: number;
        }>;
        modules: Array<{
            charge_type_id: number | null;
            name: string | null;
            ordinal: number;
            slot_group: number;
            type_id: number;
        }>;
        name: string;
        shipTypeId: number;
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
        exists: boolean;
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
        siblings: Array<{
            killmail_id: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            killmail_time: string;
            ship_group_id: number | null;
            ship_name: string | null;
            ship_type_id: number | null;
            total_value: number;
        }>;
    };
};

export type KillmailSiblingsResponse = KillmailSiblingsResponses[keyof KillmailSiblingsResponses];

export type KillsMostValuableData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Killmail category: space, ship class, tech level, or value band.
         */
        type?: '10b' | '5b' | 'abyssal' | 'battlecruisers' | 'battleships' | 'big' | 'capitals' | 'citadels' | 'cruisers' | 'destroyers' | 'faction' | 'freighters' | 'frigates' | 'highsec' | 'jove' | 'latest' | 'lowsec' | 'npc' | 'nullsec' | 'pochven' | 'solo' | 'supercarriers' | 't1' | 't2' | 't3' | 'titans' | 'wspace';
        /**
         * Size of the trailing window, in days.
         */
        days?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/kills/most-valuable';
};

export type KillsMostValuableResponses = {
    /**
     * OK
     */
    200: {
        entries: Array<{
            killmail_hash: string;
            killmail_id: number;
            ship_name: string;
            ship_type_id: number;
            total_value: number;
            victim_alliance_name: string | null;
            victim_character_id: number | null;
            victim_character_name: string | null;
            victim_corporation_name: string | null;
        }>;
    };
};

export type KillsMostValuableResponse = KillsMostValuableResponses[keyof KillsMostValuableResponses];

export type KillsTopData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Killmail category: space, ship class, tech level, or value band.
         */
        type?: '10b' | '5b' | 'abyssal' | 'battlecruisers' | 'battleships' | 'big' | 'capitals' | 'citadels' | 'cruisers' | 'destroyers' | 'faction' | 'freighters' | 'frigates' | 'highsec' | 'jove' | 'latest' | 'lowsec' | 'npc' | 'nullsec' | 'pochven' | 'solo' | 'supercarriers' | 't1' | 't2' | 't3' | 'titans' | 'wspace';
        /**
         * Which leaderboard to build.
         */
        dataType?: 'characters' | 'corporations' | 'alliances' | 'ships' | 'systems' | 'regions';
        /**
         * Size of the trailing window, in days.
         */
        days?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/kills/top';
};

export type KillsTopResponses = {
    /**
     * OK
     */
    200: {
        entries: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
            region_id?: number | null;
            type: string;
        }>;
    };
};

export type KillsTopResponse = KillsTopResponses[keyof KillsTopResponses];

export type LegacyArchiveAutocompleteData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Search text. At least two characters.
         */
        q?: string;
        /**
         * Which column to complete.
         */
        field?: 'victim' | 'attacker' | 'corp' | 'alliance' | 'system';
        /**
         * Maximum suggestions to return.
         */
        limit?: number;
    };
    url: '/legacy/autocomplete';
};

export type LegacyArchiveAutocompleteResponses = {
    /**
     * OK
     */
    200: Array<{
        id: number | null;
        name: string;
    }>;
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
        attackers: Array<{
            alliance_id?: number | null;
            character_id?: number | null;
            corporation_id?: number | null;
            damage_done?: number;
            final_blow?: boolean;
            id?: number;
            killmail_id?: number;
            name?: string | null;
            ship_type_id?: number | null;
            [key: string]: unknown;
        }>;
        items: Array<{
            flag?: number;
            id?: number;
            killmail_id?: number;
            name?: string | null;
            quantity_destroyed?: number | null;
            quantity_dropped?: number | null;
            singleton?: number;
            type_id?: number | null;
            [key: string]: unknown;
        }>;
        kill: {
            killmail_id?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            killmail_time?: string;
            security?: number | null;
            system_name?: string | null;
            total_value?: number;
            victim_alliance?: string | null;
            victim_corp?: string | null;
            victim_name?: string | null;
            victim_ship?: string | null;
            [key: string]: unknown;
        };
    };
};

export type LegacyArchiveKillResponse = LegacyArchiveKillResponses[keyof LegacyArchiveKillResponses];

export type LegacyArchiveKillsData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Victim name contains this text.
         */
        victim?: string;
        /**
         * Victim corporation name contains this text.
         */
        corp?: string;
        /**
         * Victim alliance name contains this text.
         */
        alliance?: string;
        /**
         * System name contains this text.
         */
        system?: string;
        /**
         * Comma-separated hull names; a row matches any of them.
         */
        ship?: string;
        /**
         * Attacker name contains this text.
         */
        attacker?: string;
        /**
         * Earliest killmail date, as YYYY-MM-DD.
         */
        from?: string;
        /**
         * Latest killmail date, as YYYY-MM-DD.
         */
        to?: string;
        /**
         * Sort field and direction joined by an underscore.
         */
        sort?: 'id_desc' | 'id_asc' | 'value_desc' | 'value_asc' | 'time_desc' | 'time_asc';
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
    url: '/legacy/kills';
};

export type LegacyArchiveKillsResponses = {
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
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
        alliances: number;
        characters: number;
        corporations: number;
        killmails: number;
    };
};

export type LegacyArchiveStatsResponse = LegacyArchiveStatsResponses[keyof LegacyArchiveStatsResponses];

export type LegacyArchiveTopData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Which leaderboard to build.
         */
        dataType?: 'characters' | 'corporations' | 'alliances' | 'ships' | 'systems';
        /**
         * Restrict to one year.
         */
        year?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/legacy/top';
};

export type LegacyArchiveTopResponses = {
    /**
     * OK
     */
    200: {
        entries: Array<{
            count: number;
            id: number;
            name: string;
            palette?: string | null;
            region_id?: number | null;
            type: string;
        }>;
    };
};

export type LegacyArchiveTopResponse = LegacyArchiveTopResponses[keyof LegacyArchiveTopResponses];

export type LocationData = {
    body?: never;
    path?: never;
    query: {
        /**
         * Solar system to resolve within.
         */
        system_id: number;
        /**
         * X coordinate in metres.
         */
        x: number;
        /**
         * Y coordinate in metres.
         */
        y: number;
        /**
         * Z coordinate in metres.
         */
        z: number;
    };
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
        activity: Array<{
            npc_kills: number;
            pod_kills: number;
            ship_jumps: number;
            ship_kills: number;
            system_id: number;
        }>;
        celestials: Array<{
            group_id: number;
            system_id: number;
            x: number;
            z: number;
        }>;
        constellations: Array<{
            constellation_id: number;
            constellation_name: string;
        }>;
        externalJumps: Array<{
            external_region_id: number;
            external_region_name: string;
            external_security: number;
            external_system_id: number;
            external_system_name: string;
            external_x: number;
            external_x2d: number;
            external_z: number;
            external_z2d: number;
            internal_system_id: number;
        }>;
        jumps: Array<{
            from_solar_system_id: number;
            to_solar_system_id: number;
        }>;
        region: {
            name: string;
            region_id: number;
            system_count?: number;
        };
        systems: Array<{
            constellation_id: number;
            region_id: number;
            security: number;
            solar_system_id: number;
            system_name: string;
            x: number;
            x2d: number;
            y: number;
            z: number;
            z2d: number;
        }>;
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
        abyssal: Array<{
            name: string;
            region_id: number;
            system_count?: number;
        }>;
        kspace: Array<{
            name: string;
            region_id: number;
            system_count?: number;
        }>;
        pochven: Array<{
            name: string;
            region_id: number;
            system_count?: number;
        }>;
        proving: Array<{
            name: string;
            region_id: number;
            system_count?: number;
        }>;
        wormhole: Array<{
            name: string;
            region_id: number;
            system_count?: number;
        }>;
        zarzakh: Array<{
            name: string;
            region_id: number;
            system_count?: number;
        }>;
    };
};

export type MapRegionsResponse = MapRegionsResponses[keyof MapRegionsResponses];

export type MapScopeData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Which slice of New Eden to return.
         */
        type?: 'new-eden' | 'zarzakh' | 'wormhole' | 'abyssal' | 'proving';
    };
    url: '/map/scope';
};

export type MapScopeResponses = {
    /**
     * OK
     */
    200: {
        activity: Array<{
            npc_kills: number;
            pod_kills: number;
            ship_jumps: number;
            ship_kills: number;
            system_id: number;
        }>;
        externalJumps: Array<{
            external_region_id: number;
            external_region_name: string;
            external_security: number;
            external_system_id: number;
            external_system_name: string;
            external_x: number;
            external_x2d: number;
            external_z: number;
            external_z2d: number;
            internal_system_id: number;
        }>;
        jumps: Array<{
            from_solar_system_id: number;
            to_solar_system_id: number;
        }>;
        regions: Array<{
            name: string;
            region_id: number;
            system_count?: number;
        }>;
        scope: string;
        systems: Array<{
            constellation_id: number;
            region_id: number;
            security: number;
            solar_system_id: number;
            system_name: string;
            x: number;
            x2d: number;
            y: number;
            z: number;
            z2d: number;
        }>;
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
        items: Array<{
            category_id: number;
            group_id: number;
            is_ship: boolean;
            meta_group_id: number | null;
            name: string;
            type_id: number;
        }>;
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
        groups: Array<{
            children: Array<{
                children: Array<{
                    children: Array<{
                        [key: string]: unknown;
                    }>;
                    has_types: boolean;
                    icon_id: number | null;
                    id: number;
                    name: string;
                    parent_id: number | null;
                    slug: string;
                }>;
                has_types: boolean;
                icon_id: number | null;
                id: number;
                name: string;
                parent_id: number | null;
                slug: string;
            }>;
            has_types: boolean;
            icon_id: number | null;
            id: number;
            name: string;
            parent_id: number | null;
            slug: string;
        }>;
    };
};

export type MarketTreeResponse = MarketTreeResponses[keyof MarketTreeResponses];

export type ShipMatchupData = {
    body?: never;
    path?: never;
    query: {
        /**
         * Attacking hull type ID. Required.
         */
        attacker: number;
        /**
         * Victim hull type ID. Required.
         */
        victim: number;
    };
    url: '/matchup';
};

export type ShipMatchupResponses = {
    /**
     * OK
     */
    200: {
        attacker_ship_type_id: number;
        attacker_win_rate: number;
        attacker_wins: number;
        enough: boolean;
        min_sample: number;
        mirror: boolean;
        sample: number;
        top_fits: Array<{
            family_hash: string;
            modules: Array<{
                name: string | null;
                slot_group: number;
                type_id: number;
            }>;
            pct: number;
            uses: number;
        }>;
        victim_ship_type_id: number;
        victim_wins: number;
        window_days: number;
    };
};

export type ShipMatchupResponse = ShipMatchupResponses[keyof ShipMatchupResponses];

export type McpBattleReportData = {
    body: BattleReportInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/battle_report';
};

export type McpBattleReportErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpBattleReportError = McpBattleReportErrors[keyof McpBattleReportErrors];

export type McpBattleReportResponses = {
    /**
     * OK
     */
    200: BattleReportOutput;
};

export type McpBattleReportResponse = McpBattleReportResponses[keyof McpBattleReportResponses];

export type McpCapsuleerDossierData = {
    body: DossierInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/capsuleer_dossier';
};

export type McpCapsuleerDossierErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpCapsuleerDossierError = McpCapsuleerDossierErrors[keyof McpCapsuleerDossierErrors];

export type McpCapsuleerDossierResponses = {
    /**
     * OK
     */
    200: DossierOutput;
};

export type McpCapsuleerDossierResponse = McpCapsuleerDossierResponses[keyof McpCapsuleerDossierResponses];

export type McpCharacterHistoryData = {
    body: CharacterHistoryInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/character_history';
};

export type McpCharacterHistoryErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpCharacterHistoryError = McpCharacterHistoryErrors[keyof McpCharacterHistoryErrors];

export type McpCharacterHistoryResponses = {
    /**
     * OK
     */
    200: CharacterHistoryOutput;
};

export type McpCharacterHistoryResponse = McpCharacterHistoryResponses[keyof McpCharacterHistoryResponses];

export type McpCoalitionGraphData = {
    body: CoalitionGraphInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/coalition_graph';
};

export type McpCoalitionGraphErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpCoalitionGraphError = McpCoalitionGraphErrors[keyof McpCoalitionGraphErrors];

export type McpCoalitionGraphResponses = {
    /**
     * OK
     */
    200: CoalitionGraphOutput;
};

export type McpCoalitionGraphResponse = McpCoalitionGraphResponses[keyof McpCoalitionGraphResponses];

export type McpCompareData = {
    body: CompareInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/compare';
};

export type McpCompareErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpCompareError = McpCompareErrors[keyof McpCompareErrors];

export type McpCompareResponses = {
    /**
     * OK
     */
    200: CompareOutput;
};

export type McpCompareResponse = McpCompareResponses[keyof McpCompareResponses];

export type McpDoctrineDetectData = {
    body: DoctrineDetectInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/doctrine_detect';
};

export type McpDoctrineDetectErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpDoctrineDetectError = McpDoctrineDetectErrors[keyof McpDoctrineDetectErrors];

export type McpDoctrineDetectResponses = {
    /**
     * OK
     */
    200: DoctrineDetectOutput;
};

export type McpDoctrineDetectResponse = McpDoctrineDetectResponses[keyof McpDoctrineDetectResponses];

export type McpDogmaEvalData = {
    body: DogmaEvalInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/dogma_eval';
};

export type McpDogmaEvalErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpDogmaEvalError = McpDogmaEvalErrors[keyof McpDogmaEvalErrors];

export type McpDogmaEvalResponses = {
    /**
     * OK
     */
    200: DogmaEvalOutput;
};

export type McpDogmaEvalResponse = McpDogmaEvalResponses[keyof McpDogmaEvalResponses];

export type McpEntityKillsData = {
    body: EntityKillsInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/entity_kills';
};

export type McpEntityKillsErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpEntityKillsError = McpEntityKillsErrors[keyof McpEntityKillsErrors];

export type McpEntityKillsResponses = {
    /**
     * OK
     */
    200: EntityKillsOutput;
};

export type McpEntityKillsResponse = McpEntityKillsResponses[keyof McpEntityKillsResponses];

export type McpEntityOverviewData = {
    body: EntityOverviewInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/entity_overview';
};

export type McpEntityOverviewErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpEntityOverviewError = McpEntityOverviewErrors[keyof McpEntityOverviewErrors];

export type McpEntityOverviewResponses = {
    /**
     * OK
     */
    200: EntityOverviewOutput;
};

export type McpEntityOverviewResponse = McpEntityOverviewResponses[keyof McpEntityOverviewResponses];

export type McpEntityTimelineData = {
    body: EntityTimelineInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/entity_timeline';
};

export type McpEntityTimelineErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpEntityTimelineError = McpEntityTimelineErrors[keyof McpEntityTimelineErrors];

export type McpEntityTimelineResponses = {
    /**
     * OK
     */
    200: EntityTimelineOutput;
};

export type McpEntityTimelineResponse = McpEntityTimelineResponses[keyof McpEntityTimelineResponses];

export type McpEntityTopData = {
    body: EntityTopInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/entity_top';
};

export type McpEntityTopErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpEntityTopError = McpEntityTopErrors[keyof McpEntityTopErrors];

export type McpEntityTopResponses = {
    /**
     * OK
     */
    200: EntityTopOutput;
};

export type McpEntityTopResponse = McpEntityTopResponses[keyof McpEntityTopResponses];

export type McpExpensiveLossesData = {
    body: ExpensiveLossesInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/expensive_losses';
};

export type McpExpensiveLossesErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpExpensiveLossesError = McpExpensiveLossesErrors[keyof McpExpensiveLossesErrors];

export type McpExpensiveLossesResponses = {
    /**
     * OK
     */
    200: ExpensiveLossesOutput;
};

export type McpExpensiveLossesResponse = McpExpensiveLossesResponses[keyof McpExpensiveLossesResponses];

export type McpFindBattlesData = {
    body: FindBattlesInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/find_battles';
};

export type McpFindBattlesErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpFindBattlesError = McpFindBattlesErrors[keyof McpFindBattlesErrors];

export type McpFindBattlesResponses = {
    /**
     * OK
     */
    200: FindBattlesOutput;
};

export type McpFindBattlesResponse = McpFindBattlesResponses[keyof McpFindBattlesResponses];

export type McpFitCompareData = {
    body: FitCompareInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/fit_compare';
};

export type McpFitCompareErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpFitCompareError = McpFitCompareErrors[keyof McpFitCompareErrors];

export type McpFitCompareResponses = {
    /**
     * OK
     */
    200: FitCompareOutput;
};

export type McpFitCompareResponse = McpFitCompareResponses[keyof McpFitCompareResponses];

export type McpFliesWithData = {
    body: CharacterIntelInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/flies_with';
};

export type McpFliesWithErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpFliesWithError = McpFliesWithErrors[keyof McpFliesWithErrors];

export type McpFliesWithResponses = {
    /**
     * OK
     */
    200: FliesWithOutput;
};

export type McpFliesWithResponse = McpFliesWithResponses[keyof McpFliesWithResponses];

export type McpGlobalPulseData = {
    body: GlobalPulseInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/global_pulse';
};

export type McpGlobalPulseErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpGlobalPulseError = McpGlobalPulseErrors[keyof McpGlobalPulseErrors];

export type McpGlobalPulseResponses = {
    /**
     * OK
     */
    200: GlobalPulseOutput;
};

export type McpGlobalPulseResponse = McpGlobalPulseResponses[keyof McpGlobalPulseResponses];

export type McpHuntedByData = {
    body: CharacterIntelInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/hunted_by';
};

export type McpHuntedByErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpHuntedByError = McpHuntedByErrors[keyof McpHuntedByErrors];

export type McpHuntedByResponses = {
    /**
     * OK
     */
    200: CharacterIntelOutput;
};

export type McpHuntedByResponse = McpHuntedByResponses[keyof McpHuntedByResponses];

export type McpHuntsInData = {
    body: CharacterIntelInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/hunts_in';
};

export type McpHuntsInErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpHuntsInError = McpHuntsInErrors[keyof McpHuntsInErrors];

export type McpHuntsInResponses = {
    /**
     * OK
     */
    200: HuntsInOutput;
};

export type McpHuntsInResponse = McpHuntsInResponses[keyof McpHuntsInResponses];

export type McpItemInfoData = {
    body: ItemInfoInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/item_info';
};

export type McpItemInfoErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpItemInfoError = McpItemInfoErrors[keyof McpItemInfoErrors];

export type McpItemInfoResponses = {
    /**
     * OK
     */
    200: ItemInfoOutput;
};

export type McpItemInfoResponse = McpItemInfoResponses[keyof McpItemInfoResponses];

export type McpKillmailData = {
    body: KillmailInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/killmail';
};

export type McpKillmailErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpKillmailError = McpKillmailErrors[keyof McpKillmailErrors];

export type McpKillmailResponses = {
    /**
     * OK
     */
    200: KillmailOutput;
};

export type McpKillmailResponse = McpKillmailResponses[keyof McpKillmailResponses];

export type McpKillmailFittingData = {
    body: KillmailFittingInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/killmail_fitting';
};

export type McpKillmailFittingErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpKillmailFittingError = McpKillmailFittingErrors[keyof McpKillmailFittingErrors];

export type McpKillmailFittingResponses = {
    /**
     * OK
     */
    200: KillmailFittingOutput;
};

export type McpKillmailFittingResponse = McpKillmailFittingResponses[keyof McpKillmailFittingResponses];

export type McpKillmailForensicsData = {
    body: KillmailInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/killmail_forensics';
};

export type McpKillmailForensicsErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpKillmailForensicsError = McpKillmailForensicsErrors[keyof McpKillmailForensicsErrors];

export type McpKillmailForensicsResponses = {
    /**
     * OK
     */
    200: KillmailForensicsOutput;
};

export type McpKillmailForensicsResponse = McpKillmailForensicsResponses[keyof McpKillmailForensicsResponses];

export type McpKillmailStoryData = {
    body: KillmailInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/killmail_story';
};

export type McpKillmailStoryErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpKillmailStoryError = McpKillmailStoryErrors[keyof McpKillmailStoryErrors];

export type McpKillmailStoryResponses = {
    /**
     * OK
     */
    200: KillmailStoryOutput;
};

export type McpKillmailStoryResponse = McpKillmailStoryResponses[keyof McpKillmailStoryResponses];

export type McpKillsWithData = {
    body: KillsWithInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/kills_with';
};

export type McpKillsWithErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpKillsWithError = McpKillsWithErrors[keyof McpKillsWithErrors];

export type McpKillsWithResponses = {
    /**
     * OK
     */
    200: KillsWithOutput;
};

export type McpKillsWithResponse = McpKillsWithResponses[keyof McpKillsWithResponses];

export type McpMeDossierData = {
    body: MeInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/me_dossier';
};

export type McpMeDossierErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpMeDossierError = McpMeDossierErrors[keyof McpMeDossierErrors];

export type McpMeDossierResponses = {
    /**
     * OK
     */
    200: DossierOutput;
};

export type McpMeDossierResponse = McpMeDossierResponses[keyof McpMeDossierResponses];

export type McpMeFliesWithData = {
    body: MeIntelInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/me_flies_with';
};

export type McpMeFliesWithErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpMeFliesWithError = McpMeFliesWithErrors[keyof McpMeFliesWithErrors];

export type McpMeFliesWithResponses = {
    /**
     * OK
     */
    200: FliesWithOutput;
};

export type McpMeFliesWithResponse = McpMeFliesWithResponses[keyof McpMeFliesWithResponses];

export type McpMeHuntedByData = {
    body: MeIntelInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/me_hunted_by';
};

export type McpMeHuntedByErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpMeHuntedByError = McpMeHuntedByErrors[keyof McpMeHuntedByErrors];

export type McpMeHuntedByResponses = {
    /**
     * OK
     */
    200: CharacterIntelOutput;
};

export type McpMeHuntedByResponse = McpMeHuntedByResponses[keyof McpMeHuntedByResponses];

export type McpMeHuntsInData = {
    body: MeIntelInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/me_hunts_in';
};

export type McpMeHuntsInErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpMeHuntsInError = McpMeHuntsInErrors[keyof McpMeHuntsInErrors];

export type McpMeHuntsInResponses = {
    /**
     * OK
     */
    200: HuntsInOutput;
};

export type McpMeHuntsInResponse = McpMeHuntsInResponses[keyof McpMeHuntsInResponses];

export type McpMeKillsData = {
    body: MeKillsInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/me_kills';
};

export type McpMeKillsErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpMeKillsError = McpMeKillsErrors[keyof McpMeKillsErrors];

export type McpMeKillsResponses = {
    /**
     * OK
     */
    200: EntityKillsOutput;
};

export type McpMeKillsResponse = McpMeKillsResponses[keyof McpMeKillsResponses];

export type McpMeKillsWithData = {
    body: MeKillsWithInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/me_kills_with';
};

export type McpMeKillsWithErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpMeKillsWithError = McpMeKillsWithErrors[keyof McpMeKillsWithErrors];

export type McpMeKillsWithResponses = {
    /**
     * OK
     */
    200: KillsWithOutput;
};

export type McpMeKillsWithResponse = McpMeKillsWithResponses[keyof McpMeKillsWithResponses];

export type McpMeOverviewData = {
    body: MeInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/me_overview';
};

export type McpMeOverviewErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpMeOverviewError = McpMeOverviewErrors[keyof McpMeOverviewErrors];

export type McpMeOverviewResponses = {
    /**
     * OK
     */
    200: EntityOverviewOutput;
};

export type McpMeOverviewResponse = McpMeOverviewResponses[keyof McpMeOverviewResponses];

export type McpMePreysOnData = {
    body: MeIntelInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/me_preys_on';
};

export type McpMePreysOnErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpMePreysOnError = McpMePreysOnErrors[keyof McpMePreysOnErrors];

export type McpMePreysOnResponses = {
    /**
     * OK
     */
    200: CharacterIntelOutput;
};

export type McpMePreysOnResponse = McpMePreysOnResponses[keyof McpMePreysOnResponses];

export type McpMeShipsUsedData = {
    body: MeShipsUsedInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/me_ships_used';
};

export type McpMeShipsUsedErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpMeShipsUsedError = McpMeShipsUsedErrors[keyof McpMeShipsUsedErrors];

export type McpMeShipsUsedResponses = {
    /**
     * OK
     */
    200: ShipsUsedOutput;
};

export type McpMeShipsUsedResponse = McpMeShipsUsedResponses[keyof McpMeShipsUsedResponses];

export type McpMeTimelineData = {
    body: MeTimelineInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/me_timeline';
};

export type McpMeTimelineErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpMeTimelineError = McpMeTimelineErrors[keyof McpMeTimelineErrors];

export type McpMeTimelineResponses = {
    /**
     * OK
     */
    200: EntityTimelineOutput;
};

export type McpMeTimelineResponse = McpMeTimelineResponses[keyof McpMeTimelineResponses];

export type McpMetaPulseData = {
    body: MetaPulseInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/meta_pulse';
};

export type McpMetaPulseErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpMetaPulseError = McpMetaPulseErrors[keyof McpMetaPulseErrors];

export type McpMetaPulseResponses = {
    /**
     * OK
     */
    200: MetaPulseOutput;
};

export type McpMetaPulseResponse = McpMetaPulseResponses[keyof McpMetaPulseResponses];

export type McpPilotEfficiencyData = {
    body: PilotEfficiencyInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/pilot_efficiency';
};

export type McpPilotEfficiencyErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpPilotEfficiencyError = McpPilotEfficiencyErrors[keyof McpPilotEfficiencyErrors];

export type McpPilotEfficiencyResponses = {
    /**
     * OK
     */
    200: PilotEfficiencyOutput;
};

export type McpPilotEfficiencyResponse = McpPilotEfficiencyResponses[keyof McpPilotEfficiencyResponses];

export type McpPreysOnData = {
    body: CharacterIntelInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/preys_on';
};

export type McpPreysOnErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpPreysOnError = McpPreysOnErrors[keyof McpPreysOnErrors];

export type McpPreysOnResponses = {
    /**
     * OK
     */
    200: CharacterIntelOutput;
};

export type McpPreysOnResponse = McpPreysOnResponses[keyof McpPreysOnResponses];

export type McpRouteDangerData = {
    body: RouteDangerInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/route_danger';
};

export type McpRouteDangerErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpRouteDangerError = McpRouteDangerErrors[keyof McpRouteDangerErrors];

export type McpRouteDangerResponses = {
    /**
     * OK
     */
    200: RouteDangerOutput;
};

export type McpRouteDangerResponse = McpRouteDangerResponses[keyof McpRouteDangerResponses];

export type McpSearchData = {
    body: SearchInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/search';
};

export type McpSearchErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpSearchError = McpSearchErrors[keyof McpSearchErrors];

export type McpSearchResponses = {
    /**
     * OK
     */
    200: SearchOutput;
};

export type McpSearchResponse = McpSearchResponses[keyof McpSearchResponses];

export type McpShipCompareData = {
    body: ShipCompareInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/ship_compare';
};

export type McpShipCompareErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpShipCompareError = McpShipCompareErrors[keyof McpShipCompareErrors];

export type McpShipCompareResponses = {
    /**
     * OK
     */
    200: ShipCompareOutput;
};

export type McpShipCompareResponse = McpShipCompareResponses[keyof McpShipCompareResponses];

export type McpShipInfoData = {
    body: ShipInfoInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/ship_info';
};

export type McpShipInfoErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpShipInfoError = McpShipInfoErrors[keyof McpShipInfoErrors];

export type McpShipInfoResponses = {
    /**
     * OK
     */
    200: ShipInfoOutput;
};

export type McpShipInfoResponse = McpShipInfoResponses[keyof McpShipInfoResponses];

export type McpShipsUsedData = {
    body: ShipsUsedInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/ships_used';
};

export type McpShipsUsedErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpShipsUsedError = McpShipsUsedErrors[keyof McpShipsUsedErrors];

export type McpShipsUsedResponses = {
    /**
     * OK
     */
    200: ShipsUsedOutput;
};

export type McpShipsUsedResponse = McpShipsUsedResponses[keyof McpShipsUsedResponses];

export type McpSystemInfoData = {
    body: SystemInfoInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/system_info';
};

export type McpSystemInfoErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpSystemInfoError = McpSystemInfoErrors[keyof McpSystemInfoErrors];

export type McpSystemInfoResponses = {
    /**
     * OK
     */
    200: SystemInfoOutput;
};

export type McpSystemInfoResponse = McpSystemInfoResponses[keyof McpSystemInfoResponses];

export type McpSystemPulseData = {
    body: SystemPulseInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/system_pulse';
};

export type McpSystemPulseErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpSystemPulseError = McpSystemPulseErrors[keyof McpSystemPulseErrors];

export type McpSystemPulseResponses = {
    /**
     * OK
     */
    200: SystemPulseOutput;
};

export type McpSystemPulseResponse = McpSystemPulseResponses[keyof McpSystemPulseResponses];

export type McpWarReportData = {
    body: WarReportInputWritable;
    path?: never;
    query?: never;
    url: '/mcp/tools/war_report';
};

export type McpWarReportErrors = {
    /**
     * Error
     */
    default: ErrorModel;
};

export type McpWarReportError = McpWarReportErrors[keyof McpWarReportErrors];

export type McpWarReportResponses = {
    /**
     * OK
     */
    200: WarReportOutput;
};

export type McpWarReportResponse = McpWarReportResponses[keyof McpWarReportResponses];

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
        user: {
            allianceId: number | null;
            allianceName: string | null;
            characterId: number;
            characterName: string;
            characterOwnerHash?: string;
            corporationId: number | null;
            corporationName: string | null;
            isAdmin: boolean;
            lastSeenNotificationId: number;
            settings?: {
                boards?: {
                    dismissed: Array<string>;
                    pinned: Array<string>;
                };
                /**
                 * Default tab keyed by page type.
                 */
                defaultTabs?: {
                    [key: string]: unknown;
                };
                /**
                 * User-selected theme settings.
                 */
                theme?: {
                    [key: string]: unknown;
                };
            };
        } | null;
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
        atCapacity: boolean;
        authenticated: boolean;
        boards: Array<{
            host: string;
            key: string;
            name: string;
            pinned: boolean;
            tracked: boolean;
            url: string;
        }>;
        current: {
            key: string;
            listed: boolean;
            name: string;
        } | null;
    };
};

export type AccountBoardsResponse = AccountBoardsResponses[keyof AccountBoardsResponses];

export type MyCommentsData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Identifier cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        cursor?: number;
    };
    url: '/me/comments';
};

export type MyCommentsResponses = {
    /**
     * OK
     */
    200: {
        comments: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            body_html: string;
            body_md: string;
            character_id: number;
            character_name: string;
            corporation_id: number;
            corporation_name: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            deleted_at: string | null;
            deleted_by: number | null;
            depth: number;
            domain_id: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            edited_at: string | null;
            flagged: boolean;
            id: number;
            moderation_status: number;
            parent_id: number | null;
            reply_count?: number;
            reports_count: number;
            root_id: number | null;
            target_id: number;
            target_slug: string | null;
            target_type: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
            visibility: number;
        }>;
        nextCursor: number | null;
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
        ok: boolean;
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
        alliance: {
            canEdit: boolean;
            ceo_id?: number | null;
            ceo_name?: string | null;
            custom_description: string | null;
            custom_description_format: string;
            esi_description?: string | null;
            executor_ceo_id?: number | null;
            executor_ceo_name?: string | null;
            executor_corporation_id?: number | null;
            id: number;
            name: string;
            pending_submission: {
                body: string;
                body_format: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                submitted_at: string;
            } | null;
            ticker?: string;
        } | null;
        character: {
            canEdit: boolean;
            ceo_id?: number | null;
            ceo_name?: string | null;
            custom_description: string | null;
            custom_description_format: string;
            esi_description?: string | null;
            executor_ceo_id?: number | null;
            executor_ceo_name?: string | null;
            executor_corporation_id?: number | null;
            id: number;
            name: string;
            pending_submission: {
                body: string;
                body_format: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                submitted_at: string;
            } | null;
            ticker?: string;
        };
        corporation: {
            canEdit: boolean;
            ceo_id?: number | null;
            ceo_name?: string | null;
            custom_description: string | null;
            custom_description_format: string;
            esi_description?: string | null;
            executor_ceo_id?: number | null;
            executor_ceo_name?: string | null;
            executor_corporation_id?: number | null;
            id: number;
            name: string;
            pending_submission: {
                body: string;
                body_format: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                submitted_at: string;
            } | null;
            ticker?: string;
        } | null;
    };
};

export type AccountDescriptionsResponse = AccountDescriptionsResponses[keyof AccountDescriptionsResponses];

export type AccountDescriptionUpdateData = {
    body: {
        /**
         * Description text.
         */
        description: string;
        /**
         * Which description to write.
         */
        entity: 'character' | 'corporation' | 'alliance';
        /**
         * How to interpret the text.
         */
        format: 'markdown' | 'eve_html';
    };
    path?: never;
    query?: never;
    url: '/me/descriptions';
};

export type AccountDescriptionUpdateResponses = {
    /**
     * OK
     */
    200: {
        entity: string;
        entity_id: number;
        ok: boolean;
        queue_id?: number;
        status: string;
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
        domains: Array<{
            active?: boolean;
            backgrounds?: Array<{
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            }>;
            bannerAsset?: {
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            } | null;
            campaign_policy?: number;
            campaigns?: Array<{
                campaign_id?: string;
                created_by_character_id?: number;
                description?: string | null;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                end_time?: string;
                estimated_killmails?: number | null;
                name?: string;
                public_on_domain?: boolean;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                start_time?: string;
                status?: number;
                visibility?: number;
                [key: string]: unknown;
            }>;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            custom_hostname?: string | null;
            entities?: Array<{
                id: number;
                name?: string;
                type: string;
            }>;
            id?: number;
            logoAsset?: {
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            } | null;
            navbar_links?: Array<{
                children?: Array<{
                    items: Array<{
                        external?: boolean;
                        href: string;
                        icon?: string;
                        label: string;
                    }>;
                    label?: string;
                }>;
                external?: boolean;
                href: string;
                icon?: string;
                label: string;
            }>;
            site_description?: string | null;
            site_name?: string | null;
            subdomain?: string;
            theme?: {
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
                [key: string]: unknown;
            };
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            user_id?: number;
            widgets?: {
                columnRatio: string;
                left: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
                right: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
                top: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
            };
            [key: string]: unknown;
        }>;
    };
};

export type DomainsMineResponse = DomainsMineResponses[keyof DomainsMineResponses];

export type DomainCreateData = {
    body: {
        entities: Array<{
            id: number;
            name: string;
            type: 'character' | 'corporation' | 'alliance';
        }>;
        navbar_links?: Array<{
            children?: Array<SiteDomainNavbarGroup>;
            external?: boolean;
            href: string;
            icon?: string;
            label: string;
        }>;
        site_description?: string | null;
        site_name?: string | null;
        subdomain: string;
        theme?: SiteDomainTheme;
        widgets?: SiteDomainWidgets;
    };
    path?: never;
    query?: never;
    url: '/me/domains';
};

export type DomainCreateResponses = {
    /**
     * OK
     */
    200: {
        domain: {
            active?: boolean;
            backgrounds?: Array<{
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            }>;
            bannerAsset?: {
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            } | null;
            campaign_policy?: number;
            campaigns?: Array<{
                campaign_id?: string;
                created_by_character_id?: number;
                description?: string | null;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                end_time?: string;
                estimated_killmails?: number | null;
                name?: string;
                public_on_domain?: boolean;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                start_time?: string;
                status?: number;
                visibility?: number;
                [key: string]: unknown;
            }>;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            custom_hostname?: string | null;
            entities?: Array<{
                id: number;
                name?: string;
                type: string;
            }>;
            id?: number;
            logoAsset?: {
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            } | null;
            navbar_links?: Array<{
                children?: Array<{
                    items: Array<{
                        external?: boolean;
                        href: string;
                        icon?: string;
                        label: string;
                    }>;
                    label?: string;
                }>;
                external?: boolean;
                href: string;
                icon?: string;
                label: string;
            }>;
            site_description?: string | null;
            site_name?: string | null;
            subdomain?: string;
            theme?: {
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
                [key: string]: unknown;
            };
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            user_id?: number;
            widgets?: {
                columnRatio: string;
                left: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
                right: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
                top: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
            };
            [key: string]: unknown;
        };
    };
};

export type DomainCreateResponse = DomainCreateResponses[keyof DomainCreateResponses];

export type DomainSubdomainCheckData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Subdomain to test.
         */
        subdomain?: string;
    };
    url: '/me/domains/check-subdomain';
};

export type DomainSubdomainCheckResponses = {
    /**
     * OK
     */
    200: {
        available: boolean;
        reason?: string;
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
        deleted: boolean;
    };
};

export type DomainDeleteResponse = DomainDeleteResponses[keyof DomainDeleteResponses];

export type DomainUpdateData = {
    body: {
        active?: boolean;
        campaign_ids?: Array<string>;
        campaign_policy?: 0 | 1;
        campaign_public_ids?: Array<string>;
        entities?: Array<{
            id: number;
            name: string;
            type: 'character' | 'corporation' | 'alliance';
        }>;
        navbar_links?: Array<{
            children?: Array<SiteDomainNavbarGroup>;
            external?: boolean;
            href: string;
            icon?: string;
            label: string;
        }>;
        site_description?: string | null;
        site_name?: string | null;
        theme?: SiteDomainTheme;
        widgets?: SiteDomainWidgets;
    };
    path?: never;
    query?: never;
    url: '/me/domains/{id}';
};

export type DomainUpdateResponses = {
    /**
     * OK
     */
    200: {
        domain: {
            active?: boolean;
            backgrounds?: Array<{
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            }>;
            bannerAsset?: {
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            } | null;
            campaign_policy?: number;
            campaigns?: Array<{
                campaign_id?: string;
                created_by_character_id?: number;
                description?: string | null;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                end_time?: string;
                estimated_killmails?: number | null;
                name?: string;
                public_on_domain?: boolean;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                start_time?: string;
                status?: number;
                visibility?: number;
                [key: string]: unknown;
            }>;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            custom_hostname?: string | null;
            entities?: Array<{
                id: number;
                name?: string;
                type: string;
            }>;
            id?: number;
            logoAsset?: {
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            } | null;
            navbar_links?: Array<{
                children?: Array<{
                    items: Array<{
                        external?: boolean;
                        href: string;
                        icon?: string;
                        label: string;
                    }>;
                    label?: string;
                }>;
                external?: boolean;
                href: string;
                icon?: string;
                label: string;
            }>;
            site_description?: string | null;
            site_name?: string | null;
            subdomain?: string;
            theme?: {
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
                [key: string]: unknown;
            };
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            user_id?: number;
            widgets?: {
                columnRatio: string;
                left: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
                right: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
                top: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
            };
            [key: string]: unknown;
        };
    };
};

export type DomainUpdateResponse = DomainUpdateResponses[keyof DomainUpdateResponses];

export type DomainAssetsDeleteTypeData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Asset slot to clear, for example `banner` or `logo`.
         */
        type?: string;
    };
    url: '/me/domains/{id}/assets';
};

export type DomainAssetsDeleteTypeResponses = {
    /**
     * OK
     */
    200: {
        success: boolean;
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
        assetId: number;
        message: string;
        status: string;
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
        success: boolean;
    };
};

export type DomainAssetDeleteResponse = DomainAssetDeleteResponses[keyof DomainAssetDeleteResponses];

export type DomainCampaignSearchData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Campaign name search.
         */
        q?: string;
    };
    url: '/me/domains/{id}/campaigns';
};

export type DomainCampaignSearchResponses = {
    /**
     * OK
     */
    200: {
        campaigns: Array<{
            campaign_id?: string;
            created_by_character_id?: number;
            description?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            end_time?: string;
            estimated_killmails?: number | null;
            name?: string;
            public_on_domain?: boolean;
            /**
             * UTC timestamp with millisecond precision.
             */
            start_time?: string;
            status?: number;
            visibility?: number;
            [key: string]: unknown;
        }>;
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
        rateLimit: {
            request_count: number;
        };
        responseTime: {
            avg_ms: number | null;
            p95_ms: number | null;
        };
        volumeByHour: Array<{
            errors: number;
            hour: string;
            new_items: number;
            total: number;
        }>;
    };
};

export type AccountEsiResponse = AccountEsiResponses[keyof AccountEsiResponses];

export type AccountEsiLogsData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Page number, counted from 1.
         */
        page?: number;
        /**
         * Match the recorded request source exactly.
         */
        source?: string;
        /**
         * Restrict to successful or failed requests.
         */
        status?: 'success' | 'error';
        /**
         * Restrict to one ESI endpoint family, for example `killmails`.
         */
        endpoint_type?: string;
        /**
         * Return log rows below this log ID.
         */
        after_id?: number;
    };
    url: '/me/esi/logs';
};

export type AccountEsiLogsResponses = {
    /**
     * OK
     */
    200: {
        limit?: number;
        newRows?: boolean;
        page?: number;
        pages?: number;
        rows: Array<{
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            endpoint: string;
            endpoint_action: string;
            endpoint_type: string;
            error_message: string | null;
            id: number;
            items_returned: number | null;
            method: string;
            new_items: number | null;
            request_duration_ms: number | null;
            source: string;
            status_code: number | null;
            success: boolean;
        }>;
        sources?: Array<string>;
        total?: number;
    };
};

export type AccountEsiLogsResponse = AccountEsiLogsResponses[keyof AccountEsiLogsResponses];

export type AccountNotificationReadCursorData = {
    body: {
        /**
         * An integer. A numeric string is accepted for compatibility.
         */
        id: number | string;
    };
    path?: never;
    query?: never;
    url: '/me/notifications/read-cursor';
};

export type AccountNotificationReadCursorResponses = {
    /**
     * OK
     */
    200: {
        lastSeenNotificationId: number;
    };
};

export type AccountNotificationReadCursorResponse = AccountNotificationReadCursorResponses[keyof AccountNotificationReadCursorResponses];

export type AccountNotificationRepliesData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Return replies with an ID above this one.
         */
        since?: number;
    };
    url: '/me/notifications/replies';
};

export type AccountNotificationRepliesResponses = {
    /**
     * OK
     */
    200: {
        highestId: number;
        replies: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            body_html: string;
            character_id: number;
            character_name: string;
            corporation_id: number;
            corporation_name: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            id: number;
            parent_comment_id: number;
            parent_id: number | null;
            parent_snippet: string;
            root_id: number | null;
            target_id: number;
            target_type: number;
        }>;
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
        account: {
            characterId: number;
            characterName: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            createdAt: string | null;
            isAdmin: boolean;
            /**
             * UTC timestamp with millisecond precision.
             */
            lastLogin: string | null;
        };
        esiStats: {
            errors_24h?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_request?: string | null;
            new_items_24h?: number;
            requests_24h?: number;
            total_errors?: number;
            total_new_items?: number;
            total_requests?: number;
        };
        esiToken: {
            /**
             * UTC timestamp with millisecond precision.
             */
            lastFetched: string | null;
            scopeCount: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            tokenExpiry: string | null;
        } | null;
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
        boards?: {
            dismissed: Array<string>;
            pinned: Array<string>;
        };
        /**
         * Default tab keyed by page type.
         */
        defaultTabs?: {
            [key: string]: unknown;
        };
        /**
         * User-selected theme settings.
         */
        theme?: {
            [key: string]: unknown;
        };
    };
};

export type AccountPreferencesResponse = AccountPreferencesResponses[keyof AccountPreferencesResponses];

export type AccountPreferencesUpdateData = {
    body: {
        boards?: AccountBoardsDocument;
        defaultTabs?: {
            [key: string]: string;
        };
        theme?: {
            [key: string]: string;
        };
    };
    path?: never;
    query?: never;
    url: '/me/preferences';
};

export type AccountPreferencesUpdateResponses = {
    /**
     * OK
     */
    200: {
        preferences: {
            boards?: {
                dismissed: Array<string>;
                pinned: Array<string>;
            };
            /**
             * Default tab keyed by page type.
             */
            defaultTabs?: {
                [key: string]: unknown;
            };
            /**
             * User-selected theme settings.
             */
            theme?: {
                [key: string]: unknown;
            };
        };
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
        success: boolean;
    };
};

export type SessionDeleteResponse = SessionDeleteResponses[keyof SessionDeleteResponses];

export type OtherSessionsRevokeData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Session ID to keep. Every other session is revoked.
         */
        except?: string;
    };
    url: '/me/sessions';
};

export type OtherSessionsRevokeResponses = {
    /**
     * OK
     */
    200: {
        revoked: number;
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
        sessions: Array<{
            browser: string;
            countryCode: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            createdAt: string;
            current: boolean;
            device: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            expiresAt: string;
            id: number;
            ipAddress: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            lastSeenAt: string;
            operatingSystem: string;
        }>;
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
        current: boolean;
        revoked: boolean;
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
        account: {
            characterId: number;
            characterName: string;
            characterOwnerHash: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            createdAt: string | null;
            isAdmin: boolean;
            /**
             * UTC timestamp with millisecond precision.
             */
            lastLogin: string | null;
        };
        esiToken: {
            disabled: boolean;
            effectiveScopes: Array<string>;
            /**
             * UTC timestamp with millisecond precision.
             */
            lastFetched: string | null;
            revokedScopes: Array<string>;
            scopeCount: number;
            scopes: Array<string>;
            /**
             * UTC timestamp with millisecond precision.
             */
            tokenExpiry: string | null;
        } | null;
        preferences: {
            boards?: {
                dismissed: Array<string>;
                pinned: Array<string>;
            };
            /**
             * Default tab keyed by page type.
             */
            defaultTabs?: {
                [key: string]: unknown;
            };
            /**
             * User-selected theme settings.
             */
            theme?: {
                [key: string]: unknown;
            };
        };
    };
};

export type MeSettingsResponse = MeSettingsResponses[keyof MeSettingsResponses];

export type WalletAccountData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Page number, counted from 1.
         */
        page?: number;
    };
    url: '/me/wallet';
};

export type WalletAccountResponses = {
    /**
     * OK
     */
    200: {
        availableBalance: string;
        balance: string;
        character: {
            character_id: number;
            character_name: string;
        };
        corporation: {
            corporation_id: number;
            name: string;
            ticker: string;
        };
        /**
         * UTC timestamp with millisecond precision.
         */
        depositsEnabledAt: string | null;
        hasMore: boolean;
        /**
         * UTC timestamp with millisecond precision.
         */
        lastSynced: string | null;
        page: number;
        pageSize: number;
        reservations: Array<{
            amount?: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            description?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            expires_at?: string;
            external_reference?: string;
            id?: number;
            transaction_type?: number;
            [key: string]: unknown;
        }>;
        reservedBalance: string;
        totalBalance: string;
        transactions: Array<{
            amount?: string;
            balance_after?: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            description?: string | null;
            id?: number;
            type?: number;
            [key: string]: unknown;
        }>;
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
        availableBalance: string;
        balance: string;
        reservedBalance: string;
        totalBalance: string;
    };
};

export type WalletAccountBalanceResponse = WalletAccountBalanceResponses[keyof WalletAccountBalanceResponses];

export type NotificationMarkReadCompatData = {
    body: {
        /**
         * An integer. A numeric string is accepted for compatibility.
         */
        id: number | string;
    };
    path?: never;
    query?: never;
    url: '/notifications/mark-read';
};

export type NotificationMarkReadCompatResponses = {
    /**
     * OK
     */
    200: {
        lastSeenNotificationId: number;
    };
};

export type NotificationMarkReadCompatResponse = NotificationMarkReadCompatResponses[keyof NotificationMarkReadCompatResponses];

export type NotificationRepliesCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Return replies with an ID above this one.
         */
        since?: number;
    };
    url: '/notifications/replies';
};

export type NotificationRepliesCompatResponses = {
    /**
     * OK
     */
    200: {
        highestId: number;
        replies: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            body_html: string;
            character_id: number;
            character_name: string;
            corporation_id: number;
            corporation_name: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            id: number;
            parent_comment_id: number;
            parent_id: number | null;
            parent_snippet: string;
            root_id: number | null;
            target_id: number;
            target_type: number;
        }>;
    };
};

export type NotificationRepliesCompatResponse = NotificationRepliesCompatResponses[keyof NotificationRepliesCompatResponses];

export type BulkPricesData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Comma-separated inventory type IDs. Omit for an empty price map.
         */
        types?: string;
    };
    url: '/prices/bulk';
};

export type BulkPricesResponses = {
    /**
     * OK
     */
    200: {
        prices: {
            [key: string]: number;
        };
    };
};

export type BulkPricesResponse = BulkPricesResponses[keyof BulkPricesResponses];

export type ReadyData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/ready';
};

export type ReadyResponses = {
    /**
     * OK
     */
    200: ReadyResponse;
};

export type ReadyResponse2 = ReadyResponses[keyof ReadyResponses];

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
        constellations: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            constellation_id: number;
            constellation_name: string;
            faction_id: number | null;
            faction_name: string | null;
            system_count: number;
        }>;
        region: {
            constellation_count: number;
            description: string | null;
            faction_id: number | null;
            faction_name: string | null;
            name: string;
            region_id: number;
            system_count: number;
        };
        sovDistribution: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            faction_id: number | null;
            faction_name: string | null;
            system_count: number;
        }>;
        stats: {
            kills: number;
            npc_kills: number;
            pod_kills: number;
            total_value: number;
        };
        topSystems: Array<{
            kills: number;
            security: number;
            solar_system_id: number;
            system_name: string;
            total_value: number;
        }>;
    };
};

export type RegionCompatResponse = RegionCompatResponses[keyof RegionCompatResponses];

export type RegionKilllistCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
        /**
         * Page number for offset paging. Leave at 0 to page by cursor.
         */
        page?: number;
    };
    url: '/region/{id}/killlist';
};

export type RegionKilllistCompatResponses = {
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
        totalPages?: number;
    };
};

export type RegionKilllistCompatResponse = RegionKilllistCompatResponses[keyof RegionKilllistCompatResponses];

export type RegionMostValuableCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Restrict the losses to one category of hull.
         */
        dataType?: 'most_valuable_kills' | 'most_valuable_ships' | 'most_valuable_structures';
        /**
         * Size of the trailing window, in days.
         */
        days?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/region/{id}/most-valuable';
};

export type RegionMostValuableCompatResponses = {
    /**
     * OK
     */
    200: {
        entries: Array<{
            killmail_hash: string;
            killmail_id: number;
            ship_name: string;
            ship_type_id: number;
            total_value: number;
            victim_alliance_name: string | null;
            victim_character_id: number | null;
            victim_character_name: string | null;
            victim_corporation_name: string | null;
        }>;
    };
};

export type RegionMostValuableCompatResponse = RegionMostValuableCompatResponses[keyof RegionMostValuableCompatResponses];

export type ResolveData = {
    body: {
        /**
         * Exact entity names to resolve. Matching is case-sensitive and exact; use /search for fuzzy lookup.
         */
        names: Array<string>;
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
         * Raw directional scan text.
         */
        dscan: string;
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
        hash: string;
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
        grouped: {
            [key: string]: {
                categoryId: number | null;
                groups: {
                    [key: string]: {
                        groupId: number | null;
                        types: Array<{
                            count: number;
                            typeId: number | null;
                            typeName: string;
                        }>;
                    };
                };
            };
        };
        totalCount: number;
        uniqueTypes: number;
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
        grouped: {
            [key: string]: {
                categoryId: number | null;
                groups: {
                    [key: string]: {
                        groupId: number | null;
                        types: Array<{
                            count: number;
                            typeId: number | null;
                            typeName: string;
                        }>;
                    };
                };
            };
        };
        totalCount: number;
        uniqueTypes: number;
    };
};

export type DscanGetResponse = DscanGetResponses[keyof DscanGetResponses];

export type LocalscanSaveData = {
    body: {
        /**
         * Character names from the local scan.
         */
        names: Array<string>;
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
        hash: string;
    };
};

export type LocalscanSaveResponse = LocalscanSaveResponses[keyof LocalscanSaveResponses];

export type LocalscanAnalyzeData = {
    body: Array<string>;
    path?: never;
    query?: never;
    url: '/scans/local/analyze';
};

export type LocalscanAnalyzeResponses = {
    /**
     * OK
     */
    200: {
        alliances: {
            [key: string]: {
                corporations: {
                    [key: string]: {
                        characters: Array<{
                            characterId: number;
                            kills: number;
                            name: string;
                        }>;
                        name: string;
                        ticker: string;
                    };
                };
                name: string;
                ticker: string;
            };
        };
        corporations: {
            [key: string]: {
                characters: Array<{
                    characterId: number;
                    kills: number;
                    name: string;
                }>;
                name: string;
                ticker: string;
            };
        };
        totalCharacters: number;
        totalDangerous: number;
        unresolved: Array<string>;
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
        alliances: {
            [key: string]: {
                corporations: {
                    [key: string]: {
                        characters: Array<{
                            characterId: number;
                            kills: number;
                            name: string;
                        }>;
                        name: string;
                        ticker: string;
                    };
                };
                name: string;
                ticker: string;
            };
        };
        corporations: {
            [key: string]: {
                characters: Array<{
                    characterId: number;
                    kills: number;
                    name: string;
                }>;
                name: string;
                ticker: string;
            };
        };
        totalCharacters: number;
        totalDangerous: number;
        unresolved: Array<string>;
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
    query?: {
        /**
         * Send `false` to include unpublished rows. Anything else keeps published rows only.
         */
        published?: 'true' | 'false';
    };
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
    query?: {
        /**
         * Restrict to one region.
         */
        region_id?: number;
    };
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
    query?: {
        /**
         * Send `false` to include unpublished rows. Anything else keeps published rows only.
         */
        published?: 'true' | 'false';
        /**
         * Restrict to one inventory category.
         */
        category_id?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
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
    query?: {
        /**
         * Restrict to one parent market group.
         */
        parent_id?: number;
    };
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
    query?: {
        /**
         * Market region to price against. Defaults to The Forge (10000002).
         */
        region_id?: number;
        /**
         * Most recent daily rows to return.
         */
        limit?: number;
    };
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
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
        /**
         * Descending cursor, walking newest to oldest. Pass the previous response's pagination cursor to fetch the next page. Mutually exclusive with `after`, which it overrides.
         */
        before?: number;
    };
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
    query?: {
        /**
         * Restrict to one alliance.
         */
        alliance_id?: number;
        /**
         * Restrict to one faction.
         */
        faction_id?: number;
    };
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
    query?: {
        /**
         * Restrict to one solar system.
         */
        solar_system_id?: number;
        /**
         * Restrict to one region.
         */
        region_id?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
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
    query?: {
        /**
         * Restrict to one solar system.
         */
        solar_system_id?: number;
        /**
         * Restrict to one region.
         */
        region_id?: number;
        /**
         * Restrict to one owning corporation.
         */
        owner_id?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
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
    query?: {
        /**
         * Case-insensitive system name prefix to match.
         */
        name?: string;
        /**
         * Restrict to one region.
         */
        region_id?: number;
        /**
         * Restrict to one constellation.
         */
        constellation_id?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
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
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
        /**
         * Descending cursor, walking newest to oldest. Pass the previous response's pagination cursor to fetch the next page. Mutually exclusive with `after`, which it overrides.
         */
        before?: number;
    };
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
    query?: {
        /**
         * Case-insensitive name prefix to match.
         */
        name?: string;
        /**
         * Send `false` to include unpublished rows. Anything else keeps published rows only.
         */
        published?: 'true' | 'false';
        /**
         * Restrict to one inventory group.
         */
        group_id?: number;
        /**
         * Restrict to one inventory category.
         */
        category_id?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
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
    query: {
        /**
         * Search text. Required.
         */
        q: string;
        /**
         * Comma-separated entity kinds to search: `character`, `corporation`, `alliance`, `faction`, `ship`, `shipgroup`, `system`, `region`, `constellation`. Omit to search all of them.
         */
        type?: string;
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
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
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
    url: '/ship/{id}/killlist';
};

export type ShipKilllistCompatResponses = {
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
        totalPages?: number;
    };
};

export type ShipKilllistCompatResponse = ShipKilllistCompatResponses[keyof ShipKilllistCompatResponses];

export type ShipFittingsData = {
    body?: never;
    path: {
        id: string;
    };
    query?: {
        /**
         * Comma-separated module type IDs the fit must contain.
         */
        modules?: string;
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
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
    200: Array<{
        changefreq: string;
        /**
         * UTC timestamp with millisecond precision.
         */
        lastmod?: string;
        loc: string;
        priority: number;
    }>;
};

export type SitemapResponse = SitemapResponses[keyof SitemapResponses];

export type GlobalStatsData = {
    body?: never;
    path?: never;
    query: {
        /**
         * Which leaderboard to build. Required.
         */
        dataType: 'characters' | 'corporations' | 'alliances' | 'factions' | 'ships' | 'systems' | 'regions' | 'isk_destroyers_chars' | 'isk_destroyers_corps' | 'isk_destroyers_alliances' | 'solo_killers' | 'top_points' | 'dangerous_systems' | 'deadliest_regions' | 'most_used_ships' | 'most_destroyed_ships' | 'biggest_losers' | 'pirate_characters' | 'carebear_characters' | 'most_valuable_kills' | 'most_valuable_ships' | 'most_valuable_structures';
        /**
         * Window in days. Values below 1 select the realtime hourly window instead.
         */
        days?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
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
    query: {
        /**
         * Which ranking to build. Required.
         */
        section: 'largest' | 'security' | 'growth' | 'newest' | 'achievements';
        /**
         * Which entity the ranking covers.
         */
        entityType?: 'character' | 'corporation' | 'alliance';
        /**
         * Security band. Only read when `section` is `security`.
         */
        rank?: 'pirate' | 'carebear';
        /**
         * Growth direction. Only read when `section` is `growth`.
         */
        direction?: 'growing' | 'shrinking';
        /**
         * Growth window in days. Only read when `section` is `growth`.
         */
        days?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/stats/rankings';
};

export type StatsRankingsResponses = {
    /**
     * OK
     */
    200: {
        entries: Array<{
            achievement_points?: number;
            completed_count?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            date_founded?: string;
            delta_1d?: number;
            delta_30d?: number;
            delta_7d?: number;
            growth?: number;
            id?: number;
            member_count?: number;
            name?: string;
            security_status?: number;
            type?: string;
            weighted_score?: number;
            [key: string]: unknown;
        }>;
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
        activity: {
            history: Array<{
                npc_kills: number;
                pod_kills: number;
                ship_jumps: number;
                ship_kills: number;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                timestamp?: string;
            }>;
            latest: {
                npc_kills: number;
                pod_kills: number;
                ship_jumps: number;
                ship_kills: number;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                timestamp?: string;
            } | null;
            summary_24h: {
                npc_kills: number;
                pod_kills: number;
                ship_jumps: number;
                ship_kills: number;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                timestamp?: string;
            } | null;
        };
        celestialList: Array<{
            category: string;
            group_id: number;
            item_id: number;
            item_name: string;
            type_id: number;
            type_name: string | null;
        }>;
        celestials: {
            [key: string]: number;
        };
        connections: Array<{
            is_regional: boolean;
            region_id: number;
            security: number;
            system_name: string;
            to_solar_system_id: number;
        }>;
        sovereignty: {
            alliance_id: number | null;
            alliance_name: string | null;
            corporation_id: number | null;
            corporation_name: string | null;
            faction_id: number | null;
            faction_name: string | null;
        } | null;
        sovereigntyHistory: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            corporation_id: number | null;
            corporation_name: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            date: string;
            faction_id: number | null;
            faction_name: string | null;
        }>;
        stations: Array<{
            corporation_id: number;
            corporation_name: string | null;
            operation_name: string | null;
            station_id: number;
            station_name: string;
            type_id: number;
        }>;
        stats: {
            kills: number;
            npc_kills: number;
            pod_kills: number;
            total_value: number;
        };
        structures: Array<{
            is_market: boolean;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_seen: string | null;
            name: string;
            owner_id: number;
            owner_name: string | null;
            structure_id: number;
            type_id: number;
            type_name: string | null;
        }>;
        system: {
            border: boolean;
            constellation_id: number;
            constellation_name: string;
            corridor: boolean;
            faction_id: number | null;
            fringe: boolean;
            hub: boolean;
            international: boolean;
            region_id: number;
            region_name: string;
            regional: boolean;
            security: number;
            security_class: string | null;
            solar_system_id: number;
            sun_type_id: number | null;
            sun_type_name: string | null;
            system_name: string;
        };
    };
};

export type SystemCompatResponse = SystemCompatResponses[keyof SystemCompatResponses];

export type SystemKilllistCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
        /**
         * Page number for offset paging. Leave at 0 to page by cursor.
         */
        page?: number;
    };
    url: '/system/{id}/killlist';
};

export type SystemKilllistCompatResponses = {
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
        totalPages?: number;
    };
};

export type SystemKilllistCompatResponse = SystemKilllistCompatResponses[keyof SystemKilllistCompatResponses];

export type SystemMostValuableCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Restrict the losses to one category of hull.
         */
        dataType?: 'most_valuable_kills' | 'most_valuable_ships' | 'most_valuable_structures';
        /**
         * Size of the trailing window, in days.
         */
        days?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/system/{id}/most-valuable';
};

export type SystemMostValuableCompatResponses = {
    /**
     * OK
     */
    200: {
        entries: Array<{
            killmail_hash: string;
            killmail_id: number;
            ship_name: string;
            ship_type_id: number;
            total_value: number;
            victim_alliance_name: string | null;
            victim_character_id: number | null;
            victim_character_name: string | null;
            victim_corporation_name: string | null;
        }>;
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
        grouped: {
            [key: string]: {
                categoryId: number | null;
                groups: {
                    [key: string]: {
                        groupId: number | null;
                        types: Array<{
                            count: number;
                            typeId: number | null;
                            typeName: string;
                        }>;
                    };
                };
            };
        };
        totalCount: number;
        uniqueTypes: number;
    };
};

export type DscanAnalyzeLegacyResponse = DscanAnalyzeLegacyResponses[keyof DscanAnalyzeLegacyResponses];

export type DscanSaveLegacyData = {
    body: {
        /**
         * Raw directional scan text.
         */
        dscan: string;
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
        hash: string;
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
        grouped: {
            [key: string]: {
                categoryId: number | null;
                groups: {
                    [key: string]: {
                        groupId: number | null;
                        types: Array<{
                            count: number;
                            typeId: number | null;
                            typeName: string;
                        }>;
                    };
                };
            };
        };
        totalCount: number;
        uniqueTypes: number;
    };
};

export type DscanGetLegacyResponse = DscanGetLegacyResponses[keyof DscanGetLegacyResponses];

export type LocalscanAnalyzeLegacyData = {
    body: Array<string>;
    path?: never;
    query?: never;
    url: '/tools/localscan';
};

export type LocalscanAnalyzeLegacyResponses = {
    /**
     * OK
     */
    200: {
        alliances: {
            [key: string]: {
                corporations: {
                    [key: string]: {
                        characters: Array<{
                            characterId: number;
                            kills: number;
                            name: string;
                        }>;
                        name: string;
                        ticker: string;
                    };
                };
                name: string;
                ticker: string;
            };
        };
        corporations: {
            [key: string]: {
                characters: Array<{
                    characterId: number;
                    kills: number;
                    name: string;
                }>;
                name: string;
                ticker: string;
            };
        };
        totalCharacters: number;
        totalDangerous: number;
        unresolved: Array<string>;
    };
};

export type LocalscanAnalyzeLegacyResponse = LocalscanAnalyzeLegacyResponses[keyof LocalscanAnalyzeLegacyResponses];

export type LocalscanSaveLegacyData = {
    body: {
        /**
         * Character names from the local scan.
         */
        names: Array<string>;
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
        hash: string;
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
        alliances: {
            [key: string]: {
                corporations: {
                    [key: string]: {
                        characters: Array<{
                            characterId: number;
                            kills: number;
                            name: string;
                        }>;
                        name: string;
                        ticker: string;
                    };
                };
                name: string;
                ticker: string;
            };
        };
        corporations: {
            [key: string]: {
                characters: Array<{
                    characterId: number;
                    kills: number;
                    name: string;
                }>;
                name: string;
                ticker: string;
            };
        };
        totalCharacters: number;
        totalDangerous: number;
        unresolved: Array<string>;
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
        constellation: {
            constellation_id: number;
            constellation_name: string;
            faction_id: number | null;
            region_id: number;
            region_name: string | null;
        };
        sovDistribution: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            faction_id: number | null;
            faction_name: string | null;
            system_count: number;
        }>;
        stats: {
            kills: number;
            npc_kills: number;
            pod_kills: number;
            total_value: number;
        };
        systems: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            faction_id: number | null;
            faction_name: string | null;
            security: number;
            solar_system_id: number;
            system_name: string;
        }>;
    };
};

export type UniverseConstellationResponse = UniverseConstellationResponses[keyof UniverseConstellationResponses];

export type UniverseConstellationKillmailsData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
        /**
         * Page number for offset paging. Leave at 0 to page by cursor.
         */
        page?: number;
    };
    url: '/universe/constellations/{id}/killmails';
};

export type UniverseConstellationKillmailsResponses = {
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
        totalPages?: number;
    };
};

export type UniverseConstellationKillmailsResponse = UniverseConstellationKillmailsResponses[keyof UniverseConstellationKillmailsResponses];

export type UniverseConstellationMostValuableData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Restrict the losses to one category of hull.
         */
        dataType?: 'most_valuable_kills' | 'most_valuable_ships' | 'most_valuable_structures';
        /**
         * Size of the trailing window, in days.
         */
        days?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/universe/constellations/{id}/most-valuable';
};

export type UniverseConstellationMostValuableResponses = {
    /**
     * OK
     */
    200: {
        entries: Array<{
            killmail_hash: string;
            killmail_id: number;
            ship_name: string;
            ship_type_id: number;
            total_value: number;
            victim_alliance_name: string | null;
            victim_character_id: number | null;
            victim_character_name: string | null;
            victim_corporation_name: string | null;
        }>;
    };
};

export type UniverseConstellationMostValuableResponse = UniverseConstellationMostValuableResponses[keyof UniverseConstellationMostValuableResponses];

export type UniverseGroupData = {
    body?: never;
    path?: never;
    query?: never;
    url: '/universe/groups/{id}';
};

export type UniverseGroupResponses = {
    /**
     * OK
     */
    200: {
        group: {
            category_id: number | null;
            category_name: string | null;
            category_published: boolean | null;
            group_id: number;
            icon_id: number | null;
            name: string | null;
            published: boolean | null;
            published_type_count: number;
            type_count: number;
        };
        types: Array<{
            base_price: number | null;
            description: string | null;
            mass: number | null;
            meta_group_id: number | null;
            meta_group_name: string | null;
            name: string | null;
            published: boolean | null;
            type_id: number;
            volume: number | null;
        }>;
    };
};

export type UniverseGroupResponse = UniverseGroupResponses[keyof UniverseGroupResponses];

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
        constellations: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            constellation_id: number;
            constellation_name: string;
            faction_id: number | null;
            faction_name: string | null;
            system_count: number;
        }>;
        region: {
            constellation_count: number;
            description: string | null;
            faction_id: number | null;
            faction_name: string | null;
            name: string;
            region_id: number;
            system_count: number;
        };
        sovDistribution: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            faction_id: number | null;
            faction_name: string | null;
            system_count: number;
        }>;
        stats: {
            kills: number;
            npc_kills: number;
            pod_kills: number;
            total_value: number;
        };
        topSystems: Array<{
            kills: number;
            security: number;
            solar_system_id: number;
            system_name: string;
            total_value: number;
        }>;
    };
};

export type UniverseRegionResponse = UniverseRegionResponses[keyof UniverseRegionResponses];

export type UniverseRegionKillmailsData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
        /**
         * Page number for offset paging. Leave at 0 to page by cursor.
         */
        page?: number;
    };
    url: '/universe/regions/{id}/killmails';
};

export type UniverseRegionKillmailsResponses = {
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
        totalPages?: number;
    };
};

export type UniverseRegionKillmailsResponse = UniverseRegionKillmailsResponses[keyof UniverseRegionKillmailsResponses];

export type UniverseRegionMostValuableData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Restrict the losses to one category of hull.
         */
        dataType?: 'most_valuable_kills' | 'most_valuable_ships' | 'most_valuable_structures';
        /**
         * Size of the trailing window, in days.
         */
        days?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/universe/regions/{id}/most-valuable';
};

export type UniverseRegionMostValuableResponses = {
    /**
     * OK
     */
    200: {
        entries: Array<{
            killmail_hash: string;
            killmail_id: number;
            ship_name: string;
            ship_type_id: number;
            total_value: number;
            victim_alliance_name: string | null;
            victim_character_id: number | null;
            victim_character_name: string | null;
            victim_corporation_name: string | null;
        }>;
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
        activity: {
            history: Array<{
                npc_kills: number;
                pod_kills: number;
                ship_jumps: number;
                ship_kills: number;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                timestamp?: string;
            }>;
            latest: {
                npc_kills: number;
                pod_kills: number;
                ship_jumps: number;
                ship_kills: number;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                timestamp?: string;
            } | null;
            summary_24h: {
                npc_kills: number;
                pod_kills: number;
                ship_jumps: number;
                ship_kills: number;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                timestamp?: string;
            } | null;
        };
        celestialList: Array<{
            category: string;
            group_id: number;
            item_id: number;
            item_name: string;
            type_id: number;
            type_name: string | null;
        }>;
        celestials: {
            [key: string]: number;
        };
        connections: Array<{
            is_regional: boolean;
            region_id: number;
            security: number;
            system_name: string;
            to_solar_system_id: number;
        }>;
        sovereignty: {
            alliance_id: number | null;
            alliance_name: string | null;
            corporation_id: number | null;
            corporation_name: string | null;
            faction_id: number | null;
            faction_name: string | null;
        } | null;
        sovereigntyHistory: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            corporation_id: number | null;
            corporation_name: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            date: string;
            faction_id: number | null;
            faction_name: string | null;
        }>;
        stations: Array<{
            corporation_id: number;
            corporation_name: string | null;
            operation_name: string | null;
            station_id: number;
            station_name: string;
            type_id: number;
        }>;
        stats: {
            kills: number;
            npc_kills: number;
            pod_kills: number;
            total_value: number;
        };
        structures: Array<{
            is_market: boolean;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_seen: string | null;
            name: string;
            owner_id: number;
            owner_name: string | null;
            structure_id: number;
            type_id: number;
            type_name: string | null;
        }>;
        system: {
            border: boolean;
            constellation_id: number;
            constellation_name: string;
            corridor: boolean;
            faction_id: number | null;
            fringe: boolean;
            hub: boolean;
            international: boolean;
            region_id: number;
            region_name: string;
            regional: boolean;
            security: number;
            security_class: string | null;
            solar_system_id: number;
            sun_type_id: number | null;
            sun_type_name: string | null;
            system_name: string;
        };
    };
};

export type UniverseSystemResponse = UniverseSystemResponses[keyof UniverseSystemResponses];

export type UniverseSystemKillmailsData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
        /**
         * Page number for offset paging. Leave at 0 to page by cursor.
         */
        page?: number;
    };
    url: '/universe/systems/{id}/killmails';
};

export type UniverseSystemKillmailsResponses = {
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
        totalPages?: number;
    };
};

export type UniverseSystemKillmailsResponse = UniverseSystemKillmailsResponses[keyof UniverseSystemKillmailsResponses];

export type UniverseSystemMostValuableData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Restrict the losses to one category of hull.
         */
        dataType?: 'most_valuable_kills' | 'most_valuable_ships' | 'most_valuable_structures';
        /**
         * Size of the trailing window, in days.
         */
        days?: number;
        /**
         * Maximum results to return.
         */
        limit?: number;
    };
    url: '/universe/systems/{id}/most-valuable';
};

export type UniverseSystemMostValuableResponses = {
    /**
     * OK
     */
    200: {
        entries: Array<{
            killmail_hash: string;
            killmail_id: number;
            ship_name: string;
            ship_type_id: number;
            total_value: number;
            victim_alliance_name: string | null;
            victim_character_id: number | null;
            victim_character_name: string | null;
            victim_corporation_name: string | null;
        }>;
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
        attributes: Array<{
            id: number;
            name?: string;
            value: number;
        }>;
        item: {
            base_price: number | null;
            capacity: number | null;
            category_id: number;
            category_name: string | null;
            description: string | null;
            faction_id: number | null;
            group_id: number;
            group_name: string | null;
            is_ship: boolean;
            market_group_id: number | null;
            mass: number | null;
            meta_group_id: number | null;
            meta_group_name: string | null;
            meta_level: number | null;
            name: string;
            packaged_volume: number | null;
            portion_size: number | null;
            published: boolean;
            race_id: number | null;
            radius: number | null;
            tech_level: number | null;
            type_id: number;
            volume: number | null;
        };
        marketBreadcrumb: Array<{
            id: number;
            name: string;
            slug: string;
        }>;
        materials: Array<{
            name: string | null;
            quantity: number;
            type_id: number;
        }>;
        pricing: {
            customHistory: Array<{
                date: string;
                price: number;
            }>;
            customSummary: {
                average_90d: number;
                avg_volume_90d?: number;
                highest_90d: number;
                latest: number;
                latest_date: string;
                lowest_90d: number | null;
            } | null;
            history: Array<{
                average: number;
                date: string;
                highest: number;
                lowest: number;
                volume: number;
            }>;
            insurance: Array<{
                cost?: number;
                name?: string;
                payout?: number;
                [key: string]: unknown;
            }>;
            summary: {
                average_90d: number;
                avg_volume_90d?: number;
                highest_90d: number;
                latest: number;
                latest_date: string;
                lowest_90d: number | null;
            } | null;
        };
        requiredSkills: Array<{
            level: number;
            name: string | null;
            type_id: number;
        }>;
        shipAttributes: {
            [key: string]: Array<{
                id: number;
                name?: string;
                value: number;
            }>;
        } | null;
        variations: Array<{
            meta_group_id: number | null;
            name: string;
            type_id: number;
        }>;
    };
};

export type UniverseTypeResponse = UniverseTypeResponses[keyof UniverseTypeResponses];

export type UniverseTypeKillmailsData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
    url: '/universe/types/{id}/killmails';
};

export type UniverseTypeKillmailsResponses = {
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
        totalPages?: number;
    };
};

export type UniverseTypeKillmailsResponse = UniverseTypeKillmailsResponses[keyof UniverseTypeKillmailsResponses];

export type UserBoardsUpdateCompatData = {
    body: {
        dismissed: Array<string>;
        pinned: Array<string>;
    };
    path?: never;
    query?: never;
    url: '/user/boards';
};

export type UserBoardsUpdateCompatResponses = {
    /**
     * OK
     */
    200: {
        dismissed: Array<string>;
        pinned: Array<string>;
    };
};

export type UserBoardsUpdateCompatResponse = UserBoardsUpdateCompatResponses[keyof UserBoardsUpdateCompatResponses];

export type MyCommentsLiveAliasData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Identifier cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        cursor?: number;
    };
    url: '/user/comments';
};

export type MyCommentsLiveAliasResponses = {
    /**
     * OK
     */
    200: {
        comments: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            body_html: string;
            body_md: string;
            character_id: number;
            character_name: string;
            corporation_id: number;
            corporation_name: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            deleted_at: string | null;
            deleted_by: number | null;
            depth: number;
            domain_id: number | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            edited_at: string | null;
            flagged: boolean;
            id: number;
            moderation_status: number;
            parent_id: number | null;
            reply_count?: number;
            reports_count: number;
            root_id: number | null;
            target_id: number;
            target_slug: string | null;
            target_type: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at: string;
            visibility: number;
        }>;
        nextCursor: number | null;
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
        ok: boolean;
    };
};

export type MyCommentDeleteLiveAliasResponse = MyCommentDeleteLiveAliasResponses[keyof MyCommentDeleteLiveAliasResponses];

export type UserDescriptionUpdateCompatData = {
    body: {
        /**
         * Description text.
         */
        description: string;
        /**
         * Which description to write.
         */
        entity: 'character' | 'corporation' | 'alliance';
        /**
         * How to interpret the text.
         */
        format: 'markdown' | 'eve_html';
    };
    path?: never;
    query?: never;
    url: '/user/descriptions';
};

export type UserDescriptionUpdateCompatResponses = {
    /**
     * OK
     */
    200: {
        entity: string;
        entity_id: number;
        ok: boolean;
        queue_id?: number;
        status: string;
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
        domains: Array<{
            active?: boolean;
            backgrounds?: Array<{
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            }>;
            bannerAsset?: {
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            } | null;
            campaign_policy?: number;
            campaigns?: Array<{
                campaign_id?: string;
                created_by_character_id?: number;
                description?: string | null;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                end_time?: string;
                estimated_killmails?: number | null;
                name?: string;
                public_on_domain?: boolean;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                start_time?: string;
                status?: number;
                visibility?: number;
                [key: string]: unknown;
            }>;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            custom_hostname?: string | null;
            entities?: Array<{
                id: number;
                name?: string;
                type: string;
            }>;
            id?: number;
            logoAsset?: {
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            } | null;
            navbar_links?: Array<{
                children?: Array<{
                    items: Array<{
                        external?: boolean;
                        href: string;
                        icon?: string;
                        label: string;
                    }>;
                    label?: string;
                }>;
                external?: boolean;
                href: string;
                icon?: string;
                label: string;
            }>;
            site_description?: string | null;
            site_name?: string | null;
            subdomain?: string;
            theme?: {
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
                [key: string]: unknown;
            };
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            user_id?: number;
            widgets?: {
                columnRatio: string;
                left: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
                right: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
                top: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
            };
            [key: string]: unknown;
        }>;
    };
};

export type DomainsMineCompatResponse = DomainsMineCompatResponses[keyof DomainsMineCompatResponses];

export type DomainCreateCompatData = {
    body: {
        entities: Array<{
            id: number;
            name: string;
            type: 'character' | 'corporation' | 'alliance';
        }>;
        navbar_links?: Array<{
            children?: Array<SiteDomainNavbarGroup>;
            external?: boolean;
            href: string;
            icon?: string;
            label: string;
        }>;
        site_description?: string | null;
        site_name?: string | null;
        subdomain: string;
        theme?: SiteDomainTheme;
        widgets?: SiteDomainWidgets;
    };
    path?: never;
    query?: never;
    url: '/user/domains';
};

export type DomainCreateCompatResponses = {
    /**
     * OK
     */
    200: {
        domain: {
            active?: boolean;
            backgrounds?: Array<{
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            }>;
            bannerAsset?: {
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            } | null;
            campaign_policy?: number;
            campaigns?: Array<{
                campaign_id?: string;
                created_by_character_id?: number;
                description?: string | null;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                end_time?: string;
                estimated_killmails?: number | null;
                name?: string;
                public_on_domain?: boolean;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                start_time?: string;
                status?: number;
                visibility?: number;
                [key: string]: unknown;
            }>;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            custom_hostname?: string | null;
            entities?: Array<{
                id: number;
                name?: string;
                type: string;
            }>;
            id?: number;
            logoAsset?: {
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            } | null;
            navbar_links?: Array<{
                children?: Array<{
                    items: Array<{
                        external?: boolean;
                        href: string;
                        icon?: string;
                        label: string;
                    }>;
                    label?: string;
                }>;
                external?: boolean;
                href: string;
                icon?: string;
                label: string;
            }>;
            site_description?: string | null;
            site_name?: string | null;
            subdomain?: string;
            theme?: {
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
                [key: string]: unknown;
            };
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            user_id?: number;
            widgets?: {
                columnRatio: string;
                left: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
                right: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
                top: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
            };
            [key: string]: unknown;
        };
    };
};

export type DomainCreateCompatResponse = DomainCreateCompatResponses[keyof DomainCreateCompatResponses];

export type DomainSubdomainCheckCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Subdomain to test.
         */
        subdomain?: string;
    };
    url: '/user/domains/check-subdomain';
};

export type DomainSubdomainCheckCompatResponses = {
    /**
     * OK
     */
    200: {
        available: boolean;
        reason?: string;
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
        deleted: boolean;
    };
};

export type DomainDeleteCompatResponse = DomainDeleteCompatResponses[keyof DomainDeleteCompatResponses];

export type DomainUpdatePatchCompatData = {
    body: {
        active?: boolean;
        campaign_ids?: Array<string>;
        campaign_policy?: 0 | 1;
        campaign_public_ids?: Array<string>;
        entities?: Array<{
            id: number;
            name: string;
            type: 'character' | 'corporation' | 'alliance';
        }>;
        navbar_links?: Array<{
            children?: Array<SiteDomainNavbarGroup>;
            external?: boolean;
            href: string;
            icon?: string;
            label: string;
        }>;
        site_description?: string | null;
        site_name?: string | null;
        theme?: SiteDomainTheme;
        widgets?: SiteDomainWidgets;
    };
    path?: never;
    query?: never;
    url: '/user/domains/{id}';
};

export type DomainUpdatePatchCompatResponses = {
    /**
     * OK
     */
    200: {
        domain: {
            active?: boolean;
            backgrounds?: Array<{
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            }>;
            bannerAsset?: {
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            } | null;
            campaign_policy?: number;
            campaigns?: Array<{
                campaign_id?: string;
                created_by_character_id?: number;
                description?: string | null;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                end_time?: string;
                estimated_killmails?: number | null;
                name?: string;
                public_on_domain?: boolean;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                start_time?: string;
                status?: number;
                visibility?: number;
                [key: string]: unknown;
            }>;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            custom_hostname?: string | null;
            entities?: Array<{
                id: number;
                name?: string;
                type: string;
            }>;
            id?: number;
            logoAsset?: {
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            } | null;
            navbar_links?: Array<{
                children?: Array<{
                    items: Array<{
                        external?: boolean;
                        href: string;
                        icon?: string;
                        label: string;
                    }>;
                    label?: string;
                }>;
                external?: boolean;
                href: string;
                icon?: string;
                label: string;
            }>;
            site_description?: string | null;
            site_name?: string | null;
            subdomain?: string;
            theme?: {
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
                [key: string]: unknown;
            };
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            user_id?: number;
            widgets?: {
                columnRatio: string;
                left: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
                right: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
                top: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
            };
            [key: string]: unknown;
        };
    };
};

export type DomainUpdatePatchCompatResponse = DomainUpdatePatchCompatResponses[keyof DomainUpdatePatchCompatResponses];

export type DomainUpdatePutCompatData = {
    body: {
        active?: boolean;
        campaign_ids?: Array<string>;
        campaign_policy?: 0 | 1;
        campaign_public_ids?: Array<string>;
        entities?: Array<{
            id: number;
            name: string;
            type: 'character' | 'corporation' | 'alliance';
        }>;
        navbar_links?: Array<{
            children?: Array<SiteDomainNavbarGroup>;
            external?: boolean;
            href: string;
            icon?: string;
            label: string;
        }>;
        site_description?: string | null;
        site_name?: string | null;
        theme?: SiteDomainTheme;
        widgets?: SiteDomainWidgets;
    };
    path?: never;
    query?: never;
    url: '/user/domains/{id}';
};

export type DomainUpdatePutCompatResponses = {
    /**
     * OK
     */
    200: {
        domain: {
            active?: boolean;
            backgrounds?: Array<{
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            }>;
            bannerAsset?: {
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            } | null;
            campaign_policy?: number;
            campaigns?: Array<{
                campaign_id?: string;
                created_by_character_id?: number;
                description?: string | null;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                end_time?: string;
                estimated_killmails?: number | null;
                name?: string;
                public_on_domain?: boolean;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                start_time?: string;
                status?: number;
                visibility?: number;
                [key: string]: unknown;
            }>;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            custom_hostname?: string | null;
            entities?: Array<{
                id: number;
                name?: string;
                type: string;
            }>;
            id?: number;
            logoAsset?: {
                /**
                 * UTC timestamp with millisecond precision.
                 */
                created_at?: string;
                domain_id?: number;
                id?: number;
                reject_reason?: string | null;
                status?: string;
                type?: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                updated_at?: string;
                [key: string]: unknown;
            } | null;
            navbar_links?: Array<{
                children?: Array<{
                    items: Array<{
                        external?: boolean;
                        href: string;
                        icon?: string;
                        label: string;
                    }>;
                    label?: string;
                }>;
                external?: boolean;
                href: string;
                icon?: string;
                label: string;
            }>;
            site_description?: string | null;
            site_name?: string | null;
            subdomain?: string;
            theme?: {
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
                [key: string]: unknown;
            };
            /**
             * UTC timestamp with millisecond precision.
             */
            updated_at?: string;
            user_id?: number;
            widgets?: {
                columnRatio: string;
                left: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
                right: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
                top: Array<{
                    campaignId?: string;
                    content?: string;
                    enabled?: boolean;
                    killlistType?: string;
                    title?: string;
                    type?: string;
                    [key: string]: unknown;
                }>;
            };
            [key: string]: unknown;
        };
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
        success: boolean;
    };
};

export type DomainAssetDeleteCompatResponse = DomainAssetDeleteCompatResponses[keyof DomainAssetDeleteCompatResponses];

export type DomainCampaignSearchCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Campaign name search.
         */
        q?: string;
    };
    url: '/user/domains/{id}/campaigns/search';
};

export type DomainCampaignSearchCompatResponses = {
    /**
     * OK
     */
    200: {
        campaigns: Array<{
            campaign_id?: string;
            created_by_character_id?: number;
            description?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            end_time?: string;
            estimated_killmails?: number | null;
            name?: string;
            public_on_domain?: boolean;
            /**
             * UTC timestamp with millisecond precision.
             */
            start_time?: string;
            status?: number;
            visibility?: number;
            [key: string]: unknown;
        }>;
    };
};

export type DomainCampaignSearchCompatResponse = DomainCampaignSearchCompatResponses[keyof DomainCampaignSearchCompatResponses];

export type DomainAssetsDeleteTypeCompatData = {
    body: {
        /**
         * Asset slot to clear.
         */
        type: 'banner' | 'logo';
    };
    path?: never;
    query?: never;
    url: '/user/domains/{id}/upload';
};

export type DomainAssetsDeleteTypeCompatResponses = {
    /**
     * OK
     */
    200: {
        success: boolean;
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
        assetId: number;
        message: string;
        status: string;
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
        rateLimit: {
            request_count: number;
        };
        responseTime: {
            avg_ms: number | null;
            p95_ms: number | null;
        };
        volumeByHour: Array<{
            errors: number;
            hour: string;
            new_items: number;
            total: number;
        }>;
    };
};

export type UserEsiCompatResponse = UserEsiCompatResponses[keyof UserEsiCompatResponses];

export type UserEsiLogsCompatData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Page number, counted from 1.
         */
        page?: number;
        /**
         * Match the recorded request source exactly.
         */
        source?: string;
        /**
         * Restrict to successful or failed requests.
         */
        status?: 'success' | 'error';
        /**
         * Restrict to one ESI endpoint family, for example `killmails`.
         */
        endpoint_type?: string;
        /**
         * Return log rows below this log ID.
         */
        after_id?: number;
    };
    url: '/user/esi-logs';
};

export type UserEsiLogsCompatResponses = {
    /**
     * OK
     */
    200: {
        limit?: number;
        newRows?: boolean;
        page?: number;
        pages?: number;
        rows: Array<{
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at: string;
            endpoint: string;
            endpoint_action: string;
            endpoint_type: string;
            error_message: string | null;
            id: number;
            items_returned: number | null;
            method: string;
            new_items: number | null;
            request_duration_ms: number | null;
            source: string;
            status_code: number | null;
            success: boolean;
        }>;
        sources?: Array<string>;
        total?: number;
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
        alliance: {
            canEdit: boolean;
            ceo_id?: number | null;
            ceo_name?: string | null;
            custom_description: string | null;
            custom_description_format: string;
            esi_description?: string | null;
            executor_ceo_id?: number | null;
            executor_ceo_name?: string | null;
            executor_corporation_id?: number | null;
            id: number;
            name: string;
            pending_submission: {
                body: string;
                body_format: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                submitted_at: string;
            } | null;
            ticker?: string;
        } | null;
        character: {
            canEdit: boolean;
            ceo_id?: number | null;
            ceo_name?: string | null;
            custom_description: string | null;
            custom_description_format: string;
            esi_description?: string | null;
            executor_ceo_id?: number | null;
            executor_ceo_name?: string | null;
            executor_corporation_id?: number | null;
            id: number;
            name: string;
            pending_submission: {
                body: string;
                body_format: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                submitted_at: string;
            } | null;
            ticker?: string;
        };
        corporation: {
            canEdit: boolean;
            ceo_id?: number | null;
            ceo_name?: string | null;
            custom_description: string | null;
            custom_description_format: string;
            esi_description?: string | null;
            executor_ceo_id?: number | null;
            executor_ceo_name?: string | null;
            executor_corporation_id?: number | null;
            id: number;
            name: string;
            pending_submission: {
                body: string;
                body_format: string;
                /**
                 * UTC timestamp with millisecond precision.
                 */
                submitted_at: string;
            } | null;
            ticker?: string;
        } | null;
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
        account: {
            characterId: number;
            characterName: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            createdAt: string | null;
            isAdmin: boolean;
            /**
             * UTC timestamp with millisecond precision.
             */
            lastLogin: string | null;
        };
        esiStats: {
            errors_24h?: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            last_request?: string | null;
            new_items_24h?: number;
            requests_24h?: number;
            total_errors?: number;
            total_new_items?: number;
            total_requests?: number;
        };
        esiToken: {
            /**
             * UTC timestamp with millisecond precision.
             */
            lastFetched: string | null;
            scopeCount: number;
            /**
             * UTC timestamp with millisecond precision.
             */
            tokenExpiry: string | null;
        } | null;
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
        boards?: {
            dismissed: Array<string>;
            pinned: Array<string>;
        };
        /**
         * Default tab keyed by page type.
         */
        defaultTabs?: {
            [key: string]: unknown;
        };
        /**
         * User-selected theme settings.
         */
        theme?: {
            [key: string]: unknown;
        };
    };
};

export type UserPreferencesCompatResponse = UserPreferencesCompatResponses[keyof UserPreferencesCompatResponses];

export type UserPreferencesUpdateCompatData = {
    body: {
        defaultTabs: {
            [key: string]: string;
        };
    };
    path?: never;
    query?: never;
    url: '/user/preferences';
};

export type UserPreferencesUpdateCompatResponses = {
    /**
     * OK
     */
    200: {
        preferences: {
            boards?: {
                dismissed: Array<string>;
                pinned: Array<string>;
            };
            /**
             * Default tab keyed by page type.
             */
            defaultTabs?: {
                [key: string]: unknown;
            };
            /**
             * User-selected theme settings.
             */
            theme?: {
                [key: string]: unknown;
            };
        };
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
        sessions: Array<{
            browser: string;
            countryCode: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            createdAt: string;
            current: boolean;
            device: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            expiresAt: string;
            id: number;
            ipAddress: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            lastSeenAt: string;
            operatingSystem: string;
        }>;
    };
};

export type UserSessionsLegacyResponse = UserSessionsLegacyResponses[keyof UserSessionsLegacyResponses];

export type OtherSessionsRevokeLegacyData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Session ID to keep. Every other session is revoked.
         */
        except?: string;
    };
    url: '/user/sessions/revoke-others';
};

export type OtherSessionsRevokeLegacyResponses = {
    /**
     * OK
     */
    200: {
        revoked: number;
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
        current: boolean;
        revoked: boolean;
    };
};

export type UserSessionRevokeLegacyResponse = UserSessionRevokeLegacyResponses[keyof UserSessionRevokeLegacyResponses];

export type UserThemeUpdateCompatData = {
    body: {
        theme: {
            [key: string]: string;
        };
    };
    path?: never;
    query?: never;
    url: '/user/theme';
};

export type UserThemeUpdateCompatResponses = {
    /**
     * OK
     */
    200: {
        /**
         * User-selected theme settings.
         */
        theme: {
            [key: string]: unknown;
        };
    };
};

export type UserThemeUpdateCompatResponse = UserThemeUpdateCompatResponses[keyof UserThemeUpdateCompatResponses];

export type WalletAccountLegacyData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Page number, counted from 1.
         */
        page?: number;
    };
    url: '/user/wallet';
};

export type WalletAccountLegacyResponses = {
    /**
     * OK
     */
    200: {
        availableBalance: string;
        balance: string;
        character: {
            character_id: number;
            character_name: string;
        };
        corporation: {
            corporation_id: number;
            name: string;
            ticker: string;
        };
        /**
         * UTC timestamp with millisecond precision.
         */
        depositsEnabledAt: string | null;
        hasMore: boolean;
        /**
         * UTC timestamp with millisecond precision.
         */
        lastSynced: string | null;
        page: number;
        pageSize: number;
        reservations: Array<{
            amount?: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            description?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            expires_at?: string;
            external_reference?: string;
            id?: number;
            transaction_type?: number;
            [key: string]: unknown;
        }>;
        reservedBalance: string;
        totalBalance: string;
        transactions: Array<{
            amount?: string;
            balance_after?: string;
            /**
             * UTC timestamp with millisecond precision.
             */
            created_at?: string;
            description?: string | null;
            id?: number;
            type?: number;
            [key: string]: unknown;
        }>;
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
        availableBalance: string;
        balance: string;
        reservedBalance: string;
        totalBalance: string;
    };
};

export type WalletAccountBalanceLegacyResponse = WalletAccountBalanceLegacyResponses[keyof WalletAccountBalanceLegacyResponses];

export type WalletPublicData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Page number, counted from 1.
         */
        page?: number;
        /**
         * Corporation wallet division.
         */
        division?: number;
    };
    url: '/wallet';
};

export type WalletPublicResponses = {
    /**
     * OK
     */
    200: {
        corporation: {
            corporation_id: number;
            name: string;
            ticker: string;
        };
        division: number | null;
        hasMore: boolean;
        journal: Array<{
            amount?: string;
            balance?: string;
            context_id?: number | null;
            context_id_type?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            date?: string;
            description?: string | null;
            division?: number;
            first_party_id?: number | null;
            journal_id?: number;
            reason?: string | null;
            ref_type?: string;
            second_party_id?: number | null;
            tax?: string | null;
            tax_receiver_id?: number | null;
            [key: string]: unknown;
        }>;
        /**
         * UTC timestamp with millisecond precision.
         */
        lastSynced: string | null;
        page: number;
        pageSize: number;
        totalBalance: string;
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
        stats: {
            top_ships: Array<{
                count: number;
                ship_name: string;
                ship_type_id: number;
            }>;
            total_kills: number;
            total_value: number;
        };
        war: {
            aggressor?: {
                id: number;
                name?: string;
                ticker?: string;
                type: string;
            };
            allies?: Array<{
                id: number;
                isk_destroyed: number;
                isk_lost: number;
                kills: number;
                losses: number;
                name: string;
                type: string;
            }>;
            /**
             * UTC timestamp with millisecond precision.
             */
            declared?: string;
            defender?: {
                id: number;
                name?: string;
                ticker?: string;
                type: string;
            };
            /**
             * UTC timestamp with millisecond precision.
             */
            finished?: string | null;
            mutual?: boolean;
            open_for_allies?: boolean;
            /**
             * UTC timestamp with millisecond precision.
             */
            retracted?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            started?: string | null;
            war_id?: number;
            [key: string]: unknown;
        };
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
        leaderboards: {
            aggressor: {
                topAlliances: Array<{
                    count: number;
                    id: number;
                    isk_value: number;
                    name: string;
                }>;
                topCharacters: Array<{
                    count: number;
                    id: number;
                    isk_value: number;
                    name: string;
                }>;
                topCorporations: Array<{
                    count: number;
                    id: number;
                    isk_value: number;
                    name: string;
                }>;
            };
            combined: {
                topAlliances: Array<{
                    count: number;
                    id: number;
                    isk_value: number;
                    name: string;
                }>;
                topCharacters: Array<{
                    count: number;
                    id: number;
                    isk_value: number;
                    name: string;
                }>;
                topCorporations: Array<{
                    count: number;
                    id: number;
                    isk_value: number;
                    name: string;
                }>;
            };
            defender: {
                topAlliances: Array<{
                    count: number;
                    id: number;
                    isk_value: number;
                    name: string;
                }>;
                topCharacters: Array<{
                    count: number;
                    id: number;
                    isk_value: number;
                    name: string;
                }>;
                topCorporations: Array<{
                    count: number;
                    id: number;
                    isk_value: number;
                    name: string;
                }>;
            };
            sides: {
                aggressor: {
                    alliances: Array<number>;
                    corporations: Array<number>;
                };
                defender: {
                    alliances: Array<number>;
                    corporations: Array<number>;
                };
            };
            topShips: Array<{
                count: number;
                ship_name: string;
                ship_type_id: number;
            }>;
        };
        stats: {
            top_ships: Array<{
                count: number;
                ship_name: string;
                ship_type_id: number;
            }>;
            total_kills: number;
            total_value: number;
        };
        war: {
            aggressor?: {
                id: number;
                name?: string;
                ticker?: string;
                type: string;
            };
            allies?: Array<{
                id: number;
                isk_destroyed: number;
                isk_lost: number;
                kills: number;
                losses: number;
                name: string;
                type: string;
            }>;
            /**
             * UTC timestamp with millisecond precision.
             */
            declared?: string;
            defender?: {
                id: number;
                name?: string;
                ticker?: string;
                type: string;
            };
            /**
             * UTC timestamp with millisecond precision.
             */
            finished?: string | null;
            mutual?: boolean;
            open_for_allies?: boolean;
            /**
             * UTC timestamp with millisecond precision.
             */
            retracted?: string | null;
            /**
             * UTC timestamp with millisecond precision.
             */
            started?: string | null;
            war_id?: number;
            [key: string]: unknown;
        };
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
        security_breakdown: Array<{
            isk_destroyed: number;
            kills: number;
            sec_class: string;
        }>;
        ship_groups_destroyed: Array<{
            count?: number;
            group_id?: number;
            group_name?: string | null;
            isk_destroyed?: number;
            [key: string]: unknown;
        }>;
        ships_destroyed: Array<{
            count: number;
            group_id: number | null;
            group_name: string | null;
            isk_destroyed: number;
            ship_name: string | null;
            ship_type_id: number;
        }>;
        ships_used: Array<{
            count: number;
            group_id: number | null;
            group_name: string | null;
            ship_name: string | null;
            ship_type_id: number;
        }>;
        summary: {
            constellations: number;
            isk_destroyed: number;
            kills: number;
            regions: number;
            systems: number;
        };
        top_constellations: Array<{
            constellation_id?: number;
            constellation_name?: string | null;
            isk_destroyed?: number;
            kills?: number;
            region_id?: number | null;
            region_name?: string | null;
            [key: string]: unknown;
        }>;
        top_regions: Array<{
            isk_destroyed?: number;
            kills?: number;
            region_id?: number;
            region_name?: string | null;
            [key: string]: unknown;
        }>;
        top_systems: Array<{
            isk_destroyed?: number;
            kills?: number;
            region_id?: number | null;
            region_name?: string | null;
            security?: number | null;
            system_id?: number;
            system_name?: string | null;
            [key: string]: unknown;
        }>;
        war_id: number;
    };
};

export type WarIntelResponse = WarIntelResponses[keyof WarIntelResponses];

export type WarKilllistData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
        /**
         * Page number for offset paging. Leave at 0 to page by cursor.
         */
        page?: number;
        /**
         * Override the window start.
         */
        warStart?: string;
        /**
         * Override the window end.
         */
        warEnd?: string;
        /**
         * Comma-separated corporation IDs on the side.
         */
        warSideCorps?: string;
        /**
         * Comma-separated alliance IDs on the side.
         */
        warSideAlliances?: string;
    };
    url: '/war/{id}/killlist';
};

export type WarKilllistResponses = {
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
            ship_group_id: number | null;
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
            victim_faction_id: number | null;
            [key: string]: unknown;
        }>;
        totalPages?: number;
    };
};

export type WarKilllistResponse = WarKilllistResponses[keyof WarKilllistResponses];

export type WarMembersData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Which side of the war to report on.
         */
        side?: 'aggressor' | 'defender' | 'combined';
        /**
         * Ordering for the member rows.
         */
        sort?: 'kills' | 'losses' | 'isk' | 'activity';
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Restrict to one corporation.
         */
        corporationId?: number;
        /**
         * Restrict to one alliance.
         */
        allianceId?: number;
    };
    url: '/war/{id}/members';
};

export type WarMembersResponses = {
    /**
     * OK
     */
    200: {
        count: number;
        limit: number;
        members: Array<{
            alliance_id: number | null;
            alliance_name: string | null;
            alliance_ticker: string | null;
            character_id: number;
            character_name: string;
            corporation_id: number | null;
            corporation_name: string | null;
            corporation_ticker: string | null;
            isk_destroyed: number;
            isk_lost: number;
            kills: number;
            losses: number;
            side: string;
            top_ship_count: number;
            top_ship_name: string | null;
            top_ship_type_id: number | null;
        }>;
        side: string;
        war_id: number;
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
        aggressor: {
            topAlliances: Array<{
                count: number;
                id: number;
                isk_value: number;
                name: string;
            }>;
            topCharacters: Array<{
                count: number;
                id: number;
                isk_value: number;
                name: string;
            }>;
            topCorporations: Array<{
                count: number;
                id: number;
                isk_value: number;
                name: string;
            }>;
        };
        combined: {
            topAlliances: Array<{
                count: number;
                id: number;
                isk_value: number;
                name: string;
            }>;
            topCharacters: Array<{
                count: number;
                id: number;
                isk_value: number;
                name: string;
            }>;
            topCorporations: Array<{
                count: number;
                id: number;
                isk_value: number;
                name: string;
            }>;
        };
        defender: {
            topAlliances: Array<{
                count: number;
                id: number;
                isk_value: number;
                name: string;
            }>;
            topCharacters: Array<{
                count: number;
                id: number;
                isk_value: number;
                name: string;
            }>;
            topCorporations: Array<{
                count: number;
                id: number;
                isk_value: number;
                name: string;
            }>;
        };
        sides: {
            aggressor: {
                alliances: Array<number>;
                corporations: Array<number>;
            };
            defender: {
                alliances: Array<number>;
                corporations: Array<number>;
            };
        };
        topShips: Array<{
            count: number;
            ship_name: string;
            ship_type_id: number;
        }>;
    };
};

export type WarLeaderboardsResponse = WarLeaderboardsResponses[keyof WarLeaderboardsResponses];

export type WarsData = {
    body?: never;
    path?: never;
    query?: {
        /**
         * Maximum results to return.
         */
        limit?: number;
        /**
         * Ascending cursor. Pass the previous response's pagination cursor to fetch the next page.
         */
        after?: number;
    };
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
    query?: {
        /**
         * Which kind of war target to list.
         */
        type?: 'corporations' | 'alliances';
    };
    url: '/wars/eligible';
};

export type WarsEligibleResponses = {
    /**
     * OK
     */
    200: {
        entries: Array<{
            alliance_id?: number | null;
            alliance_name?: string | null;
            corporation_count?: number;
            id?: number;
            isk_destroyed?: number;
            isk_lost?: number;
            kills?: number;
            losses?: number;
            member_count?: number;
            name?: string;
            ticker?: string;
            [key: string]: unknown;
        }>;
        limit: number;
        type: string;
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
        activeWars: number;
        eligibleAlliances: number;
        eligibleCorps: number;
        finishedWars: number;
        upcomingWars: number;
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
