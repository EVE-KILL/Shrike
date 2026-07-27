<script setup lang="ts">
import { Delaunay } from 'd3-delaunay'
import { secColorStr, secLabel } from '~/utils/map/colors'

interface FwSystem {
    solar_system_id: number
    system_name: string
    x: number
    y: number
    security: number
    region_id: number
    region_name: string
    constellation_name: string
    owner_faction_id: number
    occupier_faction_id: number
    contested: string
    victory_points: number
    victory_points_threshold: number
    kills_24h: number
    isk_24h: number
}

interface FwCelestial {
    system_id: number
    group_id: number
    x: number
    z: number
}

const props = defineProps<{
    systems: FwSystem[]
    jumps: number[][]
    celestials?: FwCelestial[]
    side1Id: number
    side2Id: number
    side1Name: string
    side2Name: string
}>()

const FACTION_COLORS: Record<number, string> = {
    500001: '#3b82f6', // Caldari - blue
    500002: '#ef4444', // Minmatar - red
    500003: '#f59e0b', // Amarr - amber
    500004: '#22c55e', // Gallente - green
}
const NEUTRAL_COLOR = '#64748b'
const FRONTLINE_COLOR = '#f87171'

// Cosmetic layout nudges — some warzone regions sit far enough from the
// frontline cluster that the full-extent fit leaves most of the map empty.
// Offsets are applied in raw CCP world-space (x2d/z2d meters) before the
// bbox is measured, so the cluster tightens without touching stored data.
// Sign: positive z shifts the region UP on screen (we negate z at render).
const PET = 1e15 // 1 petameter in raw CCP units
const REGION_Z_OFFSET: Record<number, number> = {
    // Caldari vs Gallente: pull the two western Gallente regions north toward
    // Placid/Black Rise/The Citadel.
    10000068: 85 * PET, // Verge Vendor
    10000064: 65 * PET, // Essence
    // Amarr vs Minmatar: pull Metropolis south toward Heimatar/Bleak Lands.
    10000042: -80 * PET, // Metropolis
}
const REGION_X_OFFSET: Record<number, number> = {
    // Caldari vs Gallente: pull The Citadel west toward Black Rise.
    10000033: -70 * PET, // The Citadel
    10000042: -30 * PET, // Metropolis
}

// Node disc radius in viewBox units. Systems are laid out in the normalized
// target canvas, so R scales with the bbox rather than the screen — zooming
// the SVG makes dots and labels grow together, which is what we want.
const R = 5
const RING_EXTRA = 1.5
const SUN_R = 0.75
const PLANET_R = 0.4
const PLANET_FIT = R - 1 // outer planet sits 1u inside the disc edge

interface LaidNode {
    id: number
    x: number
    y: number
    system_name: string
    security: number
    region_name: string
    constellation_name: string
    occupier_faction_id: number
    contested: string
    victory_points: number
    victory_points_threshold: number
    kills_24h: number
    isk_24h: number
}

interface LaidLink { from: number; to: number }
interface VoronoiCell { path: string; occupier_faction_id: number }
interface Bbox { x: number; y: number; w: number; h: number }

const emptyBbox: Bbox = { x: 0, y: 0, w: 1000, h: 800 }

const layout = computed<{ nodes: LaidNode[]; links: LaidLink[]; cells: VoronoiCell[]; bbox: Bbox }>(() => {
    if (!props.systems?.length) return { nodes: [], links: [], cells: [], bbox: emptyBbox }

    // Pre-bake world-space coords with region nudges so bbox/scale see the
    // tightened cluster rather than the full astronomical spread.
    const nudged = props.systems.map(s => {
        const xOff = REGION_X_OFFSET[s.region_id] ?? 0
        const zOff = REGION_Z_OFFSET[s.region_id] ?? 0
        return { ref: s, wx: s.x + xOff, wz: s.y + zOff }
    })

    let minX = Infinity, maxX = -Infinity, minZ = Infinity, maxZ = -Infinity
    for (const n of nudged) {
        if (n.wx < minX) minX = n.wx
        if (n.wx > maxX) maxX = n.wx
        if (n.wz < minZ) minZ = n.wz
        if (n.wz > maxZ) maxZ = n.wz
    }
    const targetW = 1400
    const targetH = 900
    const rangeX = (maxX - minX) || 1
    const rangeZ = (maxZ - minZ) || 1
    const scale = Math.min(targetW / rangeX, targetH / rangeZ)
    const cx = (minX + maxX) / 2
    const cz = (minZ + maxZ) / 2

    const nodes: LaidNode[] = nudged.map(({ ref: s, wx, wz }) => ({
        id: s.solar_system_id,
        x: (wx - cx) * scale,
        // Negate z so north points up, matching /map/region/:id orientation.
        y: -(wz - cz) * scale,
        system_name: s.system_name,
        security: s.security,
        region_name: s.region_name,
        constellation_name: s.constellation_name,
        occupier_faction_id: s.occupier_faction_id,
        contested: s.contested,
        victory_points: s.victory_points,
        victory_points_threshold: s.victory_points_threshold,
        kills_24h: s.kills_24h,
        isk_24h: s.isk_24h,
    }))

    const ids = new Set<number>(nodes.map(n => n.id))
    const links: LaidLink[] = (props.jumps ?? [])
        .filter(([f, t]) => ids.has(f) && ids.has(t))
        .map(([from, to]) => ({ from, to }))

    const pad = 120
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
    const vbH = (nMaxY - nMinY) + pad * 2

    // Voronoi cells — each system gets a polygon tinted by its occupier
    // faction, clipped to the bbox so hull cells don't extend to infinity.
    const delaunay = Delaunay.from(nodes, n => n.x, n => n.y)
    const voronoi = delaunay.voronoi([vbX, vbY, vbX + vbW, vbY + vbH])
    const cells: VoronoiCell[] = []
    for (let i = 0; i < nodes.length; i++) {
        const path = voronoi.renderCell(i)
        if (!path) continue
        cells.push({ path, occupier_faction_id: nodes[i]!.occupier_faction_id })
    }

    return { nodes, links, cells, bbox: { x: vbX, y: vbY, w: vbW, h: vbH } }
})

const nodeById = computed<Map<number, LaidNode>>(() => {
    const m = new Map<number, LaidNode>()
    for (const n of layout.value.nodes) m.set(n.id, n)
    return m
})

function factionColor(id: number): string {
    return FACTION_COLORS[id] || NEUTRAL_COLOR
}

// ─── System halo color blend ─────────────────────────────
// Halo behind each system blends three signals: occupier faction color
// (territory, dominant), security tint (so a nullsec pocket reads hotter),
// and 24h kill activity (drives opacity — busy systems glow brighter).
function hexToRgb(hex: string): [number, number, number] {
    const h = hex.replace('#', '')
    const v = h.length === 3 ? h.split('').map(c => c + c).join('') : h
    return [parseInt(v.slice(0, 2), 16), parseInt(v.slice(2, 4), 16), parseInt(v.slice(4, 6), 16)]
}
function mixRgb(a: [number, number, number], b: [number, number, number], t: number): [number, number, number] {
    return [
        Math.round(a[0] + (b[0] - a[0]) * t),
        Math.round(a[1] + (b[1] - a[1]) * t),
        Math.round(a[2] + (b[2] - a[2]) * t),
    ]
}
function systemHaloFill(n: LaidNode): string {
    const fRgb = hexToRgb(factionColor(n.occupier_faction_id))
    const sRgb = hexToRgb(secColorStr(n.security))
    const intensity = Math.min(1, n.kills_24h / 20)
    // Hue starts as faction color and skews toward sec tint with activity —
    // busy null systems run hotter against the faction backdrop.
    const skew = intensity * 0.35
    const mixed = mixRgb(fRgb, sRgb, skew)
    const brightness = 0.65 + intensity * 0.75
    const r = Math.min(255, Math.round(mixed[0] * brightness))
    const g = Math.min(255, Math.round(mixed[1] * brightness))
    const b = Math.min(255, Math.round(mixed[2] * brightness))
    const alpha = 0.5 + 0.4 * intensity
    return `rgba(${r}, ${g}, ${b}, ${alpha.toFixed(3)})`
}

// ─── Mini-system render ──────────────────────────────────
// Per-system sun + planet layout normalized to fit inside the node disc.
interface MiniSystem {
    hasSun: boolean
    planets: Array<{ dx: number; dy: number }>
}
const miniSystemBySystem = computed<Map<number, MiniSystem>>(() => {
    const m = new Map<number, MiniSystem>()
    const rows = props.celestials ?? []
    if (!rows.length) return m
    const bySys = new Map<number, FwCelestial[]>()
    for (const r of rows) {
        if (!bySys.has(r.system_id)) bySys.set(r.system_id, [])
        bySys.get(r.system_id)!.push(r)
    }
    for (const [sysId, list] of bySys) {
        const sun = list.find(r => r.group_id === 6)
        const planets = list.filter(r => r.group_id === 7)
        // Sun coords act as origin; if no sun, use planet centroid.
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
            // y negated so screen "up" matches CCP +z (consistent with layout).
            planets: planets.map(p => ({ dx: (p.x - cx) * scale, dy: -(p.z - cz) * scale })),
        })
    }
    return m
})

function linkStroke(l: LaidLink): string {
    const f = nodeById.value.get(l.from)
    const t = nodeById.value.get(l.to)
    if (!f || !t) return NEUTRAL_COLOR
    if (f.occupier_faction_id !== t.occupier_faction_id) return FRONTLINE_COLOR
    return factionColor(f.occupier_faction_id)
}

function linkOpacity(l: LaidLink): number {
    const f = nodeById.value.get(l.from)
    const t = nodeById.value.get(l.to)
    if (f && t && f.occupier_faction_id !== t.occupier_faction_id) return 0.6
    return 0.22
}

// ─── Pan / zoom ─────────────────────────────────────────────
const svgRef = ref<SVGSVGElement | null>(null)
const zoom = ref(1)
const panX = ref(0)
const panY = ref(0)

const viewBoxStr = computed(() => {
    const b = layout.value.bbox
    const w = b.w / zoom.value
    const h = b.h / zoom.value
    const x = b.x + (b.w - w) / 2 + panX.value
    const y = b.y + (b.h - h) / 2 + panY.value
    return `${x} ${y} ${w} ${h}`
})

watch(layout, () => {
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
    const after = svgPointFromEvent(e)
    panX.value += before.x - after.x
    panY.value += before.y - after.y
}

const panning = ref(false)
// panStart tracks raw screen pixels — reading CTM after a viewBox change
// reports a moved world and makes pans jerky.
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

// ─── Hover card ─────────────────────────────────────────────
const hoverNode = ref<LaidNode | null>(null)
const hoverPos = ref<{ x: number; y: number }>({ x: 0, y: 0 })
let hoverTimer: ReturnType<typeof setTimeout> | null = null
const HOVER_DELAY_MS = 180

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

const hoverCardStyle = computed(() => {
    const container = mapContainerRef.value
    if (!container || !hoverNode.value) return { display: 'none' }
    const rect = container.getBoundingClientRect()
    const localX = hoverPos.value.x - rect.left
    const localY = hoverPos.value.y - rect.top
    const cardW = 260
    const cardH = 220
    const offset = 14
    let left = localX + offset
    let top = localY + offset
    if (left + cardW > rect.width) left = localX - offset - cardW
    if (top + cardH > rect.height) top = localY - offset - cardH
    return { left: `${Math.max(4, left)}px`, top: `${Math.max(4, top)}px` }
})

// Show all labels only at higher zooms; otherwise reserve text for contested
// and active systems so the low-zoom view doesn't turn into a wall of names.
const showAllLabels = computed(() => zoom.value >= 1.8)

function shouldShowLabel(n: LaidNode): boolean {
    if (showAllLabels.value) return true
    return n.kills_24h > 0 || n.contested !== 'uncontested'
}

function contestedGlowFill(n: LaidNode): string | null {
    if (n.contested === 'vulnerable') return 'rgba(239, 68, 68, 0.35)'
    if (n.contested === 'contested') return 'rgba(250, 204, 21, 0.22)'
    return null
}

// Contested systems pop at full brightness, vulnerable lean in, uncontested
// fade to the Voronoi backdrop — so the eye lands on where the fighting is.
function nodeOpacity(n: LaidNode): number {
    if (n.contested === 'contested') return 1
    if (n.contested === 'vulnerable') return 0.8
    return 0.4
}
</script>

<template>
    <div class="relative">
        <!-- Legend + controls -->
        <div class="absolute top-2 right-2 z-10 flex flex-col gap-1 text-fine bg-black/60 rounded-md px-2 py-1.5">
            <div class="flex items-center gap-1.5">
                <span class="w-2.5 h-2.5 rounded-full" :style="{ background: FACTION_COLORS[side1Id] }"></span>
                <span class="text-gray-400">{{ side1Name }}</span>
            </div>
            <div class="flex items-center gap-1.5">
                <span class="w-2.5 h-2.5 rounded-full" :style="{ background: FACTION_COLORS[side2Id] }"></span>
                <span class="text-gray-400">{{ side2Name }}</span>
            </div>
            <div class="flex items-center gap-1.5 mt-0.5 pt-0.5 border-t border-white/10">
                <span class="w-2.5 h-2.5 rounded-full border border-yellow-400/40 bg-transparent"></span>
                <span class="text-gray-500">Contested</span>
            </div>
            <div class="flex items-center gap-1.5">
                <span class="w-2.5 h-2.5 rounded-full border border-red-400/40 bg-transparent"></span>
                <span class="text-gray-500">Vulnerable</span>
            </div>
            <button v-if="zoom !== 1 || panX !== 0 || panY !== 0"
                class="mt-1 pt-1 border-t border-white/10 text-gray-500 hover:text-blue-400 transition-colors text-left"
                @click="resetView">
                Reset view
            </button>
        </div>

        <!-- Zoom indicator -->
        <div v-if="zoom !== 1" class="absolute bottom-2 right-2 z-10 text-fine text-gray-600 bg-black/40 rounded px-1.5 py-0.5 tabular-nums">
            {{ Math.round(zoom * 100) }}%
        </div>

        <div ref="mapContainerRef" class="relative rounded-lg overflow-hidden border border-white/[0.08] bg-[#0a0a0f]">
            <svg
                v-if="layout.nodes.length > 0"
                ref="svgRef"
                :viewBox="viewBoxStr"
                class="w-full h-auto select-none"
                :class="panning ? 'cursor-grabbing' : 'cursor-grab'"
                style="height: 700px"
                preserveAspectRatio="xMidYMid meet"
                @wheel.prevent="onWheel"
                @pointerdown="onBgPointerDown"
                @click.capture="onSvgClickCapture"
            >
                <!-- Voronoi territory cells, blurred at the group level so
                     adjacent cells bleed into each other along shared edges. -->
                <defs>
                    <filter id="fwCellBlur" x="-10%" y="-10%" width="120%" height="120%">
                        <feGaussianBlur stdDeviation="6" />
                    </filter>
                </defs>
                <g filter="url(#fwCellBlur)" class="pointer-events-none">
                    <path
                        v-for="(cell, i) in layout.cells"
                        :key="'c-' + i"
                        :d="cell.path"
                        :fill="factionColor(cell.occupier_faction_id)"
                        fill-opacity="0.22"
                    />
                </g>

                <!-- Jump lines — frontline (cross-faction) edges glow red; inner
                     edges take the faction tint at low opacity. -->
                <line
                    v-for="(link, i) in layout.links"
                    :key="'l-' + i"
                    :x1="nodeById.get(link.from)!.x"
                    :y1="nodeById.get(link.from)!.y"
                    :x2="nodeById.get(link.to)!.x"
                    :y2="nodeById.get(link.to)!.y"
                    :stroke="linkStroke(link)"
                    :stroke-opacity="linkOpacity(link)"
                    stroke-width="1.25"
                    class="pointer-events-none"
                />

                <!-- System nodes — same layering as /map/region/:id:
                     contested outer ring → blended halo → sec-tinted disc
                     → planets → sun → name/sec labels. -->
                <g
                    v-for="n in layout.nodes"
                    :key="n.id"
                    :opacity="nodeOpacity(n)"
                    class="cursor-pointer"
                    @pointerenter="(e) => onNodeHoverEnter(e, n)"
                    @pointermove="onNodeHoverMove"
                    @pointerleave="onNodeHoverLeave"
                    @click="navigateTo(`/system/${n.id}`)"
                >
                    <circle
                        v-if="contestedGlowFill(n)"
                        :cx="n.x"
                        :cy="n.y"
                        :r="R + RING_EXTRA"
                        :fill="contestedGlowFill(n) ?? undefined"
                        class="pointer-events-none"
                    />
                    <circle
                        :cx="n.x"
                        :cy="n.y"
                        :r="R + 1"
                        :fill="systemHaloFill(n)"
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
                        stroke-width="0.3"
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
                        v-if="shouldShowLabel(n)"
                        :x="n.x"
                        :y="n.y + R + 4"
                        text-anchor="middle"
                        fill="rgba(255,255,255,0.85)"
                        font-size="5"
                        font-weight="600"
                        class="select-none pointer-events-none"
                    >
                        {{ n.system_name }}
                    </text>
                    <text
                        v-if="shouldShowLabel(n)"
                        :x="n.x"
                        :y="n.y + R + 10"
                        text-anchor="middle"
                        :fill="secColorStr(n.security)"
                        font-size="4"
                        class="select-none pointer-events-none"
                    >
                        {{ secLabel(n.security) }}
                    </text>
                </g>
            </svg>

            <div v-else class="h-[700px] flex items-center justify-center text-sm text-gray-500">
                No systems to display
            </div>

            <!-- Hover card -->
            <div
                v-if="hoverNode"
                :style="hoverCardStyle"
                class="absolute z-20 w-[260px] pointer-events-none rounded-lg border border-white/[0.12] bg-[#0f1015]/95 backdrop-blur-sm shadow-xl p-3 text-xs"
            >
                <div class="flex items-baseline justify-between gap-2 mb-1">
                    <span class="text-white font-semibold text-sm truncate">{{ hoverNode.system_name }}</span>
                    <span class="flex items-center gap-1.5">
                        <span :style="{ color: secColorStr(hoverNode.security) }" class="font-mono font-semibold">{{ secLabel(hoverNode.security) }}</span>
                        <span class="w-2 h-2 rounded-full" :style="{ background: factionColor(hoverNode.occupier_faction_id) }"></span>
                    </span>
                </div>
                <div class="text-gray-500 truncate mb-2">
                    {{ hoverNode.region_name }}
                    <span class="text-gray-600">·</span>
                    {{ hoverNode.constellation_name }}
                </div>
                <div class="flex items-center gap-2 mb-2">
                    <span :class="hoverNode.contested === 'contested' ? 'text-yellow-400' : hoverNode.contested === 'vulnerable' ? 'text-red-400' : 'text-gray-400'">
                        {{ hoverNode.contested }}
                    </span>
                </div>
                <div v-if="hoverNode.victory_points_threshold > 0" class="mb-2">
                    <div class="flex justify-between text-gray-500 mb-0.5">
                        <span>VP</span>
                        <span class="tabular-nums">{{ hoverNode.victory_points.toLocaleString() }} / {{ hoverNode.victory_points_threshold.toLocaleString() }}</span>
                    </div>
                    <div class="w-full h-1.5 bg-white/10 rounded-full overflow-hidden">
                        <div class="h-full rounded-full"
                            :class="hoverNode.contested === 'vulnerable' ? 'bg-red-500' : 'bg-amber-500'"
                            :style="{ width: `${Math.min(100, (hoverNode.victory_points / hoverNode.victory_points_threshold) * 100)}%` }"></div>
                    </div>
                </div>
                <div v-if="hoverNode.kills_24h > 0" class="text-gray-400">
                    <span class="text-white tabular-nums">{{ hoverNode.kills_24h }}</span> kills ·
                    <span class="text-isk tabular-nums">{{ formatIsk(hoverNode.isk_24h) }}</span> (24h)
                </div>
            </div>
        </div>
    </div>
</template>
