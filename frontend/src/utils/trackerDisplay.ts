import type {EntityTracker,TrackerTargetType} from '../models/trackers'

export function trackerStreamTopics(trackers: EntityTracker[]): string[] {
    return [...new Set(trackers.filter(t=>t.enabled).flatMap(t=>['character','corporation','alliance'].includes(t.target_type)?[`victim.${t.target_id}`,`attacker.${t.target_id}`]:[`${t.target_type}.${t.target_id}`]))].sort()
}
export function trackerTargetIcon(type:TrackerTargetType):string {
    if(type==='character')return 'lucide:user'
    if(type==='corporation')return 'lucide:building-2'
    if(type==='alliance')return 'lucide:flag'
    if(type==='region')return 'lucide:map'
    if(type==='constellation')return 'lucide:orbit'
    return 'lucide:map-pin'
}
export function trackerTargetImage(tracker:EntityTracker):string|null {
    if(tracker.target_type==='character')return `https://images.evetech.net/characters/${tracker.target_id}/portrait?size=64`
    if(tracker.target_type==='corporation')return `https://images.evetech.net/corporations/${tracker.target_id}/logo?size=64`
    if(tracker.target_type==='alliance')return `https://images.evetech.net/alliances/${tracker.target_id}/logo?size=64`
    return null
}
export function trackerActivityTime(value:string|null):string {return value?new Date(value).toLocaleString():'No activity yet'}
