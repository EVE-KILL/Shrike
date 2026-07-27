<script setup lang="ts">
// Solo "1v1" matchup box — shown on the killmail page for solo kills only.
// For the hull pair (attacker A vs victim B) it shows how often A beats B in
// solo fights over the last 90 days, plus the top fits the victim hull was
// flying when it died to this attacker.
//
// Win rate is a HULL-level stat: killmails only record the loser's fit, so a
// ship that *wins* a solo fight leaves no killmail — per-fit win rate can't be
// derived. The fit list is therefore loss-side only ("fits that die to this").
// Data comes from /api/matchup which is pair-keyed and cached.
const props = defineProps<{
    attackerShipTypeId: number
    attackerShipName: string | null
    victimShipTypeId: number
    victimShipName: string | null
}>()

interface MatchupFit {
    family_hash: string
    uses: number
    pct: number
    modules: { slot_group: number; type_id: number; name: string | null }[]
}
interface MatchupResponse {
    attacker_ship_type_id: number
    victim_ship_type_id: number
    window_days: number
    min_sample: number
    mirror: boolean
    attacker_wins: number
    victim_wins: number
    sample: number
    attacker_win_rate: number
    enough: boolean
    top_fits: MatchupFit[]
}

// Client-only, non-blocking — the box is contextual side content, not
// critical-path, and keeping setup synchronous avoids a Suspense requirement
// for this conditionally-rendered component.
const { data } = useApiFetch<MatchupResponse>('/api/matchup', {
    // Explicit key so the desktop + mobile instances on the same page share one
    // fetch (dedup by pair), regardless of how options hash into the auto-key.
    key: `matchup-${props.attackerShipTypeId}-${props.victimShipTypeId}`,
    query: { attacker: props.attackerShipTypeId, victim: props.victimShipTypeId },
    lazy: true,
    server: false,
})

const attackerName = computed(() => props.attackerShipName || 'Attacker')
const victimName = computed(() => props.victimShipName || 'Victim')

// Distinct module type icons for a compact preview strip (max 7 + overflow).
function moduleIcons(fit: MatchupFit) {
    const seen = new Set<number>()
    const out: { type_id: number; name: string | null }[] = []
    for (const m of fit.modules) {
        if (seen.has(m.type_id)) continue
        seen.add(m.type_id)
        out.push({ type_id: m.type_id, name: m.name })
    }
    return out
}
</script>

<template>
    <div v-if="data && data.enough"
        class="glass-panel p-3">
        <!-- Header -->
        <div class="flex items-baseline justify-between mb-3">
            <span class="text-xs font-semibold text-gray-300 flex items-center gap-1.5">
                <Icon name="lucide:swords" class="w-3.5 h-3.5 text-gray-500" />
                Solo matchup
            </span>
            <span class="text-fine text-gray-600">last {{ data.window_days }}d</span>
        </div>

        <!-- Hulls -->
        <div class="flex items-center justify-between gap-2 mb-2">
            <NuxtLink :to="`/item/${attackerShipTypeId}`"
                class="flex items-center gap-2 min-w-0 group">
                <img :src="`/images/types/${attackerShipTypeId}/icon?size=64`"
                    :alt="attackerName" class="w-8 h-8 rounded flex-shrink-0" loading="lazy">
                <span class="text-xs text-gray-200 group-hover:text-blue-400 transition-colors truncate">{{ attackerName }}</span>
            </NuxtLink>
            <span class="text-fine uppercase tracking-wider text-gray-600 flex-shrink-0">vs</span>
            <NuxtLink :to="`/item/${victimShipTypeId}`"
                class="flex items-center gap-2 min-w-0 justify-end group">
                <span class="text-xs text-gray-200 group-hover:text-blue-400 transition-colors truncate text-right">{{ victimName }}</span>
                <img :src="`/images/types/${victimShipTypeId}/icon?size=64`"
                    :alt="victimName" class="w-8 h-8 rounded flex-shrink-0" loading="lazy">
            </NuxtLink>
        </div>

        <!-- Mirror matchup note -->
        <div v-if="data.mirror" class="text-fine text-gray-500 mb-2">
            Mirror matchup — 50% by definition over {{ data.sample }} solo fight{{ data.sample === 1 ? '' : 's' }}.
        </div>

        <!-- Win-rate bar -->
        <template v-else>
            <div class="flex items-baseline justify-between text-xs mb-1">
                <span class="text-emerald-400 font-medium">
                    {{ attackerName }} wins {{ data.attacker_win_rate }}%
                </span>
                <span class="text-gray-500 tabular-nums">
                    {{ data.attacker_wins }}–{{ data.victim_wins }}
                </span>
            </div>
            <div class="h-2 rounded-full overflow-hidden bg-red-500/30 flex mb-1.5">
                <div class="h-full bg-emerald-500/70" :style="{ width: `${data.attacker_win_rate}%` }" />
            </div>
            <div class="text-fine text-gray-600 mb-3">
                {{ data.sample }} solo fight{{ data.sample === 1 ? '' : 's' }} ({{ attackerName }} vs {{ victimName }})
            </div>
        </template>

        <!-- Loss-fit breakdown -->
        <div v-if="data.top_fits.length > 0" class="border-t border-white/[0.06] pt-2.5">
            <div class="text-fine uppercase tracking-wider text-gray-600 mb-2">
                {{ victimName }} fits that die to {{ attackerName }}
            </div>
            <div class="space-y-2">
                <div v-for="fit in data.top_fits" :key="fit.family_hash"
                    class="flex items-center gap-2">
                    <div class="flex items-center gap-0.5 flex-1 min-w-0">
                        <template v-for="mod in moduleIcons(fit).slice(0, 7)" :key="mod.type_id">
                            <img :src="`/images/types/${mod.type_id}/icon?size=32`"
                                :alt="mod.name ?? ''" v-tooltip="mod.name ?? ''"
                                class="w-5 h-5 rounded flex-shrink-0" loading="lazy">
                        </template>
                        <span v-if="moduleIcons(fit).length > 7"
                            class="text-fine text-gray-600 ml-1 flex-shrink-0">+{{ moduleIcons(fit).length - 7 }}</span>
                    </div>
                    <span class="text-fine text-gray-500 tabular-nums flex-shrink-0">
                        {{ fit.uses }} · {{ fit.pct }}%
                    </span>
                </div>
            </div>
        </div>

        <!-- Honesty footnote -->
        <p class="text-fine text-gray-700 mt-2.5 leading-snug">
            From solo killmails only — disengages and bait aren't counted, and only the loser's fit is known.
        </p>
    </div>
</template>
