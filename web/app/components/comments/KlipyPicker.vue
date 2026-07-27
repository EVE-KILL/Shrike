<script setup lang="ts">
import { useKlipyFavorites, type KlipyFavorite } from '~/composables/useKlipyFavorites'

interface KlipyMediaSize { url: string; width: number; height: number; size: number }
interface KlipyItem {
    id: number
    title: string
    slug: string
    file: {
        hd: { gif: KlipyMediaSize; webp: KlipyMediaSize; mp4: KlipyMediaSize }
        md: { gif: KlipyMediaSize; webp: KlipyMediaSize; mp4: KlipyMediaSize }
        sm: { gif: KlipyMediaSize; webp: KlipyMediaSize; mp4: KlipyMediaSize }
        xs: { gif: KlipyMediaSize; webp: KlipyMediaSize; mp4: KlipyMediaSize }
    }
    blur_preview?: string
}
interface KlipyResponse {
    items: KlipyItem[]
    page: number
    per_page: number
    has_next: boolean
}

const props = defineProps<{
    modelValue: boolean
    /** Called with the markdown link to insert into the comment body. */
    onInsert: (markdown: string) => void
}>()

const emit = defineEmits<{
    (e: 'update:modelValue', v: boolean): void
}>()

const { favorites, isFavorite, toggle } = useKlipyFavorites()

const tab = ref<'trending' | 'search' | 'favorites'>('trending')
const search = ref('')
const debouncedSearch = ref('')
let debounceTimer: ReturnType<typeof setTimeout> | null = null
watch(search, (v) => {
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(() => {
        debouncedSearch.value = v.trim()
        if (debouncedSearch.value.length >= 2) tab.value = 'search'
        else if (tab.value === 'search') tab.value = 'trending'
    }, 300)
})

const items = ref<KlipyItem[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const page = ref(1)
const hasMore = ref(false)

async function loadPage(reset = false) {
    if (loading.value) return
    loading.value = true
    error.value = null
    if (reset) {
        items.value = []
        page.value = 1
    }
    try {
        const url = tab.value === 'search'
            ? `/api/comments/klipy/search?q=${encodeURIComponent(debouncedSearch.value)}&page=${page.value}&per_page=24`
            : `/api/comments/klipy/trending?page=${page.value}&per_page=24`
        if (tab.value === 'search' && debouncedSearch.value.length < 2) return
        const res = await apiFetch<KlipyResponse>(url)
        items.value = reset ? res.items : [...items.value, ...res.items]
        hasMore.value = res.has_next
    } catch (err: any) {
        error.value = extractFetchError(err, 'Failed to load GIFs')
    } finally {
        loading.value = false
    }
}

// (Re)load whenever the tab or active search changes (and the modal is open)
watch([tab, debouncedSearch, () => props.modelValue], ([t, q, open]) => {
    if (!open) return
    if (t === 'favorites') {
        items.value = []
        return
    }
    if (t === 'search' && q.length < 2) {
        items.value = []
        return
    }
    page.value = 1
    loadPage(true)
}, { immediate: false })

// First-open: load trending
watch(() => props.modelValue, (open) => {
    if (open && items.value.length === 0 && tab.value === 'trending') {
        loadPage(true)
    }
})

function loadMore() {
    if (!hasMore.value || loading.value) return
    page.value++
    loadPage(false)
}

function pickItem(item: KlipyItem) {
    // Insert the md.mp4 URL — direct video files render with autoplay+loop+muted
    // through the existing markdown pipeline (no resolver needed).
    const mp4 = item.file.md.mp4.url
    const md = `[GIF](${mp4})`
    props.onInsert(md)
    useAnalytics().track('comment.media_attach', { source: 'klipy' })
    emit('update:modelValue', false)
}

function pickFavorite(fav: KlipyFavorite) {
    const md = `[GIF](${fav.insert_url})`
    props.onInsert(md)
    useAnalytics().track('comment.media_attach', { source: 'klipy_favorite' })
    emit('update:modelValue', false)
}

function toggleFavorite(item: KlipyItem) {
    toggle({
        id: item.id,
        title: item.title,
        preview_url: item.file.sm.webp.url,
        insert_url: item.file.md.mp4.url,
        width: item.file.md.mp4.width,
        height: item.file.md.mp4.height,
    })
}
</script>

<template>
    <Modal
        :model-value="modelValue"
        :max-width="'max-w-2xl'"
        @update:model-value="emit('update:modelValue', $event)"
    >
        <template #header>
            <h2 class="text-sm font-semibold text-white flex items-center gap-2">
                <Icon name="lucide:image-play" class="text-purple-400" />
                Add a GIF
                <span class="text-xs font-normal text-gray-600">powered by KLIPY</span>
            </h2>
        </template>

        <div class="space-y-3">
            <!-- Search bar -->
            <div class="relative">
                <Icon name="lucide:search" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 text-sm" />
                <input
                    v-model="search"
                    type="search"
                    placeholder="Search KLIPY"
                    class="w-full pl-9 pr-9 py-2 bg-black/40 border border-white/[0.08] rounded-md text-sm text-white placeholder-gray-600 focus:outline-none focus:border-blue-500/50"
                    autofocus
                >
                <button
                    v-if="search"
                    class="absolute right-2 top-1/2 -translate-y-1/2 p-1 rounded text-gray-500 hover:text-white hover:bg-white/[0.06] cursor-pointer"
                    @click="search = ''"
                >
                    <Icon name="lucide:x" class="text-xs" />
                </button>
            </div>

            <!-- Tabs -->
            <div class="flex items-center gap-1 border-b border-white/[0.06]">
                <button
                    type="button"
                    class="tab"
                    :class="tab === 'trending' ? 'tab-active' : ''"
                    @click="tab = 'trending'"
                >
                    <Icon name="lucide:flame" class="text-xs" />
                    Trending
                </button>
                <button
                    v-if="debouncedSearch.length >= 2"
                    type="button"
                    class="tab"
                    :class="tab === 'search' ? 'tab-active' : ''"
                    @click="tab = 'search'"
                >
                    <Icon name="lucide:search" class="text-xs" />
                    Results
                </button>
                <button
                    type="button"
                    class="tab"
                    :class="tab === 'favorites' ? 'tab-active' : ''"
                    @click="tab = 'favorites'"
                >
                    <Icon name="lucide:star" class="text-xs" />
                    Favorites
                    <span v-if="favorites.length > 0" class="text-xs text-gray-600">({{ favorites.length }})</span>
                </button>
            </div>

            <!-- Grid -->
            <div class="klipy-grid">
                <!-- Favorites -->
                <template v-if="tab === 'favorites'">
                    <div v-if="favorites.length === 0" class="col-span-full text-center py-8 text-sm text-gray-500 italic">
                        No favorites yet — click the star on any GIF to save it.
                    </div>
                    <div
                        v-for="fav in favorites"
                        :key="fav.id"
                        class="klipy-tile group"
                        role="button"
                        tabindex="0"
                        v-tooltip="fav.title"
                        @click="pickFavorite(fav)"
                        @keydown.enter.prevent="pickFavorite(fav)"
                        @keydown.space.prevent="pickFavorite(fav)"
                    >
                        <img :src="fav.preview_url" :alt="fav.title" loading="lazy" />
                        <button
                            type="button"
                            class="favorite-btn favorited"
                            v-tooltip="'Remove favorite'"
                            @click.stop="toggle(fav)"
                        >
                            <Icon name="lucide:star" class="text-xs" />
                        </button>
                    </div>
                </template>

                <!-- Trending / Search results -->
                <template v-else>
                    <div
                        v-for="item in items"
                        :key="item.id"
                        class="klipy-tile group"
                        role="button"
                        tabindex="0"
                        v-tooltip="item.title"
                        @click="pickItem(item)"
                        @keydown.enter.prevent="pickItem(item)"
                        @keydown.space.prevent="pickItem(item)"
                    >
                        <img
                            :src="item.file.sm.webp.url"
                            :alt="item.title"
                            loading="lazy"
                        />
                        <button
                            type="button"
                            class="favorite-btn"
                            :class="{ favorited: isFavorite(item.id) }"
                            v-tooltip="isFavorite(item.id) ? 'Remove favorite' : 'Add to favorites'"
                            @click.stop="toggleFavorite(item)"
                        >
                            <Icon name="lucide:star" class="text-xs" />
                        </button>
                    </div>
                </template>

                <div v-if="loading && items.length === 0" class="col-span-full text-center py-8 text-sm text-gray-500 italic">
                    <Icon name="lucide:loader-2" class="animate-spin inline-block mr-2" />
                    Loading…
                </div>
                <div v-if="error" class="col-span-full text-center py-6 text-sm text-red-400">
                    {{ error }}
                </div>
            </div>

            <!-- Load more -->
            <div v-if="tab !== 'favorites' && hasMore && items.length > 0" class="text-center">
                <button
                    type="button"
                    class="px-3 py-1.5 rounded-md text-xs text-blue-400 hover:text-blue-300 hover:bg-blue-500/[0.08] cursor-pointer disabled:opacity-40"
                    :disabled="loading"
                    @click="loadMore"
                >
                    {{ loading ? 'Loading…' : 'Load more' }}
                </button>
            </div>
        </div>
    </Modal>
</template>

<style scoped>
.tab {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    padding: 0.5rem 0.85rem;
    font-size: 0.75rem;
    color: rgb(107, 114, 128);
    cursor: pointer;
    border: none;
    background: transparent;
    border-bottom: 2px solid transparent;
    transition: color 0.15s, border-color 0.15s;
}
.tab:hover {
    color: rgb(229, 231, 235);
}
.tab-active {
    color: white;
    border-bottom-color: rgb(96, 165, 250);
}

.klipy-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 0.5rem;
    max-height: 50vh;
    overflow-y: auto;
    padding: 0.25rem;
}
@media (min-width: 640px) {
    .klipy-grid {
        grid-template-columns: repeat(4, 1fr);
    }
}

.klipy-tile {
    position: relative;
    aspect-ratio: 1;
    border-radius: 0.5rem;
    overflow: hidden;
    background: rgba(255, 255, 255, 0.04);
    border: 1px solid transparent;
    cursor: pointer;
    padding: 0;
    transition: border-color 0.15s, transform 0.15s;
}
.klipy-tile:hover {
    border-color: rgba(96, 165, 250, 0.6);
    transform: translateY(-1px);
}
.klipy-tile img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
}

.favorite-btn {
    position: absolute;
    top: 0.35rem;
    right: 0.35rem;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 1.5rem;
    height: 1.5rem;
    border-radius: 0.25rem;
    background: rgba(0, 0, 0, 0.55);
    color: rgb(156, 163, 175);
    border: none;
    cursor: pointer;
    opacity: 0;
    transition: opacity 0.15s, color 0.15s, background-color 0.15s;
}
.klipy-tile:hover .favorite-btn,
.favorite-btn.favorited {
    opacity: 1;
}
.favorite-btn:hover {
    color: rgb(250, 204, 21);
    background: rgba(0, 0, 0, 0.75);
}
.favorite-btn.favorited {
    color: rgb(250, 204, 21);
}
</style>
