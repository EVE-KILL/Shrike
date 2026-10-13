import { describe, expect, test } from 'vitest'
import { EMBED_CONFIG, commentEmbedPreview, insertCommentText, parseKlipyFavorites, addKlipyFavorite, type KlipyFavorite } from '../shared/utils/commentInput'

describe('shared Nuxt and Go comment controls', () => {
  test('selection formatting and literal embeds preserve surrounding Unicode text', () => {
    expect(insertCommentText('🚀 hello world', 3, 8, '**', '**')).toEqual({ body: '🚀 **hello** world', start: 5, end: 10 })
    expect(insertCommentText('before selection after', 7, 16, '[GIF](https://cdn.example/a.mp4)', '', false)).toEqual({ body: 'before [GIF](https://cdn.example/a.mp4) after', start: 39, end: 39 })
    expect(insertCommentText('abc', 1, 1, '[', '](https://)')).toEqual({ body: 'a[](https://)bc', start: 2, end: 2 })
  })
  test('provider URLs match the original composer and generate the correct thumbnails', () => {
    expect(EMBED_CONFIG.image.validate('https://example.com/ship.png?size=64')).toBeNull()
    expect(EMBED_CONFIG.image.validate('javascript:alert(1).png')).not.toBeNull()
    expect(EMBED_CONFIG.youtube.validate('https://youtu.be/dQw4w9WgXcQ')).toBeNull()
    expect(commentEmbedPreview('youtube', 'https://www.youtube.com/shorts/dQw4w9WgXcQ')).toBe('https://img.youtube.com/vi/dQw4w9WgXcQ/hqdefault.jpg')
    expect(commentEmbedPreview('imgur', 'https://imgur.com/abc123')).toBe('https://i.imgur.com/abc123.jpg')
    expect(commentEmbedPreview('image', 'https://example.com/ship.webp')).toBe('https://example.com/ship.webp')
    expect(commentEmbedPreview('youtube', 'https://example.com/watch?v=dQw4w9WgXcQ')).toBeNull()
    expect(EMBED_CONFIG.imgur.toMarkdown('https://imgur.com/abc123')).toBe('[Imgur](https://imgur.com/abc123)')
  })
  test('favorites keep the existing storage format, deduplicate, and cap at one hundred', () => {
    const favorite = { id: 7, title: 'EVE', preview_url: 'https://cdn.example/7.webp', insert_url: 'https://cdn.example/7.mp4', width: 200, height: 200 }
    const first = addKlipyFavorite([], favorite, 123)
    expect(parseKlipyFavorites(JSON.stringify(first))).toEqual([{ ...favorite, added_at: 123 }])
    expect(addKlipyFavorite(first, favorite, 999)).toEqual(first)
    expect(parseKlipyFavorites('broken json')).toEqual([])
    expect(parseKlipyFavorites('{}')).toEqual([])
    const existing = Array.from({ length: 100 }, (_, id) => ({ ...favorite, id: 100 + id, added_at: 1 })) satisfies KlipyFavorite[]
    const next = addKlipyFavorite(existing, favorite, 2)
    expect(next.length).toBe(100)
    expect(next[0]?.id).toBe(7)
    expect(next[99]?.id).toBe(198)
  })
})
