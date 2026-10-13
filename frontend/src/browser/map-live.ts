import type {GoMapController,MapScopeData} from '../utils/map/goFrontendCanvas'
import type {KilllistRow} from '../../shared/utils/killlistRow'
import {createRelayConnection} from '../utils/relaySocketCore'
import {parseLiveKill} from './kill-row'
import {requiredElement,placeElement} from './shared/dom'
import {isRecord,isAbortError} from './shared/json'
import {requestJSON} from './browser-request'
export type MapFeedScope = MapScopeData & {scope?:string;kills?:KilllistRow[];anchors?:Array<{solar_system_id:number;system_name:string}>}
export type MapFeedRenderer = Pick<GoMapController,'pulseKill'|'focusLiveSystem'>
export type MapAlarmBand = 'near'|'outer'
// Live and AIID maps share PixiScopeView's relay, 60-row de-duplication,
// freshness gate, and kill-row anatomy. The map renderer owns pulse animation.
export function createMapLiveFeed(root:HTMLElement, renderer:MapFeedRenderer, scopeData:MapFeedScope) {
  return createMapKillFeed(root, renderer, scopeData, 'live');
}

export function createMapAIIDFeed(root:HTMLElement, renderer:MapFeedRenderer, scopeData:MapFeedScope, onAlarm?:(band:MapAlarmBand)=>void) {
  return createMapKillFeed(root, renderer, scopeData, 'aiid', onAlarm);
}

function createMapKillFeed(root:HTMLElement, renderer:MapFeedRenderer, scopeData:MapFeedScope, mode:'live'|'aiid', onAlarm?:(band:MapAlarmBand)=>void) {
  const panel = requiredElement<HTMLElement>(root,`[data-map-${mode}]`);
  const list = requiredElement<HTMLElement>(root,`[data-map-${mode}-list]`);
  const connection = requiredElement<HTMLElement>(root,`[data-map-${mode}-connection]`);
  const count = requiredElement<HTMLElement>(root,`[data-map-${mode}-count]`);
  const status = requiredElement<HTMLElement>(root,`[data-map-${mode}-status]`);
  const indicator = requiredElement<HTMLElement>(root,`[data-map-${mode}-indicator]`);
  const recent = requiredElement<HTMLElement>(root,`[data-map-${mode}-recent]`);
  const surface = requiredElement<HTMLElement>(root,'[data-map-surface]');
  const systems = new Set((scopeData.systems || []).map(system => system.solar_system_id));
  const bySystem = new Map((scopeData.systems || []).map(system => [system.solar_system_id, system]));
  const request = new AbortController();
  const seenFresh = new Map<number,number>();
  const rows = new Map<number,HTMLAnchorElement>();
  let bootstrap:KilllistRow[] = mode === 'aiid' ? scopeData.kills || [] : [], streamed:KilllistRow[] = [];
  let disposed = false;

  const formatIsk = (value:number|null) => {
    const number = Number(value) || 0;
    if (number >= 1e12) return `${(number / 1e12).toFixed(2)}t`;
    if (number >= 1e9) return `${(number / 1e9).toFixed(2)}b`;
    if (number >= 1e6) return `${(number / 1e6).toFixed(2)}m`;
    if (number >= 1e3) return `${(number / 1e3).toFixed(1)}k`;
    return number.toFixed(0);
  };
  const age = (timestamp:string) => {
    const seconds = Math.max(0, Math.floor((Date.now() - Date.parse(timestamp)) / 1000)) || 0;
    if (seconds < 60) return `${seconds}s ago`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    return `${Math.floor(seconds / 3600)}h ago`;
  };
  const appendSpan = (parent:HTMLElement, value:string) => {
    const span = document.createElement('span');
    span.textContent = value;
    parent.append(span);
  };
  const currentKills = () => {
    const seen = new Set<number>(), kills:KilllistRow[] = [];
    for (const kill of [...streamed, ...bootstrap]) {
      if (!kill?.killmail_id || seen.has(kill.killmail_id) || !systems.has(kill.solar_system_id)) continue;
      seen.add(kill.killmail_id);
      kills.push(kill);
      if (kills.length >= 60) break;
    }
    return kills;
  };
  const rowFor = (kill:KilllistRow) => {
    const row = document.createElement('a');
    row.className = 'map-live-row';
    row.href = `/kill/${kill.killmail_id}`;
    row.dataset.killId = String(kill.killmail_id);
    row.dataset.systemId = String(kill.solar_system_id);
    if (kill.ship_type_id) {
      const image = document.createElement('img');
      image.src = `/images/types/${kill.ship_type_id}/icon?size=64`;
      image.alt = kill.ship_name || '';
      image.loading = 'lazy';
      image.addEventListener('error', () => {
        const fallback = document.createElement('span');
        fallback.className = 'map-live-image-fallback'; fallback.textContent = '?';
        image.replaceWith(fallback);
      }, { once: true });
      row.append(image);
    } else {
      const fallback = document.createElement('span');
      fallback.className = 'map-live-image-fallback'; fallback.textContent = '?';
      row.append(fallback);
    }
    const ident = document.createElement('span'); ident.className = 'map-live-ident';
    appendSpan(ident, kill.ship_name || 'Unknown ship');
    appendSpan(ident, kill.victim_character_name || kill.victim_corporation_name || 'Unknown victim');
    appendSpan(ident, `${kill.solar_system_name || 'Unknown system'} · ${kill.region_name || 'New Eden'}`);
    const value = document.createElement('span'); value.className = 'map-live-value';
    const isk = document.createElement('strong'); isk.textContent = formatIsk(kill.total_value); value.append(isk);
    const time = document.createElement('span'); time.dataset.killTime = kill.killmail_time; time.textContent = age(kill.killmail_time); value.append(time);
    row.append(ident, value);
    const focus = () => {
      if(disposed)return;
      const host = surface.getBoundingClientRect(), rect = row.getBoundingClientRect();
      renderer.focusLiveSystem(kill.solar_system_id, { x: rect.right - host.left, y: rect.top + rect.height / 2 - host.top });
    };
    row.addEventListener('mouseenter', focus);
    row.addEventListener('focus', focus);
    row.addEventListener('mouseleave', () => renderer.focusLiveSystem(null));
    row.addEventListener('blur', () => renderer.focusLiveSystem(null));
    return row;
  };
  const render = () => {
    if (disposed) return;
    const kills = currentKills();
    if (kills.length) {
      const kept = new Set<number>();
      let next = list.firstChild;
      for (const kill of kills) {
        kept.add(kill.killmail_id);
        let row = rows.get(kill.killmail_id);
        if (!row) { row = rowFor(kill); rows.set(kill.killmail_id, row); }
        if (row !== next) placeElement(list,row,next);
        next = row.nextSibling;
      }
      for (const [id, row] of rows) {
        if (!kept.has(id)) { row.remove(); rows.delete(id); }
      }
      list.querySelector('p')?.remove();
    } else {
      rows.clear();
      const empty = document.createElement('p');
      if (mode === 'aiid' && !(scopeData.anchors || []).length) {
        const heading = document.createElement('strong'); heading.textContent = 'Choose your position';
        const help = document.createElement('span'); help.textContent = 'Add a mining, ratting, or travel system. AIID will stitch together every stargate route up to ten jumps away.';
        empty.append(heading, help);
      } else empty.textContent = mode === 'aiid' ? 'No kills found in the watch area during the last 24 hours.' : 'Waiting for killmails…';
      list.replaceChildren(empty);
    }
    count.textContent = mode === 'aiid' ? `${kills.length} kills in 24h` : `${kills.length} recent kills`;
    recent.textContent = mode === 'aiid' ? `${systems.size.toLocaleString('en-US')} systems` : `${kills.length} recent`;
  };
  const setConnected = (connected:boolean) => {
    panel.classList.toggle('connected', connected);
    status.classList.toggle('connected', connected);
    connection.textContent = connected ? mode === 'aiid' ? 'Armed' : 'Connected' : 'Connecting';
    indicator.textContent = connected ? mode === 'aiid' ? 'Danger watch armed' : 'Listening live' : 'Connecting to relay';
  };
  const acceptFresh = (kill:KilllistRow) => {
    const now = Date.now();
    for (const [id, until] of seenFresh) if (until <= now) seenFresh.delete(id);
    const occurred = Date.parse(kill.killmail_time);
    if (!Number.isFinite(occurred) || occurred > now + 60000 || now - occurred > 15 * 60000 || seenFresh.has(kill.killmail_id)) return false;
    seenFresh.set(kill.killmail_id, occurred + 16 * 60000);
    return true;
  };
  const relay = createRelayConnection({
    wsUrl:location.origin,path:'/ws/killlist',reconnect:true,visibilityPause:mode === 'live',
    shouldReconnect:()=>!disposed && (mode === 'aiid' || !document.hidden),onState:setConnected,
    onOpen:socket=>socket.send(JSON.stringify({action:'subscribe',topics:['all']})),
    onMessage:(channel,data)=>{
      if(disposed || channel !== 'killlist' || !isRecord(data))return;
      const kill = parseLiveKill(data.killmail);if(!kill)return;
      if (systems.has(kill.solar_system_id) && acceptFresh(kill)) {
        renderer.pulseKill(kill.solar_system_id, kill.total_value || 0);
        if (mode === 'aiid') {
          const distance = bySystem.get(kill.solar_system_id)?.distance;
          if (distance !== undefined && distance >= 0 && distance < 5) onAlarm?.('near');
          else if (distance !== undefined && distance >= 5 && distance <= 10) onAlarm?.('outer');
        }
      }
      streamed.unshift(kill);if(streamed.length > 200)streamed.length = 200;render();
    },
  });
  const seedPulses = () => {
    if (mode !== 'aiid') return;
    const now = Date.now(), pulsed = new Set<number>();
    for (const kill of bootstrap) {
      const ageMs = Math.max(0, now - Date.parse(kill.killmail_time));
      if (!Number.isFinite(ageMs) || ageMs > 15 * 60000 || !systems.has(kill.solar_system_id) || pulsed.has(kill.solar_system_id)) continue;
      pulsed.add(kill.solar_system_id);
      renderer.pulseKill(kill.solar_system_id, kill.total_value || 0, ageMs);
    }
  };
  const clock = setInterval(() => {
    for (const time of list.querySelectorAll<HTMLElement>('[data-kill-time]')) time.textContent = age(time.dataset.killTime ?? '');
  }, 15000);
  if (mode === 'live') {
    requestJSON<unknown>('/api/killlist?type=latest&limit=50',{signal:request.signal})
      .then(data=>{if(disposed)return;if(!isRecord(data))throw new Error('Invalid live map kill list');const kills = data.kills ?? [];if(!Array.isArray(kills))throw new Error('Invalid live map kill list');bootstrap = kills.map(parseLiveKill).filter((kill):kill is KilllistRow=>kill !== null);render();})
      .catch(error=>{if(!disposed && !isAbortError(error))console.warn('Live map bootstrap failed',error);});
  } else seedPulses();
  render();relay.connect();
  return {
    mode,
    update(nextRenderer:MapFeedRenderer, nextScope:MapFeedScope) {
      if(disposed)return;
      renderer.focusLiveSystem(null);
      renderer = nextRenderer;
      scopeData = nextScope;
      systems.clear(); bySystem.clear();
      for (const system of nextScope.systems || []) {
        systems.add(system.solar_system_id); bySystem.set(system.solar_system_id, system);
      }
      if (mode === 'aiid') bootstrap = nextScope.kills || [];
      seedPulses(); render();
    },
    destroy() {
      if(disposed)return;
      disposed = true;
      request.abort();
      clearInterval(clock);
      relay.dispose();
      for(const row of rows.values())row.remove();
      rows.clear();seenFresh.clear();
      renderer.focusLiveSystem(null);
      setConnected(false);
    },
  };
}
