<script setup lang="ts">
// ── Blog section state ──────────────────────────────────────────────────────
interface AdminBlogPost {
    id: number
    slug: string
    title: string
    excerpt: string | null
    body_md: string
    body_html: string
    cover_image_url: string | null
    status: number
    author_id: number
    author_name: string
    author_corporation_id: number | null
    author_corporation_name: string | null
    author_alliance_id: number | null
    author_alliance_name: string | null
    tags: string[]
    published_at: string | null
    created_at: string
    updated_at: string
}

// One of: all | draft | published | archived — goes straight to ?status=
const blogStatusFilter = ref('all')
const { data: blogData, refresh: refreshBlog } = useApiFetch<{ posts: AdminBlogPost[] }>('/api/admin/blog', {
    query: { status: blogStatusFilter, limit: 100 },
    immediate: true,
    lazy: true,
    watch: [blogStatusFilter],
})

const blogEditOpen = ref(false)
const blogEditId = ref<number | null>(null)
const blogForm = ref({
    title: '',
    slug: '',
    excerpt: '',
    body_md: '',
    cover_image_url: '',
    tags: '',
    status: 0 as number,
    published_at: '',
})

function parseTagInput(input: string): string[] {
    return input.split(/[,\s]+/)
        .map(t => t.trim().toLowerCase().replace(/[^a-z0-9-]/g, ''))
        .filter(Boolean)
        .slice(0, 10)
}

function blogSlugify(s: string): string {
    return s.toLowerCase().normalize('NFKD').replace(/[\u0300-\u036f]/g, '')
        .replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 80)
}

function blogNewForm() {
    blogEditId.value = null
    blogFormError.value = ''
    blogForm.value = {
        title: '',
        slug: '',
        excerpt: '',
        body_md: '',
        cover_image_url: '',
        tags: '',
        status: 0,
        published_at: '',
    }
    blogEditOpen.value = true
}

function blogEditForm(p: AdminBlogPost) {
    blogEditId.value = p.id
    blogFormError.value = ''
    blogForm.value = {
        title: p.title,
        slug: p.slug,
        excerpt: p.excerpt || '',
        body_md: p.body_md,
        cover_image_url: p.cover_image_url || '',
        tags: (p.tags || []).join(', '),
        status: p.status,
        published_at: isoToEveInput(p.published_at),
    }
    blogEditOpen.value = true
}

const toast = useToast()
const blogSaving = ref(false)
const blogFormError = ref('')

async function blogSave() {
    const title = blogForm.value.title.trim()
    if (!title) { blogFormError.value = 'Title is required'; return }
    blogFormError.value = ''

    // Auto-fill slug from title when empty on create.
    const slug = (blogForm.value.slug.trim() || blogSlugify(title))

    const payload: Record<string, unknown> = {
        title,
        slug,
        excerpt: blogForm.value.excerpt.trim() || null,
        body_md: blogForm.value.body_md,
        cover_image_url: blogForm.value.cover_image_url.trim() || null,
        tags: parseTagInput(blogForm.value.tags),
        status: blogForm.value.status,
    }
    if (blogForm.value.published_at) {
        payload.published_at = eveInputToIso(blogForm.value.published_at)
    } else if (blogEditId.value) {
        payload.published_at = null
    }

    blogSaving.value = true
    try {
        if (blogEditId.value) {
            await apiFetch(`/api/admin/blog/${blogEditId.value}`, { method: 'PATCH', body: payload })
        } else {
            await apiFetch('/api/admin/blog', { method: 'POST', body: payload })
        }
        blogEditOpen.value = false
        await refreshBlog()
    } catch (err: any) {
        // Inline rather than a toast: the modal stays open on failure, so the
        // message belongs next to the form the user still has to fix.
        blogFormError.value = extractFetchError(err, 'Failed to save')
    } finally {
        blogSaving.value = false
    }
}

// Deleting a post is permanent, so the button arms on the first click and only
// fires on the second — same two-step used elsewhere in admin.
const { pendingId: confirmDeleteId, confirm: confirmDelete } = useConfirmTwice()

async function blogDelete(id: number) {
    if (!confirmDelete(id)) return
    try {
        await apiFetch(`/api/admin/blog/${id}`, { method: 'DELETE' })
        await refreshBlog()
    } catch (err: any) {
        toast.error(extractFetchError(err, 'Failed to delete'))
    }
}


const blogStatusLabels: Record<number, { label: string; cls: string }> = {
    0: { label: 'Draft', cls: 'bg-gray-500/20 text-gray-400 border-gray-500/30' },
    1: { label: 'Published', cls: 'bg-green-500/20 text-green-300 border-green-500/30' },
    2: { label: 'Archived', cls: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30' },
}

// The parent keeps this component alive (<KeepAlive>) across tab switches.
// Matching the old tab-switch watcher: on re-activation, refetch only if no
// data ever loaded (e.g. the first fetch errored).
let activatedOnce = false
onActivated(() => {
    if (!activatedOnce) { activatedOnce = true; return }
    if (!blogData.value) refreshBlog()
})


// Helpers
</script>

<template>
    <!-- ============================================================
         Blog
         ============================================================ -->
    <div class="space-y-4">
        <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
                <SelectMenu
                    v-model="blogStatusFilter"
                    :options="[
                        { value: 'all', label: 'All posts' },
                        { value: 'draft', label: 'Draft' },
                        { value: 'published', label: 'Published' },
                        { value: 'archived', label: 'Archived' },
                    ]"
                />
            </div>
            <button
                class="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-xs font-medium transition-colors cursor-pointer"
                @click="blogNewForm"
            >
                <Icon name="lucide:plus" class="text-sm" />
                New Post
            </button>
        </div>

        <div v-if="!blogData?.posts?.length" class="text-center py-12 text-gray-500 text-sm">
            No blog posts found.
        </div>
        <div v-else class="space-y-2">
            <div
                v-for="p in blogData.posts"
                :key="p.id"
                class="glass-panel p-4"
            >
                <div class="flex items-start justify-between gap-4">
                    <div class="flex-1 min-w-0">
                        <div class="flex items-center gap-2 mb-1">
                            <span
                                class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider border"
                                :class="blogStatusLabels[p.status]?.cls"
                            >
                                {{ blogStatusLabels[p.status]?.label || 'Unknown' }}
                            </span>
                            <span class="text-[10px] text-gray-600">#{{ p.id }}</span>
                            <span class="text-[10px] text-gray-500 font-mono">/blog/{{ p.slug }}</span>
                        </div>
                        <div class="text-sm font-medium text-white">{{ p.title }}</div>
                        <div v-if="p.excerpt" class="text-xs text-gray-500 mt-1 line-clamp-2">{{ p.excerpt }}</div>
                        <div v-if="p.tags?.length" class="flex items-center gap-1 mt-2 flex-wrap">
                            <span
                                v-for="t in p.tags" :key="t"
                                class="inline-flex items-center px-1.5 py-0.5 rounded-full text-[10px] font-medium bg-blue-500/10 text-blue-300 border border-blue-500/20"
                            >
                                #{{ t }}
                            </span>
                        </div>
                        <div class="flex items-center gap-3 mt-2 text-[10px] text-gray-600">
                            <span>By {{ p.author_name }}</span>
                            <span v-if="p.published_at">Published: {{ fmtDate(p.published_at) }}</span>
                            <span>Updated: {{ fmtDate(p.updated_at) }}</span>
                        </div>
                    </div>
                    <div class="flex items-center gap-1 flex-shrink-0">
                        <NuxtLink
                            :to="`/blog/preview/${p.slug}`"
                            target="_blank"
                            class="flex items-center justify-center w-7 h-7 rounded-md text-gray-500 hover:text-yellow-400 hover:bg-yellow-500/[0.08] transition-colors cursor-pointer"
                            v-tooltip="'Preview (admin-only)'"
                        >
                            <Icon name="lucide:eye" class="text-sm" />
                        </NuxtLink>
                        <NuxtLink
                            v-if="p.status === 1"
                            :to="`/blog/${p.slug}`"
                            target="_blank"
                            class="flex items-center justify-center w-7 h-7 rounded-md text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.08] transition-colors cursor-pointer"
                            v-tooltip="'View live'"
                        >
                            <Icon name="lucide:external-link" class="text-sm" />
                        </NuxtLink>
                        <button
                            class="flex items-center justify-center w-7 h-7 rounded-md text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.08] transition-colors cursor-pointer"
                            v-tooltip="'Edit'"
                            @click="blogEditForm(p)"
                        >
                            <Icon name="lucide:pencil" class="text-sm" />
                        </button>
                        <button
                            class="flex items-center justify-center h-7 rounded-md transition-colors cursor-pointer"
                            :class="confirmDeleteId === p.id
                                ? 'px-2 gap-1 text-red-300 bg-red-500/15 ring-1 ring-red-500/40'
                                : 'w-7 text-gray-500 hover:text-red-400 hover:bg-red-500/[0.08]'"
                            v-tooltip="confirmDeleteId === p.id ? 'Click again to delete permanently' : 'Delete'"
                            @click="blogDelete(p.id)"
                        >
                            <Icon name="lucide:trash-2" class="text-sm" />
                            <span v-if="confirmDeleteId === p.id" class="text-fine font-medium">Confirm</span>
                        </button>
                    </div>
                </div>
            </div>
        </div>

        <Modal v-model="blogEditOpen" :title="blogEditId ? 'Edit Post' : 'New Post'" max-width="max-w-3xl">
            <div class="space-y-4">
                <div>
                    <label class="block text-xs font-medium text-gray-400 mb-1">Title *</label>
                    <input
                        v-model="blogForm.title"
                        type="text"
                        class="w-full bg-white/[0.04] border border-white/[0.08] rounded-lg text-sm text-white px-3 py-2 focus:outline-none focus:border-blue-500/50"
                        placeholder="Post title"
                    >
                </div>

                <div>
                    <label class="block text-xs font-medium text-gray-400 mb-1">
                        Slug <span class="text-gray-600">(optional — auto-generated from title)</span>
                    </label>
                    <input
                        v-model="blogForm.slug"
                        type="text"
                        class="w-full bg-white/[0.04] border border-white/[0.08] rounded-lg text-sm text-white px-3 py-2 focus:outline-none focus:border-blue-500/50 font-mono"
                        placeholder="my-post-slug"
                    >
                </div>

                <div>
                    <label class="block text-xs font-medium text-gray-400 mb-1">Excerpt (optional, 500 chars max)</label>
                    <textarea
                        v-model="blogForm.excerpt"
                        rows="2"
                        maxlength="500"
                        class="w-full bg-white/[0.04] border border-white/[0.08] rounded-lg text-sm text-white px-3 py-2 focus:outline-none focus:border-blue-500/50 resize-y"
                        placeholder="Short summary shown on post cards"
                    />
                </div>

                <div>
                    <label class="block text-xs font-medium text-gray-400 mb-1">Cover image URL (optional)</label>
                    <input
                        v-model="blogForm.cover_image_url"
                        type="text"
                        class="w-full bg-white/[0.04] border border-white/[0.08] rounded-lg text-sm text-white px-3 py-2 focus:outline-none focus:border-blue-500/50"
                        placeholder="https://..."
                    >
                </div>

                <div>
                    <label class="block text-xs font-medium text-gray-400 mb-1">
                        Tags <span class="text-gray-600">(comma-separated, lowercase, max 10)</span>
                    </label>
                    <input
                        v-model="blogForm.tags"
                        type="text"
                        class="w-full bg-white/[0.04] border border-white/[0.08] rounded-lg text-sm text-white px-3 py-2 focus:outline-none focus:border-blue-500/50"
                        placeholder="patch-notes, meta, pvp"
                    >
                    <div v-if="blogForm.tags.trim()" class="flex items-center gap-1 mt-2 flex-wrap">
                        <span
                            v-for="t in parseTagInput(blogForm.tags)" :key="t"
                            class="inline-flex items-center px-1.5 py-0.5 rounded-full text-[10px] font-medium bg-blue-500/10 text-blue-300 border border-blue-500/20"
                        >
                            #{{ t }}
                        </span>
                    </div>
                </div>

                <div>
                    <label class="block text-xs font-medium text-gray-400 mb-1">Body (markdown)</label>
                    <textarea
                        v-model="blogForm.body_md"
                        rows="16"
                        class="w-full bg-white/[0.04] border border-white/[0.08] rounded-lg text-sm text-white px-3 py-2 focus:outline-none focus:border-blue-500/50 resize-y font-mono"
                        placeholder="# Heading&#10;&#10;Post content in markdown..."
                    />
                </div>

                <div class="grid grid-cols-2 gap-3">
                    <div>
                        <label class="block text-xs font-medium text-gray-400 mb-1">Status</label>
                        <div class="flex gap-2">
                            <button
                                v-for="s in [{ v: 0, l: 'Draft' }, { v: 1, l: 'Published' }, { v: 2, l: 'Archived' }]"
                                :key="s.v"
                                class="flex-1 px-3 py-2 rounded-lg text-xs font-medium border transition-colors cursor-pointer"
                                :class="blogForm.status === s.v
                                    ? 'bg-blue-500/20 border-blue-500/40 text-blue-300'
                                    : 'bg-white/[0.02] border-white/[0.08] text-gray-500 hover:text-gray-300'"
                                @click="blogForm.status = s.v"
                            >
                                {{ s.l }}
                            </button>
                        </div>
                    </div>
                    <div>
                        <label class="block text-xs font-medium text-gray-400 mb-1">
                            Publish date <span class="text-gray-600">(optional, EVE time)</span>
                        </label>
                        <DateTimePicker v-model="blogForm.published_at" placeholder="Publish immediately" />
                    </div>
                </div>
            </div>

            <template #footer>
                <div class="flex items-center justify-end gap-3">
                    <p v-if="blogFormError" class="flex-1 text-xs text-red-400 flex items-center gap-1.5">
                        <Icon name="lucide:alert-circle" class="text-sm flex-shrink-0" />
                        {{ blogFormError }}
                    </p>
                    <button
                        class="px-4 py-2 rounded-lg text-sm text-gray-400 hover:text-white transition-colors cursor-pointer"
                        @click="blogEditOpen = false"
                    >
                        Cancel
                    </button>
                    <button
                        class="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-sm font-medium transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                        :disabled="blogSaving"
                        @click="blogSave"
                    >
                        <Icon v-if="blogSaving" name="lucide:loader-2" class="text-sm animate-spin mr-1 inline" />
                        {{ blogEditId ? 'Update' : 'Create' }}
                    </button>
                </div>
            </template>
        </Modal>
    </div>
</template>
