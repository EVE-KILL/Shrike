import { afterAll, expect, test } from 'bun:test'
import { loadSitemapSource } from '../server/utils/sitemapSource'
import { canServeSitemap, isCanonicalSeoHost, isSitemapPath } from '../server/utils/seoHost'

const socket = `/tmp/shrike-seo-${process.pid}.sock`
const requests: string[] = []
const server = Bun.serve({
    unix: socket,
    fetch(request) {
        const path = new URL(request.url).pathname
        requests.push(path)
        if (path === '/api/__sitemap__/ships' && request.headers.get('host') === 'eve-kill.com') {
            return Response.json([{ loc: '/item/587', priority: 0.6 }])
        }
        return new Response('Unavailable', { status: 503 })
    },
})
afterAll(() => server.stop(true))

test('sitemap source uses Go socket and preserves the URL array', async () => {
    expect(await loadSitemapSource('ships', { apiSocket: socket, apiOrigin: 'http://127.0.0.1:1' }))
        .toEqual([{ loc: '/item/587', priority: 0.6 }])
})

test('unknown sitemap categories never reach the backend', async () => {
    const before = requests.length
    await expect(loadSitemapSource('../site', { apiSocket: socket, apiOrigin: '' })).rejects.toThrow('Unknown sitemap category')
    expect(requests.length).toBe(before)
})

test('backend failure propagates instead of producing an empty successful source', async () => {
    await expect(loadSitemapSource('kills', { apiSocket: socket, apiOrigin: '' })).rejects.toThrow()
})

test('only canonical and local hosts can generate sitemaps; only apex is indexable', () => {
    for (const host of ['eve-kill.com', 'EVE-KILL.COM:443', 'eve-kill.com.']) {
        expect(isCanonicalSeoHost(host)).toBe(true)
        expect(canServeSitemap(host)).toBe(true)
    }
    for (const host of ['apps.eve-kill.com', 'www.eve-kill.com', 'my-killboard.example', 'eve-kill.com.example']) {
        expect(isCanonicalSeoHost(host)).toBe(false)
        expect(canServeSitemap(host)).toBe(false)
    }
    for (const host of ['localhost:3000', '127.0.0.1:3000', '[::1]:3000']) {
        expect(canServeSitemap(host)).toBe(true)
        expect(isCanonicalSeoHost(host)).toBe(false)
    }
    for (const path of ['/sitemap.xml', '/sitemap_index.xml', '/__sitemap__/ships.xml', '/_sitemap-data/ships']) {
        expect(isSitemapPath(path)).toBe(true)
    }
    expect(isSitemapPath('/item/587')).toBe(false)
})
