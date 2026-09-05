<script setup lang="ts">
import { Delaunay } from 'd3-delaunay'
import type { Application, Container, Graphics as PixiGraphics, Text as PixiText, Ticker } from 'pixi.js'
import type { KilllistRow } from '#shared/utils/killlistRow'
import type { MapActivity, MapActivityHours, MapActivityLayer, MapRenderBaseLayer } from '~/utils/map/layers'
import { aiidAlarmBand, heatColor, mapActivityLayerLabel, mapActivityRatio, mapActivityValue, sovereigntyAllianceColors } from '~/utils/map/layers'
import { secColorStr } from '~/utils/map/colors'
import { createSovereigntyTerritories, sovereigntyOwnerAt } from '~/utils/map/territories'
import { playAIIDAlarm } from '~/utils/map/aiidAudio'

const props = withDefaults(defineProps<{
    type: string
    baseLayer: MapRenderBaseLayer
    activityLayer: MapActivityLayer
    hours: MapActivityHours
    showConnections: boolean
    showSystems: boolean
    showLabels: boolean
    mode?: 'map' | 'sovereignty' | 'live' | 'aiid'
    showChanges?: boolean
    watchedSystemIds?: number[]
    nearAlarmEnabled?: boolean
    outerAlarmEnabled?: boolean
}>(), { mode: 'map', showChanges: false })

const emit = defineEmits<{
    (event: 'update:watchedSystemIds', value: number[]): void
}>()

const watchedSystemQuery = computed(() => (props.watchedSystemIds ?? []).join(','))
const { data, pending, error } = useApiFetch<any>(() => props.mode === 'sovereignty'
    ? `/api/map/sovereignty?hours=${props.hours}`
    : props.mode === 'aiid'
        ? `/api/map/aiid?systems=${watchedSystemQuery.value}&hours=${props.hours}`
        : `/api/map/scope?type=${props.type}&hours=${props.hours}`, { watch: [() => props.type, () => props.hours, () => props.mode, watchedSystemQuery] })
const isWatchingKills = props.mode === 'live' || props.mode === 'aiid'
const { kills: streamedKills, connected: liveConnected } = useKillStream(isWatchingKills ? ['all'] : null)
const { data: recentLiveData } = useApiFetch<{ kills: KilllistRow[] }>('/api/killlist?type=latest&limit=50', {
    immediate: props.mode === 'live',
    default: () => ({ kills: [] }),
})

interface MapNode {
    id: number
    x: number
    y: number
    region_id: number
    region_name: string
    system_name: string
    security: number
    distance?: number
    is_anchor?: boolean
}
interface MapLink { from: number; to: number }
interface MapCell { points: number[]; region_id: number; security: number }
interface SovereigntyClaim {
    system_id: number
    alliance_id: number
    alliance_name: string
    alliance_ticker: string
    member_count: number
    date_added: string
}
interface AllianceSummary {
    id: number
    name: string
    ticker: string
    memberCount: number
    systems: number
    regions: number
    recentChanges: number
    x: number
    y: number
    labelTop: number
    componentSystems: number
    nodes: MapNode[]
}
interface SovereigntyLabel extends AllianceSummary {
    label: string
    fontSize: number
}
interface Bounds { x: number; y: number; width: number; height: number }
interface RegionSummary {
    id: number
    name: string
    x: number
    y: number
    systems: number
    high: number
    low: number
    null: number
    ship_kills: number
    pod_kills: number
    npc_kills: number
    ship_jumps: number
    busiest?: MapNode
    busiestValue: number
}

const scene = computed(() => {
    const systems = data.value?.systems ?? []
    if (!systems.length) return { nodes: [] as MapNode[], links: [] as MapLink[], cells: [] as MapCell[], bounds: { x: 0, y: 0, width: 1600, height: 1200 }, delaunay: null as Delaunay<MapNode> | null }

    const regionNames = new Map<number, string>()
    for (const region of data.value?.regions ?? []) regionNames.set(region.region_id, region.name ?? '')
    let minX = Infinity, maxX = -Infinity, minZ = Infinity, maxZ = -Infinity
    for (const system of systems) {
        minX = Math.min(minX, system.x); maxX = Math.max(maxX, system.x)
        minZ = Math.min(minZ, system.z); maxZ = Math.max(maxZ, system.z)
    }
    const centerX = (minX + maxX) / 2
    const centerZ = (minZ + maxZ) / 2
    const scale = Math.min(1600 / ((maxX - minX) || 1), 1200 / ((maxZ - minZ) || 1))
    const nodes: MapNode[] = systems.map((system: any) => ({
        id: system.solar_system_id,
        x: (system.x - centerX) * scale,
        y: -(system.z - centerZ) * scale,
        region_id: system.region_id,
        region_name: regionNames.get(system.region_id) ?? '',
        system_name: system.system_name,
        security: system.security,
        distance: system.distance,
        is_anchor: system.is_anchor,
    }))
    const nodeIds = new Set(nodes.map(node => node.id))
    const links: MapLink[] = (data.value?.jumps ?? [])
        .filter((jump: any) => nodeIds.has(jump.from_solar_system_id) && nodeIds.has(jump.to_solar_system_id))
        .map((jump: any) => ({ from: jump.from_solar_system_id, to: jump.to_solar_system_id }))

    let laidMinX = Infinity, laidMaxX = -Infinity, laidMinY = Infinity, laidMaxY = -Infinity
    for (const node of nodes) {
        laidMinX = Math.min(laidMinX, node.x); laidMaxX = Math.max(laidMaxX, node.x)
        laidMinY = Math.min(laidMinY, node.y); laidMaxY = Math.max(laidMaxY, node.y)
    }
    const pad = 80
    const bounds = { x: laidMinX - pad, y: laidMinY - pad, width: laidMaxX - laidMinX + pad * 2, height: laidMaxY - laidMinY + pad * 2 }
    const delaunay = Delaunay.from(nodes, node => node.x, node => node.y)
    const voronoi = delaunay.voronoi([bounds.x, bounds.y, bounds.x + bounds.width, bounds.y + bounds.height])
    const cells: MapCell[] = []
    for (let index = 0; index < nodes.length; index++) {
        const polygon = voronoi.cellPolygon(index)
        if (!polygon) continue
        cells.push({ points: polygon.flatMap(point => [point[0], point[1]]), region_id: nodes[index]!.region_id, security: nodes[index]!.security })
    }
    return { nodes, links, cells, bounds, delaunay }
})

const nodeById = computed(() => new Map(scene.value.nodes.map(node => [node.id, node])))
const liveKills = computed<KilllistRow[]>(() => {
    const seen = new Set<number>()
    const combined: KilllistRow[] = []
    const recentKills = props.mode === 'aiid' ? (data.value?.kills ?? []) : (recentLiveData.value?.kills ?? [])
    for (const kill of [...streamedKills.value, ...recentKills]) {
        if (seen.has(kill.killmail_id) || !nodeById.value.has(kill.solar_system_id)) continue
        seen.add(kill.killmail_id)
        combined.push(kill)
        if (combined.length >= 60) break
    }
    return combined
})
const watchedSystems = computed(() => (props.watchedSystemIds ?? []).map(id => nodeById.value.get(id) ?? {
    id, system_name: `System ${id}`, region_name: '', distance: 0, is_anchor: true,
} as MapNode))
const watchAreaSystemIds = computed(() => new Set(scene.value.nodes.map(node => node.id)))
function addWatchedSystem(picked: { type: string; id: number }) {
    if (picked.type !== 'system' || (props.watchedSystemIds ?? []).includes(picked.id) || (props.watchedSystemIds?.length ?? 0) >= 8) return
    emit('update:watchedSystemIds', [...(props.watchedSystemIds ?? []), picked.id])
}
function removeWatchedSystem(systemId: number) {
    emit('update:watchedSystemIds', (props.watchedSystemIds ?? []).filter(id => id !== systemId))
}
const liveNow = ref(Date.now())
function liveKillAge(iso: string) {
    const seconds = Math.max(0, Math.floor((liveNow.value - new Date(iso).getTime()) / 1000))
    if (seconds < 60) return `${seconds}s ago`
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`
    return `${Math.floor(seconds / 3600)}h ago`
}
const sovereigntyBySystem = computed<Map<number, SovereigntyClaim>>(() => new Map(
    ((data.value?.sovereignty ?? []) as SovereigntyClaim[]).map(claim => [claim.system_id, claim]),
))
const recentChangeSystemIds = computed(() => new Set<number>(((data.value?.changes ?? []) as Array<{ system_id: number }>).map(change => change.system_id)))
const activityById = computed<Map<number, MapActivity>>(() => {
    const activities = new Map<number, MapActivity>()
    for (const row of data.value?.activity ?? []) {
        activities.set(row.system_id, { ship_kills: row.ship_kills ?? 0, pod_kills: row.pod_kills ?? 0, npc_kills: row.npc_kills ?? 0, ship_jumps: row.ship_jumps ?? 0 })
    }
    return activities
})
const activityMaximums = computed<MapActivity>(() => ({
    ship_kills: Math.max(0, ...[...activityById.value.values()].map(activity => activity.ship_kills)),
    pod_kills: Math.max(0, ...[...activityById.value.values()].map(activity => activity.pod_kills)),
    npc_kills: Math.max(0, ...[...activityById.value.values()].map(activity => activity.npc_kills)),
    ship_jumps: Math.max(0, ...[...activityById.value.values()].map(activity => activity.ship_jumps)),
}))
function nodeActivityValue(node: MapNode) {
    return mapActivityValue(props.activityLayer, activityById.value.get(node.id), activityMaximums.value)
}
const maximumActivity = computed(() => Math.max(0, ...scene.value.nodes.map(nodeActivityValue)))
const regionSummaries = computed<RegionSummary[]>(() => {
    const summaries = new Map<number, RegionSummary>()
    for (const node of scene.value.nodes) {
        let summary = summaries.get(node.region_id)
        if (!summary) {
            summary = { id: node.region_id, name: node.region_name, x: 0, y: 0, systems: 0, high: 0, low: 0, null: 0, ship_kills: 0, pod_kills: 0, npc_kills: 0, ship_jumps: 0, busiestValue: -1 }
            summaries.set(node.region_id, summary)
        }
        summary.x += node.x; summary.y += node.y; summary.systems++
        if (node.security >= 0.45) summary.high++
        else if (node.security > 0) summary.low++
        else summary.null++
        const activity = activityById.value.get(node.id)
        summary.ship_kills += activity?.ship_kills ?? 0; summary.pod_kills += activity?.pod_kills ?? 0
        summary.npc_kills += activity?.npc_kills ?? 0; summary.ship_jumps += activity?.ship_jumps ?? 0
        const kills = (activity?.ship_kills ?? 0) + (activity?.pod_kills ?? 0)
        if (kills > summary.busiestValue) { summary.busiestValue = kills; summary.busiest = node }
    }
    return [...summaries.values()].map(summary => ({ ...summary, x: summary.x / summary.systems, y: summary.y / summary.systems }))
})
const summaryById = computed(() => new Map(regionSummaries.value.map(summary => [summary.id, summary])))

const allianceSummaries = computed<AllianceSummary[]>(() => {
    const grouped = new Map<number, { claim: SovereigntyClaim; nodes: MapNode[] }>()
    for (const node of scene.value.nodes) {
        const claim = sovereigntyBySystem.value.get(node.id)
        if (!claim) continue
        const group = grouped.get(claim.alliance_id) ?? { claim, nodes: [] }
        group.nodes.push(node)
        grouped.set(claim.alliance_id, group)
    }
    const adjacency = new Map<number, number[]>()
    for (const link of scene.value.links) {
        const fromClaim = sovereigntyBySystem.value.get(link.from)
        const toClaim = sovereigntyBySystem.value.get(link.to)
        if (!fromClaim || fromClaim.alliance_id !== toClaim?.alliance_id) continue
        const from = nodeById.value.get(link.from), to = nodeById.value.get(link.to)
        if (!from || !to || Math.hypot(from.x - to.x, from.y - to.y) > 45) continue
        adjacency.set(link.from, [...(adjacency.get(link.from) ?? []), link.to])
        adjacency.set(link.to, [...(adjacency.get(link.to) ?? []), link.from])
    }
    return [...grouped.entries()].map(([allianceId, group]) => {
        const byId = new Map(group.nodes.map(node => [node.id, node]))
        const seen = new Set<number>()
        let largest: MapNode[] = []
        for (const node of group.nodes) {
            if (seen.has(node.id)) continue
            const component: MapNode[] = []
            const queue = [node.id]
            seen.add(node.id)
            while (queue.length) {
                const current = queue.pop()!
                const currentNode = byId.get(current)
                if (currentNode) component.push(currentNode)
                for (const neighbor of adjacency.get(current) ?? []) {
                    if (!seen.has(neighbor)) { seen.add(neighbor); queue.push(neighbor) }
                }
            }
            if (component.length > largest.length) largest = component
        }
        const labelNodes = largest.length ? largest : group.nodes
        return {
            id: allianceId,
            name: group.claim.alliance_name,
            ticker: group.claim.alliance_ticker,
            memberCount: group.claim.member_count,
            systems: group.nodes.length,
            regions: new Set(group.nodes.map(node => node.region_id)).size,
            recentChanges: group.nodes.filter(node => recentChangeSystemIds.value.has(node.id)).length,
            x: labelNodes.reduce((sum, node) => sum + node.x, 0) / labelNodes.length,
            y: labelNodes.reduce((sum, node) => sum + node.y, 0) / labelNodes.length,
            labelTop: Math.min(...labelNodes.map(node => node.y)),
            componentSystems: labelNodes.length,
            nodes: group.nodes,
        }
    }).sort((left, right) => right.systems - left.systems || left.id - right.id)
})
const allianceById = computed(() => new Map(allianceSummaries.value.map(alliance => [alliance.id, alliance])))
const sovereigntyLinksByAlliance = computed(() => {
    const grouped = new Map<number, MapLink[]>()
    for (const link of scene.value.links) {
        const fromClaim = sovereigntyBySystem.value.get(link.from)
        const toClaim = sovereigntyBySystem.value.get(link.to)
        if (!fromClaim || fromClaim.alliance_id !== toClaim?.alliance_id) continue
        const from = nodeById.value.get(link.from), to = nodeById.value.get(link.to)
        if (!from || !to || Math.hypot(from.x - to.x, from.y - to.y) > 45) continue
        const links = grouped.get(fromClaim.alliance_id) ?? []
        links.push(link)
        grouped.set(fromClaim.alliance_id, links)
    }
    return grouped
})

const allianceColors = computed(() => sovereigntyAllianceColors(allianceSummaries.value.map(alliance => alliance.id)))
const sovereigntyTerritories = computed(() => props.mode === 'sovereignty' && sovereigntyBySystem.value.size
    ? createSovereigntyTerritories(
        scene.value.nodes,
        scene.value.links,
        scene.value.bounds,
        new Map([...sovereigntyBySystem.value].map(([systemId, claim]) => [systemId, claim.alliance_id])),
    )
    : null)
const territoryByAlliance = computed(() => new Map(sovereigntyTerritories.value?.territories.map(territory => [territory.allianceId, territory]) ?? []))

function allianceColor(allianceId: number) {
    return allianceColors.value.get(allianceId) ?? '#64748b'
}

const sovereigntyLabels = computed<SovereigntyLabel[]>(() => {
    const placed: Array<{ left: number; right: number; top: number; bottom: number }> = []
    const result: SovereigntyLabel[] = []
    for (const alliance of allianceSummaries.value) {
        if (alliance.componentSystems < 3) continue
        const fontSize = Math.min(25, Math.max(10, 8 + Math.sqrt(alliance.componentSystems) * 1.45))
        const label = alliance.componentSystems >= 12 ? alliance.name : alliance.ticker
        const width = Math.max(30, label.length * fontSize * 0.56)
        const height = fontSize * 1.35
        const box = { left: alliance.x - width / 2, right: alliance.x + width / 2, top: alliance.y - height / 2, bottom: alliance.y + height / 2 }
        const overlaps = placed.some(other => !(box.right + 5 < other.left || box.left - 5 > other.right || box.bottom + 4 < other.top || box.top - 4 > other.bottom))
        if (overlaps) continue
        placed.push(box)
        result.push({ ...alliance, label, fontSize })
    }
    return result
})

function regionColor(regionId: number) { return `hsl(${((regionId % 1000) * 137.508) % 360}, 60%, 55%)` }
function nodeColor(node: MapNode) {
    if (props.activityLayer === 'none') {
        const claim = sovereigntyBySystem.value.get(node.id)
        if (props.baseLayer === 'sovereignty') return claim ? allianceColor(claim.alliance_id) : '#475569'
        if (props.baseLayer === 'live') return '#475569'
        if (props.baseLayer === 'aiid') return node.is_anchor ? '#67e8f9' : secColorStr(node.security)
        return secColorStr(node.security)
    }
    return heatColor(mapActivityRatio(props.activityLayer, nodeActivityValue(node), maximumActivity.value))
}
function nodeRadius(node: MapNode) {
    if (props.activityLayer === 'none') {
        if (props.baseLayer === 'live') return 1.15
        if (props.baseLayer === 'aiid') return node.is_anchor ? 5.2 : Math.max(1.8, 3.4 - (node.distance ?? 10) * 0.16)
        return 1.7
    }
    return 1.4 + mapActivityRatio(props.activityLayer, nodeActivityValue(node), maximumActivity.value) * 4.5
}
const activityWindowLabel = computed(() => props.hours === 168 ? '7 days' : `${props.hours} hour${props.hours === 1 ? '' : 's'}`)
const activeLayerLabel = computed(() => mapActivityLayerLabel(props.activityLayer))
const maximumActivityLabel = computed(() => {
    if (props.activityLayer === 'danger') return `${maximumActivity.value.toFixed(1)} / 100 jumps`
    if (props.activityLayer === 'activity') return `${Math.round(maximumActivity.value)} score`
    return formatNumber(Math.round(maximumActivity.value))
})

const hostRef = ref<HTMLElement | null>(null)
const canvasHostRef = ref<HTMLElement | null>(null)
const ready = ref(false)
const initError = ref<string | null>(null)
const initStage = ref('Waiting for the browser')
const hoveredRegionId = ref<number | null>(null)
const hoveredAllianceId = ref<number | null>(null)
const hoverCardVisible = ref(false)
const hoveredLiveKillId = ref<number | null>(null)
const hoveredLiveSystemId = ref<number | null>(null)
const liveLeaderOrigin = ref({ x: 0, y: 0 })
const hoverPos = ref({ x: 0, y: 0 })
const hoveredSummary = computed(() => hoveredRegionId.value == null ? undefined : summaryById.value.get(hoveredRegionId.value))
const hoveredAlliance = computed(() => hoveredAllianceId.value == null ? undefined : allianceById.value.get(hoveredAllianceId.value))
const searchQuery = ref('')
const searchFocused = ref(false)
function closeSearchSoon() { window.setTimeout(() => { searchFocused.value = false }, 150) }
const searchResults = computed(() => {
    const query = searchQuery.value.trim().toLowerCase()
    if (!query) return []
    if (props.mode === 'sovereignty') {
        const alliances = allianceSummaries.value
            .filter(alliance => alliance.name.toLowerCase().includes(query) || alliance.ticker.toLowerCase().includes(query))
            .slice(0, 5)
            .map(alliance => ({ id: alliance.id, name: alliance.name, type: `[${alliance.ticker}] Alliance`, to: `/alliance/${alliance.id}` }))
        const systems = scene.value.nodes
            .filter(system => system.system_name.toLowerCase().includes(query))
            .slice(0, Math.max(0, 8 - alliances.length))
            .map(system => ({ id: system.id, name: system.system_name, type: sovereigntyBySystem.value.get(system.id)?.alliance_name ?? system.region_name, to: `/system/${system.id}` }))
        return [...alliances, ...systems]
    }
    const regions = regionSummaries.value.filter(region => region.name.toLowerCase().includes(query)).slice(0, 5).map(region => ({ id: region.id, name: region.name, type: 'Region', to: mapRegionLocation(region.id) }))
    const systems = scene.value.nodes.filter(system => system.system_name.toLowerCase().includes(query)).slice(0, Math.max(0, 8 - regions.length)).map(system => ({ id: system.id, name: system.system_name, type: system.region_name, to: `/system/${system.id}` }))
    return [...regions, ...systems]
})
function mapRegionLocation(regionId: number) {
    return {
        path: `/map/region/${regionId}`,
        query: {
            base: props.baseLayer,
            activity: props.activityLayer,
            hours: props.hours,
            routes: props.showConnections ? undefined : '0',
            systems: props.showSystems ? undefined : '0',
            labels: props.showLabels ? undefined : '0',
        },
    }
}
const hoverCardStyle = computed(() => {
    const host = hostRef.value
    if (!host || (!hoveredSummary.value && !hoveredAlliance.value)) return { display: 'none' }
    const width = 280, height = props.mode === 'sovereignty' ? 150 : 190, x = hoverPos.value.x, y = hoverPos.value.y
    return { left: `${Math.max(8, x + width + 24 > host.clientWidth ? x - width - 14 : x + 14)}px`, top: `${Math.max(8, y + height + 24 > host.clientHeight ? y - height - 14 : y + 14)}px` }
})

let app: Application | null = null
let world: Container | null = null
let baseSceneLayer: Container | null = null
let hoverLayer: Container | null = null
let livePulseGlow: PixiGraphics | null = null
let livePulseCore: PixiGraphics | null = null
let liveLeaderLayer: PixiGraphics | null = null
let labels: PixiText[] = []
let pixiClasses: Pick<typeof import('pixi.js'), 'Graphics' | 'Container' | 'Text' | 'BlurFilter'> | null = null
let resizeObserver: ResizeObserver | null = null
let dragging = false
let dragMoved = false
let dragPoint = { x: 0, y: 0 }
let baseAlphaTarget = 1
let liveClockTimer: ReturnType<typeof setInterval> | null = null
const livePulses = new Map<number, { startedAt: number; value: number }>()

function updateBaseAlpha(ticker: Ticker) {
    if (!baseSceneLayer) return
    const blend = Math.min(1, ticker.deltaMS / 90)
    baseSceneLayer.alpha += (baseAlphaTarget - baseSceneLayer.alpha) * blend
    if (Math.abs(baseAlphaTarget - baseSceneLayer.alpha) < 0.002) baseSceneLayer.alpha = baseAlphaTarget
}

function updateLivePulses() {
    if (!livePulseGlow || !livePulseCore || (props.mode !== 'live' && props.mode !== 'aiid')) return
    livePulseGlow.clear(); livePulseCore.clear()
    const now = performance.now()
    const lifetime = props.mode === 'aiid' ? 15 * 60 : 45
    for (const [systemId, pulse] of livePulses) {
        const node = nodeById.value.get(systemId)
        const age = (now - pulse.startedAt) / 1000
        if (!node || age > lifetime) { livePulses.delete(systemId); continue }
        const life = 1 - age / lifetime
        const ringLife = Math.max(0, 1 - age / (props.mode === 'aiid' ? 30 : 18))
        const valueScale = Math.min(1, Math.max(0, Math.log10(Math.max(pulse.value, 1)) - 6) / 5)
        const color = valueScale > 0.72 ? '#facc15' : valueScale > 0.38 ? '#fb923c' : '#fb7185'
        const coreRadius = 3.5 + valueScale * 4 + Math.sin(age * 5) * 0.8
        livePulseGlow.circle(node.x, node.y, coreRadius * 2.4).fill({ color, alpha: 0.42 * life })
        livePulseCore.circle(node.x, node.y, Math.max(1.5, coreRadius * life)).fill({ color, alpha: 0.95 * life })
        if (ringLife > 0) livePulseCore.circle(node.x, node.y, coreRadius + age * 2.2).stroke({ color, alpha: 0.85 * ringLife, width: 1.5 })
        if (age < 5) livePulseCore.circle(node.x, node.y, coreRadius * 0.35).fill({ color: '#ffffff', alpha: 0.9 - age * 0.12 })
    }
    if (!liveLeaderLayer || !world) return
    liveLeaderLayer.clear()
    const highlighted = hoveredLiveSystemId.value == null ? undefined : nodeById.value.get(hoveredLiveSystemId.value)
    if (!highlighted) return
    const target = { x: world.x + highlighted.x * world.scale.x, y: world.y + highlighted.y * world.scale.y }
    const origin = liveLeaderOrigin.value
    const bend = Math.max(40, (target.x - origin.x) * 0.38)
    liveLeaderLayer.moveTo(origin.x, origin.y).bezierCurveTo(origin.x + bend, origin.y, target.x - bend * 0.35, target.y, target.x, target.y).stroke({ color: '#fb7185', alpha: 0.72, width: 1.4 })
    const pinnedRadius = 9 + Math.sin(now / 130) * 2
    liveLeaderLayer.circle(target.x, target.y, pinnedRadius).fill({ color: '#fb7185', alpha: 0.22 }).stroke({ color: '#ffffff', alpha: 0.9, width: 1.4 })
    liveLeaderLayer.circle(target.x, target.y, 3.5).fill({ color: '#ffffff', alpha: 1 })
}

function canvasPoint(event: PointerEvent | WheelEvent) {
    if (!app) return { x: 0, y: 0 }
    const rect = app.canvas.getBoundingClientRect()
    return { x: (event.clientX - rect.left) * app.screen.width / rect.width, y: (event.clientY - rect.top) * app.screen.height / rect.height }
}
function worldPoint(point: { x: number; y: number }) {
    if (!world) return point
    return { x: (point.x - world.x) / world.scale.x, y: (point.y - world.y) / world.scale.y }
}
function resetView() {
    if (!app || !world) return
    const bounds = scene.value.bounds
    const hasPanel = props.mode !== 'map'
    const mobile = window.matchMedia('(max-width: 639px)').matches
    const panel = hostRef.value?.querySelector('aside')?.getBoundingClientRect()
    const host = hostRef.value?.getBoundingClientRect()
    // Fit the camera to the visible space, using the actual panel geometry so
    // sidebar and bottom-sheet layouts cannot drift apart from the camera.
    const leftInset = hasPanel && !mobile && panel && host ? panel.right - host.left + 12 : 0
    const topInset = mobile ? (hasPanel ? 52 : 108) : 0
    const bottomEdge = hasPanel && mobile && panel && host ? panel.top - host.top - 12 : app.screen.height - (mobile ? 56 : 0)
    const availableHeight = Math.max(1, bottomEdge - topInset)
    const availableWidth = app.screen.width - leftInset
    const scale = Math.min(availableWidth / bounds.width, availableHeight / bounds.height) * 0.96
    world.scale.set(scale)
    world.position.set(leftInset + availableWidth / 2 - (bounds.x + bounds.width / 2) * scale, topInset + availableHeight / 2 - (bounds.y + bounds.height / 2) * scale)
}
function focusAllianceFromList(allianceId: number) {
    hoverCardVisible.value = false
    hoveredAllianceId.value = allianceId
    drawHover()
}
function clearAllianceFromList(allianceId: number) {
    if (hoveredAllianceId.value !== allianceId || hoverCardVisible.value) return
    hoveredAllianceId.value = null
    drawHover()
}
function focusLiveKill(kill: KilllistRow, event: MouseEvent) {
    const host = hostRef.value
    const row = event.currentTarget as HTMLElement | null
    if (!host || !row || !nodeById.value.has(kill.solar_system_id)) return
    const hostRect = host.getBoundingClientRect(), rowRect = row.getBoundingClientRect()
    hoveredLiveKillId.value = kill.killmail_id
    hoveredLiveSystemId.value = kill.solar_system_id
    liveLeaderOrigin.value = { x: rowRect.right - hostRect.left, y: rowRect.top + rowRect.height / 2 - hostRect.top }
}
function clearLiveKill(killmailId: number) {
    if (hoveredLiveKillId.value !== killmailId) return
    hoveredLiveKillId.value = null
    hoveredLiveSystemId.value = null
}
function drawLegacyAllianceTerritory(target: Container, allianceId: number, options: { alpha?: number; emphasized?: boolean } = {}) {
    if (!pixiClasses) return
    const alliance = allianceById.value.get(allianceId)
    if (!alliance) return
    const alpha = options.alpha ?? 0.46
    const outline = new pixiClasses.Graphics()
    const fill = new pixiClasses.Graphics()
    for (const link of sovereigntyLinksByAlliance.value.get(allianceId) ?? []) {
        const from = nodeById.value.get(link.from), to = nodeById.value.get(link.to)
        if (!from || !to) continue
        outline.moveTo(from.x, from.y).lineTo(to.x, to.y)
        fill.moveTo(from.x, from.y).lineTo(to.x, to.y)
    }
    outline.stroke({ color: options.emphasized ? '#e2e8f0' : '#030407', alpha: options.emphasized ? 0.72 : 0.88, width: options.emphasized ? 39 : 38 })
    fill.stroke({ color: allianceColor(allianceId), alpha, width: options.emphasized ? 36 : 34 })
    for (const node of alliance.nodes) {
        outline.circle(node.x, node.y, options.emphasized ? 20.5 : 20).fill({ color: options.emphasized ? '#e2e8f0' : '#030407', alpha: options.emphasized ? 0.72 : 0.88 })
        fill.circle(node.x, node.y, options.emphasized ? 19 : 18).fill({ color: allianceColor(allianceId), alpha })
    }
    target.addChild(outline, fill)
}
function drawAllianceTerritory(target: Container, allianceId: number, options: { alpha?: number; border?: boolean; emphasized?: boolean } = {}) {
    if (!pixiClasses) return
    const territory = territoryByAlliance.value.get(allianceId)
    if (!territory) {
        drawLegacyAllianceTerritory(target, allianceId, options)
        return
    }
    const alpha = options.alpha ?? 0.5
    for (const polygon of territory.polygons) {
        const outer = polygon.rings[0]
        if (!outer?.length) continue
        const fill = new pixiClasses.Graphics()
        fill.poly(outer).fill({ color: allianceColor(allianceId), alpha })
        for (const hole of polygon.rings.slice(1)) {
            if (hole.length >= 8) fill.poly(hole).cut()
        }
        target.addChild(fill)
        if (options.border === false) continue
        const border = new pixiClasses.Graphics()
        border.poly(outer).stroke({ color: options.emphasized ? '#f8fafc' : '#05060a', alpha: options.emphasized ? 0.9 : 0.92, width: options.emphasized ? 3.2 : 2.2 })
        for (const hole of polygon.rings.slice(1)) {
            if (hole.length >= 8) border.poly(hole).stroke({ color: options.emphasized ? '#f8fafc' : '#05060a', alpha: options.emphasized ? 0.8 : 0.85, width: options.emphasized ? 2.4 : 1.6 })
        }
        target.addChild(border)
    }
}
function drawHover() {
    if (!hoverLayer || !pixiClasses) return
    const removed = hoverLayer.removeChildren()
    for (const child of removed) child.destroy()
    if (props.mode === 'live' || props.mode === 'aiid') {
        baseAlphaTarget = 1
        return
    }
    if (props.mode === 'sovereignty') {
        const allianceId = hoveredAllianceId.value
        baseAlphaTarget = allianceId == null ? 1 : 0.16
        if (allianceId == null) return
        drawAllianceTerritory(hoverLayer, allianceId, { alpha: 0.7, emphasized: true })
        const alliance = allianceById.value.get(allianceId)
        if (!alliance) return
        if (props.showConnections) {
            const routes = new pixiClasses.Graphics()
            for (const link of sovereigntyLinksByAlliance.value.get(allianceId) ?? []) {
                const from = nodeById.value.get(link.from), to = nodeById.value.get(link.to)
                if (from && to) routes.moveTo(from.x, from.y).lineTo(to.x, to.y)
            }
            routes.stroke({ color: '#f8fafc', alpha: 0.75, width: 0.9 })
            hoverLayer.addChild(routes)
        }
        if (props.showSystems) {
            const systems = new pixiClasses.Graphics()
            for (const node of alliance.nodes) systems.circle(node.x, node.y, Math.max(2.2, nodeRadius(node))).fill({ color: nodeColor(node), alpha: 1 })
            hoverLayer.addChild(systems)
        }
        const label = new pixiClasses.Text({
            text: `${alliance.name.toUpperCase()} [${alliance.ticker}]`,
            style: { fill: '#ffffff', fontFamily: 'sans-serif', fontSize: 24, fontWeight: '800', letterSpacing: 1.8, stroke: { color: '#030407', width: 6 } },
        })
        label.anchor.set(0.5, 1)
        label.position.set(alliance.x, alliance.labelTop - 18)
        hoverLayer.addChild(label)
        return
    }
    const regionId = hoveredRegionId.value
    baseAlphaTarget = regionId == null ? 1 : 0.22
    if (regionId == null) return
    const highlight = new pixiClasses.Graphics()
    for (const cell of scene.value.cells) {
        if (cell.region_id === regionId) highlight.poly(cell.points).fill({ color: props.baseLayer === 'geography' ? regionColor(regionId) : '#60a5fa', alpha: props.baseLayer === 'geography' ? 0.2 : 0.08 })
    }
    if (props.showConnections) {
        for (const link of scene.value.links) {
            const from = nodeById.value.get(link.from), to = nodeById.value.get(link.to)
            if (from && to && (from.region_id === regionId || to.region_id === regionId)) highlight.moveTo(from.x, from.y).lineTo(to.x, to.y)
        }
        highlight.stroke({ color: '#94a3b8', alpha: 0.62, width: 0.8 })
    }
    const regionNodes = scene.value.nodes.filter(node => node.region_id === regionId)
    if (props.showSystems) {
        for (const node of regionNodes) highlight.circle(node.x, node.y, nodeRadius(node)).fill({ color: nodeColor(node), alpha: 1 })
    }
    hoverLayer.addChild(highlight)

    const summary = summaryById.value.get(regionId)
    if (summary && regionNodes.length) {
        const top = Math.min(...regionNodes.map(node => node.y))
        const label = new pixiClasses.Text({
            text: summary.name.toUpperCase(),
            style: { fill: '#ffffff', fontFamily: 'sans-serif', fontSize: 22, fontWeight: '800', letterSpacing: 2.4, stroke: { color: '#05060a', width: 5 } },
        })
        label.anchor.set(0.5, 1)
        label.position.set(summary.x, top - 14)
        hoverLayer.addChild(label)
    }
}
function drawScene() {
    if (!world || !app || !pixiClasses) return
    const removed = world.removeChildren()
    for (const child of removed) child.destroy()
    labels = []
    const cells = new pixiClasses.Graphics()
    const links = new pixiClasses.Graphics()
    const nodes = new pixiClasses.Graphics()
    const changes = new pixiClasses.Graphics()
    livePulseGlow = new pixiClasses.Graphics()
    livePulseCore = new pixiClasses.Graphics()
    livePulseGlow.filters = [new pixiClasses.BlurFilter({ strength: 10, quality: 2, kernelSize: 9 })]
    const territoryGlowLayer = new pixiClasses.Container()
    const territoryLayer = new pixiClasses.Container()
    const labelLayer = new pixiClasses.Container()
    hoverLayer = new pixiClasses.Container()
    baseSceneLayer = new pixiClasses.Container()

    if (props.baseLayer === 'sovereignty') {
        for (const alliance of allianceSummaries.value) {
            drawAllianceTerritory(territoryGlowLayer, alliance.id, { alpha: 0.22, border: false })
            drawAllianceTerritory(territoryLayer, alliance.id, { alpha: 0.52 })
        }
        territoryGlowLayer.filters = [new pixiClasses.BlurFilter({ strength: 8, quality: 2, kernelSize: 9 })]
    } else if (props.baseLayer !== 'live' && props.baseLayer !== 'aiid') {
        for (const cell of scene.value.cells) {
            const color = props.baseLayer === 'geography' ? regionColor(cell.region_id) : secColorStr(cell.security)
            cells.poly(cell.points).fill({ color, alpha: props.baseLayer === 'geography' ? 0.1 : 0.065 })
        }
    }
    if (props.showConnections) {
        for (const link of scene.value.links) {
            const from = nodeById.value.get(link.from), to = nodeById.value.get(link.to)
            if (from && to) links.moveTo(from.x, from.y).lineTo(to.x, to.y)
        }
        links.stroke({ color: '#64748b', alpha: props.baseLayer === 'live' ? 0.1 : props.baseLayer === 'aiid' ? 0.34 : props.baseLayer === 'geography' ? 0.28 : 0.2, width: props.baseLayer === 'live' ? 0.4 : props.baseLayer === 'aiid' ? 0.8 : 0.55 })
    }
    if (props.showSystems) {
        for (const node of scene.value.nodes) {
            const radius = nodeRadius(node)
            nodes.circle(node.x, node.y, radius).fill({ color: nodeColor(node), alpha: props.activityLayer === 'none' ? 0.82 : 0.95 })
            if (props.activityLayer !== 'none') nodes.circle(node.x, node.y, radius + 0.35).stroke({ color: secColorStr(node.security), alpha: 0.8, width: 0.45 })
            if (props.mode === 'aiid' && node.is_anchor) nodes.circle(node.x, node.y, radius + 3.5).stroke({ color: '#67e8f9', alpha: 0.85, width: 1.4 })
        }
    }
    if (props.mode === 'sovereignty' && props.showChanges) {
        for (const systemId of recentChangeSystemIds.value) {
            const node = nodeById.value.get(systemId)
            if (node) changes.circle(node.x, node.y, 5.2).stroke({ color: '#facc15', alpha: 0.9, width: 1.1 })
        }
    }
    if (props.showLabels) {
        if (props.baseLayer === 'sovereignty') {
            for (const summary of sovereigntyLabels.value) {
                const label = new pixiClasses.Text({ text: summary.label, style: { fill: allianceColor(summary.id), fontFamily: 'sans-serif', fontSize: summary.fontSize, fontWeight: '800', letterSpacing: 0.8, stroke: { color: '#030407', width: 4 } } })
                label.anchor.set(0.5); label.position.set(summary.x, summary.y); labels.push(label); labelLayer.addChild(label)
            }
        } else if (props.baseLayer === 'aiid') {
            for (const node of scene.value.nodes.filter(node => node.is_anchor || (node.distance ?? 99) <= 2)) {
                const anchor = node.is_anchor
                const label = new pixiClasses.Text({ text: node.system_name, style: { fill: anchor ? '#a5f3fc' : 'rgba(226,232,240,.7)', fontFamily: 'sans-serif', fontSize: anchor ? 15 : 10, fontWeight: anchor ? '800' : '600', letterSpacing: anchor ? 0.9 : 0.3, stroke: { color: '#020617', width: anchor ? 4 : 3 } } })
                label.anchor.set(0.5, 1); label.position.set(node.x, node.y - (anchor ? 8 : 4)); labels.push(label); labelLayer.addChild(label)
            }
        } else {
            for (const summary of regionSummaries.value) {
                const label = new pixiClasses.Text({ text: summary.name.toUpperCase(), style: { fill: 'rgba(255,255,255,.68)', fontFamily: 'sans-serif', fontSize: 13, fontWeight: '700', letterSpacing: 1.5, stroke: { color: '#000000', width: 3 } } })
                label.anchor.set(0.5); label.position.set(summary.x, summary.y); labels.push(label); labelLayer.addChild(label)
            }
        }
    }
    baseSceneLayer.addChild(cells, territoryGlowLayer, territoryLayer, links, nodes, changes, livePulseGlow, livePulseCore, labelLayer)
    baseSceneLayer.alpha = baseAlphaTarget
    world.addChild(baseSceneLayer, hoverLayer)
    drawHover()
    updateLivePulses()
}
function onPointerMove(event: PointerEvent) {
    if (!app || !world) return
    const point = canvasPoint(event)
    hoverPos.value = { x: event.clientX - app.canvas.getBoundingClientRect().left, y: event.clientY - app.canvas.getBoundingClientRect().top }
    if (dragging) {
        const dx = point.x - dragPoint.x, dy = point.y - dragPoint.y
        if (Math.abs(dx) + Math.abs(dy) > 2) dragMoved = true
        world.x += dx; world.y += dy; dragPoint = point
        hoveredRegionId.value = null; hoveredAllianceId.value = null; hoverCardVisible.value = false; drawHover(); return
    }
    const local = worldPoint(point)
    const bounds = scene.value.bounds
    const inside = local.x >= bounds.x && local.x <= bounds.x + bounds.width && local.y >= bounds.y && local.y <= bounds.y + bounds.height
    const index = inside ? scene.value.delaunay?.find(local.x, local.y) : undefined
    const nearest = index == null ? undefined : scene.value.nodes[index]
    if (props.mode === 'sovereignty') {
        const allianceId = sovereigntyTerritories.value ? sovereigntyOwnerAt(sovereigntyTerritories.value, local.x, local.y) : null
        hoverCardVisible.value = allianceId != null
        if (allianceId !== hoveredAllianceId.value) { hoveredAllianceId.value = allianceId; drawHover() }
    } else if (props.mode === 'map') {
        const regionId = nearest?.region_id ?? null
        if (regionId !== hoveredRegionId.value) { hoveredRegionId.value = regionId; drawHover() }
    }
}
function onPointerDown(event: PointerEvent) {
    if (event.button !== 0) return
    dragging = true; dragMoved = false; dragPoint = canvasPoint(event)
    app?.canvas.setPointerCapture(event.pointerId)
}
function onPointerUp(event: PointerEvent) {
    if (!dragging) return
    dragging = false
    app?.canvas.releasePointerCapture(event.pointerId)
    if (dragMoved) return
    if (props.mode === 'sovereignty' && hoveredAllianceId.value != null) navigateTo(`/alliance/${hoveredAllianceId.value}`)
    else if (hoveredRegionId.value != null) navigateTo(mapRegionLocation(hoveredRegionId.value))
}
function onPointerLeave() {
    if (dragging) return
    hoveredRegionId.value = null; hoveredAllianceId.value = null; hoverCardVisible.value = false; drawHover()
}
function onWheel(event: WheelEvent) {
    if (!world) return
    event.preventDefault()
    const point = canvasPoint(event), before = worldPoint(point)
    const next = Math.min(10, Math.max(0.25, world.scale.x * (event.deltaY < 0 ? 1.15 : 1 / 1.15)))
    world.scale.set(next); world.position.set(point.x - before.x * next, point.y - before.y * next)
}

watch(() => streamedKills.value[0]?.killmail_id, (killmailId) => {
    if (!killmailId || (props.mode !== 'live' && props.mode !== 'aiid')) return
    const kill = streamedKills.value[0]
    if (!kill || !nodeById.value.has(kill.solar_system_id)) return
    livePulses.set(kill.solar_system_id, { startedAt: performance.now(), value: kill.total_value ?? 0 })
    if (props.mode === 'aiid') {
        const band = aiidAlarmBand(nodeById.value.get(kill.solar_system_id)?.distance)
        if ((band === 'near' && props.nearAlarmEnabled) || (band === 'outer' && props.outerAlarmEnabled)) playAIIDAlarm(band)
    }
})

watch(watchedSystemQuery, () => {
    livePulses.clear()
    hoveredLiveKillId.value = null
    hoveredLiveSystemId.value = null
})

watch(() => data.value?.kills, (kills: KilllistRow[] | undefined) => {
    if (!import.meta.client || props.mode !== 'aiid') return
    const now = Date.now()
    for (const kill of kills ?? []) {
        const age = Math.max(0, now - new Date(kill.killmail_time).getTime())
        if (age > 15 * 60 * 1000 || !watchAreaSystemIds.value.has(kill.solar_system_id)) continue
        if (!livePulses.has(kill.solar_system_id)) {
            livePulses.set(kill.solar_system_id, { startedAt: performance.now() - age, value: kill.total_value ?? 0 })
        }
    }
}, { immediate: true })

onMounted(async () => {
    if (!hostRef.value || !canvasHostRef.value) return
    try {
        initStage.value = 'Loading map renderer'
        const pixi = await import('pixi.js')
        initStage.value = 'Creating renderer'
        pixiClasses = { Graphics: pixi.Graphics, Container: pixi.Container, Text: pixi.Text, BlurFilter: pixi.BlurFilter }
        app = new pixi.Application()
        await app.init({ resizeTo: hostRef.value, backgroundAlpha: 0, antialias: true, autoDensity: true, resolution: Math.min(window.devicePixelRatio || 1, 2), preference: 'webgl' })
        initStage.value = 'Drawing New Eden'
        app.canvas.className = 'absolute inset-0 h-full w-full touch-none'
        canvasHostRef.value.appendChild(app.canvas)
        world = new pixi.Container()
        app.stage.addChild(world)
        liveLeaderLayer = new pixi.Graphics()
        app.stage.addChild(liveLeaderLayer)
        app.ticker.add(updateBaseAlpha)
        app.ticker.add(updateLivePulses)
        liveClockTimer = setInterval(() => { liveNow.value = Date.now() }, 15000)
        drawScene(); resetView(); ready.value = true
        app.canvas.addEventListener('pointermove', onPointerMove)
        app.canvas.addEventListener('pointerdown', onPointerDown)
        app.canvas.addEventListener('pointerup', onPointerUp)
        app.canvas.addEventListener('pointercancel', onPointerUp)
        app.canvas.addEventListener('pointerleave', onPointerLeave)
        app.canvas.addEventListener('wheel', onWheel, { passive: false })
        resizeObserver = new ResizeObserver(() => window.requestAnimationFrame(resetView))
        resizeObserver.observe(hostRef.value)
    } catch (cause) {
        initError.value = cause instanceof Error ? cause.message : 'Unable to initialise the GPU renderer'
    }
})

watch(
    () => [props.baseLayer, props.activityLayer, props.showConnections, props.showSystems, props.showLabels, props.showChanges, props.mode],
    () => { if (app) drawScene() },
)
watch(data, () => { if (app) { drawScene(); resetView() } })
onUnmounted(() => {
    resizeObserver?.disconnect()
    if (app) {
        app.ticker.remove(updateBaseAlpha)
        app.ticker.remove(updateLivePulses)
        app.canvas.removeEventListener('pointermove', onPointerMove)
        app.canvas.removeEventListener('pointerdown', onPointerDown)
        app.canvas.removeEventListener('pointerup', onPointerUp)
        app.canvas.removeEventListener('pointercancel', onPointerUp)
        app.canvas.removeEventListener('pointerleave', onPointerLeave)
        app.canvas.removeEventListener('wheel', onWheel)
        app.destroy(true, { children: true })
    }
    if (liveClockTimer) clearInterval(liveClockTimer)
    livePulses.clear()
    app = null; world = null; baseSceneLayer = null; hoverLayer = null; livePulseGlow = null; livePulseCore = null; liveLeaderLayer = null; pixiClasses = null
})
</script>

<template>
    <div ref="hostRef" class="relative h-[78vh] overflow-hidden rounded-lg border border-white/[0.08] bg-[#08090d]">
        <div ref="canvasHostRef" class="absolute inset-0" />
        <div v-if="mode === 'map'" class="absolute left-3 right-3 top-3 z-30 w-auto sm:right-auto sm:w-[min(20rem,calc(100%-6rem))]">
            <div class="flex items-center gap-2 rounded-lg border border-white/[0.1] bg-black/70 px-3 py-2 shadow-lg backdrop-blur-md">
                <span aria-hidden="true" class="shrink-0 text-sm text-gray-500">⌕</span>
                <input v-model="searchQuery" type="search" placeholder="Find a region or system" class="min-w-0 flex-1 bg-transparent text-sm text-white outline-none placeholder:text-gray-600" @focus="searchFocused = true" @blur="closeSearchSoon">
            </div>
            <div v-if="searchFocused && searchQuery" class="mt-1 overflow-hidden rounded-lg border border-white/[0.1] bg-[#101116]/95 shadow-xl backdrop-blur-md">
                <NuxtLink v-for="result in searchResults" :key="`${result.type}-${result.id}`" :to="result.to" class="flex items-center justify-between px-3 py-2 text-sm hover:bg-white/[0.06]"><span class="text-gray-200">{{ result.name }}</span><span class="text-[10px] text-gray-500">{{ result.type }}</span></NuxtLink>
                <div v-if="!searchResults.length" class="px-3 py-2 text-xs text-gray-500">No matching regions or systems</div>
            </div>
        </div>
        <aside v-else-if="mode === 'sovereignty'" class="absolute inset-x-3 bottom-28 z-20 flex h-[44%] min-w-0 flex-col overflow-hidden rounded-lg border border-white/[0.1] bg-[#0b0c11]/94 shadow-2xl backdrop-blur-md sm:inset-y-3 sm:left-3 sm:right-auto sm:h-auto sm:w-[min(300px,30%)] sm:min-w-[260px]">
            <div class="border-b border-white/[0.07] px-3 py-3">
                <div class="flex items-center justify-between gap-2">
                    <div><div class="text-[10px] font-bold uppercase tracking-[0.15em] text-blue-300">Sovereignty holders</div><div class="mt-0.5 text-[10px] text-gray-600">Ranked by claimed systems</div></div>
                    <span class="font-mono text-xs text-gray-500">{{ allianceSummaries.length }}</span>
                </div>
                <div class="relative mt-2">
                    <div class="flex items-center gap-2 rounded border border-white/[0.08] bg-black/35 px-2.5 py-1.5">
                        <span aria-hidden="true" class="shrink-0 text-xs text-gray-600">⌕</span>
                        <input v-model="searchQuery" type="search" placeholder="Alliance or system" class="min-w-0 flex-1 bg-transparent text-xs text-white outline-none placeholder:text-gray-600" @focus="searchFocused = true" @blur="closeSearchSoon">
                    </div>
                    <div v-if="searchFocused && searchQuery" class="absolute left-0 right-0 top-full z-30 mt-1 overflow-hidden rounded border border-white/[0.1] bg-[#101116]/98 shadow-xl">
                        <NuxtLink v-for="result in searchResults" :key="`${result.type}-${result.id}`" :to="result.to" class="flex items-center justify-between gap-2 px-2.5 py-2 text-xs hover:bg-white/[0.06]"><span class="truncate text-gray-200">{{ result.name }}</span><span class="shrink-0 text-[9px] text-gray-500">{{ result.type }}</span></NuxtLink>
                        <div v-if="!searchResults.length" class="px-2.5 py-2 text-[10px] text-gray-500">No matching alliances or systems</div>
                    </div>
                </div>
            </div>
            <div class="min-h-0 flex-1 overflow-y-auto overscroll-contain py-1.5">
                <NuxtLink
                    v-for="(alliance, index) in allianceSummaries"
                    :key="alliance.id"
                    :to="`/alliance/${alliance.id}`"
                    class="group relative flex items-center gap-2 px-2.5 py-1.5 transition-colors hover:bg-white/[0.06]"
                    @mouseenter="focusAllianceFromList(alliance.id)"
                    @mouseleave="clearAllianceFromList(alliance.id)"
                >
                    <span class="w-5 shrink-0 text-right font-mono text-[10px] text-gray-600">{{ index + 1 }}</span>
                    <span class="relative h-8 w-8 shrink-0 overflow-hidden rounded border border-white/15 bg-black/35 shadow-sm"><img :src="`/images/alliances/${alliance.id}/logo?size=64`" alt="" loading="lazy" class="h-full w-full object-contain"><span class="absolute inset-x-0 bottom-0 h-0.5" :style="{ backgroundColor: allianceColor(alliance.id) }" /></span>
                    <span class="min-w-0 flex-1"><span class="block truncate text-xs font-medium text-gray-300 group-hover:text-white">{{ alliance.name }}</span><span class="mt-0.5 block truncate text-[9px] text-gray-600">[{{ alliance.ticker }}]</span></span>
                    <span class="shrink-0 text-right"><strong class="block font-mono text-xs font-medium text-white">{{ alliance.systems }}</strong><span class="block text-[9px] text-gray-600">systems</span></span>
                    <span class="absolute bottom-0 left-0 h-px bg-current opacity-30" :style="{ color: allianceColor(alliance.id), width: `${Math.max(2, alliance.systems / (allianceSummaries[0]?.systems || 1) * 100)}%` }" />
                </NuxtLink>
            </div>
            <div class="flex items-center justify-between border-t border-white/[0.07] px-3 py-2 text-[9px] text-gray-600"><span>{{ formatNumber(data?.sovereignty?.length) }} systems claimed</span><span>Hover to isolate</span></div>
        </aside>
        <aside v-else-if="mode === 'aiid'" class="absolute inset-x-3 bottom-28 z-20 flex h-[44%] min-w-0 flex-col overflow-hidden rounded-lg border border-cyan-300/15 bg-[#0b0c11]/94 shadow-2xl backdrop-blur-md sm:inset-y-3 sm:left-3 sm:right-auto sm:h-auto sm:w-[min(300px,30%)] sm:min-w-[260px]">
            <div class="border-b border-white/[0.07] px-3 py-3">
                <div class="flex items-center justify-between gap-2">
                    <div><div class="text-[10px] font-bold uppercase tracking-[0.15em] text-cyan-300">Am I In Danger?</div><div class="mt-0.5 text-[10px] text-gray-600">Watch everything within 10 jumps</div></div>
                    <span class="flex items-center gap-1.5 text-[9px] font-medium uppercase tracking-wider" :class="liveConnected ? 'text-emerald-400' : 'text-gray-600'"><span class="h-1.5 w-1.5 rounded-full" :class="liveConnected ? 'bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,.8)]' : 'bg-gray-700'" />{{ liveConnected ? 'Armed' : 'Connecting' }}</span>
                </div>
                <SearchPicker class="mt-2" :types="['system']" placeholder="Add a system to watch..." :disabled="(watchedSystemIds?.length ?? 0) >= 8" :is-picked="(type, id) => type === 'system' && (watchedSystemIds ?? []).includes(id)" @select="addWatchedSystem" />
                <div v-if="watchedSystems.length" class="mt-2 flex flex-wrap gap-1">
                    <button v-for="system in watchedSystems" :key="system.id" type="button" class="group flex items-center gap-1 rounded border border-cyan-300/15 bg-cyan-400/[0.07] px-2 py-1 text-[10px] text-cyan-100 hover:bg-cyan-400/[0.12]" :title="`Stop watching ${system.system_name}`" @click="removeWatchedSystem(system.id)">
                        <span class="max-w-32 truncate">{{ system.system_name }}</span><span class="text-cyan-500 group-hover:text-white">×</span>
                    </button>
                </div>
                <div v-if="watchedSystems.length" class="mt-2 flex items-center justify-between text-[9px] text-gray-600"><span>{{ formatNumber(scene.nodes.length) }} systems covered</span><span>{{ data?.regions?.length ?? 0 }} regions</span></div>
            </div>
            <div class="min-h-0 flex-1 overflow-y-auto overscroll-contain py-1.5">
                <div v-if="!watchedSystems.length" class="px-5 py-12 text-center"><div class="text-sm font-medium text-gray-300">Choose your position</div><div class="mt-2 text-[11px] leading-relaxed text-gray-600">Add a mining, ratting, or travel system. AIID will stitch together every stargate route up to ten jumps away.</div></div>
                <div v-else-if="!liveKills.length" class="px-4 py-10 text-center text-xs text-gray-600">No kills found in the watch area during the last 24 hours.</div>
                <NuxtLink v-for="kill in liveKills" :key="kill.killmail_id" :to="`/kill/${kill.killmail_id}`" class="group relative flex items-center gap-2.5 border-b border-white/[0.04] px-2.5 py-2 transition-colors hover:bg-rose-500/[0.07]" :class="hoveredLiveKillId === kill.killmail_id ? 'bg-rose-500/[0.09]' : ''" @mouseenter="focusLiveKill(kill, $event)" @mouseleave="clearLiveKill(kill.killmail_id)">
                    <span class="h-10 w-10 shrink-0 overflow-hidden rounded border border-white/10 bg-black/35"><EveImage v-if="kill.ship_type_id" :src="`/images/types/${kill.ship_type_id}/icon?size=64`" :alt="kill.ship_name || ''" class="h-full w-full object-cover" loading="lazy" /><span v-else class="flex h-full w-full items-center justify-center text-gray-700">?</span></span>
                    <span class="min-w-0 flex-1"><span class="block truncate text-xs font-medium text-gray-200 group-hover:text-white">{{ kill.ship_name || 'Unknown ship' }}</span><span class="mt-0.5 block truncate text-[10px] text-gray-500">{{ kill.victim_character_name || kill.victim_corporation_name || 'Unknown victim' }}</span><span class="mt-0.5 block truncate text-[9px] text-gray-600">{{ kill.solar_system_name || 'Unknown system' }} · {{ kill.region_name || 'New Eden' }}</span></span>
                    <span class="shrink-0 text-right"><strong class="block font-mono text-[11px] font-medium text-amber-300">{{ formatIsk(kill.total_value) }}</strong><span class="mt-1 block text-[9px] tabular-nums text-gray-600">{{ liveKillAge(kill.killmail_time) }}</span></span>
                </NuxtLink>
            </div>
            <div class="flex items-center justify-between border-t border-white/[0.07] px-3 py-2 text-[9px] text-gray-600"><span>{{ liveKills.length }} kills in 24h</span><span>Risk signals fade over 15m</span></div>
        </aside>
        <aside v-else class="absolute inset-x-3 bottom-28 z-20 flex h-[44%] min-w-0 flex-col overflow-hidden rounded-lg border border-white/[0.1] bg-[#0b0c11]/94 shadow-2xl backdrop-blur-md sm:inset-y-3 sm:left-3 sm:right-auto sm:h-auto sm:w-[min(300px,30%)] sm:min-w-[260px]">
            <div class="flex items-center justify-between border-b border-white/[0.07] px-3 py-3">
                <div><div class="text-[10px] font-bold uppercase tracking-[0.15em] text-rose-300">Live kills</div><div class="mt-0.5 text-[10px] text-gray-600">Newest activity across New Eden</div></div>
                <span class="flex items-center gap-1.5 text-[9px] font-medium uppercase tracking-wider" :class="liveConnected ? 'text-emerald-400' : 'text-gray-600'"><span class="h-1.5 w-1.5 rounded-full" :class="liveConnected ? 'bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,.8)]' : 'bg-gray-700'" />{{ liveConnected ? 'Connected' : 'Connecting' }}</span>
            </div>
            <div class="min-h-0 flex-1 overflow-y-auto overscroll-contain py-1.5">
                <div v-if="!liveKills.length" class="px-4 py-10 text-center text-xs text-gray-600">Waiting for killmails…</div>
                <NuxtLink v-for="kill in liveKills" :key="kill.killmail_id" :to="`/kill/${kill.killmail_id}`" class="group relative flex items-center gap-2.5 border-b border-white/[0.04] px-2.5 py-2 transition-colors hover:bg-rose-500/[0.07]" :class="hoveredLiveKillId === kill.killmail_id ? 'bg-rose-500/[0.09]' : ''" @mouseenter="focusLiveKill(kill, $event)" @mouseleave="clearLiveKill(kill.killmail_id)">
                    <span class="h-10 w-10 shrink-0 overflow-hidden rounded border border-white/10 bg-black/35"><EveImage v-if="kill.ship_type_id" :src="`/images/types/${kill.ship_type_id}/icon?size=64`" :alt="kill.ship_name || ''" class="h-full w-full object-cover" loading="lazy" /><span v-else class="flex h-full w-full items-center justify-center text-gray-700">?</span></span>
                    <span class="min-w-0 flex-1"><span class="block truncate text-xs font-medium text-gray-200 group-hover:text-white">{{ kill.ship_name || 'Unknown ship' }}</span><span class="mt-0.5 block truncate text-[10px] text-gray-500">{{ kill.victim_character_name || kill.victim_corporation_name || 'Unknown victim' }}</span><span class="mt-0.5 block truncate text-[9px] text-gray-600">{{ kill.solar_system_name || 'Unknown system' }} · {{ kill.region_name || 'New Eden' }}</span></span>
                    <span class="shrink-0 text-right"><strong class="block font-mono text-[11px] font-medium text-amber-300">{{ formatIsk(kill.total_value) }}</strong><span class="mt-1 block text-[9px] tabular-nums text-gray-600">{{ liveKillAge(kill.killmail_time) }}</span></span>
                </NuxtLink>
            </div>
            <div class="flex items-center justify-between border-t border-white/[0.07] px-3 py-2 text-[9px] text-gray-600"><span>{{ liveKills.length }} recent kills</span><span>Events fade over 45s</span></div>
        </aside>
        <div class="absolute right-3 z-20 items-center gap-2 rounded-lg border border-white/[0.08] bg-black/65 px-3 py-2 text-xs text-gray-500 backdrop-blur-md" :class="mode === 'map' ? 'top-16 flex sm:top-3' : 'hidden sm:top-3 sm:flex'">
            <span v-if="mode === 'live'" class="flex items-center gap-1.5"><span class="h-1.5 w-1.5 rounded-full" :class="liveConnected ? 'bg-rose-400 shadow-[0_0_8px_rgba(251,113,133,.9)]' : 'bg-gray-700'" />{{ liveConnected ? 'Listening live' : 'Connecting to relay' }} · {{ liveKills.length }} recent</span>
            <span v-else-if="mode === 'aiid'" class="flex items-center gap-1.5"><span class="h-1.5 w-1.5 rounded-full" :class="liveConnected ? 'bg-cyan-300 shadow-[0_0_8px_rgba(103,232,249,.9)]' : 'bg-gray-700'" />{{ liveConnected ? 'Danger watch armed' : 'Connecting to relay' }} · {{ formatNumber(scene.nodes.length) }} systems</span>
            <span v-else-if="data && mode === 'sovereignty'">{{ formatNumber(data.sovereignty?.length) }} claimed · {{ allianceSummaries.length }} alliances<span v-if="data.snapshot_at"> · {{ formatDateTime(data.snapshot_at) }}</span></span>
            <span v-else-if="data">{{ formatNumber(data.systems?.length) }} systems · {{ data.regions?.length ?? 0 }} regions</span>
            <span v-if="activityLayer !== 'none'" class="flex items-center gap-1.5 text-emerald-400/80"><span class="h-1.5 w-1.5 rounded-full bg-emerald-400" />Hourly data · {{ activityWindowLabel }}</span>
            <button type="button" class="text-blue-400 hover:text-blue-300" @click="resetView">Reset view</button>
        </div>
        <button v-if="mode !== 'map'" type="button" class="absolute right-3 top-3 z-20 rounded-lg border border-white/10 bg-black/65 px-3 py-2 text-xs text-blue-400 sm:hidden" @click="resetView">Reset view</button>
        <div v-if="pending || !ready" class="absolute inset-0 z-10 flex items-center justify-center text-sm text-gray-500 animate-pulse">{{ pending ? 'Loading map data' : initStage }}...</div>
        <div v-if="error || initError" class="absolute inset-0 z-30 flex items-center justify-center p-6 text-center text-sm text-red-300">{{ initError || 'Unable to load map data' }}</div>

        <div v-if="hoveredAlliance && hoverCardVisible" :style="hoverCardStyle" class="pointer-events-none absolute z-20 w-[280px] rounded-lg border border-white/[0.12] bg-[#101116]/95 p-3 text-xs shadow-2xl backdrop-blur-md">
            <div class="flex items-start gap-3">
                <img :src="`/images/alliances/${hoveredAlliance.id}/logo?size=64`" :alt="hoveredAlliance.name" class="h-12 w-12 shrink-0 rounded bg-black/30 object-contain">
                <div class="min-w-0 flex-1"><div class="truncate text-sm font-semibold text-white">{{ hoveredAlliance.name }}</div><div class="mt-0.5 text-[10px] text-gray-500">[{{ hoveredAlliance.ticker }}] · Click to open alliance</div><div class="mt-2 flex gap-3 text-gray-400"><span><strong class="font-mono text-white">{{ hoveredAlliance.systems }}</strong> systems</span><span><strong class="font-mono text-white">{{ hoveredAlliance.regions }}</strong> regions</span></div></div>
            </div>
            <div class="mt-2 flex items-center justify-between border-t border-white/[0.06] pt-2 text-[10px] text-gray-500"><span v-if="hoveredAlliance.memberCount > 0">{{ formatNumber(hoveredAlliance.memberCount) }} members</span><span v-else>Current sovereignty</span><span v-if="hoveredAlliance.recentChanges" class="text-amber-300">{{ hoveredAlliance.recentChanges }} changed in 7d</span><span v-else>No recent changes</span></div>
        </div>
        <div v-else-if="hoveredSummary" :style="hoverCardStyle" class="pointer-events-none absolute z-20 w-[280px] rounded-lg border border-white/[0.12] bg-[#101116]/95 p-3 text-xs shadow-2xl backdrop-blur-md">
            <div class="flex items-start justify-between gap-3"><div><div class="text-sm font-semibold text-white">{{ hoveredSummary.name }}</div><div class="mt-0.5 text-[10px] text-gray-500">Click to open the SVG region map</div></div><div class="text-right text-gray-400"><div>{{ hoveredSummary.systems }} systems</div><div class="mt-0.5 text-[9px] text-gray-600">last {{ activityWindowLabel }}</div></div></div>
            <div class="mt-2 grid grid-cols-4 gap-1 text-center"><div class="rounded bg-white/[0.04] p-1"><div class="font-mono text-white">{{ formatNumber(hoveredSummary.ship_kills) }}</div><div class="text-[9px] text-gray-500">ship kills</div></div><div class="rounded bg-white/[0.04] p-1"><div class="font-mono text-white">{{ formatNumber(hoveredSummary.pod_kills) }}</div><div class="text-[9px] text-gray-500">pod kills</div></div><div class="rounded bg-white/[0.04] p-1"><div class="font-mono text-white">{{ formatNumber(hoveredSummary.npc_kills) }}</div><div class="text-[9px] text-gray-500">NPC kills</div></div><div class="rounded bg-white/[0.04] p-1"><div class="font-mono text-white">{{ formatNumber(hoveredSummary.ship_jumps) }}</div><div class="text-[9px] text-gray-500">jumps</div></div></div>
            <div class="mt-2 flex items-center justify-between text-[10px] text-gray-500"><span><span class="text-cyan-400">{{ hoveredSummary.high }}</span> high · <span class="text-amber-400">{{ hoveredSummary.low }}</span> low · <span class="text-red-500">{{ hoveredSummary.null }}</span> null</span><span v-if="hoveredSummary.busiest">Hotspot: <span class="text-gray-300">{{ hoveredSummary.busiest.system_name }}</span></span></div>
        </div>
        <div class="pointer-events-auto absolute bottom-3 z-20 flex max-h-24 max-w-[calc(100%-1.5rem)] flex-wrap items-center gap-x-3 gap-y-1 overflow-y-auto rounded-lg border border-white/[0.08] bg-black/65 px-3 py-2 text-[10px] text-gray-400 shadow-lg backdrop-blur-md sm:pointer-events-none sm:max-h-none" :class="mode === 'sovereignty' || mode === 'live' || mode === 'aiid' ? 'left-3 sm:left-[324px]' : 'left-3'">
            <span class="flex items-center gap-1.5">
                <span v-if="baseLayer === 'live'" class="h-2.5 w-2.5 rounded-full bg-rose-400 shadow-[0_0_9px_rgba(251,113,133,.9)]" />
                <span v-else-if="baseLayer === 'aiid'" class="h-2.5 w-2.5 rounded-full bg-cyan-300 shadow-[0_0_9px_rgba(103,232,249,.9)]" />
                <span v-else-if="baseLayer === 'sovereignty'" class="h-2.5 w-16 rounded-full" style="background: linear-gradient(90deg, #2563eb, #7c3aed, #dc2626, #ca8a04, #16a34a)" />
                <span v-else-if="baseLayer === 'geography'" class="h-2.5 w-4 rounded-sm bg-gradient-to-r from-blue-500/50 via-purple-500/50 to-amber-500/50" />
                <span v-else class="h-2 w-16 rounded-full" style="background: linear-gradient(90deg, #22d3ee, #a3e635, #fbbf24, #b91c1c)" />
                {{ baseLayer === 'live' ? 'Live killmail pulse' : baseLayer === 'aiid' ? 'Watched system / danger pulse' : baseLayer === 'sovereignty' ? 'Alliance territory' : baseLayer === 'geography' ? 'Region grouping' : 'System security' }}
            </span>
            <span v-if="mode === 'sovereignty' && showChanges" class="flex items-center gap-1.5"><span class="h-2.5 w-2.5 rounded-full border border-yellow-400" /> Changed in the last 7 days</span>
            <template v-if="activityLayer !== 'none'">
                <span class="h-2 w-24 rounded-full" style="background: linear-gradient(90deg, #475569, #3b82f6, #a855f7, #f97316, #ef4444)" />
                <span class="font-semibold text-gray-300">{{ activeLayerLabel }}</span>
                <span>Quiet</span><span>Peak {{ maximumActivityLabel }}</span>
                <span class="text-gray-600">last {{ activityWindowLabel }} · log scale</span>
            </template>
        </div>
        <div class="pointer-events-none absolute bottom-3 right-3 z-20 hidden rounded bg-black/45 px-2 py-1 text-[10px] text-gray-600 backdrop-blur-sm sm:block">{{ mode === 'live' ? 'Watching New Eden · scroll to zoom · drag to pan' : mode === 'aiid' ? 'Ten-jump watch area · scroll to zoom · drag to pan' : `Scroll to zoom · drag to pan · click ${mode === 'sovereignty' ? 'an alliance' : 'a region'}` }}</div>
    </div>
</template>
