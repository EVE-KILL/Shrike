import {escapeHTML} from './shared/html';
import {Controller} from '@hotwired/stimulus';
import {registerController} from './controllers/application';
import {requiredElement,eventElement} from './shared/dom';
import {readFwSystemsScript} from './shared/war-model';
import type {LaidNode} from '../utils/fwWarzone';
import { buildFwLayout, buildFwMiniSystems, factionColor, systemHaloFill, contestedGlowFill, nodeOpacity, R, RING_EXTRA, SUN_R, PLANET_R, secColorStr, secLabel, formatIsk } from './faction-war-data.js';

// The SVG layers, dimensions, territory geometry and colors come from Nuxt's
// WarzoneMap. The shared pure module is also used by the Vue component.
function createMap(root:HTMLElement) {
  const scope=root.closest<HTMLElement>('[data-fw-systems-loaded]');if(!scope)throw new Error('Faction map data is unavailable');
  const data=readFwSystemsScript(requiredElement<HTMLElement>(scope,'#faction-war-systems-data'));
  const identity=root.closest<HTMLElement>('[data-faction-war]')?.dataset??{};
  const q=<T extends Element=HTMLElement>(selector:string)=>requiredElement<T>(root,selector);
  const optionalSvg=root.querySelector<SVGSVGElement>('[data-fw-map-svg]');
  for(const side of [1,2]) {
    q<HTMLElement>(`[data-fw-map-name="${side}"]`).textContent=identity[`side${side}Name`]??'';
    q<HTMLElement>(`[data-fw-map-dot="${side}"]`).style.background=factionColor(Number(identity[`side${side}Id`]));
  }
  if(!optionalSvg)return;const svg=optionalSvg;
  const layout=buildFwLayout(data), nodes=new Map(layout.nodes.map(n=>[n.id,n]));
  const mini=buildFwMiniSystems(data.celestials || []), container=q<HTMLElement>('[data-fw-map-container]');
  const card=q<HTMLElement>('[data-fw-map-hover]'),reset=q<HTMLElement>('[data-fw-map-reset]'),indicator=q<HTMLElement>('[data-fw-map-zoom]');
  const ns='http://www.w3.org/2000/svg';
  function element<K extends keyof SVGElementTagNameMap>(tag:K,attrs:Record<string,string|number>,parent:SVGElement,text?:string) {
    const node=document.createElementNS(ns,tag);
    for(const [key,value] of Object.entries(attrs))node.setAttribute(key,String(value));
    if(text!==undefined)node.textContent=text;
    parent.append(node);return node;
  }
  const defs=element('defs',{},svg),filter=element('filter',{id:'fwCellBlur',x:'-10%',y:'-10%',width:'120%',height:'120%'},defs);
  element('feGaussianBlur',{stdDeviation:6},filter);
  const territories=element('g',{filter:'url(#fwCellBlur)',class:'pointer-events-none'},svg);
  layout.cells.forEach(cell=>element('path',{d:cell.path,fill:factionColor(cell.occupier_faction_id),'fill-opacity':.22},territories));
  for(const link of layout.links) {
    const a=nodes.get(link.from),b=nodes.get(link.to);if(!a||!b)continue;const frontline=a.occupier_faction_id!==b.occupier_faction_id;
    element('line',{x1:a.x,y1:a.y,x2:b.x,y2:b.y,stroke:frontline?'#f87171':factionColor(a.occupier_faction_id),'stroke-opacity':frontline?.6:.22,'stroke-width':1.25,class:'pointer-events-none'},svg);
  }
  let zoom=1,panX=0,panY=0,panning=false,panStart:{x:number;y:number}|null=null,panOrigin:{x:number;y:number}|null=null,panMoved=false,hoverTimer:ReturnType<typeof setTimeout>|undefined,hoverNode:LaidNode|null=null,hoverPos={x:0,y:0},pointerID:number|undefined;
  const labels:Array<{n:LaidNode;name:SVGTextElement;sec:SVGTextElement}>=[];
  for(const n of layout.nodes) {
    const g=element('g',{opacity:nodeOpacity(n),class:'cursor-pointer','data-fw-map-system':n.id,role:'link','aria-label':`${n.system_name} ${secLabel(n.security)}`,tabindex:0},svg);
    const glow=contestedGlowFill(n);
    if(glow)element('circle',{cx:n.x,cy:n.y,r:R+RING_EXTRA,fill:glow,class:'pointer-events-none'},g);
    element('circle',{cx:n.x,cy:n.y,r:R+1,fill:systemHaloFill(n),class:'pointer-events-none'},g);
    element('circle',{cx:n.x,cy:n.y,r:R,fill:secColorStr(n.security),'fill-opacity':.18,stroke:secColorStr(n.security),'stroke-opacity':.6,'stroke-width':.3,class:'hover:brightness-125 transition-[filter]'},g);
    for(const planet of mini.get(n.id)?.planets || [])element('circle',{cx:n.x+planet.dx,cy:n.y+planet.dy,r:PLANET_R,fill:'#cbd5e1','fill-opacity':.85,class:'pointer-events-none'},g);
    if(mini.get(n.id)?.hasSun)element('circle',{cx:n.x,cy:n.y,r:SUN_R,fill:'#fbbf24',class:'pointer-events-none'},g);
    const name=element('text',{x:n.x,y:n.y+R+4,'text-anchor':'middle',fill:'rgba(255,255,255,0.85)','font-size':5,'font-weight':600,class:'select-none pointer-events-none'},g,n.system_name);
    const sec=element('text',{x:n.x,y:n.y+R+10,'text-anchor':'middle',fill:secColorStr(n.security),'font-size':4,class:'select-none pointer-events-none'},g,secLabel(n.security));
    labels.push({n,name,sec});
  }
  function view() {
    const b=layout.bbox,w=b.w/zoom,h=b.h/zoom;
    svg.setAttribute('viewBox',`${b.x+(b.w-w)/2+panX} ${b.y+(b.h-h)/2+panY} ${w} ${h}`);
    reset.hidden=zoom===1&&panX===0&&panY===0;
    indicator.hidden=zoom===1;indicator.textContent=`${Math.round(zoom*100)}%`;
    for(const {n,name,sec} of labels) {
      const visible=zoom>=1.8||n.kills_24h>0||n.contested!=='uncontested';
      name.style.display=sec.style.display=visible?'':'none';
    }
  }
  function point(event:WheelEvent) {
    const p=svg.createSVGPoint();p.x=event.clientX;p.y=event.clientY;
    const matrix=svg.getScreenCTM()?.inverse();return matrix?p.matrixTransform(matrix):{x:0,y:0};
  }
  function wheel(event:WheelEvent) {
    if(!svg?.contains(eventElement(event.target)))return;event.preventDefault();const before=point(event),next=Math.min(20,Math.max(.5,zoom*(event.deltaY<0?1.15:1/1.15)));
    if(next===zoom)return;zoom=next;view();const after=point(event);panX+=before.x-after.x;panY+=before.y-after.y;view();hideCard();
  }
  function down(event:PointerEvent) {
    if(event.button!==0||!svg?.contains(eventElement(event.target)))return;
    panning=true;panMoved=false;panStart=panOrigin={x:event.clientX,y:event.clientY};hideCard();
    svg.classList.replace('cursor-grab','cursor-grabbing');
    pointerID=event.pointerId;svg.setPointerCapture(event.pointerId);
  }
  function move(event:PointerEvent) {
    if(!panning||!panStart||!panOrigin||event.pointerId!==pointerID)return;
    if(!panMoved) {const dx=event.clientX-panOrigin.x,dy=event.clientY-panOrigin.y;if(dx*dx+dy*dy<9)return;panMoved=true;}
    const rect=svg.getBoundingClientRect();
    panX-=(event.clientX-panStart.x)*(rect.width?layout.bbox.w/zoom/rect.width:1);
    panY-=(event.clientY-panStart.y)*(rect.height?layout.bbox.h/zoom/rect.height:1);
    panStart={x:event.clientX,y:event.clientY};view();
  }
  function up(event?:PointerEvent) {
    if(event&&pointerID!==undefined&&event.pointerId!==pointerID)return;
    panning=false;panStart=panOrigin=null;svg.classList.replace('cursor-grabbing','cursor-grab');
    if(pointerID!==undefined&&svg.hasPointerCapture(pointerID))svg.releasePointerCapture(pointerID);pointerID=undefined;
  }
  function hideCard(){clearTimeout(hoverTimer);hoverTimer=undefined;hoverNode=null;card.hidden=true;}
  function positionCard() {
    if(!hoverNode)return;
    const rect=container.getBoundingClientRect(),x=hoverPos.x-rect.left,y=hoverPos.y-rect.top;
    let left=x+14,top=y+14;if(left+260>rect.width)left=x-14-260;if(top+220>rect.height)top=y-14-220;
    card.style.left=`${Math.max(4,left)}px`;card.style.top=`${Math.max(4,top)}px`;
  }
  const escape=escapeHTML;
  function showCard(n:LaidNode) {
    const statusClass=n.contested==='contested'?'text-yellow-400':n.contested==='vulnerable'?'text-red-400':'text-gray-400';
    card.innerHTML=`<div class="flex items-baseline justify-between gap-2 mb-1"><span class="text-white font-semibold text-sm truncate">${escape(n.system_name)}</span><span class="flex items-center gap-1.5"><span style="color:${secColorStr(n.security)}" class="font-mono font-semibold">${secLabel(n.security)}</span><span class="w-2 h-2 rounded-full" style="background:${factionColor(n.occupier_faction_id)}"></span></span></div><div class="text-gray-500 truncate mb-2">${escape(n.region_name)} <span class="text-gray-600">·</span> ${escape(n.constellation_name)}</div><div class="flex items-center gap-2 mb-2"><span class="${statusClass}">${escape(n.contested)}</span></div>${n.victory_points_threshold>0?`<div class="mb-2"><div class="flex justify-between text-gray-500 mb-0.5"><span>VP</span><span class="tabular-nums">${n.victory_points.toLocaleString()} / ${n.victory_points_threshold.toLocaleString()}</span></div><div class="w-full h-1.5 bg-white/10 rounded-full overflow-hidden"><div class="h-full rounded-full ${n.contested==='vulnerable'?'bg-red-500':'bg-amber-500'}" style="width:${Math.min(100,n.victory_points/n.victory_points_threshold*100)}%"></div></div></div>`:''}${n.kills_24h>0?`<div class="text-gray-400"><span class="text-white tabular-nums">${n.kills_24h}</span> kills · <span class="text-isk tabular-nums">${formatIsk(n.isk_24h)}</span> (24h)</div>`:''}`;
    card.hidden=false;
  }
  function click(event:MouseEvent){const target=eventElement(event.target);if(!target)return;if(panMoved){event.stopPropagation();event.preventDefault();panMoved=false;return}if(target.closest('[data-fw-map-reset]')){zoom=1;panX=panY=0;view();hideCard();return}const node=target.closest<SVGGElement>('[data-fw-map-system]');if(node&&nodes.has(Number(node.dataset.fwMapSystem)))location.assign(`/system/${node.dataset.fwMapSystem}`);}
  function keyboard(event:KeyboardEvent){const node=eventElement(event.target)?.closest<SVGGElement>('[data-fw-map-system]');if(event.key==='Enter'&&node&&nodes.has(Number(node.dataset.fwMapSystem))){event.preventDefault();panMoved=false;location.assign(`/system/${node.dataset.fwMapSystem}`);}}
  function hover(event:PointerEvent|FocusEvent){const target=eventElement(event.target)?.closest<SVGGElement>('[data-fw-map-system]');if(!target||panning)return;const related=eventElement(event.relatedTarget)?.closest('[data-fw-map-system]');if(related===target)return;const n=nodes.get(Number(target.dataset.fwMapSystem));if(!n)return;hideCard();if(event instanceof PointerEvent)hoverPos={x:event.clientX,y:event.clientY};else{const rect=target.getBoundingClientRect();hoverPos={x:rect.left+rect.width/2,y:rect.top+rect.height/2}}const show=()=>{if(!root.isConnected||panning)return;hoverNode=n;showCard(n);positionCard()};if(event instanceof FocusEvent)show();else hoverTimer=setTimeout(show,180);}
  function hoverMove(event:PointerEvent){if(panning)return;hoverPos={x:event.clientX,y:event.clientY};positionCard();}
  function leave(event:PointerEvent|FocusEvent){const node=eventElement(event.target)?.closest('[data-fw-map-system]');if(node&&eventElement(event.relatedTarget)?.closest('[data-fw-map-system]')!==node)hideCard();}
  function hide(){up();hideCard();panMoved=false;}
  view();return {click,keyboard,hover,hoverMove,leave,wheel,down,move,up,hide};
}
export class FactionWarMapController extends Controller<HTMLElement>{
 private ui:ReturnType<typeof createMap>;
 private initialized=false;
 connect(){if(!this.initialized){this.ui=createMap(this.element);this.initialized=true}}
 click(event:MouseEvent){this.ui?.click(event)}
 keyboard(event:KeyboardEvent){this.ui?.keyboard(event)}
 hover(event:PointerEvent|FocusEvent){this.ui?.hover(event)}
 hoverMove(event:PointerEvent){this.ui?.hoverMove(event)}
 leave(event:PointerEvent|FocusEvent){this.ui?.leave(event)}
 wheel(event:WheelEvent){this.ui?.wheel(event)}
 down(event:PointerEvent){this.ui?.down(event)}
 move(event:PointerEvent){this.ui?.move(event)}
 up(event:PointerEvent){this.ui?.up(event)}
 hide(){this.ui?.hide()}
 disconnect(){this.hide()}
}
registerController('faction-war-map',FactionWarMapController);
