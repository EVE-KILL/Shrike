import type {Fit,FitModule,FitSlot,FitSlotType,FitState} from './types'
import type {FitStats,ComputedItem,SkillMap} from '../../dogma/src/types'
import type {EditorScope} from './editor-scope'
import {initialEditorState,isEditorSlot,isEditorState,type CatalogSlot,type CatalogEntry,type ContextSlot} from './editor-model'
import type {BayKind} from '../browser/shared/fit-render'
import type {FitResourceField} from '../browser/fit-resources'
import {eventElement,requiredElement} from '../browser/shared/dom'
import {BrowserRequestError} from '../browser/browser-request'
import {isRecord,errorMessage,isAbortError} from '../browser/shared/json'
import {moveTabFocus} from '../browser/shared/tabs'
import {encodeEditorFit,decodeEditorFit} from './editor-codec'
import {editorSlots,editorToEngineFit,readSavedEditorFit,readEditorFamilies,readEditorPrices,restoreEditorWorkspace} from './editor-model'
import {escapeHTML} from '../browser/shared/html';
import { calculateFit, getHullStats } from '../browser/dogma.js';
import { renderWheel } from '../browser/fit-wheel.js';
import { renderFitStats } from '../browser/fit-stats.js';
import { renderFitResourceBar, renderFitBayPanel } from '../browser/fit-resources.js';
import { rankFitSuggestions, renderFitSuggestions, renderFitComparison } from '../browser/fit-variants.js';
import { savedFitToUI, fitToSaveBody as saveBody, enforcePropmodExclusivity } from '../browser/fit-document.js';
import { hullSlotCount, remainingHardpoints, hardpointAttribute as moduleHardpoint, fittingIssues } from '../browser/fit-capacity.js';
import { fuzzyFitMatch, fuzzyFitScore } from '../browser/fit-catalog-search.js';
import { exportFitToEft, importFitFromEft } from '../browser/fit-eft.js';
import { FIT_TABLE_COLUMN_MINIMUMS, fitTableColumns, fitTableMinimumWidth } from '../browser/fit-table-data.js';
import { readHullDrop } from '../browser/fit-drag-data.js';

type CatalogTree = {children:Map<string,CatalogTree>;items:CatalogEntry[]}
const elementValue=(element:Element)=>{if(!(element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement || element instanceof HTMLSelectElement))throw new Error('Invalid fitting input');return element.value}

export function createEditorRuntime(root:HTMLElement,scope:EditorScope) {
  const detail = root.dataset.fitDetail === 'true';
  const owner = root.dataset.fitOwner === 'true';
  let visibility = 3;
  const originalParent = root.parentNode!;
  const originalNextSibling = root.nextSibling;
  function $(selector:'[data-fit-search]'):HTMLInputElement;
  function $(selector:'[data-fit-skill]'):HTMLSelectElement;
  function $(selector:'[data-fit-hull-image]'):HTMLImageElement;
  function $(selector:'[data-fit-save-form]'|'[data-fit-rename-form]'):HTMLFormElement;
  function $(selector:'[data-fit-save-dialog]'|'[data-fit-rename-dialog]'|'[data-fit-discard-dialog]'):HTMLDialogElement;
  function $(selector:'[data-fit-description]'|'[data-fit-stage] .fit-editor-ring-frame'):HTMLElement|null;
  function $(selector:`[data-fit-action="${string}"]`|`[data-fit-rename-form] [type="submit"]`):HTMLButtonElement;
  function $(selector:string):HTMLElement;
  function $(selector:string):HTMLElement|null {const found=root.querySelector<HTMLElement>(selector);if(!found && selector !== '[data-fit-description]' && selector !== '[data-fit-stage] .fit-editor-ring-frame')throw new Error('Missing fitting component '+selector);return found}
  function formField(form:HTMLFormElement,name:'name'):HTMLInputElement;
  function formField(form:HTMLFormElement,name:'description'):HTMLTextAreaElement;
  function formField(form:HTMLFormElement,name:'visibility'):HTMLInputElement|HTMLSelectElement|RadioNodeList;
  function formField(form:HTMLFormElement,name:string):HTMLInputElement|HTMLTextAreaElement|HTMLSelectElement|RadioNodeList {
    const field=form.elements.namedItem(name);
    if(field instanceof HTMLInputElement || field instanceof HTMLTextAreaElement || field instanceof HTMLSelectElement || field instanceof RadioNodeList)return field;
    throw new Error('Missing fitting form field '+name);
  }
  const slotTypes = editorSlots;
  const slotEffects:Record<FitSlotType,string> = { High: 'hiPower', Medium: 'medPower', Low: 'loPower', Rig: 'rigSlot', SubSystem: 'subSystem' };
  const statGroups:Array<[string,Array<[string,string,string,number,number?]>]> = [
    ['Capacitor', [['Capacitor', 'capacitorCapacity', ' GJ', 0], ['Recharge', 'rechargeRate', ' s', 1, 1000], ['Peak recharge', 'capacitorPeakDelta', ' GJ/s', 1]]],
    ['Offense', [['Damage', 'damagePerSecondWithReload', ' DPS', 1], ['Volley', 'damageAlpha', ' HP', 0], ['Drone damage', 'droneDamagePerSecond', ' DPS', 1]]],
    ['Defense', [['Effective HP', 'ehp', ' EHP', 0], ['Shield', 'shieldCapacity', ' HP', 0], ['Armor', 'armorHP', ' HP', 0], ['Hull', 'hp', ' HP', 0], ['Shield repair', 'shieldBoostRate', ' HP/s', 1], ['Armor repair', 'armorRepairRate', ' HP/s', 1]]],
    ['Targeting', [['Range', 'maxTargetRange', ' km', 1, 1000], ['Scan resolution', 'scanResolution', ' mm', 0], ['Signature', 'signatureRadius', ' m', 0]]],
    ['Navigation', [['Velocity', 'maxVelocity', ' m/s', 1], ['Align time', 'alignTime', ' s', 2], ['Warp speed', 'warpSpeedMultiplier', ' AU/s', 1]]],
    ['Drones', [['Bandwidth', 'droneBandwidth', ' Mbit/s', 0], ['Control distance', 'droneControlDistance', ' km', 1, 1000]]],
  ];
  const wantedStats = [...new Set(statGroups.flatMap((group) => group[1].map((row) => row[1])).concat(['cpuOutput', 'cpuFree', 'powerOutput', 'powerFree', 'droneCapacity', 'capacity', 'droneBandwidthLoad', 'turretSlotsLeft', 'launcherSlotsLeft', 'upgradeCapacity', 'capacitorDepletesIn', 'capacitorPeakDeltaPercentage', 'shieldRechargeRate', 'shieldBoostRate', 'shieldEffectiveBoostRate', 'armorRepairRate', 'armorEffectiveRepairRate', 'hullRepairRate', 'hullEffectiveRepairRate', 'passiveShieldRechargeRate', 'passiveShieldEffectiveRechargeRate', 'damagePerSecondWithoutReload', 'remoteShieldBoostRate', 'remoteArmorRepairRate', 'remoteHullRepairRate', 'energyNeutralizerRate', 'energyNosferatuRate', 'remoteCapTransferRate', ...['shield', 'armor', ''].flatMap((layer) => ['Em', 'Thermal', 'Kinetic', 'Explosive'].map((damage) => `${layer}${layer ? damage : damage[0].toLowerCase() + damage.slice(1)}DamageResonance`))]))];
  const state = initialEditorState(scope.sde,new URLSearchParams(location.search));
  // The input is usable while SDE loads, before runtime listeners exist.
  state.search = $('[data-fit-search]').value;
  const currentFit=()=>{if(!state.fit)throw new Error('No selected hull');return state.fit};
  let startup:Promise<void>;
  let started = false;
  let documentGeneration = 0;
  let finishResize:(()=>void)|undefined;
  const originalOverflow = document.body.style.overflow;
  const current = (epoch:number)=>scope.active() && scope.epoch() === epoch;
  const tableColumnLabels = ['Name', 'DPS', 'Volley', 'CPU', 'PG', 'Range', 'Charge'];
  const tableColumnMinimums = FIT_TABLE_COLUMN_MINIMUMS;
  const tableColumnPrefsKey = 'ek-fit-table-columns-v1';
  const workspacePrefsKey = 'ek-fit-workspace-v1';
  const saveWorkspacePrefs = () => {
    if (detail) return;
    try { localStorage.setItem(workspacePrefsKey, JSON.stringify({ sidebarTab: state.tab, fittingView: state.view, expandedBay: state.bay, workspaceExpanded: state.expanded })); } catch {}
  };
  const restoreWorkspacePrefs = () => {
    if (detail) return;
    try {
      const saved = JSON.parse(localStorage.getItem(workspacePrefsKey) ?? 'null');
      Object.assign(state, restoreEditorWorkspace(saved, new URLSearchParams(location.search), window.innerWidth));
    } catch {}
    // An explicit URL view takes precedence over remembered preferences.
    if (new URLSearchParams(location.search).get('view') === 'table') state.view = 'table';
  };
  const contextMenu = document.createElement('div');
  contextMenu.className = 'fit-editor-context-menu';
  contextMenu.hidden = true;
  root.append(contextMenu);
  let contextSlot:ContextSlot|null = null;
  let pendingCatalogClick:{id:number;time:number}|null = null;
  let tableFocusSlot:string|null = null;
  let tableSelection:FitSlot|null = null;
  const escape = escapeHTML;
  const nameOf = (id:number) => state.sde?.types.get(Number(id))?.name ?? `Type ${id}`;
  const iconOf = (id:number, size = 32) => `/images/types/${id}/icon?size=${size}`;
  const uiIcon = (name:string, size = 18) => `<img src="/assets/fit-ui/${name}.webp" width="${size}" height="${size}" alt="" loading="lazy">`;
  const fitSprite = root.querySelector<HTMLElement>('.fit-editor-icon use')?.getAttribute('href')?.split('#')[0] ?? '/assets/fit-icons.svg';
  const fitSvg = (name:string, size = 12) => `<svg class="fit-editor-ui-icon" width="${size}" height="${size}" viewBox="0 0 24 24" aria-hidden="true"><use href="${fitSprite}#${name}"></use></svg>`;
  const fmt = (value:number, digits = 0) => Number.isFinite(value) ? value.toLocaleString('en-US', { maximumFractionDigits: digits, minimumFractionDigits: digits }) : '—';
  const snapshot = (fit:Fit) => structuredClone(fit);
  const emptyFit = (shipTypeId:number):Fit => ({ shipTypeId, name: 'New Fit', description: '', modules: [], drones: [], cargo: [] });
  const selectedModule = () => state.fit?.modules.find((module) => module.slot.type === state.selected?.type && module.slot.index === state.selected?.index);
  const typeAttr = (typeId:number, name:string) => {
    const id = state.sde.attributeNameToId.get(name);
    if (id === undefined) return 0;
    return state.sde.typeDogma.get(typeId)?.dogmaAttributes?.find((attribute) => attribute.attributeID === id)?.value ?? 0;
  };
  const value = (name:string) => state.values?.[name] ?? 0;
  const slotCount = (type:FitSlotType) => hullSlotCount(state.fit, state.sde, state.stats, type);
  const slotFor = (typeId:number):CatalogSlot|null => {
    const type = state.sde.types.get(typeId);
    if (!type) return null;
    if (type.categoryID === 18) return 'DroneBay';
    if (type.categoryID === 8) return 'Charge';
    const effects = state.sde.typeDogma.get(typeId)?.dogmaEffects ?? [];
    for (const [slot, effectName] of Object.entries(slotEffects)) {
      const effectId = state.effectIds.get(effectName);
      if (isEditorSlot(slot) && effects.some((effect) => effect.effectID === effectId)) return slot;
    }
    return null;
  };
  const marketPath = (groupId:number) => {
    let current = groupId;
    const names = [];
    for (let i = 0; i < 12 && current != null; i++) {
      const group = state.sde.marketGroups.get(current);
      if (!group) break;
      names.push(group.name ?? 'Other');
      const parent = group.parentGroupID;
      if (parent == null || !state.sde.marketGroups.has(parent)) break;
      current = parent;
    }
    names.pop();
    return names.reverse();
  };
  const moduleFitsHull = (typeId:number, slot:CatalogSlot|null) => {
    const hullId = state.fit?.shipTypeId;
    if (!hullId) return false;
    const type = state.sde.types.get(typeId);
    if (!type) return false;
    if (slot === 'Rig' && typeAttr(typeId, 'rigSize') !== typeAttr(hullId, 'rigSize')) return false;
    if (slot === 'DroneBay' && (state.stats ? value('droneBandwidth') : typeAttr(hullId, 'droneBandwidth')) <= 0) return false;
    if (isEditorSlot(slot) && slotCount(slot) === 0) return false;
    if ((type.volume ?? 0) >= 3500 && typeAttr(hullId, 'isCapitalSize') !== 1) return false;
    const attrs = state.sde.typeDogma.get(typeId)?.dogmaAttributes ?? [];
    const hullGroup = state.sde.types.get(hullId)?.groupID;
    let restricted = false, matched = false;
    for (const attr of attrs) {
      const name = state.sde.dogmaAttributes.get(attr.attributeID)?.name ?? '';
      if (name === 'fitsToShipType' || /^canFitShipType\d+$/.test(name)) { restricted = true; if (attr.value === hullId) matched = true; }
      if (/^canFitShipGroup\d+$/.test(name)) { restricted = true; if (attr.value === hullGroup) matched = true; }
    }
    return !restricted || matched;
  };
  let toastTimer:ReturnType<typeof setTimeout>|undefined;
  const toast = (message:string) => {
    if(!scope.active())return;
    const element = $('[data-fit-toast]');
    element.textContent = message;
    element.classList.add('visible');
    scope.clear(toastTimer);
    toastTimer = scope.later(() => element.classList.remove('visible'), 3500);
  };
  const setStatus = (message:string) => { $('[data-fit-status]').textContent = message; };
  const priceCache = new Map<number,number>();
  const compactISK = (amount:number) => amount >= 1e12 ? `${(amount / 1e12).toFixed(2)}T` : amount >= 1e9 ? `${(amount / 1e9).toFixed(2)}B` : amount >= 1e6 ? `${(amount / 1e6).toFixed(2)}M` : amount >= 1e3 ? `${(amount / 1e3).toFixed(1)}K` : fmt(amount);
  const skillProfiles = new Map<number,SkillMap>();
  const skillProfile = () => {
    if (!skillProfiles.has(state.skillLevel)) skillProfiles.set(state.skillLevel, Object.fromEntries([...state.sde.types].filter(([, type]) => type.categoryID === 16 && type.published).map(([id]) => [String(id), state.skillLevel])));
    return skillProfiles.get(state.skillLevel);
  };
  const toEngineFit = editorToEngineFit;
  async function priceForFit(fit:Fit) {
    const ids = [...new Set([fit.shipTypeId, ...fit.modules.flatMap((item) => [item.typeId, item.charge?.typeId]), ...fit.drones.map((item) => item.typeId)].filter((id):id is number=>typeof id === 'number'))];
    const missing = ids.filter((id) => !priceCache.has(id));
    if (missing.length) {
      const prices = readEditorPrices(await scope.json(`/api/prices/bulk?types=${missing.join(',')}`), missing);
      for (const [id, price] of prices) priceCache.set(id, price);
    }
    let price = priceCache.get(fit.shipTypeId) ?? 0;
    for (const item of fit.modules) price += (priceCache.get(item.typeId) ?? 0) + (priceCache.get(item.charge?.typeId ?? 0) ?? 0);
    for (const item of fit.drones) price += (priceCache.get(item.typeId) ?? 0) * (item.states.Active + item.states.Passive);
    return price;
  }
  async function calculatePrice() {
    const fit = state.fit, generation = ++state.priceGeneration;
    if (!fit) { state.price = null; return; }
    try {
      const price = await priceForFit(fit);
      if (!scope.active() || generation !== state.priceGeneration) return;
      state.price = price;
    } catch { if (!scope.active() || generation !== state.priceGeneration) return; state.price = null; }
    renderHeader();
    if (state.expanded) renderVariants();
  }
  async function calculateCustomFit() {
    const fit = state.customFit, generation = ++state.customGeneration;
    if (!fit) { state.customStats = null; state.customPrice = null; return; }
    try {
      const [stats, price] = await Promise.all([calculateFit(toEngineFit(fit), skillProfile()), priceForFit(fit)]);
      if (!scope.active() || generation !== state.customGeneration) return;
      state.customStats = stats;
      state.customPrice = price;
    } catch (error) {
      if (!scope.active() || generation !== state.customGeneration) return;
      console.error('Could not compare custom fitting', error);
      state.customStats = null;
      state.customPrice = null;
    }
    renderVariants();
  }

  const decodeFit = (input:string) => decodeEditorFit(input, state.sde, location.href);
  const encodeFit = encodeEditorFit;
  async function syncURL() {
    if (detail) return;
    const generation = ++state.urlGeneration;
    const encoded = state.fit ? await encodeFit(currentFit()) : null;
    if (!scope.active() || generation !== state.urlGeneration) return;
    const url = new URL(location.href);
    if (encoded) url.searchParams.set('fit', encoded); else url.searchParams.delete('fit');
    if (state.view === 'table') url.searchParams.set('view', 'table'); else url.searchParams.delete('view');
    history.replaceState(history.state, '', url);
  }
  function commit(next:Fit|null) {
    closeContextMenu();
    state.previewTarget = null;
    state.previewGeneration++;
    if (state.fit) state.undo.push(snapshot(currentFit()));
    if (state.undo.length > 80) state.undo.shift();
    state.redo = [];
    state.fit = next;
    const customChanged = state.expanded && !state.selectedFamily;
    if (customChanged) state.customFit = next ? snapshot(next) : null;
    state.stats = state.values = null;
    state.price = null;
    if (next) state.tab = 'hardware';
    render();
    syncURL().catch(() => toast('Could not update the shareable URL'));
    calculate();
    calculatePrice();
    if (customChanged) calculateCustomFit();
    if (state.expanded) loadVariants();
  }
  function historyMove(from:Fit[], to:Fit[]) {
    if (!from.length) return;
    closeContextMenu();
    if (state.fit) to.push(snapshot(currentFit()));
    state.fit = from.pop() ?? null;
    if (state.expanded && !state.selectedFamily) { state.customFit = state.fit ? snapshot(currentFit()) : null; calculateCustomFit(); }
    state.stats = state.values = null;
    state.price = null;
    render();
    syncURL();
    calculate();
    calculatePrice();
  }
  async function calculate() {
    const fit = state.fit;
    const generation = ++state.generation;
    if (!fit) return;
    setStatus('Calculating…');
    try {
      const stats = await calculateFit(toEngineFit(fit), skillProfile());
      const values = await getHullStats(stats, wantedStats);
      if (!scope.active() || generation !== state.generation) return;
      state.stats = stats;
      state.values = values;
      setStatus('');
      render();
    } catch (error) {
      if (!scope.active() || generation !== state.generation) return;
      console.error('Could not calculate fit', error);
      setStatus('Calculation unavailable');
      toast('The dogma engine could not calculate this fit');
    }
  }
  function renderCatalog() {
    const catalog = $('[data-fit-catalog]');
    if (!state.sde) return;
    const query = state.search.trim().toLowerCase();
    const pills = $('[data-fit-filter-pills]');
    pills.innerHTML = state.tab === 'hulls'
      ? ['fitting-local', 'fitting-character', 'fitting-corporation', 'fitting-alliance', 'fitting-hull'].map((name) => `<span title="${name}">${uiIcon(name, 20)}</span>`).join('')
      : `<button type="button" data-fit-filter-slot="Low" aria-pressed="${state.filterSlots.has('Low')}" title="Low slots">${uiIcon('fitting-lowslot', 20)}</button><button type="button" data-fit-filter-slot="Medium" aria-pressed="${state.filterSlots.has('Medium')}" title="Medium slots">${uiIcon('fitting-medslot', 20)}</button><button type="button" data-fit-filter-slot="High" aria-pressed="${state.filterSlots.has('High')}" title="High slots">${uiIcon('fitting-hislot', 20)}</button><button type="button" data-fit-filter-slot="Rig" aria-pressed="${state.filterSlots.has('Rig')}" title="Rigs">${uiIcon('fitting-rig-subsystem', 20)}</button><button type="button" data-fit-filter-slot="DroneBay" aria-pressed="${state.filterSlots.has('DroneBay')}" title="Drones">${uiIcon('fitting-drones', 20)}</button><button type="button" data-fit-hull-restriction aria-pressed="${state.hullRestriction}" title="Only show modules for this hull">${uiIcon('fitting-hull-restriction', 20)}</button>`;
    const itemHTML = (entry:Pick<CatalogEntry,'id'|'name'|'group'> & Partial<Pick<CatalogEntry,'slot'>>, kind:'hull'|'type') => `<button type="button" class="fit-editor-catalog-item" data-fit-${kind}="${entry.id}" ${kind === 'type' ? 'draggable="true"' : ''}><img src="${iconOf(entry.id)}" alt="" width="24" height="24" loading="lazy"><span>${escape(entry.name)}${kind === 'type' ? `<small>${escape(entry.group)} · ${escape(entry.slot ?? 'Cargo')}</small>` : ''}</span><b>+</b></button>`;
    const searchHTML = (entry:Pick<CatalogEntry,'id'|'name'>, kind:'hull'|'type', index:number) => `<button type="button" class="fit-editor-catalog-item fit-editor-search-result${index === state.catalogSelection ? ' selected' : ''}" data-fit-${kind}="${entry.id}" data-fit-result-index="${index}" ${kind === 'type' ? 'draggable="true"' : ''}><img src="${iconOf(entry.id)}" alt="" width="24" height="24" loading="lazy"><span>${escape(entry.name)}</span></button>`;
    if (state.tab === 'hulls') {
      const races:Record<number,string> = { 500003: 'Amarr', 500001: 'Caldari', 500004: 'Gallente', 500002: 'Minmatar' };
      let hulls = [...state.sde.types].filter(([, type]) => type.categoryID === 6 && type.published && type.marketGroupID != null).map(([id, type]) => ({ id, name: type.name, group: state.sde.groups.get(type.groupID)?.name ?? 'Other', race: races[type.factionID ?? 0] ?? 'Non-Empire' }));
      if (query) hulls = hulls.filter((hull) => fuzzyFitMatch(query, hull.name)).sort((a, b) => fuzzyFitScore(query, a.name) - fuzzyFitScore(query, b.name) || a.name.localeCompare(b.name)).slice(0, 15);
      else hulls.sort((a, b) => a.group.localeCompare(b.group) || a.name.localeCompare(b.name));
      const groups = Map.groupBy(hulls, (hull) => hull.group);
      catalog.innerHTML = !hulls.length ? '<p class="fit-editor-empty">No matching hulls</p>' : query
        ? hulls.map((item, index) => searchHTML(item, 'hull', index)).join('')
        : [...groups].map(([group, items]) => `<details class="fit-editor-hull-group"><summary>${escape(group)}</summary><div class="fit-editor-hull-races">${['Amarr', 'Caldari', 'Gallente', 'Minmatar', 'Non-Empire'].map((race) => {
          const selected = items.filter((item) => item.race === race);
          return selected.length ? `<details class="fit-editor-race"><summary>${race} · ${selected.length}</summary>${selected.map((item) => itemHTML(item, 'hull')).join('')}</details>` : '';
        }).join('')}</div></details>`).join('');
    } else {
      const tabs = `<div class="fit-editor-hardware-tabs">${['modules', 'charges'].map((mode) => `<button type="button" data-fit-mode="${mode}" aria-pressed="${state.catalogMode === mode}">${mode[0].toUpperCase() + mode.slice(1)}</button>`).join('')}</div>`;
      if (!state.fit) { catalog.innerHTML = `${tabs}<p class="fit-editor-empty">Choose a hull first</p>`; return; }
      let entries:CatalogEntry[] = [];
      for (const [id, type] of state.sde.types) {
        if (!type.published || type.marketGroupID == null) continue;
        if (query && !fuzzyFitMatch(query, type.name)) continue;
        const slot = slotFor(id);
        const include = state.catalogMode === 'modules' ? isEditorSlot(slot) || slot === 'DroneBay' : slot === 'Charge';
        if (!include) continue;
        if (state.hullRestriction && state.catalogMode === 'modules' && !moduleFitsHull(id, slot)) continue;
        if (state.catalogMode === 'modules' && state.filterSlots.size && !(slot !== null && state.filterSlots.has(slot)) && !(slot === 'SubSystem' && state.filterSlots.has('Rig'))) continue;
        const path = marketPath(type.marketGroupID);
        entries.push({ id, name: type.name, slot, group: path[0] ?? 'Other', path, meta: type.metaGroupID ?? 0 });
      }
      entries.sort((a, b) => query ? fuzzyFitScore(query, a.name) - fuzzyFitScore(query, b.name) || a.meta - b.meta || a.name.localeCompare(b.name) : a.group.localeCompare(b.group) || a.name.localeCompare(b.name));
      if (query) catalog.innerHTML = `${tabs}${entries.length ? entries.slice(0, 15).map((entry, index) => searchHTML(entry, 'type', index)).join('') : '<p class="fit-editor-empty">No matching hardware</p>'}`;
      else {
        const groups = [...Map.groupBy(entries, (entry) => entry.group)];
        state.catalogGroups = groups.map(([, items]) => items);
        catalog.innerHTML = `${tabs}${groups.map(([group], index) => `<details data-fit-catalog-group="${index}"><summary>${escape(group)}</summary><div></div></details>`).join('')}`;
      }
    }
  }
  function selectCatalogResult(index:number) {
    const results = [...$('[data-fit-catalog]').querySelectorAll<HTMLElement>('[data-fit-result-index]')];
    if (!results.length) return;
    state.catalogSelection = Math.max(0, Math.min(index, results.length - 1));
    for (const result of results) result.classList.toggle('selected', Number(result.dataset.fitResultIndex) === state.catalogSelection);
    results[state.catalogSelection].scrollIntoView({ block: 'nearest' });
  }
  function renderStage() {
    const stage = $('[data-fit-stage]');
    const focusedSlot = document.activeElement?.closest<HTMLElement>('.fit-editor-table tr[data-fit-slot]')?.dataset.fitSlot ?? tableFocusSlot;
    const compactHull = $('[data-fit-hull-compact]');
    compactHull.hidden = !state.fit || state.view !== 'table';
    if (state.fit) {
      $('[data-fit-hull-title]').textContent = nameOf(currentFit().shipTypeId);
      $('[data-fit-hull-group]').textContent = state.sde.groups.get(state.sde.types.get(currentFit().shipTypeId)?.groupID ?? 0)?.name ?? 'Ship';
      $('[data-fit-hull-image]').src = `/images/types/${currentFit().shipTypeId}/render?size=64`;
      $('[data-fit-hull-image]').alt = nameOf(currentFit().shipTypeId);
    }
    for (const button of root.querySelectorAll<HTMLElement>('[data-fit-view]')) button.setAttribute('aria-pressed', String(button.dataset.fitView === state.view));
    const rows = state.fit ? slotTypes.flatMap((type) => Array.from({ length: slotCount(type) }, (_, position) => ({ type, index: position + 1, module: currentFit().modules.find((module) => module.slot.type === type && module.slot.index === position + 1) }))) : [];
    if (state.view === 'table' && state.fit) {
      const itemValue = (item:ComputedItem|null|undefined, name:string) => {
        const id = state.sde.attributeNameToId.get(name);
        const attr = id === undefined ? null : item?.attributes?.get(id);
        return attr?.value ?? attr?.base_value ?? 0;
      };
      const number = (value:number, digits = 0) => value ? fmt(value, digits) : '';
      const groups = slotTypes.map((type) => {
        const count = Math.max(slotCount(type), ...currentFit().modules.filter((item) => item.slot.type === type).map((item) => item.slot.index), 0);
        if (!count) return '';
        const label = { High: 'High Slots', Medium: 'Med Slots', Low: 'Low Slots', Rig: 'Rigs', SubSystem: 'Subsystems' }[type];
        const body = Array.from({ length: count }, (_, position) => {
          const index = position + 1;
          const module = currentFit().modules.find((item) => item.slot.type === type && item.slot.index === index);
          const computed = module && state.stats?.items.find((item) => item.type_id === module.typeId && item.slot?.type === type && item.slot?.index === index);
          const maxRange = computed ? itemValue(computed, 'maxRange') : 0;
          const falloff = computed ? itemValue(computed, 'falloff') : 0;
          const missileRange = !maxRange && computed?.charge ? itemValue(computed.charge, 'maxVelocity') * itemValue(computed.charge, 'explosionDelay') / 1000000 : 0;
          const range = maxRange ? `${fmt(maxRange / 1000, 1)}${falloff ? `+${fmt(falloff / 1000, 1)}` : ''}km` : missileRange ? `${fmt(missileRange, 1)}km` : '';
          return `<tr data-fit-slot="${type}:${index}" tabindex="0" class="${tableSelection?.type === type && tableSelection?.index === index ? 'selected' : ''} ${module?.state === 'Passive' ? 'passive' : ''} ${module?.state === 'Preview' ? 'preview' : ''} ${!module ? 'empty' : ''}"><td>${module ? `<img src="${iconOf(module.typeId)}" alt="" width="20" height="20"><span>${escape(nameOf(module.typeId))}</span>` : '<em>Empty Slot</em>'}</td><td>${number(itemValue(computed, 'damagePerSecondWithReload'), 1)}</td><td>${number(itemValue(computed, 'damageVolley'))}</td><td>${number(itemValue(computed, 'cpu'))}</td><td>${number(itemValue(computed, 'power'))}</td><td>${range}</td><td>${module?.charge ? escape(nameOf(module.charge.typeId)) : ''}</td></tr>`;
        }).join('');
        return `<tr class="fit-editor-table-group"><td colspan="7">— ${count} ${label} —</td></tr>${body}`;
      }).join('');
      const drones = currentFit().drones.length ? `<tr class="fit-editor-table-group"><td colspan="7">— Drones —</td></tr>${currentFit().drones.map((drone) => {
        const count = drone.states.Active + drone.states.Passive;
        const engineItem = state.stats?.items.find((item) => item.type_id === drone.typeId);
        const dps = engineItem ? itemValue(engineItem, 'damagePerSecondWithReload') * drone.states.Active : 0;
        return `<tr class="fit-editor-drone-row" data-fit-drone-row="${drone.typeId}"><td><img src="${iconOf(drone.typeId)}" alt="" width="20" height="20"><span>${escape(nameOf(drone.typeId))}</span><small>×${count}</small>${drone.states.Active ? `<small class="active">(${drone.states.Active} active)</small>` : ''}</td><td>${number(dps, 1)}</td><td></td><td></td><td></td><td></td><td></td></tr>`;
      }).join('')}` : '';
      const colgroup = state.columnWidths.map((width) => `<col${width === null ? '' : ` style="width:${width}px"`}>`).join('');
      const minimumWidth = fitTableMinimumWidth(state.columnWidths);
      const headings = tableColumnLabels.map((label, index) => `<th>${label}${index < 6 ? `<span class="fit-editor-column-resize" data-fit-resize-col="${index}" role="separator" aria-label="Resize ${label} column"></span>` : ''}</th>`).join('');
      stage.innerHTML = `<div class="fit-editor-table-wrap"><table style="min-width:${minimumWidth}px" class="fit-editor-table${state.columnWidths[0] !== null ? ' has-resized-columns' : ''}"><colgroup>${colgroup}</colgroup><thead><tr>${headings}</tr></thead><tbody>${groups}${drones}</tbody><tfoot><tr><td colspan="7"></td></tr></tfoot></table></div>`;
    } else {
      stage.innerHTML = `<div class="fit-editor-ring-frame"><div class="fit-editor-ring-canvas">${renderWheel({ fit: state.fit, stats: state.stats, sde: state.sde, slotCount, value, iconOf, uiIcon, escape, nameOf, selected: state.selected })}</div></div>`;
      syncWheelScale();
    }
    renderSelection();
    if (focusedSlot) stage.querySelector<HTMLElement>(`.fit-editor-table tr[data-fit-slot="${focusedSlot}"]`)?.focus();
  }
  function syncWheelScale() {
    const frame = $('[data-fit-stage] .fit-editor-ring-frame');
    if (!frame) return;
    const main = $('.fit-editor-main');
    if (state.expanded && window.innerWidth > 900) {
      // The grid gives the stage its own row above the resource bar and bays.
      // Size the fixed 600px wheel to that row, so it never covers either.
      const availableHeight = main.clientHeight - 32;
      const maxSize = window.innerHeight >= 1000 ? 720 : 600;
      const availableWidth = Math.max(0, main.clientWidth);
      frame.style.width = `${Math.min(maxSize, availableWidth, availableHeight)}px`;
    } else {
      frame.style.removeProperty('width');
    }
    const size = frame.getBoundingClientRect().width;
    frame.style.height = `${size}px`;
    requiredElement<HTMLElement>(frame,'.fit-editor-ring-canvas').style.transform = `scale(${size / 600})`;
  }
  function renderSelection() {
    const stage = $('[data-fit-stage]');
    stage.querySelector<HTMLElement>('.fit-editor-table-selection')?.remove();
    if (state.view !== 'table' || !tableSelection) return;
    const {type,index}=tableSelection;
    const module=state.fit?.modules.find(item=>item.slot.type===type&&item.slot.index===index);
    const wrap=stage.querySelector<HTMLElement>('.fit-editor-table-wrap');
    if (!module || !wrap) return;
    const fill=type==='High'&&hardpointAttribute(module.typeId)&&hardpointsLeft(module.typeId)>0&&currentFit().modules.filter(item=>item.slot.type==='High').length<slotCount('High');
    wrap.insertAdjacentHTML('beforebegin',`<div class="fit-editor-table-selection"><span>${escape(nameOf(module.typeId))}</span>${fill?`<button type="button" data-fit-slot-action="fill-rack:${type}:${index}">Fill highs</button>`:''}<button type="button" data-fit-slot-action="${module.charge?'uncharge':'unfit'}:${type}:${index}" class="remove">${module.charge?'Unload':'Remove'}</button></div>`);
  }
  function renderResources() {
    const bar = $('[data-fit-resources]'), bays = $('[data-fit-bays]');
    bar.hidden = !state.fit;
    bays.hidden = !state.fit || (!state.expanded && !state.bay);
    bays.classList.toggle('both', state.expanded);
    if (!state.fit) return;
    const fields:FitResourceField[] = [
      { label: 'CPU', used: value('cpuOutput') - value('cpuFree'), total: value('cpuOutput'), unit: 'tf', usedDigits: 1, totalDigits: 1 },
      { label: 'Power', used: value('powerOutput') - value('powerFree'), total: value('powerOutput'), unit: 'MW', usedDigits: 1, totalDigits: 1 },
      { label: 'Drone Bay', used: droneUsed(), total: droneCapacity(), unit: 'm³', usedDigits: 1, totalDigits: 0, bay: 'drones' },
      { label: 'Bandwidth', used: value('droneBandwidthLoad'), total: value('droneBandwidth'), unit: 'Mbit', usedDigits: 0, totalDigits: 0 },
      { label: 'Cargo', used: cargoUsed(), total: cargoCapacity(), unit: 'm³', usedDigits: 1, totalDigits: 0, bay: 'cargo' },
    ];
    bar.innerHTML = renderFitResourceBar({ fields, openBay: state.bay, fmt, icon: fitSvg });
    const kinds:BayKind[] = state.expanded ? ['drones', 'cargo'] : state.bay ? [state.bay] : [];
    bays.innerHTML = kinds.map((kind) => renderFitBayPanel({ kind, fit: currentFit(), sde: state.sde, iconOf, nameOf, typeAttr, escape, fmt, icon: fitSvg })).join('');
  }
  function renderStats() {
    $('[data-fit-stats]').innerHTML = renderFitStats({
      fit: state.fit, stats: state.stats, sde: state.sde, expanded: state.expanded,
      opened: state.statExpanded, uiIcon, iconOf, nameOf, escape,
    });
  }
  function renderVariants() {
    const panel = $('[data-fit-variants]');
    panel.hidden = !state.expanded;
    if (!state.expanded) return;
    const hullType = state.fit ? state.sde.types.get(currentFit().shipTypeId) : null;
    const tech = hullType?.metaGroupID === 2 ? 'Tech II' : hullType?.metaGroupID === 3 ? 'Storyline' : hullType?.metaGroupID === 4 ? 'Faction' : hullType?.metaGroupID === 14 ? 'Tech III' : 'Tech I';
    $('[data-fit-identity]').innerHTML = state.fit ? `<img src="/images/types/${currentFit().shipTypeId}/render?size=128" alt="" width="56" height="56"><span><span class="fit-editor-identity-title"><strong>${escape(nameOf(currentFit().shipTypeId))}</strong><em>${tech}</em></span><small>${escape(state.sde.groups.get(hullType?.groupID ?? 0)?.name ?? 'Ship')} · ${escape(state.fit.name)}</small><small class="fit-editor-identity-valid">◉ Fit valid</small></span>` : '';
    const modified = Boolean(state.selectedFamily && state.observedSnapshot && state.fit && JSON.stringify(state.observedSnapshot) !== JSON.stringify(state.fit));
    const mode = state.selectedFamily ? modified ? 'Modified observed fit' : 'Previewing observed fit' : 'Editing custom fit';
    $('[data-fit-variants-subtitle]').textContent = state.fit ? `${nameOf(currentFit().shipTypeId)} · ${mode}` : 'Pick a hull to see observed fits.';
    const comparisonToggle = $('[data-fit-variant-action="toggle-comparison"]');
    comparisonToggle.hidden = !state.selectedFamily;
    comparisonToggle.textContent = state.showComparison ? 'Hide diff' : 'Compare';
    const list = $('[data-fit-variants-list]');
    const scroll = list.scrollTop;
    const hasFamilies = (state.variantsData?.families.length ?? 0) > 0;
    const custom = hasFamilies && state.customFit ? `<button type="button" class="fit-editor-variant-custom ${state.selectedFamily ? '' : 'active'}" data-fit-variant-custom><span class="fit-editor-variant-custom-icon">${fitSvg('pencil', 14)}</span><span><strong>Custom Fit</strong><small>Your working fit · always preserved</small></span></button><div class="fit-editor-variant-divider"></div>` : '';
    const comparison = state.showComparison && state.selectedFamily && state.customFit && state.fit ? renderFitComparison({ customFit: state.customFit, fit: state.fit, customStats: state.customStats, stats: state.stats, customPrice: state.customPrice, currentPrice: state.price, sde: state.sde, modified, escape, nameOf, formatIsk: compactISK }) : '';
    const groups = hasFamilies && state.fit ? rankFitSuggestions({ fit: state.fit, data: state.variantsData, sde: state.sde, slotCount, cpuFree: Math.max(0, value('cpuFree')), powerFree: Math.max(0, value('powerFree')), typeAttr, hardpointAvailable: (id) => !hardpointAttribute(id) || hardpointsLeft(id) > 0 }) : [];
    const suggestions = renderFitSuggestions(groups, Boolean(state.fit?.modules.length), { escape, iconOf, icon: fitSvg });
    list.innerHTML = state.variantsLoading ? '<p class="fit-editor-variants-empty">Loading observed fits…</p>' : custom + comparison + suggestions + (state.variantsHTML ?? '<p class="fit-editor-variants-empty">No observed fitting families for this hull.</p>');
    list.scrollTop = scroll;
    for (const link of list.querySelectorAll<HTMLElement>('[data-fit-variant]')) link.closest<HTMLElement>('.fit-editor-variant-card')?.classList.toggle('active', link.dataset.fitVariant === state.selectedFamily);
  }
  async function loadVariants() {
    if (!state.expanded || !state.fit) { renderVariants(); return; }
    const hull = currentFit().shipTypeId;
    if (state.variantHull === hull && (state.variantsHTML !== null || state.variantsLoading)) { renderVariants(); return; }
    const generation = ++state.variantGeneration;
    state.variantHull = hull;
    state.variantsHTML = state.variantsData = null;
    state.variantsLoading = true;
    renderVariants();
    try {
      const html = await scope.text(`/_fragments/fits/variants/${hull}`);
      if (!scope.active() || generation !== state.variantGeneration) return;
      const fragment = document.createElement('template');
      fragment.innerHTML = html;
      const payload = fragment.content.querySelector<HTMLElement>('#fit-variants-data');
      if (!payload) throw new Error('Observed fit data is missing');
      state.variantsData = readEditorFamilies(JSON.parse(payload.textContent));
      payload.remove();
      state.variantsHTML = fragment.innerHTML;
    } catch (error) {
      if (!scope.active() || generation !== state.variantGeneration) return;
      console.error('Could not load observed fits', error);
      state.variantsHTML = '<p class="fit-editor-variants-empty">Could not load observed fits.</p>';
    } finally {
      if (scope.active() && generation === state.variantGeneration) { state.variantsLoading = false; renderVariants(); }
    }
  }
  function renderHeader() {
    $('[data-fit-name]').textContent = state.fit?.name ?? '';
    const price = $('[data-fit-price]');
    price.hidden = !state.fit;
    price.textContent = state.price != null && state.price > 0 ? `${compactISK(state.price)} ISK` : '— ISK';
    const validity = $('[data-fit-validity]');
    validity.hidden = !state.fit || !state.values;
    if (!validity.hidden) {
      const issues = fittingIssues(state.fit, state.sde, state.stats, state.effectIds);
      validity.textContent = issues.length ? `${issues.length} issues` : '◉ Valid';
      validity.title = issues.length ? issues.join(' · ') : 'Fit is mechanically valid';
      validity.classList.toggle('invalid', issues.length > 0);
    }
  }
  function render() {
    state.previewTarget = null;
    state.previewGeneration++;
    renderHeader();
    for (const button of root.querySelectorAll<HTMLElement>('[data-fit-tab]')) {const active=button.dataset.fitTab === state.tab;button.setAttribute('aria-selected',String(active));button.tabIndex=active?0:-1}
    $('[data-fit-catalog]').setAttribute('aria-labelledby',`fit-catalog-${state.tab}`);
    $('[data-fit-search]').placeholder = state.tab === 'hulls' ? 'Search hulls…' : 'Search modules & charges…';
    for (const action of ['copy-eft', 'share', 'rename', 'save', 'fork']) $(`[data-fit-action="${action}"]`).disabled = !state.fit;
    $('[data-fit-action="new"]').disabled = !state.fit;
    $('[data-fit-save-label]').textContent = state.savedId ? 'Update' : 'Save';
    $('[data-fit-action="undo"]').disabled = !state.undo.length;
    $('[data-fit-action="redo"]').disabled = !state.redo.length;
    renderCatalog(); renderResources(); renderStage(); renderStats(); renderVariants();
  }
  function paintSuggestionPreview(fit:Fit, stats:FitStats, values:Record<string,number|null>) {
    const committed = { fit: state.fit, stats: state.stats, values: state.values };
    state.fit = fit; state.stats = stats; state.values = values;
    try { renderHeader(); renderResources(); renderStage(); renderStats(); }
    finally { state.fit = committed.fit; state.stats = committed.stats; state.values = committed.values; }
  }
  function clearSuggestionPreview() {
    if (!state.previewTarget) return;
    state.previewTarget = null;
    state.previewGeneration++;
    renderHeader(); renderResources(); renderStage(); renderStats();
  }
  async function previewFitItem(button:HTMLElement, typeId:number, slot:CatalogSlot|null) {
    if (!state.fit || state.previewTarget === button) return;
    if (!state.sde.types.has(typeId) || !isEditorSlot(slot)) return;
    const index = Array.from({ length: slotCount(slot) }, (_, position) => position + 1).find((position) => !currentFit().modules.some((module) => module.slot.type === slot && module.slot.index === position));
    if (index === undefined) return;
    const fit = snapshot(currentFit());
    fit.modules.push({ typeId, slot: { type: slot, index }, state: 'Preview' });
    const generation = ++state.previewGeneration;
    state.previewTarget = button;
    const pricePromise = priceForFit(fit).catch(() => null);
    try {
      const stats = await calculateFit(toEngineFit(fit), skillProfile());
      const values = await getHullStats(stats, wantedStats);
      if (!scope.active() || generation !== state.previewGeneration || !button.isConnected) return;
      paintSuggestionPreview(fit, stats, values);
      const price = await pricePromise;
      if (scope.active() && generation === state.previewGeneration && button.isConnected && price !== null) $('[data-fit-price]').textContent = `${compactISK(price)} ISK`;
    } catch (error) {
      if (!scope.active() || generation !== state.previewGeneration) return;
      console.error('Could not preview fitting suggestion', error);
      clearSuggestionPreview();
    }
  }
  const previewSuggestion = (button:HTMLElement) => previewFitItem(button, Number(button.dataset.fitSuggestion), isEditorSlot(button.dataset.fitSuggestionSlot) ? button.dataset.fitSuggestionSlot : null);
  function selectSlot(type:string, index:number) {
    if(!isEditorSlot(type))return;
    state.selected = { type, index };
    state.tab = 'hardware'; state.catalogMode = 'modules';
    render();
    $('[data-fit-search]').focus();
  }
  function chargeCompatible(moduleId:number, chargeId:number) {
    const charge = state.sde.types.get(chargeId);
    if (!charge) return false;
    const groups = ['chargeGroup1', 'chargeGroup2', 'chargeGroup3', 'chargeGroup4', 'chargeGroup5'].map((name) => typeAttr(moduleId, name));
    if (!groups.includes(charge.groupID)) return false;
    const moduleSize = typeAttr(moduleId, 'chargeSize'), chargeSize = typeAttr(chargeId, 'chargeSize');
    return !moduleSize || !chargeSize || moduleSize === chargeSize;
  }
  const droneUsed = () => currentFit().drones.reduce((sum, drone) => sum + (state.sde.types.get(drone.typeId)?.volume ?? 0) * (drone.states.Active + drone.states.Passive), 0);
  const cargoUsed = () => currentFit().cargo.reduce((sum, item) => sum + (state.sde.types.get(item.typeId)?.volume ?? 0) * item.quantity, 0);
  const droneCapacity = () => value('droneCapacity') || typeAttr(currentFit().shipTypeId, 'droneCapacity');
  const cargoCapacity = () => value('capacity') || state.sde.types.get(currentFit().shipTypeId)?.capacity || 0;
  function canActivateDrone(fit:Fit, typeId:number) {
    const activeCount = fit.drones.reduce((sum, drone) => sum + drone.states.Active, 0);
    const bandwidthUsed = fit.drones.reduce((sum, drone) => sum + drone.states.Active * typeAttr(drone.typeId, 'droneBandwidthUsed'), 0);
    const bandwidth = value('droneBandwidth') || typeAttr(fit.shipTypeId, 'droneBandwidth');
    const perDrone = typeAttr(typeId, 'droneBandwidthUsed');
    return activeCount < 5 && (perDrone === 0 || bandwidthUsed + perDrone <= bandwidth);
  }
  function updateDroneGroup(typeId:number, action:string) {
    if (!state.fit) return;
    const fit = snapshot(currentFit()), drone = fit.drones.find((item) => item.typeId === typeId);
    if (!drone) return;
    if (action === 'deactivate-all') {
      drone.states.Passive += drone.states.Active;
      drone.states.Active = 0;
    } else if (action === 'activate-all') {
      const canActivateCount = () => {
        const trial = snapshot(fit);
        const candidate = trial.drones.find((item) => item.typeId === typeId);if(!candidate)return 0;
        let count = 0;
        while (candidate.states.Passive > 0 && canActivateDrone(trial, typeId)) { candidate.states.Passive--; candidate.states.Active++; count++; }
        return count;
      };
      for (const other of fit.drones) {
        if (other === drone) continue;
        while (other.states.Active > 0 && canActivateCount() < drone.states.Passive) { other.states.Active--; other.states.Passive++; }
      }
      while (drone.states.Passive > 0 && canActivateDrone(fit, typeId)) { drone.states.Passive--; drone.states.Active++; }
    } else if (action === 'remove-one') {
      if (drone.states.Passive > 0) drone.states.Passive--;
      else if (drone.states.Active > 0) drone.states.Active--;
    } else if (action === 'remove-all') {
      drone.states.Active = 0;
      drone.states.Passive = 0;
    }
    if (drone.states.Active === 0 && drone.states.Passive === 0) fit.drones = fit.drones.filter((item) => item !== drone);
    commit(fit);
  }
  function addType(typeId:number, targetOnly = false, targetSlot = false) {
    if (!state.fit) return;
    const type = state.sde.types.get(typeId);
    if (!type) return;
    const slot = slotFor(typeId);
    const fit = snapshot(currentFit());
    if (isEditorSlot(slot)) {
      const selected = targetSlot && state.selected?.type === slot ? state.selected.index : null;
      const index = selected ?? Array.from({ length: slotCount(slot) }, (_, i) => i + 1).find((i) => !fit.modules.some((module) => module.slot.type === slot && module.slot.index === i));
      if (index == null || index > slotCount(slot)) { toast(`No free ${slot.toLowerCase()} slot`); return; }
      if (hardpointAttribute(typeId) && hardpointsLeft(typeId) <= 0) { toast('No turret or launcher hardpoint available'); return; }
      fit.modules = fit.modules.filter((module) => module.slot.type !== slot || module.slot.index !== index);
      const defaultState = slot === 'High' || slot === 'Medium' ? 'Active' : slot === 'Low' ? 'Online' : 'Passive';
      const otherPropulsionActive = type.groupID === 46 && fit.modules.some((module) => state.sde.types.get(module.typeId)?.groupID === 46 && (module.state === 'Active' || module.state === 'Overload'));
      fit.modules.push({ slot: { type: slot, index }, typeId, state: otherPropulsionActive ? 'Online' : defaultState });
      state.selected = { type: slot, index };
    } else if (slot === 'Charge') {
      const matches = fit.modules.filter((item) => (!targetOnly || item.slot.type === state.selected?.type && item.slot.index === state.selected?.index) && chargeCompatible(item.typeId, typeId));
      if (!matches.length) { toast('No fitted module accepts that charge'); return; }
      for (const module of matches) module.charge = { typeId };
    } else if (slot === 'DroneBay') {
      if (droneUsed() + (type.volume ?? 0) > droneCapacity()) { toast('Drone bay is full'); return; }
      const drone = fit.drones.find((item) => item.typeId === typeId);
      const active = canActivateDrone(fit, typeId);
      if (drone) drone.states[active ? 'Active' : 'Passive']++;
      else fit.drones.push({ typeId, states: { Active: active ? 1 : 0, Passive: active ? 0 : 1 } });
    } else {
      if (cargoUsed() + (type.volume ?? 0) > cargoCapacity()) { toast('Cargo hold is full'); return; }
      const cargo = fit.cargo.find((item) => item.typeId === typeId);
      if (cargo) cargo.quantity++; else fit.cargo.push({ typeId, quantity: 1 });
    }
    commit(fit);
  }
  function addCargoItem(typeId:number) {
    if (!state.fit) return;
    const type = state.sde.types.get(typeId);
    if (!type || cargoUsed() + (type.volume ?? 0) > cargoCapacity()) { toast('Cargo hold is full'); return; }
    const fit = snapshot(currentFit());
    const item = fit.cargo.find((entry) => entry.typeId === typeId);
    if (item) item.quantity++; else fit.cargo.push({ typeId, quantity: 1 });
    commit(fit);
  }
  function removeSlot(type:string, index:number) {
    if (!state.fit) return;
    const fit = snapshot(currentFit());
    fit.modules = fit.modules.filter((module) => module.slot.type !== type || module.slot.index !== index);
    commit(fit);
  }
  function cycleState(type:string, index:number, reverse = false) {
    if (!state.fit) return;
    const fit = snapshot(currentFit()), module = fit.modules.find((item) => item.slot.type === type && item.slot.index === index);
    if (!module) return;
    const maxState = state.stats?.items.find((item) => item.slot?.type === type && item.slot?.index === index)?.max_state ?? 'Overload';
    const choices:Record<FitState,FitState[]> = {Passive:['Passive'],Online:['Passive','Online'],Active:['Passive','Online','Active'],Overload:['Passive','Online','Active','Overload']};
    const states = isEditorState(maxState) ? choices[maxState] : ['Passive'] as FitState[];
    module.state = states[(states.indexOf(isEditorState(module.state) ? module.state : 'Active') + (reverse ? -1 : 1) + states.length) % states.length];
    if ((module.state === 'Active' || module.state === 'Overload') && state.sde.types.get(module.typeId)?.groupID === 46) {
      for (const other of fit.modules) if (other !== module && state.sde.types.get(other.typeId)?.groupID === 46 && (other.state === 'Active' || other.state === 'Overload')) other.state = 'Online';
    }
    commit(fit);
  }
  function setModuleState(type:string, index:number, nextState:string) {
    if(!isEditorState(nextState))return;
    if (!state.fit) return;
    const fit = snapshot(currentFit()), module = fit.modules.find((item) => item.slot.type === type && item.slot.index === index);
    if (!module) return;
    module.state = nextState;
    if ((nextState === 'Active' || nextState === 'Overload') && state.sde.types.get(module.typeId)?.groupID === 46) {
      for (const other of fit.modules) if (other !== module && state.sde.types.get(other.typeId)?.groupID === 46 && (other.state === 'Active' || other.state === 'Overload')) other.state = 'Online';
    }
    commit(fit);
  }
  function toggleRackOverload(type:string) {
    if (!state.fit) return;
    const fit = snapshot(currentFit()), rack = fit.modules.filter((module) => module.slot.type === type);
    if (!rack.length) return;
    const overloaded = rack.some((module) => module.state === 'Overload');
    for (const module of rack) {
      if (overloaded && module.state === 'Overload') module.state = 'Active';
      else if (!overloaded && module.state === 'Active') module.state = 'Overload';
    }
    commit(fit);
  }
  function hardpointAttribute(typeId:number) {
    return moduleHardpoint(typeId, state.sde, state.effectIds);
  }
  function hardpointsLeft(typeId:number) {
    return remainingHardpoints(state.fit, state.sde, state.stats, state.effectIds, typeId);
  }
  function fillHighRack(typeId:number, chargeTypeId?:number|null) {
    if (!state.fit || !hardpointAttribute(typeId)) return;
    const fit = snapshot(currentFit());
    let left = hardpointsLeft(typeId);
    for (let index = 1; index <= slotCount('High') && left > 0; index++) {
      if (fit.modules.some((module) => module.slot.type === 'High' && module.slot.index === index)) continue;
      fit.modules.push({ typeId, slot: { type: 'High', index }, state: 'Active', ...(chargeTypeId ? { charge: { typeId: chargeTypeId } } : {}) });
      left--;
    }
    if (left !== hardpointsLeft(typeId)) commit(fit);
  }
  function closeContextMenu() {
    contextMenu.hidden = true;
    contextSlot = null;
  }
  function openContextMenu(event:MouseEvent, type:string, index:number) {
    if(!isEditorSlot(type))return;
    if (!state.fit || index > slotCount(type)) return;
    event.preventDefault();
    event.stopPropagation();
    const module = currentFit().modules.find((item) => item.slot.type === type && item.slot.index === index);
    const charges = module ? [...state.sde.types].filter(([id, entry]) => entry.published && entry.categoryID === 8 && chargeCompatible(module.typeId, id)).map(([id, entry]) => ({ id, name: entry.name })).sort((a, b) => a.name.localeCompare(b.name)) : [];
    const chargeButtons = charges.map((charge) => `<button type="button" data-fit-context-action="charge:${charge.id}"><img src="${iconOf(charge.id)}" width="16" height="16" alt="">${escape(charge.name)}</button>`).join('');
    const chargeAllButtons = charges.map((charge) => `<button type="button" data-fit-context-action="charge-all:${charge.id}"><img src="${iconOf(charge.id)}" width="16" height="16" alt="">${escape(charge.name)}</button>`).join('');
    const option = (label:string, action:string, extra = '') => `<button type="button" data-fit-context-action="${action}" class="${extra}">${label}</button>`;
    const states = module ? `<div class="fit-editor-context-group">${[['Offline', 'Passive'], ['Online', 'Online'], ['Active', 'Active'], ['Overload', 'Overload']].map(([label, value]) => option(`${label}${module.state === value ? '<small>current</small>' : ''}`, `state:${value}`, module.state === value ? 'current' : '')).join('')}</div><div class="fit-editor-context-group">${option('Overload Rack', 'rack-overload', 'overload')}</div>` : '';
    const canFill = module && type === 'High' && hardpointAttribute(module.typeId) && hardpointsLeft(module.typeId) > 0 && currentFit().modules.filter((item) => item.slot.type === 'High').length < slotCount('High');
    const fill = canFill ? `<div class="fit-editor-context-group">${option('Fill Empty Highs', 'fill-rack', 'fill-rack')}</div>` : '';
    const submenu = charges.length ? `<div class="fit-editor-context-group"><div class="fit-editor-context-submenu"><button type="button">Load Charge <span>›</span></button><div class="fit-editor-context-flyout">${chargeButtons}</div></div><div class="fit-editor-context-submenu"><button type="button">Load Charge Rack <span>›</span></button><div class="fit-editor-context-flyout">${chargeAllButtons}</div></div></div>` : '';
    contextMenu.innerHTML = module ? `${states}${fill}${submenu}${module.charge ? `<div class="fit-editor-context-group">${option('Remove Charge', 'remove-charge', 'remove-charge')}</div>` : ''}<div class="fit-editor-context-group">${option('Unfit Module', 'unfit', 'unfit')}</div>` : `<div class="fit-editor-context-empty">Empty ${escape(type)} Slot</div>`;
    contextSlot = { type, index, typeId: module?.typeId ?? 0, chargeTypeId: module?.charge?.typeId ?? null };
    contextMenu.style.left = `${Math.max(8, Math.min(event.clientX, window.innerWidth - 250))}px`;
    contextMenu.style.top = `${Math.max(8, Math.min(event.clientY, window.innerHeight - 360))}px`;
    contextMenu.classList.toggle('flyout-left', event.clientX > window.innerWidth / 2);
    contextMenu.hidden = false;
  }
  function openDroneContextMenu(event:MouseEvent, typeId:number) {
    const drone = state.fit?.drones.find((item) => item.typeId === typeId);
    if (!drone) return;
    event.preventDefault();
    event.stopPropagation();
    const option = (label:string, action:string, extra = '') => `<button type="button" class="${extra}" data-fit-context-action="drone-${action}">${label}</button>`;
    contextMenu.innerHTML = `<div class="fit-editor-context-group">${drone.states.Active > 0 ? option('Deactivate All', 'deactivate-all') : option('Activate All', 'activate-all')}</div><div class="fit-editor-context-group">${option('Remove One', 'remove-one')}${option('Remove All', 'remove-all', 'unfit')}</div>`;
    contextSlot = { droneId: typeId };
    contextMenu.style.left = `${Math.max(8, Math.min(event.clientX, window.innerWidth - 250))}px`;
    contextMenu.style.top = `${Math.max(8, Math.min(event.clientY, window.innerHeight - 160))}px`;
    contextMenu.hidden = false;
  }
  function removeCharge(type:string, index:number) {
    if (!state.fit) return;
    const fit = snapshot(currentFit()), module = fit.modules.find((item) => item.slot.type === type && item.slot.index === index);
    if (!module?.charge) return;
    delete module.charge;
    commit(fit);
  }
  let saving = false;
  let uncertainCreation = false;
  let reconcileSavedId:string|null = null;
  async function openSave() {
    if (!state.fit || detail && !owner || saving) return;
    const epoch = scope.epoch(),data = await scope.json('/auth/me');
    if(!current(epoch))return;
    if (!isRecord(data))throw new Error('Invalid login session');
    if (!data.user) {
      await syncURL();if(!current(epoch))return;
      const query = new URLSearchParams({ redirect: location.pathname + location.search, charKm: '1', corpKm: '1', delay: '0' });
      const login = await scope.json(`/auth/login?${query}`);
      if(!current(epoch))return;
      if(!isRecord(login) || typeof login.url !== 'string')throw new Error('Invalid login destination');
      const destination = new URL(login.url);
      if (destination.protocol !== 'https:') throw new Error('Invalid login destination');
      location.assign(destination.href);return;
    }
    if(reconcileSavedId) {
      const saved = readSavedEditorFit(await scope.json('/api/fit/'+encodeURIComponent(reconcileSavedId)));
      if(!current(epoch))return;
      if(saved.fit.fit_id !== reconcileSavedId)throw new Error('Invalid saved fitting');
      reconcileSavedId = null;
    }
    const dialog = $('[data-fit-save-dialog]');
    const form = $('[data-fit-save-form]');
    formField(form,'name').value = state.fit.name;
    formField(form,'description').value = state.fit.description;
    for(const radio of form.querySelectorAll<HTMLInputElement>('input[name="visibility"]'))radio.checked = radio.value === String(state.savedId ? visibility : 3);
    $('#fit-save-title').textContent = state.savedId ? 'Update Fit' : 'Save Fit';
    $('[data-fit-submit-label]').textContent = state.savedId ? 'Update' : 'Save';
    $('[data-fit-save-existing]').hidden = !state.savedId;
    $('[data-fit-save-id]').textContent = state.savedId?.slice(0, 8) ?? '';
    $('[data-fit-save-error]').hidden = !uncertainCreation;
    if(uncertainCreation)$('[data-fit-save-error]').textContent = 'The save could not be confirmed. Check your saved fits before creating another copy.';
    syncSaveAvailability();
    dialog.showModal();
    dialog.focus();
  }
  function syncSaveAvailability() {
    const form = $('[data-fit-save-form]'), submit = requiredElement<HTMLButtonElement>(form,'[type="submit"]');
    submit.disabled = !formField(form,'name').value.trim() || saving || uncertainCreation;
  }
  scope.on(formField($('[data-fit-save-form]'),'name'), 'input', syncSaveAvailability);
  scope.on($('[data-fit-save-form]'), 'submit', async (event) => {
    event.preventDefault();
    if (!scope.active() || !state.fit || detail && !owner || saving || uncertainCreation) return;
    const epoch = scope.epoch(),originalFit = state.fit,originalDocument = JSON.stringify(state.fit),savedId = state.savedId,documentId = documentGeneration;
    const form = event.currentTarget;
    const submit = requiredElement<HTMLButtonElement>(form,'[type="submit"]');
    const submitLabel = $('[data-fit-submit-label]');
    const errorBox = $('[data-fit-save-error]');
    const fit = snapshot(currentFit());
    fit.name = formField(form,'name').value.trim();
    fit.description = formField(form,'description').value;
    if (!fit.name) return;
    const body = saveBody(fit, Number(requiredElement<HTMLInputElement>(form,'input[name="visibility"]:checked').value));
    saving = true;
    submit.dataset.saving = 'true';
    syncSaveAvailability();
    submitLabel.textContent = 'Saving…';
    errorBox.hidden = true;
    try {
      if(reconcileSavedId) {
        const saved = readSavedEditorFit(await scope.json('/api/fit/'+encodeURIComponent(reconcileSavedId)));
        if(!current(epoch))return;
        if(saved.fit.fit_id !== reconcileSavedId)throw new Error('Invalid saved fitting');
        reconcileSavedId = null;
      }
      if(savedId)reconcileSavedId = savedId;else uncertainCreation = true;
      const data = await scope.json(savedId ? `/api/fit/${encodeURIComponent(savedId)}` : '/api/fit', {
        method: savedId ? 'PATCH' : 'POST',
        body: savedId ? {name:body.name,description:body.description,visibility:body.visibility,items:body.items} : body,
      });
      if(!current(epoch))return;
      const id = isRecord(data) ? data.fit_id ?? (isRecord(data.fit) ? data.fit.fit_id : undefined) ?? savedId : null;
      if(typeof id !== 'string' || !/^[a-zA-Z0-9]{14}$/.test(id))throw new Error('Invalid saved fitting response');
      reconcileSavedId = null;uncertainCreation = false;
      if(documentId === documentGeneration) {state.savedId = id;visibility = body.visibility;}
      // Saving an older snapshot must not replace edits made during the request.
      if(state.fit !== originalFit || JSON.stringify(state.fit) !== originalDocument) {toast('Fitting saved; newer changes remain unsaved');return;}
      state.savedId = id;state.fit = fit;
      visibility = body.visibility;
      const description = $('[data-fit-description]');
      if (description) { description.textContent = fit.description; description.title = fit.description; }
      render();
      syncURL();
      if(formField(form,'name').value.trim() === fit.name && formField(form,'description').value === fit.description && Number(requiredElement<HTMLInputElement>(form,'input[name="visibility"]:checked').value) === body.visibility) {
        $('[data-fit-save-dialog]').close();toast('Fitting saved');
      }else toast('Fitting saved; newer changes remain unsaved');
    } catch (error) {
      if(error instanceof BrowserRequestError && error.status >= 400 && error.status < 500) {uncertainCreation = false;reconcileSavedId = null;}
      if(!current(epoch) || isAbortError(error))return;
      errorBox.textContent = uncertainCreation ? 'The save could not be confirmed. Check your saved fits before creating another copy.' : errorMessage(error);
      errorBox.hidden = false;
    } finally { if(current(epoch)) {saving = false;delete submit.dataset.saving;syncSaveAvailability();submitLabel.textContent = state.savedId ? 'Update' : 'Save';} }

  });
  function createBlankFit() {
    if (detail) { location.assign('/fits/create'); return; }
    uncertainCreation = false;reconcileSavedId = null;
    documentGeneration++;state.generation++;state.priceGeneration++;state.customGeneration++;state.variantGeneration++;state.previewGeneration++;
    state.variantHull = null;state.variantsHTML = null;state.variantsData = null;state.variantsLoading = false;
    state.fit = null; state.stats = state.values = state.price = null;
    state.selected = null; state.undo = []; state.redo = []; state.savedId = null;
    state.customFit = state.customStats = state.customPrice = state.observedSnapshot = null;
    state.selectedFamily = null; state.showComparison = false; state.tab = 'hulls';
    state.search = ''; $('[data-fit-search]').value = '';
    const discard = $('[data-fit-discard-dialog]');
    if (discard.open) discard.close();
    render(); syncURL(); setStatus('');
  }
  scope.on($('[data-fit-rename-form]'), 'submit', (event) => {
    event.preventDefault();
    if (!state.fit) return;
    const name = formField(event.currentTarget,'name').value.trim();
    if (!name) return;
    const fit = snapshot(currentFit());
    fit.name = name.slice(0, 120).replaceAll(',', ' ');
    $('[data-fit-rename-dialog]').close();
    commit(fit);
  });
  scope.on(formField($('[data-fit-rename-form]'),'name'), 'input', (event) => { const eventTargetElement=eventElement(event.target);if(!eventTargetElement)return;
    $('[data-fit-rename-form] [type="submit"]').disabled = !elementValue(eventTargetElement).trim();
  });
  async function action(name:string) {
    const epoch = scope.epoch();
    if(!scope.active())return;
    switch (name) {
      case 'undo': historyMove(state.undo, state.redo); break;
      case 'redo': historyMove(state.redo, state.undo); break;
      case 'new':
        if (state.fit?.modules.length) $('[data-fit-discard-dialog]').showModal();
        else createBlankFit();
        break;
      case 'rename': {
        if (!state.fit) return;
        const dialog = $('[data-fit-rename-dialog]');
        const input = requiredElement<HTMLInputElement>($('[data-fit-rename-form]'),'[name="name"]');
        input.value = state.fit.name;
        $('[data-fit-rename-form] [type="submit"]').disabled = !input.value.trim();
        dialog.showModal();
        input.focus(); input.select();
        break;
      }
      case 'copy-eft': {
        const eft = state.fit && state.sde ? exportFitToEft(state.fit, state.sde) : null;
        if (eft) { await navigator.clipboard.writeText(eft); if(current(epoch))toast('EFT fitting copied'); }
        break;
      }
      case 'share': {
        if (detail) {
          const url = new URL('/fits/create', location.origin); url.searchParams.set('fit', await encodeFit(currentFit()));
          if(!current(epoch))return;
          await navigator.clipboard.writeText(url.href);
        } else { await syncURL(); if(!current(epoch))return;await navigator.clipboard.writeText(location.href); }
        if(current(epoch))toast('Share link copied'); break;
      }
      case 'fork': {
        if (!state.fit) break;
        const url = new URL('/fits/create', location.origin); url.searchParams.set('fit', await encodeFit(currentFit()));
        if(current(epoch))location.assign(url.href); break;
      }
      case 'save': await openSave(); break;
      case 'expand':
        state.expanded = !state.expanded;
        if (state.expanded) document.body.append(root);
        else originalParent.insertBefore(root, originalNextSibling);
        root.classList.toggle('expanded', state.expanded);
        document.body.style.overflow = state.expanded ? 'hidden' : originalOverflow;
        $('[data-fit-expand-label]').textContent = state.expanded ? 'Exit' : 'Expand';
        $('[data-fit-action="expand"]').setAttribute('aria-label', state.expanded ? 'Exit' : 'Expand');
        if (state.expanded && state.fit && !state.customFit) state.customFit = state.fit ? snapshot(currentFit()) : null;
        if (state.expanded) state.view = 'table';
        render(); syncURL(); saveWorkspacePrefs(); loadVariants();
        if (state.expanded && state.customFit && !state.customStats) calculateCustomFit();
        break;
      case 'cancel-save': $('[data-fit-save-dialog]').close(); break;
      case 'cancel-rename': $('[data-fit-rename-dialog]').close(); break;
      case 'cancel-discard': $('[data-fit-discard-dialog]').close(); break;
      case 'close-shortcuts': $('[data-fit-shortcuts]').hidden = true; break;
      case 'confirm-discard': createBlankFit(); break;
      case 'paste': {
        const input = await navigator.clipboard.readText().catch(()=> '');
        if(current(epoch))await importText(input);
        break;
      }
      case 'cycle-state': if (state.selected) cycleState(state.selected.type, state.selected.index); break;
      case 'remove-selected': if (state.selected) removeSlot(state.selected.type, state.selected.index); break;
    }
  }
  scope.on(root, 'contextmenu', (event) => { const eventTargetElement=eventElement(event.target);if(!eventTargetElement)return;
    const drone = eventTargetElement.closest<HTMLElement>('[data-fit-drone-row]');
    if (drone && root.contains(drone)) { openDroneContextMenu(event, Number(drone.dataset.fitDroneRow)); return; }
    const slot = eventTargetElement.closest<HTMLElement>('[data-fit-slot]');
    if (!slot || !root.contains(slot)) return;
    const [type, index] = (slot.dataset.fitSlot ?? '').split(':');
    openContextMenu(event, type, Number(index));
  });
  scope.on(contextMenu, 'click', (event) => { const eventTargetElement=eventElement(event.target);if(!eventTargetElement)return;
    const button = eventTargetElement.closest<HTMLElement>('[data-fit-context-action]');
    if (!button || !contextSlot) return;
    const [action, rawValue] = (button.dataset.fitContextAction ?? '').split(':');
    if ('droneId' in contextSlot && contextSlot.droneId !== undefined) {if(action.startsWith('drone-'))updateDroneGroup(contextSlot.droneId,action.slice(6));closeContextMenu();return}
    const {type,index,typeId,chargeTypeId}=contextSlot;
    if (action === 'state') setModuleState(type, index, rawValue);
    else if (action === 'rack-overload') toggleRackOverload(type);
    else if (action === 'fill-rack') fillHighRack(typeId, chargeTypeId);
    else if (action === 'charge' || action === 'charge-all') {
      const fit = snapshot(currentFit()), chargeId = Number(rawValue);
      const modules = action === 'charge' ? fit.modules.filter((module) => module.slot.type === type && module.slot.index === index) : fit.modules;
      let loaded = false;
      for (const module of modules) if (chargeCompatible(module.typeId, chargeId)) { module.charge = { typeId: chargeId }; loaded = true; }
      if (loaded) commit(fit);
    } else if (action === 'remove-charge') removeCharge(type, index);
    else if (action === 'unfit') removeSlot(type, index);
    closeContextMenu();
  });
  scope.on(root, 'pointerdown', (event) => { const eventTargetElement=eventElement(event.target);if(!eventTargetElement)return;
    const grip = eventTargetElement.closest<HTMLElement>('[data-fit-resize-col]');
    const table = grip?.closest<HTMLElement>('.fit-editor-table');
    if (!table || !grip) return;
    finishResize?.();
    const index = Number(grip.dataset.fitResizeCol);
    const headers = [...table.querySelectorAll<HTMLElement>('thead th')];
    const current = headers[index]?.getBoundingClientRect().width ?? 0;
    const next = headers[index + 1]?.getBoundingClientRect().width ?? 0;
    if (!current || !next) return;
    event.preventDefault();
    const startX = event.clientX;
    state.columnWidths = headers.map((header) => header.getBoundingClientRect().width);
    table.classList.add('has-resized-columns');
    const columns = [...table.querySelectorAll<HTMLElement>('colgroup col')];
    const applyWidths = () => columns.forEach((column, position) => { column.style.width = `${state.columnWidths[position]}px`; });
    applyWidths();
    const previousCursor = document.body.style.cursor, previousSelection = document.body.style.userSelect;
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
    const onMove = (move:PointerEvent) => {
      const delta = Math.max(tableColumnMinimums[index] - current, Math.min(move.clientX - startX, next - tableColumnMinimums[index + 1]));
      state.columnWidths[index] = current + delta;
      state.columnWidths[index + 1] = next - delta;
      applyWidths();
    };
    const onUp = () => {
      finishResize = undefined;
      scope.off(document, 'pointermove', onMove);
      scope.off(document, 'pointerup', onUp);
      document.body.style.cursor = previousCursor;
      document.body.style.userSelect = previousSelection;
      try { localStorage.setItem(tableColumnPrefsKey, JSON.stringify(state.columnWidths)); } catch {}
    };
    finishResize = onUp;
  scope.on(document, 'pointermove', onMove);
  scope.on(document, 'pointerup', onUp);
  });
  scope.on(document, 'mousedown', (event) => { const eventTargetElement=eventElement(event.target);if(!eventTargetElement)return;
    if (!contextMenu.hidden && !contextMenu.contains(eventTargetElement)) closeContextMenu();
    if (!eventTargetElement.closest<HTMLElement>('.fit-editor-table tr[data-fit-slot]')) tableFocusSlot = null;
  });
  scope.on(document, 'focusin', (event) => { const eventTargetElement=eventElement(event.target);if(!eventTargetElement)return; tableFocusSlot = eventTargetElement.closest<HTMLElement>('.fit-editor-table tr[data-fit-slot]')?.dataset.fitSlot ?? null; });
  scope.on($('[data-fit-stats]'), 'toggle', (event) => { const eventTargetElement=eventElement(event.target);if(!eventTargetElement)return;
    const section = eventTargetElement.closest<HTMLDetailsElement>('details[data-fit-stat]');
    if (!section) return;
    if(!section.dataset.fitStat)return;state.statExpanded.set(section.dataset.fitStat, section.open);
    try { localStorage.setItem(`ek-fit-stat-${section.dataset.fitStat}`, section.open ? '1' : '0'); } catch {}
  }, true);
  scope.on(root, 'pointerover', (event) => { const eventTargetElement=eventElement(event.target);if(!eventTargetElement)return;
    const button = eventTargetElement.closest<HTMLElement>('[data-fit-suggestion]');
    if (button && !(event.relatedTarget instanceof Node && button.contains(event.relatedTarget))) previewSuggestion(button);
    const catalogItem = eventTargetElement.closest<HTMLElement>('[data-fit-type]:not([data-fit-result-index])');
    if (catalogItem && !(event.relatedTarget instanceof Node && catalogItem.contains(event.relatedTarget))) previewFitItem(catalogItem, Number(catalogItem.dataset.fitType), slotFor(Number(catalogItem.dataset.fitType)));
  });
  scope.on(root, 'pointerout', (event) => { const eventTargetElement=eventElement(event.target);if(!eventTargetElement)return;
    const button = eventTargetElement.closest<HTMLElement>('[data-fit-suggestion]');
    if (button && !(event.relatedTarget instanceof Node && button.contains(event.relatedTarget)) && state.previewTarget === button) clearSuggestionPreview();
    const catalogItem = eventTargetElement.closest<HTMLElement>('[data-fit-type]:not([data-fit-result-index])');
    if (catalogItem && !(event.relatedTarget instanceof Node && catalogItem.contains(event.relatedTarget)) && state.previewTarget === catalogItem) clearSuggestionPreview();
  });
  scope.on(root, 'click', async (event) => { const eventTargetElement=eventElement(event.target);if(!eventTargetElement)return;
    const target = eventTargetElement.closest<HTMLElement>('[data-fit-slot-action], button, tr[data-fit-slot], .slotOuter[data-fit-slot], a[data-fit-action], a[data-fit-variant]');
    if (!target || !root.contains(target)) return;
    const data = target.dataset;
    if (data.fitAction) { if (target.tagName === 'A') event.preventDefault(); try { await action(data.fitAction); } catch (error) { console.error(error); toast('Action failed'); } }
    else if (data.fitSlotAction) {
      event.stopPropagation();
      const [kind, type, rawIndex] = data.fitSlotAction.split(':');
      const index = Number(rawIndex);
      if (kind === 'unfit') removeSlot(type, index);
      else if (kind === 'uncharge') removeCharge(type, index);
      else if (kind === 'offline') cycleState(type, index);
      else if (kind === 'fill-rack') { const module=state.fit?.modules.find(item=>item.slot.type===type&&item.slot.index===index); if(module)fillHighRack(module.typeId,module.charge?.typeId); }
    }
    else if (data.fitTab === 'hulls' || data.fitTab === 'hardware') { state.tab = data.fitTab; state.search = ''; state.catalogSelection = 0; pendingCatalogClick = null; $('[data-fit-search]').value = ''; render(); saveWorkspacePrefs(); }
    else if (data.fitView === 'ship' || data.fitView === 'table') { state.view = data.fitView; tableSelection=null; renderStage(); syncURL(); saveWorkspacePrefs(); }
    else if (data.fitHull) {
      const fit = emptyFit(Number(data.fitHull));
      if (detail) { const epoch = scope.epoch(),url = new URL('/fits/create', location.origin); url.searchParams.set('fit', await encodeFit(fit)); if(current(epoch))location.assign(url.href); return; }
      documentGeneration++;state.search = ''; $('[data-fit-search]').value = ''; state.selected = null; state.savedId = null; commit(fit);
    }
    else if (data.fitVariant) {
      event.preventDefault();
      if (!(target instanceof HTMLAnchorElement)) return;
      const encoded = new URL(target.href).searchParams.get('fit');
      const epoch = scope.epoch(),fit = await decodeFit(encoded ?? '');
      if(!current(epoch))return;
      if (!fit) { toast('Could not open observed fit'); return; }
      const family = state.variantsData?.families.find((item) => item.family_hash === data.fitVariant);
      state.selectedFamily = data.fitVariant;
      state.showComparison = true;
      state.selected = null;
      fit.name = `${nameOf(fit.shipTypeId)} Observed Fit #${data.fitVariantIndex}`;
      if (family) fit.description = `${family.total_uses} observed losses in the last ${state.variantsData?.window_days ?? 90} days.`;
      state.observedSnapshot = snapshot(fit);
      commit(fit);
    }
    else if (data.fitVariantCustom !== undefined) { if (state.customFit) { state.selectedFamily = null; state.observedSnapshot = null; state.showComparison = false; commit(snapshot(state.customFit)); } }
    else if (data.fitVariantAction) {
      if (data.fitVariantAction === 'toggle-comparison') { state.showComparison = !state.showComparison; renderVariants(); }
      else if (data.fitVariantAction === 'keep-custom' && state.fit) { state.customFit = state.fit ? snapshot(currentFit()) : null; state.selectedFamily = null; state.observedSnapshot = null; state.showComparison = false; calculateCustomFit(); renderVariants(); }
      else if (data.fitVariantAction === 'reset-preview' && state.observedSnapshot) commit(snapshot(state.observedSnapshot));
    }
    else if (data.fitSuggestion) { clearSuggestionPreview(); state.selected = null; addType(Number(data.fitSuggestion)); }
    else if (data.fitMode === 'modules' || data.fitMode === 'charges') { state.catalogMode = data.fitMode; state.catalogSelection = 0; pendingCatalogClick = null; renderCatalog(); }
    else if (isEditorSlot(data.fitFilterSlot) || data.fitFilterSlot === 'DroneBay' || data.fitFilterSlot === 'Charge') { if (state.filterSlots.has(data.fitFilterSlot)) state.filterSlots.delete(data.fitFilterSlot); else state.filterSlots.add(data.fitFilterSlot); renderCatalog(); }
    else if (data.fitHullRestriction !== undefined) { state.hullRestriction = !state.hullRestriction; renderCatalog(); }
    else if (data.fitType) {
      const id = Number(data.fitType), slot = slotFor(id), now = Date.now();
      if (event.shiftKey && slot === 'High') { pendingCatalogClick = null; fillHighRack(id); }
      else if (pendingCatalogClick?.id === id && now - pendingCatalogClick.time <= 400) { pendingCatalogClick = null; addType(id); }
      else pendingCatalogClick = { id, time: now };
    }
    else if (data.fitState) { const [type, index] = data.fitState.split(':'); cycleState(type, Number(index)); }
    else if (data.fitRemove) { const [type, index] = data.fitRemove.split(':'); removeSlot(type, Number(index)); }
    else if (data.fitSlot) { const [type, index] = data.fitSlot.split(':'); if (target.tagName === 'TR') selectTableRow(type, Number(index)); else if (target.classList.contains('slotOuter')) { if (target.dataset.hasitem === 'true') cycleState(type, Number(index), event.shiftKey); else if (target.tabIndex >= 0) selectSlot(type, Number(index)); } else selectSlot(type, Number(index)); }
    else if (data.fitBay === 'drones' || data.fitBay === 'cargo') { state.bay = state.bay === data.fitBay ? null : data.fitBay; renderResources(); saveWorkspacePrefs(); }
    else if (data.fitDroneState) {
      if (!state.fit) return;
      const [direction, rawID] = data.fitDroneState.split(':'), id = Number(rawID);
      const fit = snapshot(currentFit()), drone = fit.drones.find((item) => item.typeId === id);
      if (!drone) return;
      if (direction === 'activate') { if (!drone.states.Passive) return; if (!canActivateDrone(fit, id)) { flashBayReject('drones'); return; } drone.states.Passive--; drone.states.Active++; }
      else { if (!drone.states.Active) return; drone.states.Active--; drone.states.Passive++; }
      commit(fit);
    }
    else if (data.fitBayStep) {
      if (!state.fit) return;
      const [rawStep, rawID] = data.fitBayStep.split(':'), step = Number(rawStep), id = Number(rawID);
      const fit = snapshot(currentFit());
      const bayKind = target.closest<HTMLElement>('[data-fit-bay-panel]')?.dataset.fitBayPanel ?? state.bay;
      if (bayKind === 'drones') {
        const drone = fit.drones.find((item) => item.typeId === id);
        if (!drone) return;
        if (drone.states.Passive) drone.states.Passive--; else if (drone.states.Active) drone.states.Active--;
        if (!drone.states.Passive && !drone.states.Active) fit.drones = fit.drones.filter((item) => item !== drone);
      } else {
        if (step > 0 && cargoUsed() + (state.sde.types.get(id)?.volume ?? 0) > cargoCapacity()) { flashBayReject('cargo'); return; }
        const item = fit.cargo.find((entry) => entry.typeId === id);
        if (!item) return;
        item.quantity += step;
        if (item.quantity <= 0) fit.cargo = fit.cargo.filter((entry) => entry !== item);
      }
      commit(fit);
    }
    else if (data.fitRemoveBay) {
      const fit = snapshot(currentFit()), id = Number(data.fitRemoveBay);
      const bayKind = target.closest<HTMLElement>('[data-fit-bay-panel]')?.dataset.fitBayPanel ?? state.bay;
      if (bayKind === 'drones') fit.drones = fit.drones.filter((item) => item.typeId !== id);
      else fit.cargo = fit.cargo.filter((item) => item.typeId !== id);
      commit(fit);
    }
  });
  const stageRow = (type:FitSlotType, index:number) => root.querySelector<HTMLElement>(`.fit-editor-table tr[data-fit-slot="${type}:${index}"]`);
  function selectTableRow(type:string, index:number) {
    if(!isEditorSlot(type))return;
    state.selected = { type, index };
    tableSelection = {type,index};
    for (const row of root.querySelectorAll<HTMLElement>('.fit-editor-table tr[data-fit-slot]')) row.classList.toggle('selected', row.dataset.fitSlot === `${type}:${index}`);
    renderSelection();
    stageRow(type, index)?.focus({preventScroll:true});
  }
  scope.on(root, 'dblclick', (event) => { const eventTargetElement=eventElement(event.target);if(!eventTargetElement)return;
    const row = eventTargetElement.closest<HTMLElement>('.fit-editor-table tr[data-fit-slot]');
    if (!row) return;
    const [type, rawIndex] = (row.dataset.fitSlot ?? '').split(':'), index = Number(rawIndex);
    const module = state.fit?.modules.find((item) => item.slot.type === type && item.slot.index === index);
    if (module?.charge) removeCharge(type, index);
    else if (module) removeSlot(type, index);
  });
  scope.on(root, 'keydown', (event) => { const eventTargetElement=eventElement(event.target);if(!eventTargetElement)return;
    if(eventTargetElement.closest('[data-fit-tab]')) {moveTabFocus(event,true);return}
    const row = eventTargetElement.closest<HTMLElement>('.fit-editor-table tr[data-fit-slot]');
    if (!row) return;
    const [type, rawIndex] = (row.dataset.fitSlot ?? '').split(':'), index = Number(rawIndex);
    if (event.key === 'Delete' || event.key === 'Backspace') {
      event.preventDefault();
      const module = state.fit?.modules.find((item) => item.slot.type === type && item.slot.index === index);
      if (module?.charge) removeCharge(type, index);
      else if (module) removeSlot(type, index);
      if(isEditorSlot(type))stageRow(type, index)?.focus();
    } else if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault();
      const rows = [...root.querySelectorAll<HTMLElement>('.fit-editor-table tr[data-fit-slot]')];
      const next = rows[rows.indexOf(row) + (event.key === 'ArrowDown' ? 1 : -1)];
      if (next) { const [nextType, nextIndex] = (next.dataset.fitSlot ?? '').split(':'); selectTableRow(nextType, Number(nextIndex)); }
    }
  });
  scope.on($('[data-fit-catalog]'), 'toggle', (event) => { const eventTargetElement=eventElement(event.target);if(!eventTargetElement)return;
    const group = eventTargetElement.closest<HTMLDetailsElement>('details[data-fit-catalog-group]');
    if (!group?.open || group.querySelector<HTMLElement>('[data-fit-type]')) return;
    const entries = state.catalogGroups[Number(group.dataset.fitCatalogGroup)] ?? [];
    const tree:CatalogTree = {children:new Map(),items:[]};
    for (const entry of entries) {
      let node = tree;
      for (const label of entry.path.slice(1)) {
        let child = node.children.get(label);
        if (!child) { child = {children: new Map(), items: []}; node.children.set(label, child); }
        node = child;
      }
      node.items.push(entry);
    }
    const renderNode = (node:CatalogTree):string => `${[...node.children].sort(([a], [b]) => a.localeCompare(b)).map(([label, child]) => `<details class="fit-editor-market-group"><summary>${escape(label)}</summary>${renderNode(child)}</details>`).join('')}${node.items.sort((a, b) => a.meta - b.meta || a.name.localeCompare(b.name)).map((entry) => `<button type="button" class="fit-editor-catalog-item" data-fit-type="${entry.id}" draggable="true"><img src="${iconOf(entry.id)}" alt="" width="24" height="24" loading="lazy"><span>${escape(entry.name)}</span><b>+</b></button>`).join('')}`;
    requiredElement<HTMLElement>(group,'div').innerHTML = renderNode(tree);
  }, true);
  scope.on($('[data-fit-search]'), 'input', (event) => { const eventTargetElement=eventElement(event.target);if(!eventTargetElement)return; state.search = elementValue(eventTargetElement); state.catalogSelection = 0; pendingCatalogClick = null; renderCatalog(); });
  scope.on($('[data-fit-skill]'), 'change', (event) => { const eventTargetElement=eventElement(event.target);if(!eventTargetElement)return; const level=Number(elementValue(eventTargetElement));if(level !== 3 && level !== 4 && level !== 5)return;state.skillLevel = level; try { localStorage.setItem('ek-fit-skill-level-v1', String(state.skillLevel)); } catch {} calculate(); if (state.expanded && state.customFit) calculateCustomFit(); });
  scope.on($('[data-fit-search]'), 'keydown', (event) => {
    if (!state.search.trim()) return;
    const results = [...$('[data-fit-catalog]').querySelectorAll<HTMLElement>('[data-fit-result-index]')];
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault();
      selectCatalogResult(state.catalogSelection + (event.key === 'ArrowDown' ? 1 : -1));
    } else if (event.key === 'Enter') {
      event.preventDefault();
      const selected = results[state.catalogSelection];
      if (!selected) return;
      if (selected.dataset.fitHull) selected.click();
      else if (selected.dataset.fitType) {
        const id = Number(selected.dataset.fitType);
        if (event.shiftKey && slotFor(id) === 'High') fillHighRack(id);
        else addType(id);
      }
    }
  });
  scope.on(root, 'pointerover', (event) => { const eventTargetElement=eventElement(event.target);if(!eventTargetElement)return;
    const result = eventTargetElement.closest<HTMLElement>('[data-fit-result-index]');
    if (result && root.contains(result)) selectCatalogResult(Number(result.dataset.fitResultIndex));
  });
  scope.on(root, 'dragstart', (event) => { const eventTargetElement=eventElement(event.target);if(!eventTargetElement)return;
    if (!event.dataTransfer) return;
    const item = eventTargetElement.closest<HTMLElement>('[data-fit-type]');
    if (item) {
      const id = Number(item.dataset.fitType);
      event.dataTransfer.setData('application/esf-type-id', String(id));
      event.dataTransfer.setData('application/esf-slot-type', slotFor(id) ?? 'CargoBay');
      event.dataTransfer.effectAllowed = 'copy';
      const image = new Image();
      image.src = iconOf(id, 64);
      event.dataTransfer.setDragImage(image, 32, 32);
      return;
    }
    const slot = eventTargetElement.closest<HTMLElement>('.slotOuter[data-fit-slot]');
    if (!slot || !state.fit) return;
    const [type, index] = (slot.dataset.fitSlot ?? '').split(':');
    const module = currentFit().modules.find((entry) => entry.slot.type === type && entry.slot.index === Number(index));
    if (!module) return;
    event.dataTransfer.setData('application/esf-type-id', String(module.typeId));
    event.dataTransfer.setData('application/esf-slot-type', type);
    event.dataTransfer.setData('application/esf-slot-index', index);
  });
  const flashBayReject = (kind:BayKind) => {
    for (const element of root.querySelectorAll<HTMLElement>(`[data-fit-bay="${kind}"], [data-fit-bay-panel="${kind}"]`)) {
      element.classList.add('drop-rejected');
      scope.later(() => element.classList.remove('drop-rejected'), 1200);
    }
  };
  scope.on(root, 'dragover', (event) => { const eventTargetElement=eventElement(event.target);if(!eventTargetElement)return;
    const bay = eventTargetElement.closest<HTMLElement>('[data-fit-bay], [data-fit-bay-panel]');
    if (bay && event.dataTransfer) { event.preventDefault(); event.dataTransfer.dropEffect = 'copy'; bay.classList.add('drop-active'); }
    else if (eventTargetElement.closest<HTMLElement>('[data-fit-slot], [data-fit-hull-drop]')) event.preventDefault();
  });
  scope.on(root, 'dragleave', (event) => { const eventTargetElement=eventElement(event.target);if(!eventTargetElement)return;
    const bay = eventTargetElement.closest<HTMLElement>('[data-fit-bay], [data-fit-bay-panel]');
    if (bay && !(event.relatedTarget instanceof Node && bay.contains(event.relatedTarget))) bay.classList.remove('drop-active');
  });
  scope.on(root, 'drop', (event) => { const eventTargetElement=eventElement(event.target);if(!eventTargetElement)return;
    if (!event.dataTransfer) return;
    const slot = eventTargetElement.closest<HTMLElement>('[data-fit-slot]'), id = Number(event.dataTransfer.getData('application/esf-type-id'));
    if (eventTargetElement.closest<HTMLElement>('[data-fit-hull-drop]')) {
      event.preventDefault();
      // Match HullDraggable.vue: only fresh sidebar modules/drones auto-fit.
      // Existing fitted modules and charges must retain their slot behavior.
      const item = readHullDrop(event.dataTransfer);
      if (item) addType(item.typeId);
      return;
    }
    const bay = eventTargetElement.closest<HTMLElement>('[data-fit-bay], [data-fit-bay-panel]');
    if (bay && id) {
      event.preventDefault();
      bay.classList.remove('drop-active');
      const kind = bay.dataset.fitBay ?? bay.dataset.fitBayPanel;
      if (kind === 'drones') {
        if (slotFor(id) !== 'DroneBay') return;
        if (droneUsed() + (state.sde.types.get(id)?.volume ?? 0) > droneCapacity()) { if(kind === 'drones' || kind === 'cargo')flashBayReject(kind); return; }
        addType(id);
      } else {
        if (cargoUsed() + (state.sde.types.get(id)?.volume ?? 0) > cargoCapacity()) { if(kind === 'drones' || kind === 'cargo')flashBayReject(kind); return; }
        addCargoItem(id);
      }
      return;
    }
    if (!slot || !id || slot.getAttribute('aria-label')?.endsWith('Unavailable')) return;
    event.preventDefault();
    const [type, index] = (slot.dataset.fitSlot ?? '').split(':');
    const sourceType = event.dataTransfer.getData('application/esf-slot-type');
    if (sourceType !== 'Charge' && sourceType !== type) return;
    const sourceIndex = Number(event.dataTransfer.getData('application/esf-slot-index'));
    if (sourceIndex > 0 && sourceType === type && state.fit) {
      const fit = snapshot(currentFit());
      const a = fit.modules.find((entry) => entry.slot.type === type && entry.slot.index === sourceIndex);
      const b = fit.modules.find((entry) => entry.slot.type === type && entry.slot.index === Number(index));
      if (a) { a.slot.index = Number(index); if (b) b.slot.index = sourceIndex; commit(fit); }
      return;
    }
    if(!isEditorSlot(type))return;state.selected = {type,index:Number(index)};
    addType(id, sourceType === 'Charge', true);
  });
  const shortcutHelp = root.querySelector<HTMLElement>('[data-fit-shortcuts]');
  if(shortcutHelp) scope.on(shortcutHelp, 'click', (event) => { const eventTargetElement=eventElement(event.target);if(!eventTargetElement)return; if (eventTargetElement === shortcutHelp) shortcutHelp.hidden = true; });
  scope.on(document, 'keydown', (event) => { const eventTargetElement=eventElement(event.target);if(!eventTargetElement)return;
    if (!root.isConnected || detail) return;
    const search = $('[data-fit-search]');
    const typing = eventTargetElement.matches?.('input, textarea, [contenteditable]');
    const helpKey = event.key === '?' || event.shiftKey && event.key === '+' && ['Minus', 'Equal'].includes(event.code);
    if (!typing && helpKey && shortcutHelp) {
      event.preventDefault(); event.stopImmediatePropagation();
      shortcutHelp.hidden = !shortcutHelp.hidden;
      return;
    }
    if (event.key === 'Escape') {
      if (shortcutHelp && !shortcutHelp.hidden) { event.preventDefault(); shortcutHelp.hidden = true; }
      else if (!contextMenu.hidden) { event.preventDefault(); closeContextMenu(); }
      return;
    }
    if (state.expanded && eventTargetElement === search && event.key === 'Tab') {
      event.preventDefault();
      const sections = ['hulls','modules','charges'] as const;
      const current = state.tab === 'hulls' ? 'hulls' : state.catalogMode;
      const next = sections[(sections.indexOf(current) + (event.shiftKey ? 2 : 1)) % 3];
      state.tab = next === 'hulls' ? 'hulls' : 'hardware';
      if (next !== 'hulls') state.catalogMode = next;
      state.search = ''; state.catalogSelection = 0; pendingCatalogClick = null; search.value = '';
      render(); saveWorkspacePrefs(); search.focus();
      return;
    }
    if (state.expanded && !typing && (event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
      event.preventDefault(); event.stopImmediatePropagation();
      state.tab = state.fit ? 'hardware' : 'hulls';
      state.catalogMode = 'modules';
      state.search = ''; state.catalogSelection = 0; pendingCatalogClick = null; search.value = '';
      render(); saveWorkspacePrefs(); search.focus();
      return;
    }
    if (typing && eventTargetElement !== search) return;
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'z') { event.preventDefault(); action(event.shiftKey ? 'redo' : 'undo'); }
    else if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'f') { event.preventDefault(); search.focus(); }
  }, true);
  scope.observe($('.fit-editor-main'),syncWheelScale);
  scope.observe($('.fit-editor-grid'),syncWheelScale);
  scope.on(window, 'resize', syncWheelScale);
  async function start() {
    const epoch = scope.epoch();
    try {
      restoreWorkspacePrefs();
      try {
        const savedLevel = Number(localStorage.getItem('ek-fit-skill-level-v1'));
        if (savedLevel === 3 || savedLevel === 4 || savedLevel === 5) state.skillLevel = savedLevel;
        const savedWidths = JSON.parse(localStorage.getItem(tableColumnPrefsKey) ?? 'null');
        state.columnWidths = fitTableColumns(savedWidths);
        for (const key of ['capacitor', 'offense', 'ewar', 'remote', 'defense', 'targeting', 'navigation', 'drones']) {
          const saved = localStorage.getItem(`ek-fit-stat-${key}`);
          if (saved !== null) state.statExpanded.set(key, saved === '1');
        }
      } catch {}
      $('[data-fit-skill]').value = String(state.skillLevel);
      state.sde = scope.sde;
      state.effectIds = new Map([...state.sde.dogmaEffects].map(([id, effect]) => [effect.name, id]));
      const encoded = new URLSearchParams(location.search).get('fit');
      if (detail) {
        const data = readSavedEditorFit(JSON.parse($('#saved-fit-state').textContent));
        state.fit = savedFitToUI(data); state.savedId = data.fit.fit_id; visibility = data.fit.visibility;
        state.tab = 'hardware'; state.view = 'ship';
      } else if (encoded) {
        const fit = await decodeFit(encoded);
        if(!current(epoch))return;
        state.fit = fit;
        if (!state.fit) toast('This fitting link could not be opened');
        else state.tab = 'hardware';
      } else {
        // A fresh editor always starts at hull selection, even when the
        // previous workspace remembered the hardware sidebar.
        state.tab = 'hulls';
      }
      if (state.fit) enforcePropmodExclusivity(state.fit, state.sde);
      if (state.expanded) {
        document.body.append(root);
        root.classList.add('expanded');
        document.body.style.overflow = 'hidden';
        $('[data-fit-expand-label]').textContent = 'Exit';
        $('[data-fit-action="expand"]').setAttribute('aria-label', 'Exit');
        if (state.fit) state.customFit = state.fit ? snapshot(currentFit()) : null;
      }
      if(!current(epoch))return;
      started = true;
      setStatus('');
      render();
      if (state.fit) { calculate(); calculatePrice(); }
      if (state.expanded) { loadVariants(); if (state.customFit) calculateCustomFit(); }
    } catch (error) {
      if(!current(epoch) || isAbortError(error))return;
      console.error('Fitting data unavailable', error);
      setStatus('Fitting data unavailable');
      $('[data-fit-catalog]').innerHTML = '<p class="fit-editor-empty">The fitting data could not be loaded. Please refresh the page.</p>';
    }
  }
  async function importText(input:string) {
    const epoch = scope.epoch();
    if(!input) {toast('Could not read a fitting from the clipboard');return;}
    await startup;if(!current(epoch))return;
    const esfFit = await decodeFit(input);if(!current(epoch))return;
    const fit = esfFit ?? importFitFromEft(input,state.sde);
    if(!fit) {toast('Clipboard does not contain a recognised EFT or EVE-KILL fitting');return;}
    enforcePropmodExclusivity(fit,state.sde);documentGeneration++;state.savedId = null;commit(fit);toast(`Fitting imported from ${esfFit ? 'ESF' : 'EFT'}`);
  }
  function hide() {
    state.generation++;state.priceGeneration++;state.customGeneration++;state.urlGeneration++;state.variantGeneration++;state.previewGeneration++;
    state.previewTarget = null;state.variantsLoading = false;finishResize?.();closeContextMenu();
    for(const dialog of root.querySelectorAll<HTMLDialogElement>('dialog[open]'))dialog.close();
    saving = false;delete requiredElement<HTMLButtonElement>($('[data-fit-save-form]'),'[type="submit"]').dataset.saving;syncSaveAvailability();
    const shortcuts = root.querySelector<HTMLElement>('[data-fit-shortcuts]');if(shortcuts)shortcuts.hidden = true;
    document.body.style.overflow = originalOverflow;
  }
  function show() {
    if(!started) {startup = start();return;}
    if(state.expanded)document.body.style.overflow = 'hidden';
    render();syncURL();calculate();calculatePrice();
    if(state.expanded) {loadVariants();if(state.customFit)calculateCustomFit();}
  }
  function destroy() {hide();contextMenu.remove();}
  startup = start();
  return {ready:startup,action,importText,hide,show,destroy};
}

