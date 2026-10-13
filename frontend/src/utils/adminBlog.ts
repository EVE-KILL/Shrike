import { isoToEveInput, eveInputToIso } from './eveTime'

export interface AdminBlogPost {
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

// These are the original Blog.vue rules, shared by both browser renderers.
export function parseTagInput(input: string): string[] {
    return input.split(/[,\s]+/)
        .map(t => t.trim().toLowerCase().replace(/[^a-z0-9-]/g, ''))
        .filter(Boolean).slice(0, 10)
}

export function blogSlugify(s: string): string {
    return s.toLowerCase().normalize('NFKD').replace(/[\u0300-\u036f]/g, '')
        .replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 80)
}

export function blogForm(post?: AdminBlogPost) {
    return {
        title: post?.title ?? '', slug: post?.slug ?? '', excerpt: post?.excerpt ?? '',
        body_md: post?.body_md ?? '', cover_image_url: post?.cover_image_url ?? '',
        tags: (post?.tags ?? []).join(', '), status: post?.status ?? 0,
        published_at: isoToEveInput(post?.published_at),
    }
}

export function blogPayload(form: ReturnType<typeof blogForm>, editing: boolean) {
    const title = form.title.trim()
    const payload: Record<string, unknown> = {
        title, slug: form.slug.trim() || blogSlugify(title), excerpt: form.excerpt.trim() || null,
        body_md: form.body_md, cover_image_url: form.cover_image_url.trim() || null,
        tags: parseTagInput(form.tags), status: form.status,
    }
    if (form.published_at) payload.published_at = eveInputToIso(form.published_at)
    else if (editing) payload.published_at = null
    return payload
}

export const blogStatusLabels: Record<number, { label: string; cls: string }> = {
    0: { label: 'Draft', cls: 'bg-gray-500/20 text-gray-400 border-gray-500/30' },
    1: { label: 'Published', cls: 'bg-green-500/20 text-green-300 border-green-500/30' },
    2: { label: 'Archived', cls: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30' },
}
