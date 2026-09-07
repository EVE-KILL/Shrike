export interface SavedSearchDocument {
    version: 1
    filters: Record<string, any>
    view: 'kills' | 'fits'
    dedup: 'none' | 'exact' | 'family'
    fitHash?: string
    familyHash?: string
}

export interface LocalSavedSearch {
    name: string
    filters: string
    date: string
    document?: SavedSearchDocument
}

export function parseSearchIsk(value: unknown): number | null {
    if (value === '' || value == null) return null
    const match = String(value).trim().match(/^(\d+(?:\.\d*)?|\.\d+)\s*([bmk])?$/i)
    if (!match) throw new Error('Invalid ISK amount in saved search')
    const amount = Number(match[1]) * ({ b: 1e9, m: 1e6, k: 1e3 }[match[2]?.toLowerCase() ?? ''] ?? 1)
    if (!Number.isFinite(amount)) throw new Error('Invalid ISK amount in saved search')
    return amount
}

// The URL format remains readable by older links. Stored documents explicitly
// record defaults so later UI default changes cannot reinterpret saved searches.
export function savedSearchDocument(raw: string, options: Partial<Omit<SavedSearchDocument, 'version' | 'filters'>> = {}): SavedSearchDocument {
    const parsed = JSON.parse(raw)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) throw new Error('Invalid saved search')
    const filters = structuredClone(parsed)
    if (filters.entities) {
        for (const side of ['victim', 'attacker', 'both']) {
            if (filters.entities[side]) filters.entities[side] = filters.entities[side].map(({ role: _role, ...entity }: Record<string, unknown>) => entity)
        }
    }
    for (const key of ['iskMin', 'iskMax']) {
        const amount = parseSearchIsk(filters[key])
        if (amount == null) delete filters[key]
        else filters[key] = amount
    }
    if (filters.iskMin != null || filters.iskMax != null) delete filters.iskValue
    filters.timeRange ??= { preset: '30d' }
    filters.sort ??= { field: 'killmail_time', direction: 'desc' }
    filters.items = (filters.items ?? []).map((item: Record<string, unknown>) => ({ ...item, slot: item.slot ?? 'any', side: item.side ?? 'victim' }))
    return { version: 1, filters, view: options.view ?? 'kills', dedup: options.dedup ?? 'family', ...(options.fitHash ? { fitHash: options.fitHash } : {}), ...(options.familyHash ? { familyHash: options.familyHash } : {}) }
}

export function savedSearchQuery(document: SavedSearchDocument): Record<string, string> {
    if (document.version !== 1) throw new Error('Unsupported saved-search version')
    const query: Record<string, string> = { q: JSON.stringify(document.filters) }
    if (document.view === 'fits') { query.view = 'fits'; query.dedup = document.dedup }
    if (document.fitHash) { query.dh = document.fitHash; query.dm = 'exact' }
    else if (document.familyHash) { query.dh = document.familyHash; query.dm = 'family' }
    return query
}

export function readLocalSearches(raw: string | null): LocalSavedSearch[] {
    if (!raw) return []
    const rows: unknown = JSON.parse(raw)
    if (!Array.isArray(rows)) throw new Error('Browser saved searches could not be read')
    return rows.filter((row): row is LocalSavedSearch => !!row && typeof row.name === 'string' && typeof row.filters === 'string' && typeof row.date === 'string')
}
