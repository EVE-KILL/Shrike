<script setup lang="ts">
/**
 * Faction dashboard.
 *
 * Faction *killboard* stats are thin by design — /api/faction/:id returns only
 * {losses, isk_lost}, and the entity top-lists / ship-classes / most-valuable
 * endpoints all reject the type, because factions have no rollup rows in the
 * stats pipeline and killmail_attackers.faction_id is unindexed.
 *
 * The interesting material for the four militia factions is faction warfare,
 * which is already computed for the /faction-war pages. This pulls that in per
 * faction: how the warzone stands, what changed hands lately, who is flying for
 * them, and which corporations make up the militia. Pirate and minor factions
 * are not in a matchup and fall back to the description alone.
 */
const props = defineProps<{
    faction: {
        faction_id: number
        name: string
        description: string | null
        corporation_id: number | null
        militia_corporation_id: number | null
        solar_system_id: number | null
        station_count: number | null
        station_system_count: number | null
    }
}>()

/**
 * Which matchup a faction fights in, and which side of it they are.
 * Mirrors the MATCHUPS tables in server/api/faction-war/*.
 */
const MATCHUPS: Record<number, { matchup: string; side: 1 | 2; opponent: number }> = {
    500001: { matchup: 'caldari-vs-gallente', side: 1, opponent: 500004 },
    500004: { matchup: 'caldari-vs-gallente', side: 2, opponent: 500001 },
    500003: { matchup: 'amarr-vs-minmatar', side: 1, opponent: 500002 },
    500002: { matchup: 'amarr-vs-minmatar', side: 2, opponent: 500003 },
}

const fw = computed(() => MATCHUPS[props.faction.faction_id] ?? null)

// Client-only and lazy — these are warzone extras below the fold, and the
// description above them should not wait on four faction-war queries.
const { data: overview } = useApiFetch<any>(
    () => fw.value ? `/api/faction-war/${fw.value.matchup}/overview` : '',
    { lazy: true, server: false, immediate: !!fw.value, default: () => null },
)
const { data: memberData } = useApiFetch<any>(
    () => fw.value ? `/api/faction-war/${fw.value.matchup}/members` : '',
    {
        lazy: true,
        server: false,
        immediate: !!fw.value,
        default: () => null,
        // The endpoint names the sides 'aggressor' (= matchup side1) and
        // 'defender' (= side2); anything else falls through to both sides, which
        // would list the enemy's corporations under this faction's militia.
        query: computed(() => ({ days: 30, side: fw.value?.side === 1 ? 'aggressor' : 'defender', limit: 500 })),
    },
)

const sideKey = computed(() => fw.value?.side === 2 ? 'side2' : 'side1')
const foeKey = computed(() => fw.value?.side === 2 ? 'side1' : 'side2')

const mine = computed(() => overview.value?.factionStats?.[sideKey.value] ?? null)
const theirs = computed(() => overview.value?.factionStats?.[foeKey.value] ?? null)
const myZone = computed(() => overview.value?.warzone?.[sideKey.value] ?? null)
const totalSystems = computed(() => overview.value?.warzone?.total_systems ?? 0)

/** Share of the warzone held, for the standing bar. */
const holdPct = computed(() => {
    const held = mine.value?.systems_controlled ?? 0
    return totalSystems.value > 0 ? (held / totalSystems.value) * 100 : 0
})

/**
 * Flips involving this faction, newest first, flattened out of the per-day
 * buckets. A flip where this faction is the new holder is a gain; where it is
 * the old holder, a loss. Flips between two other parties never occur in a
 * two-sided matchup, but the filter keeps that honest.
 */
const flips = computed(() => {
    const days = overview.value?.flipDays ?? []
    const id = props.faction.faction_id
    const out: { system_id: number; system: string; gained: boolean; at: string }[] = []
    for (const day of days) {
        for (const item of day.items ?? []) {
            const gained = item.new_faction_id === id
            const lost = item.old_faction_id === id
            if (!gained && !lost) continue
            out.push({ system_id: item.solar_system_id, system: item.system_name, gained, at: item.flipped_at })
        }
    }
    return out.sort((a, b) => new Date(b.at).getTime() - new Date(a.at).getTime())
})

const flipTally = computed(() => ({
    gained: flips.value.filter(f => f.gained).length,
    lost: flips.value.filter(f => !f.gained).length,
}))

const topPilots = computed(() => (overview.value?.leaderboards?.characters?.[sideKey.value] ?? []).slice(0, 10))

/** Militia corporations, ranked by how many of their pilots showed up. */
const militiaCorps = computed(() => {
    const members = memberData.value?.members ?? []
    const byCorp = new Map<number, { id: number; name: string; pilots: number; kills: number }>()
    for (const m of members) {
        if (!m.corporation_id) continue
        const row = byCorp.get(m.corporation_id) ?? { id: m.corporation_id, name: m.corporation_name || 'Unknown', pilots: 0, kills: 0 }
        row.pilots += 1
        row.kills += m.kills ?? 0
        byCorp.set(m.corporation_id, row)
    }
    return [...byCorp.values()].sort((a, b) => b.pilots - a.pilots).slice(0, 10)
})

const links = computed(() => [
    { key: 'corporation', id: props.faction.corporation_id, label: 'Executor corporation', to: (id: number) => `/corporation/${id}`, img: (id: number) => `/images/corporations/${id}/logo?size=64` },
    { key: 'corporation', id: props.faction.militia_corporation_id, label: 'Militia corporation', to: (id: number) => `/corporation/${id}`, img: (id: number) => `/images/corporations/${id}/logo?size=64` },
    { key: 'system', id: props.faction.solar_system_id, label: 'Capital system', to: (id: number) => `/system/${id}`, img: (id: number) => `/images/systems/${id}?size=64` },
].filter(l => !!l.id) as { key: string; id: number; label: string; to: (id: number) => string; img: (id: number) => string }[])

const names = ref<Record<number, string>>({})
onMounted(async () => {
    const resolved = await Promise.all(links.value.map(async (l) => {
        try {
            const r = await apiFetch<{ name: string | null }>('/api/entity/resolve', { query: { type: l.key, id: l.id } })
            return [l.id, r?.name ?? ''] as const
        }
        catch { return [l.id, ''] as const }
    }))
    names.value = Object.fromEntries(resolved.filter(([, n]) => n))
})

/*
 * Zero is not a fact worth a row: "Stations 0 / Systems with stations 0" is a
 * panel that looks like data and carries none.
 *
 * Note that as of writing these counts are 0 for *every* faction — the columns
 * exist but nothing populates them — so this panel never renders today. It is
 * left in place because the header rows it replaces were guarded the same way,
 * and it will light up on its own if the importer ever fills them in.
 */
const facts = computed(() => [
    { label: 'Stations', value: props.faction.station_count },
    { label: 'Systems with stations', value: props.faction.station_system_count },
].filter(f => (f.value ?? 0) > 0) as { label: string; value: number }[])

const fmt = (n: number | null | undefined) => (n ?? 0).toLocaleString('en-US')
</script>

<template>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <!-- Description in full. The header no longer prints it — corporation and
             alliance pages keep theirs on the dashboard too. -->
        <div>
            <div class="glass-panel p-4 md:sticky md:top-4 md:max-h-[calc(100vh-2rem)] md:overflow-y-auto">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-3">About</div>
                <p v-if="faction.description" class="text-xs text-gray-300 leading-relaxed whitespace-pre-line">{{ faction.description }}</p>
                <div v-else class="text-fine text-gray-600 italic">No description available</div>
            </div>
        </div>

        <div class="space-y-4">
            <!-- Warzone standing -->
            <div v-if="fw && mine" class="glass-panel p-4">
                <div class="flex items-center gap-2 mb-3">
                    <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">Warzone</div>
                    <NuxtLink :to="`/faction-war/${fw.matchup}`" class="ml-auto text-fine text-gray-500 hover:text-blue-400 transition-colors">
                        Full report →
                    </NuxtLink>
                </div>

                <div class="flex items-baseline justify-between text-xs mb-1">
                    <span class="text-gray-400">{{ fmt(mine.systems_controlled) }} of {{ fmt(totalSystems) }} systems</span>
                    <span class="text-gray-600 tabular-nums">{{ Math.round(holdPct) }}%</span>
                </div>
                <!-- Fixed hues: a themed board remaps blue, so the two sides would
                     otherwise collapse into the same colour. -->
                <div class="flex h-1.5 rounded-full overflow-hidden bg-white/[0.06] mb-3">
                    <div class="bg-[#38bdf8]" :style="{ width: `${holdPct}%` }" />
                    <div class="bg-[#f97316]/60 flex-1" />
                </div>

                <div class="grid grid-cols-3 gap-2 mb-3">
                    <div class="rounded bg-white/[0.03] px-2 py-1.5 text-center">
                        <div class="text-sm font-bold text-gray-200 tabular-nums">{{ fmt(myZone?.contested) }}</div>
                        <div class="text-fine text-gray-500">Contested</div>
                    </div>
                    <div class="rounded bg-white/[0.03] px-2 py-1.5 text-center">
                        <div class="text-sm font-bold tabular-nums" :class="(myZone?.vulnerable ?? 0) > 0 ? 'text-amber-400' : 'text-gray-200'">{{ fmt(myZone?.vulnerable) }}</div>
                        <div class="text-fine text-gray-500">Vulnerable</div>
                    </div>
                    <div class="rounded bg-white/[0.03] px-2 py-1.5 text-center">
                        <div class="text-sm font-bold text-gray-200 tabular-nums">{{ fmt(mine.pilots) }}</div>
                        <div class="text-fine text-gray-500">Pilots</div>
                    </div>
                </div>

                <div class="space-y-1.5 text-xs">
                    <div class="flex justify-between"><span class="text-gray-500">Kills yesterday</span><span class="text-gray-300 tabular-nums">{{ fmt(mine.kills_yesterday) }} <span class="text-gray-600">vs {{ fmt(theirs?.kills_yesterday) }}</span></span></div>
                    <div class="flex justify-between"><span class="text-gray-500">Kills last week</span><span class="text-gray-300 tabular-nums">{{ fmt(mine.kills_last_week) }} <span class="text-gray-600">vs {{ fmt(theirs?.kills_last_week) }}</span></span></div>
                    <div class="flex justify-between"><span class="text-gray-500">Victory points (week)</span><span class="text-gray-300 tabular-nums">{{ fmt(mine.vp_last_week) }}</span></div>
                </div>
            </div>

            <!-- Recent system flips -->
            <div v-if="fw && flips.length" class="glass-panel p-4">
                <div class="flex items-center gap-2 mb-3">
                    <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">Recent Flips (7d)</div>
                    <span class="ml-auto text-fine tabular-nums">
                        <span class="text-green-400">+{{ flipTally.gained }}</span>
                        <span class="text-gray-600"> / </span>
                        <span class="text-red-400">−{{ flipTally.lost }}</span>
                    </span>
                </div>
                <div class="space-y-1 max-h-64 overflow-y-auto">
                    <NuxtLink v-for="f in flips" :key="`${f.system_id}-${f.at}`" :to="`/system/${f.system_id}`"
                        class="flex items-center gap-2 px-1.5 py-1 rounded hover:bg-blue-500/[0.06] transition-colors text-xs">
                        <Icon :name="f.gained ? 'lucide:arrow-up-right' : 'lucide:arrow-down-right'"
                            class="text-fine flex-shrink-0" :class="f.gained ? 'text-green-400' : 'text-red-400'" />
                        <span class="text-gray-300 truncate flex-1">{{ f.system }}</span>
                        <span class="text-fine text-gray-600 tabular-nums flex-shrink-0">{{ formatEveDate(f.at) }}</span>
                    </NuxtLink>
                </div>
            </div>

            <!-- Militia corporations + top pilots -->
            <div v-if="fw && (militiaCorps.length || topPilots.length)" class="grid grid-cols-1 lg:grid-cols-2 gap-3">
                <div v-if="militiaCorps.length" class="glass-panel p-4">
                    <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-3">Militia Corps (30d)</div>
                    <div class="space-y-1">
                        <NuxtLink v-for="(c, i) in militiaCorps" :key="c.id" :to="`/corporation/${c.id}`"
                            class="flex items-center gap-2 px-1.5 py-1 rounded hover:bg-blue-500/[0.06] transition-colors text-xs">
                            <span class="text-fine text-gray-600 tabular-nums w-4 flex-shrink-0">{{ Number(i) + 1 }}</span>
                            <img :src="`/images/corporations/${c.id}/logo?size=32`" alt="" class="w-4 h-4 rounded flex-shrink-0" loading="lazy">
                            <span class="text-gray-300 truncate flex-1">{{ c.name }}</span>
                            <span class="text-fine text-gray-500 tabular-nums flex-shrink-0">{{ c.pilots }}</span>
                        </NuxtLink>
                    </div>
                </div>

                <div v-if="topPilots.length" class="glass-panel p-4">
                    <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-3">Top Pilots</div>
                    <div class="space-y-1">
                        <NuxtLink v-for="(p, i) in topPilots" :key="p.id" :to="`/character/${p.id}`"
                            class="flex items-center gap-2 px-1.5 py-1 rounded hover:bg-blue-500/[0.06] transition-colors text-xs">
                            <span class="text-fine text-gray-600 tabular-nums w-4 flex-shrink-0">{{ Number(i) + 1 }}</span>
                            <img :src="`/images/characters/${p.id}/portrait?size=32`" alt="" class="w-4 h-4 rounded flex-shrink-0" loading="lazy">
                            <span class="text-gray-300 truncate flex-1">{{ p.name }}</span>
                            <span class="text-fine text-isk tabular-nums flex-shrink-0">{{ fmt(p.kills) }}</span>
                        </NuxtLink>
                    </div>
                </div>
            </div>

            <!-- Identity -->
            <div v-if="links.length" class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-3">Leadership &amp; Space</div>
                <div class="space-y-2">
                    <NuxtLink v-for="l in links" :key="`${l.key}-${l.id}`" :to="l.to(l.id)"
                        class="flex items-center gap-2.5 px-2 py-1.5 -mx-2 rounded-md hover:bg-blue-500/[0.06] transition-colors">
                        <img :src="l.img(l.id)" :alt="l.label" class="w-8 h-8 rounded flex-shrink-0 bg-white/[0.04]" loading="lazy">
                        <div class="min-w-0 leading-tight">
                            <div class="text-xs text-gray-200 truncate">{{ names[l.id] || '…' }}</div>
                            <div class="text-fine text-gray-500">{{ l.label }}</div>
                        </div>
                    </NuxtLink>
                </div>
            </div>

            <div v-if="facts.length" class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-3">Holdings</div>
                <div class="space-y-1.5 text-xs">
                    <div v-for="f in facts" :key="f.label" class="flex justify-between">
                        <span class="text-gray-500">{{ f.label }}</span>
                        <span class="text-gray-300 tabular-nums">{{ f.value.toLocaleString('en-US') }}</span>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>
