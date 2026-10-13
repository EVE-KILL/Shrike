// Exact geometry and color helpers extracted from fw/WarzoneMap.vue.
import { Delaunay } from 'd3-delaunay'
import { secColorStr } from './map/colors'

export interface FwSystem {
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

export interface FwCelestial {
    system_id: number
    group_id: number
    x: number
    z: number
}

export const FACTION_COLORS: Record<number, string> = {
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
export const R = 5
export const RING_EXTRA = 1.5
export const SUN_R = 0.75
export const PLANET_R = 0.4
const PLANET_FIT = R - 1 // outer planet sits 1u inside the disc edge

export interface LaidNode {
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

export interface LaidLink { from: number; to: number }
interface VoronoiCell { path: string; occupier_faction_id: number }
export interface Bbox { x: number; y: number; w: number; h: number }

const emptyBbox: Bbox = { x: 0, y: 0, w: 1000, h: 800 }

export function buildFwLayout(props: { systems: FwSystem[]; jumps: number[][] }) {
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
    const links: LaidLink[] = (props.jumps ?? []).flatMap((jump) => {
        const [from, to] = jump
        if (from === undefined || to === undefined) return []
        return ids.has(from) && ids.has(to) ? [{ from, to }] : []
    })

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
}

export function factionColor(id: number): string {
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
export function systemHaloFill(n: LaidNode): string {
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

interface MiniSystem { hasSun: boolean; planets: Array<{dx: number; dy: number}> }
export function buildFwMiniSystems(rows: FwCelestial[]) {
    const m = new Map<number, MiniSystem>()
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
}

export function contestedGlowFill(n: LaidNode): string | null {
    if (n.contested === 'vulnerable') return 'rgba(239, 68, 68, 0.35)'
    if (n.contested === 'contested') return 'rgba(250, 204, 21, 0.22)'
    return null
}

// Contested systems pop at full brightness, vulnerable lean in, uncontested
// fade to the Voronoi backdrop — so the eye lands on where the fighting is.
export function nodeOpacity(n: LaidNode): number {
    if (n.contested === 'contested') return 1
    if (n.contested === 'vulnerable') return 0.8
    return 0.4
}
