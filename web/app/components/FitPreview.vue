<script setup lang="ts">
// Compact fit preview card for the advanced search "Fits" view.
// Renders a killmail's victim fit as a slot-grouped icon grid, without
// the full editor-centric ShipFit wheel (too heavy for a results list).
// Clicking a card opens the fit in the editor via an encoded ESF v3 URL.

import { killmailFitToEditorUrl } from '~/composables/fit/killmailToFit'

interface Module {
    slot_group: number
    type_id: number
    name: string
    charge_type_id: number | null
}

interface Drone {
    type_id: number
    name: string
    quantity: number
}

const props = defineProps<{
    killmailId: number
    killmailTime: string
    victimShipTypeId: number
    victimShipName: string
    totalValue: number
    attackerCount: number
    modules: Module[]
    drones: Drone[]
    count?: number // > 1 when deduped (grouped by fit hash)
    hash?: string // grouping key (family_hash or fit_hash) for drill-down
    dedupMode?: string // 'exact' | 'family'
}>()

const emit = defineEmits<{
    'show-kills': [hash: string, dedupMode: string]
}>()

const SLOT_META: Record<number, { label: string; color: string }> = {
    1: { label: 'High', color: 'text-amber-400' },
    2: { label: 'Med', color: 'text-gray-300' },
    3: { label: 'Low', color: 'text-green-400' },
    4: { label: 'Rig', color: 'text-blue-400' },
    5: { label: 'Sub', color: 'text-yellow-400' },
}

const slotGroups = computed(() => {
    const groups: { slot_group: number; label: string; color: string; items: Module[] }[] = []
    const bySlot = new Map<number, Module[]>()
    for (const m of props.modules) {
        let list = bySlot.get(m.slot_group)
        if (!list) { list = []; bySlot.set(m.slot_group, list) }
        list.push(m)
    }
    for (const [sg, items] of [...bySlot.entries()].sort((a, b) => a[0] - b[0])) {
        const meta = SLOT_META[sg]
        if (meta) groups.push({ slot_group: sg, ...meta, items })
    }
    return groups
})

const modIcon = (typeId: number) => `/images/types/${typeId}/icon?size=32`
const shipRender = (typeId: number) => `/images/types/${typeId}/render?size=128`

const formatDate = (iso: string) => {
    const d = new Date(iso)
    if (Number.isNaN(d.getTime())) return ''
    // EVE time, like the rest of the site — the local clock was the odd one out.
    return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', timeZone: 'UTC' })
        + ' ' + formatEveTime(d)
}

const openInEditor = async () => {
    const url = await killmailFitToEditorUrl({
        shipTypeId: props.victimShipTypeId,
        modules: props.modules.map((m, i) => ({
            slot_group: m.slot_group,
            ordinal: i,
            type_id: m.type_id,
            charge_type_id: m.charge_type_id,
        })),
        drones: props.drones,
        name: props.victimShipName,
    })
    navigateTo(url)
}
</script>

<template>
    <div class="rounded-xl border border-white/[0.08] bg-white/[0.02] hover:bg-white/[0.04] hover:border-white/[0.12] transition-colors overflow-hidden relative cursor-pointer"
         @click="openInEditor">
        <!-- Popularity badge (dedup mode) — click to drill down into individual kills -->
        <div v-if="count && count > 1"
             class="absolute top-2 right-2 px-2 py-0.5 rounded-full bg-blue-500/20 border border-blue-400/30 text-xs font-bold text-blue-400 tabular-nums hover:bg-blue-500/30 hover:border-blue-400/50 transition-colors"
             v-tooltip="`Show all ${count} killmails with this fitting`"
             @click.stop="hash && dedupMode && emit('show-kills', hash, dedupMode)">
            {{ count.toLocaleString('en-US') }}×
        </div>

        <!-- Header: ship render + meta -->
        <div class="flex gap-3 p-3 pb-2">
            <img :src="shipRender(victimShipTypeId)" :alt="victimShipName"
                 class="w-16 h-16 rounded-lg flex-shrink-0 bg-black/30" loading="lazy">
            <div class="flex-1 min-w-0">
                <div class="text-sm font-medium text-white truncate">{{ victimShipName }}</div>
                <div class="text-fine text-isk/80 tabular-nums">{{ count && count > 1 ? 'avg ' : '' }}{{ formatIsk(totalValue) }} ISK</div>
                <div class="text-fine text-gray-500 tabular-nums">
                    {{ count && count > 1 ? 'avg ' : '' }}{{ attackerCount }} attacker{{ attackerCount === 1 ? '' : 's' }}
                    <span class="mx-1 text-gray-700">&middot;</span>
                    {{ formatDate(killmailTime) }}
                </div>
            </div>
        </div>

        <!-- Modules by slot group -->
        <div class="px-3 pb-2 space-y-1">
            <div v-for="sg in slotGroups" :key="sg.slot_group" class="flex items-center gap-1.5">
                <span class="text-fine font-medium w-7 flex-shrink-0 tabular-nums" :class="sg.color">{{ sg.label }}</span>
                <div class="flex flex-wrap gap-0.5">
                    <img v-for="(m, i) in sg.items" :key="i"
                         :src="modIcon(m.type_id)" v-tooltip="m.name"
                         class="w-6 h-6 rounded border border-white/[0.06] bg-black/30" loading="lazy">
                </div>
            </div>

            <!-- Drones -->
            <div v-if="drones.length > 0" class="flex items-center gap-1.5">
                <span class="text-fine font-medium w-7 flex-shrink-0 tabular-nums text-teal-400">Drn</span>
                <div class="flex flex-wrap gap-0.5">
                    <div v-for="d in drones" :key="d.type_id" class="relative" v-tooltip="`${d.name} ×${d.quantity}`">
                        <img :src="modIcon(d.type_id)" class="w-6 h-6 rounded border border-white/[0.06] bg-black/30" loading="lazy">
                        <span v-if="d.quantity > 1"
                              class="absolute -bottom-0.5 -right-0.5 text-[8px] font-bold text-white bg-black/70 rounded px-0.5 leading-tight">
                            {{ d.quantity }}
                        </span>
                    </div>
                </div>
            </div>
        </div>

        <!-- Kill link (secondary) -->
        <div class="flex items-center justify-end px-3 pb-2">
            <NuxtLink :to="`/kill/${killmailId}`" class="text-fine text-gray-600 hover:text-blue-400 transition-colors"
                       @click.stop v-tooltip="'View killmail'">
                <Icon name="lucide:external-link" class="text-xs mr-0.5" />
                kill
            </NuxtLink>
        </div>
    </div>
</template>
