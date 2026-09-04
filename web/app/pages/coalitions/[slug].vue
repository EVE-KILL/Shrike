<script setup lang="ts">
interface CoalitionSummary {
    coalition_id: number
    slug: string
    name: string
    description: string
    source_url: string | null
    revision: number
    created_at: string
    updated_at: string
    created_by_character_id: number | null
    created_by_character_name: string | null
    updated_by_character_id: number | null
    updated_by_character_name: string | null
    alliance_count: number
    member_count: number
    system_count: number
    kills: number
    losses: number
    isk_destroyed: number
    isk_lost: number
}

interface CoalitionAlliance {
    alliance_id: number
    name: string
    ticker: string
    member_count: number
    corporation_count: number
    system_count: number
    added_at: string
    added_by_character_id: number | null
    added_by_character_name: string | null
}

interface CoalitionEdit {
    edit_id: number
    editor_character_id: number | null
    editor_character_name: string
    action: 'seed' | 'create' | 'update'
    summary: string
    changes: Record<string, unknown>
    created_at: string
}

interface CoalitionDetailResponse {
    coalition: CoalitionSummary
    alliances: CoalitionAlliance[]
    edits: CoalitionEdit[]
    stats_window_days: number
}

interface PickedAlliance {
    id: number
    name: string
    ticker: string | null
}

const route = useRoute()
const slug = String(route.params.slug || '')
if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(slug)) {
    throw createError({ statusCode: 404, statusMessage: 'Coalition not found' })
}

const { isAuthenticated, login } = useAuth()
const { data, pending, error, refresh } = await useApiFetch<CoalitionDetailResponse>(`/api/coalitions/${slug}`)
const coalition = computed(() => data.value?.coalition ?? null)
const alliances = computed(() => data.value?.alliances ?? [])
const edits = computed(() => data.value?.edits ?? [])

useHead({
    title: computed(() => coalition.value ? `${coalition.value.name} Coalition` : 'Coalition'),
    link: [{ rel: 'canonical', href: `https://eve-kill.com/coalitions/${slug}` }],
})
useSeoMeta({
    description: computed(() => coalition.value
        ? `${coalition.value.name}: ${coalition.value.alliance_count} alliances, ${Number(coalition.value.member_count).toLocaleString('en-US')} members, territory, and recent combat statistics.`
        : 'Community-maintained EVE Online coalition membership and statistics.'),
    ogTitle: computed(() => coalition.value ? `${coalition.value.name} — EVE-KILL` : 'Coalition — EVE-KILL'),
    ogDescription: computed(() => coalition.value?.description || 'Community-maintained EVE Online coalition membership and statistics.'),
    ogType: 'website',
    ogUrl: `https://eve-kill.com/coalitions/${slug}`,
})
useSchemaOrg([
    defineBreadcrumb(computed(() => ({
        itemListElement: [
            { name: 'Home', item: '/' },
            { name: 'Coalitions', item: '/coalitions' },
            { name: coalition.value?.name || 'Coalition', item: `/coalitions/${slug}` },
        ],
    }))),
])

const formatNumber = (value: number) => Number(value || 0).toLocaleString('en-US')
const combatEfficiency = computed(() => {
    const kills = Number(coalition.value?.kills ?? 0)
    const losses = Number(coalition.value?.losses ?? 0)
    return kills + losses > 0 ? Math.round((kills / (kills + losses)) * 1000) / 10 : 0
})
const iskEfficiency = computed(() => {
    const destroyed = Number(coalition.value?.isk_destroyed ?? 0)
    const lost = Number(coalition.value?.isk_lost ?? 0)
    return destroyed + lost > 0 ? Math.round((destroyed / (destroyed + lost)) * 1000) / 10 : 0
})
const formatDateTime = (value: string) => new Intl.DateTimeFormat('en-GB', {
    dateStyle: 'medium', timeStyle: 'short', timeZone: 'UTC',
}).format(new Date(value)) + ' UTC'

const editOpen = ref(false)
const editName = ref('')
const editDescription = ref('')
const editSourceURL = ref('')
const selectedAlliances = ref<PickedAlliance[]>([])
const saving = ref(false)
const editError = ref('')

const resetEditor = () => {
    const current = coalition.value
    if (!current) return
    editName.value = current.name
    editDescription.value = current.description
    editSourceURL.value = current.source_url ?? ''
    selectedAlliances.value = alliances.value.map(alliance => ({
        id: alliance.alliance_id, name: alliance.name, ticker: alliance.ticker,
    }))
    editError.value = ''
}

watch(editOpen, open => {
    if (open) resetEditor()
})

const alliancePicked = (_type: string, id: number) => selectedAlliances.value.some(alliance => alliance.id === id)
const addAlliance = (picked: { type: string; id: number; name: string; ticker: string | null }) => {
    if (picked.type !== 'alliance' || alliancePicked(picked.type, picked.id)) return
    selectedAlliances.value.push({ id: picked.id, name: picked.name, ticker: picked.ticker })
}
const removeAlliance = (id: number) => {
    selectedAlliances.value = selectedAlliances.value.filter(alliance => alliance.id !== id)
}

const saveCoalition = async () => {
    if (!coalition.value || !isAuthenticated.value || saving.value) return
    editError.value = ''
    saving.value = true
    try {
        const result = await apiFetch<CoalitionDetailResponse>(`/api/coalitions/${slug}`, {
            method: 'PATCH',
            body: {
                revision: coalition.value.revision,
                name: editName.value,
                description: editDescription.value,
                source_url: editSourceURL.value || null,
                alliance_ids: selectedAlliances.value.map(alliance => alliance.id),
            },
        })
        data.value = result
        editOpen.value = false
    } catch (err) {
        editError.value = extractFetchError(err, 'Unable to save the coalition.')
    } finally {
        saving.value = false
    }
}

const reloadAfterConflict = async () => {
    await refresh()
    resetEditor()
}
</script>

<template>
    <div class="w-full">
        <div v-if="pending" class="glass-panel p-10 text-center text-sm text-gray-500">Loading coalition...</div>
        <div v-else-if="error || !coalition" class="glass-panel p-10 text-center">
            <Icon name="lucide:circle-alert" class="mb-3 text-2xl text-red-400" />
            <h1 class="font-semibold text-white">Coalition not found</h1>
            <NuxtLink to="/coalitions" class="mt-3 inline-block text-sm text-blue-400 hover:text-blue-300">Return to the coalition directory</NuxtLink>
        </div>
        <template v-else>
            <PageHeader class="mb-6" :title="coalition.name" eyebrow="Community-maintained coalition" icon="lucide:network"
                :description="coalition.description || 'No description has been added yet.'">
                <template #meta>
                    <div class="flex flex-wrap items-center gap-2 text-xs">
                        <span class="rounded border border-white/[0.08] bg-black/20 px-2.5 py-1.5 text-gray-400">Revision {{ coalition.revision }}</span>
                        <a v-if="coalition.source_url" :href="coalition.source_url" target="_blank" rel="noopener noreferrer"
                            class="inline-flex items-center gap-1 rounded border border-blue-400/15 bg-blue-500/[0.06] px-2.5 py-1.5 text-blue-300 hover:bg-blue-500/10">
                            Verification source <Icon name="lucide:external-link" />
                        </a>
                        <a :href="`/api/coalitions/${slug}`" target="_blank"
                            class="inline-flex items-center gap-1 rounded border border-white/[0.08] bg-black/20 px-2.5 py-1.5 text-gray-400 hover:text-white">
                            Public API <Icon name="lucide:braces" />
                        </a>
                    </div>
                </template>
            </PageHeader>

            <div class="mb-6 flex justify-end">
                <ClientOnly>
                    <button v-if="isAuthenticated" type="button" @click="editOpen = !editOpen"
                        class="inline-flex items-center gap-2 rounded-lg border border-blue-400/20 bg-blue-500/10 px-4 py-2 text-sm font-medium text-blue-300 hover:bg-blue-500/15">
                        <Icon :name="editOpen ? 'lucide:x' : 'lucide:pencil'" /> {{ editOpen ? 'Close editor' : 'Edit coalition' }}
                    </button>
                    <button v-else type="button" @click="login({ redirect: `/coalitions/${slug}` })"
                        class="inline-flex items-center gap-2 rounded-lg border border-white/[0.08] bg-white/[0.03] px-4 py-2 text-sm text-gray-300 hover:text-blue-300">
                        <Icon name="lucide:log-in" /> Sign in to edit
                    </button>
                </ClientOnly>
            </div>

            <ClientOnly>
                <form v-if="editOpen && isAuthenticated" class="glass-panel mb-7 p-5 sm:p-6" @submit.prevent="saveCoalition">
                    <div class="mb-5">
                        <h2 class="font-semibold text-white">Edit {{ coalition.name }}</h2>
                        <p class="mt-1 text-xs text-gray-500">The complete change and your character name will be added to the public edit history.</p>
                    </div>
                    <div class="grid gap-4 lg:grid-cols-2">
                        <label class="block">
                            <span class="mb-1.5 block text-xs font-medium text-gray-400">Name</span>
                            <input v-model="editName" required minlength="2" maxlength="100"
                                class="w-full rounded-lg border border-white/[0.08] bg-black/20 px-3 py-2.5 text-sm text-white outline-none focus:border-blue-500/40" />
                        </label>
                        <label class="block">
                            <span class="mb-1.5 block text-xs font-medium text-gray-400">Verification source</span>
                            <input v-model="editSourceURL" type="url" maxlength="2048" placeholder="https://..."
                                class="w-full rounded-lg border border-white/[0.08] bg-black/20 px-3 py-2.5 text-sm text-white outline-none focus:border-blue-500/40" />
                        </label>
                        <label class="block lg:col-span-2">
                            <span class="mb-1.5 block text-xs font-medium text-gray-400">Description</span>
                            <textarea v-model="editDescription" maxlength="2000" rows="3"
                                class="w-full resize-y rounded-lg border border-white/[0.08] bg-black/20 px-3 py-2.5 text-sm text-white outline-none focus:border-blue-500/40" />
                        </label>
                        <div class="lg:col-span-2">
                            <div class="mb-1.5 flex items-center justify-between">
                                <span class="text-xs font-medium text-gray-400">Member alliances</span>
                                <span class="text-fine text-gray-600">{{ selectedAlliances.length }} selected</span>
                            </div>
                            <SearchPicker :types="['alliance']" :is-picked="alliancePicked" placeholder="Add an alliance..." @select="addAlliance" />
                            <div class="mt-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
                                <div v-for="alliance in selectedAlliances" :key="alliance.id"
                                    class="flex items-center gap-2 rounded-lg border border-white/[0.06] bg-black/20 p-2">
                                    <EveImage :src="`/images/alliances/${alliance.id}/logo?size=64`" :size="32" :alt="alliance.name" class="h-8 w-8 shrink-0 rounded" />
                                    <div class="min-w-0 flex-1">
                                        <div class="truncate text-xs text-gray-200">{{ alliance.name }}</div>
                                        <div class="text-fine text-gray-600">[{{ alliance.ticker }}] · {{ alliance.id }}</div>
                                    </div>
                                    <button type="button" class="p-1 text-gray-600 hover:text-red-400" @click="removeAlliance(alliance.id)"><Icon name="lucide:x" /></button>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div v-if="editError" class="mt-4 rounded-lg border border-red-500/20 bg-red-500/10 px-3 py-2 text-sm text-red-300">
                        {{ editError }}
                        <button v-if="editError.includes('reload')" type="button" class="ml-2 underline hover:text-white" @click="reloadAfterConflict">Reload now</button>
                    </div>
                    <div class="mt-5 flex justify-end">
                        <button type="submit" :disabled="saving || editName.trim().length < 2 || selectedAlliances.length === 0"
                            class="inline-flex items-center gap-2 rounded-lg bg-blue-500 px-4 py-2.5 text-sm font-semibold text-white hover:bg-blue-400 disabled:cursor-not-allowed disabled:opacity-40">
                            <Icon v-if="saving" name="lucide:loader-2" class="animate-spin" />
                            <Icon v-else name="lucide:save" /> {{ saving ? 'Saving...' : 'Save edit' }}
                        </button>
                    </div>
                </form>
            </ClientOnly>

            <section class="mb-7 grid grid-cols-2 gap-3 sm:grid-cols-4 xl:grid-cols-8">
                <div v-for="stat in [
                    { label: 'Members', value: formatNumber(coalition.member_count), color: 'text-white' },
                    { label: 'Alliances', value: formatNumber(coalition.alliance_count), color: 'text-white' },
                    { label: 'Systems', value: formatNumber(coalition.system_count), color: 'text-blue-300' },
                    { label: 'Kills', value: formatNumber(coalition.kills), color: 'text-green-400' },
                    { label: 'Losses', value: formatNumber(coalition.losses), color: 'text-red-400' },
                    { label: 'Destroyed', value: formatIsk(coalition.isk_destroyed), color: 'text-yellow-400' },
                    { label: 'Combat eff.', value: `${combatEfficiency}%`, color: 'text-gray-200' },
                    { label: 'ISK eff.', value: `${iskEfficiency}%`, color: 'text-gray-200' },
                ]" :key="stat.label" class="glass-panel px-3 py-4 text-center">
                    <div class="font-mono text-sm font-semibold tabular-nums" :class="stat.color">{{ stat.value }}</div>
                    <div class="mt-1 text-fine uppercase tracking-wide text-gray-600">{{ stat.label }}</div>
                </div>
            </section>
            <p class="-mt-5 mb-7 text-right text-fine text-gray-600">Combat statistics cover the last {{ data?.stats_window_days ?? 30 }} days.</p>

            <div class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_22rem]">
                <section>
                    <div class="mb-3 flex items-center justify-between">
                        <h2 class="flex items-center gap-2 text-sm font-semibold uppercase tracking-[0.12em] text-gray-300"><Icon name="lucide:shield" class="text-blue-400" /> Member alliances</h2>
                        <span class="text-xs text-gray-600">{{ alliances.length }} total</span>
                    </div>
                    <div class="glass-panel overflow-hidden">
                        <div v-if="alliances.length === 0" class="p-8 text-center text-sm text-gray-500">No alliances are currently assigned.</div>
                        <template v-else>
                            <NuxtLink v-for="alliance in alliances" :key="alliance.alliance_id" :to="`/alliance/${alliance.alliance_id}`"
                                class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-4 border-b border-white/[0.05] px-4 py-3 last:border-b-0 hover:bg-blue-500/[0.035] sm:grid-cols-[minmax(0,1fr)_8rem_7rem_6rem]">
                                <div class="flex min-w-0 items-center gap-3">
                                    <EveImage :src="`/images/alliances/${alliance.alliance_id}/logo?size=64`" :size="64" :alt="alliance.name" class="h-10 w-10 shrink-0 rounded" />
                                    <div class="min-w-0">
                                        <div class="truncate text-sm font-medium text-gray-200">{{ alliance.name }}</div>
                                        <div class="text-fine font-mono text-gray-600">[{{ alliance.ticker }}] · {{ alliance.alliance_id }}</div>
                                    </div>
                                </div>
                                <div class="text-right"><div class="text-sm tabular-nums text-gray-300">{{ formatNumber(alliance.member_count) }}</div><div class="text-fine text-gray-600">members</div></div>
                                <div class="hidden text-right sm:block"><div class="text-sm tabular-nums text-gray-300">{{ formatNumber(alliance.corporation_count) }}</div><div class="text-fine text-gray-600">corps</div></div>
                                <div class="hidden text-right sm:block"><div class="text-sm tabular-nums text-blue-300">{{ formatNumber(alliance.system_count) }}</div><div class="text-fine text-gray-600">systems</div></div>
                            </NuxtLink>
                        </template>
                    </div>
                </section>

                <aside>
                    <h2 class="mb-3 flex items-center gap-2 text-sm font-semibold uppercase tracking-[0.12em] text-gray-300"><Icon name="lucide:history" class="text-purple-400" /> Edit history</h2>
                    <div class="glass-panel overflow-hidden">
                        <div v-if="edits.length === 0" class="p-6 text-center text-sm text-gray-500">No recorded edits.</div>
                        <template v-else>
                            <article v-for="edit in edits" :key="edit.edit_id" class="border-b border-white/[0.05] p-4 last:border-b-0">
                                <div class="flex items-start gap-3">
                                    <EveImage v-if="edit.editor_character_id" :src="`/images/characters/${edit.editor_character_id}/portrait?size=64`" :size="32" :alt="edit.editor_character_name" class="h-8 w-8 shrink-0 rounded-full" />
                                    <div v-else class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-white/[0.05] text-gray-500"><Icon name="lucide:bot" /></div>
                                    <div class="min-w-0">
                                        <p class="text-xs leading-relaxed text-gray-400">
                                            <NuxtLink v-if="edit.editor_character_id" :to="`/character/${edit.editor_character_id}`" class="font-medium text-blue-300 hover:text-blue-200">{{ edit.editor_character_name }}</NuxtLink>
                                            <span v-else class="font-medium text-gray-300">{{ edit.editor_character_name }}</span>
                                            {{ edit.action === 'update' ? ` edited ${coalition.name}` : edit.action === 'create' ? ` created ${coalition.name}` : ` imported ${coalition.name}` }}
                                        </p>
                                        <p class="mt-1 text-xs text-gray-500">{{ edit.summary }}</p>
                                        <time :datetime="edit.created_at" class="mt-1.5 block text-fine text-gray-600">{{ formatDateTime(edit.created_at) }}</time>
                                    </div>
                                </div>
                            </article>
                        </template>
                    </div>
                </aside>
            </div>
        </template>
    </div>
</template>
