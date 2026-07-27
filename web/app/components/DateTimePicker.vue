<script setup lang="ts">
/**
 * Date + time picker that renders the same everywhere.
 *
 * `<input type="datetime-local">` is not a consistent control: Firefox offers
 * a date popup and leaves the time as a bare text segment, Chrome gives both,
 * Safari does its own thing again, and none of them can say "this is EVE
 * time". This replaces it with one calendar + time UI we control.
 *
 * The model value is an EVE (UTC) wall clock, `YYYY-MM-DDTHH:mm` — the exact
 * shape the native input used to hold under EVE-time semantics, so callers
 * keep passing it to eveInputToIso() unchanged. The viewer may flip the
 * display to their own zone; that changes what is drawn, never what is
 * stored, and the preference is shared across every picker and remembered.
 */
const props = withDefaults(defineProps<{
    /** EVE wall clock, `YYYY-MM-DDTHH:mm`. Empty string when unset. */
    modelValue: string
    disabled?: boolean
    placeholder?: string
    clearable?: boolean
    /** Compact trigger, for dense filter bars. */
    size?: 'sm' | 'md'
}>(), {
    disabled: false,
    placeholder: 'Select date and time',
    clearable: true,
    size: 'md',
})

const emit = defineEmits<{ 'update:modelValue': [string] }>()

const zone = useTimeZonePref()

// Rendering a LOCAL clock during SSR would disagree with the client (the pod
// runs UTC, the viewer does not) and blow up hydration. Until mounted, always
// draw EVE time — which is also the default, so most viewers see no change.
const mounted = ref(false)
onMounted(() => { mounted.value = true })
const displayZone = computed<'eve' | 'local'>(() => (mounted.value ? zone.value : 'eve'))

const MONTHS = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December']
const WEEKDAYS = ['Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa', 'Su']
const pad = (n: number) => String(n).padStart(2, '0')

interface Parts { y: number; mo: number; d: number; h: number; mi: number }

/** The stored value is an EVE wall clock, so `Z` makes it an instant. */
const modelInstant = computed<Date | null>(() => {
    if (!props.modelValue) return null
    const date = new Date(`${props.modelValue}Z`)
    return Number.isNaN(date.getTime()) ? null : date
})

const partsIn = (date: Date, z: 'eve' | 'local'): Parts => z === 'eve'
    ? { y: date.getUTCFullYear(), mo: date.getUTCMonth(), d: date.getUTCDate(), h: date.getUTCHours(), mi: date.getUTCMinutes() }
    : { y: date.getFullYear(), mo: date.getMonth(), d: date.getDate(), h: date.getHours(), mi: date.getMinutes() }

const instantFrom = (p: Parts, z: 'eve' | 'local'): Date => z === 'eve'
    ? new Date(Date.UTC(p.y, p.mo, p.d, p.h, p.mi))
    : new Date(p.y, p.mo, p.d, p.h, p.mi)

/** Instant → the EVE wall clock we emit. */
const toModel = (date: Date): string => date.toISOString().slice(0, 16)

// Parts currently shown, in the display zone.
const shown = computed<Parts | null>(() => modelInstant.value ? partsIn(modelInstant.value, displayZone.value) : null)

const open = ref(false)
const root = ref<HTMLElement | null>(null)

// Which month the calendar is scrolled to (display zone), independent of the
// selection so browsing away doesn't change the value.
const viewYear = ref(0)
const viewMonth = ref(0)

const syncView = () => {
    const base = shown.value ?? partsIn(new Date(), displayZone.value)
    viewYear.value = base.y
    viewMonth.value = base.mo
}

/** Panel is ~304px wide; flip it right-aligned when that would overflow. */
const PANEL_WIDTH = 304
const alignRight = ref(false)

watch(open, (isOpen) => {
    if (!isOpen) return
    syncView()
    void nextTick(() => {
        if (!root.value || typeof window === 'undefined') return
        const rect = root.value.getBoundingClientRect()
        alignRight.value = rect.left + PANEL_WIDTH > window.innerWidth - 8
    })
})
// Flipping zones can move the selection across a month boundary.
watch(displayZone, () => { if (open.value) syncView() })

const shiftMonth = (delta: number) => {
    const m = viewMonth.value + delta
    viewYear.value += Math.floor(m / 12)
    viewMonth.value = ((m % 12) + 12) % 12
}

/** Calendar cells, Monday-first, padded with the neighbouring months' days. */
const calendar = computed(() => {
    const first = new Date(Date.UTC(viewYear.value, viewMonth.value, 1))
    // getUTCDay is Sunday-0; shift so Monday is column 0.
    const lead = (first.getUTCDay() + 6) % 7
    const daysInMonth = new Date(Date.UTC(viewYear.value, viewMonth.value + 1, 0)).getUTCDate()
    const cells: Array<{ key: string; day: number; inMonth: boolean; y: number; mo: number }> = []

    const prevDays = new Date(Date.UTC(viewYear.value, viewMonth.value, 0)).getUTCDate()
    for (let i = lead - 1; i >= 0; i--) {
        const m = viewMonth.value - 1
        const y = m < 0 ? viewYear.value - 1 : viewYear.value
        const mo = (m + 12) % 12
        cells.push({ key: `p${i}`, day: prevDays - i, inMonth: false, y, mo })
    }
    for (let d = 1; d <= daysInMonth; d++) {
        cells.push({ key: `c${d}`, day: d, inMonth: true, y: viewYear.value, mo: viewMonth.value })
    }
    let next = 1
    while (cells.length % 7 !== 0 || cells.length < 42) {
        const m = viewMonth.value + 1
        const y = m > 11 ? viewYear.value + 1 : viewYear.value
        cells.push({ key: `n${next}`, day: next, inMonth: false, y, mo: m % 12 })
        next++
        if (cells.length >= 42) break
    }
    return cells
})

const todayParts = computed(() => {
    // Recomputed whenever the panel opens; a stale "today" ring is harmless.
    void open.value
    return partsIn(new Date(), displayZone.value)
})

const isToday = (cell: { day: number; y: number; mo: number }) =>
    cell.y === todayParts.value.y && cell.mo === todayParts.value.mo && cell.day === todayParts.value.d

const isSelected = (cell: { day: number; y: number; mo: number }) =>
    !!shown.value && cell.y === shown.value.y && cell.mo === shown.value.mo && cell.day === shown.value.d

/** Write new display-zone parts back through the model. */
const commit = (next: Parts) => {
    emit('update:modelValue', toModel(instantFrom(next, displayZone.value)))
}

const pickDay = (cell: { day: number; inMonth: boolean; y: number; mo: number }) => {
    const current = shown.value
    commit({ y: cell.y, mo: cell.mo, d: cell.day, h: current?.h ?? 0, mi: current?.mi ?? 0 })
    if (!cell.inMonth) {
        viewYear.value = cell.y
        viewMonth.value = cell.mo
    }
}

const setHour = (value: number) => {
    const base = shown.value ?? partsIn(new Date(), displayZone.value)
    commit({ ...base, h: value })
}

const setMinute = (value: number) => {
    const base = shown.value ?? partsIn(new Date(), displayZone.value)
    commit({ ...base, mi: value })
}

const setNow = () => {
    const now = new Date()
    now.setSeconds(0, 0)
    emit('update:modelValue', toModel(now))
}

const clear = () => {
    emit('update:modelValue', '')
    open.value = false
}

const HOURS = Array.from({ length: 24 }, (_, i) => i)
const MINUTES = Array.from({ length: 60 }, (_, i) => i)

const zoneSuffix = computed(() => displayZone.value === 'eve' ? 'EVE' : 'Local')

/** Trigger label, in the display zone. */
const label = computed(() => {
    const p = shown.value
    if (!p) return ''
    return `${p.y}-${pad(p.mo + 1)}-${pad(p.d)} ${pad(p.h)}:${pad(p.mi)}`
})

/** The same instant in the OTHER clock, as a reassurance line. */
const counterpart = computed(() => {
    if (!modelInstant.value || !mounted.value) return null
    const other = displayZone.value === 'eve' ? 'local' : 'eve'
    const p = partsIn(modelInstant.value, other)
    return `${p.y}-${pad(p.mo + 1)}-${pad(p.d)} ${pad(p.h)}:${pad(p.mi)} ${other === 'eve' ? 'EVE' : 'local'}`
})

const localName = computed(() => (mounted.value ? localZoneLabel() : 'Local'))

const onDocumentPointerDown = (event: PointerEvent) => {
    if (!open.value) return
    if (root.value && !root.value.contains(event.target as Node)) open.value = false
}
const onKeydown = (event: KeyboardEvent) => {
    if (event.key === 'Escape' && open.value) open.value = false
}
onMounted(() => {
    document.addEventListener('pointerdown', onDocumentPointerDown)
    document.addEventListener('keydown', onKeydown)
})
onUnmounted(() => {
    document.removeEventListener('pointerdown', onDocumentPointerDown)
    document.removeEventListener('keydown', onKeydown)
})
</script>

<template>
    <div ref="root" class="relative">
        <!-- Trigger -->
        <button type="button" :disabled="disabled"
            class="w-full flex items-center gap-2 rounded-lg bg-white/[0.04] border text-left transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            :class="[
                size === 'sm' ? 'px-2 py-1 text-xs' : 'px-3 py-2 text-sm',
                open ? 'border-blue-500/40' : 'border-white/[0.08] hover:border-white/[0.16]',
                disabled ? '' : 'cursor-pointer',
            ]"
            @click="open = !open">
            <Icon name="lucide:calendar-clock" class="text-sm text-gray-500 flex-shrink-0" />
            <span v-if="label" class="flex-1 min-w-0 truncate tabular-nums text-gray-200">{{ label }}</span>
            <span v-else class="flex-1 min-w-0 truncate text-gray-600">{{ placeholder }}</span>
            <span class="text-fine flex-shrink-0"
                :class="displayZone === 'eve' ? 'text-blue-400/70' : 'text-amber-400/70'">{{ zoneSuffix }}</span>
            <Icon name="lucide:chevron-down" class="text-sm text-gray-600 flex-shrink-0 transition-transform"
                :class="open ? 'rotate-180' : ''" />
        </button>

        <!-- Panel -->
        <div v-if="open"
            class="absolute z-50 mt-1 w-[19rem] max-w-[calc(100vw-1.5rem)] rounded-lg border border-white/[0.10] bg-[#141414]/97 backdrop-blur-xl shadow-2xl shadow-black/60 p-3"
            :class="alignRight ? 'right-0' : 'left-0'">
            <!-- Zone toggle -->
            <div class="flex items-center justify-between gap-2 mb-3">
                <span class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Time zone</span>
                <div class="flex rounded-md border border-white/[0.08] overflow-hidden">
                    <button v-for="z in (['eve', 'local'] as const)" :key="z" type="button"
                        class="px-2 py-1 text-fine font-medium transition-colors cursor-pointer"
                        :class="displayZone === z
                            ? 'bg-blue-500/20 text-blue-400'
                            : 'bg-white/[0.02] text-gray-500 hover:bg-white/[0.05] hover:text-gray-300'"
                        v-tooltip="z === 'local' ? localName : 'EVE time (UTC)'"
                        @click="zone = z">
                        {{ z === 'eve' ? 'EVE' : 'Local' }}
                    </button>
                </div>
            </div>

            <!-- Month nav -->
            <div class="flex items-center justify-between gap-2 mb-2">
                <button type="button" class="w-7 h-7 rounded-md flex items-center justify-center text-gray-500 hover:text-gray-200 hover:bg-white/[0.06] transition-colors cursor-pointer"
                    @click="shiftMonth(-1)">
                    <Icon name="lucide:chevron-left" class="text-sm" />
                </button>
                <div class="text-xs font-medium text-gray-200 tabular-nums">{{ MONTHS[viewMonth] }} {{ viewYear }}</div>
                <button type="button" class="w-7 h-7 rounded-md flex items-center justify-center text-gray-500 hover:text-gray-200 hover:bg-white/[0.06] transition-colors cursor-pointer"
                    @click="shiftMonth(1)">
                    <Icon name="lucide:chevron-right" class="text-sm" />
                </button>
            </div>

            <!-- Calendar -->
            <div class="grid grid-cols-7 gap-0.5 mb-1">
                <div v-for="d in WEEKDAYS" :key="d" class="text-fine text-gray-600 text-center py-1">{{ d }}</div>
            </div>
            <div class="grid grid-cols-7 gap-0.5">
                <button v-for="cell in calendar" :key="cell.key" type="button"
                    class="h-8 rounded-md text-xs tabular-nums transition-colors cursor-pointer"
                    :class="[
                        isSelected(cell)
                            ? 'bg-blue-500 text-white font-semibold'
                            : cell.inMonth ? 'text-gray-300 hover:bg-blue-500/[0.12]' : 'text-gray-700 hover:bg-white/[0.05]',
                        !isSelected(cell) && isToday(cell) ? 'ring-1 ring-inset ring-blue-500/40' : '',
                    ]"
                    @click="pickDay(cell)">
                    {{ cell.day }}
                </button>
            </div>

            <!-- Time -->
            <div class="flex items-center gap-2 mt-3 pt-3 border-t border-white/[0.06]">
                <Icon name="lucide:clock" class="text-sm text-gray-500 flex-shrink-0" />
                <select :value="shown?.h ?? 0"
                    class="flex-1 min-w-0 px-2 py-1.5 rounded-md bg-gray-900 border border-white/[0.08] text-xs text-gray-200 tabular-nums focus:border-blue-500/40 focus:outline-none cursor-pointer"
                    @change="setHour(Number(($event.target as HTMLSelectElement).value))">
                    <option v-for="h in HOURS" :key="h" :value="h">{{ pad(h) }}</option>
                </select>
                <span class="text-gray-600">:</span>
                <select :value="shown?.mi ?? 0"
                    class="flex-1 min-w-0 px-2 py-1.5 rounded-md bg-gray-900 border border-white/[0.08] text-xs text-gray-200 tabular-nums focus:border-blue-500/40 focus:outline-none cursor-pointer"
                    @change="setMinute(Number(($event.target as HTMLSelectElement).value))">
                    <option v-for="m in MINUTES" :key="m" :value="m">{{ pad(m) }}</option>
                </select>
                <button type="button"
                    class="px-2 py-1.5 rounded-md text-fine font-medium text-gray-400 bg-white/[0.04] border border-white/[0.08] hover:bg-blue-500/[0.08] hover:text-blue-400 transition-colors cursor-pointer flex-shrink-0"
                    @click="setNow">
                    Now
                </button>
            </div>

            <!-- The same instant on the other clock -->
            <div v-if="counterpart" class="mt-2 text-fine text-gray-600 tabular-nums">
                = {{ counterpart }}
            </div>

            <div class="flex items-center justify-between gap-2 mt-3">
                <button v-if="clearable" type="button" class="text-fine text-gray-500 hover:text-red-400 transition-colors cursor-pointer" @click="clear">
                    Clear
                </button>
                <span v-else />
                <button type="button"
                    class="px-3 py-1.5 rounded-md text-fine font-medium bg-blue-500 text-white hover:bg-blue-600 transition-colors cursor-pointer"
                    @click="open = false">
                    Done
                </button>
            </div>
        </div>
    </div>
</template>
