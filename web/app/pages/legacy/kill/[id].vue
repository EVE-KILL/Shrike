<script setup lang="ts">
const route = useRoute()
const id = computed(() => Number(route.params.id))

const { data, error } = await useApiFetch<{
    kill: any
    attackers: any[]
    items: any[]
}>(`/api/legacy/kill/${id.value}`)

if (error.value || !data.value?.kill) {
    throw createError({ statusCode: 404, statusMessage: 'Killmail not found' })
}

const kill = computed(() => data.value!.kill)
const attackers = computed(() => data.value!.attackers)
const items = computed(() => data.value!.items)
const finalBlow = computed(() => attackers.value.find((a: any) => a.final_blow) ?? attackers.value[0] ?? null)

const killTitle = computed(() => {
    const who = kill.value.victim_name || 'Unknown'
    const ship = kill.value.victim_ship || 'Unknown'
    const isk = formatIsk(kill.value.total_value || 0)
    return `${ship} | ${who} | ${isk} ISK`
})

useHead({ title: killTitle })
useSeoMeta({
    description: computed(() => `${kill.value.victim_name} lost a ${kill.value.victim_ship} in ${kill.value.system_name}`),
    ogTitle: computed(() => `${killTitle.value} — EVE-KILL Legacy`),
    ogImage: computed(() => kill.value.victim_ship_type_id ? `/images/types/${kill.value.victim_ship_type_id}/overlayrender?size=512` : ''),
})

const secColor = (sec: number | null): string => {
    if (sec === null) return 'text-gray-500'
    if (sec >= 0.5) return 'text-blue-400'
    if (sec > 0.0) return 'text-amber-400'
    return 'text-red-400'
}

const formatTimestamp = (dateStr: string): string => {
    const d = new Date(dateStr)
    const day = d.getUTCDate()
    const month = d.toLocaleString('en-US', { month: 'short', timeZone: 'UTC' })
    const year = d.getUTCFullYear()
    const hh = String(d.getUTCHours()).padStart(2, '0')
    const mm = String(d.getUTCMinutes()).padStart(2, '0')
    return `${day} ${month} ${year}, ${hh}:${mm} EVE`
}

function img(type: string, id: number | null | undefined, variant: string, size: number): string {
    if (id == null || id === 0) {
        if (type === 'types') return `/images/types/670/${variant}?size=${size}`
        if (type === 'characters') return `/images/characters/1/portrait?size=${size}`
        return ''
    }
    return `/images/${type}/${id}/${variant}?size=${size}`
}

// Group items by location
const fittedItems = computed(() => items.value.filter((i: any) => !i.location))
const cargoItems = computed(() => items.value.filter((i: any) => i.location === 'Cargo'))
const droneBayItems = computed(() => items.value.filter((i: any) => i.location === 'Drone Bay'))

const slotGroups = computed(() => {
    const groups: { label: string; items: any[] }[] = []
    if (fittedItems.value.length) groups.push({ label: 'Fitted', items: fittedItems.value })
    if (cargoItems.value.length) groups.push({ label: 'Cargo', items: cargoItems.value })
    if (droneBayItems.value.length) groups.push({ label: 'Drone Bay', items: droneBayItems.value })
    return groups
})

// Attacker pagination
const attackerLimit = ref(10)
const visibleAttackers = computed(() => attackers.value.slice(0, attackerLimit.value))
const hasMoreAttackers = computed(() => attackers.value.length > attackerLimit.value)
const showMoreAttackers = () => { attackerLimit.value += 25 }

// Attacker hover
const hoveredCell = ref<{ idx: number; cell: string } | null>(null)
const setHover = (idx: number, cell: string) => { hoveredCell.value = { idx, cell } }
const clearHover = () => { hoveredCell.value = null }
const isLit = (idx: number, cell: string) => hoveredCell.value?.idx === idx && hoveredCell.value?.cell === cell
</script>

<template>
    <div v-if="kill">
        <h1 class="sr-only">{{ killTitle }}</h1>

        <!-- ===== DESKTOP ===== -->
        <div class="hidden md:grid grid-cols-[1fr_minmax(0,374px)] gap-4">
            <!-- LEFT: Fitting wheel + Info + Items -->
            <div class="space-y-4">
                <div class="flex gap-4 items-start">
                    <!-- Fitting wheel (empty — just ship render) -->
                    <div class="flex-1 min-w-[350px] max-w-[650px]">
                        <KillFittingWheel :ship-type-id="kill.victim_ship_type_id || 670" :ship-name="kill.victim_ship" :items="[]" />
                    </div>

                    <!-- Info box -->
                    <div class="flex-shrink-0 w-[252px]">
                        <div class="rounded-xl border border-white/[0.08] overflow-hidden">
                            <!-- Portrait -->
                            <div class="relative">
                                <div class="w-full aspect-square bg-white/[0.04]">
                                    <img :src="img('characters', kill.victim_character_id, 'portrait', 512)" :alt="kill.victim_name || ''" class="w-full h-full object-cover">
                                </div>
                                <div class="absolute inset-0 bg-gradient-to-t from-black/90 via-transparent to-transparent pointer-events-none"></div>
                                <!-- Corp + Alliance logos -->
                                <div class="absolute bottom-2 right-2 flex gap-1 z-10">
                                    <div v-if="kill.victim_corporation_id" class="w-10 h-10 rounded-lg overflow-hidden bg-black/60 border border-white/[0.1]">
                                        <img :src="img('corporations', kill.victim_corporation_id, 'logo', 64)" :alt="kill.victim_corp || ''" class="w-full h-full">
                                    </div>
                                    <div v-if="kill.victim_alliance_id" class="w-10 h-10 rounded-lg overflow-hidden bg-black/60 border border-white/[0.1]">
                                        <img :src="img('alliances', kill.victim_alliance_id, 'logo', 64)" :alt="kill.victim_alliance || ''" class="w-full h-full">
                                    </div>
                                </div>
                                <!-- Name overlay -->
                                <div class="absolute inset-x-0 bottom-0 p-3 pr-24 space-y-0.5 z-10">
                                    <div class="text-sm text-white font-semibold drop-shadow-lg truncate">{{ kill.victim_name || 'Unknown' }}</div>
                                    <div v-if="kill.victim_corp" class="text-xs text-gray-300/80 drop-shadow truncate">{{ kill.victim_corp }}</div>
                                    <div v-if="kill.victim_alliance" class="text-fine text-gray-400/80 drop-shadow truncate">{{ kill.victim_alliance }}</div>
                                </div>
                            </div>
                            <!-- Stats -->
                            <div class="bg-white/[0.04] px-4 py-3 space-y-2 text-xs">
                                <div class="flex justify-between"><span class="text-gray-500">Ship</span><span class="text-gray-300">{{ kill.victim_ship || 'Unknown' }}</span></div>
                                <div class="flex justify-between items-center"><span class="text-gray-500">System</span><span><span :class="secColor(kill.security)" class="font-medium tabular-nums">{{ kill.security?.toFixed(1) }}</span><span class="text-gray-300 ml-1">{{ kill.system_name }}</span></span></div>
                                <div class="flex justify-between"><span class="text-gray-500">Time</span><span class="text-gray-300">{{ formatTimestamp(kill.killmail_time) }}</span></div>
                                <div class="flex justify-between"><span class="text-gray-500">Attackers</span><span class="text-gray-300 tabular-nums">{{ attackers.length }}</span></div>
                            </div>
                            <div class="border-t border-white/[0.08] bg-white/[0.04] px-4 py-3 text-xs">
                                <div class="flex justify-between">
                                    <span class="text-gray-400 font-medium">Total Value</span>
                                    <span class="text-white font-semibold tabular-nums">{{ formatIsk(kill.total_value || 0) }} ISK</span>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Items table -->
                <div v-if="items.length > 0" class="glass-panel p-2">
                    <div class="grid grid-cols-[28px_1fr_50px] gap-2 px-2 py-1.5 text-fine font-bold uppercase tracking-wider text-gray-600 border-b border-white/[0.08]">
                        <div></div>
                        <div>Item</div>
                        <div class="text-center">Qty</div>
                    </div>

                    <!-- Ship hull -->
                    <div class="grid grid-cols-[28px_1fr_50px] gap-2 px-2 py-1.5 items-center bg-red-500/[0.05] border-b border-white/[0.04]">
                        <div class="w-6 h-6 rounded overflow-hidden bg-white/[0.04]">
                            <img :src="img('types', kill.victim_ship_type_id, 'icon', 64)" :alt="kill.victim_ship" class="w-full h-full object-cover">
                        </div>
                        <div class="text-xs text-red-300/80 font-medium truncate">{{ kill.victim_ship || 'Unknown Ship' }}</div>
                        <div class="text-center"><span class="px-1.5 py-0.5 rounded text-fine font-medium bg-red-500/10 text-red-400">1</span></div>
                    </div>

                    <template v-for="group in slotGroups" :key="group.label">
                        <div class="grid grid-cols-[28px_1fr_50px] gap-2 px-2 py-1.5 mt-1 bg-white/[0.03] border-b border-white/[0.04]">
                            <div></div>
                            <div class="text-xs font-bold uppercase tracking-wider text-gray-400">
                                {{ group.label }} <span class="text-gray-600 font-normal normal-case ml-1">({{ group.items.length }})</span>
                            </div>
                            <div></div>
                        </div>
                        <div v-for="item in group.items" :key="item.id"
                            class="grid grid-cols-[28px_1fr_50px] gap-2 px-2 py-1 items-center bg-red-500/[0.05] border-b border-white/[0.02]">
                            <div class="w-6 h-6 rounded overflow-hidden bg-white/[0.04]">
                                <img :src="img('types', item.type_id, 'icon', 64)" :alt="item.name" class="w-full h-full object-cover" loading="lazy">
                            </div>
                            <div class="text-xs text-red-300/80 truncate">{{ item.name }}</div>
                            <div class="text-center">
                                <span class="px-1.5 py-0.5 rounded text-fine font-medium bg-red-500/10 text-red-400">{{ item.quantity }}</span>
                            </div>
                        </div>
                    </template>
                </div>

                <!-- Raw killmail text -->
                <details>
                    <summary class="text-sm text-gray-500 hover:text-blue-400 cursor-pointer transition-colors">View original killmail text</summary>
                    <pre class="mt-2 p-4 rounded-lg bg-white/[0.02] border border-white/[0.06] text-xs text-gray-400 whitespace-pre-wrap font-mono overflow-x-auto">{{ kill.raw_text }}</pre>
                </details>
            </div>

            <!-- RIGHT: Featured Attackers + All Attackers -->
            <div class="space-y-4">
                <div class="glass-panel p-3">
                    <!-- Final Blow featured -->
                    <div v-if="finalBlow" class="relative rounded-xl border border-amber-500/20 overflow-hidden group mb-3">
                        <div class="relative">
                            <img v-if="finalBlow.ship_type_id"
                                :src="img('types', finalBlow.ship_type_id, 'overlayrender', 512)"
                                :alt="finalBlow.ship || 'Ship'"
                                class="w-full h-auto object-contain transition-transform duration-300 group-hover:scale-110"
                                loading="lazy">
                            <div v-else class="w-full aspect-square bg-white/[0.04]"></div>
                            <div class="absolute inset-0 bg-gradient-to-t from-black/90 via-black/30 to-transparent"></div>
                            <div class="absolute top-2 left-2 px-2 py-0.5 rounded bg-amber-500/20 backdrop-blur-sm">
                                <span class="text-fine font-bold uppercase tracking-[0.15em] text-amber-400">Final Blow</span>
                            </div>
                            <div v-if="finalBlow.character_id" class="absolute top-2 right-2 w-16 h-16 rounded-lg overflow-hidden border border-white/[0.1]">
                                <img :src="img('characters', finalBlow.character_id, 'portrait', 128)" :alt="finalBlow.name" class="w-full h-full object-cover">
                            </div>
                            <div class="absolute inset-x-0 bottom-0 p-2.5 space-y-0.5">
                                <div class="text-sm text-white font-semibold drop-shadow-lg truncate">{{ finalBlow.name || 'Unknown' }}</div>
                                <div class="text-fine drop-shadow truncate text-gray-300/80">{{ finalBlow.ship || '' }} <template v-if="finalBlow.weapon">— {{ finalBlow.weapon }}</template></div>
                                <div v-if="finalBlow.corp" class="text-fine text-gray-400/80 drop-shadow truncate">{{ finalBlow.corp }}</div>
                            </div>
                        </div>
                    </div>

                    <!-- All Attackers -->
                    <div class="px-2 pb-2 mb-2 text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 border-b border-white/[0.08]">
                        Attackers ({{ attackers.length }})
                    </div>
                    <div class="space-y-1">
                        <div v-for="(atk, idx) in visibleAttackers" :key="idx"
                            class="rounded-lg transition-colors"
                            :class="atk.final_blow ? 'ring-1 ring-amber-500/30' : ''">
                            <div class="flex">
                                <!-- Portrait -->
                                <div class="relative w-12 h-12 bg-white/[0.04] overflow-visible" @mouseenter="setHover(idx, 'char')" @mouseleave="clearHover">
                                    <img :src="img('characters', atk.character_id, 'portrait', 128)" :alt="atk.name || ''"
                                        class="absolute inset-0 w-full h-full object-cover transition-all duration-150 rounded-sm"
                                        :class="isLit(idx, 'char') ? 'scale-115 z-20 shadow-lg shadow-black/50' : ''" loading="lazy">
                                </div>
                                <!-- Corp logo -->
                                <div v-if="atk.corporation_id" class="relative w-12 h-12 bg-white/[0.04] overflow-visible" @mouseenter="setHover(idx, 'corp')" @mouseleave="clearHover">
                                    <img :src="img('corporations', atk.corporation_id, 'logo', 128)" :alt="atk.corp || ''"
                                        class="absolute inset-0 w-full h-full object-cover transition-all duration-150 rounded-sm"
                                        :class="isLit(idx, 'corp') ? 'scale-115 z-20 shadow-lg shadow-black/50' : ''" loading="lazy">
                                </div>
                                <!-- Alliance logo -->
                                <div v-if="atk.alliance_id" class="relative w-12 h-12 bg-white/[0.04] overflow-visible" @mouseenter="setHover(idx, 'alli')" @mouseleave="clearHover">
                                    <img :src="img('alliances', atk.alliance_id, 'logo', 128)" :alt="atk.alliance || ''"
                                        class="absolute inset-0 w-full h-full object-cover transition-all duration-150 rounded-sm"
                                        :class="isLit(idx, 'alli') ? 'scale-115 z-20 shadow-lg shadow-black/50' : ''" loading="lazy">
                                </div>
                                <div v-else class="w-12 h-12 bg-white/[0.04]"></div>
                                <!-- Ship -->
                                <div v-if="atk.ship_type_id" class="relative w-12 h-12 bg-white/[0.04] overflow-visible" @mouseenter="setHover(idx, 'ship')" @mouseleave="clearHover">
                                    <img :src="img('types', atk.ship_type_id, 'icon', 128)" :alt="atk.ship || ''"
                                        class="absolute inset-0 w-full h-full object-cover transition-all duration-150 rounded-sm"
                                        :class="isLit(idx, 'ship') ? 'scale-115 z-20 shadow-lg shadow-black/50' : ''" loading="lazy">
                                </div>
                                <div v-else class="w-12 h-12 bg-white/[0.04]"></div>
                                <!-- Weapon -->
                                <div v-if="atk.weapon_type_id" class="relative w-12 h-12 bg-white/[0.04] overflow-visible" @mouseenter="setHover(idx, 'wpn')" @mouseleave="clearHover">
                                    <img :src="img('types', atk.weapon_type_id, 'icon', 128)" :alt="atk.weapon || ''"
                                        class="absolute inset-0 w-full h-full object-cover transition-all duration-150 rounded-sm"
                                        :class="isLit(idx, 'wpn') ? 'scale-115 z-20 shadow-lg shadow-black/50' : ''" loading="lazy">
                                </div>
                                <div v-else class="w-12 h-12 bg-white/[0.04]"></div>
                            </div>
                            <!-- Labels -->
                            <div class="flex text-fine text-gray-500 truncate px-0.5 pb-1">
                                <div class="w-12 truncate text-center" :class="isLit(idx, 'char') ? 'text-blue-400' : ''">{{ atk.name || '?' }}</div>
                                <div v-if="atk.corporation_id" class="w-12 truncate text-center" :class="isLit(idx, 'corp') ? 'text-blue-400' : ''">{{ atk.corp || '' }}</div>
                                <div v-if="atk.alliance_id" class="w-12 truncate text-center" :class="isLit(idx, 'alli') ? 'text-blue-400' : ''">{{ atk.alliance || '' }}</div>
                                <div v-else class="w-12"></div>
                                <div class="w-12 truncate text-center" :class="isLit(idx, 'ship') ? 'text-blue-400' : ''">{{ atk.ship || '' }}</div>
                                <div class="w-12 truncate text-center" :class="isLit(idx, 'wpn') ? 'text-blue-400' : ''">{{ atk.weapon || '' }}</div>
                            </div>
                        </div>
                    </div>
                    <button v-if="hasMoreAttackers" @click="showMoreAttackers"
                        class="w-full mt-2 py-2 text-xs text-gray-500 hover:text-blue-400 transition-colors text-center">
                        Show more ({{ attackers.length - attackerLimit }} remaining)
                    </button>
                </div>
            </div>
        </div>

        <!-- ===== MOBILE ===== -->
        <div class="md:hidden space-y-4">
            <!-- Ship render + info -->
            <div class="rounded-xl border border-white/[0.08] overflow-hidden">
                <div class="relative">
                    <div class="w-full aspect-square bg-white/[0.04]">
                        <img :src="img('characters', kill.victim_character_id, 'portrait', 512)" :alt="kill.victim_name" class="w-full h-full object-cover">
                    </div>
                    <div class="absolute inset-0 bg-gradient-to-t from-black/90 via-transparent to-transparent"></div>
                    <div class="absolute inset-x-0 bottom-0 p-3 space-y-0.5">
                        <div class="text-lg text-white font-semibold drop-shadow-lg truncate">{{ kill.victim_name || 'Unknown' }}</div>
                        <div class="text-sm text-gray-300/80 drop-shadow truncate">{{ kill.victim_ship || 'Unknown Ship' }}</div>
                        <div v-if="kill.victim_corp" class="text-xs text-gray-400/80 drop-shadow truncate">{{ kill.victim_corp }}</div>
                    </div>
                </div>
                <div class="bg-white/[0.04] px-4 py-3 space-y-2 text-xs">
                    <div class="flex justify-between"><span class="text-gray-500">System</span><span><span :class="secColor(kill.security)" class="tabular-nums">{{ kill.security?.toFixed(1) }}</span> {{ kill.system_name }}</span></div>
                    <div class="flex justify-between"><span class="text-gray-500">Time</span><span class="text-gray-300">{{ formatTimestamp(kill.killmail_time) }}</span></div>
                    <div class="flex justify-between"><span class="text-gray-500">Attackers</span><span class="text-gray-300">{{ attackers.length }}</span></div>
                    <div class="flex justify-between"><span class="text-gray-400 font-medium">Total Value</span><span class="text-white font-semibold tabular-nums">{{ formatIsk(kill.total_value || 0) }} ISK</span></div>
                </div>
            </div>

            <!-- Attackers -->
            <div class="glass-panel p-3">
                <div class="text-fine font-bold uppercase tracking-wider text-blue-400/80 border-b border-white/[0.08] pb-2 mb-2">Attackers ({{ attackers.length }})</div>
                <div v-for="atk in attackers" :key="atk.id" class="flex items-center gap-2 py-1.5 border-b border-white/[0.03]" :class="atk.final_blow ? 'bg-amber-500/[0.05]' : ''">
                    <img :src="img('characters', atk.character_id, 'portrait', 64)" class="w-8 h-8 rounded flex-shrink-0">
                    <div class="flex-1 min-w-0">
                        <div class="text-xs text-gray-300 truncate">{{ atk.name || 'Unknown' }} <span v-if="atk.final_blow" class="text-amber-400">(FB)</span></div>
                        <div class="text-fine text-gray-500 truncate">{{ atk.ship || '' }} — {{ atk.weapon || '' }}</div>
                    </div>
                </div>
            </div>

            <!-- Items -->
            <div v-if="items.length" class="glass-panel p-3">
                <div class="text-fine font-bold uppercase tracking-wider text-blue-400/80 border-b border-white/[0.08] pb-2 mb-2">Destroyed Items ({{ items.length }})</div>
                <div v-for="item in items" :key="item.id" class="flex items-center gap-2 py-1 border-b border-white/[0.03]">
                    <img :src="img('types', item.type_id, 'icon', 64)" class="w-6 h-6 rounded flex-shrink-0" loading="lazy">
                    <span class="text-xs text-gray-300 truncate flex-1">{{ item.name }}</span>
                    <span v-if="item.quantity > 1" class="text-fine text-gray-500 tabular-nums">x{{ item.quantity }}</span>
                </div>
            </div>

            <!-- Raw text -->
            <details>
                <summary class="text-sm text-gray-500 hover:text-blue-400 cursor-pointer">View original killmail text</summary>
                <pre class="mt-2 p-3 rounded-lg bg-white/[0.02] border border-white/[0.06] text-xs text-gray-400 whitespace-pre-wrap font-mono overflow-x-auto">{{ kill.raw_text }}</pre>
            </details>
        </div>
    </div>
</template>
