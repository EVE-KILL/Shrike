import {Controller} from '@hotwired/stimulus';
import {registerController} from './controllers/application';
import {iconHref} from './icon-hrefs';
import {requiredElement,cloneComponent,eventElement} from './shared/dom';
import {readComposition,isCompositionView,isCompositionSortKey,type CompositionTeam,type CompositionView,type CompositionPilot,type CompositionAggregate,type CompositionRow,type CompositionFilters,type CompositionSort,type CompositionSortKey} from './shared/battle-model';
const numberValue=(value:unknown)=>typeof value==='number'&&Number.isFinite(value)?value:0;
const pilot=(row:CompositionRow):row is CompositionPilot=>'character_id'in row;
const textValue=(row:CompositionRow,key:CompositionSortKey)=>key==='name'?(!pilot(row)?row.name:''):key==='ship_name'?(pilot(row)?row.ship_name:''):key==='character_name'?(pilot(row)?row.character_name:''):'';
const numericValue=(row:CompositionRow,key:CompositionSortKey)=>key==='count'||key==='losses'?(!pilot(row)?row[key]:0):key==='isk_lost'||key==='damage_done'||key==='damage_taken'?numberValue(row[key]):0;
export function selectBattleCompositionRows(team:CompositionTeam|undefined,view:'individuals',filters:CompositionFilters,sort:CompositionSort):CompositionPilot[];
export function selectBattleCompositionRows(team:CompositionTeam|undefined,view:'by_ship'|'by_group',filters:CompositionFilters,sort:CompositionSort):CompositionAggregate[];
export function selectBattleCompositionRows(team:CompositionTeam|undefined,view:CompositionView,filters:CompositionFilters,sort:CompositionSort):CompositionRow[];
export function selectBattleCompositionRows(team:CompositionTeam|undefined,view:CompositionView,filters:CompositionFilters,sort:CompositionSort):CompositionRow[]{
 const matches=(name:string)=>!filters.search||name.toLowerCase().includes(filters.search.toLowerCase());
 const source:CompositionRow[]=view==='individuals'?team?.individuals??[]:view==='by_ship'?team?.by_ship??[]:team?.by_group??[];
 const rows=source.filter(row=>(!filters.group||row.ship_group_id===filters.group)&&(!filters.losses||(pilot(row)?row.deaths:row.losses)>0)&&(pilot(row)?matches(row.ship_name)||matches(row.character_name)||matches(row.corporation_name):matches(row.name)));
 if(sort.key==='default')return rows.sort((a,b)=>b.rank-a.rank||(pilot(a)&&pilot(b)?b.isk_lost-a.isk_lost||numberValue(b.damage_done)-numberValue(a.damage_done)||a.character_name.localeCompare(b.character_name):!pilot(a)&&!pilot(b)?b.count-a.count||b.isk_lost-a.isk_lost:0));
 return rows.sort((a,b)=>(sort.dir==='asc'?1:-1)*(['ship_name','character_name','name'].includes(sort.key)?textValue(a,sort.key).localeCompare(textValue(b,sort.key)):numericValue(a,sort.key)-numericValue(b,sort.key)));
}
function createComposition(section:HTMLElement) {
  const q=<T extends Element=HTMLElement>(selector:string)=>requiredElement<T>(section,selector);
  const field=<T extends Element=HTMLElement>(node:ParentNode,name:string)=>requiredElement<T>(node,`[data-battle-${name}]`);
  const get=<T extends Element=HTMLElement>(node:ParentNode,selector:string)=>requiredElement<T>(node,selector);
  const data=readComposition(JSON.parse(q('#battle-composition-data').textContent??'null'));
  const search=q<HTMLInputElement>('[data-battle-composition-search]'),group=q<HTMLSelectElement>('[data-battle-composition-group]'),loss=q<HTMLInputElement>('[data-battle-composition-loss]');
  const filters={search:'',group:Number(group.value)||null,losses:false},limits=data.teams.map(()=>500);
  const sorts:Record<CompositionView,CompositionSort>={individuals:{key:'default',dir:'desc'},by_ship:{key:'default',dir:'desc'},by_group:{key:'default',dir:'desc'}};
  let view:CompositionView='individuals';
  const number=(value:number|undefined)=>Number(value??0).toLocaleString('en-US');
  const compact=(input:number)=>{const value=numberValue(input);return value>=1e12?(value/1e12).toFixed(2)+'t':value>=1e9?(value/1e9).toFixed(2)+'b':value>=1e6?(value/1e6).toFixed(2)+'m':value>=1e3?(value/1e3).toFixed(1)+'k':Number(value).toFixed(0);};
  const entity=(kind:string,id:number)=>`https://images.evetech.net/${kind}/${id}/logo?size=32`;
  const text=(node:HTMLElement,name:string,value:string)=>{field(node,name).textContent=value;};
  const image=(node:HTMLElement,name:string,src:string)=>{const img=field<HTMLImageElement>(node,name);img.hidden=!src;if(src)img.src=src;else img.removeAttribute('src');};
  const link=(node:HTMLElement,name:string,path:string|null,value:string)=>{const el=field<HTMLAnchorElement>(node,name);el.textContent=value;if(path)el.href=path;else el.removeAttribute('href');};
  const lost=(node:HTMLElement,name:string,value:number,format:(value:number)=>string)=>{const cell=field(node,name);cell.textContent=value>0?format(value):'—';cell.classList.toggle('text-red-400',value>0);cell.classList.toggle(view==='individuals'?'text-gray-700':'text-gray-600',value<=0);};
  const rowNode=(row:CompositionRow)=>{
    const pilots=pilot(row);
    const node=cloneComponent(section,pilots?'[data-battle-individual-template]':'[data-battle-aggregate-template]');
    const id=view==='by_group'?row.ship_group_id:row.ship_type_id;
    image(node,'ship-image',view==='by_group'||!id?'':`/images/types/${id}/render?size=32`);
    link(node,'ship-link',id?`/${view==='by_group'?'group':'item'}/${id}`:null,pilot(row)?row.ship_name||'?':row.name);
    if(pilot(row)){
      const ship=field(node,'ship-link');ship.classList.toggle('text-red-300',row.deaths>0);ship.classList.toggle('text-gray-300',!row.deaths);
      link(node,'group-link',row.ship_group_id?`/group/${row.ship_group_id}`:null,row.ship_group_name||'?');
      image(node,'pilot-image',row.alliance_id?entity('alliances',row.alliance_id):row.corporation_id?entity('corporations',row.corporation_id):'');
      link(node,'pilot-link',row.character_id?`/character/${row.character_id}`:null,row.character_name||'Unknown');text(node,'pilot-corp',row.corporation_name||'');
    }else{text(node,'count',number(row.count));lost(node,'losses',row.losses,number);}
    lost(node,'lost',row.isk_lost,compact);text(node,'damage',number(numberValue(row.damage_done)));text(node,'taken',number(numberValue(row.damage_taken)));return node;
  };
  const render=()=>{
    const pilots=view==='individuals';
    for(const button of section.querySelectorAll<HTMLElement>('[data-battle-composition-view]')){const active=button.dataset.battleCompositionView===view;button.setAttribute('aria-pressed',String(active));for(const cls of ['bg-blue-500/10','text-blue-400'])button.classList.toggle(cls,active);button.classList.toggle('text-gray-500',!active);}
    for(const button of section.querySelectorAll<HTMLElement>('[data-battle-composition-sort]')){
      const sort=sorts[button.dataset.battleSortView==='individuals'?'individuals':view==='by_group'?'by_group':'by_ship'];
      const use=get<SVGUseElement>(button,'use');use.setAttribute('href',iconHref(sort.key===button.dataset.battleCompositionSort?sort.dir==='asc'?'chevron-up':'chevron-down':'chevrons-up-down'));
      if(button.dataset.battleSortView==='aggregate'&&button.dataset.battleCompositionSort==='name')get<HTMLElement>(button,'span').textContent=view==='by_group'?'Ship Group':'Ship';
    }
    for(const panel of section.querySelectorAll<HTMLElement>('[data-battle-composition-team]')){
      const i=Number(panel.dataset.battleCompositionTeam),rows=selectBattleCompositionRows(data.teams[i],view,filters,sorts[view]);
      get<HTMLElement>(panel,'[data-battle-composition-count]').textContent=`${number(rows.length)} ${pilots?'pilots':view==='by_group'?'ship groups':'ship types'}`;
      get<HTMLElement>(panel,'[data-battle-individual-table]').hidden=!pilots;get<HTMLElement>(panel,'[data-battle-aggregate-table]').hidden=pilots;
      const body=get(panel,pilots?'[data-battle-composition-individuals]':'[data-battle-composition-aggregates]');body.replaceChildren(...(pilots?rows.slice(0,limits[i]):rows).map(rowNode));
      const empty=get<HTMLElement>(panel,'[data-battle-composition-empty]');empty.hidden=rows.length>0;empty.textContent=pilots?'No pilots':view==='by_ship'?'No ships':'No ship groups';
      const more=get<HTMLElement>(panel,'[data-battle-composition-more]');more.hidden=!pilots||rows.length<=limits[i];more.textContent=`Show more (${number(rows.length-limits[i])} remaining)`;
    }
  };
  function input(event:Event){if(event.target===search){filters.search=search.value;limits.fill(500);render();}}
  function change(event:Event){if(event.target===group)filters.group=Number(group.value)||null;else if(event.target===loss)filters.losses=loss.checked;else return;limits.fill(500);render();}
  function click(event:MouseEvent){
    const button=eventElement(event.target)?.closest<HTMLButtonElement>('button');if(!button)return;
    if(button.hasAttribute('data-battle-composition-view')){const value=button.dataset.battleCompositionView;if(!isCompositionView(value))return;view=value;}
    else if(button.hasAttribute('data-battle-composition-sort')){const target=button.dataset.battleSortView==='individuals'?'individuals':view;const current=sorts[target],key=button.dataset.battleCompositionSort;if(!isCompositionSortKey(key))return;sorts[target]={key,dir:current.key===key&&current.dir==='desc'?'asc':'desc'};}
    else if(button.hasAttribute('data-battle-composition-more')){const i=Number(button.closest<HTMLElement>('[data-battle-composition-team]')!.dataset.battleCompositionTeam);limits[i]+=500;}
    else if(button.hasAttribute('data-battle-composition-reset')){search.value='';group.value='';loss.checked=false;Object.assign(filters,{search:'',group:null,losses:false});limits.fill(500);}
    else return;render();
  }
  // The server already rendered the initial rows. Only initialize visibility
  // controls; leave those image nodes intact until the user changes a view.
  for(const panel of section.querySelectorAll<HTMLElement>('[data-battle-composition-team]')){const i=Number(panel.dataset.battleCompositionTeam),rows=selectBattleCompositionRows(data.teams[i],view,filters,sorts[view]);const empty=get<HTMLElement>(panel,'[data-battle-composition-empty]');empty.hidden=rows.length>0;const more=get<HTMLElement>(panel,'[data-battle-composition-more]');more.hidden=rows.length<=500;more.textContent=`Show more (${number(rows.length-500)} remaining)`;}
  return {click,input,change};
}

export class BattleCompositionController extends Controller<HTMLElement>{
 private ui:ReturnType<typeof createComposition>|undefined
 connect(){this.ui??=createComposition(this.element)}
 click(event:MouseEvent){this.ui?.click(event)}
 input(event:Event){this.ui?.input(event)}
 change(event:Event){this.ui?.change(event)}
}
registerController('battle-composition',BattleCompositionController)
