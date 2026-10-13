import {isRecord} from './json'

export const legacySorts = ['id_desc','id_asc','value_desc','value_asc','time_desc','time_asc'] as const
export type LegacySort = typeof legacySorts[number]
export function isLegacySort(value:unknown):value is LegacySort {return legacySorts.some(sort=>sort === value)}
export const legacyPresetShips:Readonly<Record<string,string>> = {
  capitals:'Avatar,Erebus,Ragnarok,Leviathan,Revelation,Moros,Naglfar,Phoenix,Archon,Thanatos,Nidhoggur,Chimera,Aeon,Hel,Nyx,Wyvern',
  battleships:'Raven,Megathron,Apocalypse,Tempest,Dominix,Rokh,Maelstrom,Abaddon,Hyperion,Typhoon,Scorpion,Armageddon',
  pods:'Capsule',
}
export const legacyAutocompleteFields = new Set(['victim','attacker','corp','alliance','system'])
export function readLegacySuggestions(value:unknown):string[] {
  if(!Array.isArray(value)) throw new Error('Suggestions are unavailable. You can still search.')
  return value.flatMap((row:unknown)=>isRecord(row) && typeof row.name === 'string' && row.name.trim() ? [row.name] : [])
}
export function legacyYear(value:unknown):number|undefined {
  if(typeof value !== 'string' || !/^(0|200[3-7])$/.test(value)) return
  return Number(value)
}
