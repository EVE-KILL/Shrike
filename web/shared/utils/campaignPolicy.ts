/**
 * Frontend validation limits for campaign creation.
 *
 * These mirror the canonical Go campaign policy. They are presentation rules,
 * not persistence models; the Go API remains authoritative and validates every
 * submitted campaign independently.
 */
export const CAMPAIGN_MAX_LOCATION_SYSTEMS = 10
export const CAMPAIGN_MAX_LOCATION_CONSTELLATIONS = 5
export const CAMPAIGN_MAX_LOCATION_REGIONS = 5
export const CAMPAIGN_MAX_PUBLIC_WINDOW_DAYS = 365
export const CAMPAIGN_MAX_PRIVATE_WINDOW_DAYS = 730
export const CAMPAIGN_MAX_PUBLIC_ONGOING_PER_USER = 3

export const CAMPAIGN_PRIZE_METRIC = {
    KILLS: 0,
    LOSSES: 1,
    ISK_DESTROYED: 2,
    ISK_LOST: 3,
} as const

export const CAMPAIGN_PRIZE_DEFAULT_SPLITS: Readonly<
    Record<number, readonly number[]>
> = {
    3: [70, 20, 10],
    4: [60, 20, 12, 8],
    5: [50, 20, 12, 10, 8],
    6: [45, 20, 12, 9, 8, 6],
    7: [40, 20, 13, 9, 7, 6, 5],
    8: [38, 19, 12, 9, 7, 6, 5, 4],
    9: [36, 18, 12, 9, 7, 6, 5, 4, 3],
    10: [34, 18, 12, 9, 7, 6, 5, 4, 3, 2],
}
