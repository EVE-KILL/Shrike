<script setup lang="ts">
import { AU, hasReplayPosition, replayPoint, replaySide, replayBounds, replayLossTotals, bigKillClass } from '~/utils/map/replay'
import type { ReplayKill } from '~/utils/map/replay'
const props = defineProps<{ endpoint: string; startTime: string; endTime: string; teams: any[]; teamEntities: { corps: number[]; alliances: number[] }[] }>()
interface Landmark { id: number; name: string | null; solar_system_id: number; group_id: number; x: number; y: number; z: number }
// Resolve data before initial system selection; SSR watchers do not rerun
// when an un-awaited fetch completes during server prefetch.
const { data, pending, error, refresh } = await useApiFetch<{ kills: ReplayKill[]; landmarks: Landmark[] }>(props.endpoint)
const rawKills = computed(() => (data.value?.kills ?? []).filter(kill => Number.isFinite(Date.parse(kill.killmail_time))))
const { filters, filtered: kills, clear: clearFilters } = useBattleExplorer(rawKills, () => props.teamEntities)
const route = useRoute()
const hasFilters = computed(() => filters.value.side != null || filters.value.group != null || filters.value.minIsk > 0 || filters.value.from != null || filters.value.to != null)
const systems = computed(() => Array.from(new Map(kills.value.map(kill => [kill.solar_system_id, kill.solar_system_name ?? `System ${kill.solar_system_id}`])).entries()))
const systemId = ref<number | null>(null)
watch(systems, value => { if (!value.some(([id]) => id === systemId.value)) systemId.value = value[0]?.[0] ?? null }, { immediate: true })
const systemKills = computed(() => kills.value.filter(kill => kill.solar_system_id === systemId.value))
const positioned = computed(() => systemKills.value.filter(hasReplayPosition))
const points = computed(() => positioned.value.map(kill => ({ ...replayPoint(kill, 'xz'), kill, time: Date.parse(kill.killmail_time), side: replaySide(kill, props.teamEntities ?? []) })))
const showOrbits = ref(true)
const showMoons = ref(true)
const showLabels = ref(true)
const landmarkName = (landmark: Landmark) => (landmark.name ?? `Object ${landmark.id}`).replace(`${systems.value.find(([id]) => id === systemId.value)?.[1]} `, '').replace(/^- /, '')
const landmarkKind = (group: number) => ({ 6: 'Sun', 7: 'Planet', 8: 'Moon', 9: 'Belt', 10: 'Gate', 15: 'Station' }[group] ?? 'Celestial')
const landmarkColor = (group: number) => ({ 6: '#fbbf24', 7: '#7dd3fc', 8: '#64748b', 9: '#78716c', 10: '#34d399', 15: '#c084fc' }[group] ?? '#94a3b8')
const systemLandmarks = computed(() => (data.value?.landmarks ?? []).filter(item => item.solar_system_id === systemId.value))
const landmarkPoints = computed(() => systemLandmarks.value.map(item => ({ ...item, px: item.x / AU, py: -item.z / AU })))
const bounds = ref(replayBounds([]))
const fit = () => { bounds.value = replayBounds(points.value) }
// Match internal/images/maps.go: planet rings use the projected distance
// from the star. These are schematic orbit guides, not orbital ephemerides.
const orbits = computed(() => {
    const sun = landmarkPoints.value.find(item => item.group_id === 6)
    if (!sun) return []
    return landmarkPoints.value.filter(item => item.group_id === 7).map(planet => ({
        id: planet.id, x: sun.px, y: sun.py, radius: Math.hypot(planet.px - sun.px, planet.py - sun.py),
    })).filter(orbit => orbit.radius > 0)
})
const fitSystem = () => {
    bounds.value = replayBounds([
        ...points.value, ...landmarkPoints.value.map(item => ({ x: item.px, y: item.py })),
        ...orbits.value.flatMap(orbit => [{ x: orbit.x - orbit.radius, y: orbit.y - orbit.radius }, { x: orbit.x + orbit.radius, y: orbit.y + orbit.radius }]),
    ])
}
watch([points, landmarkPoints], fitSystem, { immediate: true })
const visibleLandmarks = computed(() => landmarkPoints.value.filter(item => (showMoons.value || item.group_id !== 8) && item.px >= bounds.value.x && item.px <= bounds.value.x + bounds.value.width && item.py >= bounds.value.y && item.py <= bounds.value.y + bounds.value.height))
// Declutter labels in screen space while keeping every celestial marker.
const landmarkLabels = computed(() => {
    if (!showLabels.value) return new Set<number>()
    const occupied: { x: number; y: number }[] = []
    const result = new Set<number>()
    const priorities = [6, 10, 15, 7, 8, 9]
    for (const item of [...visibleLandmarks.value].sort((a, b) => priorities.indexOf(a.group_id) - priorities.indexOf(b.group_id))) {
        const x = (item.px - bounds.value.x) / bounds.value.width * 1000
        const y = (item.py - bounds.value.y) / bounds.value.height * 600
        if (occupied.some(point => Math.abs(point.x - x) < 125 && Math.abs(point.y - y) < 22)) continue
        occupied.push({ x, y }); result.add(item.id)
    }
    return result
})
const start = computed(() => Math.max(Date.parse(props.startTime), Math.min(filters.value.from ?? Date.parse(props.startTime), Date.parse(props.endTime))))
const end = computed(() => Math.max(start.value + 1, Math.min(Date.parse(props.endTime), filters.value.to ?? Date.parse(props.endTime))))
const requestedTime = () => {
    const at = Number(route.query.at)
    return Number.isFinite(at) && at > 0 ? Math.min(end.value, Math.max(start.value, at)) : start.value
}
const time = ref(requestedTime())
const playing = ref(false)
const speed = ref(60)
const selectedId = ref<number | null>(Number(route.query.kill) || null)
watch(systemId, () => { selectedId.value = null })
watch(() => [route.query.at, route.query.kill], () => { time.value = requestedTime(); playing.value = false; selectedId.value = Number(route.query.kill) || null })
watch([start, end], () => { time.value = requestedTime(); playing.value = false })
const selected = computed(() => kills.value.find(kill => kill.killmail_id === selectedId.value))
const visiblePoints = computed(() => points.value.filter(point => point.time <= time.value))
const occurred = computed(() => systemKills.value.filter(kill => Date.parse(kill.killmail_time) <= time.value))
const recent = computed(() => occurred.value.slice(-30).reverse())
const lossTotals = computed(() => replayLossTotals(systemKills.value, props.teamEntities ?? [], time.value))
const teamLabel = (index: number) => `Team ${String.fromCharCode(65 + index)}`
const teamIdentity = (index: number) => {
    const team = props.teams[index]
    return team?.name || team?.alliances?.[0]?.alliance_name || team?.alliances?.[0]?.corporations?.[0]?.corporation_name || ''
}
const lossCards = computed(() => [
    ...props.teams.map((_, index) => ({ label: teamLabel(index), identity: teamIdentity(index), side: index, ...(lossTotals.value.teams[index] ?? { losses: 0, isk: 0 }) })),
    ...(lossTotals.value.unassigned.losses ? [{ label: 'Unassigned', identity: 'Outside the assigned teams', side: -1, ...lossTotals.value.unassigned }] : []),
])
const destroyed = computed(() => occurred.value.reduce((sum, kill) => sum + kill.total_value, 0))
const viewBox = computed(() => `${bounds.value.x} ${bounds.value.y} ${bounds.value.width} ${bounds.value.height}`)
const palette = ['#f87171', '#60a5fa', '#c084fc', '#34d399', '#fbbf24']
const color = (side: number) => side < 0 ? '#94a3b8' : palette[side % palette.length]
const clock = (value: number) => new Date(value).toISOString().slice(11, 19)
const spanLabel = computed(() => bounds.value.width >= 0.01 ? `${bounds.value.width.toFixed(2)} AU across` : `${Math.round(bounds.value.width * 149597870.7).toLocaleString()} km across`)
const systemName = computed(() => systems.value.find(([id]) => id === systemId.value)?.[1] ?? 'System')
const pulseWindow = computed(() => Math.max(30000, speed.value * 3000))
const freshPoints = computed(() => {
    const recent = visiblePoints.value.filter(point => time.value - point.time < pulseWindow.value)
    return [...recent.filter(point => !bigKillClass(point.kill)).slice(-16), ...recent.filter(point => bigKillClass(point.kill))]
})
const bigKill = computed(() => [...occurred.value].reverse().find(kill => bigKillClass(kill) && time.value - Date.parse(kill.killmail_time) < Math.max(60000, speed.value * 6000)))
const pulseAge = (point: typeof points.value[number]) => Math.min(1, Math.max(0, (time.value - point.time) / pulseWindow.value))
const pulseSize = (point: typeof points.value[number]) => (bigKillClass(point.kill) ? 0.022 : 0.008) + Math.max(0, Math.log10(Math.max(1, point.kill.total_value)) - 7) * 0.002
const stars = Array.from({ length: 130 }, (_, i) => ({ x: ((i * 7919 + 17) % 1000), y: ((i * 3571 + 31) % 600), r: i % 13 === 0 ? 1.2 : 0.55, opacity: 0.15 + (i % 5) * 0.09 }))
const histogram = computed(() => {
    const bins = Array.from({ length: 90 }, () => ({ count: 0, big: false }))
    for (const kill of systemKills.value) {
        const index = Math.max(0, Math.min(89, Math.floor((Date.parse(kill.killmail_time) - start.value) / (end.value - start.value) * 90)))
        bins[index]!.count++
        if (bigKillClass(kill)) bins[index]!.big = true
    }
    const max = Math.max(1, ...bins.map(bin => bin.count))
    return bins.map((bin, index) => ({ count: bin.count, big: bin.big, height: 3 + Math.sqrt(bin.count / max) * 29, played: start.value + index / 90 * (end.value - start.value) <= time.value }))
})
const stageRef = ref<HTMLElement | null>(null)
const plotRef = ref<SVGSVGElement | null>(null)
const feedRef = ref<HTMLElement | null>(null)
const hoverKillId = ref<number | null>(null)
const highlightedId = computed(() => hoverKillId.value ?? selectedId.value ?? (playing.value ? recent.value[0]?.killmail_id : null))
const highlighted = computed(() => points.value.find(point => point.kill.killmail_id === highlightedId.value && point.time <= time.value))
const geometry = ref({ left: 0, top: 0, width: 1, height: 1, originX: 0, originY: 0, stageWidth: 1, stageHeight: 1 })
function measureStage() {
    const stage = stageRef.value?.getBoundingClientRect(), plot = plotRef.value?.getBoundingClientRect()
    if (!stage || !plot) return
    const row = feedRef.value?.querySelector<HTMLElement>(`[data-replay-kill="${highlightedId.value}"]`)
    const rowRect = row?.getBoundingClientRect(), feedRect = feedRef.value?.getBoundingClientRect()
    const rowVisible = rowRect && feedRect && rowRect.top >= feedRect.top && rowRect.bottom <= feedRect.bottom
    geometry.value = { left: plot.left - stage.left, top: plot.top - stage.top, width: plot.width, height: plot.height, originX: (rowVisible ? rowRect.left : feedRect?.left ?? plot.right) - stage.left, originY: (rowVisible ? rowRect.top + rowRect.height / 2 : (feedRect?.top ?? plot.top) + 24) - stage.top, stageWidth: stage.width, stageHeight: stage.height }
}
useResizeObserver(stageRef, measureStage)
watch([highlightedId, () => recent.value[0]?.killmail_id, systemId], () => nextTick(measureStage), { flush: 'post' })
const leader = computed(() => {
    const point = highlighted.value
    if (!point) return null
    const g = geometry.value, b = bounds.value
    const scale = Math.min(g.width / b.width, g.height / b.height)
    const x = g.left + (g.width - b.width * scale) / 2 + (point.x - b.x) * scale
    const y = g.top + (g.height - b.height * scale) / 2 + (point.y - b.y) * scale
    if (x < g.left || x > g.left + g.width || y < g.top || y > g.top + g.height) return null
    const bend = Math.max(50, Math.abs(g.originX - x) * 0.35)
    return { x, y, path: `M ${g.originX} ${g.originY} C ${g.originX - bend} ${g.originY}, ${x + bend} ${y}, ${x} ${y}`, color: color(point.side), labelX: Math.max(g.left + 12, Math.min(x + 23, g.left + g.width - 170)), labelY: Math.max(g.top + 90, Math.min(y - 22, g.top + g.height - 65)) }
})
function hoverKill(kill: ReplayKill) { hoverKillId.value = kill.killmail_id }
function jumpToBin(index: number) { playing.value = false; selectedId.value = null; time.value = start.value + index / 90 * (end.value - start.value) }
let timer: ReturnType<typeof setInterval> | undefined
let lastTick = 0
onMounted(() => {
    timer = setInterval(() => {
        const now = performance.now()
        if (playing.value && !document.hidden && lastTick) {
            time.value = Math.min(end.value, time.value + Math.min(now - lastTick, 250) * speed.value)
            if (time.value >= end.value) playing.value = false
        }
        lastTick = now
    }, 50)
})
onUnmounted(() => { if (timer) clearInterval(timer) })
function togglePlay() {
    if (time.value >= end.value) time.value = start.value
    playing.value = !playing.value
}
const nearestLandmark = computed(() => {
    const kill = selected.value
    if (!kill || !hasReplayPosition(kill)) return null
    let nearest: Landmark | null = null, distance = Infinity
    for (const item of systemLandmarks.value) {
        const metres = Math.hypot(kill.position_x! - item.x, kill.position_y! - item.y, kill.position_z! - item.z)
        if (metres < distance) { nearest = item; distance = metres }
    }
    return nearest ? { name: nearest.name, distance: distance > AU / 100 ? `${(distance / AU).toFixed(2)} AU` : `${Math.round(distance / 1000).toLocaleString()} km` } : null
})
function focusSelected() {
    if (!selected.value || !hasReplayPosition(selected.value)) return
    const point = replayPoint(selected.value, 'xz')
    const width = 0.0001
    bounds.value = { x: point.x - width / 2, y: point.y - width * 0.3, width, height: width * 0.6 }
}
function selectKill(kill: ReplayKill) { selectedId.value = kill.killmail_id; playing.value = false; nextTick(measureStage) }
function seek(event: Event) { playing.value = false; time.value = Number((event.target as HTMLInputElement).value); selectedId.value = null }
let drag: { x: number; y: number } | null = null
function down(event: PointerEvent) {
    if (event.button !== 0 || (event.target as Element).closest('a')) return
    drag = { x: event.clientX, y: event.clientY }
    ;(event.currentTarget as SVGSVGElement).setPointerCapture(event.pointerId)
}
function move(event: PointerEvent) {
    if (!drag) return
    const rect = (event.currentTarget as SVGSVGElement).getBoundingClientRect()
    const scale = Math.min(rect.width / bounds.value.width, rect.height / bounds.value.height)
    bounds.value = { ...bounds.value, x: bounds.value.x - (event.clientX - drag.x) / scale, y: bounds.value.y - (event.clientY - drag.y) / scale }
    drag = { x: event.clientX, y: event.clientY }
}
</script>

<template>
    <section aria-label="Battle map replay" class="replay space-y-3">
        <div v-if="pending" role="status" class="glass-panel p-10 text-center text-gray-400">Loading kill positions…</div>
        <div v-else-if="error" role="alert" class="glass-panel p-6 text-red-300">Unable to load the replay. <button class="underline" @click="refresh()">Retry</button></div>
        <template v-else>
            <div v-if="hasFilters" class="glass-panel flex items-center gap-3 p-3 text-xs text-sky-300">Showing losses matching your selected side, class, value and time filters.<button class="ml-auto underline" @click="clearFilters">Show full battle</button></div>
            <div class="grid grid-cols-2 gap-2" :class="lossCards.length > 2 ? 'lg:grid-cols-4' : ''" aria-label="Team losses at replay time">
                <div v-for="team in lossCards" :key="team.side" class="relative overflow-hidden rounded-lg border border-white/[0.08] bg-slate-900/40 px-4 py-3">
                    <div class="absolute inset-y-0 left-0 w-0.5" :style="{ background: color(team.side) }" />
                    <div class="flex items-center justify-between gap-2"><span class="text-[10px] font-semibold uppercase tracking-[0.15em]" :style="{ color: color(team.side) }">{{ team.label }}</span><span class="font-mono text-[9px] text-slate-600">{{ clock(time) }} EVE</span></div>
                    <div v-if="team.identity" class="mt-1 truncate text-[10px] text-slate-500">{{ team.identity }}</div>
                    <div class="mt-2 flex flex-wrap items-baseline justify-between gap-1"><span class="font-mono text-xl text-slate-100">{{ formatNumber(team.losses) }}<span class="ml-1.5 text-[10px] text-slate-500">losses</span></span><span class="font-mono text-sm text-amber-100">{{ formatIsk(team.isk) }}<span class="ml-1 text-[9px] text-slate-500">ISK lost</span></span></div>
                </div>
            </div>
            <div class="flex flex-wrap items-center gap-3">
                <label class="text-xs text-gray-400">System <select v-model="systemId" class="ml-2 rounded border border-white/10 bg-[#141414] px-3 py-2 text-gray-200"><option v-for="[id, name] in systems" :key="id" :value="id">{{ name }}</option></select></label>
                <button class="rounded border border-white/10 px-3 py-2 text-xs text-blue-300" @click="fitSystem">Whole system</button>
                <button class="rounded border border-white/10 px-3 py-2 text-xs text-blue-300" @click="fit">Fit all kills</button>
                <label class="flex items-center gap-1 text-xs text-gray-400"><input v-model="showOrbits" type="checkbox">Orbits</label>
                <label class="flex items-center gap-1 text-xs text-gray-400"><input v-model="showMoons" type="checkbox">Moons</label>
                <label class="flex items-center gap-1 text-xs text-gray-400"><input v-model="showLabels" type="checkbox">Labels</label>
                <span class="ml-auto text-xs text-gray-500">{{ positioned.length }} of {{ systemKills.length }} kills have positions</span>
            </div>
            <p v-if="!systemLandmarks.length" class="text-xs text-amber-300">No celestial coordinates are available for this system.</p>
            <p class="text-xs text-gray-500">Recorded destruction locations, coloured by the victim’s side. Top-down view · drag to pan. Orbit guides follow the system images; ship movement is not recorded.</p>
            <div ref="stageRef" class="replay-stage relative grid overflow-hidden rounded-xl border border-slate-700/40 lg:grid-cols-[minmax(0,1fr)_288px]">
                <svg v-if="leader && highlighted" class="pointer-events-none absolute inset-0 z-20 h-full w-full overflow-visible" :viewBox="`0 0 ${geometry.stageWidth} ${geometry.stageHeight}`" aria-hidden="true">
                    <path :d="leader.path" fill="none" :stroke="leader.color" stroke-width="9" opacity="0.12" class="replay-beam-glow" />
                    <path :d="leader.path" fill="none" :stroke="leader.color" stroke-width="1.4" opacity="0.8" />
                    <path :d="leader.path" fill="none" stroke="#fff" stroke-width="1.6" stroke-dasharray="4 24" opacity="0.7" class="replay-beam-flow" />
                    <circle :cx="leader.x" :cy="leader.y" r="22" :stroke="leader.color" stroke-width="1" fill="none" opacity="0.7" />
                    <circle :cx="leader.x" :cy="leader.y" r="5" fill="#fff" />
                    <path :d="`M ${leader.x-30} ${leader.y} h 10 M ${leader.x+20} ${leader.y} h 10 M ${leader.x} ${leader.y-30} v 10 M ${leader.x} ${leader.y+20} v 10`" :stroke="leader.color" stroke-width="1.5" />
                    <g :transform="`translate(${leader.labelX} ${leader.labelY})`">
                        <rect x="-7" y="-16" width="166" height="40" rx="5" fill="#080e1b" fill-opacity="0.94" :stroke="leader.color" stroke-opacity="0.35" />
                        <text :fill="leader.color" font-size="11" font-weight="600">{{ (highlighted.kill.ship_name || 'Unknown ship').slice(0, 24) }}</text>
                        <text fill="#94a3b8" font-size="10" y="15">{{ formatIsk(highlighted.kill.total_value) }} ISK · {{ clock(highlighted.time) }}</text>
                    </g>
                </svg>
                <div class="replay-space relative min-w-0 overflow-hidden">
                    <svg class="pointer-events-none absolute inset-0 h-full w-full" viewBox="0 0 1000 600" preserveAspectRatio="none" aria-hidden="true"><circle v-for="(star, index) in stars" :key="index" :cx="star.x" :cy="star.y" :r="star.r" fill="#b9d6ff" :opacity="star.opacity" /></svg>
                    <svg ref="plotRef" :viewBox="viewBox" class="relative h-[64vh] min-h-[440px] w-full touch-none" role="img" aria-label="Kill positions in the selected system"
                        @pointerdown="down" @pointermove="move" @pointerup="drag = null" @pointercancel="drag = null">
                        <defs>
                            <radialGradient id="replay-sun-glow"><stop offset="0" stop-color="#fff7d6" stop-opacity="0.9" /><stop offset="0.12" stop-color="#fbbf24" stop-opacity="0.6" /><stop offset="0.4" stop-color="#fb923c" stop-opacity="0.12" /><stop offset="1" stop-color="#fb923c" stop-opacity="0" /></radialGradient>
                            <radialGradient id="replay-planet"><stop offset="0" stop-color="#d5f2ff" /><stop offset="0.5" stop-color="#6198b8" /><stop offset="1" stop-color="#18354e" /></radialGradient>
                        </defs>
                        <g v-if="showOrbits" fill="none" stroke="#7294b6" stroke-opacity="0.25" stroke-width="1">
                            <circle v-for="orbit in orbits" :key="orbit.id" :cx="orbit.x" :cy="orbit.y" :r="orbit.radius" vector-effect="non-scaling-stroke" />
                        </g>
                        <g v-for="item in visibleLandmarks" :key="item.id" :transform="`translate(${item.px} ${item.py})`">
                            <title>{{ landmarkKind(item.group_id) }} · {{ item.name }}</title>
                            <circle v-if="item.group_id === 6" :r="bounds.width * 0.055" fill="url(#replay-sun-glow)" />
                            <circle v-if="item.group_id === 6" :r="bounds.width * 0.02" fill="none" stroke="#fbbf24" stroke-opacity="0.12" vector-effect="non-scaling-stroke" />
                            <path v-if="item.group_id === 10" :d="`M 0 ${-bounds.width * 0.006} L ${bounds.width * 0.006} 0 L 0 ${bounds.width * 0.006} L ${-bounds.width * 0.006} 0 Z`" fill="#08090d" :stroke="landmarkColor(item.group_id)" vector-effect="non-scaling-stroke" />
                            <rect v-else-if="item.group_id === 15" :x="-bounds.width * 0.005" :y="-bounds.width * 0.005" :width="bounds.width * 0.01" :height="bounds.width * 0.01" fill="#08090d" :stroke="landmarkColor(item.group_id)" vector-effect="non-scaling-stroke" />
                            <circle v-else :r="bounds.width * (item.group_id === 6 ? 0.007 : item.group_id === 7 ? 0.004 : 0.002)" :fill="item.group_id === 7 ? 'url(#replay-planet)' : item.group_id === 6 ? '#fff2c4' : landmarkColor(item.group_id)" :fill-opacity="item.group_id === 8 ? 0.5 : 1" />
                            <text v-if="landmarkLabels.has(item.id)" :x="bounds.width * 0.01" :y="bounds.width * 0.004" :font-size="bounds.width * 0.012" :fill="landmarkColor(item.group_id)" stroke="#08090d" :stroke-width="bounds.width * 0.003" paint-order="stroke" font-family="sans-serif">{{ landmarkName(item) }}</text>
                        </g>
                        <a v-for="point in visiblePoints" :key="point.kill.killmail_id" :href="`/kill/${point.kill.killmail_id}`" @click.prevent="selectKill(point.kill)" @mouseenter="hoverKill(point.kill)" @mouseleave="hoverKillId = null">
                            <circle :cx="point.x" :cy="point.y" :r="bounds.width * 0.0025" :fill="color(point.side)" fill-opacity="0.38">
                                <title>{{ point.kill.ship_name }} · {{ point.kill.victim_character_name || 'Unknown pilot' }} · {{ clock(point.time) }} EVE</title>
                            </circle>
                        </a>
                        <g v-for="point in freshPoints" :key="`pulse-${point.kill.killmail_id}`" :transform="`translate(${point.x} ${point.y})`" pointer-events="none">
                            <circle :r="bounds.width * pulseSize(point) * 2.5" :fill="color(point.side)" :opacity="(1 - pulseAge(point)) * 0.08" />
                            <circle :r="bounds.width * pulseSize(point) * (0.7 + pulseAge(point) * 2)" fill="none" :stroke="color(point.side)" :stroke-opacity="(1 - pulseAge(point)) * 0.85" stroke-width="1.8" vector-effect="non-scaling-stroke" />
                            <circle :r="bounds.width * pulseSize(point) * (0.4 + pulseAge(point))" fill="none" :stroke="color(point.side)" :stroke-opacity="(1 - pulseAge(point)) * 0.5" vector-effect="non-scaling-stroke" />
                            <circle :r="bounds.width * pulseSize(point) * 0.35" :fill="color(point.side)" :opacity="1 - pulseAge(point) * 0.7" />
                            <circle :r="bounds.width * 0.002" fill="#fff" :opacity="1 - pulseAge(point)" />
                        </g>
                    </svg>
                    <div class="pointer-events-none absolute left-5 right-5 top-5 flex items-start justify-between gap-4">
                        <div><div class="mb-1 flex items-center gap-2 text-[9px] font-semibold uppercase tracking-[0.24em] text-sky-300/65"><span class="h-1.5 w-1.5 rounded-full" :class="playing ? 'bg-rose-400 animate-pulse' : 'bg-sky-400'" />{{ playing ? 'Battle replay · playing' : 'Battle replay · paused' }}</div><div class="text-xl font-light tracking-wider text-slate-100">{{ systemName }}</div><div class="mt-1 font-mono text-[10px] text-slate-500">{{ spanLabel }}</div></div>
                        <div class="text-right"><div class="font-mono text-lg tracking-wider text-slate-200">{{ clock(time) }}</div><div class="mt-1 text-[9px] uppercase tracking-[0.2em] text-slate-500">EVE time · {{ speed }}×</div></div>
                    </div>
                    <div v-if="bigKill" :key="bigKill.killmail_id" class="replay-big-kill absolute left-1/2 top-24 z-30 flex w-[min(340px,calc(100%-2rem))] -translate-x-1/2 items-center gap-3 rounded-lg border border-amber-300/40 bg-[#1c150e]/95 p-3 shadow-[0_0_45px_rgba(245,158,11,0.18)]">
                        <img v-if="bigKill.ship_type_id" :src="`/images/types/${bigKill.ship_type_id}/icon?size=64`" alt="" class="h-12 w-12 rounded border border-amber-200/20 bg-black/30">
                        <div class="min-w-0 flex-1"><div class="flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-[0.22em] text-amber-300"><Icon name="lucide:flame" class="h-3.5 w-3.5" />BIG KILL · {{ bigKillClass(bigKill) }}</div><button class="mt-1 block max-w-full truncate text-left text-sm font-semibold text-amber-50 hover:text-white" @click="selectKill(bigKill)">{{ bigKill.ship_name }}</button><div class="mt-0.5 text-[10px] text-amber-200/60">{{ formatIsk(bigKill.total_value) }} ISK lost · {{ clock(Date.parse(bigKill.killmail_time)) }}</div></div>
                        <NuxtLink :to="`/kill/${bigKill.killmail_id}`" class="shrink-0 text-amber-300" aria-label="Open big kill"><Icon name="lucide:arrow-up-right" class="h-4 w-4" /></NuxtLink>
                    </div>
                    <div v-if="!positioned.length" class="pointer-events-none absolute inset-0 flex items-center justify-center p-6 text-center text-sm text-gray-400">No recorded positions in this system. Kills still appear in the replay feed.</div>
                    <div class="pointer-events-none absolute bottom-4 left-5 right-5 flex flex-wrap gap-3 border-t border-slate-500/15 pt-3 text-[10px] text-slate-400">
                        <span v-for="(team, index) in teams" :key="index" class="flex items-center gap-1.5"><span class="h-2 w-2 rounded-full" :style="{ background: color(index) }" />{{ teamLabel(index) }}</span>
                        <span class="flex items-center gap-1.5"><span class="h-2 w-2 rounded-full bg-slate-400" />Unassigned</span>
                        <span class="ml-auto text-slate-600">◇ Gate · □ Station · {{ systemLandmarks.length }} celestials</span>
                    </div>
                </div>
                <aside class="replay-feed relative z-10 flex h-80 flex-col overflow-hidden border-t border-slate-700/30 lg:h-[64vh] lg:min-h-[440px] lg:border-l lg:border-t-0">
                    <div class="border-b border-slate-700/30 p-4"><div class="flex items-center justify-between text-[9px] uppercase tracking-[0.2em] text-slate-500"><span>Destruction feed</span><Icon name="lucide:radio" class="h-3 w-3 text-rose-400" /></div><div class="mt-3 flex items-baseline justify-between gap-2"><span class="font-mono text-xl text-slate-100">{{ formatNumber(occurred.length) }}<span class="ml-1 text-[10px] text-slate-500">kills</span></span><span class="font-mono text-sm text-amber-200">{{ formatIsk(destroyed) }}<span class="ml-1 text-[9px] text-slate-500">ISK</span></span></div></div>
                    <div ref="feedRef" class="min-h-0 flex-1 overflow-y-auto" @scroll="measureStage" @mouseleave="hoverKillId = null">
                        <button v-for="kill in recent" :key="kill.killmail_id" class="replay-feed-row relative flex w-full items-center gap-2.5 border-b border-white/[0.04] px-3 py-2.5 text-left transition-colors" :class="highlightedId === kill.killmail_id ? 'bg-sky-300/[0.07]' : ''" :data-replay-kill="kill.killmail_id" :style="{ '--side': color(replaySide(kill, teamEntities ?? [])) }" @click="selectKill(kill)" @mouseenter="hoverKill(kill)" @focus="hoverKill(kill)" @blur="hoverKillId = null">
                            <img v-if="kill.ship_type_id" :src="`/images/types/${kill.ship_type_id}/icon?size=64`" alt="" class="h-10 w-10 rounded-md border border-white/10 bg-slate-900" loading="lazy">
                            <span class="min-w-0 flex-1"><span v-if="bigKillClass(kill)" class="mb-0.5 block text-[8px] font-bold tracking-[0.15em] text-amber-300">BIG KILL · {{ bigKillClass(kill) }}</span><span class="block truncate text-xs" :style="{ color: color(replaySide(kill, teamEntities ?? [])) }">{{ kill.ship_name || 'Unknown ship' }}</span><span class="block truncate text-[10px] text-gray-500">{{ kill.victim_character_name || 'Unknown pilot' }}</span></span>
                            <span class="shrink-0 text-right"><span class="block font-mono text-[10px] text-amber-100/80">{{ formatIsk(kill.total_value) }}</span><span class="mt-1 block text-[9px] text-slate-600">{{ clock(Date.parse(kill.killmail_time)) }}</span></span>
                        </button>
                        <p v-if="!recent.length" class="p-5 text-xs text-gray-500">Press play or scrub forward to follow the losses.</p>
                    </div>
                <div class="border-t border-slate-700/30 px-4 py-2 text-[9px] text-slate-600">Hover to locate · click to inspect</div>
                </aside>
            </div>
            <div class="replay-transport rounded-xl border border-slate-700/30 p-3">
                <div class="mb-2 flex h-9 items-end gap-[2px]" aria-label="Battle activity timeline">
                    <button v-for="(bin, index) in histogram" :key="index" class="min-w-0 flex-1 rounded-t-sm transition-colors hover:bg-sky-200" :class="bin.big ? 'bg-amber-400/80' : bin.played ? 'bg-sky-400/60' : 'bg-slate-600/40'" :style="{ height: `${bin.height}px` }" :title="`${clock(start + index / 90 * (end - start))} · ${bin.count} kills`" :aria-label="`Jump to ${clock(start + index / 90 * (end - start))}: ${bin.count} kills`" @click="jumpToBin(index)" />
                </div>
                <div class="flex flex-wrap items-center gap-3">
                <button :disabled="!kills.length" class="flex items-center gap-2 rounded-md bg-blue-500 px-4 py-2 text-sm text-white disabled:opacity-40" @click="togglePlay"><Icon :name="playing ? 'lucide:pause' : 'lucide:play'" />{{ playing ? 'Pause' : 'Play' }}</button>
                <button class="text-xs text-gray-400" @click="playing = false; time = start; selectedId = null">Restart</button>
                <input aria-label="Replay time" type="range" :min="start" :max="end" :step="1000" :value="time" class="min-w-40 flex-1 accent-blue-400" @input="seek">
                <span class="font-mono text-xs text-gray-300">{{ clock(time) }} EVE</span>
                <label class="text-xs text-gray-400">Speed <select v-model="speed" class="ml-1 rounded bg-[#141414] p-2"><option v-for="rate in [1, 10, 30, 60, 120, 300]" :key="rate" :value="rate">{{ rate }}×</option></select></label>
            </div>
            </div>
            <div v-if="selected" class="glass-panel flex flex-wrap items-center gap-3 p-3 text-sm">
                <span class="text-gray-200">{{ selected.ship_name }} · {{ selected.victim_character_name || 'Unknown pilot' }}</span><span class="text-gray-400">{{ formatIsk(selected.total_value) }} ISK</span>
                <span v-if="!hasReplayPosition(selected)" class="text-xs text-amber-300">Position unavailable</span>
                <span v-if="nearestLandmark" class="text-xs text-gray-400">{{ nearestLandmark.distance }} from {{ nearestLandmark.name }}</span>
                <button v-if="hasReplayPosition(selected)" class="text-xs text-blue-300" @click="focusSelected">Focus kill location</button>
                <NuxtLink :to="`/kill/${selected.killmail_id}`" class="ml-auto text-blue-300 hover:underline">Open killmail →</NuxtLink>
            </div>
        </template>
    </section>
</template>

<style scoped>
.replay-space {
    background: radial-gradient(ellipse at 38% 44%, rgba(34, 72, 111, .16), transparent 55%), radial-gradient(ellipse at 76% 85%, rgba(50, 30, 88, .15), transparent 50%), #060b14;
    box-shadow: inset 0 0 110px rgba(0, 0, 0, .45);
}
.replay-stage { box-shadow: 0 16px 60px rgba(0,0,0,.3), 0 0 35px rgba(56,189,248,.025); }
.replay-feed { background: linear-gradient(160deg, #0d1420, #080d16); }
.replay-feed-row:hover { background: rgba(125,211,252,.055); }
.replay-feed-row::before { content: ''; position: absolute; inset: 10px auto 10px 0; width: 2px; background: var(--side); opacity: .45; }
.replay-feed-row:hover::before { opacity: 1; box-shadow: 0 0 12px var(--side); }
.replay-transport { background: linear-gradient(110deg, #0c1420, #0a0e17); }
.replay-beam-glow { filter: blur(3px); }
.replay-big-kill { animation: big-kill-arrival .45s ease-out; }
@keyframes big-kill-arrival { from { opacity: 0; margin-top: -8px; } to { opacity: 1; margin-top: 0; } }
.replay-beam-flow { animation: beam-flow 1.8s linear infinite; }
@keyframes beam-flow { to { stroke-dashoffset: 112; } }
@media (prefers-reduced-motion: reduce) { .replay-beam-flow, .replay-big-kill { animation: none; } }
</style>
