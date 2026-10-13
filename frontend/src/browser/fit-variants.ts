import type {Fit,FitSlotType} from '../fit/types'
import type {FitStats} from '../../dogma/src/types'
import type {FittingsShipFamiliesResponses} from '../../shared/api/types.gen'
import type {FitRenderState,FitRenderAssets,FitIcon,SlotCount} from './shared/fit-render'
type WireFamily = FittingsShipFamiliesResponses[200]['families'][number]
export type SuggestionFamily = Pick<WireFamily,'total_uses'> & {modules:Array<Pick<WireFamily['modules'][number],'type_id'|'slot_group'|'name'>>}
export type SuggestionScore = {typeId:number;name:string;score:number;families:number;cpu:number;power:number;uses:number}
export type SuggestionGroup = {slot:FitSlotType;label:string;suggestions:Array<SuggestionScore & {frequency:number}>}
export type SuggestionsProps = Pick<FitRenderState,'fit'|'sde'> & {data:{families:readonly SuggestionFamily[]}|null;slotCount:SlotCount;cpuFree:number;powerFree:number;typeAttr:(id:number,name:string)=>number;hardpointAvailable:(id:number)=>boolean}
export type ComparisonProps = FitRenderState & Pick<FitRenderAssets,'escape'|'nameOf'> & {customFit:Fit|null;customStats:FitStats|null;customPrice:number|null;currentPrice:number|null;modified:boolean;formatIsk:(value:number)=>string}
/** Client-side parts of FullscreenFitVariants.vue; family data is Go-rendered once. */
const slotGroups:Array<{group:number;slot:FitSlotType;label:string}> = [
  { group: 1, slot: 'High', label: 'High slots' },
  { group: 2, slot: 'Medium', label: 'Mid slots' },
  { group: 3, slot: 'Low', label: 'Low slots' },
  { group: 4, slot: 'Rig', label: 'Rigs' },
  { group: 5, slot: 'SubSystem', label: 'Subsystems' },
];

export function rankFitSuggestions({ fit, data, sde, slotCount, cpuFree, powerFree, typeAttr, hardpointAvailable }:SuggestionsProps):SuggestionGroup[] {
  if (!fit || !data?.families?.length) return [];
  const fitted = new Set(fit.modules.map((module) => module.typeId));
  const observedTotal = data.families.reduce((sum, family) => sum + family.total_uses, 0);
  return slotGroups.flatMap(({ group, slot, label }) => {
    if (fit.modules.filter((module) => module.slot.type === slot).length >= slotCount(slot)) return [];
    const scores = new Map<number,SuggestionScore>();
    for (const family of data.families) {
      const overlap = family.modules.filter((module) => fitted.has(module.type_id)).length;
      if (fitted.size && !overlap) continue;
      const relevance = Math.max(1, overlap);
      const candidates = new Map(family.modules.filter((module) => module.slot_group === group).map((module) => [module.type_id, module]));
      for (const module of candidates.values()) {
        const cpu = typeAttr(module.type_id, 'cpu'), power = typeAttr(module.type_id, 'power');
        if (cpu > cpuFree || power > powerFree) continue;
        if (slot === 'High' && !hardpointAvailable(module.type_id)) continue;
        const current = scores.get(module.type_id) ?? { typeId: module.type_id, name: module.name || sde.types.get(module.type_id)?.name || `Type ${module.type_id}`, score: 0, families: 0, cpu, power, uses: 0 };
        current.score += family.total_uses * relevance;
        current.families++;
        current.uses += family.total_uses;
        scores.set(module.type_id, current);
      }
    }
    const suggestions = [...scores.values()].sort((a, b) => b.score - a.score).slice(0, 2).map((item) => ({ ...item, frequency: observedTotal ? Math.round(item.uses / observedTotal * 100) : 0 }));
    return suggestions.length ? [{ slot, label, suggestions }] : [];
  });
}

export function renderFitSuggestions(groups:readonly SuggestionGroup[], hasModules:boolean, { escape, iconOf, icon }:Pick<FitRenderAssets,'escape'|'iconOf'> & {icon:FitIcon}) {
  if (!groups.length) return '';
  const sections = groups.map((group) => `<div class="fit-editor-suggestion-group"><div class="fit-editor-suggestion-label">${escape(group.label)}</div>${group.suggestions.map((item) => `<button type="button" class="fit-editor-suggestion" data-fit-suggestion="${item.typeId}" data-fit-suggestion-slot="${group.slot}" title="${escape(`${item.name} · seen in ${item.frequency}% of observed fits · ${item.cpu.toFixed(0)} tf · ${item.power.toFixed(0)} MW`)}"><img src="${iconOf(item.typeId)}" width="28" height="28" alt=""><span><strong>${escape(item.name)}</strong><small>${item.frequency}% observed · ${item.cpu.toFixed(0)} tf · ${item.power.toFixed(0)} MW</small></span><em>${icon('plus', 14)}</em></button>`).join('')}</div>`).join('');
  return `<div class="fit-editor-suggestions"><header><span>✧</span> Suggestions</header>${sections}<p>${hasModules ? 'Ranked by observed fits sharing your current modules.' : 'Ranked by the most common observed fits for this hull.'} Only options fitting open slots, hardpoints, CPU and powergrid are shown. Hover to preview.</p></div>`;
}

export function fitModuleDifference(customFit:Fit|null, observedFit:Fit|null, nameOf:FitRenderAssets['nameOf']) {
  const counts = (fit:Fit|null) => {
    const result = new Map<number,number>();
    for (const module of fit?.modules ?? []) result.set(module.typeId, (result.get(module.typeId) ?? 0) + 1);
    return result;
  };
  const custom = counts(customFit), observed = counts(observedFit);
  let added = 0, removed = 0;
  const addedNames:string[] = [], removedNames:string[] = [];
  for (const [id, count] of observed) { const difference = Math.max(0, count - (custom.get(id) ?? 0)); added += difference; if (difference) addedNames.push(`${difference}× ${nameOf(id)}`); }
  for (const [id, count] of custom) { const difference = Math.max(0, count - (observed.get(id) ?? 0)); removed += difference; if (difference) removedNames.push(`${difference}× ${nameOf(id)}`); }
  return { added, removed, addedNames, removedNames };
}

export function renderFitComparison({ customFit, fit, customStats, stats, customPrice, currentPrice, sde, modified, escape, nameOf, formatIsk }:ComparisonProps) {
  if (!customFit || !fit) return '';
  const modules = fitModuleDifference(customFit, fit, nameOf);
  const hullStat = (source:FitStats|null, name:string) => {
    const id = sde.attributeNameToId.get(name);
    const attr = id === undefined ? null : source?.hull.attributes.get(id);
    return Number(attr?.value ?? attr?.base_value ?? 0);
  };
  const metrics = customStats && stats ? [['DPS', 'damagePerSecondWithReload', ''], ['EHP', 'ehp', ''], ['Speed', 'maxVelocity', 'm/s'], ['CPU free', 'cpuFree', 'tf'], ['PG free', 'powerFree', 'MW']].map(([label, name, unit]) => {
    const custom = hullStat(customStats, name), current = hullStat(stats, name), delta = current - custom;
    return `<div class="fit-editor-diff-row"><span>${label}</span><span>${custom.toFixed(1)}</span><strong class="${delta > 0 ? 'positive' : delta < 0 ? 'negative' : ''}">${delta > 0 ? '+' : ''}${delta.toFixed(1)} ${unit}</strong></div>`;
  }).join('') : '';
  const priceDelta = (currentPrice ?? 0) - (customPrice ?? 0);
  const price = `<div class="fit-editor-diff-row"><span>Price</span><span>${formatIsk(customPrice ?? 0)}</span><strong class="${priceDelta > 0 ? 'negative' : priceDelta < 0 ? 'positive' : ''}">${priceDelta > 0 ? '+' : ''}${formatIsk(priceDelta)}</strong></div>`;
  const names = `<div class="fit-editor-diff-names">${modules.addedNames.length ? `<div class="positive" title="${escape(modules.addedNames.join(', '))}">+ ${escape(modules.addedNames.slice(0, 3).join(', '))}</div>` : ''}${modules.removedNames.length ? `<div class="negative" title="${escape(modules.removedNames.join(', '))}">− ${escape(modules.removedNames.slice(0, 3).join(', '))}</div>` : ''}</div>`;
  const changed = modified ? `<div class="fit-editor-diff-modified"><div>This observed fit has been modified.</div><button type="button" data-fit-variant-action="keep-custom">Keep as custom</button><button type="button" data-fit-variant-action="reset-preview">Reset preview</button></div>` : '';
  return `<div class="fit-editor-comparison"><header><strong>Observed vs custom</strong><span>+${modules.added} / −${modules.removed} modules</span></header>${metrics}${price}${names}${changed}</div>`;
}
