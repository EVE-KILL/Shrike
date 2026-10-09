// The item and fitting search interfaces accept the same shorthand values.
export const fitMetrics = ['ehp','dps','alpha','speed','align','repair','shield_repair','armor_repair','hull_repair','passive_shield','npc_ehp'] as const
export type FitMetric = typeof fitMetrics[number]
export function isFitMetric(value:unknown):value is FitMetric {return fitMetrics.some(metric=>metric === value)}
export function parseFitNumber(raw:string):number|null {
  const match = raw.trim().toLowerCase().replaceAll(',','').match(/^([0-9]*\.?[0-9]+)\s*([kmb])?$/)
  if(!match) return null
  const suffix = match[2],value = Number(match[1])*(suffix === 'k' ? 1e3 : suffix === 'm' ? 1e6 : suffix === 'b' ? 1e9 : 1)
  return Number.isFinite(value) ? value : null
}
export function sortForFitMetric(metric:FitMetric):string {return ['shield_repair','armor_repair','hull_repair','passive_shield'].includes(metric) ? 'repair' : metric}
export function fitFilterKey(params:URLSearchParams):string {const copy = new URLSearchParams(params);copy.delete('sort');copy.sort();return copy.toString()}
export function itemFitsKey(params:URLSearchParams):string {const copy = new URLSearchParams(params);if(copy.get('sort') === 'observed') copy.delete('sort');copy.sort();return copy.toString()}
