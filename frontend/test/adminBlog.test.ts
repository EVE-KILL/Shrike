import { expect, test } from 'vitest'
import { blogForm, blogPayload, blogSlugify, parseTagInput, type AdminBlogPost } from '../src/utils/adminBlog'

test('blog editor shares slug/tag rules and preserves Markdown as text', () => {
    expect(blogSlugify('  Héllø, New Éden! ')).toBe('hell-new-eden')
    expect(blogSlugify('x'.repeat(90))).toHaveLength(80)
    expect(parseTagInput('PvP, PATCH-NOTES <script> a_b pvp')).toEqual(['pvp', 'patch-notes', 'script', 'ab', 'pvp'])
    expect(parseTagInput(Array.from({length: 20}, (_, i) => `tag-${i}`).join(' '))).toHaveLength(10)
    const form = { ...blogForm(), title: '  Héllo  ', tags: 'PvP', body_md: '<script>plain draft</script>\n**Markdown**' }
    expect(blogPayload(form, false)).toEqual({title: 'Héllo', slug: 'hello', excerpt: null, body_md: form.body_md, cover_image_url: null, tags: ['pvp'], status: 0})
})

test('blog dates use EVE UTC and distinguish create omission from edit clearing', () => {
    const form = blogForm({title: 'Post', slug: 'post', tags: ['pvp', 'news'], status: 1, published_at: '2026-10-05T13:45:37Z'} as AdminBlogPost)
    expect(form.tags).toBe('pvp, news')
    expect(form.published_at).toBe('2026-10-05T13:45')
    expect(blogPayload(form, true).published_at).toBe('2026-10-05T13:45:00.000Z')
    form.published_at = ''
    expect(blogPayload(form, true).published_at).toBeNull()
    expect(Object.hasOwn(blogPayload(form, false), 'published_at')).toBe(false)
    form.published_at = 'invalid'
    expect(() => blogPayload(form, true)).toThrow()
})
