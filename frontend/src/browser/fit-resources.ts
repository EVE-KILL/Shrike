import type {Fit,FitDrone,FitCargo} from '../fit/types'
import type {SdeData} from '../../dogma/src/metadata'
import type {BayKind,FitRenderAssets,FitIcon,FitFormatter} from './shared/fit-render'

export type FitResourceField = {label:string;used:number;total:number;unit:string;usedDigits?:number;totalDigits?:number;bay?:BayKind}
export type ResourceBarProps = {fields:readonly FitResourceField[];openBay:BayKind|null;fmt:FitFormatter;icon:FitIcon}
export type BayPanelProps = FitRenderAssets & {kind:BayKind;fit:Fit;sde:SdeData;typeAttr:(id:number,name:string)=>number;fmt:FitFormatter;icon:FitIcon}
/** Markup port of ResourceBar.vue, DroneBay.vue, and CargoBay.vue. */
export function renderFitResourceBar({ fields, openBay, fmt, icon }:ResourceBarProps) {
  return fields.map(({ label, used, total, unit, usedDigits, totalDigits, bay }) => {
    const percent = total > 0 ? Math.max(0, used / total * 100) : 0;
    const tone = percent > 100 ? 'danger' : percent > 90 ? 'warning' : 'normal';
    const tag = bay ? 'button' : 'div';
    const active = bay && openBay === bay ? ' open' : '';
    return `<${tag} ${bay ? `type="button" data-fit-bay="${bay}"` : ''} class="fit-editor-resource-cell ${tone}${active}">
      <span class="fit-editor-resource-heading"><span>${bay ? icon(bay === 'drones' ? 'radar' : 'package', 12) : ''}${label}</span>${bay ? icon(active ? 'chevron-up' : 'chevron-down', 12) : ''}</span>
      <span class="fit-editor-resource-values"><strong>${fmt(used, usedDigits)}</strong><em>/</em><span>${fmt(total, totalDigits)}</span><small>${unit}</small></span>
      <span class="fit-editor-resource-track"><i style="width:${Math.min(100, percent)}%"></i></span>
    </${tag}>`;
  }).join('');
}

export function renderFitBayPanel({ kind, fit, sde, iconOf, nameOf, typeAttr, escape, fmt, icon }:BayPanelProps) {
  const drones = kind === 'drones';
  const active = fit.drones.reduce((sum, entry) => sum + entry.states.Active, 0);
  const heading = `<header><strong>${icon(drones ? 'radar' : 'package', 14)}${drones ? 'Drone Bay' : 'Cargo Hold'}</strong><span>${drones ? `<span class="fit-editor-bay-active-count${active >= 5 ? ' full' : ''}">${icon('zap', 12)} ${active} / <em>5</em> <small>in space</small></span><i></i>` : ''}<small>${icon('mouse-pointer-click', 12)}Drag ${drones ? '' : 'items '}from sidebar</small></span></header>`;
  const items = drones ? fit.drones : fit.cargo;
  const content = items.length ? items.map((item:FitDrone|FitCargo) => {
    const type = sde.types.get(item.typeId);
    const name = escape(nameOf(item.typeId));
    const volume = type?.volume ?? 0;
    const details = drones ? `<span>${volume}m³</span><span>·</span><span>${typeAttr(item.typeId, 'droneBandwidthUsed')}Mbit</span>` : `<span>${volume}m³ each</span>`;
    const controls = 'states' in item
      ? `<div class="fit-editor-bay-drone-states"><button type="button" data-fit-drone-state="deactivate:${item.typeId}" class="${item.states.Active ? 'active' : ''}" title="In space — click to recall one to the bay" ${item.states.Active ? '' : 'disabled'}>${icon('zap', 12)} ${item.states.Active}</button><button type="button" data-fit-drone-state="activate:${item.typeId}" title="In bay — click to launch one into space" ${item.states.Passive ? '' : 'disabled'}>${icon('archive', 12)} ${item.states.Passive}</button></div><button type="button" class="fit-editor-bay-control" data-fit-bay-step="-1:${item.typeId}" title="Remove one drone">${icon('minus', 12)}</button>`
      : `<div class="fit-editor-bay-stepper"><button type="button" data-fit-bay-step="-1:${item.typeId}" title="Remove one">${icon('minus', 12)}</button><span>${fmt(item.quantity)}</span><button type="button" data-fit-bay-step="1:${item.typeId}" title="Add one">${icon('plus', 12)}</button></div><small class="fit-editor-bay-total">${fmt(volume * item.quantity, 1)}m³</small>`;
    return `<div class="fit-editor-bay-item"><img src="${iconOf(item.typeId)}" width="32" height="32" alt="${name}" loading="lazy"><span class="fit-editor-bay-name"><a href="/item/${item.typeId}">${name}</a><small>${details}</small></span>${controls}<button type="button" class="fit-editor-bay-control fit-editor-bay-remove" data-fit-remove-bay="${item.typeId}" title="Remove all">${icon('x', 12)}</button></div>`;
  }).join('') : `<div class="fit-editor-bay-empty">${icon('package-plus', 32)}<span>${drones ? 'Drone bay' : 'Cargo hold'} is empty</span><small>Drag ${drones ? 'drones here from the Modules' : 'any item here from the'} sidebar</small></div>`;
  return `<div class="fit-editor-bay-panel" data-fit-bay-panel="${kind}">${heading}<div class="fit-editor-bay-content">${content}</div></div>`;
}
