<script setup lang="ts">
import { Delaunay } from 'd3-delaunay'
import type { MapActivity, MapActivityHours, MapActivityLayer, MapRenderBaseLayer } from '~/utils/map/layers'
import { heatColor, isMapActivityHours, isMapActivityLayer, isMapBaseLayer, isMapLayer, mapActivityLayerLabel, mapActivityRatio, mapActivityValue } from '~/utils/map/layers'
import { secColorStr, secLabel } from '~/utils/map/colors'

const route = useRoute()
const router = useRouter()
const id = Number(route.params.id)
if (!Number.isInteger(id) || id < 1 || id > 2147483647) throw createError({ statusCode: 404, statusMessage: 'Region not found' })

const legacyLayer = isMapLayer(route.query.layer) ? route.query.layer : undefined
const baseLayer = ref<MapRenderBaseLayer>(isMapBaseLayer(route.query.base) ? route.query.base : legacyLayer === 'security' ? 'security' : 'geography')
const legacyActivityLayer: MapActivityLayer = isMapActivityLayer(legacyLayer) ? legacyLayer : 'none'
const activityLayer = ref<MapActivityLayer>(isMapActivityLayer(route.query.activity) ? route.query.activity : legacyActivityLayer)
const requestedHours = Number(route.query.hours)
const activityHours = ref<MapActivityHours>(isMapActivityHours(requestedHours) ? requestedHours : 24)
const showConnections = ref(route.query.routes !== '0')
const showSystems = ref(route.query.systems !== '0')
const showLabels = ref(route.query.labels !== '0')
const sharedMapQuery = computed(() => ({
    base: baseLayer.value,
    activity: activityLayer.value,
    hours: activityHours.value,
    routes: showConnections.value ? undefined : '0',
    systems: showSystems.value ? undefined : '0',
    labels: showLabels.value ? undefined : '0',
}))

function navigateToExternalMap(regionId: number | null, systemId: number) {
    if (regionId) return navigateTo({ path: `/map/region/${regionId}`, query: sharedMapQuery.value })
    return navigateTo(`/system/${systemId}`)
}

const { data, pending, error } = await useApiFetch<any>(() => `/api/map/region/${id}?hours=${activityHours.value}`, { watch: [activityHours] })

if (error.value) {
    throw createError({
        statusCode: error.value.statusCode || 404,
        statusMessage: (error.value.data as any)?.message || 'Region not found',
    })
}

const region = computed(() => data.value?.region)
watch(
    [baseLayer, activityLayer, activityHours, showConnections, showSystems, showLabels],
    ([base, activity, hours, connections, systems, labels]) => router.replace({ query: {
        base, activity, hours,
        routes: connections ? undefined : '0',
        systems: systems ? undefined : '0',
        labels: labels ? undefined : '0',
    } }),
)

useHead({ title: computed(() => region.value?.name ? `${region.value.name} Map` : 'Region Map') })
useSeoMeta({
    description: computed(() => region.value?.name ? `Interactive map of ${region.value.name} — all systems and jump connections.` : 'Region map'),
})

const CONST_COLORS = [
    '#3b82f6', '#22c55e', '#eab308', '#ef4444', '#8b5cf6',
    '#06b6d4', '#f97316', '#ec4899', '#14b8a6', '#a855f7',
    '#64748b', '#84cc16', '#f43f5e', '#0ea5e9', '#d946ef',
]

// Dense regions can place neighbouring systems very close together. Keeping
// the canonical CCP layout while slightly reducing each system footprint gives
// routes and labels breathing room without changing the universe overview.
const R = 16
// Inner sun + planet sizes for the mini-system render
const SUN_R = 2
const PLANET_R = 1

interface LaidNode {
    id: number
    x: number
    y: number
    constellation_id: number
    system_name: string
    security: number
}

interface LaidLink {
    from: number
    to: number
}

interface VoronoiCell {
    path: string
    constellation_id: number
    security: number
}

function relaxSystemSpacing(nodes: LaidNode[]): void {
    const anchors = nodes.map(node => ({ x: node.x, y: node.y }))
    const minimumDistance = R * 4.25
    for (let iteration = 0; iteration < 64; iteration++) {
        for (let leftIndex = 0; leftIndex < nodes.length; leftIndex++) {
            const left = nodes[leftIndex]!
            for (let rightIndex = leftIndex + 1; rightIndex < nodes.length; rightIndex++) {
                const right = nodes[rightIndex]!
                let dx = right.x - left.x
                let dy = right.y - left.y
                let distance = Math.hypot(dx, dy)
                if (distance >= minimumDistance) continue
                if (distance < 0.001) {
                    const angle = ((left.id + right.id) % 360) * Math.PI / 180
                    dx = Math.cos(angle); dy = Math.sin(angle); distance = 1
                }
                const push = (minimumDistance - distance) * 0.24
                const pushX = dx / distance * push
                const pushY = dy / distance * push
                left.x -= pushX; left.y -= pushY
                right.x += pushX; right.y += pushY
            }
        }
        // Keep the recognizable CCP layout while allowing congested local
        // clusters to open up just enough for system names and routes.
        for (let index = 0; index < nodes.length; index++) {
            nodes[index]!.x += (anchors[index]!.x - nodes[index]!.x) * 0.015
            nodes[index]!.y += (anchors[index]!.y - nodes[index]!.y) * 0.015
        }
    }
}

const constIds = computed<number[]>(() => (data.value?.constellations ?? []).map((c: any) => c.constellation_id))

function constColor(constId: number): string {
    const idx = constIds.value.indexOf(constId)
    return CONST_COLORS[Math.max(0, idx) % CONST_COLORS.length]!
}

// Normalize CCP 2D coords to a fixed canvas — no force sim, layout is whatever CCP ships.
// Scale uniformly so aspect ratio of the region is preserved.
interface Bbox { x: number; y: number; w: number; h: number }
interface Transform { has2d: boolean; cx: number; cz: number; scale: number }

const layout = computed<{ nodes: LaidNode[]; links: LaidLink[]; cells: VoronoiCell[]; bbox: Bbox; transform: Transform }>(() => {
    const empty: Bbox = { x: 0, y: 0, w: 1000, h: 800 }
    const emptyT: Transform = { has2d: false, cx: 0, cz: 0, scale: 1 }
    if (!data.value) return { nodes: [], links: [], cells: [], bbox: empty, transform: emptyT }
    const { systems, jumps } = data.value
    if (!systems?.length) return { nodes: [], links: [], cells: [], bbox: empty, transform: emptyT }

    const has2d = systems.some((s: any) => s.x2d != null && s.z2d != null)
    const getX = (s: any) => has2d && s.x2d != null ? s.x2d : s.x
    const getZ = (s: any) => has2d && s.z2d != null ? s.z2d : s.z

    let minX = Infinity, maxX = -Infinity, minZ = Infinity, maxZ = -Infinity
    for (const s of systems) {
        const sx = getX(s), sz = getZ(s)
        if (sx < minX) minX = sx
        if (sx > maxX) maxX = sx
        if (sz < minZ) minZ = sz
        if (sz > maxZ) maxZ = sz
    }

    // Target canvas: ~1200x900 content area, uniform scale to preserve region shape.
    const targetW = 1200
    const targetH = 900
    const rangeX = (maxX - minX) || 1
    const rangeZ = (maxZ - minZ) || 1
    const scale = Math.min(targetW / rangeX, targetH / rangeZ)

    const cx = (minX + maxX) / 2
    const cz = (minZ + maxZ) / 2

    const nodes: LaidNode[] = systems.map((s: any) => ({
        id: s.solar_system_id,
        // screen coords: y negated (CCP z grows away from viewer; we want south-down)
        x: (getX(s) - cx) * scale,
        y: -(getZ(s) - cz) * scale,
        constellation_id: s.constellation_id,
        system_name: s.system_name,
        security: s.security,
    }))
    relaxSystemSpacing(nodes)

    const ids = new Set<number>(nodes.map(n => n.id))
    const links: LaidLink[] = (jumps ?? [])
        .filter((j: any) => ids.has(j.from_solar_system_id) && ids.has(j.to_solar_system_id))
        .map((j: any) => ({ from: j.from_solar_system_id, to: j.to_solar_system_id }))

    // viewBox fits content + label room (names sit below each circle).
    // Padding leaves space for external-jump stubs poking outside the cluster.
    const pad = 150
    const labelRoom = 24
    let nMinX = Infinity, nMaxX = -Infinity, nMinY = Infinity, nMaxY = -Infinity
    for (const n of nodes) {
        if (n.x < nMinX) nMinX = n.x
        if (n.x > nMaxX) nMaxX = n.x
        if (n.y < nMinY) nMinY = n.y
        if (n.y > nMaxY) nMaxY = n.y
    }
    const vbX = nMinX - pad
    const vbY = nMinY - pad
    const vbW = (nMaxX - nMinX) + pad * 2
    const vbH = (nMaxY - nMinY) + pad * 2 + labelRoom

    // Voronoi cells — each system gets a polygon tinted by its constellation.
    // Clipped to the bbox so cells along the hull don't extend to infinity.
    const delaunay = Delaunay.from(nodes, (n) => n.x, (n) => n.y)
    const voronoi = delaunay.voronoi([vbX, vbY, vbX + vbW, vbY + vbH])
    const cells: VoronoiCell[] = []
    for (let i = 0; i < nodes.length; i++) {
        const path = voronoi.renderCell(i)
        if (!path) continue
        cells.push({ path, constellation_id: nodes[i]!.constellation_id, security: nodes[i]!.security })
    }

    return {
        nodes, links, cells,
        bbox: { x: vbX, y: vbY, w: vbW, h: vbH },
        transform: { has2d, cx, cz, scale },
    }
})

// ─── Pan / zoom state ────────────────────────────────────
const svgRef = ref<SVGSVGElement | null>(null)
const zoom = ref(1) // 1 = fit to bbox; >1 = zoomed in
const panX = ref(0) // offset in viewBox units
const panY = ref(0)

const viewBoxStr = computed(() => {
    const b = layout.value.bbox
    const w = b.w / zoom.value
    const h = b.h / zoom.value
    // Pan is applied after zoom so panX/Y are in current-scale viewBox units.
    const x = b.x + (b.w - w) / 2 + panX.value
    const y = b.y + (b.h - h) / 2 + panY.value
    return `${x} ${y} ${w} ${h}`
})

watch(layout, () => {
    // Reset view whenever the region changes.
    zoom.value = 1
    panX.value = 0
    panY.value = 0
})

function resetView() {
    zoom.value = 1
    panX.value = 0
    panY.value = 0
}

function svgPointFromEvent(e: PointerEvent | WheelEvent): { x: number; y: number } {
    const svg = svgRef.value
    if (!svg) return { x: 0, y: 0 }
    const pt = svg.createSVGPoint()
    pt.x = e.clientX
    pt.y = e.clientY
    const ctm = svg.getScreenCTM()?.inverse()
    if (!ctm) return { x: 0, y: 0 }
    const p = pt.matrixTransform(ctm)
    return { x: p.x, y: p.y }
}

function onWheel(e: WheelEvent) {
    e.preventDefault()
    const before = svgPointFromEvent(e)
    const factor = e.deltaY < 0 ? 1.15 : 1 / 1.15
    const next = Math.min(20, Math.max(0.5, zoom.value * factor))
    if (next === zoom.value) return
    zoom.value = next
    // After zoom, keep cursor's world point stationary by adjusting pan.
    const after = svgPointFromEvent(e)
    panX.value += before.x - after.x
    panY.value += before.y - after.y
}

const panning = ref(false)
// panStart holds raw screen pixels, not SVG coords — avoids CTM-feedback jitter
// (re-reading the CTM after a viewBox change reports a moved world, and the
// subsequent delta gets clobbered, making pan jerky).
let panStart: { x: number; y: number } | null = null
let panOrigin: { x: number; y: number } | null = null
let panMoved = false
const DRAG_THRESHOLD_PX = 3

function pxToViewBox(): { sx: number; sy: number } {
    const svg = svgRef.value
    if (!svg) return { sx: 1, sy: 1 }
    const rect = svg.getBoundingClientRect()
    const vbW = layout.value.bbox.w / zoom.value
    const vbH = layout.value.bbox.h / zoom.value
    return {
        sx: rect.width > 0 ? vbW / rect.width : 1,
        sy: rect.height > 0 ? vbH / rect.height : 1,
    }
}

function onBgPointerDown(e: PointerEvent) {
    // Left-click anywhere on the svg starts a pan — even over a node/stub. If the
    // pointer doesn't travel past DRAG_THRESHOLD_PX, onClickCapture lets the click
    // through so nodes/stubs still navigate. We deliberately DON'T setPointerCapture
    // because that retargets the synthesized click event to the captured element,
    // breaking @click on child <g>s. Window-level listeners track the rest of the drag.
    if (e.button !== 0) return
    panning.value = true
    panMoved = false
    panStart = { x: e.clientX, y: e.clientY }
    panOrigin = { x: e.clientX, y: e.clientY }
    onNodeHoverLeave()
    window.addEventListener('pointermove', onWindowPointerMove)
    window.addEventListener('pointerup', onWindowPointerUp)
    window.addEventListener('pointercancel', onWindowPointerUp)
}

function onWindowPointerMove(e: PointerEvent) {
    if (!panning.value || !panStart || !panOrigin) return
    if (!panMoved) {
        const dx = e.clientX - panOrigin.x
        const dy = e.clientY - panOrigin.y
        if (dx * dx + dy * dy >= DRAG_THRESHOLD_PX * DRAG_THRESHOLD_PX) panMoved = true
        else return
    }
    const { sx, sy } = pxToViewBox()
    panX.value -= (e.clientX - panStart.x) * sx
    panY.value -= (e.clientY - panStart.y) * sy
    panStart = { x: e.clientX, y: e.clientY }
}

function onWindowPointerUp() {
    panning.value = false
    panStart = null
    panOrigin = null
    window.removeEventListener('pointermove', onWindowPointerMove)
    window.removeEventListener('pointerup', onWindowPointerUp)
    window.removeEventListener('pointercancel', onWindowPointerUp)
}

function onSvgClickCapture(e: MouseEvent) {
    if (panMoved) {
        e.stopPropagation()
        e.preventDefault()
        panMoved = false
    }
}

onUnmounted(() => {
    window.removeEventListener('pointermove', onWindowPointerMove)
    window.removeEventListener('pointerup', onWindowPointerUp)
    window.removeEventListener('pointercancel', onWindowPointerUp)
})

const nodeById = computed<Map<number, LaidNode>>(() => {
    const m = new Map<number, LaidNode>()
    for (const n of layout.value.nodes) m.set(n.id, n)
    return m
})

function linkColor(link: LaidLink): string {
    const src = nodeById.value.get(link.from)
    return src ? constColor(src.constellation_id) : '#64748b'
}

// ─── Lookups for hover card ──────────────────────────────
const activityById = computed<Map<number, MapActivity>>(() => {
    const m = new Map<number, MapActivity>()
    for (const row of (data.value?.activity ?? []) as any[]) {
        m.set(row.system_id, {
            ship_kills: row.ship_kills ?? 0,
            pod_kills: row.pod_kills ?? 0,
            npc_kills: row.npc_kills ?? 0,
            ship_jumps: row.ship_jumps ?? 0,
        })
    }
    return m
})

const activityMaximums = computed<MapActivity>(() => ({
    ship_kills: Math.max(0, ...[...activityById.value.values()].map(activity => activity.ship_kills)),
    pod_kills: Math.max(0, ...[...activityById.value.values()].map(activity => activity.pod_kills)),
    npc_kills: Math.max(0, ...[...activityById.value.values()].map(activity => activity.npc_kills)),
    ship_jumps: Math.max(0, ...[...activityById.value.values()].map(activity => activity.ship_jumps)),
}))

function systemActivityValue(node: LaidNode): number {
    return mapActivityValue(activityLayer.value, activityById.value.get(node.id), activityMaximums.value)
}

const maximumActivity = computed(() => Math.max(0, ...layout.value.nodes.map(systemActivityValue)))
const activityWindowLabel = computed(() => activityHours.value === 168 ? '7 days' : `${activityHours.value} hour${activityHours.value === 1 ? '' : 's'}`)
const activeLayerLabel = computed(() => mapActivityLayerLabel(activityLayer.value))
const maximumActivityLabel = computed(() => {
    if (activityLayer.value === 'danger') return `${maximumActivity.value.toFixed(1)} / 100 jumps`
    if (activityLayer.value === 'activity') return `${Math.round(maximumActivity.value)} score`
    return formatNumber(Math.round(maximumActivity.value))
})

function systemHaloFill(node: LaidNode): string {
    if (activityLayer.value === 'none') return baseLayer.value === 'geography' ? constColor(node.constellation_id) : secColorStr(node.security)
    return heatColor(mapActivityRatio(activityLayer.value, systemActivityValue(node), maximumActivity.value))
}

function systemHaloRadius(node: LaidNode): number {
    if (activityLayer.value === 'none') return R + 2.5
    return R + 2.5 + mapActivityRatio(activityLayer.value, systemActivityValue(node), maximumActivity.value) * 10
}

const gateCountById = computed<Map<number, number>>(() => {
    const m = new Map<number, number>()
    for (const l of layout.value.links) {
        m.set(l.from, (m.get(l.from) ?? 0) + 1)
        m.set(l.to, (m.get(l.to) ?? 0) + 1)
    }
    return m
})

const constNameById = computed<Map<number, string>>(() => {
    const m = new Map<number, string>()
    for (const c of (data.value?.constellations ?? []) as any[]) m.set(c.constellation_id, c.constellation_name)
    return m
})

const constellationLabels = computed(() => {
    const groups = new Map<number, { x: number; y: number; count: number }>()
    for (const node of layout.value.nodes) {
        const group = groups.get(node.constellation_id) ?? { x: 0, y: 0, count: 0 }
        group.x += node.x; group.y += node.y; group.count++
        groups.set(node.constellation_id, group)
    }
    return [...groups].map(([constellationId, group]) => ({
        id: constellationId,
        name: constNameById.value.get(constellationId) ?? '',
        x: group.x / group.count,
        y: group.y / group.count,
    }))
})

const systemQuery = ref('')
const searchFocused = ref(false)
function closeSearchSoon() { window.setTimeout(() => { searchFocused.value = false }, 150) }
const systemResults = computed(() => {
    const query = systemQuery.value.trim().toLowerCase()
    if (!query) return []
    return layout.value.nodes.filter(node => node.system_name.toLowerCase().includes(query)).slice(0, 8)
})

const systemRawById = computed<Map<number, any>>(() => {
    const m = new Map<number, any>()
    for (const s of (data.value?.systems ?? []) as any[]) m.set(s.solar_system_id, s)
    return m
})

// ─── Mini-system render data ─────────────────────────────
// For each system, we draw a tiny sun + planets inside the node disc. Distances
// are normalized per-system so the outer planet sits near the disc edge.
interface MiniSystem {
    hasSun: boolean
    planets: Array<{ dx: number; dy: number }>
}
const PLANET_FIT = R - 2 // outer planet sits 2u inside the disc edge

const miniSystemBySystem = computed<Map<number, MiniSystem>>(() => {
    const m = new Map<number, MiniSystem>()
    const rows = (data.value?.celestials ?? []) as Array<{ system_id: number; group_id: number; x: number; z: number }>
    const bySys = new Map<number, typeof rows>()
    for (const r of rows) {
        if (!bySys.has(r.system_id)) bySys.set(r.system_id, [])
        bySys.get(r.system_id)!.push(r)
    }
    for (const [sysId, list] of bySys) {
        const sun = list.find(r => r.group_id === 6)
        // Sun coords act as the local origin. If no sun, use the planet centroid.
        const planets = list.filter(r => r.group_id === 7)
        const cx = sun ? sun.x : (planets.length ? planets.reduce((s, r) => s + r.x, 0) / planets.length : 0)
        const cz = sun ? sun.z : (planets.length ? planets.reduce((s, r) => s + r.z, 0) / planets.length : 0)
        let maxDist = 0
        for (const p of planets) {
            const d = Math.hypot(p.x - cx, p.z - cz)
            if (d > maxDist) maxDist = d
        }
        const scale = maxDist > 0 ? PLANET_FIT / maxDist : 0
        m.set(sysId, {
            hasSun: !!sun,
            // y negated so screen "up" matches CCP +z (consistent with the region layout).
            planets: planets.map(p => ({ dx: (p.x - cx) * scale, dy: -(p.z - cz) * scale })),
        })
    }
    return m
})

interface NeighborInfo {
    system_id: number
    system_name: string
    security: number
    external: false
}
interface ExternalNeighborInfo {
    system_id: number
    system_name: string | null
    region_id: number | null
    region_name: string | null
    security: number | null
    external: true
    /** Unit vector in screen coords pointing from the internal system toward this external system. */
    dx: number
    dy: number
}

// In-region gates per system: both directions of each link contribute.
const internalNeighborsById = computed<Map<number, NeighborInfo[]>>(() => {
    const m = new Map<number, NeighborInfo[]>()
    const add = (fromId: number, toId: number) => {
        const to = nodeById.value.get(toId)
        if (!to) return
        if (!m.has(fromId)) m.set(fromId, [])
        m.get(fromId)!.push({
            system_id: to.id,
            system_name: to.system_name,
            security: to.security,
            external: false,
        })
    }
    for (const l of layout.value.links) {
        add(l.from, l.to)
        add(l.to, l.from)
    }
    return m
})

// Out-of-region jumps per system, with screen-space direction unit vector.
// Direction is computed by transforming the external system's CCP coords
// through the same cx/cz/scale used for internal nodes.
const externalNeighborsById = computed<Map<number, ExternalNeighborInfo[]>>(() => {
    const m = new Map<number, ExternalNeighborInfo[]>()
    const jumps = (data.value?.externalJumps ?? []) as any[]
    const { has2d, cx, cz, scale } = layout.value.transform
    for (const j of jumps) {
        const internal = nodeById.value.get(j.internal_system_id)
        if (!internal) continue
        const ex = has2d && j.external_x2d != null ? j.external_x2d : j.external_x
        const ez = has2d && j.external_z2d != null ? j.external_z2d : j.external_z
        let dx = 0, dy = 0
        if (ex != null && ez != null) {
            const extScreenX = (ex - cx) * scale
            const extScreenY = -(ez - cz) * scale
            dx = extScreenX - internal.x
            dy = extScreenY - internal.y
            const len = Math.hypot(dx, dy)
            if (len > 1e-6) { dx /= len; dy /= len }
            else { dx = 0; dy = -1 }
        } else {
            // No external coords → point straight up as a last-resort fallback.
            dx = 0; dy = -1
        }
        if (!m.has(j.internal_system_id)) m.set(j.internal_system_id, [])
        m.get(j.internal_system_id)!.push({
            system_id: j.external_system_id,
            system_name: j.external_system_name,
            region_id: j.external_region_id,
            region_name: j.external_region_name,
            security: j.external_security ?? null,
            external: true,
            dx, dy,
        })
    }
    return m
})

// "Stub" markers — dotlan-style fake system attached to each in-region system
// that has out-of-region jumps. Direction is correct, distance is fixed (cosmetic).
interface ExternalStub {
    internal_id: number
    external_id: number
    region_id: number | null
    name: string | null
    region_name: string | null
    security: number | null
    /** Stub center in viewBox coords. */
    cx: number; cy: number
    /** Connector line from the in-region circle's edge to the stub circle's edge. */
    line_x1: number; line_y1: number; line_x2: number; line_y2: number
}

const STUB_DIST = 90
const STUB_R = 3.5

const externalStubs = computed<ExternalStub[]>(() => {
    const out: ExternalStub[] = []
    for (const [internalId, list] of externalNeighborsById.value) {
        const n = nodeById.value.get(internalId)
        if (!n) continue
        for (const e of list) {
            const cx = n.x + e.dx * STUB_DIST
            const cy = n.y + e.dy * STUB_DIST
            out.push({
                internal_id: internalId,
                external_id: e.system_id,
                region_id: e.region_id,
                name: e.system_name,
                region_name: e.region_name,
                security: e.security,
                cx, cy,
                line_x1: n.x + e.dx * (R + 1),
                line_y1: n.y + e.dy * (R + 1),
                line_x2: cx - e.dx * (STUB_R + 1),
                line_y2: cy - e.dy * (STUB_R + 1),
            })
        }
    }
    return out
})

// ─── Hover card state ────────────────────────────────────
const hoverNode = ref<LaidNode | null>(null)
const hoverPos = ref<{ x: number; y: number }>({ x: 0, y: 0 })
let hoverTimer: ReturnType<typeof setTimeout> | null = null
const HOVER_DELAY_MS = 250

function onNodeHoverEnter(e: PointerEvent, node: LaidNode) {
    hoverPos.value = { x: e.clientX, y: e.clientY }
    if (hoverTimer) clearTimeout(hoverTimer)
    hoverTimer = setTimeout(() => { hoverNode.value = node }, HOVER_DELAY_MS)
}

function onNodeHoverMove(e: PointerEvent) {
    hoverPos.value = { x: e.clientX, y: e.clientY }
}

function onNodeHoverLeave() {
    if (hoverTimer) { clearTimeout(hoverTimer); hoverTimer = null }
    hoverNode.value = null
}

const mapContainerRef = ref<HTMLElement | null>(null)

// Position card relative to the map container, offset from the cursor, kept inside
// the container's right/bottom edges (flips to top-left of cursor if it would overflow).
const hoverCardStyle = computed(() => {
    const container = mapContainerRef.value
    if (!container || !hoverNode.value) return { display: 'none' }
    const rect = container.getBoundingClientRect()
    const localX = hoverPos.value.x - rect.left
    const localY = hoverPos.value.y - rect.top
    const cardW = 260
    const cardH = 320 // rough — card is content-sized, grows with gate list
    const offset = 14
    let left = localX + offset
    let top = localY + offset
    if (left + cardW > rect.width) left = localX - offset - cardW
    if (top + cardH > rect.height) top = localY - offset - cardH
    return { left: `${Math.max(4, left)}px`, top: `${Math.max(4, top)}px` }
})
</script>

<template>
    <div>
        <!-- Header -->
        <div class="glass-panel overflow-hidden mb-4">
            <div class="p-4 flex items-center justify-between gap-4 flex-wrap">
                <div class="flex items-center gap-2 text-sm">
                    <NuxtLink :to="{ path: '/map', query: sharedMapQuery }" class="text-gray-400 hover:text-blue-400 transition-colors">
                        Map <span class="text-gray-600 ml-2">/</span>
                    </NuxtLink>
                    <span class="text-white font-semibold text-lg" :class="pochvenClass(id)">{{ region?.name ?? 'Loading...' }}</span>
                </div>
                <div class="flex items-center gap-2 text-xs text-gray-500">
                    <span v-if="data">{{ data.systems?.length }} systems</span>
                    <template v-if="activityLayer !== 'none'">
                        <span class="text-gray-600">|</span>
                        <span class="flex items-center gap-1.5 text-emerald-400/80"><span class="h-1.5 w-1.5 rounded-full bg-emerald-400" />Hourly data · {{ activityWindowLabel }}</span>
                    </template>
                    <span class="text-gray-600">|</span>
                    <span class="text-gray-400">scroll to zoom · drag to pan</span>
                    <button
                        type="button"
                        class="text-blue-400 hover:text-blue-300 transition-colors"
                        @click="resetView"
                    >
                        Reset
                    </button>
                    <span class="text-gray-600">|</span>
                    <NuxtLink :to="`/region/${id}`" class="text-blue-400 hover:text-blue-300 transition-colors">
                        Region stats
                    </NuxtLink>
                </div>
            </div>
            <MapPixiLayerControls
                v-model:base-layer="baseLayer"
                v-model:activity-layer="activityLayer"
                v-model:hours="activityHours"
                v-model:show-connections="showConnections"
                v-model:show-systems="showSystems"
                v-model:show-labels="showLabels"
            />
        </div>

        <!-- Map -->
        <div ref="mapContainerRef" class="relative rounded-lg overflow-hidden border border-white/[0.08] bg-[#0a0a0f]">
            <div class="absolute left-3 top-3 z-20 w-[min(20rem,calc(100%-6rem))]">
                <div class="flex items-center gap-2 rounded-lg border border-white/[0.1] bg-black/70 px-3 py-2 shadow-lg backdrop-blur-md">
                    <span aria-hidden="true" class="shrink-0 text-sm text-gray-500">⌕</span>
                    <input v-model="systemQuery" type="search" placeholder="Find a system in this region" class="min-w-0 flex-1 bg-transparent text-sm text-white outline-none placeholder:text-gray-600" @focus="searchFocused = true" @blur="closeSearchSoon">
                </div>
                <div v-if="searchFocused && systemQuery" class="mt-1 overflow-hidden rounded-lg border border-white/[0.1] bg-[#101116]/95 shadow-xl backdrop-blur-md">
                    <NuxtLink v-for="system in systemResults" :key="system.id" :to="`/system/${system.id}`" class="flex items-center justify-between px-3 py-2 text-sm hover:bg-white/[0.06]">
                        <span class="text-gray-200">{{ system.system_name }}</span><span class="font-mono text-[10px]" :style="{ color: secColorStr(system.security) }">{{ secLabel(system.security) }}</span>
                    </NuxtLink>
                    <div v-if="!systemResults.length" class="px-3 py-2 text-xs text-gray-500">No matching systems</div>
                </div>
            </div>
            <div v-if="pending" class="h-[80vh] flex items-center justify-center">
                <div class="text-gray-400 text-sm animate-pulse">Loading map data...</div>
            </div>

            <svg
                v-else-if="layout.nodes.length > 0"
                ref="svgRef"
                :viewBox="viewBoxStr"
                class="w-full h-auto select-none"
                :class="[panning ? 'cursor-grabbing' : 'cursor-grab', { 'map-labels-hidden': !showLabels }]"
                preserveAspectRatio="xMidYMid meet"
                @wheel.prevent="onWheel"
                @pointerdown="onBgPointerDown"
                @click.capture="onSvgClickCapture"
            >
                <!-- Base geography or security remains visible beneath an optional
                     activity overlay, matching the universe map. -->
                <defs>
                    <filter id="cellBlur" x="-10%" y="-10%" width="120%" height="120%">
                        <feGaussianBlur stdDeviation="4" />
                    </filter>
                </defs>
                <g filter="url(#cellBlur)" class="pointer-events-none">
                    <path
                        v-for="(cell, i) in layout.cells"
                        :key="'c-' + i"
                        :d="cell.path"
                        :fill="baseLayer === 'geography' ? constColor(cell.constellation_id) : secColorStr(cell.security)"
                        :fill-opacity="baseLayer === 'geography' ? 0.12 : 0.07"
                    />
                </g>

                <!-- Jump lines -->
                <g v-show="showConnections">
                    <line
                        v-for="(link, i) in layout.links"
                        :key="'l-' + i"
                        :x1="nodeById.get(link.from)!.x"
                        :y1="nodeById.get(link.from)!.y"
                        :x2="nodeById.get(link.to)!.x"
                        :y2="nodeById.get(link.to)!.y"
                        :stroke="baseLayer === 'geography' ? linkColor(link) : '#64748b'"
                        :stroke-opacity="baseLayer === 'geography' ? 0.35 : 0.22"
                        stroke-width="1.25"
                        class="pointer-events-none"
                    />
                </g>

                <g class="map-label pointer-events-none">
                    <text
                        v-for="label in constellationLabels"
                        :key="label.id"
                        :x="label.x"
                        :y="label.y"
                        text-anchor="middle"
                        dominant-baseline="middle"
                        fill="rgba(255,255,255,.48)"
                        stroke="rgba(0,0,0,.96)"
                        stroke-width="4"
                        paint-order="stroke"
                        font-size="20"
                        font-weight="800"
                        letter-spacing="2.5"
                    >
                        {{ label.name.toUpperCase() }}
                    </text>
                </g>

                <!-- Out-of-region stubs — dotlan-style fake system attached to the
                     in-region system in the correct direction (distance is fixed). -->
                <g v-show="showConnections">
                    <g
                        v-for="s in externalStubs"
                        :key="'stub-' + s.internal_id + '-' + s.external_id"
                        class="cursor-pointer"
                        @click="navigateToExternalMap(s.region_id, s.external_id)"
                    >
                    <line
                        :x1="s.line_x1"
                        :y1="s.line_y1"
                        :x2="s.line_x2"
                        :y2="s.line_y2"
                        stroke="#93c5fd"
                        stroke-opacity="0.55"
                        stroke-width="0.9"
                        stroke-dasharray="2 2"
                        class="pointer-events-none"
                    />
                    <circle
                        :cx="s.cx"
                        :cy="s.cy"
                        :r="STUB_R"
                        :fill="secColorStr(s.security)"
                        stroke="rgba(0,0,0,0.6)"
                        stroke-width="0.6"
                        class="hover:brightness-125 transition-[filter]"
                    />
                    <text
                        :x="s.cx"
                        :y="s.cy + STUB_R + 7"
                        text-anchor="middle"
                        fill="rgba(255,255,255,0.96)"
                        stroke="rgba(0,0,0,.95)"
                        stroke-width="1.6"
                        paint-order="stroke"
                        font-size="8.5"
                        font-weight="700"
                        class="map-label select-none pointer-events-none"
                    >
                        {{ s.name ?? '?' }}
                    </text>
                    <text
                        :x="s.cx"
                        :y="s.cy + STUB_R + 14"
                        text-anchor="middle"
                        stroke="rgba(0,0,0,.95)"
                        stroke-width="1.2"
                        paint-order="stroke"
                        font-size="7"
                        class="map-label select-none pointer-events-none"
                    >
                        <tspan fill="#9ca3af">{{ s.region_name ?? '' }}</tspan>
                        <tspan dx="3" :fill="secColorStr(s.security)">{{ secLabel(s.security) }}</tspan>
                    </text>
                    </g>
                </g>

                <!-- System nodes — click navigates unless a pan drag was in progress
                     (suppressed by onSvgClickCapture on the svg). -->
                <g v-show="showSystems">
                    <g
                        v-for="n in layout.nodes"
                        :key="n.id"
                        class="cursor-pointer"
                        @pointerenter="(e) => onNodeHoverEnter(e, n)"
                        @pointermove="onNodeHoverMove"
                        @pointerleave="onNodeHoverLeave"
                        @click="navigateTo(`/system/${n.id}`)"
                    >
                    <!-- Mini-system: blended halo (const + sec + activity) behind the
                         sec-tinted disc; planets + sun on top. -->
                    <circle
                        :cx="n.x"
                        :cy="n.y"
                        :r="systemHaloRadius(n)"
                        :fill="systemHaloFill(n)"
                        :fill-opacity="activityLayer === 'none' ? 0.55 : 0.9"
                        class="pointer-events-none"
                    />
                    <circle
                        :cx="n.x"
                        :cy="n.y"
                        :r="R"
                        :fill="secColorStr(n.security)"
                        fill-opacity="0.18"
                        :stroke="secColorStr(n.security)"
                        stroke-opacity="0.6"
                        stroke-width="0.6"
                        class="hover:brightness-125 transition-[filter]"
                    />
                    <circle
                        v-for="(p, i) in (miniSystemBySystem.get(n.id)?.planets ?? [])"
                        :key="'p-' + n.id + '-' + i"
                        :cx="n.x + p.dx"
                        :cy="n.y + p.dy"
                        :r="PLANET_R"
                        fill="#cbd5e1"
                        fill-opacity="0.85"
                        class="pointer-events-none"
                    />
                    <circle
                        v-if="miniSystemBySystem.get(n.id)?.hasSun"
                        :cx="n.x"
                        :cy="n.y"
                        :r="SUN_R"
                        fill="#fbbf24"
                        class="pointer-events-none"
                    />
                    <text
                        :x="n.x"
                        :y="n.y + R + 9"
                        text-anchor="middle"
                        fill="rgba(255,255,255,0.98)"
                        stroke="rgba(0,0,0,.98)"
                        stroke-width="2"
                        paint-order="stroke"
                        font-size="10.5"
                        font-weight="700"
                        class="map-label select-none pointer-events-none"
                        :class="pochvenClass(id)"
                    >
                        {{ n.system_name }}
                    </text>
                    <text
                        :x="n.x"
                        :y="n.y + R + 19"
                        text-anchor="middle"
                        :fill="secColorStr(n.security)"
                        stroke="rgba(0,0,0,.98)"
                        stroke-width="1.5"
                        paint-order="stroke"
                        font-size="8.5"
                        font-weight="700"
                        class="map-label select-none pointer-events-none"
                    >
                        {{ secLabel(n.security) }}
                    </text>
                    </g>
                </g>

            </svg>

            <div class="pointer-events-none absolute bottom-3 left-3 z-10 flex max-w-[calc(100%-1rem)] flex-wrap items-center gap-x-3 gap-y-1 rounded-lg border border-white/[0.08] bg-black/65 px-3 py-2 text-[10px] text-gray-400 shadow-lg backdrop-blur-md">
                <span class="flex items-center gap-1.5">
                    <span v-if="baseLayer === 'geography'" class="h-2.5 w-4 rounded-sm bg-gradient-to-r from-blue-500/50 via-purple-500/50 to-amber-500/50" />
                    <span v-else class="h-2 w-16 rounded-full" style="background: linear-gradient(90deg, #22d3ee, #a3e635, #fbbf24, #b91c1c)" />
                    {{ baseLayer === 'geography' ? 'Constellation grouping' : 'System security' }}
                </span>
                <template v-if="activityLayer !== 'none'">
                    <span class="h-2 w-24 rounded-full" style="background: linear-gradient(90deg, #475569, #3b82f6, #a855f7, #f97316, #ef4444)" />
                    <span class="font-semibold text-gray-300">{{ activeLayerLabel }}</span>
                    <span>Quiet</span><span>Peak {{ maximumActivityLabel }}</span>
                    <span class="text-gray-600">last {{ activityWindowLabel }} · log scale</span>
                </template>
                <span v-if="showConnections && externalStubs.length" class="flex items-center gap-1.5"><span class="w-4 border-t border-dashed border-blue-300/70" />Out-of-region gate</span>
            </div>

            <!-- Hover card -->
            <div
                v-if="hoverNode"
                :style="hoverCardStyle"
                class="absolute z-10 w-[260px] pointer-events-none rounded-lg border border-white/[0.12] bg-[#0f1015]/95 backdrop-blur-sm shadow-xl p-3 text-xs"
            >
                <div class="flex items-baseline justify-between gap-2 mb-1">
                    <span class="text-white font-semibold text-sm truncate" :class="pochvenClass(id)">{{ hoverNode.system_name }}</span>
                    <span :style="{ color: secColorStr(hoverNode.security) }" class="font-mono font-semibold">{{ secLabel(hoverNode.security) }}</span>
                </div>
                <div class="text-gray-400 truncate mb-2">
                    {{ constNameById.get(hoverNode.constellation_id) ?? '—' }}
                    <span class="text-gray-600">·</span>
                    {{ region?.name ?? '' }}
                </div>

                <div class="grid grid-cols-2 gap-1.5 mb-2">
                    <div class="rounded bg-white/[0.04] px-2 py-1 flex justify-between">
                        <span class="text-gray-400">Ship kills</span>
                        <span class="font-mono text-white">{{ (activityById.get(hoverNode.id)?.ship_kills ?? 0).toLocaleString() }}</span>
                    </div>
                    <div class="rounded bg-white/[0.04] px-2 py-1 flex justify-between">
                        <span class="text-gray-400">Pod kills</span>
                        <span class="font-mono text-white">{{ (activityById.get(hoverNode.id)?.pod_kills ?? 0).toLocaleString() }}</span>
                    </div>
                    <div class="rounded bg-white/[0.04] px-2 py-1 flex justify-between">
                        <span class="text-gray-400">NPC kills</span>
                        <span class="font-mono text-white">{{ (activityById.get(hoverNode.id)?.npc_kills ?? 0).toLocaleString() }}</span>
                    </div>
                    <div class="rounded bg-white/[0.04] px-2 py-1 flex justify-between">
                        <span class="text-gray-400">Jumps</span>
                        <span class="font-mono text-white">{{ (activityById.get(hoverNode.id)?.ship_jumps ?? 0).toLocaleString() }}</span>
                    </div>
                </div>

                <div class="mb-2">
                    <div class="flex items-center justify-between text-gray-500 mb-1">
                        <span>
                            {{ (internalNeighborsById.get(hoverNode.id)?.length ?? 0) + (externalNeighborsById.get(hoverNode.id)?.length ?? 0) }}
                            gate{{ ((internalNeighborsById.get(hoverNode.id)?.length ?? 0) + (externalNeighborsById.get(hoverNode.id)?.length ?? 0)) === 1 ? '' : 's' }}
                        </span>
                        <span v-if="externalNeighborsById.get(hoverNode.id)?.length" class="text-blue-300">
                            {{ externalNeighborsById.get(hoverNode.id)!.length }} out-of-region
                        </span>
                    </div>
                    <div class="flex flex-wrap gap-1">
                        <span
                            v-for="nb in internalNeighborsById.get(hoverNode.id) ?? []"
                            :key="'in-' + nb.system_id"
                            class="px-1.5 py-0.5 rounded bg-white/[0.04] border border-white/[0.08] text-gray-300"
                        >
                            {{ nb.system_name }}
                            <span :style="{ color: secColorStr(nb.security) }" class="font-mono ml-1">{{ secLabel(nb.security) }}</span>
                        </span>
                        <span
                            v-for="ex in externalNeighborsById.get(hoverNode.id) ?? []"
                            :key="'ex-' + ex.system_id"
                            class="px-1.5 py-0.5 rounded bg-blue-500/15 border border-blue-400/30 text-blue-200"
                        >
                            → {{ ex.system_name ?? '?' }}
                            <span v-if="ex.region_name" class="text-blue-300/70 ml-1">{{ ex.region_name }}</span>
                        </span>
                    </div>
                </div>

                <div class="flex items-center justify-end text-gray-500">
                    <span class="flex gap-1">
                        <span v-if="systemRawById.get(hoverNode.id)?.hub" class="px-1.5 py-0.5 rounded bg-amber-500/15 text-amber-300 border border-amber-400/20">hub</span>
                        <span v-if="systemRawById.get(hoverNode.id)?.border" class="px-1.5 py-0.5 rounded bg-red-500/15 text-red-300 border border-red-400/20">border</span>
                        <span v-if="systemRawById.get(hoverNode.id)?.international" class="px-1.5 py-0.5 rounded bg-blue-500/15 text-blue-300 border border-blue-400/20">int'l</span>
                    </span>
                </div>
                <div class="text-[10px] text-gray-600 mt-1">last {{ activityWindowLabel }} · ESI</div>
            </div>
        </div>
    </div>
</template>

<style scoped>
.map-labels-hidden .map-label {
    display: none;
}
</style>
