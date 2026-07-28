// Adapts the OpenAPI document into the shapes /docs already renders.
//
// The page used to read a hand-maintained index from GET /api/. That index
// covered 98 endpoints while the API serves 441, and it had to be edited by
// hand whenever a route changed. Reading the generated document instead means
// the page cannot drift from the API, and every endpoint appears without
// anyone remembering to add it.
//
// The conversion targets the existing ApiEndpoint type rather than a new one,
// so ApiDocsEndpoint.vue — its parameter inputs, curl builder, and try-it
// runner — keeps working untouched.

import type {
    ApiBody,
    ApiBodyField,
    ApiEndpoint,
    ApiParam,
    EntityKind,
    HttpMethod,
    ParamType,
} from './apiDocs'

export interface DocsTag {
    name: string
    description?: string
    endpoints: ApiEndpoint[]
}

export interface DocsGroup {
    name: string
    tags: DocsTag[]
}

export interface DocsModel {
    title: string
    version: string
    description?: string
    groups: DocsGroup[]
    total: number
}

interface OpenAPIDocument {
    info?: { title?: string; version?: string; description?: string }
    paths?: Record<string, Record<string, any>>
    components?: { schemas?: Record<string, any> }
    tags?: Array<{ name: string; description?: string }>
    'x-tagGroups'?: Array<{ name: string; tags: string[] }>
}

const METHODS: HttpMethod[] = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE']

/**
 * Resolve a `$ref` against the document, following chains but refusing to
 * loop. Schemas reference each other freely, and a self-referential type
 * would otherwise hang the page rather than fail visibly.
 */
function resolveSchema(
    schema: any,
    doc: OpenAPIDocument,
    seen = new Set<string>(),
): any {
    if (!schema || typeof schema !== 'object') return schema
    const ref = schema.$ref
    if (typeof ref !== 'string') return schema
    if (seen.has(ref)) return {}
    seen.add(ref)
    const name = ref.replace('#/components/schemas/', '')
    const target = doc.components?.schemas?.[name]
    return target ? resolveSchema(target, doc, seen) : {}
}

/**
 * Map a JSON Schema onto the narrower input type the form controls use.
 */
function paramType(schema: any): ParamType {
    if (!schema) return 'string'
    if (Array.isArray(schema.enum) && schema.enum.length > 0) return 'enum'
    if (schema.format === 'date' || schema.format === 'date-time') return 'date'
    if (schema.type === 'integer' || schema.type === 'number') return 'int'
    // A boolean has no input of its own here. Rendering it as a two-value
    // choice beats a free-text box that accepts anything and means nothing.
    if (schema.type === 'boolean') return 'enum'
    return 'string'
}

/**
 * Enumerated values for the control, including the synthetic pair booleans
 * need to render as a choice.
 */
function paramValues(schema: any): string[] | undefined {
    if (Array.isArray(schema?.enum) && schema.enum.length > 0) {
        return schema.enum.map(String)
    }
    if (schema?.type === 'boolean') return ['true', 'false']
    return undefined
}

// Path segments that identify what an id refers to, so the entity picker can
// offer a real search instead of a bare number field. Only exact, unambiguous
// segments are listed — a wrong guess here sends people to the wrong picker,
// which is worse than no picker at all.
const ENTITY_BY_SEGMENT: Record<string, EntityKind> = {
    characters: 'character',
    character: 'character',
    corporations: 'corporation',
    corporation: 'corporation',
    alliances: 'alliance',
    alliance: 'alliance',
    ships: 'ship',
    ship: 'ship',
    systems: 'system',
    system: 'system',
    regions: 'region',
    region: 'region',
    constellations: 'constellation',
    constellation: 'constellation',
    factions: 'faction',
    faction: 'faction',
    types: 'item',
    type: 'item',
    items: 'item',
    item: 'item',
    oldcharacters: 'character',
    sovereignty: 'system',
    prices: 'item',
}

function entityKind(path: string, paramName: string): EntityKind | undefined {
    if (!/^(id|.*Id)$/.test(paramName)) return undefined
    const segments = path.split('/').filter(Boolean)
    const index = segments.findIndex(s => s === `{${paramName}}`)
    if (index <= 0) return undefined
    return ENTITY_BY_SEGMENT[segments[index - 1]!]
}

function toParam(raw: any, doc: OpenAPIDocument, path: string): ApiParam {
    const schema = resolveSchema(raw.schema, doc)
    // A path parameter's kind is read from the segment before it. A query
    // parameter has no segment to read, so the API states the kind on the
    // parameter itself and we take it at its word.
    const entity: EntityKind | undefined =
        raw.in === 'path'
            ? entityKind(path, raw.name)
            : (raw['x-entity'] as EntityKind | undefined)
    return {
        name: raw.name,
        // 'entity' is what switches the card from a bare number box to the
        // search picker. Knowing the kind without setting this leaves the
        // picker unrendered, which is how it stayed dark until now.
        type: entity ? 'entity' : paramType(schema),
        required: Boolean(raw.required),
        description: raw.description || schema?.description || undefined,
        default: schema?.default,
        values: paramValues(schema),
        example: raw.example ?? schema?.examples?.[0] ?? schema?.example,
        entityType: entity,
    }
}

function bodyFieldType(schema: any): ApiBodyField['type'] {
    if (schema?.type === 'array') {
        const item = schema.items?.type
        return item === 'integer' || item === 'number' ? 'int[]' : 'string[]'
    }
    if (Array.isArray(schema?.enum) && schema.enum.length > 0) return 'enum'
    if (schema?.format === 'date' || schema?.format === 'date-time') return 'date'
    if (schema?.type === 'integer' || schema?.type === 'number') return 'int'
    return 'string'
}

function toBody(operation: any, doc: OpenAPIDocument): ApiBody | undefined {
    const media = operation.requestBody?.content?.['application/json']
    if (!media) return undefined
    const schema = resolveSchema(media.schema, doc)
    const properties = schema?.properties
    if (!properties || typeof properties !== 'object') return undefined

    const required: string[] = Array.isArray(schema.required) ? schema.required : []
    const fields: Record<string, ApiBodyField> = {}
    for (const [name, rawField] of Object.entries<any>(properties)) {
        const field = resolveSchema(rawField, doc)
        fields[name] = {
            type: bodyFieldType(field),
            description: field?.description,
            required: required.includes(name),
            values: Array.isArray(field?.enum) ? field.enum.map(String) : undefined,
            default: field?.default,
            maxItems: field?.maxItems,
        }
    }
    return { contentType: 'application/json', fields, example: media.example }
}

/**
 * Describe the success response in one line, for the collapsed card.
 */
function describeReturns(operation: any, doc: OpenAPIDocument): string | undefined {
    const responses = operation.responses ?? {}
    const key = Object.keys(responses).find(code => code.startsWith('2'))
    if (!key) return undefined
    const response = responses[key]
    const schema = resolveSchema(
        response?.content?.['application/json']?.schema,
        doc,
    )
    if (schema?.type === 'array') return `${key} — array`
    const description = (response?.description || '').trim()
    if (description && !/^(ok|created|no content)$/i.test(description)) {
        return `${key} — ${description}`
    }
    return key
}

/**
 * ApiDocsEndpoint fills `:name` tokens. OpenAPI writes `{name}`.
 */
function toColonPath(path: string): string {
    return path.replace(/\{([^}]+)\}/g, ':$1')
}

function toEndpoint(
    method: HttpMethod,
    path: string,
    operation: any,
    doc: OpenAPIDocument,
): ApiEndpoint {
    const parameters: any[] = (operation.parameters ?? []).map((p: any) =>
        p.$ref ? resolveSchema(p, doc) : p,
    )
    const pathParams = parameters
        .filter(p => p.in === 'path')
        .map(p => toParam(p, doc, path))
    const query = parameters
        .filter(p => p.in === 'query')
        .map(p => toParam(p, doc, path))

    return {
        method,
        path: toColonPath(path),
        summary: operation.summary || `${method} ${path}`,
        description: operation.description || undefined,
        pathParams: pathParams.length ? pathParams : undefined,
        query: query.length ? query : undefined,
        body: toBody(operation, doc),
        returns: describeReturns(operation, doc),
        operationId: operation.operationId,
        tags: operation.tags ?? [],
        deprecated: Boolean(operation.deprecated),
    }
}

/**
 * Build the grouped model the sidebar renders.
 *
 * Ordering comes from the document: x-tagGroups gives the sections, and the
 * top-level tags array gives the order inside each one. Anything the document
 * failed to place lands in a trailing "Other" group rather than disappearing,
 * because a missing endpoint is harder to notice than a misfiled one.
 */
export function buildDocsModel(doc: OpenAPIDocument): DocsModel {
    const tagDescription = new Map<string, string>()
    for (const tag of doc.tags ?? []) {
        if (tag.description) tagDescription.set(tag.name, tag.description)
    }

    const byTag = new Map<string, ApiEndpoint[]>()
    let total = 0
    for (const [path, item] of Object.entries(doc.paths ?? {})) {
        for (const method of METHODS) {
            const operation = item[method.toLowerCase()]
            if (!operation) continue
            const endpoint = toEndpoint(method, path, operation, doc)
            total++
            const tags = endpoint.tags?.length ? endpoint.tags : ['other']
            for (const tag of tags) {
                if (!byTag.has(tag)) byTag.set(tag, [])
                byTag.get(tag)!.push(endpoint)
            }
        }
    }

    for (const endpoints of byTag.values()) {
        endpoints.sort((a, b) =>
            a.path.localeCompare(b.path) || a.method.localeCompare(b.method),
        )
    }

    const placed = new Set<string>()
    const groups: DocsGroup[] = []
    for (const group of doc['x-tagGroups'] ?? []) {
        const tags: DocsTag[] = []
        for (const name of group.tags) {
            const endpoints = byTag.get(name)
            if (!endpoints?.length) continue
            placed.add(name)
            tags.push({ name, description: tagDescription.get(name), endpoints })
        }
        if (tags.length) groups.push({ name: group.name, tags })
    }

    const leftover = [...byTag.keys()].filter(t => !placed.has(t)).sort()
    if (leftover.length) {
        groups.push({
            name: 'Other',
            tags: leftover.map(name => ({
                name,
                description: tagDescription.get(name),
                endpoints: byTag.get(name)!,
            })),
        })
    }

    return {
        title: doc.info?.title ?? 'API',
        version: doc.info?.version ?? '',
        description: doc.info?.description,
        groups,
        total,
    }
}
