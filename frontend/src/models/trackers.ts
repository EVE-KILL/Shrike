import type {AccountTrackerCreateData,AccountTrackersResponses,AccountTrackerNotificationsResponses} from '../../shared/api/types.gen'

// The generated Go contract owns the wire shape and its validated enum values.
export type TrackerTargetType=AccountTrackerCreateData['body']['target_type']
export type EntityTracker=AccountTrackersResponses[200]['trackers'][number]
export type TrackerNotification=AccountTrackerNotificationsResponses[200]['notifications'][number]
export function isTrackerTargetType(value:unknown):value is TrackerTargetType {
  return value==='character'||value==='corporation'||value==='alliance'||value==='region'||value==='constellation'||value==='system'
}
