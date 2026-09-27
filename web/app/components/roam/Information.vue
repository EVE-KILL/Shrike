<script setup lang="ts">
import type { RoamReport } from '~/utils/roamReport'

const props = defineProps<{ report: RoamReport }>()
const busiestSystem = computed(() => [...props.report.systems].sort((a, b) => (b.kills + b.losses) - (a.kills + a.losses))[0])
const mostActivePilot = computed(() => [...props.report.roster].sort((a, b) =>
    (b.kill_participations + b.losses) - (a.kill_participations + a.losses))[0])
const firstEngagements = computed(() => props.report.engagements.slice(0, 8))
const topPilots = computed(() => [...props.report.roster].sort((a, b) =>
    (b.kill_participations + b.losses) - (a.kill_participations + a.losses)).slice(0, 8))
const basePath = `/tools/roam-report/${props.report.id}`
</script>

<template>
    <div class="space-y-6">
        <div class="grid gap-4 lg:grid-cols-3">
            <section class="glass-panel p-5">
                <p class="text-xs font-semibold uppercase tracking-wider text-blue-300">Where you fought</p>
                <p class="mt-2 text-lg font-semibold text-white">{{ report.summary.systems }} systems · {{ report.summary.regions }} regions</p>
                <p v-if="busiestSystem" class="mt-2 text-sm leading-6 text-gray-400">Most activity in <strong class="text-gray-200">{{ busiestSystem.solar_system_name }}</strong>: {{ busiestSystem.kills }} kills and {{ busiestSystem.losses }} losses.</p>
                <p v-else class="mt-2 text-sm text-gray-400">No recorded combat in this window.</p>
            </section>
            <section class="glass-panel p-5">
                <p class="text-xs font-semibold uppercase tracking-wider text-amber-300">Who you met</p>
                <p class="mt-2 text-lg font-semibold text-white">{{ report.summary.unique_character_targets }} named {{ report.summary.unique_character_targets === 1 ? 'target' : 'targets' }}</p>
                <p v-if="report.targets[0]" class="mt-2 text-sm leading-6 text-gray-400">Most frequent opposing corporation: <strong class="text-gray-200">{{ report.targets[0].corporation_name || 'Unknown corporation' }}</strong> on {{ report.targets[0].kills }} killmails.</p>
            </section>
            <section class="glass-panel p-5">
                <p class="text-xs font-semibold uppercase tracking-wider text-emerald-300">Fleet activity</p>
                <p class="mt-2 text-lg font-semibold text-white">{{ report.summary.active_pilots }} of {{ report.roster.length }} pilots recorded</p>
                <p class="mt-2 text-sm leading-6 text-gray-400">{{ report.summary.coordinated_kills }} kills show two or more listed pilots.</p>
                <p v-if="mostActivePilot?.engagements" class="mt-1 text-sm leading-6 text-gray-400">{{ mostActivePilot.name }} appears in the most killmails.</p>
            </section>
        </div>

        <div class="grid items-start gap-6 xl:grid-cols-3">
            <section class="glass-panel overflow-hidden xl:col-span-2">
                <div class="flex items-center justify-between gap-3 border-b border-white/[0.07] p-5">
                    <div>
                        <h2 class="text-lg font-semibold text-white">Engagement trail</h2>
                        <p class="mt-1 text-xs text-gray-500">First recorded combat in each stretch of fighting</p>
                    </div>
                    <NuxtLink :to="`${basePath}/systems`" class="shrink-0 text-sm text-blue-300 hover:underline">All systems →</NuxtLink>
                </div>
                <div v-if="firstEngagements.length" class="divide-y divide-white/[0.06]">
                    <div v-for="engagement in firstEngagements" :key="engagement.number" class="flex items-center gap-4 px-5 py-3">
                        <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-blue-500/10 text-xs font-semibold text-blue-300">{{ engagement.number }}</span>
                        <div class="min-w-0 flex-1">
                            <NuxtLink :to="`/system/${engagement.solar_system_id}`" class="block truncate text-sm font-medium text-gray-100 hover:text-blue-300">{{ engagement.solar_system_name }}</NuxtLink>
                            <p class="truncate text-xs text-gray-500">{{ engagement.region_name }} · {{ formatEveDateTime(engagement.start_time, true) }}</p>
                        </div>
                        <span class="shrink-0 text-xs tabular-nums"><span class="text-emerald-300">{{ engagement.kills }} K</span> · <span class="text-rose-300">{{ engagement.losses }} L</span></span>
                    </div>
                </div>
                <div v-else class="p-8 text-center text-sm text-gray-400">No recorded combat. Try extending the time window or checking the pilot list.</div>
                <p v-if="report.engagements.length > firstEngagements.length" class="border-t border-white/[0.06] px-5 py-3 text-xs text-gray-500">Showing the first {{ firstEngagements.length }} of {{ report.engagements.length }} engagements.</p>
            </section>
            <section class="glass-panel overflow-hidden">
                <div class="border-b border-white/[0.07] p-5">
                    <h2 class="text-lg font-semibold text-white">Fleet roster</h2>
                    <p class="mt-1 text-xs text-gray-500">Current affiliations and recorded activity</p>
                </div>
                <div class="divide-y divide-white/[0.06]">
                    <div v-for="pilot in topPilots" :key="pilot.character_id" class="flex items-center gap-3 px-5 py-2.5">
                        <img :src="`/images/characters/${pilot.character_id}/portrait?size=64`" :alt="pilot.name" class="h-8 w-8 rounded-full bg-gray-900" loading="lazy">
                        <div class="min-w-0 flex-1">
                            <NuxtLink :to="`/character/${pilot.character_id}`" class="block truncate text-sm text-gray-100 hover:text-blue-300">{{ pilot.name }}</NuxtLink>
                            <p class="truncate text-xs text-gray-500">{{ pilot.corporation_name || 'Unknown corporation' }}</p>
                        </div>
                        <span class="shrink-0 text-xs tabular-nums text-gray-400">{{ pilot.kill_participations }} K / {{ pilot.losses }} L</span>
                    </div>
                </div>
                <NuxtLink v-if="report.roster.length > topPilots.length" :to="`${basePath}/composition`" class="block border-t border-white/[0.06] px-5 py-3 text-sm text-blue-300 hover:bg-white/[0.03]">See all {{ report.roster.length }} pilots →</NuxtLink>
            </section>
        </div>
        <p class="text-xs leading-5 text-gray-500">This is an inferred combat trail from public killmails. It cannot show jumps without combat or prove fleet membership. Affiliations shown are current. Separate engagements begin when fighting moves systems or pauses for more than 30 minutes.</p>
    </div>
</template>
