import { expect, test } from 'bun:test'
import { siteBackgroundCSS } from '../shared/utils/siteBackground'

test('built-in backgrounds use format negotiation while the viewer retains the original', () => {
    expect(siteBackgroundCSS('/backgrounds/bg2.webp')).toBe('image-set(url("/backgrounds/optimized-v1/bg2.avif") type("image/avif"), url("/backgrounds/optimized-v1/bg2.webp") type("image/webp"))')
    expect(siteBackgroundCSS('/backgrounds/bg2.webp', true)).toBe('url("/backgrounds/bg2.webp")')
})

test('custom and external backgrounds retain their URLs', () => {
    for (const path of ['/domain-assets/background.webp', 'https://example.com/backgrounds/bg2.webp', '/backgrounds/bg9.webp']) {
        expect(siteBackgroundCSS(path)).toBe(`url(${JSON.stringify(path)})`)
    }
})
