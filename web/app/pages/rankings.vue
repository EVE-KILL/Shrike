<script setup lang="ts">
const entityTypes = [
    ['character', 'Characters'], ['corporation', 'Corporations'], ['alliance', 'Alliances'],
    ['ship', 'Ships'], ['system', 'Systems'], ['region', 'Regions'],
] as const
const windows = [
    ['hourly', '1h'], ['weekly', '7d'], ['fourteen_days', '14d'], ['thirty_days', '30d'],
    ['ninety_days', '90d'], ['one_eighty_days', '180d'], ['one_year', '365d'], ['all_time', 'All Time'],
] as const

const route = useRoute()
const router = useRouter()
const validEntityTypes = new Set(entityTypes.map(item => item[0]))
const validWindows = new Set(windows.map(item => item[0]))
const queryValue = (value: unknown) => typeof value === 'string' ? value : ''
const initialEntityType = queryValue(route.query.entity)
const initialWindow = queryValue(route.query.window)
const entityType = ref(validEntityTypes.has(initialEntityType as any) ? initialEntityType : 'character')
const rankingWindow = ref(validWindows.has(initialWindow as any) ? initialWindow : 'all_time')

watch([entityType, rankingWindow], ([entity, window]) => {
    const query = { ...route.query, entity, window }
    if (route.query.entity === entity && route.query.window === window) return
    void router.replace({ query })
})

const { data, pending, refresh } = await useApiFetch('/api/stats/rankings', {
    query: computed(() => ({
        section: 'eve-kill', entityType: entityType.value,
        window: rankingWindow.value, limit: 50,
    })),
})
let hourlyRefresh: ReturnType<typeof setInterval> | undefined
onMounted(() => {
    hourlyRefresh = setInterval(() => {
        if (rankingWindow.value === 'hourly') void refresh()
    }, 60_000)
})
onBeforeUnmount(() => clearInterval(hourlyRefresh))
const entries = computed<any[]>(() => (data.value as any)?.entries || [])
const podium = computed(() => entries.value.slice(0, 3))
const remainingEntries = computed(() => entries.value.slice(3))

function entityImage(entry: any) {
    const id = entry.entity_id
    const paths: Record<string, string> = {
        character: `/images/characters/${id}/portrait?size=128`,
        corporation: `/images/corporations/${id}/logo?size=128`,
        alliance: `/images/alliances/${id}/logo?size=128`,
        ship: `/images/types/${id}/icon?size=128`,
        system: `/images/systems/${id}?size=128`,
        region: `/images/regions/${id}?size=128`,
    }
    return paths[entityType.value]
}

const selectedEntityLabel = computed(() => entityTypes.find(item => item[0] === entityType.value)?.[1] ?? 'Entities')
const selectedWindowLabel = computed(() => windows.find(item => item[0] === rankingWindow.value)?.[1] ?? 'All Time')

function entityLink(entry: any) {
    const paths: Record<string, string> = {
        character: 'character', corporation: 'corporation', alliance: 'alliance',
        ship: 'item', system: 'system', region: 'region',
    }
    return `/${paths[entityType.value]}/${entry.entity_id}`
}

useSeoMeta({
    title: 'EVE-KILL Rankings',
    description: 'Explore live hourly, weekly, 90-day, and all-time EVE-KILL combat rankings.',
})
</script>

<template>
    <main class="max-w-6xl mx-auto px-4 py-6 space-y-5">
        <PageHeader
            title="EVE-KILL Rankings"
            eyebrow="The combat ladder"
            icon="lucide:trophy"
            description="Combat points follow damage and participation. Character ratings add an achievement bonus worth up to 30% of the combined, uncapped rating."
        />

        <div class="flex flex-wrap gap-2 justify-between">
            <div class="flex flex-wrap gap-1.5">
                <button v-for="item in entityTypes" :key="item[0]" type="button"
                    class="px-3 py-1.5 rounded border text-xs transition-colors"
                    :class="entityType === item[0] ? 'border-blue-500/40 bg-blue-500/10 text-blue-300' : 'border-white/10 text-gray-400 hover:text-white'"
                    @click="entityType = item[0]">{{ item[1] }}</button>
            </div>
            <div class="flex gap-1.5">
                <button v-for="item in windows" :key="item[0]" type="button"
                    class="px-3 py-1.5 rounded border text-xs transition-colors"
                    :class="rankingWindow === item[0] ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-300' : 'border-white/10 text-gray-400 hover:text-white'"
                    @click="rankingWindow = item[0]">{{ item[1] }}</button>
            </div>
        </div>

        <section v-if="!pending && podium.length" class="grid gap-3 sm:grid-cols-3">
            <NuxtLink v-for="entry in podium" :key="entry.entity_id" :to="entityLink(entry)"
                class="group relative overflow-hidden rounded-xl border p-4 transition-all hover:-translate-y-0.5 hover:bg-white/[0.04]"
                :class="entry.overall_rank === 1 ? 'border-yellow-400/25 bg-yellow-400/[0.05] sm:-translate-y-1' : 'border-white/[0.08] bg-black/25'">
                <div class="flex items-center gap-3">
                    <div class="relative">
                        <EveImage :src="entityImage(entry)" :size="64" :alt="entry.name" class="h-14 w-14 rounded-lg bg-gray-900 object-cover ring-1 ring-white/10" />
                        <span class="absolute -bottom-1.5 -right-1.5 flex h-6 min-w-6 items-center justify-center rounded-full border border-black/80 px-1 text-xs font-bold"
                            :class="entry.overall_rank === 1 ? 'bg-yellow-400 text-black' : 'bg-gray-700 text-gray-200'">#{{ entry.overall_rank }}</span>
                    </div>
                    <div class="min-w-0">
                        <div class="truncate text-sm font-semibold text-white group-hover:text-blue-300">{{ entry.name }}</div>
                        <div class="mt-1 text-lg font-bold tabular-nums text-blue-300">{{ entry.eve_kill_rating.toLocaleString('en-US') }}</div>
                        <div class="text-fine uppercase tracking-wider text-gray-600">EVE-KILL rating</div>
                    </div>
                </div>
            </NuxtLink>
        </section>

        <div class="rounded-xl border border-white/10 bg-black/20 overflow-hidden">
            <div class="flex items-center justify-between border-b border-white/[0.08] px-4 py-3">
                <div><span class="text-sm font-semibold text-gray-200">{{ selectedEntityLabel }}</span><span class="ml-2 text-xs text-gray-600">{{ selectedWindowLabel }}</span></div>
                <span class="text-fine uppercase tracking-wider text-gray-600">Top 50</span>
            </div>
            <div class="hidden sm:grid grid-cols-[56px_1fr_100px_120px_100px] gap-3 px-4 py-2 border-b border-white/10 text-fine uppercase tracking-wider text-gray-600">
                <span>Rank</span><span>Entity</span><span class="text-right">Rating</span>
                <span class="text-right">Combat Points</span><span class="text-right">Achievements</span>
            </div>
            <div v-if="pending" class="p-8 text-center text-sm text-gray-500">Loading rankings…</div>
            <NuxtLink v-for="entry in remainingEntries" v-else :key="entry.entity_id" :to="entityLink(entry)"
                class="grid grid-cols-[40px_1fr_auto] sm:grid-cols-[56px_1fr_100px_120px_100px] gap-3 items-center px-4 py-2.5 border-b border-white/[0.05] last:border-0 hover:bg-white/[0.03] transition-colors">
                <span class="font-mono text-sm text-gray-500">#{{ entry.overall_rank }}</span>
                <span class="flex min-w-0 items-center gap-2.5"><EveImage :src="entityImage(entry)" :size="32" :alt="entry.name" class="h-8 w-8 shrink-0 rounded bg-gray-900 object-cover" /><span class="truncate text-sm text-gray-200">{{ entry.name }}</span></span>
                <span class="text-right font-bold tabular-nums text-blue-300">{{ entry.eve_kill_rating.toLocaleString('en-US') }}</span>
                <span class="hidden text-right text-xs tabular-nums text-gray-400 sm:block">{{ entry.combat_points.toLocaleString('en-US') }}</span>
                <span class="hidden text-right text-xs tabular-nums text-emerald-400 sm:block">{{ entry.achievement_points.toLocaleString('en-US') }}</span>
            </NuxtLink>
            <div v-if="!pending && entries.length === 0" class="p-8 text-center text-sm text-gray-500">
                {{ rankingWindow === 'hourly' ? 'No scored combat activity in the last hour.' : 'Rankings are being generated.' }}
            </div>
        </div>
    </main>
</template>
