// Shared types and helpers for the /docs API explorer.
// These mirror the schema served by the backend at GET /.

export type ParamType = 'int' | 'string' | 'date' | 'enum' | 'entity'

// The generated document covers the whole API, not just the read surface the
// hand-maintained index described, so every method the routes register has to
// render.
export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

export type EntityKind =
    | 'character'
    | 'corporation'
    | 'alliance'
    | 'ship'
    | 'item'
    | 'system'
    | 'region'
    | 'constellation'
    | 'faction'

export interface ApiParam {
    name: string
    type: ParamType
    required?: boolean
    description?: string
    default?: string | number
    values?: string[]
    entityType?: EntityKind
    example?: string | number
    pattern?: string
}

export interface ApiBodyField {
    type: 'int' | 'string' | 'date' | 'enum' | 'int[]' | 'string[]'
    description?: string
    required?: boolean
    values?: string[]
    default?: string | number
    maxItems?: number
}

export interface ApiBody {
    contentType: 'application/json'
    fields: Record<string, ApiBodyField>
    example?: unknown
}

export interface ApiExample {
    label: string
    path?: string
    body?: unknown
}

export interface ApiEndpoint {
    method: HttpMethod
    path: string
    summary: string
    description?: string
    pathParams?: ApiParam[]
    query?: ApiParam[]
    body?: ApiBody
    examples?: ApiExample[]
    notes?: string
    returns?: string
    // Present when the endpoint came from the OpenAPI document. The operation
    // ID is the stable anchor for deep links; tags drive sidebar placement.
    operationId?: string
    tags?: string[]
    deprecated?: boolean
}

export interface ApiCategory {
    key: string
    label: string
    description?: string
    endpoints: ApiEndpoint[]
}

export interface ApiIndex {
    name: string
    version: string
    baseUrl: string
    description: string
    categories: ApiCategory[]
}

/**
 * Substitute :pathParam tokens in a path template with their resolved values.
 * Missing values are kept as-is so the user sees what's still required.
 */
export function fillPath(path: string, values: Record<string, string | number>): string {
    return path.replace(/:([A-Za-z0-9_]+)/g, (_, name) => {
        const v = values[name]
        return v != null && v !== '' ? String(v) : `:${name}`
    })
}

/**
 * Build the full URL for an endpoint, including query string.
 */
export function buildUrl(
    baseUrl: string,
    endpoint: ApiEndpoint,
    pathValues: Record<string, string | number>,
    queryValues: Record<string, string | number>,
): string {
    const path = fillPath(endpoint.path, pathValues)
    const query = new URLSearchParams()
    for (const p of endpoint.query ?? []) {
        const v = queryValues[p.name]
        if (v != null && v !== '') query.set(p.name, String(v))
    }
    const qs = query.toString()
    return `${baseUrl}${path}${qs ? `?${qs}` : ''}`
}

/**
 * Build a curl command for the given endpoint, ready to copy/paste.
 */
export function buildCurl(
    baseUrl: string,
    endpoint: ApiEndpoint,
    pathValues: Record<string, string | number>,
    queryValues: Record<string, string | number>,
    bodyValue: unknown,
): string {
    const url = buildUrl(baseUrl, endpoint, pathValues, queryValues)
    if (endpoint.method === 'GET') {
        return `curl '${url}'`
    }
    if (!endpoint.body) {
        return `curl -X ${endpoint.method} '${url}'`
    }
    const body = JSON.stringify(bodyValue ?? {})
    return `curl -X ${endpoint.method} '${url}' \\\n  -H 'Content-Type: application/json' \\\n  -d '${body.replace(/'/g, `'\\''`)}'`
}
