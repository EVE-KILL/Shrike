<script setup lang="ts">
// Compact quick-filter panel that pops down below the killlist header. Lets
// users narrow the list by region OR system, victim ship, and security space —
// a convenience subset of /advancedsearch. Selections apply live (no Apply
// button); the parent serialises them into the /api/killlist/advanced `filters`
// payload. Region and system are mutually exclusive (the backend prefers
// system), so picking one clears the other.
import { useDebounceFn } from '@vueuse/core'

interface FilterState {
    regionId?: number
    regionName?: string
    systemId?: number
    systemName?: string
    shipId?: number
    shipName?: string
    securityTypes: string[]
    soloOnly?: boolean
    npcOnly?: boolean
    iskValue?: string   // '5b' | '10b' — single-select
    techLevel?: string  // 't1' | 't2' | 't3' — single-select
}

const props = defineProps<{ modelValue: FilterState }>()
const emit = defineEmits<{ 'update:modelValue': [FilterState] }>()

// Matches the security keys understood by /api/killlist/advanced.
const SECURITY_OPTIONS = [
    { key: 'highsec', label: 'High' },
    { key: 'lowsec', label: 'Low' },
    { key: 'nullsec', label: 'Null' },
    { key: 'wspace', label: 'W-Space' },
    { key: 'pochven', label: 'Pochven' },
    { key: 'abyssal', label: 'Abyssal' },
]

const state = reactive<FilterState>({
    regionId: props.modelValue.regionId,
    regionName: props.modelValue.regionName,
    systemId: props.modelValue.systemId,
    systemName: props.modelValue.systemName,
    shipId: props.modelValue.shipId,
    shipName: props.modelValue.shipName,
    securityTypes: [...(props.modelValue.securityTypes ?? [])],
    soloOnly: props.modelValue.soloOnly,
    npcOnly: props.modelValue.npcOnly,
    iskValue: props.modelValue.iskValue,
    techLevel: props.modelValue.techLevel,
})

function emitState() {
    emit('update:modelValue', {
        regionId: state.regionId,
        regionName: state.regionName,
        systemId: state.systemId,
        systemName: state.systemName,
        shipId: state.shipId,
        shipName: state.shipName,
        securityTypes: [...state.securityTypes],
        soloOnly: state.soloOnly,
        npcOnly: state.npcOnly,
        iskValue: state.iskValue,
        techLevel: state.techLevel,
    })
}

// Shared pill styling for the toggle buttons.
function pillClass(active: boolean) {
    return active
        ? 'bg-blue-500/15 text-blue-400 border border-blue-500/30'
        : 'text-gray-500 hover:text-blue-400 border border-white/[0.08] hover:bg-blue-500/[0.04]'
}

interface Hit { id: string; name: string; type: string }

// One debounced /api/search picker per filterable type. Kept as plain reactive
// objects (not a composable) so we can spin up three identical ones cheaply.
function makePicker(type: 'region' | 'system' | 'ship') {
    const q = ref('')
    const results = ref<Hit[]>([])
    const open = ref(false)
    const run = useDebounceFn(async () => {
        const term = q.value.trim()
        if (term.length < 2) { results.value = []; open.value = false; return }
        try {
            const res = await apiFetch<{ hits: Hit[] }>('/api/search', { query: { type, q: term, limit: 8 } })
            results.value = res.hits ?? []
            open.value = results.value.length > 0
        } catch { results.value = []; open.value = false }
    }, 250)
    const closeSoon = () => setTimeout(() => { open.value = false }, 150)
    return { q, results, open, run, closeSoon }
}

const region = makePicker('region')
const system = makePicker('system')
const ship = makePicker('ship')

// Search hit ids look like "<type>_<numericId>"; the numeric part is everything
// after the first underscore.
function hitId(hit: Hit) { return Number(hit.id.slice(hit.id.indexOf('_') + 1)) }

function selectRegion(hit: Hit) {
    state.regionId = hitId(hit); state.regionName = hit.name
    state.systemId = undefined; state.systemName = undefined
    region.q.value = ''; region.open.value = false
    emitState()
}
function selectSystem(hit: Hit) {
    state.systemId = hitId(hit); state.systemName = hit.name
    state.regionId = undefined; state.regionName = undefined
    system.q.value = ''; system.open.value = false
    emitState()
}
function selectShip(hit: Hit) {
    state.shipId = hitId(hit); state.shipName = hit.name
    ship.q.value = ''; ship.open.value = false
    emitState()
}

function clearRegion() { state.regionId = undefined; state.regionName = undefined; emitState() }
function clearSystem() { state.systemId = undefined; state.systemName = undefined; emitState() }
function clearShip() { state.shipId = undefined; state.shipName = undefined; emitState() }

function toggleSecurity(key: string) {
    const i = state.securityTypes.indexOf(key)
    if (i >= 0) state.securityTypes.splice(i, 1)
    else state.securityTypes.push(key)
    emitState()
}

function toggleSolo() { state.soloOnly = state.soloOnly ? undefined : true; emitState() }
function toggleNpc() { state.npcOnly = state.npcOnly ? undefined : true; emitState() }
// ISK and tech are single-select — clicking the active option clears it.
function setIsk(v: string) { state.iskValue = state.iskValue === v ? undefined : v; emitState() }
function setTech(v: string) { state.techLevel = state.techLevel === v ? undefined : v; emitState() }

function clearAll() {
    state.regionId = undefined; state.regionName = undefined
    state.systemId = undefined; state.systemName = undefined
    state.shipId = undefined; state.shipName = undefined
    state.securityTypes = []
    state.soloOnly = undefined; state.npcOnly = undefined
    state.iskValue = undefined; state.techLevel = undefined
    region.q.value = ''; system.q.value = ''; ship.q.value = ''
    emitState()
}

const hasAny = computed(() => !!(
    state.regionId || state.systemId || state.shipId || state.securityTypes.length
    || state.soloOnly || state.npcOnly || state.iskValue || state.techLevel
))
</script>

<template>
    <div class="rounded-lg bg-white/[0.03] border border-white/[0.08] p-3 mb-2">
        <div class="flex flex-wrap items-start gap-3">
            <!-- Region -->
            <div class="relative min-w-[10rem] flex-1">
                <label class="block text-fine uppercase tracking-wider text-gray-600 mb-1">Region</label>
                <div v-if="state.regionId"
                    class="flex items-center justify-between gap-2 px-2 py-1.5 rounded bg-blue-500/10 border border-blue-500/20 text-xs text-gray-200">
                    <span class="truncate">{{ state.regionName }}</span>
                    <button class="text-gray-500 hover:text-red-400 flex-shrink-0" @click="clearRegion">
                        <Icon name="lucide:x" class="w-3.5 h-3.5" />
                    </button>
                </div>
                <template v-else>
                    <input
                        v-model="region.q.value"
                        type="text"
                        placeholder="Search region…"
                        class="w-full px-2 py-1.5 rounded bg-white/[0.04] border border-white/[0.08] text-xs text-gray-200 placeholder:text-gray-600 focus:outline-none focus:border-blue-500/40"
                        @input="region.run"
                        @focus="region.open.value = region.results.value.length > 0"
                        @blur="region.closeSoon"
                    >
                    <div v-if="region.open.value"
                        class="absolute z-30 mt-1 w-full max-h-56 overflow-y-auto rounded-md bg-[#0f1115] border border-white/[0.12] shadow-xl">
                        <button v-for="hit in region.results.value" :key="hit.id"
                            class="block w-full text-left px-2.5 py-1.5 text-xs text-gray-300 hover:bg-blue-500/15 truncate"
                            @mousedown.prevent="selectRegion(hit)">
                            {{ hit.name }}
                        </button>
                    </div>
                </template>
            </div>

            <!-- System -->
            <div class="relative min-w-[10rem] flex-1">
                <label class="block text-fine uppercase tracking-wider text-gray-600 mb-1">System</label>
                <div v-if="state.systemId"
                    class="flex items-center justify-between gap-2 px-2 py-1.5 rounded bg-blue-500/10 border border-blue-500/20 text-xs text-gray-200">
                    <span class="truncate">{{ state.systemName }}</span>
                    <button class="text-gray-500 hover:text-red-400 flex-shrink-0" @click="clearSystem">
                        <Icon name="lucide:x" class="w-3.5 h-3.5" />
                    </button>
                </div>
                <template v-else>
                    <input
                        v-model="system.q.value"
                        type="text"
                        placeholder="Search system…"
                        class="w-full px-2 py-1.5 rounded bg-white/[0.04] border border-white/[0.08] text-xs text-gray-200 placeholder:text-gray-600 focus:outline-none focus:border-blue-500/40"
                        @input="system.run"
                        @focus="system.open.value = system.results.value.length > 0"
                        @blur="system.closeSoon"
                    >
                    <div v-if="system.open.value"
                        class="absolute z-30 mt-1 w-full max-h-56 overflow-y-auto rounded-md bg-[#0f1115] border border-white/[0.12] shadow-xl">
                        <button v-for="hit in system.results.value" :key="hit.id"
                            class="block w-full text-left px-2.5 py-1.5 text-xs text-gray-300 hover:bg-blue-500/15 truncate"
                            @mousedown.prevent="selectSystem(hit)">
                            {{ hit.name }}
                        </button>
                    </div>
                </template>
            </div>

            <!-- Ship -->
            <div class="relative min-w-[10rem] flex-1">
                <label class="block text-fine uppercase tracking-wider text-gray-600 mb-1">Ship</label>
                <div v-if="state.shipId"
                    class="flex items-center justify-between gap-2 px-2 py-1.5 rounded bg-blue-500/10 border border-blue-500/20 text-xs text-gray-200">
                    <span class="flex items-center gap-1.5 min-w-0">
                        <img :src="`/images/types/${state.shipId}/icon?size=32`" alt="" class="w-4 h-4 rounded flex-shrink-0">
                        <span class="truncate">{{ state.shipName }}</span>
                    </span>
                    <button class="text-gray-500 hover:text-red-400 flex-shrink-0" @click="clearShip">
                        <Icon name="lucide:x" class="w-3.5 h-3.5" />
                    </button>
                </div>
                <template v-else>
                    <input
                        v-model="ship.q.value"
                        type="text"
                        placeholder="Search ship…"
                        class="w-full px-2 py-1.5 rounded bg-white/[0.04] border border-white/[0.08] text-xs text-gray-200 placeholder:text-gray-600 focus:outline-none focus:border-blue-500/40"
                        @input="ship.run"
                        @focus="ship.open.value = ship.results.value.length > 0"
                        @blur="ship.closeSoon"
                    >
                    <div v-if="ship.open.value"
                        class="absolute z-30 mt-1 w-full max-h-56 overflow-y-auto rounded-md bg-[#0f1115] border border-white/[0.12] shadow-xl">
                        <button v-for="hit in ship.results.value" :key="hit.id"
                            class="flex items-center gap-2 w-full text-left px-2.5 py-1.5 text-xs text-gray-300 hover:bg-blue-500/15 truncate"
                            @mousedown.prevent="selectShip(hit)">
                            <img :src="`/images/types/${hitId(hit)}/icon?size=32`" alt="" class="w-4 h-4 rounded flex-shrink-0">
                            <span class="truncate">{{ hit.name }}</span>
                        </button>
                    </div>
                </template>
            </div>
        </div>

        <!-- Toggle filters — single row: security space, kind, value, tech -->
        <div class="flex flex-wrap items-center gap-1.5 mt-3">
            <button v-for="opt in SECURITY_OPTIONS" :key="opt.key"
                class="px-2 py-1 rounded text-xs font-medium transition-colors"
                :class="pillClass(state.securityTypes.includes(opt.key))"
                @click="toggleSecurity(opt.key)">
                {{ opt.label }}
            </button>

            <span class="w-px h-4 bg-white/[0.12] mx-1"></span>

            <button class="px-2 py-1 rounded text-xs font-medium transition-colors"
                :class="pillClass(!!state.soloOnly)" @click="toggleSolo">Solo</button>
            <button class="px-2 py-1 rounded text-xs font-medium transition-colors"
                :class="pillClass(!!state.npcOnly)" @click="toggleNpc">NPC</button>
            <button class="px-2 py-1 rounded text-xs font-medium transition-colors"
                :class="pillClass(state.iskValue === '5b')" @click="setIsk('5b')">+5b</button>
            <button class="px-2 py-1 rounded text-xs font-medium transition-colors"
                :class="pillClass(state.iskValue === '10b')" @click="setIsk('10b')">+10b</button>
            <button class="px-2 py-1 rounded text-xs font-medium transition-colors"
                :class="pillClass(state.techLevel === 't1')" @click="setTech('t1')">T1</button>
            <button class="px-2 py-1 rounded text-xs font-medium transition-colors"
                :class="pillClass(state.techLevel === 't2')" @click="setTech('t2')">T2</button>
            <button class="px-2 py-1 rounded text-xs font-medium transition-colors"
                :class="pillClass(state.techLevel === 't3')" @click="setTech('t3')">T3</button>

            <button v-if="hasAny"
                class="ml-auto text-fine text-gray-500 hover:text-red-400 flex items-center gap-1"
                @click="clearAll">
                <Icon name="lucide:x" class="w-3.5 h-3.5" /> Clear all
            </button>
        </div>
    </div>
</template>
