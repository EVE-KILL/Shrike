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
        expect(withQuery.length).toBeGreaterThan(150)
        const enums = all.flatMap(e => e.query ?? []).filter(p => p.type === 'enum')
        expect(enums.every(p => (p.values?.length ?? 0) > 0)).toBe(true)
    })

    // The advanced search is a GET whose entire query is its search form. It
    // rendered with no inputs at all while the document declared none of it.
    test('gives the advanced killmail search its search inputs', () => {
        const all = model.groups.flatMap(g => g.tags.flatMap(t => t.endpoints))
        const advanced = all.find(e => e.path === '/killlist/advanced')
        expect(advanced).toBeDefined()
        const names = (advanced!.query ?? []).map(p => p.name)
        expect(names).toEqual(
            expect.arrayContaining([
                'filters', 'view', 'dedup', 'fitHash', 'familyHash',
                'limit', 'after',
            ]),
        )
        const view = advanced!.query!.find(p => p.name === 'view')
        expect(view?.type).toBe('enum')
        expect(view?.values).toEqual(['kills', 'fits'])
        expect(view?.default).toBe('kills')
    })

    // A query parameter has no path segment to infer its kind from, so the
    // API states it. Without this the picker stays a bare number box.
    test('offers the entity picker on entity-typed query parameters', () => {
        const all = model.groups.flatMap(g => g.tags.flatMap(t => t.endpoints))
        const matchup = all.find(e => e.path === '/matchup')
        const attacker = matchup?.query?.find(p => p.name === 'attacker')
        expect(attacker?.type).toBe('entity')
        expect(attacker?.entityType).toBe('ship')
    })

    // A boolean has no control of its own on the card. It has to arrive as a
    // two-value choice or it renders as free text that accepts anything.
    test('renders boolean filters as a two-value choice', () => {
        const all = model.groups.flatMap(g => g.tags.flatMap(t => t.endpoints))
        const wars = all.find(e => e.path === '/conflicts/wars')
        const mutual = wars?.query?.find(p => p.name === 'mutual')
        expect(mutual?.type).toBe('enum')
        expect(mutual?.values).toEqual(['true', 'false'])
    })

    // Write endpoints now carry generated schemas, so the card renders real
    // fields instead of an empty textarea.
    test('exposes request bodies as editable fields', () => {
        const all = model.groups.flatMap(g => g.tags.flatMap(t => t.endpoints))
        const withBody = all.filter(e => e.method !== 'GET' && e.body)
        expect(withBody.length).toBeGreaterThan(50)

        const resolve = all.find(e => e.path === '/resolve' && e.method === 'POST')
        expect(Object.keys(resolve!.body!.fields)).toContain('names')
        expect(resolve!.body!.fields.names!.required).toBe(true)
    })

    // The action endpoints read no body at all, and the two uploads are
    // multipart. The converter must leave those as body-less cards rather
    // than inventing an empty form.
    test('leaves body-less write endpoints without a form', () => {
        const all = model.groups.flatMap(g => g.tags.flatMap(t => t.endpoints))
        const logout = all.find(e => e.path === '/auth/logout' && e.method === 'POST')
        expect(logout).toBeDefined()
        expect(logout!.body).toBeUndefined()
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
