<script setup lang="ts">
interface CoalitionSummary {
    coalition_id: number
    slug: string
    name: string
    description: string
    source_url: string | null
    revision: number
    alliance_count: number
    member_count: number
    system_count: number
    kills: number
    losses: number
    isk_destroyed: number
    isk_lost: number
    updated_at: string
    updated_by_character_id: number | null
    updated_by_character_name: string | null
}

interface CoalitionListResponse {
    coalitions: CoalitionSummary[]
    stats_window_days: number
}

interface PickedAlliance {
    id: number
    name: string
    ticker: string | null
}

interface CoalitionDetailResponse {
    coalition: CoalitionSummary
}

useHead({
    title: 'EVE Online Coalitions',
    link: [{ rel: 'canonical', href: 'https://eve-kill.com/coalitions' }],
})
useSeoMeta({
    description: 'Browse community-maintained EVE Online coalitions, their member alliances, territory, membership, and recent combat activity.',
    ogTitle: 'EVE Online Coalitions — EVE-KILL',
    ogDescription: 'A public, community-maintained directory of EVE Online coalitions and their member alliances.',
    ogType: 'website',
    ogUrl: 'https://eve-kill.com/coalitions',
})
useSchemaOrg([
    defineWebPage({
        '@type': 'CollectionPage',
        'name': 'EVE Online Coalitions',
        'description': 'Community-maintained coalition membership and recent coalition statistics.',
        'url': 'https://eve-kill.com/coalitions',
    }),
    defineBreadcrumb({
        itemListElement: [
            { name: 'Home', item: '/' },
            { name: 'Coalitions', item: '/coalitions' },
        ],
    }),
])

const { isAuthenticated, login } = useAuth()
const { data, pending, error } = await useApiFetch<CoalitionListResponse>('/api/coalitions', {
    default: () => ({ coalitions: [], stats_window_days: 30 }),
})

const query = ref('')
const coalitions = computed(() => {
    const needle = query.value.trim().toLowerCase()
    const rows = data.value?.coalitions ?? []
    if (!needle) return rows
    return rows.filter(coalition =>
        coalition.name.toLowerCase().includes(needle)
        || coalition.description.toLowerCase().includes(needle),
    )
})

const formatNumber = (value: number) => Number(value || 0).toLocaleString('en-US')
const efficiency = (coalition: CoalitionSummary) => {
    const total = Number(coalition.kills) + Number(coalition.losses)
    return total > 0 ? Math.round((Number(coalition.kills) / total) * 1000) / 10 : 0
}

const createOpen = ref(false)
const createName = ref('')
const createDescription = ref('')
const createSourceURL = ref('')
const selectedAlliances = ref<PickedAlliance[]>([])
const creating = ref(false)
const createError = ref('')

const alliancePicked = (_type: string, id: number) => selectedAlliances.value.some(alliance => alliance.id === id)
const addAlliance = (picked: { type: string; id: number; name: string; ticker: string | null }) => {
    if (picked.type !== 'alliance' || alliancePicked(picked.type, picked.id)) return
    selectedAlliances.value.push({ id: picked.id, name: picked.name, ticker: picked.ticker })
}
const removeAlliance = (id: number) => {
    selectedAlliances.value = selectedAlliances.value.filter(alliance => alliance.id !== id)
}

const createCoalition = async () => {
    if (!isAuthenticated.value || creating.value) return
    createError.value = ''
    creating.value = true
    try {
        const result = await apiFetch<CoalitionDetailResponse>('/api/coalitions', {
            method: 'POST',
            body: {
                name: createName.value,
                description: createDescription.value,
                source_url: createSourceURL.value || null,
                alliance_ids: selectedAlliances.value.map(alliance => alliance.id),
            },
        })
        await navigateTo(`/coalitions/${result.coalition.slug}`)
    } catch (err) {
        createError.value = extractFetchError(err, 'Unable to create the coalition.')
    } finally {
        creating.value = false
    }
}
</script>

<template>
    <div class="max-w-7xl mx-auto px-4 py-8">
        <PageHeader class="mb-6" title="Coalitions" eyebrow="Community-maintained politics" icon="lucide:network"
            description="Coalitions are player-created political blocs rather than official EVE entities. This directory is maintained by signed-in pilots, with every change recorded publicly.">
            <template #meta>
                <div class="flex flex-wrap gap-2 text-xs text-gray-500">
                    <span class="rounded border border-white/[0.08] bg-black/20 px-2.5 py-1.5">
                        <strong class="text-gray-300">{{ data?.coalitions?.length ?? 0 }}</strong> coalitions
                    </span>
                    <span class="rounded border border-white/[0.08] bg-black/20 px-2.5 py-1.5">
                        <strong class="text-gray-300">{{ formatNumber((data?.coalitions ?? []).reduce((sum, coalition) => sum + Number(coalition.alliance_count), 0)) }}</strong> alliance records
                    </span>
                    <span class="rounded border border-white/[0.08] bg-black/20 px-2.5 py-1.5">
                        {{ data?.stats_window_days ?? 30 }}-day combat window
                    </span>
                </div>
            </template>
        </PageHeader>

        <div class="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div class="relative w-full max-w-xl">
                <Icon name="lucide:search" class="absolute left-3 top-1/2 -translate-y-1/2 text-sm text-gray-600" />
                <input v-model="query" type="search" placeholder="Filter coalitions..."
                    class="w-full rounded-lg border border-white/[0.08] bg-white/[0.03] py-2.5 pl-9 pr-3 text-sm text-gray-200 placeholder:text-gray-600 focus:border-blue-500/40 focus:outline-none" />
            </div>
            <ClientOnly>
                <button v-if="isAuthenticated" type="button" @click="createOpen = !createOpen"
                    class="inline-flex items-center justify-center gap-2 rounded-lg border border-blue-400/20 bg-blue-500/10 px-4 py-2.5 text-sm font-medium text-blue-300 transition-colors hover:bg-blue-500/15">
                    <Icon :name="createOpen ? 'lucide:x' : 'lucide:plus'" />
                    {{ createOpen ? 'Close creator' : 'Add coalition' }}
                </button>
                <button v-else type="button" @click="login({ redirect: '/coalitions' })"
                    class="inline-flex items-center justify-center gap-2 rounded-lg border border-white/[0.08] bg-white/[0.03] px-4 py-2.5 text-sm text-gray-300 transition-colors hover:border-blue-400/20 hover:text-blue-300">
                    <Icon name="lucide:log-in" /> Sign in to contribute
                </button>
            </ClientOnly>
        </div>

        <ClientOnly>
            <form v-if="createOpen && isAuthenticated" class="glass-panel mb-7 p-5 sm:p-6" @submit.prevent="createCoalition">
                <div class="mb-5 flex items-start gap-3">
                    <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-500/10 text-blue-400">
                        <Icon name="lucide:network" />
                    </div>
                    <div>
                        <h2 class="font-semibold text-white">Add a coalition</h2>
                        <p class="mt-1 text-xs text-gray-500">Add a public source when possible. Your character will be attached to the creation record.</p>
                    </div>
                </div>
                <div class="grid gap-4 lg:grid-cols-2">
                    <label class="block">
                        <span class="mb-1.5 block text-xs font-medium text-gray-400">Name</span>
                        <input v-model="createName" required minlength="2" maxlength="100" placeholder="Coalition name"
                            class="w-full rounded-lg border border-white/[0.08] bg-black/20 px-3 py-2.5 text-sm text-white outline-none focus:border-blue-500/40" />
                    </label>
                    <label class="block">
                        <span class="mb-1.5 block text-xs font-medium text-gray-400">Verification source</span>
                        <input v-model="createSourceURL" type="url" maxlength="2048" placeholder="https://..."
                            class="w-full rounded-lg border border-white/[0.08] bg-black/20 px-3 py-2.5 text-sm text-white outline-none focus:border-blue-500/40" />
                    </label>
                    <label class="block lg:col-span-2">
                        <span class="mb-1.5 block text-xs font-medium text-gray-400">Description</span>
                        <textarea v-model="createDescription" maxlength="2000" rows="3" placeholder="What this coalition represents..."
                            class="w-full resize-y rounded-lg border border-white/[0.08] bg-black/20 px-3 py-2.5 text-sm text-white outline-none focus:border-blue-500/40" />
                    </label>
                    <div class="lg:col-span-2">
                        <span class="mb-1.5 block text-xs font-medium text-gray-400">Member alliances</span>
                        <SearchPicker :types="['alliance']" :is-picked="alliancePicked" placeholder="Search for an alliance..." @select="addAlliance" />
                        <div v-if="selectedAlliances.length" class="mt-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
                            <div v-for="alliance in selectedAlliances" :key="alliance.id"
                                class="flex items-center gap-2 rounded-lg border border-white/[0.06] bg-black/20 p-2">
                                <EveImage :src="`/images/alliances/${alliance.id}/logo?size=64`" :size="32" :alt="alliance.name" class="h-8 w-8 shrink-0 rounded" />
                                <div class="min-w-0 flex-1">
                                    <div class="truncate text-xs text-gray-200">{{ alliance.name }}</div>
                                    <div class="text-fine text-gray-600">[{{ alliance.ticker }}] · {{ alliance.id }}</div>
                                </div>
                                <button type="button" class="p-1 text-gray-600 hover:text-red-400" @click="removeAlliance(alliance.id)">
                                    <Icon name="lucide:x" />
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
                <div v-if="createError" class="mt-4 rounded-lg border border-red-500/20 bg-red-500/10 px-3 py-2 text-sm text-red-300">{{ createError }}</div>
                <div class="mt-5 flex justify-end">
                    <button type="submit" :disabled="creating || createName.trim().length < 2 || selectedAlliances.length === 0"
                        class="inline-flex items-center gap-2 rounded-lg bg-blue-500 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-blue-400 disabled:cursor-not-allowed disabled:opacity-40">
                        <Icon v-if="creating" name="lucide:loader-2" class="animate-spin" />
                        <Icon v-else name="lucide:plus" />
                        {{ creating ? 'Creating...' : 'Create coalition' }}
                    </button>
                </div>
            </form>
        </ClientOnly>

        <div v-if="pending" class="glass-panel p-10 text-center text-sm text-gray-500">Loading coalitions...</div>
        <div v-else-if="error" class="glass-panel p-10 text-center text-sm text-red-400">Unable to load the coalition directory.</div>
        <div v-else-if="coalitions.length === 0" class="glass-panel p-10 text-center text-sm text-gray-500">
            {{ query ? `No coalitions match “${query}”.` : 'No coalitions have been added yet.' }}
        </div>
        <div v-else class="grid gap-4 lg:grid-cols-2">
            <NuxtLink v-for="coalition in coalitions" :key="coalition.coalition_id" :to="`/coalitions/${coalition.slug}`"
                class="group glass-panel overflow-hidden p-5 transition-colors hover:border-blue-400/20 hover:bg-blue-500/[0.025]">
                <div class="flex items-start justify-between gap-4">
                    <div class="min-w-0">
                        <div class="mb-1 flex items-center gap-2">
                            <Icon name="lucide:network" class="text-blue-400" />
                            <h2 class="truncate text-lg font-semibold text-white transition-colors group-hover:text-blue-300">{{ coalition.name }}</h2>
                        </div>
                        <p class="line-clamp-2 min-h-10 text-xs leading-relaxed text-gray-500">{{ coalition.description || 'No description has been added yet.' }}</p>
                    </div>
                    <Icon name="lucide:arrow-up-right" class="shrink-0 text-gray-600 transition-colors group-hover:text-blue-400" />
                </div>
                <div class="mt-5 grid grid-cols-3 gap-2 border-y border-white/[0.05] py-4 text-center">
                    <div><div class="font-mono text-base font-semibold text-white">{{ formatNumber(coalition.member_count) }}</div><div class="text-fine uppercase tracking-wide text-gray-600">Members</div></div>
                    <div><div class="font-mono text-base font-semibold text-white">{{ formatNumber(coalition.alliance_count) }}</div><div class="text-fine uppercase tracking-wide text-gray-600">Alliances</div></div>
                    <div><div class="font-mono text-base font-semibold text-white">{{ formatNumber(coalition.system_count) }}</div><div class="text-fine uppercase tracking-wide text-gray-600">Systems</div></div>
                </div>
                <div class="mt-4 grid grid-cols-2 gap-x-5 gap-y-2 text-xs">
                    <div class="flex justify-between"><span class="text-gray-600">Kills</span><span class="tabular-nums text-green-400">{{ formatNumber(coalition.kills) }}</span></div>
                    <div class="flex justify-between"><span class="text-gray-600">Losses</span><span class="tabular-nums text-red-400">{{ formatNumber(coalition.losses) }}</span></div>
                    <div class="flex justify-between"><span class="text-gray-600">Destroyed</span><span class="tabular-nums text-yellow-400">{{ formatIsk(coalition.isk_destroyed) }}</span></div>
                    <div class="flex justify-between"><span class="text-gray-600">Efficiency</span><span class="tabular-nums text-gray-300">{{ efficiency(coalition) }}%</span></div>
                </div>
                <div class="mt-4 flex items-center justify-between text-fine text-gray-600">
                    <span>Revision {{ coalition.revision }}</span>
                    <span v-if="coalition.updated_by_character_name">Last edited by {{ coalition.updated_by_character_name }}</span>
                    <span v-else>Initial verified import</span>
                </div>
            </NuxtLink>
        </div>
    </div>
</template>
