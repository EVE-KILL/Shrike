<script setup lang="ts">
import { Delaunay } from 'd3-delaunay'
import { secColorStr } from '~/utils/map/colors'

const props = defineProps<{
    type: string
}>()

const { data, pending, error } = await useApiFetch<any>(() => `/api/map/scope?type=${props.type}`, { watch: [() => props.type] })

if (error.value) {
    throw createError({
        statusCode: error.value.statusCode || 500,
        statusMessage: (error.value.data as any)?.message || 'Failed to load scope',
    })
}

const R = 1.6

function regionColor(regionId: number): string {
    const hue = ((regionId % 1000) * 137.508) % 360
    return `hsl(${hue}, 65%, 55%)`
}
function hslToRgb(h: number, s: number, l: number): [number, number, number] {
    s /= 100; l /= 100
    const k = (n: number) => (n + h / 30) % 12
    const a = s * Math.min(l, 1 - l)
    const f = (n: number) => l - a * Math.max(-1, Math.min(k(n) - 3, 9 - k(n), 1))
    return [Math.round(255 * f(0)), Math.round(255 * f(8)), Math.round(255 * f(4))]
}
function regionRgb(regionId: number): [number, number, number] {
    const hue = ((regionId % 1000) * 137.508) % 360
    return hslToRgb(hue, 65, 55)
}
function hexToRgb(hex: string): [number, number, number] {
    const h = hex.replace('#', '')
    return [parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16)]
}
function mixRgb(a: [number, number, number], b: [number, number, number], t: number): [number, number, number] {
    return [
        Math.round(a[0] + (b[0] - a[0]) * t),
        Math.round(a[1] + (b[1] - a[1]) * t),
        Math.round(a[2] + (b[2] - a[2]) * t),
    ]
}

interface LaidNode {
    id: number
    x: number
    y: number
    region_id: number
    region_name: string
    system_name: string
    security: number
}
interface LaidLink { from: number; to: number }
interface VoronoiCell { path: string; region_id: number }
interface Bbox { x: number; y: number; w: number; h: number }

const layout = computed<{ nodes: LaidNode[]; links: LaidLink[]; cells: VoronoiCell[]; bbox: Bbox }>(() => {
    const empty: Bbox = { x: 0, y: 0, w: 1000, h: 800 }
    if (!data.value) return { nodes: [], links: [], cells: [], bbox: empty }
    const { systems, jumps, regions: regionRows } = data.value
    if (!systems?.length) return { nodes: [], links: [], cells: [], bbox: empty }

    const regionNameById = new Map<number, string>()
    for (const r of regionRows ?? []) regionNameById.set(r.region_id, r.name ?? '')

    // Universe scale: use true 3D galactic coords (x, z plane = top-down).
    const getX = (s: any) => s.x
    const getZ = (s: any) => s.z

    let minX = Infinity, maxX = -Infinity, minZ = Infinity, maxZ = -Infinity
    for (const s of systems) {
        const sx = getX(s), sz = getZ(s)
        if (sx == null || sz == null) continue
        if (sx < minX) minX = sx
        if (sx > maxX) maxX = sx
        if (sz < minZ) minZ = sz
        if (sz > maxZ) maxZ = sz
    }
    const cx = (minX + maxX) / 2
    const cz = (minZ + maxZ) / 2
    const targetW = 1600
    const targetH = 1200
    const scale = Math.min(targetW / ((maxX - minX) || 1), targetH / ((maxZ - minZ) || 1))

    const nodes: LaidNode[] = systems.map((s: any) => ({
        id: s.solar_system_id,
        x: (getX(s) - cx) * scale,
        y: -(getZ(s) - cz) * scale,
        region_id: s.region_id,
        region_name: regionNameById.get(s.region_id) ?? '',
        system_name: s.system_name,
        security: s.security,
    }))

    const idSet = new Set<number>(nodes.map(n => n.id))
    const links: LaidLink[] = (jumps ?? [])
        .filter((j: any) => idSet.has(j.from_solar_system_id) && idSet.has(j.to_solar_system_id))
        .map((j: any) => ({ from: j.from_solar_system_id, to: j.to_solar_system_id }))

    let nMinX = Infinity, nMaxX = -Infinity, nMinY = Infinity, nMaxY = -Infinity
    for (const n of nodes) {
        if (n.x < nMinX) nMinX = n.x
        if (n.x > nMaxX) nMaxX = n.x
        if (n.y < nMinY) nMinY = n.y
        if (n.y > nMaxY) nMaxY = n.y
    }
    const pad = 80
    const vbX = nMinX - pad
    const vbY = nMinY - pad
    const vbW = (nMaxX - nMinX) + pad * 2
    const vbH = (nMaxY - nMinY) + pad * 2

    const delaunay = Delaunay.from(nodes, (n) => n.x, (n) => n.y)
    const voronoi = delaunay.voronoi([vbX, vbY, vbX + vbW, vbY + vbH])
    const cells: VoronoiCell[] = []
    for (let i = 0; i < nodes.length; i++) {
        const path = voronoi.renderCell(i)
        if (!path) continue
        cells.push({ path, region_id: nodes[i]!.region_id })
    }

    return { nodes, links, cells, bbox: { x: vbX, y: vbY, w: vbW, h: vbH } }
})

const viewBoxStr = computed(() => `${layout.value.bbox.x} ${layout.value.bbox.y} ${layout.value.bbox.w} ${layout.value.bbox.h}`)

const nodeById = computed<Map<number, LaidNode>>(() => {
    const m = new Map<number, LaidNode>()
    for (const n of layout.value.nodes) m.set(n.id, n)
    return m
})

interface SystemActivity { ship_kills: number; pod_kills: number; npc_kills: number; ship_jumps: number }
const activityById = computed<Map<number, SystemActivity>>(() => {
    const m = new Map<number, SystemActivity>()
    for (const r of (data.value?.activity ?? []) as any[]) {
        m.set(r.system_id, {
            ship_kills: r.ship_kills ?? 0,
            pod_kills: r.pod_kills ?? 0,
            npc_kills: r.npc_kills ?? 0,
            ship_jumps: r.ship_jumps ?? 0,
        })
    }
    return m
})

function linkColor(link: LaidLink): string {
    const src = nodeById.value.get(link.from)
    return src ? regionColor(src.region_id) : '#64748b'
}

// Region label anchors (centroid of each region's systems).
interface RegionLabel { region_id: number; name: string; x: number; y: number }
const regionLabels = computed<RegionLabel[]>(() => {
    const groups = new Map<number, { sx: number; sy: number; n: number; name: string }>()
    for (const node of layout.value.nodes) {
        const g = groups.get(node.region_id)
        if (g) { g.sx += node.x; g.sy += node.y; g.n += 1 }
        else groups.set(node.region_id, { sx: node.x, sy: node.y, n: 1, name: node.region_name })
    }
    const out: RegionLabel[] = []
    for (const [region_id, g] of groups) out.push({ region_id, name: g.name, x: g.sx / g.n, y: g.sy / g.n })
    return out
})

// Precomputed dot fill per node — avoids recomputing on every render.
const dotFillById = computed<Map<number, string>>(() => {
    const m = new Map<number, string>()
    const acts = activityById.value
    for (const n of layout.value.nodes) {
        const cRgb = regionRgb(n.region_id)
        const sRgb = hexToRgb(secColorStr(n.security))
        const act = acts.get(n.id)
        const kills = (act?.ship_kills ?? 0) + (act?.pod_kills ?? 0)
        const intensity = Math.min(1, kills / 20)
        const mixed = mixRgb(cRgb, sRgb, intensity * 0.4)
        const brightness = 0.85 + intensity * 0.6
        const r = Math.min(255, Math.round(mixed[0] * brightness))
        const g = Math.min(255, Math.round(mixed[1] * brightness))
        const b = Math.min(255, Math.round(mixed[2] * brightness))
        m.set(n.id, `rgb(${r}, ${g}, ${b})`)
    }
    return m
})
function systemDotFill(n: LaidNode): string { return dotFillById.value.get(n.id) ?? '#888' }

// ─── Region spotlight ────────────────────────────────────
const hoveredRegionId = ref<number | null>(null)
function enterRegion(id: number) { hoveredRegionId.value = id }
function clearRegion() { hoveredRegionId.value = null }

function onContentOver(e: PointerEvent) {
    const rid = (e.target as Element | null)?.getAttribute?.('data-rid')
    if (rid) enterRegion(+rid)
}

// Click anywhere on the map → if a region is currently spotlighted, navigate to
// its region map. Covered by the full-map click handler so dots, cells, or
// empty space within a region's cell area all work.
function onContentClick() {
    const rid = hoveredRegionId.value
    if (rid != null) navigateTo(`/map/region/${rid}`)
}

const hoveredCells = computed(() => {
    const rid = hoveredRegionId.value
    if (rid == null) return []
    return layout.value.cells.filter(c => c.region_id === rid)
})
const hoveredNodes = computed(() => {
    const rid = hoveredRegionId.value
    if (rid == null) return []
    return layout.value.nodes.filter(n => n.region_id === rid)
})
const hoveredJumps = computed(() => {
    const rid = hoveredRegionId.value
    if (rid == null) return []
    const nm = nodeById.value
    return layout.value.links.filter(l => nm.get(l.from)?.region_id === rid || nm.get(l.to)?.region_id === rid)
})
const hoveredLabel = computed(() => {
    const rid = hoveredRegionId.value
    if (rid == null) return null
    return regionLabels.value.find(rl => rl.region_id === rid) ?? null
})
</script>

<template>
    <div class="relative rounded-lg overflow-hidden border border-white/[0.08] bg-[#0a0a0f]">
        <div class="absolute top-2 right-2 z-10 text-xs text-gray-400 bg-black/40 backdrop-blur-sm rounded px-2 py-1 pointer-events-none">
            <span v-if="data">{{ data.systems?.length?.toLocaleString() ?? 0 }} systems · {{ data.regions?.length ?? 0 }} regions · hover a region to light it up</span>
        </div>

        <div v-if="pending" class="h-[80vh] flex items-center justify-center">
            <div class="text-gray-400 text-sm animate-pulse">Loading map data...</div>
        </div>

        <svg
            v-else-if="layout.nodes.length > 0"
            :viewBox="viewBoxStr"
            class="w-full h-[85vh] select-none"
            :class="hoveredRegionId != null ? 'cursor-pointer' : ''"
            preserveAspectRatio="xMidYMid meet"
            @pointerover="onContentOver"
            @pointerleave="clearRegion"
            @click="onContentClick"
        >
            <!-- BASE LAYER: entire map, muted. Drops further when a region is
                 spotlighted so the overlay can stand out. -->
            <g class="map-base" :style="{ opacity: hoveredRegionId != null ? 0.2 : 0.72 }">
                <path
                    v-for="(cell, i) in layout.cells"
                    :key="'c-' + i"
                    :d="cell.path"
                    :fill="regionColor(cell.region_id)"
                    fill-opacity="0.18"
                    :data-rid="cell.region_id"
                />
                <line
                    v-for="(link, i) in layout.links"
                    :key="'l-' + i"
                    :x1="nodeById.get(link.from)!.x"
                    :y1="nodeById.get(link.from)!.y"
                    :x2="nodeById.get(link.to)!.x"
                    :y2="nodeById.get(link.to)!.y"
                    :stroke="linkColor(link)"
                    stroke-opacity="0.25"
                    stroke-width="0.6"
                    class="pointer-events-none"
                />
                <circle
                    v-for="n in layout.nodes"
                    :key="n.id"
                    :cx="n.x"
                    :cy="n.y"
                    :r="R"
                    :fill="systemDotFill(n)"
                    :data-rid="n.region_id"
                    shape-rendering="optimizeSpeed"
                />
            </g>

            <!-- OVERLAY: spotlighted region, fades in/out. -->
            <Transition name="map-overlay">
                <g v-if="hoveredRegionId != null" class="pointer-events-none map-overlay">
                    <path
                        v-for="(cell, i) in hoveredCells"
                        :key="'hc-' + i"
                        :d="cell.path"
                        :fill="regionColor(cell.region_id)"
                        fill-opacity="0.38"
                    />
                    <line
                        v-for="(link, i) in hoveredJumps"
                        :key="'hl-' + i"
                        :x1="nodeById.get(link.from)!.x"
                        :y1="nodeById.get(link.from)!.y"
                        :x2="nodeById.get(link.to)!.x"
                        :y2="nodeById.get(link.to)!.y"
                        :stroke="linkColor(link)"
                        stroke-opacity="0.6"
                        stroke-width="0.8"
                    />
                    <circle
                        v-for="n in hoveredNodes"
                        :key="'hn-' + n.id"
                        :cx="n.x"
                        :cy="n.y"
                        :r="R"
                        :fill="systemDotFill(n)"
                        shape-rendering="optimizeSpeed"
                    />
                    <text
                        v-if="hoveredLabel"
                        :x="hoveredLabel.x"
                        :y="hoveredLabel.y"
                        text-anchor="middle"
                        dominant-baseline="middle"
                        font-size="26"
                        font-weight="700"
                        letter-spacing="4"
                        fill="rgba(255,255,255,0.88)"
                        stroke="rgba(0,0,0,0.6)"
                        stroke-width="0.6"
                        paint-order="stroke"
                        class="select-none"
                    >
                        {{ hoveredLabel.name.toUpperCase() }}
                    </text>
                </g>
            </Transition>
        </svg>
    </div>
</template>

<style scoped>
.map-base { transition: opacity 220ms ease-out; }
.map-overlay-enter-active,
.map-overlay-leave-active { transition: opacity 220ms ease-out; }
.map-overlay-enter-from,
.map-overlay-leave-to { opacity: 0; }
</style>
