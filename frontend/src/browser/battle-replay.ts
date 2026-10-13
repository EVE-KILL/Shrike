import { AU, hasReplayPosition, replayPoint, replaySide, replayBounds, replayLossTotals, bigKillClass } from './battle-data.js';

import {Controller} from '@hotwired/stimulus';
import {registerController} from './controllers/application';
import {iconHref} from './icon-hrefs';
import {requiredElement,cloneComponent,eventElement,placeElement} from './shared/dom';
import {readReplaySource,type ReplayLandmark} from './shared/battle-model';
import type {ReplayKill} from '../utils/map/replay';
type ReplayPoint={x:number;y:number;kill:ReplayKill;time:number;side:number};
type Landmark=ReplayLandmark&{px:number;py:number};
type Orbit={x:number;y:number;radius:number};
function createReplay(section:HTMLElement) {
  const data=readReplaySource(JSON.parse(requiredElement(section,'#battle-replay-data').textContent??'null')),kills=data.kills;
  const query=<T extends Element=HTMLElement>(selector:string)=>requiredElement<T>(section,selector),get=<T extends Element=HTMLElement>(scope:ParentNode,selector:string)=>requiredElement<T>(scope,selector),plot=query<SVGSVGElement>('[data-replay-plot]'),stage=query('[data-replay-stage]'),feed=query('[data-replay-feed]'),leader=query<SVGSVGElement>('[data-replay-leader]');
  const clock=(ms:number)=>new Date(ms).toISOString().slice(11,19),num=(v:number)=>Number(v??0).toLocaleString('en-US');
  const isk=(v:number)=>v>=1e12?(v/1e12).toFixed(2)+'t':v>=1e9?(v/1e9).toFixed(2)+'b':v>=1e6?(v/1e6).toFixed(2)+'m':v>=1e3?(v/1e3).toFixed(1)+'k':Number(v).toFixed(0);
  const palette=['#f87171','#60a5fa','#c084fc','#34d399','#fbbf24'],color=(side:number)=>side<0?'#94a3b8':palette[side%palette.length];
  const svg=<K extends keyof SVGElementTagNameMap>(tag:K,attrs:Record<string,string|number>={},label:string|null=null)=>{const node=document.createElementNS('http://www.w3.org/2000/svg',tag);for(const [key,value]of Object.entries(attrs))node.setAttribute(key,String(value));if(label!==null)node.textContent=label;return node;};
  const setText=(selector:string,text:string|null)=>{query(selector).textContent=text;};
  let system:number|null=Number(query<HTMLSelectElement>('[data-replay-system]').value)||null,time=data.time,selected:number|null=data.selected||null,hover:number|null=null,playing=false,speed=60,bounds=replayBounds([]),points:ReplayPoint[]=[],landmarks:Landmark[]=[],orbits:Orbit[]=[],systemKills:ReplayKill[]=[],geometryDirty=true,lastFeed:string|null='';
  const pointNodes=new Map<number,{anchor:SVGAElement;circle:SVGCircleElement}>(),feedNodes=new Map<number,HTMLElement>();
  const show={orbits:true,moons:true,labels:true};
  const landmarkColors:Record<number,string>={6:'#fbbf24',7:'#7dd3fc',8:'#64748b',9:'#78716c',10:'#34d399',15:'#c084fc'},landmarkGroups:Record<number,string>={6:'Sun',7:'Planet',8:'Moon',9:'Belt',10:'Gate',15:'Station'};
  const landmarkColor=(group:number)=>landmarkColors[group]??'#94a3b8';
  const systemName=()=>query<HTMLSelectElement>('[data-replay-system]').selectedOptions[0]?.textContent??'System';
  const landmarkName=(l:Landmark)=>(l.name??`Object ${l.id}`).replace(`${systemName()} `,'').replace(/^- /,'');
  const fitKills=()=>{bounds=replayBounds(points);geometryDirty=true;};
  const fitSystem=()=>{bounds=replayBounds([...points,...landmarks.map(l=>({x:l.px,y:l.py})),...orbits.flatMap(o=>[{x:o.x-o.radius,y:o.y-o.radius},{x:o.x+o.radius,y:o.y+o.radius}])]);geometryDirty=true;};
  function loadSystem() {
    systemKills=kills.filter(k=>k.solar_system_id===system);
    points=systemKills.filter(hasReplayPosition).map(kill=>({...replayPoint(kill,'xz'),kill,time:Date.parse(kill.killmail_time),side:replaySide(kill,data.team_entities??[])}));
    landmarks=(data.landmarks??[]).filter(l=>l.solar_system_id===system).map(l=>({...l,px:l.x/AU,py:-l.z/AU}));
    const sun=landmarks.find(l=>l.group_id===6);orbits=sun?landmarks.filter(l=>l.group_id===7).map(l=>({x:sun.px,y:sun.py,radius:Math.hypot(l.px-sun.px,l.py-sun.py)})).filter(o=>o.radius>0):[];
    query('[data-replay-no-celestials]').hidden=landmarks.length>0;query('[data-replay-no-positions]').hidden=points.length>0;
    setText('[data-replay-system-name]',systemName());setText('[data-replay-position-count]',`${points.length} of ${systemKills.length} kills have positions`);setText('[data-replay-celestial-count]',`◇ Gate · □ Station · ${landmarks.length} celestials`);
    pointNodes.clear();query('[data-replay-points]').replaceChildren();
    for(const point of points){const anchor=svg('a',{href:`/kill/${point.kill.killmail_id}`,'data-replay-point':point.kill.killmail_id}),circle=svg('circle',{cx:point.x,cy:point.y,fill:color(point.side),'fill-opacity':'.38'});circle.append(svg('title',{},`${point.kill.ship_name} · ${point.kill.victim_character_name||'Unknown pilot'} · ${clock(point.time)} EVE`));anchor.append(circle);query('[data-replay-points]').append(anchor);pointNodes.set(point.kill.killmail_id,{anchor,circle});}
    const bins=Array.from({length:90},()=>({count:0,big:false}));
    for(const kill of systemKills){const index=Math.max(0,Math.min(89,Math.floor((Date.parse(kill.killmail_time)-data.start)/(data.end-data.start)*90)));bins[index].count++;if(bigKillClass(kill))bins[index].big=true;}
    const max=Math.max(1,...bins.map(b=>b.count));
    query('[data-replay-histogram]').replaceChildren(...bins.map((bin,index)=>{const button=document.createElement('button');button.type='button';button.className='min-w-0 flex-1 rounded-t-sm transition-colors hover:bg-sky-200';button.style.height=`${3+Math.sqrt(bin.count/max)*29}px`;button.dataset.replayBin=String(index);button.dataset.big=String(bin.big);button.title=`${clock(data.start+index/90*(data.end-data.start))} · ${bin.count} kills`;button.setAttribute('aria-label',`Jump to ${clock(data.start+index/90*(data.end-data.start))}: ${bin.count} kills`);return button;}));
    feedNodes.clear();lastFeed=null;fitSystem();
  }
  function drawGeometry() {
    plot.setAttribute('viewBox',`${bounds.x} ${bounds.y} ${bounds.width} ${bounds.height}`);
    setText('[data-replay-span]',bounds.width>=.01?`${bounds.width.toFixed(2)} AU across`:`${num(Math.round(bounds.width*149597870.7))} km across`);
    query('[data-replay-orbits]').replaceChildren(...(show.orbits?orbits.map(o=>svg('circle',{cx:o.x,cy:o.y,r:o.radius,'vector-effect':'non-scaling-stroke'})):[]));
    const visible=landmarks.filter(l=>(show.moons||l.group_id!==8)&&l.px>=bounds.x&&l.px<=bounds.x+bounds.width&&l.py>=bounds.y&&l.py<=bounds.y+bounds.height);
    const labels=new Set<number>(),occupied:Array<{x:number;y:number}>=[],priorities=[6,10,15,7,8,9];
    if(show.labels)for(const l of [...visible].sort((a,b)=>priorities.indexOf(a.group_id)-priorities.indexOf(b.group_id))){const x=(l.px-bounds.x)/bounds.width*1000,y=(l.py-bounds.y)/bounds.height*600;if(occupied.some(p=>Math.abs(p.x-x)<125&&Math.abs(p.y-y)<22))continue;occupied.push({x,y});labels.add(l.id);}
    query('[data-replay-landmarks]').replaceChildren(...visible.map(l=>{
      const g=svg('g',{transform:`translate(${l.px} ${l.py})`});g.append(svg('title',{},`${(landmarkGroups[l.group_id]??'Celestial')} · ${l.name}`));
      if(l.group_id===6){g.append(svg('circle',{r:bounds.width*.055,fill:'url(#replay-sun-glow)'}),svg('circle',{r:bounds.width*.02,fill:'none',stroke:'#fbbf24','stroke-opacity':'.12','vector-effect':'non-scaling-stroke'}));}
      if(l.group_id===10){const r=bounds.width*.006;g.append(svg('path',{d:`M 0 ${-r} L ${r} 0 L 0 ${r} L ${-r} 0 Z`,fill:'#08090d',stroke:landmarkColor(l.group_id),'vector-effect':'non-scaling-stroke'}));}
      else if(l.group_id===15){const r=bounds.width*.005;g.append(svg('rect',{x:-r,y:-r,width:r*2,height:r*2,fill:'#08090d',stroke:landmarkColor(l.group_id),'vector-effect':'non-scaling-stroke'}));}
      else g.append(svg('circle',{r:bounds.width*(l.group_id===6?.007:l.group_id===7?.004:.002),fill:l.group_id===7?'url(#replay-planet)':l.group_id===6?'#fff2c4':landmarkColor(l.group_id),'fill-opacity':l.group_id===8?.5:1}));
      if(labels.has(l.id))g.append(svg('text',{x:bounds.width*.01,y:bounds.width*.004,'font-size':bounds.width*.012,fill:landmarkColor(l.group_id),stroke:'#08090d','stroke-width':bounds.width*.003,'paint-order':'stroke','font-family':'sans-serif'},landmarkName(l)));return g;
    }));
    for(const {circle} of pointNodes.values())circle.setAttribute('r',String(bounds.width*.0025));geometryDirty=false;
  }
  function drawLeader() {
    const id=hover??selected??(playing?systemKills.filter(k=>Date.parse(k.killmail_time)<=time).at(-1)?.killmail_id:null);
    for(const [killId,node]of feedNodes)node.classList.toggle('bg-sky-300/[0.07]',killId===id);
    const point=points.find(p=>p.kill.killmail_id===id&&p.time<=time);leader.replaceChildren();if(!point)return;
    const sr=stage.getBoundingClientRect(),pr=plot.getBoundingClientRect(),fr=feed.getBoundingClientRect(),rr=feedNodes.get(point.kill.killmail_id)?.getBoundingClientRect();
    const rowVisible=rr&&rr.top>=fr.top&&rr.bottom<=fr.bottom,originX=(rowVisible?rr.left:fr.left)-sr.left,originY=(rowVisible?rr.top+rr.height/2:fr.top+24)-sr.top;
    const scale=Math.min(pr.width/bounds.width,pr.height/bounds.height),left=pr.left-sr.left,top=pr.top-sr.top;
    const x=left+(pr.width-bounds.width*scale)/2+(point.x-bounds.x)*scale,y=top+(pr.height-bounds.height*scale)/2+(point.y-bounds.y)*scale;
    if(x<left||x>left+pr.width||y<top||y>top+pr.height)return;
    leader.setAttribute('viewBox',`0 0 ${sr.width} ${sr.height}`);const bend=Math.max(50,Math.abs(originX-x)*.35),path=`M ${originX} ${originY} C ${originX-bend} ${originY}, ${x+bend} ${y}, ${x} ${y}`,c=color(point.side);
    leader.append(svg('path',{d:path,fill:'none',stroke:c,'stroke-width':9,opacity:.12,class:'replay-beam-glow'}),svg('path',{d:path,fill:'none',stroke:c,'stroke-width':1.4,opacity:.8}),svg('path',{d:path,fill:'none',stroke:'#fff','stroke-width':1.6,'stroke-dasharray':'4 24',opacity:.7,class:'replay-beam-flow'}),svg('circle',{cx:x,cy:y,r:22,stroke:c,'stroke-width':1,fill:'none',opacity:.7}),svg('circle',{cx:x,cy:y,r:5,fill:'#fff'}),svg('path',{d:`M ${x-30} ${y} h 10 M ${x+20} ${y} h 10 M ${x} ${y-30} v 10 M ${x} ${y+20} v 10`,stroke:c,'stroke-width':1.5}));
    const g=svg('g',{transform:`translate(${Math.max(left+12,Math.min(x+23,left+pr.width-170))} ${Math.max(top+90,Math.min(y-22,top+pr.height-65))})`});g.append(svg('rect',{x:-7,y:-16,width:166,height:40,rx:5,fill:'#080e1b','fill-opacity':'.94',stroke:c,'stroke-opacity':'.35'}),svg('text',{fill:c,'font-size':11,'font-weight':600},(point.kill.ship_name||'Unknown ship').slice(0,24)),svg('text',{fill:'#94a3b8','font-size':10,y:15},`${isk(point.kill.total_value)} ISK · ${clock(point.time)}`));leader.append(g);
  }
  function renderSelected() {
    const kill=kills.find(k=>k.killmail_id===selected),panel=query('[data-replay-selected]');panel.hidden=!kill;if(!kill)return;
    setText('[data-replay-selected-name]',`${kill.ship_name} · ${kill.victim_character_name||'Unknown pilot'}`);setText('[data-replay-selected-value]',`${isk(kill.total_value)} ISK`);query<HTMLAnchorElement>('[data-replay-open]').href=`/kill/${kill.killmail_id}`;
    const positioned=hasReplayPosition(kill);query('[data-replay-position-unavailable]').hidden=positioned;query('[data-replay-focus]').hidden=!positioned;
    let nearest:Landmark|null=null,distance=Infinity;if(positioned)for(const l of landmarks){const metres=Math.hypot(kill.position_x-l.x,kill.position_y-l.y,kill.position_z-l.z);if(metres<distance){distance=metres;nearest=l;}}
    setText('[data-replay-nearest]',nearest?`${distance>AU/100?(distance/AU).toFixed(2)+' AU':num(Math.round(distance/1000))+' km'} from ${nearest.name}`:'');
  }
  function render() {
    if(geometryDirty)drawGeometry();
    const occurred=systemKills.filter(k=>Date.parse(k.killmail_time)<=time),recent=occurred.slice(-30).reverse(),totals=replayLossTotals(systemKills,data.team_entities??[],time);
    const cards=query('[data-replay-loss-cards]');let unsided=cards.querySelector<HTMLElement>('[data-replay-loss-side="-1"]');
    if(totals.unassigned.losses&&!unsided){unsided=cloneComponent(section,'[data-replay-loss-card-template]');cards.append(unsided);}if(!totals.unassigned.losses&&unsided){unsided.remove();unsided=null;}
    cards.classList.toggle('lg:grid-cols-4',cards.children.length>2);
    for(const card of cards.querySelectorAll<HTMLElement>(':scope > [data-replay-loss-side]')){const i=Number(card.dataset.replayLossSide),total=i<0?totals.unassigned:totals.teams[i]??{losses:0,isk:0};get<HTMLElement>(card,'[data-replay-loss-count]').textContent=num(total.losses);get<HTMLElement>(card,'[data-replay-loss-isk]').textContent=isk(total.isk);}
    for(const node of section.querySelectorAll<HTMLElement>('[data-replay-clock]'))node.textContent=clock(time);query<HTMLInputElement>('[data-replay-time]').value=String(time);
    const play=query('[data-replay-play]');get<HTMLElement>(play,'span').textContent=playing?'Pause':'Play';const use=get<SVGUseElement>(play,'use');use.setAttribute('href',iconHref(playing?'pause':'play'));
    setText('[data-replay-playing-label]',playing?'Battle replay · playing':'Battle replay · paused');query('[data-replay-playing-dot]').classList.toggle('bg-rose-400',playing);query('[data-replay-playing-dot]').classList.toggle('animate-pulse',playing);query('[data-replay-playing-dot]').classList.toggle('bg-sky-400',!playing);
    setText('[data-replay-speed-label]',`EVE time · ${speed}×`);setText('[data-replay-occurred]',num(occurred.length));setText('[data-replay-destroyed]',isk(occurred.reduce((sum,k)=>sum+k.total_value,0)));
    for(const p of points){const node=pointNodes.get(p.kill.killmail_id);if(node)node.anchor.style.display=p.time<=time?'':'none';}
    const pulseWindow=Math.max(30000,speed*3000),fresh=points.filter(p=>p.time<=time&&time-p.time<pulseWindow),pulses=[...fresh.filter(p=>!bigKillClass(p.kill)).slice(-16),...fresh.filter(p=>bigKillClass(p.kill))];
    query('[data-replay-pulses]').replaceChildren(...pulses.map(p=>{const age=Math.min(1,Math.max(0,(time-p.time)/pulseWindow)),r=bounds.width*((bigKillClass(p.kill)?.022:.008)+Math.max(0,Math.log10(Math.max(1,p.kill.total_value))-7)*.002),c=color(p.side),g=svg('g',{transform:`translate(${p.x} ${p.y})`});g.append(svg('circle',{r:r*2.5,fill:c,opacity:(1-age)*.08}),svg('circle',{r:r*(.7+age*2),fill:'none',stroke:c,'stroke-opacity':(1-age)*.85,'stroke-width':1.8,'vector-effect':'non-scaling-stroke'}),svg('circle',{r:r*(.4+age),fill:'none',stroke:c,'stroke-opacity':(1-age)*.5,'vector-effect':'non-scaling-stroke'}),svg('circle',{r:r*.35,fill:c,opacity:1-age*.7}),svg('circle',{r:bounds.width*.002,fill:'#fff',opacity:1-age}));return g;}));
    const key=recent.map(k=>k.killmail_id).join(',');if(key!==lastFeed){
      const nodes=recent.map(kill=>{let node=feedNodes.get(kill.killmail_id);if(!node){node=cloneComponent(section,'[data-replay-feed-template]');node.dataset.replayKill=String(kill.killmail_id);node.style.setProperty('--side',color(replaySide(kill,data.team_entities??[])));const image=get<HTMLImageElement>(node,'img');image.hidden=!kill.ship_type_id;if(kill.ship_type_id)image.src=`/images/types/${kill.ship_type_id}/icon?size=64`;const big=get<HTMLElement>(node,'[data-replay-row-big]');big.hidden=!bigKillClass(kill);big.textContent=`BIG KILL · ${bigKillClass(kill)}`;get<HTMLElement>(node,'[data-replay-row-ship]').textContent=kill.ship_name||'Unknown ship';get<HTMLElement>(node,'[data-replay-row-ship]').style.color=color(replaySide(kill,data.team_entities??[]));get<HTMLElement>(node,'[data-replay-row-pilot]').textContent=kill.victim_character_name||'Unknown pilot';get<HTMLElement>(node,'[data-replay-row-value]').textContent=isk(kill.total_value);get<HTMLElement>(node,'[data-replay-row-time]').textContent=clock(Date.parse(kill.killmail_time));feedNodes.set(kill.killmail_id,node);}return node;});
      if(!nodes.length){const empty=document.createElement('p');empty.className='p-5 text-xs text-gray-500';empty.textContent='Press play or scrub forward to follow the losses.';feed.replaceChildren(empty);}else{for(let i=0;i<nodes.length;i++)if(feed.children[i]!==nodes[i])placeElement(feed,nodes[i],feed.children[i]??null);while(feed.children.length>nodes.length)feed.lastElementChild?.remove();}
      lastFeed=key;
      const keep=new Set(recent.map(k=>k.killmail_id));for(const id of feedNodes.keys())if(!keep.has(id))feedNodes.delete(id);
    }
    const big=[...occurred].reverse().find(k=>bigKillClass(k)&&time-Date.parse(k.killmail_time)<Math.max(60000,speed*6000));let bigPanel=query('[data-replay-big]');if(big&&bigPanel.dataset.kill!==String(big.killmail_id)){const next=bigPanel.cloneNode(true);if(!(next instanceof HTMLElement))throw new Error("Invalid replay panel");next.dataset.kill=String(big.killmail_id);bigPanel.replaceWith(next);bigPanel=next;}bigPanel.hidden=!big;
    if(big){const image=get<HTMLImageElement>(bigPanel,'img');image.hidden=!big.ship_type_id;if(big.ship_type_id && image.getAttribute('src')!==`/images/types/${big.ship_type_id}/icon?size=64`)image.src=`/images/types/${big.ship_type_id}/icon?size=64`;setText('[data-replay-big-class]',`BIG KILL · ${bigKillClass(big)}`);setText('[data-replay-big-select]',big.ship_name);query('[data-replay-big-select]').dataset.kill=String(big.killmail_id);setText('[data-replay-big-description]',`${isk(big.total_value)} ISK lost · ${clock(Date.parse(big.killmail_time))}`);get<HTMLAnchorElement>(bigPanel,'a').href=`/kill/${big.killmail_id}`;}
    for(const bin of query('[data-replay-histogram]').querySelectorAll<HTMLElement>(':scope > [data-replay-bin]')){const played=data.start+Number(bin.dataset.replayBin)/90*(data.end-data.start)<=time;bin.classList.toggle('bg-amber-400/80',bin.dataset.big==='true');bin.classList.toggle('bg-sky-400/60',bin.dataset.big!=='true'&&played);bin.classList.toggle('bg-slate-600/40',bin.dataset.big!=='true'&&!played);}
    renderSelected();drawLeader();
  }
  function select(id:number){selected=id;playing=false;render();}
  function click(event:MouseEvent){
    const target=eventElement(event.target);if(!target)return;
    const point=target.closest<HTMLElement>('[data-replay-point]');if(point){event.preventDefault();select(Number(point.dataset.replayPoint));return;}
    const row=target.closest<HTMLElement>('[data-replay-kill]');if(row){select(Number(row.dataset.replayKill));return;}
    const bin=target.closest<HTMLElement>('[data-replay-bin]');if(bin){playing=false;selected=null;time=data.start+Number(bin.dataset.replayBin)/90*(data.end-data.start);render();return;}
    if(target.closest<HTMLElement>('[data-replay-play]')){if(time>=data.end)time=data.start;playing=!playing;}
    else if(target.closest<HTMLElement>('[data-replay-restart]')){playing=false;time=data.start;selected=null;}
    else if(target.closest<HTMLElement>('[data-replay-fit-system]'))fitSystem();
    else if(target.closest<HTMLElement>('[data-replay-fit-kills]'))fitKills();
    else if(target.closest<HTMLElement>('[data-replay-focus]')){const kill=kills.find(k=>k.killmail_id===selected);if(kill&&hasReplayPosition(kill)){const p=replayPoint(kill,'xz'),width=.0001;bounds={x:p.x-width/2,y:p.y-width*.3,width,height:width*.6};geometryDirty=true;}}
    else if(target.closest<HTMLElement>('[data-replay-big-select]')){select(Number(query('[data-replay-big-select]').dataset.kill));return;}
    else return;render();
  }
  function input(event:Event){const target=event.target;if(target instanceof HTMLInputElement&&target.matches('[data-replay-time]')){const next=Number(target.value);if(!Number.isFinite(next))return;time=Math.max(data.start,Math.min(data.end,next));playing=false;selected=null;render();}}
  function change(event:Event){const target=event.target;if(!(target instanceof HTMLSelectElement||target instanceof HTMLInputElement))return;if(target.matches('[data-replay-system]')){system=Number(target.value);selected=null;hover=null;loadSystem();}else if(target.matches('[data-replay-speed]'))speed=Number(target.value);else if(target.matches('[data-replay-layer]')){if(target instanceof HTMLInputElement&&(target.name==='orbits'||target.name==='moons'||target.name==='labels'))show[target.name]=target.checked;geometryDirty=true;}else return;render();}
  const hoverAt=(event:MouseEvent|FocusEvent)=>{const target=eventElement(event.target);if(!target)return;const node=target.closest<HTMLElement>('[data-replay-point],[data-replay-kill]');if(node){hover=Number(node.dataset.replayPoint??node.dataset.replayKill);drawLeader();}};
  function mouseout(event:MouseEvent){const target=eventElement(event.target);if(target?.closest('[data-replay-point],[data-replay-kill]')&&!eventElement(event.relatedTarget)?.closest('[data-replay-point],[data-replay-kill]')){hover=null;drawLeader();}}
  function focusout(){hover=null;drawLeader();}
  let drag:{x:number;y:number;pointer:number}|null=null;
  function down(event:PointerEvent){if(event.button!==0||eventElement(event.target)?.closest('a'))return;drag={x:event.clientX,y:event.clientY,pointer:event.pointerId};plot.setPointerCapture(event.pointerId);}
  function move(event:PointerEvent){if(!drag||drag.pointer!==event.pointerId)return;const rect=plot.getBoundingClientRect(),scale=Math.min(rect.width/bounds.width,rect.height/bounds.height);if(!(scale>0))return;bounds={...bounds,x:bounds.x-(event.clientX-drag.x)/scale,y:bounds.y-(event.clientY-drag.y)/scale};drag={x:event.clientX,y:event.clientY,pointer:event.pointerId};geometryDirty=true;render();}
  function up(){if(drag&&plot.hasPointerCapture(drag.pointer))plot.releasePointerCapture(drag.pointer);drag=null;}
  const observer=new ResizeObserver(drawLeader);observer.observe(stage);
  function tick(delta:number){if(playing){time=Math.min(data.end,time+Math.min(delta,250)*speed);if(time>=data.end)playing=false;render();}}
  function pause(){playing=false;up();render();}
  function resume(){observer.observe(stage);geometryDirty=true;render();}
  loadSystem();render();return {click,input,change,hoverAt,mouseout,focusout,drawLeader,down,move,up,tick,pause,resume,playing:()=>playing,dispose:()=>observer.disconnect()};
}
export class BattleReplayController extends Controller<HTMLElement>{
 private ui:ReturnType<typeof createReplay>|undefined
 private frame:number|undefined
 private last=0
 connect(){if(!this.ui)this.ui=createReplay(this.element);else this.ui.resume()}
 private loop=(now:number)=>{this.frame=undefined;if(!this.element.isConnected||document.hidden||!this.ui?.playing())return;const delta=now-this.last;if(delta>=50){this.last=now;this.ui.tick(delta)}this.schedule()}
 private schedule(){if(this.frame===undefined&&this.ui?.playing()&&!document.hidden){this.frame=requestAnimationFrame(this.loop)}}
 private update(){if(this.ui?.playing()){if(this.frame===undefined)this.last=performance.now();this.schedule()}else this.stop()}
 private stop(){if(this.frame!==undefined)cancelAnimationFrame(this.frame);this.frame=undefined}
 click(event:MouseEvent){this.ui?.click(event);this.update()}
 input(event:Event){this.ui?.input(event);this.update()}
 change(event:Event){this.ui?.change(event);this.update()}
 hover(event:MouseEvent|FocusEvent){this.ui?.hoverAt(event)}
 mouseout(event:MouseEvent){this.ui?.mouseout(event)}
 focusout(){this.ui?.focusout()}
 leader(){this.ui?.drawLeader()}
 down(event:PointerEvent){this.ui?.down(event)}
 move(event:PointerEvent){this.ui?.move(event)}
 up(){this.ui?.up()}
 visibility(){if(document.hidden)this.stop();else this.update()}
 hide(){this.stop();this.ui?.pause()}
 disconnect(){this.hide();this.ui?.dispose()}
}
registerController('battle-replay',BattleReplayController)
