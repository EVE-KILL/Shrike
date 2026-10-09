import type {SearchResponse} from '../../../shared/api/types.gen'
import {isRecord} from './json'

export const searchTypes = {
  character:{label:'Characters',icon:'users'},corporation:{label:'Corporations',icon:'network'},alliance:{label:'Alliances',icon:'shield'},faction:{label:'Factions',icon:'flag'},
  system:{label:'Systems',icon:'map'},region:{label:'Regions',icon:'map'},constellation:{label:'Constellations',icon:'map'},ship:{label:'Ships',icon:'swords'},item:{label:'Items',icon:'gem'},shipgroup:{label:'Ship groups',icon:'list-ordered'},group:{label:'Groups',icon:'list-ordered'},
} as const
export type SearchType = keyof typeof searchTypes
type WireHit = SearchResponse['hits'][number]
export type SearchEntity = Pick<WireHit,'name'|'ticker'|'corporation_id'|'corporation_name'|'corporation_ticker'|'alliance_id'|'alliance_name'|'alliance_ticker'> & {id:number;type:SearchType}
export type SearchResults = Pick<SearchResponse,'total'|'processingTimeMs'> & {hits:SearchEntity[];counts:Partial<Record<SearchType,number>>}
export function isSearchType(value:unknown):value is SearchType {return typeof value === 'string' && Object.hasOwn(searchTypes,value)}
export function searchID(value:unknown):number {
  const raw = typeof value === 'number' ? String(value) : typeof value === 'string' ? value.replace(/^[a-z]+_/i,'') : ''
  if(!/^\d+$/.test(raw)) return 0
  const id = Number(raw);return Number.isSafeInteger(id) && id > 0 && id <= 2147483647 ? id : 0
}
const text = (value:unknown)=>typeof value === 'string' ? value : undefined
const nonnegative = (value:unknown)=>typeof value === 'number' && Number.isFinite(value) && value >= 0 ? value : 0
export function readSearchResults(value:unknown):SearchResults {
  if(!isRecord(value) || !Array.isArray(value.hits)) throw new Error('Invalid search response')
  const hits = value.hits.flatMap((hit:unknown):SearchEntity[]=>{
    if(!isRecord(hit) || !isSearchType(hit.type)) return []
    const id = searchID(hit.id);if(!id) return []
    return [{id,type:hit.type,name:text(hit.name),ticker:text(hit.ticker),corporation_id:searchID(hit.corporation_id) || undefined,corporation_name:text(hit.corporation_name),corporation_ticker:text(hit.corporation_ticker),alliance_id:searchID(hit.alliance_id) || undefined,alliance_name:text(hit.alliance_name),alliance_ticker:text(hit.alliance_ticker)}]
  })
  const counts:Partial<Record<SearchType,number>> = {}
  if(isRecord(value.entityCounts)) for(const [type,count] of Object.entries(value.entityCounts)) if(isSearchType(type) && Number.isSafeInteger(count) && typeof count === 'number' && count > 0) counts[type] = count
  return {hits,counts,total:nonnegative(value.total) || hits.length,processingTimeMs:nonnegative(value.processingTimeMs)}
}
export function searchEntityPath(hit:Pick<SearchEntity,'id'|'type'>):string {return `/${hit.type === 'ship' || hit.type === 'item' ? 'item' : hit.type === 'shipgroup' || hit.type === 'group' ? 'group' : hit.type}/${hit.id}`}
export function searchEntityImage(hit:Pick<SearchEntity,'id'|'type'>):string {
  switch(hit.type) {
    case 'character':return `https://images.evetech.net/characters/${hit.id}/portrait?size=512`
    case 'corporation':case 'faction':return `https://images.evetech.net/corporations/${hit.id}/logo?size=256`
    case 'alliance':return `https://images.evetech.net/alliances/${hit.id}/logo?size=256`
    case 'ship':return `/images/types/${hit.id}/overlayrender?size=512`
    case 'item':return `/images/types/${hit.id}/icon?size=64`
    case 'system':case 'region':case 'constellation':return `/images/${hit.type}s/${hit.id}?size=128`
    default:return ''
  }
}
