import {escapeHTML} from './shared/html';
// Markup and labels ported from web/app/pages/graph.vue. Values are always
// escaped; links are restricted to the existing numeric entity routes.
import {formatCompact, formatIsk} from './formatters';
import {readGraphData,isGraphMode,type GraphEntity,type GraphMigrant,type GraphResult,type GraphResults,type GraphEntityType} from './shared/graph-model';
type Cell=readonly [string,string?];

export const escapeGraph = escapeHTML;
const e = escapeGraph;
const panel = 'rounded-lg border border-white/[0.08] bg-white/[0.02]';
const linkClass = 'text-blue-400 hover:text-blue-300';
const id = (value:unknown) => Number.isSafeInteger(Number(value)) && Number(value)>0 ? Number(value) : 0;
const link = (kind:string, entity:GraphEntity, classes='') => `<a href="/${kind}/${id(entity.id)}" class="${linkClass} ${classes}">${e(entity.name)}</a>`;
const compact = (value:unknown) => e(formatCompact(Number(value || 0)));
export const graphMessage = (text:unknown, empty=true) => `<div class="${panel} ${empty ? 'bg-[#0a0a0f] min-h-[200px]' : ''} p-8 flex items-center justify-center"><div class="text-gray-${empty?'600':'400'} text-sm text-center">${e(text)}</div></div>`;
const table = (heading:string, headers:Cell[], rows:string, color='white/[0.04]') => `<div class="${panel} overflow-hidden graph-table">${heading?`<div class="px-4 py-2 bg-${color} border-b border-white/[0.06] text-xs text-gray-500">${e(heading)}</div>`:''}<table class="w-full text-sm"><thead><tr class="text-left text-gray-500 border-b border-white/[0.06]">${headers.map(h=>`<th class="px-4 py-2 ${h[1]||''}">${e(h[0])}</th>`).join('')}</tr></thead><tbody>${rows}</tbody></table></div>`;
const row = (cells:Cell[]) => `<tr class="border-b border-white/[0.04] hover:bg-white/[0.04]">${cells.map(c=>`<td class="px-4 py-2 ${c[1]||''}">${c[0]}</td>`).join('')}</tr>`;
const rank = (i:number):Cell => [String(i+1),'text-gray-600 tabular-nums'];
const number = (value:unknown, color='white'):Cell => [e(value),'text-right tabular-nums text-'+color];
const arrow = '<svg class="icon" aria-hidden="true"><use href="/assets/icons.svg#arrow-right"></use></svg>';
const column = (title:string, subtitle:string, color:string, body:string, empty:string) => `<div class="${panel} overflow-hidden"><div class="px-4 py-2 bg-${color}-500/10 border-b border-white/[0.06]"><span class="text-sm font-medium text-${color}-400">${e(title)}</span><span class="text-xs text-gray-500 ml-2">${e(subtitle)}</span></div>${body?`<div class="divide-y divide-white/[0.04]">${body}</div>`:`<div class="p-4 text-sm text-gray-600 text-center">${e(empty)}</div>`}</div>`;

function path(data:GraphResults['path_finder']) {
  if(!data.path) return graphMessage('No connection found within 6 hops',false);
  const nodes=data.path.nodes||[], edges=data.path.edges||[];
  return `<div class="${panel} p-4"><div class="flex items-center gap-2 mb-4"><span class="text-sm font-medium text-green-400">Connected</span><span class="text-xs text-gray-500">${e(data.path.hops)} ${data.path.hops===1?'hop':'hops'}</span></div><div class="flex flex-wrap items-center gap-1">${nodes.map((n,i)=>`<div class="graph-path-node flex flex-col items-center px-3 py-2 rounded-lg bg-white/[0.04] border border-white/[0.06] min-w-[120px]">${link('character',n,'text-sm font-medium text-center')}${n.corporation?`<a href="/corporation/${id(n.corporation.id)}" class="text-xs text-gray-400 hover:text-gray-300 mt-0.5">${e(n.corporation.name)}</a>`:''}${n.alliance?`<a href="/alliance/${id(n.alliance.id)}" class="text-xs text-gray-600 hover:text-gray-500">[${e(n.alliance.name)}]</a>`:''}</div>${i<edges.length?`<div class="flex flex-col items-center px-2 text-gray-600 shrink-0"><div class="text-xs tabular-nums text-gray-500">×${compact(edges[i].weight)}</div>${arrow}</div>`:''}`).join('')}</div></div>`;
}
function systems(mode:'hunting_grounds'|'hot_zones',data:GraphResults['hunting_grounds']|GraphResults['hot_zones']) {
  if(!data.systems?.length) return graphMessage(mode==='hot_zones'?'No hot zones detected in the last 24 hours':'No recent activity found');
  const hot=mode==='hot_zones';
  return table(hot?'Systems with cross-alliance conflict in the last 24 hours':'Activity in the last 24 hours', [['#','w-10'],['System'],...(hot?[['Alliances','text-right'],['Characters','text-right']] as const:[['Active Characters','text-right']] as const)],data.systems.map((s,i)=>row([rank(i),[link('system',s)],...(hot?[number(s.alliances,'yellow-400'),number(s.characters)]:[number(s.active_characters)])])).join(''),hot?'red-500/10':'white/[0.04]');
}
function census(data:GraphResults['census']) {
  if(!data.corps?.length)return graphMessage('No military composition data found');
  const totals=data.totals||{};
  const cards=[['characters','Characters','white',true],['fcs','FCs','blue',false],['logis','Logi Pilots','green',true],['caps','Cap Pilots','orange',true]] as const;
  return `<div class="graph-census-totals grid grid-cols-4 gap-3 mb-4">${cards.map(([key,label,color,formatted])=>`<div class="${color==='white'?'glass-panel':`rounded-lg bg-${color}-500/10 border border-${color}-500/20`} p-3 text-center"><div class="text-lg font-bold text-${color}${color==='white'?'':'-400'} tabular-nums">${formatted?compact(totals[key]):e(totals[key]??0)}</div><div class="text-xs text-gray-500">${label}</div></div>`).join('')}</div>`+table('',[['Corporation'],['Chars','text-right'],['FCs','text-right'],['Logi','text-right'],['Caps','text-right']],data.corps.map(c=>row([[link('corporation',c,'text-sm')],number(c.total),number(c.fcs||'-','blue-400'),number(c.logis||'-','green-400'),number(c.caps||'-','orange-400')]).replaceAll('py-2','py-1.5')).join(''));
}
function spy(data:GraphResults['spy_check']) {
  if(!data.suspects?.length)return graphMessage('No suspicious cross-alliance activity detected');
  return `<div class="${panel} overflow-hidden"><div class="px-4 py-2 bg-yellow-500/10 border-b border-white/[0.06] text-xs text-gray-500">Characters with significant FLEW_WITH connections to outside groups</div><div class="divide-y divide-white/[0.04]">${data.suspects.map(s=>`<div class="px-4 py-2 hover:bg-white/[0.04]"><div class="graph-suspect-heading flex items-center gap-3">${link('character',s,'text-sm font-medium')}${s.corp_name?`<span class="text-xs text-gray-500">${e(s.corp_name)}</span>`:''}<span class="ml-auto text-xs text-yellow-400 tabular-nums">${compact(s.total_flights)} flights</span></div><div class="flex gap-2 mt-1 flex-wrap">${(s.connections||[]).map(c=>`<span class="text-xs px-2 py-0.5 rounded bg-white/[0.06] text-gray-400">${e(c.name)} <span class="text-gray-600">×${compact(c.flights)}</span></span>`).join('')}</div></div>`).join('')}</div></div>`;
}
function migration(data:GraphResults['migration']) {
  const chars=(entries:GraphMigrant[],field:'current_corp'|'previous_corp',prefix:string)=>(entries||[]).map(c=>`<div class="flex items-center gap-3 px-4 py-1.5 hover:bg-white/[0.04]">${link('character',c,'text-sm flex-1 truncate')}${c[field]?.name?`<span class="text-xs text-gray-500 truncate max-w-[150px]">${prefix} ${e(c[field].name)}</span>`:''}</div>`).join('');
  return `<div class="grid grid-cols-1 md:grid-cols-2 gap-4">${column('Departed','formerly in '+(data.corp_name||'this corp'),'red',chars(data.departed,'current_corp','now in'),'No departures detected')}${column('Joined','currently active, came from elsewhere','green',chars(data.joined,'previous_corp','from'),'No recent joins from other corps')}</div>`;
}
function intel(data:GraphResults['entity_intel'],type:GraphEntityType) {
  const ally=(data.allies||[]).map((a,i)=>`<div class="flex items-center gap-3 px-4 py-1.5 hover:bg-white/[0.04]"><span class="text-xs text-gray-600 tabular-nums w-5">${i+1}</span>${link(type,a,'text-sm flex-1 truncate')}<span class="text-xs text-gray-500 tabular-nums shrink-0" title="Shared enemy kills: ${e(a.shared_enemy_kills)}, Mutual kills: ${e(a.mutual_kills)}">${compact(a.shared_enemy_kills)} shared</span></div>`).join('');
  const enemy=(data.enemies||[]).map((a,i)=>`<div class="flex items-center gap-3 px-4 py-1.5 hover:bg-white/[0.04]"><span class="text-xs text-gray-600 tabular-nums w-5">${i+1}</span>${link(type,a,'text-sm flex-1 truncate')}<div class="flex gap-3 shrink-0"><span class="text-xs text-green-400/70 tabular-nums" title="Kills given">${compact(a.kills_given)}</span><span class="text-xs text-red-400/70 tabular-nums" title="Kills taken">${compact(a.kills_taken)}</span></div></div>`).join('');
  return `<div class="grid grid-cols-1 md:grid-cols-2 gap-4">${column('Allies','by shared enemy activity','green',ally,'No allies detected')}${column('Enemies','by mutual kills','red',enemy,'No enemies detected')}</div>`;
}
function coalitions(data:GraphResults['coalitions']) {
  const items=data.coalitions||[];
  if(!items.length)return graphMessage('No coalitions detected');
  return `<div class="space-y-3"><div class="text-xs text-gray-500 mb-2">Detected ${items.length} coalition${items.length>1?'s':''} based on shared enemy analysis</div>${items.map((c,i)=>`<div class="${panel} overflow-hidden"><button type="button" data-graph-coalition="${i}" aria-expanded="false" class="graph-coalition-heading w-full px-4 py-3 bg-white/[0.04] flex items-center gap-3 hover:bg-white/[0.06] transition-colors text-left"><svg class="icon text-gray-500 shrink-0" aria-hidden="true"><use href="/assets/icons.svg#chevron-right"></use></svg><span class="text-sm font-medium text-white">Coalition ${e(c.id)}</span><span class="text-xs text-gray-500">${c.alliances.length} alliances</span><span class="flex-1 text-xs text-gray-600 truncate ml-2">${e(c.alliances.slice(0,6).map(a=>a.name).join(', '))}${c.alliances.length>6?', ...':''}</span></button><div hidden data-graph-coalition-members class="divide-y divide-white/[0.04]">${c.alliances.map(a=>`<div class="flex items-center gap-3 px-4 py-1.5 hover:bg-white/[0.04] transition-colors">${link('alliance',a,'text-sm flex-1 truncate')}<span class="text-xs text-gray-500 tabular-nums shrink-0">${compact(a.cooperation)}</span></div>`).join('')}</div></div>`).join('')}</div>`;
}
function rivalries(data:GraphResults['rivalries'],type:GraphEntityType) {
  if(!data.items?.length)return graphMessage('No rivalry data available');
  const side=(entity:GraphEntity)=>link(type,entity)+(entity.corp_name?`<div class="text-xs text-gray-500">${e(entity.corp_name)}</div>`:'')+(entity.alliance_name?`<div class="text-xs text-gray-600">[${e(entity.alliance_name)}]</div>`:'');
  return table('',[['#','w-10'],['Side A'],['','text-center w-8'],['Side B'],['Mutual Kills','text-right'],['ISK Destroyed','text-right']],data.items.map((v,i)=>row([rank(i),[side(v.entity_a)],['vs','text-center text-red-500 text-xs font-bold'],[side(v.entity_b)],number(compact(v.mutual_kills)),number(formatIsk(Number(v.total_isk||0)),'gray-400')])).join(''));
}
export function renderGraphData(data:GraphResult,type:GraphEntityType='alliance') {
  switch(data.mode) {
    case 'path_finder':return path(data);
    case 'entity_intel':return intel(data,type);
    case 'hunting_grounds':case 'hot_zones':return systems(data.mode,data);
    case 'census':return census(data);
    case 'spy_check':return spy(data);
    case 'migration':return migration(data);
    case 'coalitions':return coalitions(data);
    case 'rivalries':return rivalries(data,type);
    default:return graphMessage('Unknown graph tool');
  }
}

export function renderGraph(mode:string,raw:unknown,type:GraphEntityType='alliance'):string {return isGraphMode(mode)?renderGraphData(readGraphData(mode,raw),type):graphMessage('Unknown graph tool')}
