import { describe, expect, test } from 'bun:test'
import { savedSearchDocument, savedSearchQuery, readLocalSearches } from '../app/utils/savedSearches'
import { adjacentKill, parseKillNavigation } from '../app/utils/killNavigation'

describe('saved searches', () => {
    test('legacy saves preserve display names and normalize shorthand and defaults', () => {
        const document = savedSearchDocument(JSON.stringify({ entities: { victim: [{ id: 42, type: 'character', name: 'Pilot', role: 'victim' }] }, location: { systemId: 30000142, systemName: 'Jita' }, iskMin: '1.5b', iskMax: '2000m', items: [{ typeId: 34, name: 'Tritanium' }] }))
        expect(document.filters.iskMin).toBe(1_500_000_000)
        expect(document.filters.iskMax).toBe(2_000_000_000)
        expect(document.filters.entities.victim[0].role).toBeUndefined()
        expect(document.filters.location.systemName).toBe('Jita')
        expect(document.filters.items[0]).toMatchObject({ slot: 'any', side: 'victim' })
        expect(document.filters.timeRange).toEqual({ preset: '30d' })
        expect(document.filters.sort).toEqual({ field: 'killmail_time', direction: 'desc' })
        expect(JSON.parse(savedSearchQuery(document).q!)).toEqual(document.filters)
    })
    test('fit views and family drilldowns survive the account round trip', () => {
        const document = savedSearchDocument('{"timeRange":{"preset":"7d"},"sort":{"field":"total_value","direction":"asc"}}', { view: 'fits', dedup: 'exact', familyHash: 'a'.repeat(64) })
        const query = savedSearchQuery(JSON.parse(JSON.stringify(document)))
        expect(query).toMatchObject({ view: 'fits', dedup: 'exact', dh: 'a'.repeat(64), dm: 'family' })
        expect(JSON.parse(query.q!).timeRange).toEqual({ preset: '7d' })
    })
    test('custom dates remain absolute and zero ISK is retained', () => {
        const document = savedSearchDocument('{"timeRange":{"from":"2026-01-01","to":"2026-01-02"},"iskMin":"0"}')
        expect(document.filters.timeRange).toEqual({ from: '2026-01-01', to: '2026-01-02' })
        expect(document.filters.iskMin).toBe(0)
    })
    test('invalid browser data is rejected without rewriting storage', () => {
        expect(() => readLocalSearches('{bad')).toThrow()
        expect(() => savedSearchDocument('null')).toThrow()
        expect(() => savedSearchDocument('{"iskMin":"1.2.3b"}')).toThrow()
        expect(readLocalSearches('[{"name":"Existing","filters":"{}","date":"2026-09-07"}]')[0]?.name).toBe('Existing')
    })
})

describe('contextual kill navigation', () => {
    const context = { version: 1 as const, createdAt: 1000, returnTo: '/advancedsearch?q=%7B%7D', label: 'Advanced Search', ids: [42, 100, 9] }
    test('follows displayed ordering instead of sorting by ID', () => {
        expect(adjacentKill(context, 100, -1)).toBe(42)
        expect(adjacentKill(context, 100, 1)).toBe(9)
        expect(adjacentKill(context, 42, -1)).toBeNull()
        expect(adjacentKill(context, 999, 1)).toBeNull()
    })
    test('expires and rejects malformed or external return paths', () => {
        expect(parseKillNavigation(JSON.stringify(context), 1001)).toEqual(context)
        expect(parseKillNavigation(JSON.stringify(context), 9_000_000)).toBeNull()
        for (const returnTo of ['https://evil.example', '//evil.example', '/%2fevil.example', '/\\evil.example']) {
            expect(parseKillNavigation(JSON.stringify({ ...context, returnTo }), 1001)).toBeNull()
        }
        expect(parseKillNavigation(JSON.stringify({ ...context, ids: Array(101).fill(42) }), 1001)).toBeNull()
        expect(parseKillNavigation(JSON.stringify({ ...context, ids: [42,42] }), 1001)).toBeNull()
    })
})
