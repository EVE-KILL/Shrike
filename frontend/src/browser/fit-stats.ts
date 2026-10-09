import type {ComputedItem} from '../../dogma/src/types'
import type {FitRenderState,FitRenderAssets,FitIcon} from './shared/fit-render'
export type FitStatsProps = FitRenderState & FitRenderAssets & {expanded:boolean;opened:ReadonlyMap<string,boolean>;uiIcon:FitIcon}
type EwarKind = 'web'|'scram'|'ecm'|'damp'|'painter'|'td'|'burst'|'hic'
/** Port of web/app/components/fit/stats/ShipStatistics.vue and its row components. */
export function renderFitStats({ fit, stats, sde, expanded, opened, uiIcon, iconOf, nameOf, escape }:FitStatsProps) {
  const attrId = (name:string) => sde?.attributeNameToId.get(name);
  const raw = (name:string, kind:'hull'|'char' = 'hull') => {
    if (!stats || !sde) return null;
    const id = attrId(name);
    if (id === undefined) return null;
    const attribute = (kind === 'char' ? stats.char : stats.hull)?.attributes.get(id);
    return attribute?.value ?? attribute?.base_value ?? sde.dogmaAttributes.get(id)?.defaultValue ?? 0;
  };
  const itemValue = (item:ComputedItem|null|undefined, name:string) => {
    const id = attrId(name);
    if (!item || id === undefined) return 0;
    const attribute = item.attributes?.get(id);
    return attribute?.value ?? attribute?.base_value ?? 0;
  };
  const format = (value:number|null|undefined, fixed = 0, divisor = 1, roundDown = false) => {
    if (value === null || value === undefined || !Number.isFinite(value)) return '?';
    const scale = 10 ** fixed;
    const divided = value / divisor;
    const rounded = (roundDown ? Math.floor(divided * scale) : Math.round(divided * scale)) / scale;
    return (Object.is(rounded, -0) ? 0 : rounded).toLocaleString('en', { minimumFractionDigits: fixed, maximumFractionDigits: fixed });
  };
  const display = (name:string, fixed = 0, divisor = 1, roundDown = false, kind:'hull'|'char' = 'hull') => format(raw(name, kind), fixed, divisor, roundDown);
  const img = (name:string, size = 20) => uiIcon(name, size);
  const symbol = (name:string) => {
    const paths:Record<string,string> = {
      target: '<circle cx="12" cy="12" r="9"/><circle cx="12" cy="12" r="4"/><path d="M12 1v3m0 16v3M1 12h3m16 0h3"/>',
      crosshair: '<circle cx="12" cy="12" r="8"/><circle cx="12" cy="12" r="2"/><path d="M12 2v3m0 14v3M2 12h3m14 0h3"/>',
      zap: '<path d="m13 2-9 11h7l-1 9 10-12h-7l1-8Z"/>',
      'zap-off': '<path d="m13 2-2 4M9 9l-5 4h7l-1 9 10-12h-6M2 2l20 20"/>',
      droplets: '<path d="M12 22a7 7 0 0 0 7-7c0-4-7-13-7-13S5 11 5 15a7 7 0 0 0 7 7Z"/>',
    };
    return `<svg class="fit-editor-lucide" viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${paths[name] ?? ''}</svg>`;
  };
  const cell = (content:string, title = '') => `<span class="fit-editor-stat-cell"${title ? ` title="${escape(title)}"` : ''}>${content}</span>`;
  const row = (...cells:string[]) => `<div class="fit-editor-stat-row">${cells.join('')}</div>`;
  const section = (key:string, heading:string, headline:string, content:string, defaultOpen = true, extra = '') => {
    const isOpen = opened.has(key) ? opened.get(key) : defaultOpen || expanded;
    return `<details data-fit-stat="${key}" ${isOpen ? 'open' : ''}><summary><span class="fit-editor-stat-chevron" aria-hidden="true">›</span><span class="fit-editor-stat-label">${heading}</span><span class="fit-editor-stat-headline ${extra}">${headline}</span></summary><div class="fit-editor-stat-body">${content}</div></details>`;
  };

  const capSeconds = raw('capacitorDepletesIn');
  const capacitorState = capSeconds === null || capSeconds < 0 ? 'Stable' : `Depletes in ${[Math.floor(capSeconds / 3600), Math.floor((capSeconds % 3600) / 60), Math.floor(capSeconds % 60)].map((part) => String(part).padStart(2, '0')).join(':')}`;
  const capacitor = section('capacitor', 'Capacitor', capacitorState,
    row(cell(`${display('capacitorCapacity', 1)} GJ / ${display('rechargeRate', 2, 1000)} s`)) +
    row(cell(`Δ ${display('capacitorPeakDelta', 1)} GJ/s <small>(${display('capacitorPeakDeltaPercentage', 1)}%)</small>`)),
    true, capacitorState === 'Stable' ? 'stable' : 'depleting');

  const effectIds = new Map([...sde.dogmaEffects].map(([id, effect]) => [effect.name, id]));
  const hasEffect = (typeId:number, effectName:string) => (sde.typeDogma.get(typeId)?.dogmaEffects ?? []).some((effect) => effect.effectID === effectIds.get(effectName));
  const primaryWeapon = stats?.items.find((item) => hasEffect(item.type_id, 'turretFitted')) ?? stats?.items.find((item) => hasEffect(item.type_id, 'launcherFitted'));
  const turret = primaryWeapon && hasEffect(primaryWeapon.type_id, 'turretFitted');
  let weaponMax = itemValue(primaryWeapon, 'maxRange');
  if (primaryWeapon && !turret && !weaponMax && primaryWeapon.charge) weaponMax = itemValue(primaryWeapon.charge, 'maxVelocity') * itemValue(primaryWeapon.charge, 'explosionDelay') / 1000;
  const weaponFalloff = itemValue(primaryWeapon, 'falloff');
  const weaponTracking = itemValue(primaryWeapon, 'trackingSpeed');
  const weaponRange = primaryWeapon ? cell(`${symbol('target')}<span>${format(weaponMax, 2, 1000)} km${turret && weaponFalloff > 0 ? ` + ${format(weaponFalloff, 2, 1000)} km` : ''}</span>`, turret ? 'Optimal + falloff' : 'Missile flight range') : cell('');
  const droneDamage = raw('droneDamagePerSecond') ?? 0;
  const offense = section('offense', 'Offense', `${display('damagePerSecondWithReload', 1)} dps`,
    row(cell(`${img('damage-dps')}<span>${display('damagePerSecondWithReload', 1)} dps <small>(${display('damagePerSecondWithoutReload', 1)} dps)</small></span>`, 'Damage per second with and without reload'), cell(`${img('damage-alpha')}<span>${display('damageAlpha')} HP</span>`, 'Volley damage')) +
    ((primaryWeapon || droneDamage > 0) ? row(weaponRange, droneDamage > 0 ? cell(`${img('fitting-drones')}<span>${display('droneDamagePerSecond', 1)} dps</span>`, 'Drone DPS') : cell('')) : '') +
    (primaryWeapon && turret && weaponTracking > 0 ? row(cell(`${symbol('crosshair')}<span>${format(weaponTracking, 4)} rad/s</span>`, 'Tracking speed'), cell('')) : ''));

  const ewarKinds = new Map<number,EwarKind>([[65, 'web'], [52, 'scram'], [201, 'ecm'], [208, 'damp'], [211, 'painter'], [291, 'td'], [643, 'burst'], [899, 'hic']]);
  const groupedEwar = new Map<number,{kind:EwarKind;items:ComputedItem[]}>();
  for (const item of stats?.items ?? []) {
    const kind = ewarKinds.get(sde.types.get(item.type_id)?.groupID ?? 0);
    if (!kind) continue;
    if (!groupedEwar.has(item.type_id)) groupedEwar.set(item.type_id, { kind, items: [] });
    groupedEwar.get(item.type_id)!.items.push(item);
  }
  const percent = (number:number) => `${number > 0 ? '+' : ''}${format(number, 0)}%`;
  const ewarEntries = [...groupedEwar].map(([typeId, group]) => {
    const first = group.items[0], kind = group.kind;
    const optimal = itemValue(first, 'maxRange'), falloff = itemValue(first, 'falloff');
    const range = optimal > 0 ? `${format(optimal, optimal >= 10000 ? 0 : 1, 1000)}${falloff > 0 ? `+${format(falloff, falloff >= 10000 ? 0 : 1, 1000)}` : ''} km` : '';
    let effect = '';
    if (kind === 'web') { const n = itemValue(first, 'speedFactor'); if (n) effect = percent(n); }
    if (kind === 'scram' || kind === 'hic') { const n = itemValue(first, 'warpScrambleStrength'); effect = n > 0 ? `+${format(n)} str` : kind === 'hic' ? 'bubble' : ''; }
    if (kind === 'ecm') { const n = Math.max(...['scanGravimetricStrengthBonus', 'scanLadarStrengthBonus', 'scanMagnetometricStrengthBonus', 'scanRadarStrengthBonus'].map((name) => itemValue(first, name))); if (n > 0) effect = `${format(n, 1)} jam str`; }
    if (kind === 'damp') effect = [['scanResolutionBonus', 'res'], ['maxTargetRangeBonus', 'lock']].map(([name, label]) => itemValue(first, name) ? `${percent(itemValue(first, name))} ${label}` : '').filter(Boolean).join(' / ');
    if (kind === 'painter') { const n = itemValue(first, 'signatureRadiusBonus'); if (n) effect = `${percent(n)} sig`; }
    if (kind === 'td') effect = [['trackingSpeedBonus', 'trk'], ['maxRangeBonus', 'opt']].map(([name, label]) => itemValue(first, name) ? `${percent(itemValue(first, name))} ${label}` : '').filter(Boolean).join(' / ');
    if (kind === 'burst') effect = 'AoE';
    return { typeId, kind, count: group.items.length, range, effect, name: nameOf(typeId) };
  }).sort((a, b) => ['web', 'scram', 'hic', 'ecm', 'damp', 'painter', 'td', 'burst'].indexOf(a.kind) - ['web', 'scram', 'hic', 'ecm', 'damp', 'painter', 'td', 'burst'].indexOf(b.kind) || a.name.localeCompare(b.name));
  const ewar = ewarEntries.length ? section('ewar', 'EWar', `${ewarEntries.length} module${ewarEntries.length === 1 ? '' : 's'}`, ewarEntries.map((entry) => row(cell(`<img src="${iconOf(entry.typeId)}" width="20" height="20" alt=""><span class="fit-editor-stat-truncate">${escape(entry.name)}${entry.count > 1 ? ` <small>×${entry.count}</small>` : ''}</span>`, entry.name), cell(`<span>${entry.effect}</span><span>${entry.range}</span>`))).join('')) : '';

  const remoteNames = [['remoteShieldBoostRate', 'shield-boost-rate', 'hp/s', 'Remote Shield Boost'], ['remoteArmorRepairRate', 'armor-repair-rate', 'hp/s', 'Remote Armor Repair'], ['remoteHullRepairRate', 'hull-repair-rate', 'hp/s', 'Remote Hull Repair'], ['remoteCapTransferRate', 'zap', 'GJ/s', 'Remote Capacitor Transfer'], ['energyNeutralizerRate', 'zap-off', 'GJ/s', 'Energy Neutralizer Drain'], ['energyNosferatuRate', 'droplets', 'GJ/s', 'Energy Nosferatu Drain']];
  const remoteParts = remoteNames.filter(([name]) => (raw(name) ?? 0) > 0);
  const remote = remoteParts.length ? section('remote', 'Remote', 'Assistance / Warfare', `<div class="fit-editor-stat-wrap">${remoteParts.map(([name, icon, unit, title]) => cell(`${icon.includes('-rate') ? img(icon, 18) : symbol(icon)}<span>${display(name, 1)} ${unit}</span>`, title)).join('')}</div>`) : '';

  const tankNames = [['shieldBoostRate', 'shieldEffectiveBoostRate', 'shield-boost-rate', 'Shield Boost Rate'], ['armorRepairRate', 'armorEffectiveRepairRate', 'armor-repair-rate', 'Armor Repair Rate'], ['hullRepairRate', 'hullEffectiveRepairRate', 'hull-repair-rate', 'Hull Repair Rate']];
  const tank = `<div class="fit-editor-stat-wrap">${tankNames.filter(([name]) => (raw(name) ?? 0) > 0).map(([name, effective, icon, title]) => cell(`${img(icon, 18)}<span>${display(effective, 1)} ehp/s <small>(${display(name, 1)} hp/s)</small></span>`, title)).join('')}${cell(`${img('passive-shield-recharge', 18)}<span>${display('passiveShieldEffectiveRechargeRate', 1)} ehp/s <small>(${display('passiveShieldRechargeRate', 1)} hp/s)</small></span>`, 'Passive Shield Recharge')}</div>`;
  const hp = row(cell(`${img('shield-hp')}<span class="fit-editor-stat-stack">${display('shieldCapacity', 0, 1, true)} hp<small>${display('shieldRechargeRate', 0, 1000, true)}s recharge</small></span>`, 'Shield HP'), cell(`${img('armor-hp')}<span>${display('armorHP', 0, 1, true)} hp</span>`, 'Armor HP'), cell(`${img('hull-hp')}<span>${display('hp', 0, 1, true)} hp</span>`, 'Hull HP'));
  const resistTypes = [['Em', 'em-resistance', '#195e8c'], ['Thermal', 'thermal-resistance', '#8c1919'], ['Kinetic', 'kinetic-resistance', '#727272'], ['Explosive', 'explosive-resistance', '#8c5e19']];
  const resistHeader = `<div class="fit-editor-resist-grid"><span></span>${resistTypes.map(([, icon]) => img(icon, 18)).join('')}</div>`;
  const resistRow = (icon:string, prefix:string) => `<div class="fit-editor-resist-grid">${img(icon, 18)}${resistTypes.map(([damage, , color]) => {
    const name = prefix ? `${prefix}${damage}DamageResonance` : `${damage[0].toLowerCase() + damage.slice(1)}DamageResonance`;
    const resonance = raw(name);
    const resistance = resonance === null ? null : Math.max(0, Math.ceil((100 - resonance * 100 - .01) * 10) / 10);
    return `<span class="fit-editor-resist-bar"><i style="width:${resistance === null ? 0 : Math.min(100, resistance)}%;background:${color}"></i><b>${format(resistance, 1)}%</b></span>`;
  }).join('')}</div>`;
  const defense = section('defense', 'Defense', `${display('ehp', 0, 1, true)} ehp`, tank + hp + `<div class="fit-editor-resistances">${resistHeader}${resistRow('shield-hp', 'shield')}${resistRow('armor-hp', 'armor')}${resistRow('hull-hp', '')}</div>`);

  const targeting = section('targeting', 'Targeting', `${display('maxTargetRange', 2, 1000)} km`,
    row(cell(`${img('sensor-strength')}<span>${display('scanStrength', 2)} points</span>`, 'Scan Strength'), cell(`${img('scan-resolution')}<span>${display('scanResolution')} mm</span>`, 'Scan Resolution')) +
    row(cell(`${img('signature-radius')}<span>${display('signatureRadius')} m</span>`, 'Signature Radius'), cell(`${img('maximum-locked-targets')}<span>${display('maxLockedTargets')}x</span>`, 'Maximum Locked Targets')), false);
  const isStructure = fit ? sde.types.get(fit.shipTypeId)?.categoryID === 65 : false;
  const navigation = isStructure ? '' : section('navigation', 'Navigation', `${display('maxVelocity', 1)} m/s`,
    row(cell(`${img('mass')}<span>${display('mass', 2, 1000)} t</span>`, 'Mass'), cell(`${img('inertia-modifier')}<span>${display('agility', 4)}x</span>`, 'Inertia Modifier')) +
    row(cell(`${img('warp-speed')}<span>${display('warpSpeedMultiplier', 2)} AU/s</span>`, 'Ship Warp Speed'), cell(`${img('align-time')}<span>${display('alignTime', 2)}s</span>`, 'Align Time')), false);
  const drones = isStructure ? '' : section('drones', 'Drones', `${display('droneDamagePerSecond', 1)} dps`,
    row(cell(`${img('mass')}<span>${display('droneBandwidthLoad')} / ${display('droneBandwidth')} Mbit/sec</span>`, 'Drone Bandwidth'), cell(`${img('inertia-modifier')}<span>${display('droneControlDistance', 2, 1000, false, 'char')} km</span>`, 'Drone Control Range')) +
    row(cell(`${display('droneActive')} Active`)), false);

  return capacitor + (((raw('damagePerSecondWithReload') ?? 0) > 0 || !remoteParts.length) ? offense : '') + ewar + remote + defense + targeting + navigation + drones;
}
