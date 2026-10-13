export type MapLayer = 'geography' | 'security' | 'ship-kills' | 'pod-kills' | 'npc-kills' | 'jumps'
export type MapBaseLayer = 'geography' | 'security'
export type MapRenderBaseLayer = MapBaseLayer | 'sovereignty' | 'live' | 'aiid'
export type MapActivityLayer = 'none' | 'activity' | 'danger' | 'ship-kills' | 'pod-kills' | 'npc-kills' | 'jumps'
export type MapActivityHours = 1 | 6 | 24 | 168
export type AIIDAlarmBand = 'near' | 'outer'

export interface MapActivity {
    ship_kills: number
    pod_kills: number
    npc_kills: number
    ship_jumps: number
}

export const MAP_LAYERS: Array<{ id: MapLayer; label: string; shortLabel: string }> = [
    { id: 'geography', label: 'Geography', shortLabel: 'Geography' },
    { id: 'security', label: 'Security status', shortLabel: 'Security' },
    { id: 'ship-kills', label: 'Ship kills', shortLabel: 'Ship kills' },
    { id: 'pod-kills', label: 'Pod kills', shortLabel: 'Pod kills' },
    { id: 'npc-kills', label: 'NPC kills', shortLabel: 'NPC kills' },
    { id: 'jumps', label: 'Ship jumps', shortLabel: 'Jumps' },
]

export const MAP_LAYER_IDS = new Set<MapLayer>(MAP_LAYERS.map(layer => layer.id))

export const MAP_BASE_LAYERS: Array<{ id: MapBaseLayer; label: string }> = [
    { id: 'geography', label: 'Geography' },
    { id: 'security', label: 'Security' },
]

export const SOVEREIGNTY_BASE_LAYERS: Array<{ id: MapRenderBaseLayer; label: string }> = [
    { id: 'sovereignty', label: 'Sovereignty' },
    { id: 'live', label: 'Live' },
    { id: 'aiid', label: 'AIID' },
    ...MAP_BASE_LAYERS,
]

export const MAP_ACTIVITY_LAYERS: Array<{ id: MapActivityLayer; label: string }> = [
    { id: 'none', label: 'Off' },
    { id: 'activity', label: 'Activity' },
    { id: 'danger', label: 'Danger' },
    { id: 'ship-kills', label: 'Ship kills' },
    { id: 'pod-kills', label: 'Pod kills' },
    { id: 'npc-kills', label: 'NPC kills' },
    { id: 'jumps', label: 'Jumps' },
]

export const MAP_ACTIVITY_WINDOWS: Array<{ value: MapActivityHours; label: string }> = [
    { value: 1, label: '1h' },
    { value: 6, label: '6h' },
    { value: 24, label: '24h' },
    { value: 168, label: '7d' },
]

export function isMapBaseLayer(value: unknown): value is MapBaseLayer {
    return MAP_BASE_LAYERS.some(layer => layer.id === value)
}

export function isMapRenderBaseLayer(value: unknown): value is MapRenderBaseLayer {
    return value === 'sovereignty' || value === 'live' || value === 'aiid' || isMapBaseLayer(value)
}

export function isMapActivityLayer(value: unknown): value is MapActivityLayer {
    return MAP_ACTIVITY_LAYERS.some(layer => layer.id === value)
}

export function isMapActivityHours(value: unknown): value is MapActivityHours {
    return typeof value === 'number' && MAP_ACTIVITY_WINDOWS.some(window => window.value === value)
}

export function isMapLayer(value: unknown): value is MapLayer {
    return typeof value === 'string' && MAP_LAYER_IDS.has(value as MapLayer)
}

export function aiidAlarmBand(distance: number | undefined): AIIDAlarmBand | null {
    if (distance == null || distance < 0 || distance > 10) return null
    return distance < 5 ? 'near' : 'outer'
}

export function activityValue(layer: MapLayer, activity?: MapActivity): number {
    if (!activity) return 0
    if (layer === 'ship-kills') return activity.ship_kills
    if (layer === 'pod-kills') return activity.pod_kills
    if (layer === 'npc-kills') return activity.npc_kills
    if (layer === 'jumps') return activity.ship_jumps
    return 0
}

export function activityLayerLabel(layer: MapLayer): string {
    return MAP_LAYERS.find(candidate => candidate.id === layer)?.label ?? 'Activity'
}

export interface MapActivityMaximums extends MapActivity {}

export function mapActivityValue(layer: MapActivityLayer, activity: MapActivity | undefined, maximums: MapActivityMaximums): number {
    if (!activity || layer === 'none') return 0
    if (layer === 'activity') {
        const score = activityRatio(activity.ship_jumps, maximums.ship_jumps) * 0.35
            + activityRatio(activity.npc_kills, maximums.npc_kills) * 0.25
            + activityRatio(activity.ship_kills, maximums.ship_kills) * 0.25
            + activityRatio(activity.pod_kills, maximums.pod_kills) * 0.15
        return score * 100
    }
    if (layer === 'danger') {
        const losses = activity.ship_kills + activity.pod_kills * 2.5
        return losses * 100 / Math.max(activity.ship_jumps, 20)
    }
    if (layer === 'ship-kills') return activity.ship_kills
    if (layer === 'pod-kills') return activity.pod_kills
    if (layer === 'npc-kills') return activity.npc_kills
    return activity.ship_jumps
}

export function mapActivityRatio(layer: MapActivityLayer, value: number, maximum: number): number {
    if (value <= 0 || maximum <= 0) return 0
    if (layer === 'activity') return Math.min(1, value / maximum)
    return activityRatio(value, maximum)
}

export function mapActivityLayerLabel(layer: MapActivityLayer): string {
    return MAP_ACTIVITY_LAYERS.find(candidate => candidate.id === layer)?.label ?? 'Activity'
}

export function activityRatio(value: number, maximum: number): number {
    if (value <= 0 || maximum <= 0) return 0
    // Log scaling keeps one trade hub from making every other active system
    // indistinguishable from a system with no traffic at all.
    return Math.min(1, Math.log1p(value) / Math.log1p(maximum))
}

export function heatColor(ratio: number): string {
    const stops: Array<[number, [number, number, number]]> = [
        [0, [71, 85, 105]],
        [0.28, [59, 130, 246]],
        [0.55, [168, 85, 247]],
        [0.78, [249, 115, 22]],
        [1, [239, 68, 68]],
    ]
    const value = Math.max(0, Math.min(1, ratio))
    let lower = stops[0]!
    let upper = stops[stops.length - 1]!
    for (let index = 1; index < stops.length; index++) {
        if (value <= stops[index]![0]) {
            lower = stops[index - 1]!
            upper = stops[index]!
            break
        }
    }
    const distance = upper[0] - lower[0]
    const mix = distance > 0 ? (value - lower[0]) / distance : 0
    const rgb = lower[1].map((channel, index) => Math.round(channel + (upper[1][index]! - channel) * mix))
    return `rgb(${rgb[0]}, ${rgb[1]}, ${rgb[2]})`
}

export function sovereigntyAllianceColors(allianceIds: number[]): Map<number, string> {
    const colors = new Map<number, string>()
    allianceIds.forEach((allianceId, index) => {
        // Golden-angle spacing gives the most prominent owners strongly
        // separated hues. The bands guarantee a unique value per list entry.
        const hue = (214 + index * 137.508) % 360
        const saturation = [72, 84, 62][Math.floor(index / 12) % 3]!
        const lightness = [48, 58, 40][Math.floor(index / 3) % 3]!
        colors.set(allianceId, `hsl(${hue.toFixed(3)}, ${saturation}%, ${lightness}%)`)
    })
    return colors
}
