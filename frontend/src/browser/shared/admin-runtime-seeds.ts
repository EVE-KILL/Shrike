import {isRecord} from './json'
import type {RiverJob,PrizeSettlementRow} from '../admin-data'

const text=(value:unknown)=>typeof value==='string'?value:''
const nullable=(value:unknown)=>typeof value==='string'?value:null
const number=(value:unknown)=>typeof value==='number'&&Number.isFinite(value)?value:0
function rows(source:string|undefined,key:string):unknown[]{try{const value:unknown=JSON.parse(source??'null');return isRecord(value)&&Array.isArray(value[key])?value[key]:[]}catch{return []}}
export function readRiverQueues(source:string|undefined){return rows(source,'queues').flatMap(row=>isRecord(row)&&typeof row.name==='string'?[{name:row.name}]:[])}
export function readRiverJobs(source:string|undefined):RiverJob[]{
  return rows(source,'jobs').flatMap(row=>{
    if(!isRecord(row)||typeof row.id!=='number'||!Number.isSafeInteger(row.id)||row.id<=0)return []
    return [{id:row.id,queue:text(row.queue),kind:text(row.kind),state:text(row.state),attempt:number(row.attempt),max_attempts:number(row.max_attempts),priority:number(row.priority),created_at:text(row.created_at),scheduled_at:text(row.scheduled_at),attempted_at:nullable(row.attempted_at),finalized_at:nullable(row.finalized_at),attempted_by:Array.isArray(row.attempted_by)?row.attempted_by.filter((value):value is string=>typeof value==='string'):[],args:row.args,output:row.output,metadata:row.metadata,errors:Array.isArray(row.errors)?row.errors.filter(isRecord):[]}]
  })
}
export function readWalletSettlements(source:string|undefined):PrizeSettlementRow[]{
  return rows(source,'prizeSettlements').flatMap(row=>{
    if(!isRecord(row)||typeof row.campaign_id!=='string')return []
    return [{campaign_id:row.campaign_id,campaign_name:text(row.campaign_name),pool_status:number(row.pool_status),funded_total:text(row.funded_total),finalized_at:nullable(row.finalized_at),rank:typeof row.rank==='number'?row.rank:null,character_id:typeof row.character_id==='number'?row.character_id:null,character_name:nullable(row.character_name),metric_value:nullable(row.metric_value),payout_percentage:typeof row.payout_percentage==='number'?row.payout_percentage:null,payout_amount:nullable(row.payout_amount),claimed_at:nullable(row.claimed_at),paid_at:nullable(row.paid_at),payment_note:nullable(row.payment_note)}]
  })
}
