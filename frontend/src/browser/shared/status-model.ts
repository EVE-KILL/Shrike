import {isRecord} from './json'

type Metrics<K extends string> = Partial<Record<K,number>>
type Labels<K extends string> = Partial<Record<K,string>>
export type StatusHistory = Metrics<'queries_per_second'|'transactions_per_second'|'read_bytes_per_second'|'write_bytes_per_second'|'wal_bytes_per_second'>
type DatabaseTable = Labels<'data_size'|'total_size'> & Metrics<'rows'>
type CloudflarePeriod = Metrics<'requests'|'bandwidth_gb'|'bandwidth_cached_gb'|'cache_hit_rate'|'threats_blocked'|'page_views'|'unique_visitors'> & {status_codes?:Record<string,number>}
type FetchPeriod = Metrics<'total'|'failed'|'killmails_found'|'new_killmails'>
export interface StatusSnapshot {
  system?:Labels<'platform'|'arch'|'memory_total'|'memory_free'> & Metrics<'cpus'> & {load_average:number[]}
  esi?:Labels<'tq_status'> & Metrics<'tq_players'|'error_budget'> & {paused:boolean}
  wallet?:{total_balance:string|number|undefined}
  redis?:Labels<'used_memory'> & Metrics<'keyspace_hits'|'keyspace_misses'|'connected_clients'>
  queues:Record<string,Metrics<'waiting'|'active'|'failed'>>
  database?:Labels<'size'|'version'|'role'> & Metrics<'cache_hit_ratio'|'waiting_locks'|'uptime_seconds'> & {
    connections?:Metrics<'total'|'max'|'active'|'waiting'|'idle_in_transaction'>
    cluster?:Metrics<'replicas'|'streaming'|'max_lag_seconds'|'max_lag_bytes'>
    workload?:Metrics<'transactions_per_second'|'rollback_percent'|'read_bytes_per_second'|'write_bytes_per_second'|'wal_bytes_per_second'|'read_latency_ms'|'write_latency_ms'|'rows_changed_per_second'|'deadlocks'|'temp_bytes_per_second'>
    statements?:Metrics<'queries_per_second'|'average_latency_ms'>
    maintenance?:Metrics<'checkpoints'|'requested_checkpoints'|'average_checkpoint_time_ms'|'dead_rows'|'active_autovacuums'>
    history:StatusHistory[]
    tables:Record<string,DatabaseTable>
  }
  zkb_ingest?:Metrics<'window_hours'> & {new_hourly:number[];repost_hourly:number[]}
  coverage?:Metrics<'window_days'> & {characters?:Metrics<'covered'|'active'>;corporations?:Metrics<'covered'|'active'>;alliances?:Metrics<'active'|'corps_covered'|'corps_active'>}
  esi_tokens?:{tokens?:Metrics<'active'|'total'|'revoked'|'can_fetch_character'|'can_fetch_corporation'>;fetches?:{last_24h?:FetchPeriod;all_time?:FetchPeriod}}
  cloudflare?:Labels<'last_updated'> & {last_24h?:CloudflarePeriod;last_1h?:CloudflarePeriod}
  image_server?:{folderStats:Record<string,Metrics<'fileCount'|'sizeKB'|'metadataFileCount'>>;cacheValidation?:Metrics<'totalValidationRuns'|'totalImagesValidated'|'totalImagesRemoved'|'validationErrors'>}
  websocket?:Labels<'status'> & Metrics<'uptime'|'connections'> & {redis:boolean;channels:string[];endpoints:Record<string,Metrics<'connections'> & {subscriptions:Record<string,number>}>}
}
function numeric(value:unknown):number|undefined {
  if(typeof value !== 'number' && (typeof value !== 'string' || !value.trim())) return
  const number = Number(value);if(Number.isFinite(number)) return number
}
export function statusNumber(value:unknown):number {return numeric(value) ?? 0}
function metrics<K extends string>(value:unknown,keys:readonly K[]):Metrics<K>|undefined {
  if(!isRecord(value)) return
  const result:Metrics<K> = {}
  for(const key of keys) {const number = numeric(value[key]);if(number !== undefined) result[key] = Math.max(0,number)}
  return result
}
function labels<K extends string>(value:unknown,keys:readonly K[]):Labels<K> {
  const result:Labels<K> = {}
  if(isRecord(value)) for(const key of keys) if(typeof value[key] === 'string') result[key] = value[key]
  return result
}
function dictionary<T>(value:unknown,read:(value:Record<string,unknown>)=>T):Record<string,T> {
  const rows:Array<[string,T]> = []
  if(isRecord(value)) for(const [key,item] of Object.entries(value)) if(isRecord(item)) rows.push([key,read(item)])
  return Object.fromEntries(rows)
}
function numbers(value:unknown):number[] {return Array.isArray(value) ? value.map(item=>Math.max(0,statusNumber(item))) : []}
function counts(value:unknown):Record<string,number> {return isRecord(value) ? Object.fromEntries(Object.entries(value).map(([key,item])=>[key,Math.max(0,statusNumber(item))])) : {}}
function cloudflarePeriod(value:unknown):CloudflarePeriod|undefined {
  if(!isRecord(value)) return
  return {...metrics(value,['requests','bandwidth_gb','bandwidth_cached_gb','cache_hit_rate','threats_blocked','page_views','unique_visitors']),status_codes:counts(value.status_codes)}
}
function fetchPeriod(value:unknown) {return metrics(value,['total','failed','killmails_found','new_killmails'])}
export function readStatusSnapshot(value:unknown):StatusSnapshot|undefined {
  if(!isRecord(value) || ![value.system,value.database,value.queues,value.esi].some(isRecord)) return
  const data:StatusSnapshot = {queues:dictionary(value.queues,row=>metrics(row,['waiting','active','failed']) ?? {})}
  if(isRecord(value.system)) data.system = {...labels(value.system,['platform','arch','memory_total','memory_free']),...metrics(value.system,['cpus']),load_average:numbers(value.system.load_average)}
  if(isRecord(value.esi)) data.esi = {...labels(value.esi,['tq_status']),...metrics(value.esi,['tq_players','error_budget']),paused:value.esi.paused === true}
  if(isRecord(value.redis)) data.redis = {...labels(value.redis,['used_memory']),...metrics(value.redis,['keyspace_hits','keyspace_misses','connected_clients'])}
  if(isRecord(value.wallet)) {const balance = value.wallet.total_balance;data.wallet = {total_balance:typeof balance === 'string' || typeof balance === 'number' && Number.isFinite(balance) ? balance : undefined}}
  if(isRecord(value.database)) {
    const db = value.database
    data.database = {...labels(db,['size','version','role']),...metrics(db,['cache_hit_ratio','waiting_locks','uptime_seconds']),
      connections:metrics(db.connections,['total','max','active','waiting','idle_in_transaction']),cluster:metrics(db.cluster,['replicas','streaming','max_lag_seconds','max_lag_bytes']),
      workload:metrics(db.workload,['transactions_per_second','rollback_percent','read_bytes_per_second','write_bytes_per_second','wal_bytes_per_second','read_latency_ms','write_latency_ms','rows_changed_per_second','deadlocks','temp_bytes_per_second']),
      statements:metrics(db.statements,['queries_per_second','average_latency_ms']),maintenance:metrics(db.maintenance,['checkpoints','requested_checkpoints','average_checkpoint_time_ms','dead_rows','active_autovacuums']),
      history:Array.isArray(db.history) ? db.history.flatMap(row=>{const point = metrics(row,['queries_per_second','transactions_per_second','read_bytes_per_second','write_bytes_per_second','wal_bytes_per_second']);return point ? [point] : []}) : [],
      tables:dictionary(db.tables,row=>({...labels(row,['data_size','total_size']),...metrics(row,['rows'])}))}
  }
  if(isRecord(value.zkb_ingest)) data.zkb_ingest = {...metrics(value.zkb_ingest,['window_hours']),new_hourly:numbers(value.zkb_ingest.new_hourly),repost_hourly:numbers(value.zkb_ingest.repost_hourly)}
  if(isRecord(value.coverage)) {const cov = value.coverage;data.coverage = {...metrics(cov,['window_days']),characters:metrics(cov.characters,['covered','active']),corporations:metrics(cov.corporations,['covered','active']),alliances:metrics(cov.alliances,['active','corps_covered','corps_active'])}}
  if(isRecord(value.esi_tokens)) {const tokens = value.esi_tokens;data.esi_tokens = {tokens:metrics(tokens.tokens,['active','total','revoked','can_fetch_character','can_fetch_corporation']),fetches:isRecord(tokens.fetches) ? {last_24h:fetchPeriod(tokens.fetches.last_24h),all_time:fetchPeriod(tokens.fetches.all_time)} : undefined}}
  if(isRecord(value.cloudflare)) data.cloudflare = {...labels(value.cloudflare,['last_updated']),last_24h:cloudflarePeriod(value.cloudflare.last_24h),last_1h:cloudflarePeriod(value.cloudflare.last_1h)}
  if(isRecord(value.image_server)) data.image_server = {folderStats:dictionary(value.image_server.folderStats,row=>metrics(row,['fileCount','sizeKB','metadataFileCount']) ?? {}),cacheValidation:metrics(value.image_server.cacheValidation,['totalValidationRuns','totalImagesValidated','totalImagesRemoved','validationErrors'])}
  if(isRecord(value.websocket)) {const ws = value.websocket;data.websocket = {...labels(ws,['status']),...metrics(ws,['uptime','connections']),redis:ws.redis === true,channels:Array.isArray(ws.channels) ? ws.channels.filter((value):value is string=>typeof value === 'string') : [],endpoints:dictionary(ws.endpoints,row=>({...metrics(row,['connections']),subscriptions:counts(row.subscriptions)}))}}
  return data
}
