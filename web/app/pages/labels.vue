<script setup lang="ts">
interface KillmailLabel {
    id: string
    name: string
    description: string
    category: string
    count: number
    view_url: string
    search_filters: Record<string, unknown>
}

interface LabelsResponse {
    labels: KillmailLabel[]
}

useHead({ title: 'Killmail Labels' })
useSeoMeta({
    description: 'Explore the classifications EVE-KILL applies to EVE Online killmails, with historical counts and direct search links.',
    ogTitle: 'Killmail Labels — EVE-KILL',
    ogDescription: 'Browse EVE Online killmail classifications by space, engagement, value, hull class, and technology.',
})

const { data, pending, error } = await useApiFetch<LabelsResponse>('/api/labels')
const query = ref('')

const categoryOrder = [
    'Space', 'Timezone', 'Engagement', 'Killmail Type', 'Value', 'Value Bands',
    'Victim Category', 'Victim Hull', 'Technology',
]

const filteredLabels = computed(() => {
    const needle = query.value.trim().toLowerCase()
    const labels = data.value?.labels ?? []
    if (!needle) return labels
    return labels.filter(label =>
        label.name.toLowerCase().includes(needle)
        || label.id.toLowerCase().includes(needle)
        || label.description.toLowerCase().includes(needle)
        || label.category.toLowerCase().includes(needle),
    )
})

const groups = computed(() => {
    const known = new Set(categoryOrder)
    const extra = [...new Set(filteredLabels.value.map(label => label.category))]
        .filter(category => !known.has(category))
        .sort((a, b) => a.localeCompare(b))

    return [...categoryOrder, ...extra]
    .map(category => ({
        category,
        labels: filteredLabels.value.filter(label => label.category === category),
    }))
        .filter(group => group.labels.length > 0)
})

const searchTarget = (label: KillmailLabel) => ({
    path: '/advancedsearch',
    query: { q: JSON.stringify(label.search_filters) },
})

const canSearch = (label: KillmailLabel) => Object.keys(label.search_filters ?? {}).length > 0
const formatCount = (count: number) => new Intl.NumberFormat('en-US').format(count)
</script>

<template>
    <div class="max-w-6xl mx-auto px-4 py-8">
        <header class="mb-6">
            <div class="flex items-center gap-3 mb-2">
                <span class="w-10 h-10 rounded-lg bg-blue-500/15 flex items-center justify-center">
                    <Icon name="lucide:tags" class="text-xl text-blue-400" />
                </span>
                <div>
                    <h1 class="text-2xl font-bold text-white">Killmail Labels</h1>
                    <p class="text-sm text-gray-500">The authoritative classifications used by EVE-KILL.</p>
                </div>
            </div>
            <p class="text-sm text-gray-400 max-w-3xl">
                Labels overlap by design: a solo titan loss worth ten billion ISK in nullsec belongs to several
                classifications. Counts cover the complete indexed killmail history.
            </p>
        </header>

        <div class="relative mb-6 max-w-xl">
            <Icon name="lucide:search" class="absolute left-3 top-1/2 -translate-y-1/2 text-sm text-gray-600" />
            <input
                v-model="query"
                type="search"
                placeholder="Filter labels..."
                class="w-full rounded-lg border border-white/[0.08] bg-white/[0.03] py-2.5 pl-9 pr-3 text-sm text-gray-200 placeholder:text-gray-600 focus:border-blue-500/40 focus:outline-none"
            />
        </div>

        <div v-if="pending" class="glass-panel p-8 text-center text-sm text-gray-500">
            Loading classifications...
        </div>
        <div v-else-if="error" class="glass-panel p-8 text-center text-sm text-red-400">
            Unable to load killmail labels.
        </div>
        <div v-else-if="groups.length === 0" class="glass-panel p-8 text-center text-sm text-gray-500">
            No labels match “{{ query }}”.
        </div>
        <div v-else class="space-y-7">
            <section v-for="group in groups" :key="group.category">
                <div class="flex items-baseline gap-2 mb-3">
                    <h2 class="text-sm font-bold uppercase tracking-[0.14em] text-gray-300">{{ group.category }}</h2>
                    <span class="text-fine text-gray-600">{{ group.labels.length }} labels</span>
                </div>
                <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
                    <article
                        v-for="label in group.labels"
                        :key="label.id"
                        class="rounded-lg border border-white/[0.08] bg-white/[0.02] p-4 flex flex-col gap-3"
                    >
                        <div class="flex items-start justify-between gap-3">
                            <div class="min-w-0">
                                <h3 class="text-sm font-semibold text-white">{{ label.name }}</h3>
                                <code class="text-fine text-blue-400/70">{{ label.id }}</code>
                            </div>
                            <span class="text-xs tabular-nums text-gray-400 whitespace-nowrap" :title="`${formatCount(label.count)} killmails`">
                                {{ formatCount(label.count) }}
                            </span>
                        </div>
                        <p class="text-xs leading-relaxed text-gray-500 flex-1">{{ label.description }}</p>
                        <div class="flex gap-2">
                            <NuxtLink
                                :to="label.view_url"
                                class="px-2.5 py-1.5 rounded border border-white/[0.08] bg-white/[0.03] text-xs text-gray-300 hover:border-blue-500/30 hover:text-white transition-colors"
                            >
                                View kills
                            </NuxtLink>
                            <NuxtLink
                                v-if="canSearch(label)"
                                :to="searchTarget(label)"
                                class="px-2.5 py-1.5 rounded border border-blue-500/20 bg-blue-500/[0.08] text-xs text-blue-300 hover:bg-blue-500/[0.14] transition-colors"
                            >
                                Advanced search
                            </NuxtLink>
                        </div>
                    </article>
                </div>
            </section>
        </div>
    </div>
</template>
