import {isRecord} from './json'
import {searchID} from './search-model'
export const marketSorts = ['quantity','price','min_volume','expires'] as const
export type MarketSort = typeof marketSorts[number]
export const marketSecurities = ['high','low','null'] as const
export type MarketSecurity = typeof marketSecurities[number]
export type MarketFilters = {region:string;days:'30'|'90';security:MarketSecurity[]}
export type MarketHit = {id:number;name:string;type:'ship'|'item'}
export function isMarketSort(value:unknown):value is MarketSort {return marketSorts.some(key=>key === value)}
export function isMarketSecurity(value:unknown):value is MarketSecurity {return marketSecurities.some(key=>key === value)}
export function isMarketRegion(value:unknown):value is string {return typeof value === 'string' && /^(0|[1-9]\d*)$/.test(value) && Number.isSafeInteger(Number(value)) && Number(value) <= 2147483647}
export function marketID(value:unknown):number {if(typeof value !== 'string' && typeof value !== 'number') return 0;const n = Number(value);return Number.isSafeInteger(n) && n > 0 && n <= 2147483647 ? n : 0}
export function readMarketFilters(data:{region?:string;days?:string;security?:string}):MarketFilters {
  if(!isMarketRegion(data.region) || data.days !== '30' && data.days !== '90') throw new Error('Invalid market filters')
  const raw = data.security?.split(',') ?? [],security = marketSecurities.filter(value=>raw.includes(value))
  if(!security.length || raw.some(value=>!isMarketSecurity(value))) throw new Error('Invalid market filters')
  return {region:data.region,days:data.days,security:[...security]}
}
export function marketParams(filters:MarketFilters):URLSearchParams {return new URLSearchParams({region:filters.region,days:filters.days,security:filters.security.join(',')})}
export function toggleMarketSecurity(filters:MarketFilters,value:MarketSecurity):MarketFilters|undefined {
  const selected = filters.security.includes(value) ? filters.security.filter(item=>item !== value) : [...filters.security,value]
  return selected.length ? {...filters,security:marketSecurities.filter(item=>selected.includes(item))} : undefined
}
const number = (value:string|null)=>{const n = Number(value);return Number.isFinite(n) ? n : 0}
const orderID = (row:HTMLElement)=>{const id = row.dataset.orderId ?? '';return /^\d+$/.test(id) ? BigInt(id) : 0n}
export function sortMarketBook(book:HTMLElement,key:MarketSort,direction:1|-1):void {
  const tbody = book.querySelector('tbody');if(!tbody) return
  const attribute = 'data-'+key.replaceAll('_','-'),rows = [...tbody.querySelectorAll<HTMLTableRowElement>('tr[data-order-id]')]
  rows.sort((a,b)=>{const difference = (number(a.getAttribute(attribute))-number(b.getAttribute(attribute)))*direction;if(difference) return difference;const left = orderID(a),right = orderID(b);return left < right ? -1 : left > right ? 1 : 0})
  if(rows.length) tbody.replaceChildren(...rows)
  for(const header of book.querySelectorAll('th[aria-sort]')) header.setAttribute('aria-sort',header.querySelector<HTMLElement>('[data-market-sort]')?.dataset.marketSort === key ? direction === 1 ? 'ascending' : 'descending' : 'none')
}
export function readMarketHits(value:unknown,requested:'ship'|'item'):MarketHit[] {
  if(!isRecord(value) || !Array.isArray(value.hits)) throw new Error('Search unavailable')
  return value.hits.flatMap((hit:unknown)=>{
    if(!isRecord(hit) || typeof hit.name !== 'string') return []
    const type = hit.type ?? requested;if(type !== 'ship' && type !== 'item') return []
    const id = searchID(hit.id)
    return id ? [{id,name:hit.name,type}] : []
  })
}
export function mergeMarketHits(groups:MarketHit[][]):MarketHit[] {const hits = new Map<number,MarketHit>();for(const group of groups) for(const hit of group) if(!hits.has(hit.id)) hits.set(hit.id,hit);return [...hits.values()].slice(0,12)}
