<script setup lang="ts">
import type { CommentRow } from '~/composables/useCommentThread'

interface FeedComment extends CommentRow {
    reply_count: number
}

interface FeedPage {
    comments: FeedComment[]
    nextCursor: number | null
}

useSeoMeta({
    title: 'Comments — EVE-KILL',
    description: 'Site-wide comment feed across killmails, characters, corporations, alliances, systems, and battles.',
})

const { user, isAuthenticated } = useAuth()

// ── Filter state ─────────────────────────────────────────────────────────────
type FilterType = 'all' | 'killmail' | 'character' | 'corporation' | 'alliance' | 'system' | 'battle' | 'fit'
const sortBy = ref<'newest' | 'oldest'>('newest')
const filterType = ref<FilterType>('all')
const search = ref('')
const debouncedSearch = ref('')
let debounceTimer: ReturnType<typeof setTimeout> | null = null
watch(search, (v) => {
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(() => { debouncedSearch.value = v.trim() }, 300)
})

const TARGET_TYPE_MAP: Record<Exclude<FilterType, 'all'>, number> = {
    killmail: 1, character: 2, corporation: 3, alliance: 4, system: 5, battle: 7, fit: 8,
}

// Type definitions used across the page (chips, filter pills, target links).
interface TargetMeta {
    key: FilterType
    label: string
    plural: string
    icon: string
    /** Tailwind text class — used for filter pill icon, chip icon. */
    color: string
    /** Tailwind classes applied when chip is active/highlighted. */
    chipBg: string
    chipBorder: string
    chipText: string
}
const TYPE_META: Record<Exclude<FilterType, 'all'>, TargetMeta> = {
    killmail:    { key: 'killmail',    label: 'Killmail',    plural: 'Killmails',    icon: 'lucide:swords',   color: 'text-red-400',    chipBg: 'bg-red-500/[0.08]',    chipBorder: 'border-red-500/20',    chipText: 'text-red-300' },
    character:   { key: 'character',   label: 'Character',   plural: 'Characters',   icon: 'lucide:user',     color: 'text-blue-400',   chipBg: 'bg-blue-500/[0.08]',   chipBorder: 'border-blue-500/20',   chipText: 'text-blue-300' },
    corporation: { key: 'corporation', label: 'Corporation', plural: 'Corporations', icon: 'lucide:building', color: 'text-amber-400',  chipBg: 'bg-amber-500/[0.08]',  chipBorder: 'border-amber-500/20',  chipText: 'text-amber-300' },
    alliance:    { key: 'alliance',    label: 'Alliance',    plural: 'Alliances',    icon: 'lucide:flag',     color: 'text-purple-400', chipBg: 'bg-purple-500/[0.08]', chipBorder: 'border-purple-500/20', chipText: 'text-purple-300' },
    system:      { key: 'system',      label: 'System',      plural: 'Systems',      icon: 'lucide:globe',    color: 'text-emerald-400',chipBg: 'bg-emerald-500/[0.08]',chipBorder: 'border-emerald-500/20',chipText: 'text-emerald-300' },
    battle:      { key: 'battle',      label: 'Battle',      plural: 'Battles',      icon: 'lucide:shield',   color: 'text-orange-400', chipBg: 'bg-orange-500/[0.08]', chipBorder: 'border-orange-500/20', chipText: 'text-orange-300' },
    fit:         { key: 'fit',         label: 'Fit',         plural: 'Fits',         icon: 'lucide:wrench',   color: 'text-cyan-400',   chipBg: 'bg-cyan-500/[0.08]',   chipBorder: 'border-cyan-500/20',   chipText: 'text-cyan-300' },
}

const FILTER_PILLS: { key: FilterType; label: string; icon: string; color: string }[] = [
    { key: 'all',         label: 'All',          icon: 'lucide:layers',  color: 'text-gray-400' },
    { key: 'killmail',    label: 'Killmails',    icon: TYPE_META.killmail.icon,    color: TYPE_META.killmail.color },
    { key: 'character',   label: 'Characters',   icon: TYPE_META.character.icon,   color: TYPE_META.character.color },
    { key: 'corporation', label: 'Corporations', icon: TYPE_META.corporation.icon, color: TYPE_META.corporation.color },
    { key: 'alliance',    label: 'Alliances',    icon: TYPE_META.alliance.icon,    color: TYPE_META.alliance.color },
    { key: 'system',      label: 'Systems',      icon: TYPE_META.system.icon,      color: TYPE_META.system.color },
    { key: 'battle',      label: 'Battles',      icon: TYPE_META.battle.icon,      color: TYPE_META.battle.color },
    { key: 'fit',         label: 'Fits',         icon: TYPE_META.fit.icon,         color: TYPE_META.fit.color },
]

const hasActiveFilters = computed(() =>
    sortBy.value !== 'newest' || filterType.value !== 'all' || debouncedSearch.value.length > 0,
)

function clearAllFilters() {
    sortBy.value = 'newest'
    filterType.value = 'all'
    search.value = ''
    debouncedSearch.value = ''
}

// ── Data fetch (cursor pagination) ───────────────────────────────────────────
// We accumulate pages in `allComments`. Whenever a filter changes the watcher
// resets the list and re-fetches from page 1. Load-More appends.
const PAGE_SIZE = 30
const allComments = ref<FeedComment[]>([])
const cursor = ref<number | null>(null)
const hasMore = ref(false)
const loading = ref(false)
const loadError = ref<string | null>(null)

async function fetchPage(reset: boolean) {
    if (loading.value) return
    loading.value = true
    loadError.value = null
    try {
        const params = new URLSearchParams({ limit: String(PAGE_SIZE) })
        if (debouncedSearch.value.length >= 2) params.set('q', debouncedSearch.value)
        if (filterType.value !== 'all') params.set('target_type', String(TARGET_TYPE_MAP[filterType.value]))
        if (!reset && cursor.value) params.set('cursor', String(cursor.value))

        const data = await apiFetch<FeedPage>(
            `/api/comments?${params.toString()}`,
        )
        if (reset) {
            allComments.value = data.comments
        } else {
            // De-dupe in case live edits arrived between pages.
            const seen = new Set(allComments.value.map((c) => c.id))
            for (const c of data.comments) if (!seen.has(c.id)) allComments.value.push(c)
        }
        cursor.value = data.nextCursor
        hasMore.value = data.nextCursor !== null
    } catch (err: any) {
        loadError.value = err?.statusMessage || err?.message || 'Failed to load comments'
    } finally {
        loading.value = false
    }
}

// Initial load is payload-backed so SSR and hydration start from the same
// comments instead of repeating a side-effect-only handler on the client.
const { data: initialPage, error: initialError } = await useAsyncData<FeedPage>(
    'comments-feed-initial',
    () => apiFetch<FeedPage>(`/api/comments?limit=${PAGE_SIZE}`),
)
if (initialPage.value) {
    allComments.value = initialPage.value.comments
    cursor.value = initialPage.value.nextCursor
    hasMore.value = initialPage.value.nextCursor !== null
} else if (initialError.value) {
    loadError.value = initialError.value.statusMessage || initialError.value.message || 'Failed to load comments'
}

// Re-fetch from page 1 when filters change.
watch([debouncedSearch, filterType], () => { fetchPage(true) })

const comments = computed(() => {
    if (sortBy.value === 'oldest') return [...allComments.value].reverse()
    return allComments.value
})

async function refreshFeed() {
    await fetchPage(true)
}
async function loadMore() {
    if (!hasMore.value) return
    await fetchPage(false)
}

// ── Helpers ──────────────────────────────────────────────────────────────────
interface TargetLink {
    label: string
    href: string
    meta: TargetMeta | null
}
function targetLink(c: FeedComment): TargetLink {
    switch (c.target_type) {
        case 1: return { label: `Killmail #${c.target_id}`, href: `/kill/${c.target_id}#comment-${c.id}`, meta: TYPE_META.killmail }
        case 2: return { label: `Character profile`, href: `/character/${c.target_id}`, meta: TYPE_META.character }
        case 3: return { label: `Corporation profile`, href: `/corporation/${c.target_id}`, meta: TYPE_META.corporation }
        case 4: return { label: `Alliance profile`, href: `/alliance/${c.target_id}`, meta: TYPE_META.alliance }
        case 5: return { label: `Solar system`, href: `/system/${c.target_id}`, meta: TYPE_META.system }
        case 7: return { label: `Battle #${c.target_id}`, href: `/battle/${c.target_id}/comments#comment-${c.id}`, meta: TYPE_META.battle }
        case 8: return {
            label: c.target_slug ? `Fit ${c.target_slug}` : 'Fit',
            href: c.target_slug ? `/fit/${c.target_slug}#comment-${c.id}` : '#',
            meta: TYPE_META.fit,
        }
        default: return { label: 'Page', href: '#', meta: null }
    }
}

function timeAgo(iso: string): string {
    const d = new Date(iso).getTime()
    const diff = Date.now() - d
    if (diff < 60_000) return 'just now'
    if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`
    if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`
    if (diff < 30 * 86_400_000) return `${Math.floor(diff / 86_400_000)}d ago`
    // Past a month the relative form stops being useful and we print a date.
    // toLocaleDateString made that date depend on the runtime's locale — it was
    // rendering "28.5.2026" here — so it disagreed between server and client.
    return formatEveDate(iso)
}

// ── Delete ───────────────────────────────────────────────────────────────────
const deletingId = ref<number | null>(null)
async function deleteOwn(id: number) {
    if (!confirm('Delete this comment?')) return
    deletingId.value = id
    try {
        await apiFetch(`/api/comments/${id}`, { method: 'DELETE' })
        // Drop locally rather than refetching everything.
        allComments.value = allComments.value.filter((c) => c.id !== id)
    } catch (err: any) {
        alert(err?.statusMessage || err?.message || 'Failed to delete')
    } finally {
        deletingId.value = null
    }
}
</script>

<template>
    <div class="w-full">
        <PageHeader class="mb-4" title="Comments" eyebrow="Community activity" icon="lucide:message-square"
            description="Latest activity across killmails, entities, systems, and battles.">
            <template #actions>
            <div class="flex items-center gap-2">
                <span v-if="comments.length" class="text-xs text-gray-500 px-2 py-1 rounded-md bg-white/[0.03] border border-white/[0.06]">
                    <span class="text-gray-300 font-semibold">{{ comments.length }}</span> shown
                </span>
                <button
                    class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs text-gray-400 hover:text-white hover:bg-white/[0.06] border border-white/[0.06] transition-colors cursor-pointer"
                    :disabled="loading"
                    @click="refreshFeed"
                >
                    <Icon name="lucide:refresh-cw" class="text-xs" :class="{ 'animate-spin': loading }" />
                    Refresh
                </button>
            </div>
            </template>
        </PageHeader>

        <!-- ═══════════════ FILTERS ═══════════════ -->
        <div class="mb-6 rounded-xl border border-white/[0.08] bg-white/[0.02] p-3">
            <!-- Type pills (segmented) -->
            <div class="flex flex-wrap gap-1.5">
                <button
                    v-for="pill in FILTER_PILLS"
                    :key="pill.key"
                    class="filter-pill"
                    :class="filterType === pill.key ? 'filter-pill-active' : ''"
                    @click="filterType = pill.key"
                >
                    <Icon :name="pill.icon" class="text-xs" :class="filterType === pill.key ? pill.color : 'text-gray-500'" />
                    {{ pill.label }}
                </button>
            </div>

            <!-- Search + sort row -->
            <div class="mt-2.5 pt-2.5 border-t border-white/[0.05] flex items-center gap-2 flex-wrap">
                <div class="relative flex-1 min-w-[200px]">
                    <Icon name="lucide:search" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 text-sm pointer-events-none" />
                    <input
                        v-model="search"
                        type="search"
                        placeholder="Search comment text…"
                        class="filter-input pl-9 pr-9"
                    >
                    <button
                        v-if="search"
                        class="absolute right-2 top-1/2 -translate-y-1/2 p-1 rounded text-gray-500 hover:text-white hover:bg-white/[0.06] cursor-pointer"
                        @click="search = ''; debouncedSearch = ''"
                    >
                        <Icon name="lucide:x" class="text-xs" />
                    </button>
                </div>

                <select v-model="sortBy" class="filter-input w-auto pr-8">
                    <option value="newest">Newest first</option>
                    <option value="oldest">Oldest first</option>
                </select>

                <button
                    v-if="hasActiveFilters"
                    class="flex items-center gap-1 px-2.5 py-2 rounded-md text-xs text-blue-400 hover:text-blue-300 hover:bg-blue-500/[0.08] cursor-pointer transition-colors"
                    @click="clearAllFilters"
                >
                    <Icon name="lucide:x" class="text-xs" />
                    Clear
                </button>
            </div>
        </div>

        <!-- ═══════════════ ERROR ═══════════════ -->
        <div v-if="loadError" class="mb-4 rounded-lg border border-red-500/30 bg-red-500/[0.06] p-3 text-sm text-red-300 flex items-center gap-2">
            <Icon name="lucide:alert-circle" />
            {{ loadError }}
        </div>

        <!-- ═══════════════ LOADING SKELETONS (initial) ═══════════════ -->
        <div v-if="loading && !comments.length" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            <div v-for="i in 9" :key="i" class="comment-card animate-pulse">
                <div class="flex items-center gap-3 mb-3 pb-3 border-b border-white/[0.04]">
                    <div class="w-10 h-10 rounded-md bg-white/[0.06]" />
                    <div class="flex-1 space-y-2">
                        <div class="h-3 w-24 rounded bg-white/[0.06]" />
                        <div class="h-2 w-32 rounded bg-white/[0.04]" />
                    </div>
                </div>
                <div class="space-y-2 mb-4">
                    <div class="h-3 w-full rounded bg-white/[0.06]" />
                    <div class="h-3 w-5/6 rounded bg-white/[0.06]" />
                    <div class="h-3 w-3/4 rounded bg-white/[0.04]" />
                </div>
                <div class="pt-3 border-t border-white/[0.04] flex items-center justify-between">
                    <div class="h-6 w-24 rounded bg-white/[0.06]" />
                    <div class="h-6 w-12 rounded bg-white/[0.04]" />
                </div>
            </div>
        </div>

        <!-- ═══════════════ EMPTY STATE ═══════════════ -->
        <div v-else-if="!comments.length" class="rounded-xl border border-white/[0.08] bg-white/[0.02] py-20 px-6 text-center">
            <div class="inline-flex items-center justify-center w-16 h-16 rounded-full bg-white/[0.03] border border-white/[0.06] mb-4">
                <Icon name="lucide:message-circle-off" class="text-3xl text-gray-600" />
            </div>
            <h2 class="text-base font-semibold text-white">No comments found</h2>
            <p class="text-sm text-gray-500 mt-1.5 max-w-sm mx-auto">
                {{ hasActiveFilters ? 'Nothing matches the current filters. Try clearing them.' : 'Be the first to leave one on a killmail or battle.' }}
            </p>
            <button
                v-if="hasActiveFilters"
                class="mt-4 px-3 py-1.5 rounded-md text-xs text-blue-300 bg-blue-500/[0.1] border border-blue-500/20 hover:bg-blue-500/[0.2] cursor-pointer transition-colors"
                @click="clearAllFilters"
            >
                Clear filters
            </button>
        </div>

        <!-- ═══════════════ COMMENTS GRID ═══════════════ -->
        <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            <article
                v-for="c in comments"
                :key="c.id"
                class="comment-card group"
                :class="{ 'opacity-60': c.deleted_at }"
            >
                <!-- Type tag (top corner) + timestamp -->
                <div class="flex items-center justify-between gap-2 mb-2.5">
                    <span
                        v-if="targetLink(c).meta"
                        class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-fine font-semibold uppercase tracking-wider"
                        :class="[targetLink(c).meta!.chipBg, targetLink(c).meta!.chipText]"
                    >
                        <Icon :name="targetLink(c).meta!.icon" class="text-xs" />
                        {{ targetLink(c).meta!.label }}
                    </span>
                    <span v-else />
                    <Tooltip :text="formatEveDateTime(c.created_at, true)">
                        <span class="text-xs text-gray-600 whitespace-nowrap">{{ timeAgo(c.created_at) }}</span>
                    </Tooltip>
                </div>

                <!-- Author -->
                <div class="flex items-center gap-3 mb-3 pb-3 border-b border-white/[0.06]">
                    <NuxtLink :to="`/character/${c.character_id}`" class="flex-shrink-0">
                        <img
                            :src="`/images/characters/${c.character_id}/portrait?size=64`"
                            :alt="c.character_name"
                            class="w-10 h-10 rounded-md ring-1 ring-white/[0.06]"
                            loading="lazy"
                        >
                    </NuxtLink>
                    <div class="flex-1 min-w-0">
                        <NuxtLink
                            :to="`/character/${c.character_id}`"
                            class="block text-sm font-semibold text-white hover:text-blue-400 truncate transition-colors"
                        >
                            {{ c.character_name }}
                        </NuxtLink>
                        <div class="text-xs text-gray-500 truncate">
                            <NuxtLink :to="`/corporation/${c.corporation_id}`" class="hover:text-blue-400 hover:underline">
                                {{ c.corporation_name }}
                            </NuxtLink>
                            <template v-if="c.alliance_name">
                                <span class="text-gray-600"> / </span>
                                <NuxtLink :to="`/alliance/${c.alliance_id}`" class="hover:text-blue-400 hover:underline">
                                    {{ c.alliance_name }}
                                </NuxtLink>
                            </template>
                        </div>
                    </div>
                </div>

                <!-- Body -->
                <div class="flex-grow mb-4">
                    <div
                        class="comment-body text-sm text-gray-200 prose prose-invert max-w-none prose-sm line-clamp-6"
                        v-html="c.body_html"
                    />
                </div>

                <!-- Footer -->
                <div class="mt-auto pt-3 border-t border-white/[0.06] flex items-center gap-2">
                    <NuxtLink
                        :to="targetLink(c).href"
                        class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-md border text-xs font-medium transition-colors hover:brightness-125"
                        :class="targetLink(c).meta
                            ? [targetLink(c).meta!.chipBg, targetLink(c).meta!.chipBorder, targetLink(c).meta!.chipText]
                            : 'bg-white/[0.04] border-white/[0.08] text-gray-300'"
                    >
                        <Icon :name="targetLink(c).meta?.icon ?? 'lucide:link'" class="text-xs" />
                        {{ targetLink(c).label }}
                        <Icon name="lucide:arrow-up-right" class="text-xs opacity-60" />
                    </NuxtLink>

                    <span
                        v-if="c.reply_count > 0"
                        class="flex items-center gap-1 px-2 py-1 rounded text-xs text-gray-400 bg-white/[0.03]"
                        v-tooltip="`${c.reply_count} ${c.reply_count === 1 ? 'reply' : 'replies'}`"
                    >
                        <Icon name="lucide:corner-down-right" class="text-xs" />
                        {{ c.reply_count }}
                    </span>

                    <div class="flex-1" />

                    <button
                        v-if="isAuthenticated && user?.characterId === c.character_id && !c.deleted_at"
                        class="p-1.5 rounded-md text-gray-500 hover:text-red-400 hover:bg-red-500/[0.08] cursor-pointer transition-colors disabled:opacity-50"
                        :disabled="deletingId === c.id"
                        v-tooltip="'Delete'"
                        @click="deleteOwn(c.id)"
                    >
                        <Icon :name="deletingId === c.id ? 'lucide:loader-2' : 'lucide:trash-2'" class="text-xs" :class="{ 'animate-spin': deletingId === c.id }" />
                    </button>
                </div>
            </article>
        </div>

        <!-- ═══════════════ LOAD MORE ═══════════════ -->
        <div v-if="comments.length && hasMore" class="mt-6 flex justify-center">
            <button
                class="flex items-center gap-2 px-5 py-2.5 rounded-lg text-sm font-medium text-blue-300 bg-blue-500/[0.1] border border-blue-500/20 hover:bg-blue-500/[0.2] disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer transition-colors"
                :disabled="loading"
                @click="loadMore"
            >
                <Icon :name="loading ? 'lucide:loader-2' : 'lucide:chevron-down'" class="text-base" :class="{ 'animate-spin': loading }" />
                {{ loading ? 'Loading…' : 'Load more comments' }}
            </button>
        </div>
        <div v-else-if="comments.length && !hasMore" class="mt-6 text-center text-xs text-gray-600">
            — End of feed —
        </div>
    </div>
</template>

<style scoped>
.filter-input {
    display: block;
    width: 100%;
    height: 36px;
    padding: 0.5rem 0.75rem;
    font-size: 0.8125rem;
    color: white;
    background-color: rgba(0, 0, 0, 0.4);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 0.375rem;
    outline: none;
    transition: border-color 0.15s, box-shadow 0.15s;
}
.filter-input:focus {
    border-color: rgba(96, 165, 250, 0.5);
    box-shadow: 0 0 0 2px rgba(96, 165, 250, 0.15);
}
.filter-input::placeholder {
    color: rgb(75, 85, 99);
}

.filter-pill {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.4rem 0.75rem;
    border-radius: 0.5rem;
    font-size: 0.75rem;
    font-weight: 500;
    color: rgb(156, 163, 175);
    background-color: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.06);
    cursor: pointer;
    transition: color 0.15s, background-color 0.15s, border-color 0.15s;
}
.filter-pill:hover {
    color: white;
    background-color: rgba(255, 255, 255, 0.05);
    border-color: rgba(255, 255, 255, 0.1);
}
.filter-pill-active {
    color: white;
    background-color: rgba(96, 165, 250, 0.12);
    border-color: rgba(96, 165, 250, 0.35);
}
.filter-pill-active:hover {
    background-color: rgba(96, 165, 250, 0.18);
}

.comment-card {
    display: flex;
    flex-direction: column;
    padding: 1rem;
    border-radius: 0.75rem;
    border: 1px solid rgba(255, 255, 255, 0.08);
    background: linear-gradient(180deg, rgba(255, 255, 255, 0.03), rgba(255, 255, 255, 0.01));
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.2);
    transition: border-color 0.15s, transform 0.15s, box-shadow 0.15s;
}
.comment-card:hover {
    border-color: rgba(255, 255, 255, 0.14);
    transform: translateY(-1px);
    box-shadow: 0 8px 16px -3px rgba(0, 0, 0, 0.4);
}

.comment-body :deep(p) {
    margin: 0.25rem 0;
}
.comment-body :deep(p:first-child) {
    margin-top: 0;
}
.comment-body :deep(a) {
    color: rgb(96, 165, 250);
    text-decoration: underline;
}
.comment-body :deep(.embed) {
    margin: 0.5rem 0;
    border-radius: 0.5rem;
    overflow: hidden;
    max-height: 200px;
}
.comment-body :deep(.embed-image img),
.comment-body :deep(.embed-gif img) {
    max-width: 100%;
    max-height: 180px;
    border-radius: 0.5rem;
    object-fit: cover;
}
.comment-body :deep(.embed-youtube) {
    position: relative;
    padding-bottom: 56.25%;
    height: 0;
}
.comment-body :deep(.embed-youtube iframe) {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    border: 0;
}
.comment-body :deep(pre) {
    background: rgba(0, 0, 0, 0.4);
    padding: 0.5rem;
    border-radius: 0.25rem;
    overflow-x: auto;
    font-size: 0.7rem;
}
.comment-body :deep(code) {
    background: rgba(255, 255, 255, 0.08);
    padding: 0.1rem 0.3rem;
    border-radius: 0.2rem;
    font-size: 0.85em;
}
.comment-body :deep(blockquote) {
    border-left: 2px solid rgba(255, 255, 255, 0.15);
    padding-left: 0.6rem;
    color: rgb(156, 163, 175);
    margin: 0.4rem 0;
}
</style>
