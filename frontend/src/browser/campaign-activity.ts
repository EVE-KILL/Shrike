import {escapeHTML} from './shared/html';
import { formatIsk as compactISK } from './campaign-data.js';
import {Controller} from '@hotwired/stimulus';
import {registerController} from './controllers/application';
import {eventElement,requiredElement} from './shared/dom';
import {isRecord} from './shared/json';
const svgEscape = escapeHTML;
type ActivityRow={period:string;side_index:number;kills:number;losses:number;isk_destroyed:number;isk_lost:number};
type ActivitySource={rows:ActivityRow[];sides:Array<{index:number;name:string;color:string}>;area:boolean;granularity:string};
type Bar={period:string;x:number;width:number;totalShips:number;totalIsk:number;segments:Array<{side:number;ships:number;isk:number;value:number;y:number;height:number;topmost:boolean}>};
const number=(value:unknown)=>Number.isFinite(Number(value))?Number(value):0;
export function readCampaignActivity(raw:unknown):ActivitySource {
  const source=isRecord(raw)?raw:{};
  return {area:source.area===true,granularity:typeof source.granularity==='string'?source.granularity:'day',rows:Array.isArray(source.rows)?source.rows.flatMap(row=>isRecord(row)&&typeof row.period==='string'&&/^\d{4}-\d{2}-\d{2}$/.test(row.period)?[{period:row.period,side_index:number(row.side_index),kills:number(row.kills),losses:number(row.losses),isk_destroyed:number(row.isk_destroyed),isk_lost:number(row.isk_lost)}]:[]):[],sides:Array.isArray(source.sides)?source.sides.flatMap(side=>isRecord(side)&&typeof side.name==='string'?[{index:number(side.index),name:side.name,color:typeof side.color==='string'&&/^#[0-9a-f]{6}$/i.test(side.color)?side.color:'#6b7280'}]:[]):[]};
}
const niceCeil = (value:number) => {
  if (value <= 4) return 4;
  const power = 10 ** Math.floor(Math.log10(value));
  return ([1, 1.2, 1.5, 2, 2.5, 3, 4, 5, 6, 8, 10].find(step => step >= value / power) || 10) * power;
};
const roundedTop = (x:number,y:number,width:number,height:number) => {
  const radius = Math.min(3, width / 2, height);
  return `M${x},${y + height} L${x},${y + radius} Q${x},${y} ${x + radius},${y} L${x + width - radius},${y} Q${x + width},${y} ${x + width},${y + radius} L${x + width},${y + height} Z`;
};

function createActivity(section:HTMLElement) {
  const q=<T extends Element=HTMLElement>(selector:string)=>requiredElement<T>(section,selector);
  let source:ActivitySource;
  try { source = readCampaignActivity(JSON.parse(q('script[type="application/json"]').textContent??"null")); }
  catch { section.hidden = true; return; }
  const rows = Array.isArray(source.rows) ? source.rows : [];
  const sides = Array.isArray(source.sides) ? source.sides : [];
  const oneSided = !source.area && sides.length === 1;
  const svg = q<SVGSVGElement>('[data-campaign-chart]');
  const tooltip = q('[data-campaign-chart-tooltip]');
  const legend = q('[data-campaign-chart-legend]');
  const title = q('.campaign-detail-activity-heading h2');
  const colors = new Map(sides.map(side => [Number(side.index), /^#[0-9a-f]{6}$/i.test(side.color) ? side.color : '#6b7280']));
  const colorOf = (side:number) => side === -1 ? '#3b82f6' : side === 100 ? '#059669' : side === 101 ? '#ef4444' : colors.get(side) || '#6b7280';
  const nameOf = (side:number) => side === -1 ? 'All activity' : side === 100 ? 'Kills' : side === 101 ? 'Losses' : sides.find(value => Number(value.index) === side)?.name || `Side ${side + 1}`;
  const periods = new Map<string,Map<number,{ships:number;isk:number}>>();
  for (const row of rows) {
    const key = String(row.period || '');
    if (!/^\d{4}-\d{2}-\d{2}$/.test(key)) continue;
    const bySide = periods.get(key) || new Map<number,{ships:number;isk:number}>();
    const add = (side:number,ships:number,isk:number) => {
      const entry = bySide.get(side) || { ships: 0, isk: 0 };
      entry.ships += number(ships); entry.isk += number(isk); bySide.set(side, entry);
    };
    if (oneSided) {
      add(100, row.kills, row.isk_destroyed);
      add(101, row.losses, row.isk_lost);
    } else {
      add(Number(row.side_index), source.area ? row.kills : row.losses, source.area ? row.isk_destroyed : row.isk_lost);
    }
    periods.set(key, bySide);
  }
  const ordered = [...periods.entries()].sort((a, b) => a[0].localeCompare(b[0]));
  if (ordered.length < 2) { section.hidden = true; return; }
  const plot = { x0: 48, x1: 950, y0: 10, y1: 166 };
  const plotHeight = plot.y1 - plot.y0;
  let metric:'ships'|'isk'='ships',bars:Bar[]=[],width=960,step=0;
  const fmtTick = (value:number) => metric === 'isk' ? compactISK(value) : value >= 1e6 ? `${(value / 1e6).toFixed(1)}m` : value >= 1e3 ? `${(value / 1e3).toFixed(value >= 1e4 ? 0 : 1)}k` : `${Math.round(value)}`;
  const fullDate = (period:string) => new Date(`${period}T00:00:00Z`).toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric', timeZone: 'UTC' });
  const render = () => {
    width=Math.max(240,Math.min(960,svg.getBoundingClientRect().width||960));plot.x1=width-10;step=(plot.x1-plot.x0)/ordered.length;svg.setAttribute('viewBox',`0 0 ${width} 190`);
    title.textContent = `Activity — ${metric === 'isk' ? 'ISK' : 'ships'} ${source.area ? 'destroyed' : oneSided ? 'destroyed vs lost' : 'lost per side'} per ${source.granularity || 'day'}`;
    section.querySelectorAll<HTMLButtonElement>('[data-campaign-chart-metric]').forEach(button => button.classList.toggle('active', button.dataset.campaignChartMetric === metric));
    const barWidth = Math.max(Math.min(step * .72, 28), 1.5);
    const rawMax = Math.max(...ordered.map(([, bySide]) => [...bySide.values()].reduce((sum, entry) => sum + entry[metric], 0)), 1);
    const max = niceCeil(rawMax);
    bars = ordered.map(([period, bySide], index) => {
      const values = [...bySide.entries()].map(([side, entry]) => ({ side, ...entry, value: entry[metric] })).filter(entry => entry.value > 0).sort((a, b) => a.side - b.side);
      const total = values.reduce((sum, entry) => sum + entry.value, 0);
      const stackHeight = total / max * plotHeight;
      const gaps = 2 * Math.max(values.length - 1, 0);
      const scale = stackHeight > gaps ? (stackHeight - gaps) / stackHeight : 0;
      let cursor = plot.y1;
      const segments = values.map((entry, position) => {
        const height = Math.max(entry.value / max * plotHeight * scale, 1);
        cursor -= height;
        const segment = { ...entry, y: cursor, height, topmost: position === values.length - 1 };
        cursor -= 2;
        return segment;
      });
      return { period, x: plot.x0 + index * step + (step - barWidth) / 2, width: barWidth, segments,
        totalShips: values.reduce((sum, entry) => sum + entry.ships, 0), totalIsk: values.reduce((sum, entry) => sum + entry.isk, 0) };
    });
    const yearLabels = ordered[0][0].slice(0, 4) !== ordered[ordered.length-1][0].slice(0, 4);
    const labelStep = Math.max(1, Math.ceil(bars.length / Math.min(7,Math.max(2,Math.floor((width-60)/90)))));
    const fmtX = (period:string) => new Date(`${period}T00:00:00Z`).toLocaleDateString('en-US', yearLabels ? { month: 'short', day: 'numeric', year: '2-digit', timeZone: 'UTC' } : { month: 'short', day: 'numeric', timeZone: 'UTC' });
    svg.innerHTML = [
      ...[1, 2, 3, 4].map(tick => { const y = plot.y1 - tick / 4 * plotHeight; return `<line x1="${plot.x0}" x2="${plot.x1}" y1="${y}" y2="${y}" stroke="rgba(255,255,255,.05)"/><text x="${plot.x0 - 8}" y="${y + 3}" text-anchor="end" fill="#6b7280" font-size="10">${svgEscape(fmtTick(max * tick / 4))}</text>`; }),
      `<line x1="${plot.x0}" x2="${plot.x1}" y1="${plot.y1}" y2="${plot.y1}" stroke="rgba(255,255,255,.1)"/>`,
      `<rect data-campaign-chart-highlight x="${plot.x0}" y="${plot.y0}" width="${step}" height="${plotHeight}" fill="rgba(255,255,255,.04)" rx="3" style="display:none"/>`,
      ...bars.map((bar, index) => `<g data-campaign-chart-bar="${index}">${bar.segments.map(segment => segment.topmost ? `<path d="${roundedTop(bar.x, segment.y, bar.width, segment.height)}" fill="${colorOf(segment.side)}" fill-opacity=".9"/>` : `<rect x="${bar.x}" y="${segment.y}" width="${bar.width}" height="${segment.height}" fill="${colorOf(segment.side)}" fill-opacity=".9"/>`).join('')}</g>`),
      ...bars.filter((_, index) => index % labelStep === 0).map(bar => `<text x="${bar.x + bar.width / 2}" y="${plot.y1 + 16}" text-anchor="middle" fill="#6b7280" font-size="10">${svgEscape(fmtX(bar.period))}</text>`),
      ...bars.map((bar, index) => `<rect data-campaign-chart-target="${index}" x="${plot.x0 + index * step}" y="${plot.y0}" width="${step}" height="${plotHeight}" fill="transparent"/>`),
    ].join('');
    tooltip.hidden = true;
  };
  const highlightTarget=()=>svg.querySelector<SVGElement>('[data-campaign-chart-highlight]');
  function hover(event:PointerEvent){const target=eventElement(event.target)?.closest<SVGElement>('[data-campaign-chart-target]');if(!target)return;
      const index = Number(target.dataset.campaignChartTarget), bar = bars[index];if(!bar||!highlightTarget())return;
      const highlight = highlightTarget();if(!highlight)return;
      highlight.setAttribute('x', String(plot.x0 + index * step));
      highlight.style.display = '';
      svg.querySelectorAll<SVGElement>('[data-campaign-chart-bar]').forEach(group => { group.style.opacity = Number(group.dataset.campaignChartBar) === index ? '1' : '.4'; });
      tooltip.innerHTML = `<strong>${svgEscape(source.granularity === 'week' ? 'Week of ' + fullDate(bar.period) : fullDate(bar.period))}</strong>${[...bar.segments].reverse().map(segment => `<span><i style="background:${colorOf(segment.side)}"></i><b>${svgEscape(nameOf(segment.side))}</b><em>${Math.round(segment.ships).toLocaleString('en-US')}</em><small>${compactISK(segment.isk)}</small></span>`).join('')}<span class="total"><i></i><b>Total</b><em>${Math.round(bar.totalShips).toLocaleString('en-US')}</em><small>${compactISK(bar.totalIsk)}</small></span>`;
      tooltip.hidden = false;
      const renderedWidth=svg.getBoundingClientRect().width||width,min=Math.min(50,Math.max(12,(tooltip.getBoundingClientRect().width||200)/2/renderedWidth*100+1));
      const percentage=Math.min(Math.max((bar.x+bar.width/2)/width*100,min),100-min);tooltip.style.left=`${percentage}%`;
  }
  function leave(){tooltip.hidden=true;const highlight=highlightTarget();if(highlight)highlight.style.display='none';for(const group of svg.querySelectorAll<SVGElement>('[data-campaign-chart-bar]'))group.style.opacity='1';}
  function click(event:MouseEvent){const next=eventElement(event.target)?.closest<HTMLElement>('[data-campaign-chart-metric]')?.dataset.campaignChartMetric;if(next==='ships'||next==='isk'){metric=next;render();}}
  if (source.area) legend.hidden = true;
  else {
    const items = oneSided ? [{ name: 'Kills', color: colorOf(100) }, { name: 'Losses', color: colorOf(101) }] : sides.map(side => ({ name: side.name, color: colorOf(Number(side.index)) }));
    legend.replaceChildren(...items.map(item => { const span = document.createElement('span'); const marker = document.createElement('i'); marker.style.backgroundColor = item.color; span.append(marker, document.createTextNode(item.name)); return span; }));
  }
  render();
  return {click,hover,leave,render};
}

export class CampaignActivityController extends Controller<HTMLElement>{
  private ui:ReturnType<typeof createActivity>|undefined
  private observer:ResizeObserver|undefined
  connect(){this.ui??=createActivity(this.element);if(this.ui){this.observer=new ResizeObserver(()=>this.ui?.render());this.observer.observe(requiredElement(this.element,'.campaign-detail-chart'))}}
  click(event:MouseEvent){this.ui?.click(event)}
  hover(event:PointerEvent){this.ui?.hover(event)}
  leave(){this.ui?.leave()}
  disconnect(){this.observer?.disconnect();this.observer=undefined;this.leave()}
}
registerController('campaign-activity',CampaignActivityController)
