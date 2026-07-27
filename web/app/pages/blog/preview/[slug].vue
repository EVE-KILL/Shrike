<script setup lang="ts">
const route = useRoute()
const slug = computed(() => String(route.params.slug))

interface BlogPost {
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

// Gate: await full admin status from /api/auth/me (cookie hint lacks isAdmin).
const { data: authCheck } = await useApiFetch('/api/auth/me')
if (!authCheck.value?.user?.isAdmin) {
    throw createError({ statusCode: 404, statusMessage: 'Not found', fatal: true })
}

const { data, error } = await useApiFetch<{ post: BlogPost }>(() => `/api/admin/blog/preview/${slug.value}`)
if (error.value || !data.value?.post) {
    throw createError({ statusCode: 404, statusMessage: 'Post not found', fatal: true })
}
const post = data.value.post

const COMMENT_TARGET_BLOG_POST = 9

const statusMeta: Record<number, { label: string; cls: string }> = {
    0: { label: 'Draft', cls: 'bg-gray-500/20 text-gray-300 border-gray-500/30' },
    1: { label: 'Published', cls: 'bg-green-500/20 text-green-300 border-green-500/30' },
    2: { label: 'Archived', cls: 'bg-amber-500/20 text-amber-300 border-amber-500/30' },
}

// Preview pages must never be indexed or cached by the browser.
useHead({
    title: `[Preview] ${post.title} — EVE-KILL Blog`,
    meta: [
        { name: 'robots', content: 'noindex, nofollow' },
    ],
})

function fmtDateLong(iso: string | null): string {
    if (!iso) return ''
    return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'long', day: 'numeric' })
}
</script>

<template>
    <article class="max-w-3xl mx-auto px-4 py-8 space-y-6">
        <!-- Preview banner -->
        <div class="rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-3 flex items-center justify-between gap-4">
            <div class="flex items-center gap-3 text-sm">
                <Icon name="lucide:eye" class="text-amber-300 text-base" />
                <span class="text-amber-200 font-medium">Preview mode</span>
                <span
                    class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider border"
                    :class="statusMeta[post.status]?.cls"
                >
                    {{ statusMeta[post.status]?.label || 'Unknown' }}
                </span>
                <span class="text-xs text-amber-300/70">This view is admin-only and not cached.</span>
            </div>
            <NuxtLink
                to="/admin/blog"
                class="text-xs text-amber-200 hover:text-white underline whitespace-nowrap"
            >
                Edit in admin
            </NuxtLink>
        </div>

        <nav class="text-xs text-gray-500">
            <NuxtLink to="/blog" class="hover:text-gray-300">← Blog</NuxtLink>
        </nav>

        <header class="space-y-4">
            <div class="flex items-center gap-2 text-xs text-gray-500 flex-wrap">
                <time v-if="post.published_at" :datetime="post.published_at" class="font-medium text-gray-400">
                    {{ fmtDateLong(post.published_at) }}
                </time>
                <span v-else class="italic text-gray-500">unpublished</span>
                <template v-if="post.tags.length">
                    <span class="text-gray-700">·</span>
                    <div class="flex items-center gap-1.5 flex-wrap">
                        <span
                            v-for="t in post.tags" :key="t"
                            class="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium bg-blue-500/10 text-blue-300 border border-blue-500/20"
                        >
                            #{{ t }}
                        </span>
                    </div>
                </template>
            </div>
            <h1 class="text-3xl sm:text-4xl font-bold text-white leading-tight">{{ post.title }}</h1>
            <p v-if="post.excerpt" class="text-lg text-gray-400 leading-relaxed">{{ post.excerpt }}</p>
        </header>

        <img
            v-if="post.cover_image_url"
            :src="post.cover_image_url"
            :alt="post.title"
            class="w-full rounded-xl border border-white/[0.08]"
            loading="eager"
        >

        <div class="blog-body" v-html="post.body_html" />

        <BlogAuthorCard
            :author-id="post.author_id"
            :author-name="post.author_name"
            :corporation-id="post.author_corporation_id"
            :corporation-name="post.author_corporation_name"
            :alliance-id="post.author_alliance_id"
            :alliance-name="post.author_alliance_name"
            :published-at="post.published_at"
        />

        <section class="pt-8 border-t border-white/[0.08]">
            <h2 class="text-lg font-semibold text-white mb-4">Comments</h2>
            <p v-if="post.status !== 1" class="text-xs text-gray-500 italic mb-3">
                Comments are mounted but will appear under this post's id only after publish.
            </p>
            <CommentsCommentList
                :target-type="COMMENT_TARGET_BLOG_POST"
                :target-id="post.id"
                :show-header="false"
            />
        </section>
    </article>
</template>

<style scoped>
.blog-body {
    color: rgb(229 231 235);
    line-height: 1.7;
    font-size: 1rem;
}
.blog-body :deep(h1), .blog-body :deep(h2), .blog-body :deep(h3),
.blog-body :deep(h4), .blog-body :deep(h5), .blog-body :deep(h6) {
    color: white;
    font-weight: 600;
    margin-top: 2rem;
    margin-bottom: 0.75rem;
    line-height: 1.3;
}
.blog-body :deep(h1) { font-size: 1.875rem; }
.blog-body :deep(h2) { font-size: 1.5rem; }
.blog-body :deep(h3) { font-size: 1.25rem; }
.blog-body :deep(p) { margin: 1rem 0; }
.blog-body :deep(a) { color: rgb(96 165 250); text-decoration: underline; }
.blog-body :deep(a:hover) { color: rgb(147 197 253); }
.blog-body :deep(code) {
    background: rgba(255,255,255,0.06);
    padding: 0.125rem 0.375rem;
    border-radius: 0.25rem;
    font-size: 0.875em;
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}
.blog-body :deep(pre) {
    background: rgba(0,0,0,0.4);
    border: 1px solid rgba(255,255,255,0.08);
    border-radius: 0.5rem;
    padding: 1rem;
    overflow-x: auto;
    margin: 1rem 0;
}
.blog-body :deep(pre code) { background: transparent; padding: 0; }
.blog-body :deep(blockquote) {
    border-left: 3px solid rgba(255,255,255,0.16);
    padding-left: 1rem;
    color: rgb(156 163 175);
    margin: 1rem 0;
}
.blog-body :deep(ul), .blog-body :deep(ol) { margin: 1rem 0; padding-left: 1.5rem; }
.blog-body :deep(li) { margin: 0.25rem 0; }
.blog-body :deep(img) {
    max-width: 100%;
    height: auto;
    border-radius: 0.5rem;
    margin: 1rem 0;
}
.blog-body :deep(hr) {
    border: 0;
    border-top: 1px solid rgba(255,255,255,0.08);
    margin: 2rem 0;
}
.blog-body :deep(table) {
    width: 100%;
    border-collapse: collapse;
    margin: 1rem 0;
    font-size: 0.875rem;
}
.blog-body :deep(th), .blog-body :deep(td) {
    border: 1px solid rgba(255,255,255,0.08);
    padding: 0.5rem 0.75rem;
    text-align: left;
}
.blog-body :deep(th) { background: rgba(255,255,255,0.04); font-weight: 600; }
</style>
