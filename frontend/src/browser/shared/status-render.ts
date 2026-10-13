import {escapeHTML} from './html'
import {statusNumber,type StatusSnapshot,type StatusHistory} from './status-model'

// Keep the existing Go dashboard markup and formatting; only its inputs are typed.
export function renderStatusPanels(data:StatusSnapshot):string {
  const number = (value:unknown) => new Intl.NumberFormat('en-US').format(n(value));
  const compact = (value:unknown) => new Intl.NumberFormat('en-US', {notation: 'compact', maximumFractionDigits: 1}).format(n(value));
  const h = escapeHTML;
  const n = statusNumber;
  const fixed = (value:unknown, digits = 1) => n(value).toFixed(digits);
  const pct = (a:number, b:number) => b > 0 ? `${fixed(a / b * 100)}%` : '—';
  const title = (value:unknown) => `<div class="status-panel-title">${h(value)}</div>`;
  const tile = (label:unknown, value:string, detail = '', tone = '') => `<div class="status-tile"><div class="status-tile-label">${h(label)}</div><div class="status-tile-value ${tone}">${value}</div>${detail ? `<div class="status-tile-detail">${detail}</div>` : ''}</div>`;
  const panel = (content:string, extra = '') => `<section class="glass-panel status-panel ${extra}">${content}</section>`;
  const row = (label:unknown, value:string) => `<div class="status-row"><span>${h(label)}</span><span>${value}</span></div>`;
  const duration = (seconds:number|undefined, includeSeconds = false) => {
    seconds = Math.max(0, n(seconds));
    const parts = [];
    const days = Math.floor(seconds / 86400), hours = Math.floor(seconds % 86400 / 3600), minutes = Math.floor(seconds % 3600 / 60);
    if (days) parts.push(`${days}d`);
    if (hours) parts.push(`${hours}h`);
    if (minutes || !parts.length) parts.push(`${minutes}m`);
    if (includeSeconds) parts.push(`${Math.floor(seconds % 60)}s`);
    return parts.join(' ');
  };
  const rate = (value:unknown) => n(value) >= 1000 ? `${fixed(n(value)/1000)}k/s` : n(value) >= 10 ? `${n(value).toFixed(0)}/s` : `${fixed(value)}/s`;
  const bytes = (value:unknown) => n(value) >= 1073741824 ? `${fixed(n(value)/1073741824)} GB/s` : n(value) >= 1048576 ? `${fixed(n(value)/1048576)} MB/s` : n(value) >= 1024 ? `${fixed(n(value)/1024)} KB/s` : `${n(value).toFixed(0)} B/s`;
  const wallet = (value:string|number|undefined) => {
    const v = Number(value);
    if (!Number.isFinite(v)) return '— ISK';
    for (const [unit, suffix] of [[1e12,'t'],[1e9,'b'],[1e6,'m'],[1e3,'k']] as const) if (Math.abs(v) >= unit) return `${Math.round(v/unit)}${suffix} ISK`;
    return `${number(v)} ISK`;
  };
  const summary = (data:StatusSnapshot) => {
    const queues = Object.values(data.queues || {});
    const queued = queues.reduce((sum, q) => sum + n(q.waiting), 0);
    const active = queues.reduce((sum, q) => sum + n(q.active), 0);
    const redis = data.redis || {};
    const hitDenominator = n(redis.keyspace_hits) + n(redis.keyspace_misses);
    const hit = hitDenominator ? `${Math.round(n(redis.keyspace_hits) / hitDenominator * 100)}% hit · ` : '';
    const tq = String(data.esi?.tq_status || 'unknown').toUpperCase();
    return `<div class="status-summary">
      ${panel(`<div class="status-summary-label">Tranquility</div><strong class="${tq === 'ONLINE' ? 'status-green' : tq === 'VIP' ? 'status-amber' : 'status-red'}">${h(tq)}</strong><small>${number(data.esi?.tq_players)} pilots</small>`)}
      ${panel(`<div class="status-summary-label">EK Wallet</div><strong class="status-amber" title="${h(data.wallet?.total_balance)} ISK">${h(wallet(data.wallet?.total_balance))}</strong><small>corp balance</small>`)}
      ${panel(`<div class="status-summary-label">Queued</div><strong class="status-amber">${number(queued)}</strong><small>${number(active)} active</small>`)}
      ${panel(`<div class="status-summary-label">ESI Tokens</div><strong class="status-purple">${number(data.esi_tokens?.tokens?.active)} <em>/ ${number(data.esi_tokens?.tokens?.total)}</em></strong><small>active keys</small>`)}
      ${panel(`<div class="status-summary-label">Database</div><strong class="status-blue">${h(data.database?.size || '?')}</strong><small>${number(Object.keys(data.database?.tables || {}).length)} tables</small>`)}
      ${panel(`<div class="status-summary-label">Cache</div><strong class="status-cyan">${h(redis.used_memory || '—')}</strong><small>${hit}${number(redis.connected_clients)} clients</small>`)}</div>`;
  };
  const system = (data:StatusSnapshot) => panel(`${title('System')}<div class="status-tile-grid status-system-grid">
    ${tile('Platform', `${h(data.system?.platform)} (${h(data.system?.arch)})`)}
    ${tile('CPUs', number(data.system?.cpus))}
    ${tile('Memory', h(data.system?.memory_total))}
    ${tile('Free Memory', h(data.system?.memory_free))}
    ${tile('Load Average', (data.system?.load_average || []).map(x => fixed(x, 2)).join(' / '))}</div>`);
  const cloudflare = (data:StatusSnapshot['cloudflare']) => {
    if (!data) return '';
    const day = data.last_24h || {}, hour = data.last_1h || {};
    const codes = day.status_codes || {};
    const updated = data.last_updated ? new Date(data.last_updated).toLocaleTimeString() : '—';
    return panel(`<div class="status-panel-heading">${title('Cloudflare')}<small>Updated ${h(updated)}</small></div><div class="status-tile-grid status-four">
      ${tile('Requests (24h)', number(day.requests), `${number(hour.requests)} last hour`)}
      ${tile('Bandwidth (24h)', `${h(day.bandwidth_gb)} GB`, `${h(day.bandwidth_cached_gb)} GB cached`)}
      ${tile('Cache Hit Rate', `${h(day.cache_hit_rate)}%`, `${h(hour.cache_hit_rate)}% last hour`, n(day.cache_hit_rate)>50?'status-green':'status-amber')}
      ${tile('Threats Blocked (24h)', number(day.threats_blocked), `${number(hour.threats_blocked)} last hour`, n(day.threats_blocked)>0?'status-red':'status-green')}
      ${tile('Page Views (24h)', number(day.page_views))}${tile('Unique Visitors (24h)', number(day.unique_visitors))}
      ${tile('API Requests (24h)', number(n(codes[200])+n(codes[304])), '2xx + 304')}
      ${tile('Errors (24h)', number([500,502,503,504].reduce((sum,code)=>sum+n(codes[code]),0)), '5xx errors', n(codes[500])>0?'status-red':'status-green')}</div>`);
  };
  const ingest = (data:StatusSnapshot['zkb_ingest']) => {
    if (!data) return '';
    const fresh = data.new_hourly || [], reposts = data.repost_hourly || [];
    const total = (values:number[]) => values.reduce((sum, value) => sum + n(value), 0);
    const freshTotal = total(fresh), repostTotal = total(reposts);
    const max = Math.max(1, ...fresh.map((value, i) => n(value)+n(reposts[i])));
    const bars = fresh.map((value, i) => {
      const recent = n(value), repost = n(reposts[i]);
      const height = recent+repost ? Math.max(1, Math.round((recent+repost)/max*100)) : 0;
      return `<div class="status-ingest-hour" data-status-hour="${i}" tabindex="0" role="img" aria-label="Hour ${i+1}: ${recent} new kills and ${repost} reposts" data-new="${recent}" data-repost="${repost}"><div class="status-ingest-stack" style="height:${height}%"><div class="status-ingest-repost" style="flex-grow:${repost}"></div><div class="status-ingest-new" style="flex-grow:${recent}"></div></div></div>`;
    }).join('');
    return panel(`<div class="status-panel-heading">${title('zKB Ingest')}<small>Last ${number(data.window_hours)}h · hourly</small></div>
      <div class="status-tile-grid status-four">${tile('New kills',number(freshTotal),`${number(fresh.at(-1))} last hour`)}
      ${tile('Reposts skipped',number(repostTotal),`${number(reposts.at(-1))} last hour`)}
      ${tile('Repost ratio',`${freshTotal+repostTotal ? Math.round(repostTotal/(freshTotal+repostTotal)*100) : 0}%`,'of all feed events', repostTotal>freshTotal?'status-amber':'status-green')}
      ${tile('New / hour avg',number(n(data.window_hours) ? Math.round(freshTotal/n(data.window_hours)) : 0),`over ${number(data.window_hours)}h`)}</div>
      <div class="status-ingest-chart" data-window-hours="${n(data.window_hours)}"><div class="status-ingest-lines"></div><div class="status-ingest-bars">${bars}</div><div class="status-ingest-tooltip" hidden></div></div>
      <div class="status-ingest-legend"><span>-${number(data.window_hours)}h</span><span><i class="status-swatch-new"></i>new&nbsp;&nbsp; <i class="status-swatch-repost"></i>repost</span><span>now</span></div>`,'status-ingest-panel');
  };
  const coverage = (data:StatusSnapshot) => {
    const cov = data.coverage, tokens = data.esi_tokens;
    if (!cov && !tokens) return '';
    const coverageRows = cov ? [
      ['Characters', cov.characters?.covered, cov.characters?.active, 'via own or corp key'],
      ['Corporations', cov.corporations?.covered, cov.corporations?.active, 'with corp key'],
      ['Alliances', cov.alliances?.corps_covered, cov.alliances?.corps_active, `member corps · ${number(cov.alliances?.active)} alliances`],
    ] : [];
    const activity = tokens ? [
      ['Character keys', tokens.tokens?.can_fetch_character], ['Corporation keys', tokens.tokens?.can_fetch_corporation],
      ['Fetches (24h)', tokens.fetches?.last_24h?.total], ['Failed (24h)', tokens.fetches?.last_24h?.failed],
      ['Keyed KMs (24h)', tokens.fetches?.last_24h?.killmails_found], ['New KMs (24h)', tokens.fetches?.last_24h?.new_killmails],
    ] : [];
    return panel(`<div class="status-panel-heading">${title('ESI Coverage & Tokens')}<small><b class="status-green">${number(tokens?.tokens?.active)} active</b> &nbsp;/&nbsp; ${number(tokens?.tokens?.total)} keys ${n(tokens?.tokens?.revoked) ? `<span class="status-red">&nbsp; ${number(tokens?.tokens?.revoked)} revoked</span>` : ''}</small></div>
      <div class="status-coverage-grid"><section><div class="status-subheading">Coverage <small>entities active last ${number(cov?.window_days)}d</small></div>
      ${coverageRows.length ? coverageRows.map(([label, covered, active, detail])=>`<div class="status-coverage-row"><div>${h(label)}<b>${pct(n(covered),n(active))}</b></div><div class="status-progress"><span style="width:${active ? Math.min(100,n(covered)/n(active)*100) : 0}%"></span></div><small>${number(covered)} / ${number(active)} ${h(detail)}</small></div>`).join('') : '<div class="status-muted">Coverage data unavailable</div>'}</section>
      <section><div class="status-subheading">Token activity</div><div class="status-activity-grid">${activity.length ? activity.map(([label,value])=>`<div class="status-activity-tile"><span>${h(label)}</span><strong class="${label==='New KMs (24h)'?'status-green':label==='Failed (24h)' && n(value)?'status-red':''}">${number(value)}</strong></div>`).join('') : '<div class="status-muted">Token data unavailable</div>'}</div>
      ${tokens ? `<div class="status-all-time"><span>All time</span><span>${number(tokens.fetches?.all_time?.total)} <em>fetches</em>&nbsp;&nbsp; <b class="status-green">${number(tokens.fetches?.all_time?.new_killmails)}</b> <em>new KMs</em></span></div>` : ''}</section></div>`,'status-coverage-panel');
  };
  const queues = (data:StatusSnapshot) => panel(`<div class="status-panel-heading">${title('Queues')}<small>ESI Budget&nbsp; <b class="${n(data.esi?.error_budget)>50?'status-green':'status-red'}">${number(data.esi?.error_budget)}%</b>${data.esi?.paused?' <b class="status-red">PAUSED</b>':''}</small></div>
    <div class="status-rows">${Object.entries(data.queues||{}).map(([name,q])=>row(name,`<span class="${n(q.waiting)?'status-amber':'status-muted'}">${number(q.waiting)} <small>wait</small></span><span class="${n(q.active)?'status-green':'status-muted'}">${number(q.active)} <small>active</small></span>${n(q.failed)?`<span class="status-red">${number(q.failed)} <small>fail</small></span>`:''}`)).join('')}</div>`,'status-queue-panel');
  const linePoints = (history:StatusHistory[], key:keyof StatusHistory, keys:Array<keyof StatusHistory>) => {
    if (history.length < 2) return '';
    const max = Math.max(1,...history.flatMap(point=>keys.map(name=>n(point[name]))));
    return history.map((point,i)=>`${(i*100/(history.length-1)).toFixed(2)},${(38-n(point[key])/max*36-1).toFixed(2)}`).join(' ');
  };
  const graph = (history:StatusHistory[], keys:Array<keyof StatusHistory>, colors:string[], legend:string[], heading:string) => `<div class="status-db-graph"><div class="status-db-graph-header"><span>${h(heading)}</span><span>${legend.map((name,i)=>`<em style="color:${colors[i]}">— ${h(name)}</em>`).join(' · ')}</span></div><svg viewBox="0 0 100 40" preserveAspectRatio="none" aria-hidden="true"><line x1="0" y1="39" x2="100" y2="39" stroke="rgba(255,255,255,.06)" stroke-width=".5"/>${keys.map((key,i)=>`<polyline points="${linePoints(history,key,keys)}" fill="none" stroke="${colors[i]}" stroke-width="1.5" vector-effect="non-scaling-stroke"/>`).join('')}</svg></div>`;
  const postgres = (data:StatusSnapshot) => {
    const db:NonNullable<StatusSnapshot['database']> = data.database || {history:[],tables:{}}, con = db.connections || {}, work = db.workload, stats = db.statements, maintenance = db.maintenance, cluster = db.cluster || {};
    const max = n(con.max)||1;
    const bar = (value:unknown, className:string) => `<span class="${className}" style="width:${Math.min(100,n(value)/max*100)}%"></span>`;
    const lag = n(cluster.max_lag_seconds)>=1 ? `${fixed(cluster.max_lag_seconds)}s` : n(cluster.max_lag_seconds)>0 ? `${Math.round(n(cluster.max_lag_seconds)*1000)}ms` : n(cluster.max_lag_bytes)>=1048576 ? `${fixed(n(cluster.max_lag_bytes)/1048576)} MB` : n(cluster.max_lag_bytes)>=1024 ? `${fixed(n(cluster.max_lag_bytes)/1024)} KB` : n(cluster.max_lag_bytes) ? `${number(cluster.max_lag_bytes)} B` : '0';
    const history = db.history || [];
    return panel(`<div class="status-panel-heading">${title('PostgreSQL')}<small class="${db.role==='primary'?'status-green':'status-amber'} status-uppercase">${h(db.role||'unknown')}</small></div>
      <div class="status-tile-grid status-three">${tile('Role',h(db.role||'unknown'),'','status-green')}${tile('Replicas',n(cluster.replicas)?`${number(cluster.streaming)} / ${number(cluster.replicas)}`:'Standalone',n(cluster.replicas)?'streaming':'')}${tile('Max Lag',h(lag),'',n(cluster.max_lag_seconds)>5?'status-red':'status-green')}</div>
      <div class="status-rows status-db-rows">
        ${row('Connections',`${number(con.total)} / ${number(con.max)} <small>· ${number(con.active)} active · ${number(con.waiting)} waiting</small>`)}
        <div class="status-connection-bar">${bar(con.active,'status-bar-green')}${bar(con.idle_in_transaction,'status-bar-amber')}${bar(con.waiting,'status-bar-red')}</div>
        ${row('Transactions',work?`${rate(work.transactions_per_second)} <small>· ${fixed(work.rollback_percent,2)}% rollback</small>`:'<small>collecting…</small>')}
        ${row('Queries',stats?`${rate(stats.queries_per_second)} <small>· ${fixed(stats.average_latency_ms,2)} ms avg</small>`:'<small>statistics unavailable</small>')}
        ${row('I/O Throughput',work?`<b class="status-blue">↓ ${bytes(work.read_bytes_per_second)}</b> · <b class="status-amber">↑ ${bytes(work.write_bytes_per_second)}</b>`:'<small>collecting…</small>')}
        ${row('WAL',work?bytes(work.wal_bytes_per_second):'<small>collecting…</small>')}
        ${row('I/O Latency',work?`${fixed(work.read_latency_ms,2)} ms read <small>· ${fixed(work.write_latency_ms,2)} ms write</small>`:'<small>collecting…</small>')}
        ${row('Row Churn',work?rate(work.rows_changed_per_second):'<small>collecting…</small>')}
        ${row('Buffer Cache',`<b class="${n(db.cache_hit_ratio)>=95?'status-green':'status-amber'}">${fixed(db.cache_hit_ratio,2)}%</b>`)}
        ${row('Contention',`<b class="${n(db.waiting_locks)?'status-red':''}">${number(db.waiting_locks)} locks</b> <small>· ${number(work?.deadlocks)} deadlocks</small>`)}</div>
      ${maintenance ? `<div class="status-tile-grid status-four status-maintenance">${tile('Checkpoints',number(maintenance.checkpoints),`${number(maintenance.requested_checkpoints)} requested · ${n(maintenance.average_checkpoint_time_ms).toFixed(0)} ms avg`)}${tile('Temp Spill',bytes(work?.temp_bytes_per_second),'sorts and hashes')}${tile('Dead Rows',compact(maintenance.dead_rows),'planner estimate')}${tile('Autovacuum',`${number(maintenance.active_autovacuums)} active`,'workers now')}</div>` : ''}
      ${graph(history,['queries_per_second','transactions_per_second'],['#34d399','#60a5fa'],['queries','transactions'],'Workload · last 5 minutes')}
      ${graph(history,['read_bytes_per_second','write_bytes_per_second','wal_bytes_per_second'],['#60a5fa','#fbbf24','#c084fc'],['read','write','WAL'],'Storage throughput · last 5 minutes')}
      <div class="status-db-footer"><span>v${h(db.version||'—')}</span><span>${duration(db.uptime_seconds)} uptime</span><span>${h(db.size||'—')}</span></div>`,'status-db-panel');
  };
  const imageServer = (data:StatusSnapshot['image_server']) => {
    if (!data) return '';
    const folders = Object.entries(data.folderStats||{}).filter(([name])=>name.toLowerCase()!=='types').map(([name,stats])=>({name,...stats})).sort((a,b)=>n(b.fileCount)-n(a.fileCount));
    const size = (kb:unknown) => n(kb)>=1048576?`${fixed(n(kb)/1048576)} GB`:n(kb)>=1024?`${fixed(n(kb)/1024)} MB`:`${number(kb)} KB`;
    return panel(`<div class="status-panel-heading">${title('Image Server')}<small>${number(folders.reduce((sum,folder)=>sum+n(folder.fileCount),0))} files · ${size(folders.reduce((sum,folder)=>sum+n(folder.sizeKB),0))}</small></div><div class="status-tile-grid status-system-grid">${folders.map(folder=>tile(folder.name,number(folder.fileCount),`${size(folder.sizeKB)}${n(folder.metadataFileCount)?` · ${number(folder.metadataFileCount)} metadata`:''}`)).join('')}</div>${data.cacheValidation?`<div class="status-tile-grid status-four status-maintenance">${tile('Validation Runs',number(data.cacheValidation.totalValidationRuns))}${tile('Images Validated',number(data.cacheValidation.totalImagesValidated))}${tile('Images Removed',number(data.cacheValidation.totalImagesRemoved))}${tile('Validation Errors',number(data.cacheValidation.validationErrors))}</div>`:''}`);
  };
  const websocketServer = (data:StatusSnapshot['websocket']) => {
    if (!data) return '';
    const endpoints = Object.entries(data.endpoints||{}).map(([path,ep])=>{
      const groups=new Map<string,number>(); for(const [sub,count] of Object.entries(ep.subscriptions||{})){const key=sub.includes('.')?sub.split('.')[0]:sub;groups.set(key,(groups.get(key)??0)+n(count))}
      return {path, connections:n(ep.connections), groups:[...groups].sort((a,b)=>b[1]-a[1])};
    }).sort((a,b)=>b.connections-a.connections);
    return panel(`<div class="status-panel-heading">${title('WebSocket Server')}<small>Redis ${data.redis?'connected':'disconnected'} · ${duration(data.uptime,true)} uptime</small></div><div class="status-tile-grid status-four">${tile('Connections',number(data.connections))}${tile('Channels',number(data.channels?.length))}${tile('Endpoints',number(endpoints.length))}${tile('Status',h(String(data.status||'').toUpperCase()),'',data.status==='ok'?'status-green':'status-red')}</div><div class="status-rows">${endpoints.map(ep=>row(ep.path,`<span class="${ep.connections?'status-green':'status-muted'}">${number(ep.connections)} <small>conn</small></span>${ep.groups.map(([name,count])=>`<span class="status-blue">${number(count)} <small>${h(name)}</small></span>`).join('')}`)).join('')}</div>`);
  };
  const tables = (data:StatusSnapshot) => {
    const sorted = Object.entries(data.database?.tables||{}).map(([name,table])=>({name,...table})).sort((a,b)=>n(b.rows)-n(a.rows)).slice(0,15);
    const table = (name:string,rows:Array<{name:string;rows?:number;data_size?:string;total_size?:string}>) => panel(`${title(name)}<div class="status-table-scroll" tabindex="0" role="region" aria-label="${h(name)}"><table class="status-table"><thead><tr><th>Table</th><th>Rows</th><th>Data</th><th>Total</th></tr></thead><tbody>${rows.map(t=>`<tr><td>${h(t.name)}</td><td>${number(t.rows)}</td><td>${h(t.data_size)}</td><td>${h(t.total_size)}</td></tr>`).join('')}</tbody></table></div>`);
    return `<div class="status-two">${table('Database Tables (1-8)',sorted.slice(0,8))}${table('Database Tables (9-15)',sorted.slice(8,15))}</div>`;
  };
  return [summary(data),system(data),cloudflare(data.cloudflare),ingest(data.zkb_ingest),coverage(data),`<div class="status-two">${queues(data)}${postgres(data)}</div>`,imageServer(data.image_server),websocketServer(data.websocket),tables(data)].join('');
}
