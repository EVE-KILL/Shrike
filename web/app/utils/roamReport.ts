export interface RoamPilot {
    character_id: number
    name: string
    corporation_id: number
    corporation_name: string
    corporation_ticker: string
    alliance_id: number
    alliance_name: string
    alliance_ticker: string
    kill_participations: number
    final_blows: number
    losses: number
    engagements: number
    damage_done: number
}

export interface RoamKillmail {
    killmail_id: number
    killmail_time: string
    solar_system_id: number
    solar_system_name: string
    region_id: number
    region_name: string
    victim_character_id: number
    victim_name: string
    victim_corporation_id: number
    victim_corporation_name: string
    victim_alliance_id: number
    victim_alliance_name: string
    victim_ship_type_id: number
    victim_ship_name: string
    total_value: number
    attacker_count: number
    role: 'kill' | 'loss'
    friendly_fire: boolean
    fleet_attacker_ids: number[]
    fleet_final_blow: boolean
    final_blow_character_name: string
    final_blow_corporation_name: string
}

export interface RoamEngagement {
    number: number
    solar_system_id: number
    solar_system_name: string
    region_id: number
    region_name: string
    start_time: string
    end_time: string
    kills: number
    losses: number
    isk_destroyed: number
    isk_lost: number
    pilot_ids: number[]
    killmails: RoamKillmail[]
}

export interface RoamSystem {
    solar_system_id: number
    solar_system_name: string
    region_id: number
    region_name: string
    first_seen: string
    last_seen: string
    engagements: number
    kills: number
    losses: number
    isk_destroyed: number
    isk_lost: number
}

export interface RoamTarget {
    corporation_id: number
    corporation_name: string
    alliance_id: number
    alliance_name: string
    kills: number
    isk_destroyed: number
}

export interface RoamReport {
    id: string
    created_at: string
    start_time: string
    end_time: string
    input_count: number
    roster: RoamPilot[]
    unresolved: string[]
    truncated: boolean
    summary: {
        kills: number
        losses: number
        friendly_fire_losses: number
        isk_destroyed: number
        isk_lost: number
        efficiency: number
        engagements: number
        systems: number
        regions: number
        active_pilots: number
        coordinated_kills: number
        unique_character_targets: number
    }
    systems: RoamSystem[]
    targets: RoamTarget[]
    engagements: RoamEngagement[]
}
