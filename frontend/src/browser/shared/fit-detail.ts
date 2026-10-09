import type {FittingDetailLegacyResponses,FittingRatingPutLegacyResponses,FittingRatingDeleteLegacyResponses} from '../../../shared/api/types.gen'
import {isRecord} from './json'

type Wire = FittingDetailLegacyResponses[200]
export type FitRatingSeed = {id:string;viewer:Wire['viewer_rating'];average:NonNullable<Wire['fit']>['rating_avg'];count:number}
const average = (value:unknown):number|null=>value === null || value === undefined ? null : typeof value === 'number' && Number.isFinite(value) && value >= 0 && value <= 5 ? value : (()=>{throw new Error('Invalid fitting rating')})()
export const isRating = (value:unknown):value is number=>typeof value === 'number' && Number.isInteger(value) && value >= 1 && value <= 5
export function readFitRatingSeed(value:unknown):FitRatingSeed {
  if(!isRecord(value) || !isRecord(value.fit) || typeof value.fit.fit_id !== 'string' || !/^[a-zA-Z0-9_-]{1,128}$/.test(value.fit.fit_id)) throw new Error('Invalid saved fitting')
  const viewer = value.viewer_rating ?? null,count = value.fit.rating_count ?? 0
  if(viewer !== null && !isRating(viewer) || typeof count !== 'number' || !Number.isSafeInteger(count) || count < 0) throw new Error('Invalid fitting rating')
  return {id:value.fit.fit_id,viewer,average:average(value.fit.rating_avg),count}
}
export type RatingResult = FittingRatingPutLegacyResponses[200] | FittingRatingDeleteLegacyResponses[200]
export function readRatingResult(value:unknown,remove:boolean):RatingResult {
  if(!isRecord(value) || !isRecord(value.aggregate) || typeof value.aggregate.rating_count !== 'number' || !Number.isSafeInteger(value.aggregate.rating_count) || value.aggregate.rating_count < 0) throw new Error('Invalid fitting rating response')
  const aggregate = {rating_avg:average(value.aggregate.rating_avg),rating_count:value.aggregate.rating_count}
  if(remove) {if(typeof value.deleted !== 'boolean') throw new Error('Invalid fitting rating response');return {deleted:value.deleted,aggregate}}
  if(!isRating(value.rating)) throw new Error('Invalid fitting rating response')
  return {rating:value.rating,aggregate}
}
