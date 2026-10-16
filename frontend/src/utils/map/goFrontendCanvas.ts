// The Go frontend's shared map renderer. Geometry, colors, labels, and battle
// markers follow PixiScopeView.vue; Canvas2D keeps the renderer independent of
// Nuxt and available when WebGL cannot initialize.
import { Delaunay } from 'd3-delaunay'
import { secColorStr } from './colors'
import { heatColor, mapActivityLayerLabel, mapActivityRatio, mapActivityValue, sovereigntyAllianceColors } from './layers'
import type { MapActivity, MapActivityLayer, MapRenderBaseLayer } from './layers'
import type { BattleMapSystem } from './battles'
import { battleRegionTotals } from './battles'
import { formatIsk } from '../formatters'
import { createSovereigntyTerritories, sovereigntyOwnerAt } from './territories'
import type { SovereigntyTerritoryMap } from './territories'
import {createMapMarkerPicker,type MapMarkerChoice} from './markerPicker'

type ScopeSystem = {
    solar_system_id: number
    system_name: string
    region_id: number
    security: number
    x: number
    z: number
    distance?: number
    is_anchor?: boolean
}
type ScopeJump = { from_solar_system_id: number; to_solar_system_id: number }
type ScopeRegion = { region_id: number; name: string }
type ScopeActivity = { system_id: number } & MapActivity
type SovereigntyClaim = { system_id: number; alliance_id: number; alliance_name: string; alliance_ticker: string; member_count?: number }
export type SovereigntyAllianceSummary = { id: number; name: string; ticker: string; memberCount: number; systems: number; regions: number; recentChanges: number }
export type MapRegionSummary = {
    id: number; name: string; systems: number; high: number; low: number; null: number
    ship_kills: number; pod_kills: number; npc_kills: number; ship_jumps: number
    busiest?: { id: number; system_name: string }
}
export type MapActivityLegend = { label: string; maximum: number; peak: string }
export type MapScopeData = {
    systems: ScopeSystem[]
    jumps?: ScopeJump[]
    regions?: ScopeRegion[]
    activity?: ScopeActivity[]
    sovereignty?: SovereigntyClaim[]
    changes?: Array<{ system_id: number }>
}

type Node = {
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
type Cell = { points: number[]; region_id: number; security: number }
type Link = { from: Node; to: Node }
type Region = MapRegionSummary & { x: number; y: number; busiestValue: number }
let mapSequence = 0
type Scene = {
    nodes: Node[]
    links: Link[]
    cells: Cell[]
    regions: Region[]
    byID: Map<number, Node>
    bounds: { x: number; y: number; width: number; height: number }
    delaunay: Delaunay<Node> | null
}

export type GoMapOptions = {
    baseLayer?: MapRenderBaseLayer
    activityLayer?: MapActivityLayer
    showConnections?: boolean
    showSystems?: boolean
    showLabels?: boolean
    showChanges?: boolean
    battleSystems?: BattleMapSystem[]
    markerNoun?: string
    markerValueLabel?: string
    showScopeDetails?: boolean
    regionId?: number | null
    onBattleRegion?: (regionId: number) => void
    onBattleSystem?: (system: BattleMapSystem) => void
    onAlliance?: (allianceId: number) => void
    onHoverAlliance?: (alliance: SovereigntyAllianceSummary | null, point: { x: number; y: number }) => void
    onHoverRegion?: (region: MapRegionSummary | null, point: { x: number; y: number }) => void
    panel?: HTMLElement | null
}

export type GoMapController = {
    setData(data: MapScopeData): void
    setRegion(regionId: number | null): void
    setBattleSystems(systems: BattleMapSystem[]): void
    setLayers(options: Pick<GoMapOptions, 'baseLayer' | 'activityLayer' | 'showConnections' | 'showSystems' | 'showLabels' | 'showChanges'>): void
    pulseKill(systemId: number, value: number, ageMs?: number): void
    focusLiveSystem(systemId: number | null, origin?: { x: number; y: number }): void
    allianceSummaries(): SovereigntyAllianceSummary[]
    allianceColor(allianceId: number): string
    focusAlliance(allianceId: number | null): void
    activityLegend(): MapActivityLegend
    resetView(): void
    destroy(): void
}

function regionColor(regionId: number): string {
    return `hsl(${((regionId % 1000) * 137.508) % 360}, 60%, 55%)`
}

function emptyScene(): Scene {
    return { nodes: [], links: [], cells: [], regions: [], byID: new Map(),
        bounds: { x: 0, y: 0, width: 1600, height: 1200 }, delaunay: null }
}

function buildScene(data: MapScopeData, regionId: number | null): Scene {
    const systems = (data.systems || []).filter(system => !regionId || system.region_id === regionId)
    if (!systems.length) return emptyScene()
    const names = new Map((data.regions || []).map(region => [region.region_id, region.name || '']))
    let minX = Infinity, maxX = -Infinity, minZ = Infinity, maxZ = -Infinity
    for (const system of systems) {
        minX = Math.min(minX, system.x); maxX = Math.max(maxX, system.x)
        minZ = Math.min(minZ, system.z); maxZ = Math.max(maxZ, system.z)
    }
    const centerX = (minX + maxX) / 2, centerZ = (minZ + maxZ) / 2
    const scale = Math.min(1600 / ((maxX - minX) || 1), 1200 / ((maxZ - minZ) || 1))
    const nodes: Node[] = systems.map(system => ({
        id: system.solar_system_id,
        x: (system.x - centerX) * scale,
        y: -(system.z - centerZ) * scale,
        region_id: system.region_id,
        region_name: names.get(system.region_id) || '',
        system_name: system.system_name,
        security: system.security,
        distance: system.distance,
        is_anchor: system.is_anchor,
    }))
    const byID = new Map(nodes.map(node => [node.id, node]))
    const links = (data.jumps || []).flatMap(jump => {
        const from = byID.get(jump.from_solar_system_id), to = byID.get(jump.to_solar_system_id)
        return from && to ? [{ from, to }] : []
    })
    let laidMinX = Infinity, laidMaxX = -Infinity, laidMinY = Infinity, laidMaxY = -Infinity
    for (const node of nodes) {
        laidMinX = Math.min(laidMinX, node.x); laidMaxX = Math.max(laidMaxX, node.x)
        laidMinY = Math.min(laidMinY, node.y); laidMaxY = Math.max(laidMaxY, node.y)
    }
    const pad = 80
    const bounds = { x: laidMinX - pad, y: laidMinY - pad,
        width: laidMaxX - laidMinX + pad * 2, height: laidMaxY - laidMinY + pad * 2 }
    const delaunay = Delaunay.from(nodes, node => node.x, node => node.y)
    const voronoi = delaunay.voronoi([bounds.x, bounds.y, bounds.x + bounds.width, bounds.y + bounds.height])
    const cells: Cell[] = []
    for (let index = 0; index < nodes.length; index++) {
        const polygon = voronoi.cellPolygon(index)
        if (polygon) cells.push({ points: polygon.flatMap(point => [point[0], point[1]]), region_id: nodes[index]!.region_id, security: nodes[index]!.security })
    }
    const regionMap = new Map<number, Region>()
    const activityByID = new Map((data.activity || []).map(row => [row.system_id, row]))
    for (const node of nodes) {
        const region: Region = regionMap.get(node.region_id) || { id: node.region_id, name: node.region_name, x: 0, y: 0, systems: 0,
            high: 0, low: 0, null: 0, ship_kills: 0, pod_kills: 0, npc_kills: 0, ship_jumps: 0, busiestValue: -1 }
        region.x += node.x; region.y += node.y; region.systems++
        if (node.security >= .45) region.high++
        else if (node.security > 0) region.low++
        else region.null++
        const activity = activityByID.get(node.id)
        region.ship_kills += activity?.ship_kills || 0; region.pod_kills += activity?.pod_kills || 0
        region.npc_kills += activity?.npc_kills || 0; region.ship_jumps += activity?.ship_jumps || 0
        const kills = (activity?.ship_kills || 0) + (activity?.pod_kills || 0)
        if (kills > region.busiestValue) { region.busiestValue = kills; region.busiest = node }
        regionMap.set(node.region_id, region)
    }
    const regions = [...regionMap.values()].map(region => ({ ...region, x: region.x / region.systems, y: region.y / region.systems }))
    return { nodes, links, cells, regions, byID, bounds, delaunay }
}

export function createGoMap(host: HTMLElement, initial: GoMapOptions = {}): GoMapController {
    let options: GoMapOptions = { baseLayer: 'geography', activityLayer: 'none', showConnections: true, showSystems: true, showLabels: true, ...initial }
    let data: MapScopeData = { systems: [] }
    let scene = emptyScene()
    let activityValues: number[] | null = null
    let activityMaximum = 0
    let battleSystems = options.battleSystems || []
    let regionId = options.regionId || null
    let hoverRegionId: number | null = null
    let hoverAllianceId: number | null = null
    let sovereigntyTerritories: SovereigntyTerritoryMap | null = null
    let sovereigntyBySystem = new Map<number, SovereigntyClaim>()
    let allianceColors = new Map<number, string>()
    let sovereigntySummaries: SovereigntyAllianceSummary[] = []
    let allianceLabels: Array<{ id: number; label: string; x: number; y: number; fontSize: number }> = []
    let changeSystemIDs = new Set<number>()
    let camera = { x: 0, y: 0, scale: 1 }
    let minimumScale = .25
    let drag: { x: number; y: number; moved: boolean } | null = null
    let frame = 0
    let pulseFrame = 0
    let destroyed = false
    const livePulses = new Map<number, { startedAt: number; value: number }>()
    let focusedLiveSystem: { id: number; origin: { x: number; y: number } } | null = null
    let markerNodes: Array<MapMarkerChoice & { x: number; y: number }> = []

    host.replaceChildren()
    host.classList.add('go-map-host')
    const canvas = document.createElement('canvas')
    canvas.className = 'go-map-canvas'
    canvas.setAttribute('aria-label', 'EVE Online star map')
    canvas.setAttribute('role', 'application')
    canvas.tabIndex = 0
    const keyboardHelp = document.createElement('span')
    keyboardHelp.id = `go-map-keyboard-help-${++mapSequence}`
    keyboardHelp.className = 'go-map-keyboard-help'
    keyboardHelp.textContent = 'Use arrow keys to pan, plus or minus to zoom, 0 to reset, and Enter to inspect the center. Tab leaves the map. Use map search to find a system or region by name.'
    canvas.setAttribute('aria-describedby', keyboardHelp.id)
    const context = canvas.getContext('2d', { alpha: false })
    if (!context) throw new Error('Canvas2D is unavailable')
    const pulseCanvas = document.createElement('canvas')
    pulseCanvas.className = 'go-map-pulses'
    pulseCanvas.setAttribute('aria-hidden', 'true')
    const pulseContext = pulseCanvas.getContext('2d')
    if (!pulseContext) throw new Error('Canvas2D is unavailable')
    const markerLayer = document.createElement('div')
    markerLayer.className = 'go-map-markers'
    const reset = document.createElement('button')
    reset.className = 'go-map-reset'
    reset.type = 'button'
    reset.textContent = 'Reset view'
    const controls = document.createElement('div')
    controls.className = 'go-map-controls'
    controls.setAttribute('role', 'group')
    controls.setAttribute('aria-label', 'Map zoom')
    const zoomIn = document.createElement('button'), zoomOut = document.createElement('button')
    for (const [button, label, path] of [[zoomIn, 'Zoom in', 'M12 5v14M5 12h14'], [zoomOut, 'Zoom out', 'M5 12h14']] as const) {
        button.type = 'button'
        button.setAttribute('aria-label', label)
        button.title = label
        button.innerHTML = `<svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="${path}"/></svg>`
        controls.append(button)
    }
    const hint = document.createElement('span')
    hint.className = 'go-map-hint'
    hint.textContent = initial.battleSystems
        ? 'Scroll to zoom · drag to pan · select crossed swords'
        : initial.baseLayer === 'sovereignty' ? 'Scroll to zoom · drag to pan · click an alliance'
        : initial.baseLayer === 'live' ? 'Watching New Eden · scroll to zoom · drag to pan'
        : initial.baseLayer === 'aiid' ? 'Ten-jump watch area · scroll to zoom · drag to pan'
        : 'Scroll to zoom · drag to pan · click a region'
    host.append(canvas, pulseCanvas, markerLayer, reset, controls, hint, keyboardHelp)
    const markerPicker=createMapMarkerPicker(host)
    const overview = initial.showScopeDetails ? document.createElement('div') : null
    const overviewCount = document.createElement('span')
    const legend = initial.showScopeDetails ? document.createElement('div') : null
    const legendChip = document.createElement('i')
    const legendLabel = document.createElement('span')
    if (overview && legend) {
        overview.className = 'go-map-overview'
        overview.hidden = true
        overview.append(overviewCount, reset)
        legend.className = 'go-map-legend'
        legend.append(legendChip, legendLabel)
        host.append(overview, legend)
    }
    const refreshScopeDetails = () => {
        if (!overview || !legend) return
        overview.hidden = scene.nodes.length === 0
        overviewCount.textContent = `${scene.nodes.length.toLocaleString('en-US')} systems · ${scene.regions.length} regions`
        legend.dataset.base = options.baseLayer || 'geography'
        legendLabel.textContent = options.baseLayer === 'security' ? 'System security' : 'Region grouping'
    }

    const buildActivity = () => {
        const rows = data.activity || []
        const byID = new Map(rows.map(row => [row.system_id, row]))
        const maximums = {
            ship_kills: Math.max(0, ...rows.map(row => row.ship_kills || 0)),
            pod_kills: Math.max(0, ...rows.map(row => row.pod_kills || 0)),
            npc_kills: Math.max(0, ...rows.map(row => row.npc_kills || 0)),
            ship_jumps: Math.max(0, ...rows.map(row => row.ship_jumps || 0)),
        }
        const layer = options.activityLayer || 'none'
        activityValues = layer === 'none' ? null : scene.nodes.map(node => mapActivityValue(layer, byID.get(node.id), maximums))
        activityMaximum = activityValues ? Math.max(0, ...activityValues) : 0
    }
    const buildSovereignty = () => {
        sovereigntyBySystem = new Map((data.sovereignty || []).map(claim => [claim.system_id, claim]))
        changeSystemIDs = new Set((data.changes || []).map(change => change.system_id))
        sovereigntyTerritories = null
        allianceColors = new Map()
        sovereigntySummaries = []
        allianceLabels = []
        if (options.baseLayer !== 'sovereignty' || !sovereigntyBySystem.size) return
        const grouped = new Map<number, { claim: SovereigntyClaim; nodes: Node[] }>()
        for (const node of scene.nodes) {
            const claim = sovereigntyBySystem.get(node.id)
            if (!claim) continue
            const group = grouped.get(claim.alliance_id) || { claim, nodes: [] }
            group.nodes.push(node)
            grouped.set(claim.alliance_id, group)
        }
        const adjacency = new Map<number, number[]>()
        for (const link of scene.links) {
            const fromClaim = sovereigntyBySystem.get(link.from.id), toClaim = sovereigntyBySystem.get(link.to.id)
            if (!fromClaim || fromClaim.alliance_id !== toClaim?.alliance_id || Math.hypot(link.from.x - link.to.x, link.from.y - link.to.y) > 45) continue
            adjacency.set(link.from.id, [...(adjacency.get(link.from.id) || []), link.to.id])
            adjacency.set(link.to.id, [...(adjacency.get(link.to.id) || []), link.from.id])
        }
        const summaries = [...grouped].map(([id, group]) => {
            const byID = new Map(group.nodes.map(node => [node.id, node]))
            const seen = new Set<number>()
            let largest: Node[] = []
            for (const node of group.nodes) {
                if (seen.has(node.id)) continue
                const component: Node[] = [], queue = [node.id]
                seen.add(node.id)
                while (queue.length) {
                    const current = queue.pop()!, found = byID.get(current)
                    if (found) component.push(found)
                    for (const neighbor of adjacency.get(current) || []) {
                        if (!seen.has(neighbor)) { seen.add(neighbor); queue.push(neighbor) }
                    }
                }
                if (component.length > largest.length) largest = component
            }
            const labelNodes = largest.length ? largest : group.nodes
            return { id, claim: group.claim, systems: group.nodes.length, componentSystems: labelNodes.length,
                name: group.claim.alliance_name, ticker: group.claim.alliance_ticker, memberCount: group.claim.member_count || 0,
                regions: new Set(group.nodes.map(node => node.region_id)).size,
                recentChanges: group.nodes.filter(node => changeSystemIDs.has(node.id)).length,
                x: labelNodes.reduce((sum, node) => sum + node.x, 0) / labelNodes.length,
                y: labelNodes.reduce((sum, node) => sum + node.y, 0) / labelNodes.length }
        }).sort((left, right) => right.systems - left.systems || left.id - right.id)
        sovereigntySummaries = summaries.map(({ id, name, ticker, memberCount, systems, regions, recentChanges }) =>
            ({ id, name, ticker, memberCount, systems, regions, recentChanges }))
        allianceColors = sovereigntyAllianceColors(summaries.map(item => item.id))
        const placed: Array<{ left: number; right: number; top: number; bottom: number }> = []
        for (const summary of summaries) {
            if (summary.componentSystems < 3) continue
            const fontSize = Math.min(25, Math.max(10, 8 + Math.sqrt(summary.componentSystems) * 1.45))
            const label = summary.componentSystems >= 12 ? summary.claim.alliance_name : summary.claim.alliance_ticker
            const width = Math.max(30, label.length * fontSize * .56), height = fontSize * 1.35
            const box = { left: summary.x - width / 2, right: summary.x + width / 2,
                top: summary.y - height / 2, bottom: summary.y + height / 2 }
            if (placed.some(other => !(box.right + 5 < other.left || box.left - 5 > other.right || box.bottom + 4 < other.top || box.top - 4 > other.bottom))) continue
            placed.push(box)
            allianceLabels.push({ id: summary.id, label, x: summary.x, y: summary.y, fontSize })
        }
        sovereigntyTerritories = createSovereigntyTerritories(
            scene.nodes, scene.links.map(link => ({ from: link.from.id, to: link.to.id })), scene.bounds,
            new Map([...sovereigntyBySystem].map(([systemID, claim]) => [systemID, claim.alliance_id])),
        )
    }
    const toScreen = (x: number, y: number) => ({ x: camera.x + x * camera.scale, y: camera.y + y * camera.scale })
    const toWorld = (x: number, y: number) => ({ x: (x - camera.x) / camera.scale, y: (y - camera.y) / camera.scale })
    const pointer = (event: PointerEvent | WheelEvent) => {
        const rect = canvas.getBoundingClientRect()
        return { x: event.clientX - rect.left, y: event.clientY - rect.top }
    }
    const updateMarkerPositions = () => {
        for (const marker of markerNodes) {
            const point = toScreen(marker.x, marker.y)
            marker.node.style.left = `${point.x}px`
            marker.node.style.top = `${point.y}px`
        }
    }
    const requestDraw = () => {
        if (frame || destroyed) return
        frame = requestAnimationFrame(() => { frame = 0; draw() })
    }
    // The star field can contain thousands of systems. Animate live signals on
    // their own transparent canvas so each pulse does not redraw the full map.
    const drawPulses = () => {
        pulseFrame = 0
        if (destroyed) return
        const width = host.clientWidth, height = host.clientHeight
        if (!width || !height) return
        const ratio = Math.min(devicePixelRatio || 1, 2)
        if (pulseCanvas.width !== Math.round(width * ratio) || pulseCanvas.height !== Math.round(height * ratio)) {
            pulseCanvas.width = Math.round(width * ratio)
            pulseCanvas.height = Math.round(height * ratio)
        }
        pulseContext.setTransform(ratio, 0, 0, ratio, 0, 0)
        pulseContext.clearRect(0, 0, width, height)
        if (options.baseLayer !== 'live' && options.baseLayer !== 'aiid') return
        const now = performance.now()
        const lifetime = options.baseLayer === 'aiid' ? 15 * 60 : 45
        pulseContext.translate(camera.x, camera.y)
        pulseContext.scale(camera.scale, camera.scale)
        for (const [id, pulse] of livePulses) {
            const node = scene.byID.get(id)
            const age = (now - pulse.startedAt) / 1000
            if (!node || age > lifetime) { livePulses.delete(id); continue }
            const life = 1 - age / lifetime
            const ringLife = Math.max(0, 1 - age / (options.baseLayer === 'aiid' ? 30 : 18))
            const valueScale = Math.min(1, Math.max(0, Math.log10(Math.max(pulse.value, 1) / 1e6) / 5))
            const color = valueScale > .72 ? '#facc15' : valueScale > .38 ? '#fb923c' : '#fb7185'
            const radius = 3.5 + valueScale * 4 + Math.sin(age * 5) * .8
            pulseContext.fillStyle = color
            pulseContext.globalAlpha = .42 * life
            pulseContext.beginPath(); pulseContext.arc(node.x, node.y, radius * 2.4, 0, Math.PI * 2); pulseContext.fill()
            pulseContext.globalAlpha = .95 * life
            pulseContext.beginPath(); pulseContext.arc(node.x, node.y, Math.max(1.5, radius * life), 0, Math.PI * 2); pulseContext.fill()
            if (ringLife > 0) {
                pulseContext.globalAlpha = .85 * ringLife
                pulseContext.strokeStyle = color
                pulseContext.lineWidth = 1.5
                pulseContext.beginPath(); pulseContext.arc(node.x, node.y, radius + age * 2.2, 0, Math.PI * 2); pulseContext.stroke()
            }
            if (age < 5) {
                pulseContext.globalAlpha = .9 - age * .12
                pulseContext.fillStyle = '#fff'
                pulseContext.beginPath(); pulseContext.arc(node.x, node.y, radius * .35, 0, Math.PI * 2); pulseContext.fill()
            }
        }
        pulseContext.setTransform(ratio, 0, 0, ratio, 0, 0)
        const focused = focusedLiveSystem && scene.byID.get(focusedLiveSystem.id)
        if (focused && focusedLiveSystem) {
            const target = toScreen(focused.x, focused.y), origin = focusedLiveSystem.origin
            const bend = Math.max(40, (target.x - origin.x) * .38)
            pulseContext.globalAlpha = .72
            pulseContext.strokeStyle = '#fb7185'; pulseContext.lineWidth = 1.4
            pulseContext.beginPath(); pulseContext.moveTo(origin.x, origin.y)
            pulseContext.bezierCurveTo(origin.x + bend, origin.y, target.x - bend * .35, target.y, target.x, target.y); pulseContext.stroke()
            pulseContext.globalAlpha = .22; pulseContext.fillStyle = '#fb7185'
            pulseContext.beginPath(); pulseContext.arc(target.x, target.y, 9 + Math.sin(now / 130) * 2, 0, Math.PI * 2); pulseContext.fill()
            pulseContext.globalAlpha = .9; pulseContext.strokeStyle = '#fff'
            pulseContext.beginPath(); pulseContext.arc(target.x, target.y, 9 + Math.sin(now / 130) * 2, 0, Math.PI * 2); pulseContext.stroke()
            pulseContext.globalAlpha = 1; pulseContext.fillStyle = '#fff'
            pulseContext.beginPath(); pulseContext.arc(target.x, target.y, 3.5, 0, Math.PI * 2); pulseContext.fill()
        }
        pulseContext.globalAlpha = 1
        if ((livePulses.size || focused) && !document.hidden) requestPulseDraw()
    }
    const requestPulseDraw = () => {
        if (pulseFrame || destroyed || document.hidden) return
        pulseFrame = requestAnimationFrame(drawPulses)
    }
    const drawPolygon = (points: number[]) => {
        if (points.length < 6) return
        context.beginPath()
        context.moveTo(points[0]!, points[1]!)
        for (let i = 2; i < points.length; i += 2) context.lineTo(points[i]!, points[i + 1]!)
        context.closePath()
        context.fill()
    }
    const drawTerritories = (onlyAlliance: number | null, alpha: number) => {
        if (!sovereigntyTerritories) return
        for (const territory of sovereigntyTerritories.territories) {
            if (onlyAlliance != null && territory.allianceId !== onlyAlliance) continue
            const color = allianceColors.get(territory.allianceId) || '#64748b'
            for (const polygon of territory.polygons) {
                context.beginPath()
                for (const ring of polygon.rings) {
                    if (ring.length < 8) continue
                    context.moveTo(ring[0]!, ring[1]!)
                    for (let index = 2; index < ring.length; index += 2) context.lineTo(ring[index]!, ring[index + 1]!)
                    context.closePath()
                }
                context.fillStyle = color
                context.globalAlpha = alpha
                context.shadowColor = color
                context.shadowBlur = 8
                context.fill('evenodd')
                context.shadowBlur = 0
                context.globalAlpha = onlyAlliance == null ? .9 * alpha : .9
                context.strokeStyle = onlyAlliance == null ? '#05060a' : '#f8fafc'
                context.lineWidth = onlyAlliance == null ? 2.2 : 3.2
                context.stroke()
            }
        }
    }
    const draw = () => {
        if (destroyed) return
        const width = host.clientWidth, height = host.clientHeight
        if (!width || !height) return
        const ratio = Math.min(devicePixelRatio || 1, 2)
        if (canvas.width !== Math.round(width * ratio) || canvas.height !== Math.round(height * ratio)) {
            canvas.width = Math.round(width * ratio)
            canvas.height = Math.round(height * ratio)
        }
        context.setTransform(ratio, 0, 0, ratio, 0, 0)
        context.globalAlpha = 1
        context.fillStyle = '#08090d'
        context.fillRect(0, 0, width, height)
        context.translate(camera.x, camera.y)
        context.scale(camera.scale, camera.scale)
        const hovered = hoverRegionId != null || hoverAllianceId != null
        const baseAlpha = hovered ? options.baseLayer === 'sovereignty' ? .16 : .22 : 1
        if (options.baseLayer === 'sovereignty') drawTerritories(null, .52 * baseAlpha)
        if (options.baseLayer !== 'live' && options.baseLayer !== 'aiid' && options.baseLayer !== 'sovereignty') {
            context.globalAlpha = (options.baseLayer === 'geography' ? .1 : .065) * baseAlpha
            for (const cell of scene.cells) {
                context.fillStyle = options.baseLayer === 'geography' ? regionColor(cell.region_id) : secColorStr(cell.security)
                drawPolygon(cell.points)
            }
        }
        if (options.showConnections) {
            context.beginPath()
            for (const link of scene.links) {
                context.moveTo(link.from.x, link.from.y)
                context.lineTo(link.to.x, link.to.y)
            }
            context.globalAlpha = (options.baseLayer === 'live' ? .1 : options.baseLayer === 'aiid' ? .34 : options.baseLayer === 'geography' ? .28 : .2) * baseAlpha
            context.strokeStyle = '#64748b'
            context.lineWidth = options.baseLayer === 'live' ? .4 : options.baseLayer === 'aiid' ? .8 : .55
            context.stroke()
        }
        const layer = options.activityLayer || 'none'
        const values = activityValues, maximum = activityMaximum
        if (options.showSystems) {
            scene.nodes.forEach((node, index) => {
                const ratio = values ? mapActivityRatio(layer, values[index]!, maximum) : 0
                const radius = values ? 1.4 + ratio * 4.5
                    : options.baseLayer === 'live' ? 1.15
                    : options.baseLayer === 'aiid' ? node.is_anchor ? 5.2 : Math.max(1.8, 3.4 - (node.distance ?? 10) * .16) : 1.7
                context.beginPath()
                context.arc(node.x, node.y, radius, 0, Math.PI * 2)
                context.globalAlpha = (values ? .95 : .82) * baseAlpha
                const claim = sovereigntyBySystem.get(node.id)
                context.fillStyle = values ? heatColor(ratio)
                    : options.baseLayer === 'sovereignty' ? claim ? allianceColors.get(claim.alliance_id) || '#64748b' : '#475569'
                    : options.baseLayer === 'live' ? '#475569'
                    : options.baseLayer === 'aiid' && node.is_anchor ? '#67e8f9' : secColorStr(node.security)
                context.fill()
                if (values) {
                    context.globalAlpha = .8 * baseAlpha
                    context.strokeStyle = secColorStr(node.security)
                    context.lineWidth = .45
                    context.beginPath(); context.arc(node.x, node.y, radius + .35, 0, Math.PI * 2); context.stroke()
                }
                if (options.baseLayer === 'aiid' && node.is_anchor) {
                    context.globalAlpha = .85
                    context.strokeStyle = '#67e8f9'
                    context.lineWidth = 1.4
                    context.beginPath(); context.arc(node.x, node.y, radius + 3.5, 0, Math.PI * 2); context.stroke()
                }
            })
        }
        if (options.baseLayer === 'sovereignty' && options.showChanges) {
            context.globalAlpha = .9
            context.strokeStyle = '#facc15'
            context.lineWidth = 1.1
            for (const systemID of changeSystemIDs) {
                const node = scene.byID.get(systemID)
                if (!node) continue
                context.beginPath()
                context.arc(node.x, node.y, 5.2, 0, Math.PI * 2)
                context.stroke()
            }
        }
        if (options.showLabels) {
            context.textAlign = 'center'
            context.textBaseline = 'middle'
            if (options.baseLayer === 'sovereignty') {
                for (const label of allianceLabels) {
                    context.globalAlpha = baseAlpha
                    context.font = `800 ${label.fontSize}px sans-serif`
                    context.lineWidth = 4
                    context.strokeStyle = '#030407'
                    context.fillStyle = allianceColors.get(label.id) || '#64748b'
                    context.strokeText(label.label, label.x, label.y)
                    context.fillText(label.label, label.x, label.y)
                }
            } else if (options.baseLayer === 'aiid') {
                context.textAlign = 'center'
                context.textBaseline = 'bottom'
                context.lineWidth = 4
                context.strokeStyle = '#020617'
                for (const node of scene.nodes) {
                    if (!node.is_anchor && (node.distance ?? 99) > 2) continue
                    const anchor = node.is_anchor
                    context.globalAlpha = anchor ? 1 : .7
                    context.font = `${anchor ? '800 15' : '600 10'}px sans-serif`
                    context.fillStyle = anchor ? '#a5f3fc' : '#e2e8f0'
                    context.strokeText(node.system_name, node.x, node.y - (anchor ? 8 : 4))
                    context.fillText(node.system_name, node.x, node.y - (anchor ? 8 : 4))
                }
            } else {
                context.globalAlpha = .68 * baseAlpha
                context.font = '700 13px sans-serif'
                context.letterSpacing = '1.5px'
                context.lineWidth = 3
                context.strokeStyle = '#000'
                context.fillStyle = '#fff'
                for (const region of scene.regions) {
                    context.strokeText(region.name.toUpperCase(), region.x, region.y)
                    context.fillText(region.name.toUpperCase(), region.x, region.y)
                }
                context.letterSpacing = '0px'
            }
        }
        if (hoverAllianceId != null && options.baseLayer === 'sovereignty') drawTerritories(hoverAllianceId, .7)
        if (hoverRegionId != null) {
            context.globalAlpha = options.baseLayer === 'geography' ? .2 : .08
            for (const cell of scene.cells) {
                if (cell.region_id !== hoverRegionId) continue
                context.fillStyle = options.baseLayer === 'geography' ? regionColor(cell.region_id) : '#60a5fa'
                drawPolygon(cell.points)
            }
            if (options.showConnections) {
                context.beginPath()
                for (const link of scene.links) {
                    if (link.from.region_id !== hoverRegionId && link.to.region_id !== hoverRegionId) continue
                    context.moveTo(link.from.x, link.from.y)
                    context.lineTo(link.to.x, link.to.y)
                }
                context.globalAlpha = .62
                context.strokeStyle = '#94a3b8'
                context.lineWidth = .8
                context.stroke()
            }
            if (options.showSystems) {
                scene.nodes.forEach((node, index) => {
                    if (node.region_id !== hoverRegionId) return
                    const ratio = values ? mapActivityRatio(layer, values[index]!, maximum) : 0
                    context.globalAlpha = 1
                    context.fillStyle = values ? heatColor(ratio) : secColorStr(node.security)
                    context.beginPath(); context.arc(node.x, node.y, values ? 1.4 + ratio * 4.5 : 1.7, 0, Math.PI * 2); context.fill()
                })
            }
            const region = scene.regions.find(item => item.id === hoverRegionId)
            if (region) {
                const top = Math.min(...scene.nodes.filter(node => node.region_id === region.id).map(node => node.y))
                context.globalAlpha = 1
                context.font = '800 22px sans-serif'
                context.textAlign = 'center'
                context.lineWidth = 5
                context.strokeStyle = '#05060a'
                context.fillStyle = '#fff'
                context.strokeText(region.name.toUpperCase(), region.x, top - 14)
                context.fillText(region.name.toUpperCase(), region.x, top - 14)
            }
        }
        context.globalAlpha = 1
        updateMarkerPositions()
        requestPulseDraw()
    }
    const buildMarkers = () => {
        markerPicker.hide()
        markerLayer.replaceChildren()
        markerNodes = []
        if (!battleSystems.length || !scene.nodes.length) { requestDraw(); return }
        const regionTotals = regionId ? null : battleRegionTotals(battleSystems)
        const markers = regionId
            ? battleSystems.flatMap(system => {
                const node = scene.byID.get(system.solar_system_id)
                return node ? [{ id: system.solar_system_id, name: node.system_name, count: system.battle_count,
                    isk: system.total_isk_destroyed, x: node.x, y: node.y, system }] : []
            })
            : scene.regions.flatMap(region => {
                const total = regionTotals?.get(region.id)
                return total ? [{ id: region.id, name: region.name, count: total.battle_count,
                    isk: total.total_isk_destroyed, x: region.x, y: region.y, system: null }] : []
            })
        for (const marker of markers) {
            const button = document.createElement('button')
            button.type = 'button'
            button.className = 'go-map-battle-marker'
            const noun = options.markerNoun || 'battles'
            const countLabel = `${marker.count} ${marker.count === 1 ? noun.replace(/s$/, '') : noun}`
            const valueLabel = options.markerValueLabel || 'ISK destroyed'
            button.setAttribute('aria-label', `${marker.name}: ${countLabel}, ${formatIsk(marker.isk)} ${valueLabel}`)
            button.title = `${marker.name} · ${countLabel} · ${formatIsk(marker.isk)} ${valueLabel}`
            const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg')
            svg.setAttribute('aria-hidden', 'true')
            const use = document.createElementNS('http://www.w3.org/2000/svg', 'use')
            use.setAttribute('href', '/assets/icons.svg#swords')
            svg.append(use)
            const amount = document.createElement('span')
            amount.textContent = String(marker.count)
            button.append(svg, amount)
            if (regionId) {
                const name = document.createElement('small')
                name.textContent = marker.name
                button.append(name)
            }
            const choice={node:button,name:marker.name,count:countLabel,select:()=>marker.system
                ? options.onBattleSystem?.(marker.system)
                : options.onBattleRegion?.(marker.id)}
            button.addEventListener('click', event => markerPicker.activate(event,choice,markerNodes))
            markerLayer.append(button)
            markerNodes.push({ ...choice, x: marker.x, y: marker.y })
        }
        requestDraw()
    }
    const resetView = () => {
        const width = host.clientWidth, height = host.clientHeight
        const bounds = scene.bounds
        const hasPanel = !!options.panel
        const mobile = window.matchMedia('(max-width: 639px)').matches
        const panel = options.panel?.getBoundingClientRect(), hostRect = host.getBoundingClientRect()
        const leftInset = hasPanel && !mobile && panel ? panel.right - hostRect.left + 12 : 0
        const topInset = mobile ? hasPanel ? 52 : 108 : 0
        const bottomEdge = hasPanel && mobile && panel ? panel.top - hostRect.top - 12 : height - (mobile ? 56 : 0)
        const availableWidth = Math.max(1, width - leftInset), availableHeight = Math.max(1, bottomEdge - topInset)
        const scale = Math.min(availableWidth / bounds.width, availableHeight / bounds.height) * .96
        minimumScale = Math.min(.25, scale / 2)
        camera = { x: leftInset + availableWidth / 2 - (bounds.x + bounds.width / 2) * scale,
            y: topInset + availableHeight / 2 - (bounds.y + bounds.height / 2) * scale, scale }
        requestDraw()
    }
    const rebuild = () => { scene = buildScene(data, regionId); buildActivity(); hoverRegionId = hoverAllianceId = null; livePulses.clear(); focusedLiveSystem = null; buildSovereignty(); resetView(); buildMarkers(); refreshScopeDetails() }
    const onMove = (event: PointerEvent) => {
        const point = pointer(event)
        if (drag) {
            const dx = point.x - drag.x, dy = point.y - drag.y
            if (Math.abs(dx) + Math.abs(dy) > 2) drag.moved = true
            camera.x += dx; camera.y += dy
            drag.x = point.x; drag.y = point.y
            hoverRegionId = null
            options.onHoverAlliance?.(null, point)
            options.onHoverRegion?.(null, point)
            requestDraw()
            return
        }
        const world = toWorld(point.x, point.y)
        const bounds = scene.bounds
        const inside = world.x >= bounds.x && world.x <= bounds.x + bounds.width && world.y >= bounds.y && world.y <= bounds.y + bounds.height
        if (options.baseLayer === 'sovereignty') {
            const alliance = inside && sovereigntyTerritories ? sovereigntyOwnerAt(sovereigntyTerritories, world.x, world.y) : null
            if (hoverAllianceId !== alliance) { hoverAllianceId = alliance; requestDraw() }
            options.onHoverAlliance?.(sovereigntySummaries.find(summary => summary.id === alliance) || null, point)
        } else if (options.baseLayer !== 'live' && options.baseLayer !== 'aiid') {
            const index = inside ? scene.delaunay?.find(world.x, world.y) : undefined
            const region = index == null ? null : scene.nodes[index]?.region_id ?? null
            if (hoverRegionId !== region) { hoverRegionId = region; requestDraw() }
            options.onHoverRegion?.(scene.regions.find(summary => summary.id === region) || null, point)
        }
    }
    const onDown = (event: PointerEvent) => {
        if (event.button !== 0) return
        const point = pointer(event)
        drag = { x: point.x, y: point.y, moved: false }
        canvas.setPointerCapture(event.pointerId)
    }
    const onUp = (event: PointerEvent) => {
        if (!drag) return
        const moved = drag.moved
        drag = null
        if (canvas.hasPointerCapture(event.pointerId)) canvas.releasePointerCapture(event.pointerId)
        if (moved) return
        inspectPoint(pointer(event))
    }
    const inspectPoint = (point: { x: number; y: number }) => {
        const world = toWorld(point.x, point.y), bounds = scene.bounds
        const inside = world.x >= bounds.x && world.x <= bounds.x + bounds.width && world.y >= bounds.y && world.y <= bounds.y + bounds.height
        if (!inside) return
        if (options.baseLayer === 'sovereignty') {
            const alliance = sovereigntyTerritories ? sovereigntyOwnerAt(sovereigntyTerritories, world.x, world.y) : null
            if (alliance != null) options.onAlliance?.(alliance)
        } else if (!regionId && options.baseLayer !== 'live' && options.baseLayer !== 'aiid') {
            const index = scene.delaunay?.find(world.x, world.y)
            const region = index == null ? null : scene.nodes[index]?.region_id ?? null
            if (region != null) options.onBattleRegion?.(region)
        }
    }
    const onWheel = (event: WheelEvent) => {
        event.preventDefault()
        zoomAt(event.deltaY < 0 ? 1.15 : 1 / 1.15, pointer(event))
    }
    const zoomAt = (factor: number, point = { x: host.clientWidth / 2, y: host.clientHeight / 2 }) => {
        const before = toWorld(point.x, point.y)
        const scale = Math.min(10, Math.max(minimumScale, camera.scale * factor))
        camera = { x: point.x - before.x * scale, y: point.y - before.y * scale, scale }
        requestDraw()
    }
    const onKey = (event: KeyboardEvent) => {
        if (event.altKey || event.ctrlKey || event.metaKey) return
        const step = event.shiftKey ? 120 : 40
        switch (event.key) {
            case 'ArrowLeft': camera.x += step; break
            case 'ArrowRight': camera.x -= step; break
            case 'ArrowUp': camera.y += step; break
            case 'ArrowDown': camera.y -= step; break
            case '+': case '=': zoomAt(1.15); break
            case '-': case '_': zoomAt(1 / 1.15); break
            case '0': resetView(); break
            case 'Enter': inspectPoint({ x: host.clientWidth / 2, y: host.clientHeight / 2 }); break
            default: return
        }
        event.preventDefault()
        requestDraw()
    }
    canvas.addEventListener('pointermove', onMove)
    canvas.addEventListener('pointerdown', onDown)
    canvas.addEventListener('pointerup', onUp)
    canvas.addEventListener('pointercancel', onUp)
    canvas.addEventListener('pointerleave', () => {
        if (!drag) { hoverRegionId = null; hoverAllianceId = null; options.onHoverAlliance?.(null, { x: 0, y: 0 }); options.onHoverRegion?.(null, { x: 0, y: 0 }); requestDraw() }
    })
    canvas.addEventListener('wheel', onWheel, { passive: false })
    canvas.addEventListener('keydown', onKey)
    zoomIn.addEventListener('click', () => zoomAt(1.15))
    zoomOut.addEventListener('click', () => zoomAt(1 / 1.15))
    reset.addEventListener('click', resetView)
    const onVisibility = () => { if (!document.hidden) requestPulseDraw() }
    document.addEventListener('visibilitychange', onVisibility)
    const observer = new ResizeObserver(resetView)
    observer.observe(host)

    return {
        setData(next) { data = next; rebuild() },
        setRegion(next) { regionId = next; rebuild() },
        setBattleSystems(next) { battleSystems = next; buildMarkers() },
        setLayers(next) {
            const changedBase = next.baseLayer != null && next.baseLayer !== options.baseLayer
            options = { ...options, ...next }
            if (changedBase) buildSovereignty()
            buildActivity()
            refreshScopeDetails()
            requestDraw()
        },
        pulseKill(systemId, value, ageMs = 0) {
            if (!scene.byID.has(systemId)) return
            livePulses.set(systemId, { startedAt: performance.now() - ageMs, value })
            requestPulseDraw()
        },
        focusLiveSystem(systemId, origin) {
            focusedLiveSystem = systemId != null && origin ? { id: systemId, origin } : null
            requestPulseDraw()
        },
        allianceSummaries() { return sovereigntySummaries },
        allianceColor(allianceId) { return allianceColors.get(allianceId) || '#64748b' },
        focusAlliance(allianceId) {
            hoverAllianceId = allianceId
            hoverRegionId = null
            requestDraw()
        },
        activityLegend() {
            const layer = options.activityLayer || 'none'
            const peak = layer === 'danger' ? `${activityMaximum.toFixed(1)} / 100 jumps`
                : layer === 'activity' ? `${Math.round(activityMaximum)} score`
                : Math.round(activityMaximum).toLocaleString('en-US')
            return { label: mapActivityLayerLabel(layer), maximum: activityMaximum, peak }
        },
        resetView,
        destroy() {
            destroyed = true
            markerPicker.hide()
            if (frame) cancelAnimationFrame(frame)
            if (pulseFrame) cancelAnimationFrame(pulseFrame)
            observer.disconnect()
            canvas.removeEventListener('pointermove', onMove)
            canvas.removeEventListener('pointerdown', onDown)
            canvas.removeEventListener('pointerup', onUp)
            canvas.removeEventListener('pointercancel', onUp)
            canvas.removeEventListener('wheel', onWheel)
            canvas.removeEventListener('keydown', onKey)
            document.removeEventListener('visibilitychange', onVisibility)
            host.replaceChildren()
        },
    }
}
