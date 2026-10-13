import type {DscanAnalyzeResponses,LocalscanAnalyzeResponses,DscanSaveData,LocalscanSaveData} from '../../../shared/api/types.gen'
import {isRecord} from './json'

export type ScanKind = 'dscan' | 'localscan'
type DscanResult = DscanAnalyzeResponses[200]
type LocalResult = LocalscanAnalyzeResponses[200]
export type ScanAnalysis =
  | {kind:'dscan'; input:string; result:DscanResult}
  | {kind:'localscan'; input:string[]; result:LocalResult}
const count = (value:unknown):value is number => typeof value === 'number' && Number.isSafeInteger(value) && value >= 0
const nullableID = (value:unknown):value is number|null => value === null || count(value)
function dictionary<T>(value:unknown,valid:(item:unknown)=>item is T):value is Record<string,T> {
  return isRecord(value) && Object.values(value).every(valid)
}
type DscanCategory = DscanResult['grouped'][string]
type DscanGroup = DscanCategory['groups'][string]
type DscanType = DscanGroup['types'][number]
function dscanType(value:unknown):value is DscanType {
  return isRecord(value) && typeof value.typeName === 'string' && nullableID(value.typeId) && count(value.count)
}
function dscanGroup(value:unknown):value is DscanGroup {
  return isRecord(value) && nullableID(value.groupId) && Array.isArray(value.types) && value.types.every(dscanType)
}
function dscanCategory(value:unknown):value is DscanCategory {
  return isRecord(value) && nullableID(value.categoryId) && dictionary(value.groups,dscanGroup)
}
function dscanResult(value:unknown):value is DscanResult {
  return isRecord(value) && dictionary(value.grouped,dscanCategory) && count(value.totalCount) && count(value.uniqueTypes)
}
type LocalCorp = LocalResult['corporations'][string]
type LocalCharacter = LocalCorp['characters'][number]
type LocalAlliance = LocalResult['alliances'][string]
function localCharacter(value:unknown):value is LocalCharacter {
  return isRecord(value) && count(value.characterId) && typeof value.name === 'string' && count(value.kills)
}
function localCorp(value:unknown):value is LocalCorp {
  return isRecord(value) && typeof value.name === 'string' && typeof value.ticker === 'string' && Array.isArray(value.characters) && value.characters.every(localCharacter)
}
function localAlliance(value:unknown):value is LocalAlliance {
  return isRecord(value) && typeof value.name === 'string' && typeof value.ticker === 'string' && dictionary(value.corporations,localCorp)
}
function localResult(value:unknown):value is LocalResult {
  return isRecord(value) && dictionary(value.alliances,localAlliance) && dictionary(value.corporations,localCorp) && count(value.totalCharacters) && count(value.totalDangerous) && Array.isArray(value.unresolved) && value.unresolved.every(name=>typeof name === 'string')
}
export function isScanKind(value:unknown):value is ScanKind {return value === 'dscan' || value === 'localscan'}
export function localScanNames(text:string):string[] {return text.split('\n').map(name=>name.trim()).filter(name=>name.length > 0 && name.length <= 64)}
export function readScanAnalysis(kind:ScanKind,text:string,value:unknown):ScanAnalysis {
  if(kind === 'dscan' && dscanResult(value)) return {kind,input:text,result:value}
  if(kind === 'localscan' && localResult(value)) return {kind,input:localScanNames(text),result:value}
  throw new Error('Could not read scan results. Please try again.')
}
export function scanSaveBody(scan:ScanAnalysis):DscanSaveData['body']|LocalscanSaveData['body'] {
  return scan.kind === 'dscan' ? {dscan:scan.input,result:scan.result} : {names:scan.input,result:scan.result}
}
export function readScanHash(value:unknown):string {
  if(isRecord(value) && typeof value.hash === 'string' && /^[0-9a-f]{64}$/.test(value.hash)) return value.hash
  throw new Error('Failed to save scan. Please try again.')
}
