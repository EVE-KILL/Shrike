<script setup lang="ts">
interface IntelPilot {
    character_id: number
    character_name: string | null
    corporation_id: number | null
    corporation_name: string | null
    alliance_id: number | null
    alliance_name: string | null
    ship_type_id: number | null
    ship_name: string | null
    ship_group_id: number | null
    ship_group_name: string | null
    damage_done: number
    died: boolean
}

interface IntelFC extends IntelPilot {
    confirmed: boolean
}

interface IntelTeam {
    team_index: number
    fcs: IntelFC[]
    logistics: IntelPilot[]
    capitals: IntelPilot[]
}

const props = defineProps<{
    apiEndpoint: string
    extraParams?: Record<string, any>
}>()

const { data, pending, error, refresh } = await useApiFetch<{ teams: IntelTeam[] }>(
    props.apiEndpoint,
    {
        params: props.extraParams,
        default: () => ({ teams: [] }),
    },
)

const route = useRoute()
const lossLink = (group: number, side: number) => ({ path: `/battle/${route.params.id}/kills`, query: { ...route.query, group: String(group), side: String(side), from: undefined, to: undefined } })
const teams = computed(() => data.value?.teams ?? [])
const teamA = computed(() => teams.value[0])
const teamB = computed(() => teams.value[1])

const fmtNum = (v: number) => (v ?? 0).toLocaleString('en-US')

const teamLabel = (idx: number) => idx === 0 ? 'Team A' : 'Team B'
const teamBorder = (idx: number) => idx === 0 ? 'border-red-500/20' : 'border-blue-500/20'
const teamAccent = (idx: number) => idx === 0 ? 'text-red-400' : 'text-blue-400'

const shipRender = (typeId: number | null) =>
    typeId ? `/images/types/${typeId}/render?size=32` : null
const charPortrait = (charId: number) =>
    `/images/characters/${charId}/portrait?size=32`

// Per-team summary stats
const CAP_ORDER: Record<string, number> = { Titan: 6, Supercarrier: 5, 'Lancer Dreadnought': 4, Dreadnought: 3, Carrier: 2, 'Force Auxiliary': 1 }

function teamStats(team: IntelTeam | undefined) {
    if (!team) return { fcConfirmed: 0, fcPossible: 0, logiTotal: 0, logiSurvived: 0, logiDied: 0, capTotal: 0, capSurvived: 0, capDied: 0, capGroups: [] as { id: number | null, name: string, count: number, died: number }[] }
    const fcConfirmed = team.fcs.filter(f => f.confirmed).length
    const fcPossible = team.fcs.filter(f => !f.confirmed).length
    const logiSurvived = team.logistics.filter(p => !p.died).length
    const logiDied = team.logistics.filter(p => p.died).length
    const byGroup = new Map<string, { id: number | null, count: number, died: number }>()
    for (const p of team.capitals) {
        const name = p.ship_group_name || 'Unknown'
        const e = byGroup.get(name) || { id: p.ship_group_id, count: 0, died: 0 }
        e.count++
        if (p.died) e.died++
        byGroup.set(name, e)
    }
    const capGroups = [...byGroup.entries()].map(([name, s]) => ({ name, ...s })).sort((a, b) => (CAP_ORDER[b.name] ?? 0) - (CAP_ORDER[a.name] ?? 0))
    return { fcConfirmed, fcPossible, logiTotal: team.logistics.length, logiSurvived, logiDied, capTotal: team.capitals.length, capSurvived: team.capitals.filter(p => !p.died).length, capDied: team.capitals.filter(p => p.died).length, capGroups }
}

const statsA = computed(() => teamStats(teamA.value))
const statsB = computed(() => teamStats(teamB.value))
</script>

<template>
    <div>
        <p class="mb-4 text-xs text-gray-500">Full-battle killmail observations. FC candidates come from command hulls or prior identification. Pilot loss flags include losses in other hulls or capsules; inspect class losses for destroyed ships.</p>
        <div v-if="error" role="alert" class="glass-panel p-5 text-red-300">Unable to load intelligence. <button class="underline" @click="refresh()">Retry</button></div>
        <div v-else-if="pending" class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div v-for="i in 2" :key="i" class="h-96 rounded-lg bg-white/[0.04] animate-pulse"></div>
        </div>

        <div v-else-if="teams.length < 2" class="glass-panel p-8 text-center text-sm text-gray-500">
            Intel data is not available for this battle.
        </div>

        <template v-else>
            <!-- Summary bar — per side -->
            <div class="grid grid-cols-1 md:grid-cols-2 gap-3 mb-4">
                <template v-for="(stats, sIdx) in [statsA, statsB]" :key="sIdx">
                    <div class="rounded-lg bg-white/[0.04] border p-3 space-y-2" :class="sIdx === 0 ? 'border-red-500/20' : 'border-blue-500/20'">
                        <div class="text-xs font-bold mb-1" :class="sIdx === 0 ? 'text-red-400' : 'text-blue-400'">{{ sIdx === 0 ? 'Team A' : 'Team B' }}</div>
                        <div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs">
                            <!-- FCs -->
                            <div class="flex items-center gap-1.5">
                                <Icon name="lucide:crown" class="text-fine text-yellow-400" />
                                <span class="text-gray-400">FC</span>
                                <span class="text-yellow-400 font-medium tabular-nums">{{ stats.fcConfirmed }}</span>
                                <span v-if="stats.fcPossible > 0" class="text-gray-500">+ <span class="text-gray-400 tabular-nums">{{ stats.fcPossible }}</span> possible</span>
                            </div>
                            <!-- Logi -->
                            <div class="flex items-center gap-1.5">
                                <Icon name="lucide:heart-pulse" class="text-fine text-green-400" />
                                <span class="text-gray-400">Logi</span>
                                <span class="text-green-400 font-medium tabular-nums">{{ stats.logiSurvived }}</span>
                                <span v-if="stats.logiDied > 0" class="text-red-400/80 tabular-nums">/ {{ stats.logiDied }} with losses</span>
                            </div>
                            <!-- Capitals -->
                            <div class="flex items-center gap-1.5">
                                <Icon name="lucide:shield" class="text-fine text-orange-400" />
                                <span class="text-gray-400">Caps</span>
                                <span class="text-orange-400 font-medium tabular-nums">{{ stats.capTotal }}</span>
                                <span v-if="stats.capDied > 0" class="text-red-400/80 tabular-nums">/ {{ stats.capDied }} with losses</span>
                            </div>
                        </div>
                        <!-- Capital breakdown by type -->
                        <div v-if="stats.capGroups.length > 0" class="flex flex-wrap gap-x-3 gap-y-0.5 text-fine text-gray-500 pl-0.5">
                            <span v-for="g in stats.capGroups" :key="g.name">
                                {{ g.count }}x <NuxtLink v-if="g.id" :to="lossLink(g.id,sIdx)" class="hover:text-blue-400" title="Inspect this class’s losses">{{ g.name }}</NuxtLink><span v-else>{{ g.name }}</span><span v-if="g.died > 0" class="text-red-400/60 ml-0.5">({{ g.died }} pilots with losses)</span>
                            </span>
                        </div>
                    </div>
                </template>
            </div>

            <!-- Per-team details -->
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <template v-for="(team, tIdx) in [teamA, teamB]" :key="tIdx">
                <div class="space-y-4">
                    <!-- Fleet Commanders -->
                    <div class="rounded-lg bg-white/[0.04] border p-3" :class="teamBorder(tIdx)">
                        <div class="flex items-center justify-between mb-3 pb-2 border-b border-white/[0.06]">
                            <div class="flex items-center gap-2">
                                <Icon name="lucide:crown" class="text-sm text-yellow-400" />
                                <span class="text-sm font-bold" :class="teamAccent(tIdx)">{{ teamLabel(tIdx) }} — Fleet Commanders</span>
                            </div>
                            <div class="text-fine text-gray-500">{{ team?.fcs?.length || 0 }} pilots</div>
                        </div>

                        <table v-if="team?.fcs?.length" class="w-full text-xs table-fixed">
                            <thead class="text-gray-500 text-left">
                                <tr class="border-b border-white/[0.06]">
                                    <th class="py-1.5 pr-2 font-medium w-[40%]">Pilot</th>
                                    <th class="py-1.5 px-2 font-medium w-[30%]">Ship</th>
                                    <th class="py-1.5 px-2 font-medium text-right w-[15%]">Damage</th>
                                    <th class="py-1.5 pl-2 font-medium text-right w-[15%]">Status</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-for="fc in team.fcs" :key="fc.character_id"
                                    class="border-b border-white/[0.03] hover:bg-white/[0.02] align-top">
                                    <td class="py-1.5 pr-2">
                                        <div class="flex items-center gap-2 min-w-0">
                                            <img :src="charPortrait(fc.character_id)" class="w-8 h-8 rounded flex-shrink-0" loading="lazy">
                                            <div class="min-w-0 leading-tight">
                                                <NuxtLink :to="`/character/${fc.character_id}`" class="block truncate text-gray-300 hover:text-blue-400">
                                                    {{ fc.character_name || 'Unknown' }}
                                                </NuxtLink>
                                                <div class="truncate text-fine text-gray-500">
                                                    {{ fc.corporation_name || '' }}
                                                </div>
                                            </div>
                                        </div>
                                    </td>
                                    <td class="py-1.5 px-2">
                                        <div class="flex items-center gap-2 min-w-0">
                                            <img v-if="shipRender(fc.ship_type_id)" :src="shipRender(fc.ship_type_id)!" class="w-8 h-8 rounded flex-shrink-0" loading="lazy">
                                            <div class="min-w-0 leading-tight">
                                                <NuxtLink v-if="fc.ship_type_id" :to="`/item/${fc.ship_type_id}`" class="block truncate hover:text-blue-400" :class="fc.died ? 'text-red-300' : 'text-gray-300'">{{ fc.ship_name || '?' }}</NuxtLink>
                                                <NuxtLink v-if="fc.ship_group_id" :to="`/group/${fc.ship_group_id}`" class="block truncate text-fine text-gray-500 hover:text-blue-400">{{ fc.ship_group_name || '' }}</NuxtLink>
                                            </div>
                                        </div>
                                    </td>
                                    <td class="py-1.5 px-2 text-right tabular-nums text-gray-300">
                                        {{ fmtNum(fc.damage_done) }}
                                    </td>
                                    <td class="py-1.5 pl-2 text-right">
                                        <div class="flex flex-col items-end gap-0.5">
                                            <span v-if="fc.confirmed"
                                                class="inline-block px-1.5 py-0.5 rounded text-fine font-medium bg-yellow-500/20 text-yellow-400">
                                                Command hull
                                            </span>
                                            <span v-else
                                                class="inline-block px-1.5 py-0.5 rounded text-fine font-medium bg-gray-500/20 text-gray-400">
                                                Prior FC signal
                                            </span>
                                            <span v-if="fc.died"
                                                class="inline-block px-1.5 py-0.5 rounded text-fine font-medium bg-red-500/20 text-red-400">
                                                Died
                                            </span>
                                        </div>
                                    </td>
                                </tr>
                            </tbody>
                        </table>
                        <div v-else class="py-4 text-center text-fine text-gray-600">
                            No fleet commanders identified
                        </div>
                    </div>

                    <!-- Logistics -->
                    <div class="rounded-lg bg-white/[0.04] border p-3" :class="teamBorder(tIdx)">
                        <div class="flex items-center justify-between mb-3 pb-2 border-b border-white/[0.06]">
                            <div class="flex items-center gap-2">
                                <Icon name="lucide:heart-pulse" class="text-sm text-green-400" />
                                <span class="text-sm font-bold" :class="teamAccent(tIdx)">{{ teamLabel(tIdx) }} — Logistics</span>
                            </div>
                            <div class="text-fine text-gray-500">{{ team?.logistics?.length || 0 }} pilots</div>
                        </div>

                        <table v-if="team?.logistics?.length" class="w-full text-xs table-fixed">
                            <thead class="text-gray-500 text-left">
                                <tr class="border-b border-white/[0.06]">
                                    <th class="py-1.5 pr-2 font-medium w-[40%]">Pilot</th>
                                    <th class="py-1.5 px-2 font-medium w-[30%]">Ship</th>
                                    <th class="py-1.5 px-2 font-medium text-right w-[15%]">Damage</th>
                                    <th class="py-1.5 pl-2 font-medium text-right w-[15%]">Status</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-for="pilot in team.logistics" :key="pilot.character_id"
                                    class="border-b border-white/[0.03] hover:bg-white/[0.02] align-top">
                                    <td class="py-1.5 pr-2">
                                        <div class="flex items-center gap-2 min-w-0">
                                            <img :src="charPortrait(pilot.character_id)" class="w-8 h-8 rounded flex-shrink-0" loading="lazy">
                                            <div class="min-w-0 leading-tight">
                                                <NuxtLink :to="`/character/${pilot.character_id}`" class="block truncate text-gray-300 hover:text-blue-400">
                                                    {{ pilot.character_name || 'Unknown' }}
                                                </NuxtLink>
                                                <div class="truncate text-fine text-gray-500">
                                                    {{ pilot.corporation_name || '' }}
                                                </div>
                                            </div>
                                        </div>
                                    </td>
                                    <td class="py-1.5 px-2">
                                        <div class="flex items-center gap-2 min-w-0">
                                            <img v-if="shipRender(pilot.ship_type_id)" :src="shipRender(pilot.ship_type_id)!" class="w-8 h-8 rounded flex-shrink-0" loading="lazy">
                                            <div class="min-w-0 leading-tight">
                                                <NuxtLink v-if="pilot.ship_type_id" :to="`/item/${pilot.ship_type_id}`" class="block truncate hover:text-blue-400" :class="pilot.died ? 'text-red-300' : 'text-gray-300'">{{ pilot.ship_name || '?' }}</NuxtLink>
                                                <NuxtLink v-if="pilot.ship_group_id" :to="`/group/${pilot.ship_group_id}`" class="block truncate text-fine text-gray-500 hover:text-blue-400">{{ pilot.ship_group_name || '' }}</NuxtLink>
                                            </div>
                                        </div>
                                    </td>
                                    <td class="py-1.5 px-2 text-right tabular-nums text-gray-300">
                                        {{ fmtNum(pilot.damage_done) }}
                                    </td>
                                    <td class="py-1.5 pl-2 text-right">
                                        <span v-if="pilot.died"
                                            class="inline-block px-1.5 py-0.5 rounded text-fine font-medium bg-red-500/20 text-red-400">
                                            Died
                                        </span>
                                        <span v-else class="text-fine text-green-400">No recorded loss</span>
                                    </td>
                                </tr>
                            </tbody>
                        </table>
                        <div v-else class="py-4 text-center text-fine text-gray-600">
                            No logistics pilots identified
                        </div>
                    </div>

                    <!-- Capital Pilots -->
                    <div class="rounded-lg bg-white/[0.04] border p-3" :class="teamBorder(tIdx)">
                        <div class="flex items-center justify-between mb-3 pb-2 border-b border-white/[0.06]">
                            <div class="flex items-center gap-2">
                                <Icon name="lucide:shield" class="text-sm text-orange-400" />
                                <span class="text-sm font-bold" :class="teamAccent(tIdx)">{{ teamLabel(tIdx) }} — Capital Pilots</span>
                            </div>
                            <div class="text-fine text-gray-500">{{ team?.capitals?.length || 0 }} pilots</div>
                        </div>

                        <table v-if="team?.capitals?.length" class="w-full text-xs table-fixed">
                            <thead class="text-gray-500 text-left">
                                <tr class="border-b border-white/[0.06]">
                                    <th class="py-1.5 pr-2 font-medium w-[40%]">Pilot</th>
                                    <th class="py-1.5 px-2 font-medium w-[30%]">Ship</th>
                                    <th class="py-1.5 px-2 font-medium text-right w-[15%]">Damage</th>
                                    <th class="py-1.5 pl-2 font-medium text-right w-[15%]">Status</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-for="pilot in team.capitals" :key="pilot.character_id"
                                    class="border-b border-white/[0.03] hover:bg-white/[0.02] align-top">
                                    <td class="py-1.5 pr-2">
                                        <div class="flex items-center gap-2 min-w-0">
                                            <img :src="charPortrait(pilot.character_id)" class="w-8 h-8 rounded flex-shrink-0" loading="lazy">
                                            <div class="min-w-0 leading-tight">
                                                <NuxtLink :to="`/character/${pilot.character_id}`" class="block truncate text-gray-300 hover:text-blue-400">
                                                    {{ pilot.character_name || 'Unknown' }}
                                                </NuxtLink>
                                                <div class="truncate text-fine text-gray-500">
                                                    {{ pilot.corporation_name || '' }}
                                                </div>
                                            </div>
                                        </div>
                                    </td>
                                    <td class="py-1.5 px-2">
                                        <div class="flex items-center gap-2 min-w-0">
                                            <img v-if="shipRender(pilot.ship_type_id)" :src="shipRender(pilot.ship_type_id)!" class="w-8 h-8 rounded flex-shrink-0" loading="lazy">
                                            <div class="min-w-0 leading-tight">
                                                <NuxtLink v-if="pilot.ship_type_id" :to="`/item/${pilot.ship_type_id}`" class="block truncate hover:text-blue-400" :class="pilot.died ? 'text-red-300' : 'text-gray-300'">{{ pilot.ship_name || '?' }}</NuxtLink>
                                                <NuxtLink v-if="pilot.ship_group_id" :to="`/group/${pilot.ship_group_id}`" class="block truncate text-fine text-gray-500 hover:text-blue-400">{{ pilot.ship_group_name || '' }}</NuxtLink>
                                            </div>
                                        </div>
                                    </td>
                                    <td class="py-1.5 px-2 text-right tabular-nums text-gray-300">
                                        {{ fmtNum(pilot.damage_done) }}
                                    </td>
                                    <td class="py-1.5 pl-2 text-right">
                                        <span v-if="pilot.died"
                                            class="inline-block px-1.5 py-0.5 rounded text-fine font-medium bg-red-500/20 text-red-400">
                                            Died
                                        </span>
                                        <span v-else class="text-fine text-green-400">No recorded loss</span>
                                    </td>
                                </tr>
                            </tbody>
                        </table>
                        <div v-else class="py-4 text-center text-fine text-gray-600">
                            No capital pilots in this battle
                        </div>
                    </div>
                </div>
            </template>
            </div>
        </template>
    </div>
</template>
