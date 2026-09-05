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

const { data, error } = await useApiFetch<{ post: BlogPost }>(() => `/api/blog/${slug.value}`)
if (error.value || !data.value?.post) {
    throw createError({ statusCode: 404, statusMessage: 'Post not found', fatal: true })
}
const post = data.value.post

const COMMENT_TARGET_BLOG_POST = 9

useHead({
    title: `${post.title} — EVE-KILL Blog`,
    meta: [
        { name: 'description', content: post.excerpt || post.title },
        { property: 'og:title', content: post.title },
        { property: 'og:description', content: post.excerpt || post.title },
        { property: 'og:type', content: 'article' },
        ...(post.cover_image_url ? [{ property: 'og:image', content: post.cover_image_url }] : []),
    ],
})

function fmtDateLong(iso: string | null): string {
    if (!iso) return ''
    return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'long', day: 'numeric' })
}
</script>

<template>
    <article class="w-full space-y-6">
        <nav class="text-xs text-gray-500">
            <NuxtLink to="/blog" class="hover:text-gray-300">← Blog</NuxtLink>
        </nav>

        <PageHeader :title="post.title" :description="post.excerpt || undefined" eyebrow="From EVE-KILL" icon="lucide:newspaper">
            <template #meta>
            <div class="flex items-center gap-2 text-xs text-gray-500 flex-wrap">
                <time v-if="post.published_at" :datetime="post.published_at" class="font-medium text-gray-400">
                    {{ fmtDateLong(post.published_at) }}
                </time>
                <template v-if="post.tags.length">
                    <span class="text-gray-700">·</span>
                    <div class="flex items-center gap-1.5 flex-wrap">
                        <NuxtLink
                            v-for="t in post.tags" :key="t"
                            :to="`/blog?tag=${t}`"
                            class="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium bg-blue-500/10 text-blue-300 border border-blue-500/20 hover:bg-blue-500/20 transition-colors"
                        >
                            #{{ t }}
                        </NuxtLink>
                    </div>
                </template>
                <template v-if="post.status === 2">
                    <span class="text-gray-700">·</span>
                    <span class="inline-flex items-center px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider border bg-amber-500/20 text-amber-300 border-amber-500/30">
                        Archived
                    </span>
                </template>
            </div>
            </template>
        </PageHeader>

        <div class="mx-auto max-w-3xl space-y-6">
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
            <CommentsCommentList
                :target-type="COMMENT_TARGET_BLOG_POST"
                :target-id="post.id"
                :show-header="false"
            />
        </section>
        </div>
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
