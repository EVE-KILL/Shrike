import { describe, expect, test } from 'bun:test'
import { buildDocsModel } from '../app/composables/openapiDocs'
import spec from '../shared/api.openapi.json'

// The page renders whatever this produces, so it is tested against the real
// generated document rather than a fixture. A fixture would keep passing after
// the API changed shape, which is the failure this is meant to catch.
const model = buildDocsModel(spec as any)

describe('OpenAPI to docs model', () => {
    test('places every operation in the document into a group', () => {
        const grouped = model.groups.reduce(
            (sum, group) => sum + group.tags.reduce((n, t) => n + t.endpoints.length, 0),
            0,
        )
        // Operations carrying two tags appear under both, so the grouped count
        // is at least the operation count and never below it.
        expect(grouped).toBeGreaterThanOrEqual(model.total)
        expect(model.total).toBeGreaterThan(400)
    })

    test('nothing lands in the Other fallback group', () => {
        // Other exists so a newly tagged route stays reachable, but a document
        // whose x-tagGroups are complete should never need it.
        expect(model.groups.map(g => g.name)).not.toContain('Other')
    })

    test('keeps the section order the document declares', () => {
        expect(model.groups[0]!.name).toBe('Killboard')
        expect(model.groups.map(g => g.name)).toContain('Administration')
    })

    test('converts path parameters to the :name form the card fills', () => {
        const all = model.groups.flatMap(g => g.tags.flatMap(t => t.endpoints))
        const killmail = all.find(e => e.path === '/killmails/:id' && e.method === 'GET')
        expect(killmail).toBeDefined()
        expect(killmail!.pathParams?.[0]?.name).toBe('id')
        // No OpenAPI brace syntax should survive the conversion.
        expect(all.some(e => e.path.includes('{'))).toBe(false)
    })

    test('carries the descriptions written for the public surface', () => {
        const all = model.groups.flatMap(g => g.tags.flatMap(t => t.endpoints))
        const count = all.find(e => e.path === '/killmails/count')
        expect(count?.description).toContain('pg_class.reltuples')
    })

    test('reads query parameters, enums, and defaults', () => {
        const all = model.groups.flatMap(g => g.tags.flatMap(t => t.endpoints))
        const withQuery = all.filter(e => (e.query?.length ?? 0) > 0)
        expect(withQuery.length).toBeGreaterThan(0)
        const enums = all.flatMap(e => e.query ?? []).filter(p => p.type === 'enum')
        expect(enums.every(p => (p.values?.length ?? 0) > 0)).toBe(true)
    })

    // No write operation currently declares a JSON body schema: the handlers
    // read req.Body directly, so Huma has no input struct to describe. The
    // converter must therefore degrade to a body-less card rather than throw,
    // and this pins that until the handlers grow typed inputs.
    test('survives write operations that declare no body schema', () => {
        const all = model.groups.flatMap(g => g.tags.flatMap(t => t.endpoints))
        const writes = all.filter(e => e.method !== 'GET')
        expect(writes.length).toBeGreaterThan(0)
        expect(writes.every(e => e.body === undefined)).toBe(true)
    })

    test('surfaces every method the API registers, not just GET and POST', () => {
        const all = model.groups.flatMap(g => g.tags.flatMap(t => t.endpoints))
        const methods = new Set(all.map(e => e.method))
        expect(methods.has('PUT')).toBe(true)
        expect(methods.has('DELETE')).toBe(true)
    })

    test('assigns entity pickers only where the segment is unambiguous', () => {
        const all = model.groups.flatMap(g => g.tags.flatMap(t => t.endpoints))
        const character = all.find(e => e.path === '/characters/:id' && e.method === 'GET')
        expect(character?.pathParams?.[0]?.entityType).toBe('character')
        // A generic id with no entity segment before it must not guess.
        const blog = all.find(e => e.path === '/blog/:slug')
        expect(blog?.pathParams?.[0]?.entityType).toBeUndefined()
    })
})
