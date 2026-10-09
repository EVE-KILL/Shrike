import type {FitModule,FitSlot,FitSlotType} from '../fit/types'
import type {ComputedItem} from '../../dogma/src/types'
import type {FitRenderState,FitRenderAssets,FitIcon,SlotCount} from './shared/fit-render'
type WheelAssets = FitRenderAssets & {uiIcon:FitIcon;selected:FitSlot|null}
export type WheelProps = FitRenderState & WheelAssets & {slotCount:SlotCount;value:(name:string)=>number}
/*
 * Direct DOM port of web/app/components/fit/ring/{ShipFit,RingOuter,RingInner,
 * RingTopItem,RadialMenu,Usage,Slot,Hull,FitLink}.vue. Angles, viewBoxes,
 * paths, sizes and CSS class names follow the existing Nuxt implementation.
 * The fitting state and event handlers live in fit-editor.js.
 */

const slotRotations = {
  High: [-36.5, 71 / 7, 8],
  Medium: [53, 72 / 7, 8],
  Low: [142, 72 / 7, 8],
  Rig: [-74, 21 / 2, 3],
  SubSystem: [-128, 38 / 3, 4],
} as const satisfies Record<FitSlotType,readonly [number,number,number]>;
const placeholder:Partial<Record<FitSlotType,string>> = { High: 'fitting-hislot', Medium: 'fitting-medslot', Low: 'fitting-lowslot', Rig: 'fitting-rig-subsystem' };

const topItem = (rotation:number, contents:string, background = false) => `<div class="ringTopItem${background ? ' background' : ''}" style="--rotation:${rotation}deg">${contents}</div>`;

function outerRing() {
  return `<svg viewBox="24 24 464 464" xmlns="http://www.w3.org/2000/svg" class="ringOuter" aria-hidden="true"><g><circle style="fill:none;stroke:#000000;stroke-width:16" cx="256" cy="256" r="224"/><rect style="fill:#000000" width="17" height="17" x="98" y="89"/><rect style="fill:#000000" width="17" height="17" x="401" y="93"/><rect style="fill:#000000" width="17" height="17" x="402" y="401"/><rect style="fill:#000000" width="17" height="17" x="94" y="402"/><rect style="fill:#000000" width="12" height="12" x="196" y="82" transform="rotate(56)"/></g></svg>`;
}

function innerRing() {
  return `<svg viewBox="24 24 464 464" xmlns="http://www.w3.org/2000/svg" class="ringInner" aria-hidden="true"><defs><mask id="slot-corners"><rect fill="#ffffff" width="512" height="512" x="0" y="0"/><rect fill="#000000" width="17" height="17" x="133" y="126"/><rect fill="#000000" width="17" height="17" x="366" y="129"/><rect fill="#000000" width="17" height="17" x="366" y="366"/><rect fill="#000000" width="17" height="17" x="132" y="369"/><rect fill="#000000" width="12" height="12" x="230" y="44" transform="rotate(56)"/></mask></defs><g><circle style="fill:none;stroke:#000000;stroke-width:46;stroke-opacity:.6" cx="256" cy="256" r="195" mask="url(#slot-corners)"/></g></svg>`;
}

function slotDefs() {
  return `<svg viewBox="235 40 52 50" xmlns="http://www.w3.org/2000/svg" preserveAspectRatio="xMidYMin slice" style="display:none"><g id="slot"><path style="fill-opacity:.1;stroke-width:1;stroke-opacity:.5" d="M 243 46 A 210 210 0 0 1 279 46.7 L 276 84.7 A 172 172 0 0 0 246 84 L 243 46"/></g><g id="slot-active"><path style="fill-opacity:.6;stroke-width:1" d="M 250 84 L 254 79 L 268 79 L 272 84"/></g><g id="slot-passive"><path style="stroke-width:1" d="M 245 48 A 208 208 0 0 1 250 47.5 L 248 50 L 245 50"/><path style="stroke-width:1" d="M 277.5 48.5 A 208 208 0 0 0 273 48 L 275 50.5 L 277.5 50.5"/><path style="stroke-width:1" d="M 247 82 A 170 170 0 0 1 252 82 L 250 80 L 246.8 80"/><path style="stroke-width:1" d="M 275 82.5 A 170 170 0 0 0 270 82 L 272 80 L 275.2 80"/></g></svg>`;
}

function radialMenu(type:'High'|'Medium'|'Low') {
  const highlight = {
    Low: [12, 3, 0, 9], Medium: [3, 12, 9, 0], High: [12, 3, 0, 0],
  }[type];
  const [width, height, x, y] = highlight;
  return `<svg viewBox="0 0 12 12" xmlns="http://www.w3.org/2000/svg" class="radialMenu" aria-hidden="true"><defs><mask id="radial-menu-${type}"><rect fill="#ffffff" fill-opacity=".5" width="12" height="12" x="0" y="0"/><rect fill="#ffffff" width="${width}" height="${height}" x="${x}" y="${y}"/><circle fill="#000000" cx="6" cy="6" r="5"/><rect fill="#000000" width="3" height="3" x="0" y="0"/><rect fill="#000000" width="4" height="3" x="9" y="0"/><rect fill="#000000" width="3" height="4" x="0" y="9"/><rect fill="#000000" width="4" height="4" x="9" y="9"/></mask></defs><g><rect fill="#ffffff" width="12" height="12" x="0" y="0" mask="url(#radial-menu-${type})"/></g></svg>`;
}

function position(angle:number, radius:number) {
  const radians = angle * Math.PI / 180;
  return { x: radius * Math.cos(radians) + 256, y: radius * Math.sin(radians) + 256 };
}

function usageSVG(angle:number, intervals:number, markers:number, color:string, fraction:number) {
  const ticks = Array.from({ length: intervals }, (_, index) => {
    const a = index * angle / intervals - 90;
    const inner = position(a, 217), outer = position(a, 223);
    return `<path fill="none" stroke="#7a7a7a" stroke-width="1" d="M${inner.x} ${inner.y} L${outer.x} ${outer.y}"/>`;
  }).join('');
  const longTicks = Array.from({ length: markers }, (_, index) => {
    const a = index * angle / (markers - 1) - 90;
    const inner = position(a, 217), outer = position(a, 230);
    return `<path fill="none" stroke="#7a7a7a" stroke-width="1" d="M${inner.x} ${inner.y} L${outer.x} ${outer.y}"/>`;
  }).join('');
  const start = position(-90, 222), end = position(-90 + angle * Math.max(0, Math.min(1, fraction)), 222);
  const arc = fraction > 0 ? `<path fill="none" stroke="${color}" stroke-width="10" d="M${start.x} ${start.y} A222 222 0 0 ${angle >= 0 ? 1 : 0} ${end.x} ${end.y}"/>` : '';
  return `<svg viewBox="24 24 464 464" xmlns="http://www.w3.org/2000/svg" class="ringOuter" aria-hidden="true"><g>${arc}${longTicks}${ticks}</g></svg>`;
}

function slotMarkup(type:FitSlotType, index:number, available:boolean, module:FitModule|undefined, computed:ComputedItem|undefined, assets:WheelAssets) {
  const { iconOf, uiIcon, escape, selected } = assets;
  const active = module && !['Passive', 'Online'].includes(computed?.max_state ?? 'Active');
  const state = computed?.state ?? module?.state;
  const display = !module ? available ? '' : 'Unavailable' : state === 'Passive' && computed?.max_state !== 'Passive' ? 'Offline' : state;
  const iconID = module?.charge?.typeId ?? module?.typeId ?? 0;
  const svg = `<svg viewBox="235 40 52 50" xmlns="http://www.w3.org/2000/svg" class="ringInner" preserveAspectRatio="xMidYMin slice" aria-hidden="true"><use href="#slot"/>${module && available ? `<use href="#slot-${active ? 'active' : 'passive'}"/>` : ''}</svg>`;
  const image = module ? `<div class="slotImage"><img src="${iconOf(iconID, 64)}" class="${module.state === 'Preview' ? 'preview' : ''}" alt="" width="50" height="50" draggable="true"/></div>` : available && placeholder[type] ? `<div class="slotImagePlaceholder"><img src="/assets/fit-ui/${placeholder[type]}.webp" alt=""/></div>` : '';
  return `<div class="slotOuter${selected?.type === type && selected?.index === index ? ' selected' : ''}" data-fit-slot="${type}:${index}" data-hasitem="${!!module}" role="button" tabindex="${available ? 0 : -1}" aria-label="${escape(type)} ${index}: ${escape(module ? assets.nameOf(module.typeId) : available ? 'Empty' : 'Unavailable')}"><div class="slot" data-state="${display}">${svg}${image}</div></div>`;
}

export function renderWheel({ fit, stats, sde, slotCount, value, iconOf, uiIcon, escape, nameOf, selected }:WheelProps) {
  const assets = { iconOf, uiIcon, escape, nameOf, selected };
  const hull = fit ? `<div class="hull"><img src="https://images.evetech.net/types/${fit.shipTypeId}/render?size=1024" alt="${escape(fit.name)}" width="564" height="564"/></div>` : '';
  const link = `<div class="fitLink"><svg viewBox="0 0 730 730" xmlns="http://www.w3.org/2000/svg"><path id="fitlink" fill="none" d="M18,365 A25,25 0 0,1 712,365"/><a href="/fits/create" data-fit-action="share"><text text-anchor="middle"><textPath start-offset="50%" href="#fitlink" fill="#cdcdcd">share fit</textPath></text></a></svg></div>`;
  const itemValue = (item:ComputedItem|null|undefined, name:string) => {
    const id = sde?.attributeNameToId.get(name);
    const attr = id === undefined ? null : item?.attributes?.get(id);
    return attr?.value ?? attr?.base_value ?? 0;
  };
  const effectIDs = new Map([...sde.dogmaEffects].map(([id, effect]) => [effect.name, id]));
  const hardpoint = (effect:string) => {
    const id = effectIDs.get(effect);
    return id == null ? 0 : stats?.items.filter((item) => sde.typeDogma.get(item.type_id)?.dogmaEffects?.some((entry) => entry.effectID === id)).length ?? 0;
  };
  const turretTotal = Math.floor(value('turretSlotsLeft'));
  const launcherTotal = Math.floor(value('launcherSlotsLeft'));
  const turretUsed = hardpoint('turretFitted'), launcherUsed = hardpoint('launcherFitted');
  const hardpoints = topItem(-45, `<div class="turretLauncherIcon">${uiIcon('hardpoint-turret', 16)}</div>`, true)
    + Array.from({ length: Math.max(0, turretTotal) }, (_, index) => topItem(-40 + index * 3, `<div class="turretLauncherItem${index < turretUsed ? ' turretLauncherItemUsed' : ''}">&nbsp;</div>`, true)).join('')
    + topItem(43, `<div class="turretLauncherIcon">${uiIcon('hardpoint-launcher', 16)}</div>`, true)
    + Array.from({ length: Math.max(0, launcherTotal) }, (_, index) => topItem(39 - index * 3, `<div class="turretLauncherItem${index < launcherUsed ? ' turretLauncherItemUsed' : ''}">&nbsp;</div>`, true)).join('');
  const rigTotal = value('upgradeCapacity');
  const rigUsed = stats?.items.reduce((total, item) => total + itemValue(item, 'upgradeCost'), 0) ?? 0;
  const meters = topItem(-47, `<div class="usage">${usageSVG(-30, 30, 2, '#3d4547', rigTotal ? rigUsed / rigTotal : 0)}</div>`, true)
    + topItem(134.5, `<div class="usage">${usageSVG(-44, 40, 5, '#2a504f', value('cpuOutput') ? (value('cpuOutput') - value('cpuFree')) / value('cpuOutput') : 0)}</div>`, true)
    + topItem(135, `<div class="usage">${usageSVG(44, 40, 5, '#541208', value('powerOutput') ? (value('powerOutput') - value('powerFree')) / value('powerOutput') : 0)}</div>`, true);
  const menus = topItem(-45, radialMenu('High'), true) + topItem(43, radialMenu('Medium'), true) + topItem(133, radialMenu('Low'), true);
  const slots = (Object.keys(slotRotations) as FitSlotType[]).map(type => {const [first,step,total] = slotRotations[type];return Array.from({ length: total }, (_, position) => {
    const index = position + 1;
    const available = !!fit && index <= slotCount(type);
    const module = fit?.modules.find((item) => item.slot.type === type && item.slot.index === index);
    const computed = stats?.items.find((item) => item.slot?.type === type && item.slot?.index === index);
    return topItem(first + step * position, slotMarkup(type, index, available, module, computed, assets));
  }).join('')}).join('');
  return `<div class="fit-editor-ring fit">${outerRing()}${innerRing()}${hull}${link}${slotDefs()}<div class="ringTop">${hardpoints}${meters}${menus}${slots}</div><div class="hullDraggable" data-fit-hull-drop></div>${fit ? '' : '<div class="fit-editor-ring-prompt"><strong>Pick a hull to start</strong><span>Choose a ship from the sidebar or paste an EFT / share link from the top bar.</span></div>'}</div>`;
}
