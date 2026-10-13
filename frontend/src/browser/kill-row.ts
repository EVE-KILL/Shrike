import type {KilllistRow} from '../../shared/utils/killlistRow'
import {isRecord} from './shared/json'
type EntityKind='character'|'corporation'|'alliance'|'item'|'system'|'kill'|'group'|'region'|''
// Shared browser row renderer extracted from the existing homepage implementation.
  const safeID = (value:unknown) => {
    const id = Number(value);
    return Number.isSafeInteger(id) && id > 0 ? id : 0;
  };
  const compact = (value:unknown) => {
    const number = Number(value) || 0;
    if (number >= 1e12) return `${(number / 1e12).toFixed(2)}t`;
    if (number >= 1e9) return `${(number / 1e9).toFixed(2)}b`;
    if (number >= 1e6) return `${(number / 1e6).toFixed(2)}m`;
    if (number >= 1e3) return `${(number / 1e3).toFixed(1)}k`;
    return number.toFixed(0);
  };
  const nameOrUnknown = (value:string|null|undefined) => value || 'Unknown';
  const pathFor = (kind:EntityKind, value:unknown) => {
    const id = safeID(value);
    return id ? `/${kind}/${id}` : '#';
  };
  const imageFor = (kind:EntityKind, value:unknown, size = 64) => {
    const id = safeID(value);
    if (!id) return '';
    if (kind === 'character') return `https://images.evetech.net/characters/${id}/portrait?size=${size}`;
    if (kind === 'corporation') return `https://images.evetech.net/corporations/${id}/logo?size=${size}`;
    if (kind === 'alliance') return `https://images.evetech.net/alliances/${id}/logo?size=${size}`;
    if (kind === 'item') return `/images/types/${id}/icon?size=${size}`;
    if (kind === 'system') return `/images/systems/${id}?size=${size}`;
    return '';
  };
  function element<K extends keyof HTMLElementTagNameMap>(tag:K, className='', value?:string|number):HTMLElementTagNameMap[K] {
    const item = document.createElement(tag);
    if (className) item.className = className;
    if (value !== undefined) item.textContent = String(value);
    return item;
  }
  function entityLink(kind:EntityKind, id:unknown, name:string|null|undefined, className = '') {
    const valid = safeID(id);
    const item = element(valid ? 'a' : 'span', className, nameOrUnknown(name));
    if (item instanceof HTMLAnchorElement) item.href = pathFor(kind, valid);
    return item;
  }
  function portrait(kind:EntityKind, id:unknown, name:string|null|undefined, destination?:string, size = 64) {
    const valid = safeID(id);
    const item = element(valid ? 'a' : 'span', 'kill-portrait');
    if (valid) {
      if(item instanceof HTMLAnchorElement)item.href = destination || pathFor(kind, valid);
      const image = document.createElement('img');
      image.src = imageFor(kind, valid, size);
      image.alt = name || '';
      image.width = image.height = 40;
      image.loading = 'lazy';
      item.append(image);
    } else {
      item.classList.add('kill-image-empty');
      const svg=document.createElementNS('http://www.w3.org/2000/svg','svg');
      svg.setAttribute('class','icon');svg.setAttribute('aria-hidden','true');svg.setAttribute('viewBox','0 0 24 24');
      const use=document.createElementNS(svg.namespaceURI,'use');use.setAttribute('href','/assets/icons.svg#'+(name==='NPC'?'crosshair':'image'));svg.append(use);item.append(svg);
    }
    return item;
  }
  function identityCell(kill:KilllistRow, side:'victim'|'final') {
    const victim = side === 'victim';
    const charID = safeID(kill[victim ? 'victim_character_id' : 'final_blow_character_id']);
    const corpID = safeID(kill[victim ? 'victim_corporation_id' : 'final_blow_corporation_id']);
    const allianceID = safeID(kill[victim ? 'victim_alliance_id' : 'final_blow_alliance_id']);
    const shipID = safeID(kill.final_blow_ship_type_id);
    const charName = kill[victim ? 'victim_character_name' : 'final_blow_character_name'];
    const corpName = kill[victim ? 'victim_corporation_name' : 'final_blow_corporation_name'];
    const allianceName = kill[victim ? 'victim_alliance_name' : 'final_blow_alliance_name'];
    const primaryKind = charID ? 'character' : corpID ? 'corporation' : !victim && shipID ? 'item' : '';
    const primaryID = charID || corpID || (!victim && shipID) || 0;
    const primaryName = charID ? charName : corpID ? corpName : !victim && shipID ? kill.final_blow_ship_name : victim ? 'Unknown' : 'NPC';
    const cell = element('div', 'kill-identity');
    cell.append(portrait(primaryKind, primaryID, primaryName));
    if (corpID || allianceID) {
      const badges = element('div', 'kill-identity-badges');
      for (const [kind, id, name] of [['corporation', corpID, corpName], ['alliance', allianceID, allianceName]] as const) {
        if (!id) continue;
        const link = entityLink(kind, id, name);
        link.setAttribute('aria-label', `${kind}: ${nameOrUnknown(name)}`);
        link.textContent = '';
        const image = document.createElement('img');
        image.src = imageFor(kind, id, 32); image.alt = '';
        image.width = image.height = 19; image.loading = 'lazy';
        link.append(image); badges.append(link);
      }
      cell.append(badges);
    }
    const copy = element('div', 'kill-identity-copy');
    copy.append(entityLink(primaryKind, primaryID, primaryName,
      !victim && !charID && !corpID ? 'identity-primary npc' : 'identity-primary'));
    if (charID && corpID) copy.append(entityLink('corporation', corpID, corpName));
    if (allianceID) copy.append(entityLink('alliance', allianceID, allianceName));
    cell.append(copy);
    return cell;
  }
  function securityParts(value:number|null|undefined) {
    if (value === null || value === undefined) return { label: '?', className: 'security-unknown' };
    const number = Number(value);
    return { label: number.toFixed(1), className: number >= 0.5 ? 'security-high'
      : number > 0 ? 'security-low' : 'security-null' };
  }
  function liveRow(kill:KilllistRow) {
    const id = safeID(kill.killmail_id);
    const date = new Date(kill.killmail_time);
    if (!id || Number.isNaN(date.getTime())) return null;
    const row = element('article', 'kill-row');
    row.dataset.killId = String(id);
    row.dataset.victimCharacterId=String(safeID(kill.victim_character_id));
    row.dataset.victimCorporationId=String(safeID(kill.victim_corporation_id));
    row.dataset.victimAllianceId=String(safeID(kill.victim_alliance_id));
    const overlay = element('a', 'kill-row-overlay');
    overlay.href = pathFor('kill', id);
    overlay.setAttribute('aria-label', `Killmail: ${nameOrUnknown(kill.ship_name)} — ${compact(kill.total_value)} ISK`);
    row.append(overlay);

    const ship = element('div', 'kill-ship');
    ship.append(portrait('item', kill.ship_type_id, kill.ship_name, pathFor('kill', id)));
    const shipCopy = element('div', 'kill-ship-copy');
    shipCopy.append(entityLink('item', kill.ship_type_id, kill.ship_name));
    shipCopy.append(safeID(kill.ship_group_id)?entityLink('group',kill.ship_group_id,kill.ship_group_name||'', 'kill-ship-group'):element('span','kill-ship-group',kill.ship_group_name||''));
    shipCopy.append(element('span', 'kill-ship-isk', `${compact(kill.total_value)} ISK`));
    shipCopy.append(element('span', 'kill-mobile-summary', kill.victim_character_name || kill.victim_corporation_name || 'Unknown'));
    const mobileLocation = element('span', 'kill-mobile-location');
    const security = securityParts(kill.solar_system_security);
    mobileLocation.append(element('em', security.className, security.label));
    mobileLocation.append(document.createTextNode(` ${kill.solar_system_name || ''}`));
    shipCopy.append(mobileLocation); ship.append(shipCopy); row.append(ship);
    row.append(identityCell(kill, 'victim'), identityCell(kill, 'final'));

    const locationCell = element('div', 'kill-location');
    if (safeID(kill.solar_system_id)) {
      const icon = entityLink('system', kill.solar_system_id, '', 'kill-location-icon');
      icon.textContent = '';
      const image = document.createElement('img');
      image.src = imageFor('system', kill.solar_system_id, 32); image.alt = '';
      image.width = image.height = 24; image.loading = 'lazy';
      icon.append(image); locationCell.append(icon);
    }
    const locationCopy = element('div');
    const topLine = element('span');
    topLine.append(element('em', security.className, security.label));
    topLine.append(document.createTextNode(' '));
    topLine.append(entityLink('system', kill.solar_system_id, kill.solar_system_name || ''));
    locationCopy.append(topLine);
    if (safeID(kill.region_id)) locationCopy.append(entityLink('region', kill.region_id, kill.region_name, 'kill-region'));
    locationCell.append(locationCopy); row.append(locationCell);

    const details = element('div', 'kill-details');
    details.append(element('span', 'kill-mobile-isk', compact(kill.total_value)));
    const time = element('time'); time.dateTime = date.toISOString(); details.append(time);
    const count = element('span', 'kill-attacker-count');
    count.setAttribute('aria-label', `${kill.attacker_count ?? 0} involved`);
    const icon = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    icon.setAttribute('class', 'icon'); icon.setAttribute('aria-hidden', 'true');
    const use = document.createElementNS('http://www.w3.org/2000/svg', 'use');
    use.setAttribute('href', '/assets/icons.svg#users');
    icon.append(use); count.append(icon, element('span', '', kill.attacker_count ?? 0));
    details.append(count); row.append(details);
    return row;
  }

  function friendlyDay(value:string) {
    const date = new Date(value);
    const day = date.getUTCDate();
    const suffix = day % 100 >= 11 && day % 100 <= 13 ? 'th'
      : (day%10===1?'st':day%10===2?'nd':day%10===3?'rd':'th');
    const month = new Intl.DateTimeFormat('en-GB', { month: 'long', timeZone: 'UTC' }).format(date);
    const now = new Date();
    const dateKey = date.toISOString().slice(0, 10);
    const today = now.toISOString().slice(0, 10);
    const yesterday = new Date(now.getTime() - 86400000).toISOString().slice(0, 10);
    const label = `${day}${suffix} ${month}${date.getUTCFullYear() === now.getUTCFullYear() ? '' : ` ${date.getUTCFullYear()}`}`;
    return dateKey === today ? `Today (${label})` : dateKey === yesterday ? `Yesterday (${label})` : label;
  }


export {safeID,element,liveRow,friendlyDay};

// Relay frames enter as unknown. Normalize the shared REST row fields without
// granting a raw websocket object a trusted application type.
export function parseLiveKill(raw:unknown):KilllistRow|null {
  if(!isRecord(raw)||!safeID(raw.killmail_id)||typeof raw.killmail_time!=='string'||!Number.isFinite(Date.parse(raw.killmail_time)))return null;
  const number=(value:unknown)=>typeof value==='number'&&Number.isFinite(value)?value:null;
  const string=(value:unknown)=>typeof value==='string'?value:null;
  return {
    killmail_id:safeID(raw.killmail_id),killmail_time:raw.killmail_time,killmail_hash:string(raw.killmail_hash)??'',
    attacker_count:number(raw.attacker_count)??0,total_value:number(raw.total_value)??0,
    is_npc:raw.is_npc===true,is_solo:raw.is_solo===true,
    final_blow_alliance_id:number(raw.final_blow_alliance_id),
    final_blow_character_id:number(raw.final_blow_character_id),
    final_blow_corporation_id:number(raw.final_blow_corporation_id),
    final_blow_ship_type_id:number(raw.final_blow_ship_type_id),
    meta_group_id:number(raw.meta_group_id),
    region_id:number(raw.region_id),
    ship_group_id:number(raw.ship_group_id),
    ship_type_id:number(raw.ship_type_id),
    solar_system_security:number(raw.solar_system_security),
    victim_alliance_id:number(raw.victim_alliance_id),
    victim_character_id:number(raw.victim_character_id),
    victim_corporation_id:number(raw.victim_corporation_id),
    victim_faction_id:number(raw.victim_faction_id),
    solar_system_id:number(raw.solar_system_id)??0,
    final_blow_alliance_name:string(raw.final_blow_alliance_name),
    final_blow_character_name:string(raw.final_blow_character_name),
    final_blow_corporation_name:string(raw.final_blow_corporation_name),
    final_blow_ship_name:string(raw.final_blow_ship_name),
    region_name:string(raw.region_name),
    ship_group_name:string(raw.ship_group_name),
    ship_market_path:string(raw.ship_market_path),
    ship_name:string(raw.ship_name),
    solar_system_name:string(raw.solar_system_name),
    victim_alliance_name:string(raw.victim_alliance_name),
    victim_character_name:string(raw.victim_character_name),
    victim_corporation_name:string(raw.victim_corporation_name),
  };
}
