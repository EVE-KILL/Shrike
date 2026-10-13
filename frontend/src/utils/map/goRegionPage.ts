// Dedicated Go region map, ported from pages/map/region/[id].vue. The region
// uses CCP's 2D coordinates and a lightly relaxed SVG layout; it deliberately
// differs from the universe-wide map projection.
import { Delaunay } from 'd3-delaunay'
import { secColorStr } from './colors'
import { heatColor, mapActivityLayerLabel, mapActivityRatio, mapActivityValue } from './layers'
import type { MapActivity, MapActivityLayer } from './layers'

type System = { solar_system_id: number; system_name: string; constellation_id: number; security: number; x: number; z: number; x2d?: number | null; z2d?: number | null }
type Jump = { from_solar_system_id: number; to_solar_system_id: number }
type ExternalJump = { internal_system_id: number; external_system_id: number; external_system_name: string | null; external_region_id: number | null; external_region_name: string | null; external_security: number | null; external_x: number | null; external_z: number | null; external_x2d: number | null; external_z2d: number | null }
type Celestial = { system_id: number; group_id: number; x: number; z: number }
type Activity = { system_id: number } & MapActivity
type RegionData = { region: { region_id: number; name: string }; systems: System[]; jumps: Jump[]; externalJumps: ExternalJump[]; constellations: Array<{ constellation_id: number; constellation_name: string }>; celestials: Celestial[]; activity: Activity[] }
type Node = { id: number; x: number; y: number; constellation_id: number; system_name: string; security: number }
type Layout = { nodes: Node[]; byID: Map<number, Node>; links: Array<{ from: Node; to: Node }>; cells: Array<{ path: string; constellation_id: number; security: number }>; bbox: { x: number; y: number; w: number; h: number }; transform: { has2d: boolean; cx: number; cz: number; scale: number } }

const R = 16, STUB_R = 3.5, STUB_DIST = 90
const CONST_COLORS = ['#3b82f6','#22c55e','#eab308','#ef4444','#8b5cf6','#06b6d4','#f97316','#ec4899','#14b8a6','#a855f7','#64748b','#84cc16','#f43f5e','#0ea5e9','#d946ef']
const svgNS = 'http://www.w3.org/2000/svg'
const create = (tag: string, attributes: Record<string, string | number> = {}, label?: string) => {
    const element = document.createElementNS(svgNS, tag)
    for (const [key, value] of Object.entries(attributes)) element.setAttribute(key, String(value))
    if (label != null) element.textContent = label
    return element
}
const compact = (value: number) => value.toLocaleString('en-US')

function relaxSystemSpacing(nodes: Node[]) {
    const anchors = nodes.map(node => ({ x: node.x, y: node.y }))
    const minimumDistance = R * 4.25
    for (let iteration = 0; iteration < 64; iteration++) {
        for (let leftIndex = 0; leftIndex < nodes.length; leftIndex++) {
            const left = nodes[leftIndex]!
            for (let rightIndex = leftIndex + 1; rightIndex < nodes.length; rightIndex++) {
                const right = nodes[rightIndex]!
                let dx = right.x - left.x, dy = right.y - left.y, distance = Math.hypot(dx, dy)
                if (distance >= minimumDistance) continue
                if (distance < .001) {
                    const angle = ((left.id + right.id) % 360) * Math.PI / 180
                    dx = Math.cos(angle); dy = Math.sin(angle); distance = 1
                }
                const push = (minimumDistance - distance) * .24
                left.x -= dx / distance * push; left.y -= dy / distance * push
                right.x += dx / distance * push; right.y += dy / distance * push
            }
        }
        for (let index = 0; index < nodes.length; index++) {
            nodes[index]!.x += (anchors[index]!.x - nodes[index]!.x) * .015
            nodes[index]!.y += (anchors[index]!.y - nodes[index]!.y) * .015
        }
    }
}

function layoutRegion(data: RegionData): Layout {
    const systems = data.systems || []
    const empty = { nodes: [], byID: new Map(), links: [], cells: [], bbox: { x: 0, y: 0, w: 1000, h: 800 }, transform: { has2d: false, cx: 0, cz: 0, scale: 1 } } as Layout
    if (!systems.length) return empty
    const has2d = systems.some(system => system.x2d != null && system.z2d != null)
    const getX = (system: System) => has2d && system.x2d != null ? system.x2d : system.x
    const getZ = (system: System) => has2d && system.z2d != null ? system.z2d : system.z
    let minX = Infinity, maxX = -Infinity, minZ = Infinity, maxZ = -Infinity
    for (const system of systems) {
        minX = Math.min(minX, getX(system)); maxX = Math.max(maxX, getX(system))
        minZ = Math.min(minZ, getZ(system)); maxZ = Math.max(maxZ, getZ(system))
    }
    const scale = Math.min(1200 / ((maxX - minX) || 1), 900 / ((maxZ - minZ) || 1))
    const cx = (minX + maxX) / 2, cz = (minZ + maxZ) / 2
    const nodes = systems.map(system => ({ id: system.solar_system_id,
        x: (getX(system) - cx) * scale, y: -(getZ(system) - cz) * scale,
        constellation_id: system.constellation_id, system_name: system.system_name, security: system.security }))
    relaxSystemSpacing(nodes)
    const byID = new Map(nodes.map(node => [node.id, node]))
    const links = (data.jumps || []).flatMap(jump => {
        const from = byID.get(jump.from_solar_system_id), to = byID.get(jump.to_solar_system_id)
        return from && to ? [{ from, to }] : []
    })
    let minNodeX = Infinity, maxNodeX = -Infinity, minNodeY = Infinity, maxNodeY = -Infinity
    for (const node of nodes) {
        minNodeX = Math.min(minNodeX, node.x); maxNodeX = Math.max(maxNodeX, node.x)
        minNodeY = Math.min(minNodeY, node.y); maxNodeY = Math.max(maxNodeY, node.y)
    }
    const pad = 150
    const bbox = { x: minNodeX - pad, y: minNodeY - pad, w: maxNodeX - minNodeX + pad * 2, h: maxNodeY - minNodeY + pad * 2 + 24 }
    const voronoi = Delaunay.from(nodes, node => node.x, node => node.y).voronoi([bbox.x, bbox.y, bbox.x + bbox.w, bbox.y + bbox.h])
    const cells = nodes.flatMap((node, index) => {
        const path = voronoi.renderCell(index)
        return path ? [{ path, constellation_id: node.constellation_id, security: node.security }] : []
    })
    return { nodes, byID, links, cells, bbox, transform: { has2d, cx, cz, scale } }
}

function initialData(): RegionData {
    const element = document.getElementById('region-map-data')
    if (!element?.textContent) throw new Error('Region map data is missing')
    return JSON.parse(element.textContent) as RegionData
}

export function initializeRegionMap() {
    const root = document.querySelector<HTMLElement>('[data-region-map-root]')
    if (!root) return
    const svg = root.querySelector<SVGSVGElement>('[data-region-svg]')!
    const surface = root.querySelector<HTMLElement>('[data-region-map-surface]')!
    const loading = root.querySelector<HTMLElement>('[data-region-loading]')!
    const search = root.querySelector<HTMLInputElement>('[data-region-search]')!
    const searchResults = root.querySelector<HTMLElement>('[data-region-search-results]')!
    const legend = root.querySelector<HTMLElement>('[data-region-legend]')!
    const hoverCard = root.querySelector<HTMLElement>('[data-region-hover]')!
    const id = Number(root.dataset.regionId)
    let data = initialData(), layout = layoutRegion(data)
    let base = root.dataset.regionBase === 'security' ? 'security' : 'geography'
    let activity = (root.dataset.regionActivity || 'none') as MapActivityLayer
    let hours = Number(root.dataset.regionHours) || 24
    let routes = new URLSearchParams(location.search).get('routes') !== '0'
    let systems = new URLSearchParams(location.search).get('systems') !== '0'
    let labels = new URLSearchParams(location.search).get('labels') !== '0'
    let zoom = 1, panX = 0, panY = 0, drag: { x: number; y: number; moved: boolean } | null = null
    let suppressClick = false, controller: AbortController | null = null, hoverTimer = 0
    const constIDs = () => (data.constellations || []).map(item => item.constellation_id)
    const constColor = (constellationID: number) => CONST_COLORS[Math.max(0, constIDs().indexOf(constellationID)) % CONST_COLORS.length]!
    const dataActivity = () => new Map((data.activity || []).map(row => [row.system_id, row]))
    const activityMaximums = () => {
        const rows = data.activity || []
        return { ship_kills: Math.max(0, ...rows.map(row => row.ship_kills || 0)), pod_kills: Math.max(0, ...rows.map(row => row.pod_kills || 0)), npc_kills: Math.max(0, ...rows.map(row => row.npc_kills || 0)), ship_jumps: Math.max(0, ...rows.map(row => row.ship_jumps || 0)) }
    }
    const currentQuery = () => {
        const q = new URLSearchParams({ base, activity, hours: String(hours) })
        if (!routes) q.set('routes', '0')
        if (!systems) q.set('systems', '0')
        if (!labels) q.set('labels', '0')
        return q
    }
    const updateURL = () => {
        const q = currentQuery()
        history.replaceState(null, '', `/map/region/${id}?${q}`)
        root.querySelector<HTMLAnchorElement>('[data-region-back]')!.href = `/map?${q}`
    }
    const viewBox = () => {
        const box = layout.bbox, width = box.w / zoom, height = box.h / zoom
        svg.setAttribute('viewBox', `${box.x + (box.w - width) / 2 + panX} ${box.y + (box.h - height) / 2 + panY} ${width} ${height}`)
    }
    const resetView = () => { zoom = 1; panX = panY = 0; viewBox() }
    const sync = () => {
        for (const button of root.querySelectorAll<HTMLButtonElement>('[data-region-base-option]')) {
            const active = button.dataset.regionBaseOption === base
            button.classList.toggle('active', active); button.setAttribute('aria-pressed', String(active))
        }
        for (const button of root.querySelectorAll<HTMLButtonElement>('[data-region-activity-option]')) {
            const active = button.dataset.regionActivityOption === activity
            button.classList.toggle('active', active); button.setAttribute('aria-pressed', String(active))
        }
        for (const button of root.querySelectorAll<HTMLButtonElement>('[data-region-hours-option]')) {
            const active = Number(button.dataset.regionHoursOption) === hours
            button.classList.toggle('active', active); button.setAttribute('aria-pressed', String(active)); button.disabled = activity === 'none'
        }
        for (const button of root.querySelectorAll<HTMLButtonElement>('[data-region-toggle]')) {
            const active = { routes, systems, labels }[button.dataset.regionToggle as 'routes' | 'systems' | 'labels']
            button.classList.toggle('active', active); button.setAttribute('aria-pressed', String(active))
        }
        const note = root.querySelector<HTMLElement>('[data-region-activity-note]')!
        note.hidden = activity === 'none'; note.textContent = `Hourly data · ${hours === 168 ? '7 days' : `${hours} hour${hours === 1 ? '' : 's'}`}`
        svg.classList.toggle('region-map-labels-hidden', !labels)
        updateURL()
    }
    const appendText = (parent: SVGElement, x: number, y: number, text: string, attrs: Record<string, string | number> = {}) => {
        const element = create('text', { x, y, 'text-anchor': 'middle', 'paint-order': 'stroke', ...attrs }, text)
        parent.append(element); return element
    }
    const render = () => {
        layout = layoutRegion(data)
        const activityByID = dataActivity(), maximums = activityMaximums()
        const activityValues = layout.nodes.map(node => mapActivityValue(activity, activityByID.get(node.id), maximums))
        const maximumActivity = Math.max(0, ...activityValues)
        const constellationNames = new Map((data.constellations || []).map(item => [item.constellation_id, item.constellation_name]))
        const constellationGroups = new Map<number, { x: number; y: number; count: number }>()
        for (const node of layout.nodes) {
            const group = constellationGroups.get(node.constellation_id) || { x: 0, y: 0, count: 0 }
            group.x += node.x; group.y += node.y; group.count++
            constellationGroups.set(node.constellation_id, group)
        }
        svg.replaceChildren()
        const cells = create('g', { class: 'region-map-cell' })
        for (const cell of layout.cells) cells.append(create('path', { d: cell.path, fill: base === 'geography' ? constColor(cell.constellation_id) : secColorStr(cell.security), 'fill-opacity': base === 'geography' ? .12 : .07 }))
        svg.append(cells)
        if (routes) {
            const links = create('g', { class: 'region-map-link' })
            for (const link of layout.links) links.append(create('line', { x1: link.from.x, y1: link.from.y, x2: link.to.x, y2: link.to.y, stroke: base === 'geography' ? constColor(link.from.constellation_id) : '#64748b', 'stroke-opacity': base === 'geography' ? .35 : .22, 'stroke-width': 1.25 }))
            svg.append(links)
        }
        const labelGroup = create('g', { class: 'region-map-label' })
        for (const [constellationID, group] of constellationGroups) appendText(labelGroup, group.x / group.count, group.y / group.count, (constellationNames.get(constellationID) || '').toUpperCase(), { fill: 'rgba(255,255,255,.48)', stroke: 'rgba(0,0,0,.96)', 'stroke-width': 4, 'font-size': 20, 'font-weight': 800, 'letter-spacing': 2.5 })
        svg.append(labelGroup)
        const miniBySystem = new Map<number, { sun: boolean; planets: Array<{ dx: number; dy: number }> }>()
        const celestialGroups = new Map<number, Celestial[]>()
        for (const celestial of data.celestials || []) celestialGroups.set(celestial.system_id, [...(celestialGroups.get(celestial.system_id) || []), celestial])
        for (const [systemID, list] of celestialGroups) {
            const sun = list.find(item => item.group_id === 6), planets = list.filter(item => item.group_id === 7)
            const cx = sun ? sun.x : planets.length ? planets.reduce((sum, item) => sum + item.x, 0) / planets.length : 0
            const cz = sun ? sun.z : planets.length ? planets.reduce((sum, item) => sum + item.z, 0) / planets.length : 0
            const maxDistance = Math.max(0, ...planets.map(item => Math.hypot(item.x - cx, item.z - cz)))
            const scale = maxDistance > 0 ? (R - 2) / maxDistance : 0
            miniBySystem.set(systemID, { sun: !!sun, planets: planets.map(item => ({ dx: (item.x - cx) * scale, dy: -(item.z - cz) * scale })) })
        }
        if (routes) {
            const stubs = create('g')
            for (const jump of data.externalJumps || []) {
                const internal = layout.byID.get(jump.internal_system_id)
                if (!internal) continue
                const ex = layout.transform.has2d && jump.external_x2d != null ? jump.external_x2d : jump.external_x
                const ez = layout.transform.has2d && jump.external_z2d != null ? jump.external_z2d : jump.external_z
                let dx = 0, dy = -1
                if (ex != null && ez != null) {
                    dx = (ex - layout.transform.cx) * layout.transform.scale - internal.x
                    dy = -(ez - layout.transform.cz) * layout.transform.scale - internal.y
                    const length = Math.hypot(dx, dy)
                    if (length > 1e-6) { dx /= length; dy /= length } else { dx = 0; dy = -1 }
                }
                const cx = internal.x + dx * STUB_DIST, cy = internal.y + dy * STUB_DIST
                const link = create('a', { href: jump.external_region_id ? `/map/region/${jump.external_region_id}?${currentQuery()}` : `/system/${jump.external_system_id}`, class: 'region-map-stub', 'aria-label': `${jump.external_system_name || 'External system'} in ${jump.external_region_name || 'another region'}` })
                link.append(create('line', { x1: internal.x + dx * (R + 1), y1: internal.y + dy * (R + 1), x2: cx - dx * (STUB_R + 1), y2: cy - dy * (STUB_R + 1), stroke: '#93c5fd', 'stroke-opacity': .55, 'stroke-width': .9, 'stroke-dasharray': '2 2' }))
                link.append(create('circle', { cx, cy, r: STUB_R, fill: secColorStr(jump.external_security) }))
                appendText(link, cx, cy + STUB_R + 7, jump.external_system_name || '?', { class: 'region-map-label', fill: '#fff', stroke: '#000', 'stroke-width': 1.6, 'font-size': 8.5, 'font-weight': 700 })
                appendText(link, cx, cy + STUB_R + 14, `${jump.external_region_name || ''} ${jump.external_security?.toFixed(1) || ''}`, { class: 'region-map-label', fill: '#9ca3af', stroke: '#000', 'stroke-width': 1.2, 'font-size': 7 })
                stubs.append(link)
            }
            svg.append(stubs)
        }
        if (systems) {
            const systemGroup = create('g')
            for (let index = 0; index < layout.nodes.length; index++) {
                const node = layout.nodes[index]!, ratio = mapActivityRatio(activity, activityValues[index]!, maximumActivity)
                const link = create('a', { href: `/system/${node.id}`, class: 'region-map-system', 'aria-label': `${node.system_name} ${node.security.toFixed(1)}` })
                link.append(create('circle', { cx: node.x, cy: node.y, r: R + 2.5 + (activity === 'none' ? 0 : ratio * 10), fill: activity === 'none' ? base === 'geography' ? constColor(node.constellation_id) : secColorStr(node.security) : heatColor(ratio), 'fill-opacity': activity === 'none' ? .55 : .9 }))
                link.append(create('circle', { cx: node.x, cy: node.y, r: R, fill: secColorStr(node.security), 'fill-opacity': .18, stroke: secColorStr(node.security), 'stroke-opacity': .6, 'stroke-width': .6, class: 'region-map-disc' }))
                const mini = miniBySystem.get(node.id)
                for (const planet of mini?.planets || []) link.append(create('circle', { cx: node.x + planet.dx, cy: node.y + planet.dy, r: 1, fill: '#cbd5e1', 'fill-opacity': .85 }))
                if (mini?.sun) link.append(create('circle', { cx: node.x, cy: node.y, r: 2, fill: '#fbbf24' }))
                appendText(link, node.x, node.y + R + 9, node.system_name, { class: 'region-map-label', fill: '#fff', stroke: '#000', 'stroke-width': 2, 'font-size': 10.5, 'font-weight': 700 })
                appendText(link, node.x, node.y + R + 19, node.security.toFixed(1), { class: 'region-map-label', fill: secColorStr(node.security), stroke: '#000', 'stroke-width': 1.5, 'font-size': 8.5, 'font-weight': 700 })
                link.addEventListener('pointerenter', event => showHoverLater(node, event))
                link.addEventListener('pointermove', positionHover)
                link.addEventListener('pointerleave', hideHover)
                link.addEventListener('focus', () => {
                    const box = link.getBoundingClientRect()
                    showHoverLater(node, { clientX: box.left + box.width / 2, clientY: box.top + box.height / 2 })
                })
                link.addEventListener('blur', hideHover)
                systemGroup.append(link)
            }
            svg.append(systemGroup)
        }
        legend.replaceChildren()
        const swatch = document.createElement('i')
        swatch.className = base === 'security' ? 'security-swatch' : ''
        swatch.style.background = base === 'geography' ? 'linear-gradient(90deg,#3b82f6,#8b5cf6,#eab308)' : 'linear-gradient(90deg,#22d3ee,#a3e635,#fbbf24,#b91c1c)'
        legend.append(swatch, document.createTextNode(base === 'geography' ? 'Constellation grouping' : 'System security'))
        if (activity !== 'none') {
            const gradient = document.createElement('i'); gradient.className = 'activity-gradient'
            const label = document.createElement('b'); label.textContent = mapActivityLayerLabel(activity)
            const peak = activity === 'danger' ? `${maximumActivity.toFixed(1)} / 100 jumps` : activity === 'activity' ? `${Math.round(maximumActivity)} score` : compact(Math.round(maximumActivity))
            legend.append(gradient, label, document.createTextNode(`Quiet · Peak ${peak}`), document.createTextNode(`last ${hours === 168 ? '7 days' : `${hours} hour${hours === 1 ? '' : 's'}`} · log scale`))
        }
        if (routes && data.externalJumps?.length) {
            const gate = document.createElement('span'); gate.className = 'region-gate-key'
            const line = document.createElement('i'); gate.append(line, document.createTextNode('Out-of-region gate'))
            legend.append(gate)
        }
        resetView(); sync(); loading.hidden = true
    }
    const positionHover = (event: { clientX: number; clientY: number }) => {
        const rect = surface.getBoundingClientRect(), x = event.clientX - rect.left, y = event.clientY - rect.top
        const left = x + 274 > rect.width ? x - 274 : x + 14
        const top = y + 330 > rect.height ? y - 330 : y + 14
        hoverCard.style.left = `${Math.max(4, left)}px`; hoverCard.style.top = `${Math.max(4, top)}px`
    }
    const hideHover = () => { clearTimeout(hoverTimer); hoverCard.hidden = true }
    const showHoverLater = (node: Node, event: { clientX: number; clientY: number }) => {
        positionHover(event); clearTimeout(hoverTimer)
        hoverTimer = window.setTimeout(() => {
            hoverCard.replaceChildren()
            const title = document.createElement('div'); title.className = 'region-hover-title'
            const name = document.createElement('strong'); name.textContent = node.system_name
            const security = document.createElement('span'); security.textContent = node.security.toFixed(1); security.style.color = secColorStr(node.security)
            title.append(name, security)
            const constellation = data.constellations.find(item => item.constellation_id === node.constellation_id)?.constellation_name || ''
            const sub = document.createElement('small'); sub.textContent = `${constellation} · ${data.region.name}`
            hoverCard.append(title, sub)
            const row = data.activity.find(item => item.system_id === node.id)
            const metrics = document.createElement('section'); metrics.className = 'region-hover-metrics'
            for (const [label, value] of [['Ship kills', row?.ship_kills || 0], ['Pod kills', row?.pod_kills || 0], ['NPC kills', row?.npc_kills || 0], ['Jumps', row?.ship_jumps || 0]] as const) {
                const line = document.createElement('div'), item = document.createElement('span'), number = document.createElement('b')
                item.textContent = label; number.textContent = compact(value); line.append(item, number); metrics.append(line)
            }
            hoverCard.append(metrics)
            const internal = layout.links.flatMap(link => link.from.id === node.id ? [link.to] : link.to.id === node.id ? [link.from] : [])
            const external = (data.externalJumps || []).filter(jump => jump.internal_system_id === node.id)
            const gates = document.createElement('section'); gates.className = 'region-hover-gates'
            const heading = document.createElement('div'); heading.className = 'region-hover-gates-heading'
            heading.textContent = `${internal.length + external.length} gate${internal.length + external.length === 1 ? '' : 's'}`
            if (external.length) {
                const out = document.createElement('span'); out.textContent = `${external.length} out-of-region`; heading.append(out)
            }
            gates.append(heading)
            const neighbors = document.createElement('div'); neighbors.className = 'region-hover-neighbors'
            for (const neighbor of internal) {
                const pill = document.createElement('span'); pill.textContent = `${neighbor.system_name}  `
                const sec = document.createElement('b'); sec.textContent = neighbor.security.toFixed(1); sec.style.color = secColorStr(neighbor.security)
                pill.append(sec); neighbors.append(pill)
            }
            for (const neighbor of external) {
                const pill = document.createElement('span'); pill.className = 'external'
                pill.textContent = `→ ${neighbor.external_system_name || '?'}  ${neighbor.external_region_name || ''}`
                neighbors.append(pill)
            }
            gates.append(neighbors); hoverCard.append(gates)
            const raw = data.systems.find(system => system.solar_system_id === node.id) as (System & { hub?: boolean; border?: boolean; international?: boolean }) | undefined
            const flags = document.createElement('div'); flags.className = 'region-hover-flags'
            for (const [enabled, label] of [[raw?.hub, 'hub'], [raw?.border, 'border'], [raw?.international, "int'l"]] as const) {
                if (!enabled) continue
                const flag = document.createElement('span'); flag.textContent = label; flags.append(flag)
            }
            if (flags.childElementCount) hoverCard.append(flags)
            const updated = document.createElement('small'); updated.textContent = `last ${hours === 168 ? '7 days' : `${hours} hour${hours === 1 ? '' : 's'}`} · ESI`
            hoverCard.append(updated)
            hoverCard.hidden = false
        }, 250)
    }
    const searchSystems = (query: string) => {
        searchResults.replaceChildren()
        if (!query) { searchResults.hidden = true; return }
        const hits = layout.nodes.filter(node => node.system_name.toLowerCase().includes(query.toLowerCase())).slice(0, 8)
        for (const node of hits) {
            const link = document.createElement('a'); link.href = `/system/${node.id}`; link.textContent = node.system_name
            const security = document.createElement('small'); security.textContent = node.security.toFixed(1); security.style.color = secColorStr(node.security)
            link.append(security); searchResults.append(link)
        }
        if (!hits.length) { const empty = document.createElement('span'); empty.textContent = 'No matching systems'; searchResults.append(empty) }
        searchResults.hidden = false
    }
    const fetchHours = async () => {
        controller?.abort()
        const request = new AbortController(); controller = request
        loading.hidden = false; loading.textContent = 'Loading map data...'
        try {
            const response = await fetch(`/api/map/region/${id}?hours=${hours}`, { credentials: 'same-origin', signal: request.signal })
            if (!response.ok) throw new Error('Unable to load map data')
            const next = await response.json() as RegionData
            if (controller !== request) return
            data = next; render(); searchSystems(search.value.trim())
        } catch (error) { if ((error as Error).name !== 'AbortError') loading.textContent = (error as Error).message || 'Unable to load map data' }
    }
    root.addEventListener('click', event => {
        const button = (event.target as Element).closest<HTMLButtonElement>('button')
        if (!button || !root.contains(button) || button.disabled) return
        if (button.matches('[data-region-base-option]')) { base = button.dataset.regionBaseOption!; render() }
        else if (button.matches('[data-region-activity-option]')) { activity = button.dataset.regionActivityOption as MapActivityLayer; render() }
        else if (button.matches('[data-region-hours-option]')) { hours = Number(button.dataset.regionHoursOption); sync(); fetchHours() }
        else if (button.matches('[data-region-toggle]')) {
            const key = button.dataset.regionToggle
            if (key === 'routes') routes = !routes
            else if (key === 'systems') systems = !systems
            else if (key === 'labels') labels = !labels
            render()
        } else if (button.matches('[data-region-reset]')) resetView()
    })
    search.addEventListener('input', () => searchSystems(search.value.trim()))
    search.addEventListener('focus', () => { if (searchResults.childElementCount) searchResults.hidden = false })
    search.addEventListener('blur', () => window.setTimeout(() => { searchResults.hidden = true }, 150))
    svg.addEventListener('wheel', event => {
        event.preventDefault()
        const rect = svg.getBoundingClientRect(), before = svgPoint(event.clientX, event.clientY)
        zoom = Math.max(.5, Math.min(20, zoom * (event.deltaY < 0 ? 1.15 : 1 / 1.15)))
        viewBox()
        const after = svgPoint(event.clientX, event.clientY)
        panX += before.x - after.x; panY += before.y - after.y
        viewBox()
    }, { passive: false })
    const svgPoint = (clientX: number, clientY: number) => {
        const point = svg.createSVGPoint(); point.x = clientX; point.y = clientY
        return point.matrixTransform(svg.getScreenCTM()!.inverse())
    }
    svg.addEventListener('pointerdown', event => {
        if (event.button !== 0) return
        drag = { x: event.clientX, y: event.clientY, moved: false }
    })
    window.addEventListener('pointermove', event => {
        if (!drag) return
        const dx = event.clientX - drag.x, dy = event.clientY - drag.y
        if (Math.abs(dx) + Math.abs(dy) > 3) drag.moved = true
        if (!drag.moved) return
        const rect = svg.getBoundingClientRect()
        panX -= dx * (layout.bbox.w / zoom) / rect.width
        panY -= dy * (layout.bbox.h / zoom) / rect.height
        drag.x = event.clientX; drag.y = event.clientY
        svg.classList.add('panning'); hideHover(); viewBox()
    })
    window.addEventListener('pointerup', () => {
        if (!drag) return
        suppressClick = drag.moved; drag = null
        svg.classList.remove('panning')
        if (suppressClick) window.setTimeout(() => { suppressClick = false }, 0)
    })
    svg.addEventListener('click', event => {
        if (!suppressClick) return
        suppressClick = false; event.preventDefault(); event.stopPropagation()
    }, true)
    render()
}

initializeRegionMap()
