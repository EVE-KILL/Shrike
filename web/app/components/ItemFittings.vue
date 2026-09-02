<script setup lang="ts">
// Item Fittings tab — renders the most popular fits for a ship over the last
// 90 days. Data comes from /api/item/:id/fittings which groups by family_hash
// (T1/T2/meta variants of the same fit collapse into one row).
//
// See backend/src/services/FittingExtractor.ts for how the slot_group bucketing
// works. Visual order is high → med → low → rig → subsystem.
import { killmailFitToEditorUrl } from "~/composables/fit/killmailToFit"

const props = defineProps<{
    shipTypeId: number
    hullGroupName?: string | null
}>()

interface FittingModule {
    slot_group: number
    ordinal: number
    type_id: number
    name: string | null
    charge_type_id: number | null
    charge_name: string | null
}

interface FittingDrone {
    type_id: number
    name: string | null
    quantity: number
}

interface AllianceUsage {
    alliance_id: number
    name: string | null
    uses: number
    pct_of_alliance_losses: number
}

interface FittingFamily {
    family_hash: string
    canonical_fit_hash: string
    total_uses: number
    canonical_uses: number
    variant_count: number
    last_used: string
    fit_cost: number
    modules: FittingModule[]
    drones: FittingDrone[]
    top_alliances: AllianceUsage[]
    context?: FitFamilyContext
}

interface FittingsResponse {
    ship_type_id: number
    window_days: number
    is_rare_hull: boolean
    hull_cost: number | null
    families: FittingFamily[]
}

const { data, pending, error } = await useApiFetch<FittingsResponse>(
    () => `/api/item/${props.shipTypeId}/fittings`,
)

// Module group popularity — "97% use prop mods, 82% webs" etc.
interface FitMetaGroup {
    group_id: number
    name: string
    kill_count: number
    pct: number
}
interface FitMetaResponse {
    ship_type_id: number
    window_days: number
    total_kills: number
    groups: FitMetaGroup[]
}
const { data: metaData } = useApiFetch<FitMetaResponse>(
    () => `/api/item/${props.shipTypeId}/fit-meta`,
    { lazy: true, server: false },
)

type FitSort = 'observed' | 'recent' | 'cheapest' | 'expensive'
const fitSort = ref<FitSort>('observed')
const expandedFamily = ref<string | null>(null)

const sortedFamilies = computed(() => {
    const families = [...(data.value?.families ?? [])]
    switch (fitSort.value) {
        case 'recent':
            return families.sort((a, b) => new Date(b.last_used).getTime() - new Date(a.last_used).getTime())
        case 'cheapest':
            return families.sort((a, b) => a.fit_cost - b.fit_cost)
        case 'expensive':
            return families.sort((a, b) => b.fit_cost - a.fit_cost)
        default:
            return families.sort((a, b) => b.total_uses - a.total_uses)
    }
})

watch(sortedFamilies, (families) => {
    if (expandedFamily.value === null && families[0]) {
        expandedFamily.value = families[0].family_hash
    }
}, { immediate: true })

const toggleFamily = (familyHash: string) => {
    expandedFamily.value = expandedFamily.value === familyHash ? null : familyHash
}

const lastObserved = (iso: string): string => {
    const days = Math.max(0, Math.floor((Date.now() - new Date(iso).getTime()) / 86_400_000))
    if (days === 0) return 'today'
    if (days === 1) return 'yesterday'
    return `${days}d ago`
}

// Slot group metadata — matches the extractor's numbering.
const SLOT_LABELS: Record<number, string> = {
    1: 'High',
    2: 'Med',
    3: 'Low',
    4: 'Rig',
    5: 'Subsystem',
}
const SLOT_ORDER = [1, 2, 3, 4, 5] as const

function groupBySlot(modules: FittingModule[]): Record<number, FittingModule[]> {
    const result: Record<number, FittingModule[]> = {}
    for (const m of modules) {
        if (!result[m.slot_group]) result[m.slot_group] = []
        result[m.slot_group]!.push(m)
    }
    return result
}

/**
 * Convert the killmail-derived family into a UI Fit, encode it, and
 * navigate to the editor. Heavy lifting lives in killmailFitToEditorUrl
 * so this function only adds the editor-friendly description.
 */
async function loadIntoEditor(family: FittingFamily) {
    const url = await killmailFitToEditorUrl({
        shipTypeId: props.shipTypeId,
        modules: family.modules,
        drones: family.drones,
        name: 'Community Fit',
        description: `Loaded from /fits — ${family.total_uses} recorded use${family.total_uses === 1 ? '' : 's'} in the last ${data.value?.window_days ?? 90} days.`,
    })
    await navigateTo(url)
}
</script>

<template>
    <div>
        <div v-if="pending" class="py-12 text-center text-gray-500 text-sm">
            Loading fits…
        </div>

        <div v-else-if="error" class="py-12 text-center text-red-400 text-sm">
            Failed to load fittings.
        </div>

        <div v-else-if="!data || data.families.length === 0" class="py-12 text-center">
            <Icon name="lucide:wrench" class="w-12 h-12 mx-auto text-gray-700 mb-3" />
            <p class="text-gray-500 text-sm">
                No fits recorded for this ship in the last {{ data?.window_days ?? 90 }} days.
            </p>
            <p class="text-gray-600 text-xs mt-2">
                We extract fits from killmails — if a ship isn't losing any, we have nothing to learn from.
            </p>
        </div>

        <div v-else>
            <!-- Rare hull callout -->
            <div v-if="data.is_rare_hull"
                class="mb-4 rounded-lg border border-amber-500/30 bg-amber-500/5 p-3 text-xs text-amber-300 flex items-start gap-2">
                <Icon name="lucide:sparkles" class="w-4 h-4 flex-shrink-0 mt-0.5" />
                <div>
                    <div class="font-semibold mb-0.5">Rare hull</div>
                    <div class="text-amber-200/80">
                        Sample size is small, so we're showing every fit we've seen — even the one-offs.
                    </div>
                </div>
            </div>

            <!-- Module meta breakdown -->
            <div v-if="metaData && metaData.groups.length > 0"
                 class="mb-4 rounded-lg border border-white/[0.08] bg-white/[0.02] p-3">
                <div class="flex items-baseline justify-between mb-2">
                    <span class="text-xs font-semibold text-gray-400 uppercase tracking-wide">Module Usage</span>
                    <span class="text-fine text-gray-600">{{ metaData.total_kills.toLocaleString('en-US') }} kills in {{ metaData.window_days }}d</span>
                </div>
                <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-x-4 gap-y-1">
                    <div v-for="g in metaData.groups" :key="g.group_id" class="flex items-center gap-2">
                        <span class="text-xs text-gray-400 truncate w-32 flex-shrink-0 text-right">{{ g.name }}</span>
                        <div class="flex-1 h-3 rounded-full bg-white/[0.04] overflow-hidden">
                            <div class="h-full rounded-full bg-blue-500/40" :style="{ width: `${g.pct}%` }" />
                        </div>
                        <span class="text-fine text-gray-500 tabular-nums w-10 text-right flex-shrink-0">{{ g.pct }}%</span>
                    </div>
                </div>
            </div>

            <!-- Header row -->
            <div class="flex items-center justify-between gap-3 mb-3">
                <h2 class="text-sm font-semibold text-gray-300">
                    Popular Fits
                    <span class="text-gray-600 font-normal ml-2">(last {{ data.window_days }} days)</span>
                </h2>
                <div class="flex items-center gap-2">
                    <span class="text-xs text-gray-600 hidden sm:inline">
                        {{ data.families.length }} {{ data.families.length === 1 ? 'family' : 'families' }}
                    </span>
                    <label class="relative">
                        <span class="sr-only">Sort fitting families</span>
                        <select v-model="fitSort"
                            class="appearance-none rounded-md border border-white/[0.08] bg-black/30 py-1.5 pl-3 pr-8 text-xs text-gray-300 outline-none transition-colors hover:border-blue-500/30 focus:border-blue-500/40">
                            <option value="observed">Most Observed</option>
                            <option value="recent">Recently Seen</option>
                            <option value="cheapest">Cheapest</option>
                            <option value="expensive">Most Expensive</option>
                        </select>
                        <Icon name="lucide:chevron-down" class="pointer-events-none absolute right-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-gray-600" />
                    </label>
                </div>
            </div>

            <!-- Fit cards -->
            <div class="space-y-3">
                <div v-for="(family, index) in sortedFamilies" :key="family.family_hash"
                    class="rounded-lg border bg-white/[0.025] overflow-hidden transition-colors"
                    :class="expandedFamily === family.family_hash ? 'border-blue-500/25' : 'border-white/[0.08] hover:border-white/[0.14]'">
                    <!-- Family header -->
                    <div class="flex items-stretch border-b border-white/[0.06] bg-white/[0.02]">
                        <button type="button"
                            class="flex min-w-0 flex-1 items-center gap-3 px-3 py-2.5 text-left transition-colors hover:bg-blue-500/[0.035] sm:px-4"
                            :aria-expanded="expandedFamily === family.family_hash"
                            @click="toggleFamily(family.family_hash)">
                            <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md border border-white/[0.07] bg-black/20 text-fine font-bold tabular-nums text-gray-600">
                                {{ String(index + 1).padStart(2, '0') }}
                            </span>
                            <Icon name="lucide:chevron-right"
                                class="h-4 w-4 shrink-0 text-gray-600 transition-transform duration-200"
                                :class="expandedFamily === family.family_hash ? 'rotate-90 text-blue-400' : ''" />
                            <div class="min-w-0 flex-1">
                                <div class="truncate text-sm font-semibold text-gray-200">
                                    {{ classifyFitFamily(family.modules, family.drones, { hullGroupName: props.hullGroupName }) }}
                                </div>
                                <div class="mt-0.5 flex flex-wrap items-center gap-x-5 gap-y-1">
                                    <div>
                                        <span class="text-base font-bold text-white tabular-nums">{{ family.total_uses }}</span>
                                        <span class="ml-1.5 text-xs text-gray-500">{{ family.total_uses === 1 ? 'loss' : 'losses' }}</span>
                                    </div>
                                    <div class="text-xs text-gray-500">
                                        <span class="font-medium text-gray-300 tabular-nums">{{ family.variant_count }}</span>
                                        variant{{ family.variant_count === 1 ? '' : 's' }}
                                    </div>
                                    <div class="hidden text-xs text-gray-500 md:block">
                                        Seen <span class="text-gray-300">{{ lastObserved(family.last_used) }}</span>
                                    </div>
                                    <div v-if="family.fit_cost > 0" class="text-xs text-gray-500 tabular-nums">
                                        <span class="font-semibold text-yellow-400">{{ formatIsk(family.fit_cost + (data?.hull_cost ?? 0)) }}</span>
                                        ISK total
                                    </div>
                                </div>
                                <div v-if="fitFamilyContextParts(family.context).length"
                                    class="mt-0.5 w-full truncate text-fine text-gray-600"
                                    :title="fitFamilyContextParts(family.context).join(' · ')">
                                    {{ fitFamilyContextParts(family.context).join(' · ') }}
                                </div>
                            </div>
                        </button>
                        <button type="button"
                            class="m-2 ml-0 inline-flex shrink-0 items-center gap-1.5 rounded-md border border-blue-500/30 bg-blue-500/15 px-2.5 text-fine font-bold uppercase tracking-[0.1em] text-blue-400 transition-colors hover:bg-blue-500/25 sm:px-3"
                            @click="loadIntoEditor(family)">
                            <Icon name="lucide:square-pen" class="w-3.5 h-3.5" />
                            <span class="hidden sm:inline">Load in Editor</span>
                        </button>
                    </div>

                    <!-- Slot grid -->
                    <div v-if="expandedFamily === family.family_hash"
                        class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-0 divide-y md:divide-y-0 md:divide-x divide-white/[0.06]">
                        <div v-for="slot in SLOT_ORDER" :key="slot" class="p-3">
                            <div class="text-fine uppercase tracking-wider text-gray-600 mb-2 flex items-center gap-1.5">
                                {{ SLOT_LABELS[slot] }}
                                <span v-if="groupBySlot(family.modules)[slot]?.length"
                                    class="text-gray-700">·&nbsp;{{ groupBySlot(family.modules)[slot]!.length }}</span>
                            </div>
                            <ul v-if="groupBySlot(family.modules)[slot]?.length" class="space-y-1">
                                <li v-for="mod in groupBySlot(family.modules)[slot]"
                                    :key="`${slot}-${mod.ordinal}`"
                                    class="text-xs">
                                    <div class="flex items-center gap-2">
                                        <img :src="`/images/types/${mod.type_id}/icon?size=32`"
                                            :alt="mod.name ?? ''"
                                            class="w-5 h-5 rounded flex-shrink-0" loading="lazy">
                                        <NuxtLink :to="`/item/${mod.type_id}`"
                                            class="text-gray-300 hover:text-blue-400 transition-colors truncate">
                                            {{ mod.name ?? `Type ${mod.type_id}` }}
                                        </NuxtLink>
                                    </div>
                                    <!-- Charge / loaded ammo, when paired by the extractor.
                                         Indented under the parent module. -->
                                    <div v-if="mod.charge_type_id"
                                        class="flex items-center gap-2 mt-0.5 ml-7 text-fine text-gray-500">
                                        <img :src="`/images/types/${mod.charge_type_id}/icon?size=32`"
                                            :alt="mod.charge_name ?? ''"
                                            class="w-3.5 h-3.5 rounded flex-shrink-0" loading="lazy">
                                        <NuxtLink :to="`/item/${mod.charge_type_id}`"
                                            class="hover:text-blue-400 transition-colors truncate">
                                            {{ mod.charge_name ?? `Type ${mod.charge_type_id}` }}
                                        </NuxtLink>
                                    </div>
                                </li>
                            </ul>
                            <div v-else class="text-fine text-gray-700 italic">empty</div>
                        </div>
                    </div>

                    <!-- Drone bay row — only shown when the canonical fit had drones. -->
                    <div v-if="expandedFamily === family.family_hash && family.drones.length > 0"
                        class="flex items-center gap-3 flex-wrap px-4 py-2 border-t border-white/[0.06] bg-white/[0.01]">
                        <div class="flex items-center gap-1.5 flex-shrink-0">
                            <Icon name="lucide:radar" class="w-3.5 h-3.5 text-blue-400/70" />
                            <span class="text-fine uppercase tracking-wider text-gray-600">Drone bay</span>
                        </div>
                        <div v-for="d in family.drones" :key="`d-${d.type_id}`"
                            class="flex items-center gap-1.5 px-2 py-0.5 rounded text-xs bg-white/[0.04]">
                            <img :src="`/images/types/${d.type_id}/icon?size=32`"
                                :alt="d.name ?? ''"
                                class="w-4 h-4 rounded flex-shrink-0" loading="lazy">
                            <NuxtLink :to="`/item/${d.type_id}`"
                                class="text-gray-300 hover:text-blue-400 transition-colors truncate">
                                {{ d.name ?? `Type ${d.type_id}` }}
                            </NuxtLink>
                            <span class="text-gray-500 tabular-nums">×{{ d.quantity }}</span>
                        </div>
                    </div>

                    <!-- Alliance usage footer -->
                    <div v-if="expandedFamily === family.family_hash && family.top_alliances.length > 0"
                        class="flex items-center gap-2 flex-wrap px-4 py-2 border-t border-white/[0.06] bg-white/[0.01]">
                        <Icon name="lucide:users" class="w-3.5 h-3.5 text-gray-600" />
                        <span class="text-fine uppercase tracking-wider text-gray-600 mr-1">Flown by</span>
                        <NuxtLink v-for="alliance in family.top_alliances" :key="alliance.alliance_id"
                            :to="`/alliance/${alliance.alliance_id}`"
                            class="flex items-center gap-1.5 px-2 py-0.5 rounded text-xs bg-white/[0.04] hover:bg-white/[0.08] transition-colors">
                            <img :src="`/images/alliances/${alliance.alliance_id}/logo?size=32`"
                                :alt="alliance.name ?? ''" class="w-4 h-4 rounded" loading="lazy">
                            <span class="text-gray-300">{{ alliance.name ?? `Alliance ${alliance.alliance_id}` }}</span>
                            <span class="text-gray-500 tabular-nums">{{ alliance.pct_of_alliance_losses }}%</span>
                        </NuxtLink>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>
