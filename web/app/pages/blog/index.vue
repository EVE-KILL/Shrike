<script setup lang="ts">
interface BlogListItem {
    id: number
    slug: string
    title: string
    excerpt: string | null
    cover_image_url: string | null
    tags: string[]
    author_id: number
    author_name: string
    author_corporation_id: number | null
    author_corporation_name: string | null
    author_alliance_id: number | null
    author_alliance_name: string | null
    published_at: string | null
}

const route = useRoute()
const activeTag = computed(() => {
    const t = Array.isArray(route.query.tag) ? route.query.tag[0] : route.query.tag
    return typeof t === 'string' && t.trim() ? t.trim().toLowerCase() : null
})

const cursor = ref<string | null>(null)
const posts = ref<BlogListItem[]>([])
const loading = ref(false)
const nextCursor = ref<string | null>(null)

async function loadMore() {
    if (loading.value) return
    loading.value = true
    try {
        const query: Record<string, string> = { limit: '20' }
        if (cursor.value) query.cursor = cursor.value
        if (activeTag.value) query.tag = activeTag.value
        const res = await apiFetch<{ posts: BlogListItem[]; nextCursor: string | null }>('/api/blog', { query })
        posts.value.push(...res.posts)
        nextCursor.value = res.nextCursor
        cursor.value = res.nextCursor
    } finally {
        loading.value = false
    }
}

// Initial fetch via useApiFetch for SSR-friendly first paint.
const { data: initial } = await useApiFetch<{ posts: BlogListItem[]; nextCursor: string | null }>('/api/blog', {
    query: computed(() => ({
        limit: 20,
        ...(activeTag.value ? { tag: activeTag.value } : {}),
    })),
    watch: [activeTag],
})
watchEffect(() => {
    if (initial.value) {
        posts.value = initial.value.posts
        nextCursor.value = initial.value.nextCursor
        cursor.value = initial.value.nextCursor
    }
})

useHead(() => ({
    title: activeTag.value ? `#${activeTag.value} — EVE-KILL Blog` : 'Blog — EVE-KILL',
    meta: [
        { name: 'description', content: activeTag.value
            ? `Posts tagged #${activeTag.value} on EVE-KILL.`
            : 'News, updates, and long-form posts from EVE-KILL.' },
    ],
}))

function fmtDate(iso: string | null): string {
    if (!iso) return ''
    return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}
</script>

<template>
    <div class="max-w-4xl mx-auto px-4 py-8 space-y-6">
        <header class="space-y-2">
            <h1 class="text-3xl font-bold text-white">Blog</h1>
            <p class="text-sm text-gray-400">News, updates, and long-form posts.</p>
            <div v-if="activeTag" class="flex items-center gap-2 pt-1">
                <span class="text-xs text-gray-500">Filtering by tag:</span>
                <span class="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium bg-blue-500/10 text-blue-300 border border-blue-500/20">
                    #{{ activeTag }}
                </span>
                <NuxtLink to="/blog" class="text-xs text-gray-500 hover:text-gray-300 underline">Clear</NuxtLink>
            </div>
        </header>

        <div v-if="!posts.length" class="text-center py-16 text-gray-500 text-sm">
            No posts yet.
        </div>

        <div v-else class="space-y-4">
            <NuxtLink
                v-for="p in posts"
                :key="p.id"
                :to="`/blog/${p.slug}`"
                class="block rounded-xl bg-white/[0.03] border border-white/[0.08] hover:border-white/[0.16] transition-colors overflow-hidden"
            >
                <div class="flex flex-col sm:flex-row">
                    <div
                        v-if="p.cover_image_url"
                        class="sm:w-48 sm:flex-shrink-0 aspect-video sm:aspect-auto bg-cover bg-center"
                        :style="{ backgroundImage: `url(${p.cover_image_url})` }"
                    />
                    <div class="p-5 flex-1 min-w-0">
                        <div class="flex items-center gap-2 text-xs text-gray-500 mb-2 flex-wrap">
                            <time v-if="p.published_at" :datetime="p.published_at" class="font-medium text-gray-400">
                                {{ fmtDate(p.published_at) }}
                            </time>
                            <template v-if="p.tags.length">
                                <span class="text-gray-700">·</span>
                                <span
                                    v-for="t in p.tags.slice(0, 3)" :key="t"
                                    class="inline-flex items-center px-1.5 py-0.5 rounded-full text-[10px] font-medium bg-blue-500/10 text-blue-300 border border-blue-500/20"
                                >
                                    #{{ t }}
                                </span>
                                <span v-if="p.tags.length > 3" class="text-[10px] text-gray-600">
                                    +{{ p.tags.length - 3 }}
                                </span>
                            </template>
                        </div>
                        <h2 class="text-lg font-semibold text-white mb-1">{{ p.title }}</h2>
                        <p v-if="p.excerpt" class="text-sm text-gray-400 line-clamp-3">{{ p.excerpt }}</p>
                    </div>
                </div>
            </NuxtLink>
        </div>

        <div v-if="nextCursor" class="flex justify-center pt-4">
            <button
                :disabled="loading"
                class="px-5 py-2 rounded-lg bg-white/[0.04] border border-white/[0.08] text-sm text-gray-300 hover:bg-white/[0.08] transition-colors disabled:opacity-50"
                @click="loadMore"
            >
                {{ loading ? 'Loading...' : 'Load more' }}
            </button>
        </div>
    </div>
</template>
