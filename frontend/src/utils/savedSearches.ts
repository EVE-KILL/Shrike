import {isRecord} from '../browser/shared/json'
import {restoreAdvancedFilters,advancedRoles,advancedLocations,advancedScalarFilters} from './advancedSearch'
import type {AdvancedFilters,SavedSearchDocument} from '../../shared/api/types.gen'
export type {SavedSearchDocument} from '../../shared/api/types.gen'

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
    const parsed:unknown = JSON.parse(raw)
    if (!isRecord(parsed)) throw new Error('Invalid saved search')
    const restored=restoreAdvancedFilters(parsed),filters:AdvancedFilters={items:restored.items.map(item=>({...item})),sort:{...restored.sort}}
    if(restored.label)filters.label=restored.label
    for(const key of advancedScalarFilters){const value=restored[key];if(value)filters[key]=value}
    if(isRecord(parsed.entities)){
        filters.entities={}
        for(const role of advancedRoles)if(Array.isArray(parsed.entities[role]))filters.entities[role]=restored.entities[role].map(entity=>({id:entity.id,type:entity.type,name:entity.name,...(entity.exclude?{exclude:true}:{})}))
    }
    if(isRecord(parsed.location)){
        filters.location={}
        if(Array.isArray(parsed.location.securityTypes))filters.location.securityTypes=[...restored.location.securityTypes]
        for(const kind of advancedLocations){const id=restored.location[`${kind}Id`],name=restored.location[`${kind}Name`];if(id!==null)filters.location[`${kind}Id`]=id;if(name!==null)filters.location[`${kind}Name`]=name}
    }
    const range=restored.timeRange
    filters.timeRange='from' in range?{from:range.from,...(range.to!==undefined?{to:range.to}:{})}:range.preset===null?{}:{preset:range.preset}
    for(const key of ['iskMin','iskMax'] as const){const amount=parseSearchIsk(parsed[key]);if(amount!==null)filters[key]=amount}
    if(filters.iskMin!==undefined||filters.iskMax!==undefined)delete filters.iskValue
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
    return rows.flatMap(row=>isRecord(row)&&typeof row.name==='string'&&typeof row.filters==='string'&&typeof row.date==='string'?[{name:row.name,filters:row.filters,date:row.date,...(row.document!==undefined?{document:readSavedSearchDocument(row.document)}:{})}]:[])
}

export function readSavedSearchDocument(value:unknown):SavedSearchDocument {
    if(!isRecord(value)||value.version!==1||(value.view!=='kills'&&value.view!=='fits')||!['none','exact','family'].includes(String(value.dedup)))throw new Error('Invalid saved-search document')
    const dedup=value.dedup;if(dedup!=='none'&&dedup!=='exact'&&dedup!=='family')throw new Error('Invalid saved-search grouping')
    return savedSearchDocument(JSON.stringify(value.filters),{view:value.view,dedup,...(typeof value.fitHash==='string'?{fitHash:value.fitHash}:{}),...(typeof value.familyHash==='string'?{familyHash:value.familyHash}:{})})
}
