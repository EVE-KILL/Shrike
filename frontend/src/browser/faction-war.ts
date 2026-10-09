import './faction-war-map'
import {FragmentSections} from './controllers/fragment-sections'
import {registerController} from './controllers/application'
import {eventElement,requiredElement,placeElement} from './shared/dom'
import {readFwSystemsScript,isMemberSort,type MemberSort,type FwMapData} from './shared/war-model'
type Side='combined'|'side1'|'side2'
type Tab=Side|'members'|'intel'|'map'|'systems'
type Part='sidebar'|'kills'|'members'|'intel'|'systems'
type Period='yesterday'|'last_week'|'active_total'
type MemberSide='combined'|'aggressor'|'defender'
const tableSorts=['name','region','status','vp','kills','isk'] as const
type TableSort=typeof tableSorts[number]
const isSide=(value:unknown):value is Side=>value==='combined'||value==='side1'||value==='side2'
const isTab=(value:unknown):value is Tab=>isSide(value)||value==='members'||value==='intel'||value==='map'||value==='systems'
const isPeriod=(value:unknown):value is Period=>value==='yesterday'||value==='last_week'||value==='active_total'
const isMemberSide=(value:unknown):value is MemberSide=>value==='combined'||value==='aggressor'||value==='defender'
const isTableSort=(value:unknown):value is TableSort=>tableSorts.some(key=>key===value)

export class FactionWarController extends FragmentSections<Part>{
 protected retryAttribute='data-fw-retry'
 private tab:Tab='combined'
 private combatSide:Side='combined'
 private period:Period='last_week'
 private page=1
 private limit=50
 private killsDirty=false
 private memberSide:MemberSide='combined'
 private memberSort:MemberSort='activity'
 private cursors=new Map<number,string>([[1,'']])
 private sidebarCache=new Map<string,string>()
 private intelLoaded=false
 private systemsLoaded=false
 private systemsData:FwMapData|undefined
 private tableSort:TableSort='kills'
 private tableDirection:'asc'|'desc'='desc'
 private initialized=false
 private interrupted=false
 private get slug(){return this.element.dataset.matchup??''}
 private get sidebar(){return this.q<HTMLElement>('[data-fw-sidebar]')}
 private get kills(){return this.q<HTMLElement>('[data-fw-kills]')}
 private get memberResults(){return this.q<HTMLElement>('[data-fw-member-results]')}
 private get intel(){return this.q<HTMLElement>('[data-fw-intel]')}
 private get systems(){return this.q<HTMLElement>('[data-fw-systems]')}
 private side():Side{return isSide(this.tab)?this.tab:'combined'}
 private path(part:Part,query?:URLSearchParams){return `/_fragments/faction-war/${encodeURIComponent(this.slug)}/${part}${query?.size?'?'+query:''}`}
 connect(){if(!this.initialized){this.sidebarCache.set('combined:last_week',this.sidebar.innerHTML);const cursor=this.kills.querySelector<HTMLElement>('.kill-list')?.dataset.nextCursor;if(cursor)this.cursors.set(2,cursor);this.initialized=true}this.activate(location.hash.slice(1),false)}
 private async loadSidebar(){const side=this.side(),period=this.period,key=`${side}:${period}`;this.cancelSection('sidebar');const cached=this.sidebarCache.get(key);if(cached!==undefined){this.sidebar.innerHTML=cached;return}await this.section('sidebar',{container:this.sidebar,path:this.path('sidebar',new URLSearchParams({side,period})),selector:'[data-fw-sidebar-loaded]',message:'Could not load leaderboards.',loading:'Loading…',loaded:()=>this.sidebarCache.set(key,this.sidebar.innerHTML)})}
 private async loadKills(){const page=this.page,query=new URLSearchParams({side:this.side(),page:String(page),limit:String(this.limit)}),cursor=this.cursors.get(page);if(cursor)query.set('after',cursor);this.killsDirty=true;await this.section('kills',{container:this.kills,path:this.path('kills',query),selector:'.kill-list',message:'Could not load kills.',loaded:content=>{const next=content.dataset.nextCursor;if(next)this.cursors.set(page+1,next);else this.cursors.delete(page+1);this.killsDirty=false}})}
 private async loadMembers(){await this.section('members',{container:this.memberResults,path:this.path('members',new URLSearchParams({side:this.memberSide,sort:this.memberSort})),selector:'.war-members-table',message:'Could not load members.',loading:'Loading…'})}
 private async loadIntel(){if(this.intelLoaded||this.sectionPending('intel'))return;await this.section('intel',{container:this.intel,path:this.path('intel'),selector:'.war-intel-content',message:'Could not load intel.',loading:'Loading…',loaded:()=>{this.intelLoaded=true}})}
 private showSystems(){const map=this.systems.querySelector<HTMLElement>('[data-fw-map]'),table=this.systems.querySelector<HTMLElement>('[data-fw-system-table]');if(map){map.hidden=this.tab!=='map';if(map.hidden)map.dispatchEvent(new Event('fw-map-hide'))}if(table)table.hidden=this.tab==='map'}
 private async loadSystems(){if(this.systemsLoaded){this.showSystems();return}if(this.sectionPending('systems'))return;await this.section('systems',{container:this.systems,path:this.path('systems'),selector:'[data-fw-systems-loaded]',message:'Could not load systems.',loading:'Loading…',beforeCommit:content=>{readFwSystemsScript(requiredElement<HTMLElement>(content,'#faction-war-systems-data'))},loaded:content=>{this.systemsData=readFwSystemsScript(requiredElement<HTMLElement>(content,'#faction-war-systems-data'));this.systemsLoaded=true;this.showSystems();this.syncTable()}})}
 private sync(){
  for(const button of this.all<HTMLElement>('[data-fw-tab]'))button.setAttribute('aria-pressed',String(button.dataset.fwTab===this.tab))
  for(const button of this.all<HTMLElement>('[data-fw-period]'))button.setAttribute('aria-pressed',String(button.dataset.fwPeriod===this.period))
  for(const button of this.all<HTMLElement>('[data-member-side]')){const active=button.dataset.memberSide===this.memberSide;button.classList.toggle('active',active);button.setAttribute('aria-pressed',String(active))}
  for(const button of this.all<HTMLElement>('[data-member-sort]')){const active=button.dataset.memberSort===this.memberSort;button.classList.toggle('active',active);button.setAttribute('aria-pressed',String(active))}
  this.q<HTMLElement>('[data-fw-overview]').hidden=!isSide(this.tab);this.q<HTMLElement>('[data-fw-members]').hidden=this.tab!=='members';this.intel.hidden=this.tab!=='intel';this.systems.hidden=this.tab!=='map'&&this.tab!=='systems';this.showSystems()
 }
 private activate(value:unknown,push=true){
  const tab=isTab(value)?value:'combined',previous=this.combatSide;this.tab=tab;this.sync()
  if(push){const url=new URL(location.href);url.hash=tab==='combined'?'':tab;history.pushState(history.state,'',url)}
  if(tab!=='members')this.cancelSection('members');if(tab!=='intel')this.cancelSection('intel');if(tab!=='map'&&tab!=='systems')this.cancelSection('systems')
  if(!isSide(tab)){this.cancelSection('sidebar');this.cancelSection('kills');if(tab==='members')void this.loadMembers();else if(tab==='intel')void this.loadIntel();else void this.loadSystems();return}
  this.combatSide=this.side();const changed=this.combatSide!==previous;if(changed){this.page=1;this.limit=50;this.cursors=new Map([[1,'']])}void this.loadSidebar();if(changed||this.killsDirty)void this.loadKills()
 }
 private syncTable(){
  if(!this.systemsData)return;const key=this.tableSort,dir=this.tableDirection==='asc'?1:-1,rank:Record<string,number>={vulnerable:3,contested:2,captured:1,uncontested:0}
  const rows=[...this.systemsData.systems].sort((a,b)=>{let c=0;switch(key){case 'name':c=a.system_name.localeCompare(b.system_name);break;case 'region':c=a.region_name.localeCompare(b.region_name);break;case 'status':c=(rank[a.contested]??0)-(rank[b.contested]??0);break;case 'vp':c=(a.victory_points_threshold>0?a.victory_points/a.victory_points_threshold:0)-(b.victory_points_threshold>0?b.victory_points/b.victory_points_threshold:0);break;case 'kills':c=a.kills_24h-b.kills_24h;break;case 'isk':c=a.isk_24h-b.isk_24h;break}return dir*c})
  const body=requiredElement<HTMLTableSectionElement>(this.systems,'tbody'),elements=new Map([...body.children].flatMap(row=>row instanceof HTMLElement?[[Number(row.dataset.fwSystemId),row] as const]:[]));let position=0;for(const row of rows){const element=elements.get(row.solar_system_id);if(element){if(body.children[position]!==element)placeElement(body,element,body.children[position]??null);position++}}
  for(const button of this.systems.querySelectorAll<HTMLElement>('[data-fw-system-sort]')){const active=button.dataset.fwSystemSort===key;requiredElement(button,'[data-fw-sort-icon]').textContent=active?(this.tableDirection==='desc'?'⌄':'⌃'):'⌃⌄';button.closest('th')?.setAttribute('aria-sort',active?(this.tableDirection==='desc'?'descending':'ascending'):'none')}
 }
 click(event:MouseEvent){
  const target=eventElement(event.target),button=target?.closest<HTMLButtonElement>('button');if(!target||button?.disabled)return
  const tab=target.closest<HTMLElement>('[data-fw-tab]');if(tab){if(tab.dataset.fwTab!==this.tab)this.activate(tab.dataset.fwTab);return}
  const period=target.closest<HTMLElement>('[data-fw-period]');if(period&&isPeriod(period.dataset.fwPeriod)){this.period=period.dataset.fwPeriod;this.sync();void this.loadSidebar();return}
  const side=target.closest<HTMLElement>('[data-member-side]');if(side&&isMemberSide(side.dataset.memberSide)){this.memberSide=side.dataset.memberSide;this.sync();void this.loadMembers();return}
  const sort=target.closest<HTMLElement>('[data-member-sort]');if(sort&&isMemberSort(sort.dataset.memberSort)){this.memberSort=sort.dataset.memberSort;this.sync();void this.loadMembers();return}
  const tableSort=target.closest<HTMLElement>('[data-fw-system-sort]');if(tableSort&&isTableSort(tableSort.dataset.fwSystemSort)){const key=tableSort.dataset.fwSystemSort;this.tableDirection=this.tableSort===key&&this.tableDirection==='desc'?'asc':'desc';this.tableSort=key;this.syncTable();return}
  const limit=target.closest<HTMLElement>('[data-kill-limit]');if(limit&&this.kills.contains(limit)){const value=Number(limit.dataset.killLimit);if(![10,25,50,100].includes(value))return;this.limit=value;this.page=1;this.cursors=new Map([[1,'']]);void this.loadKills();return}
  const page=target.closest<HTMLElement>('[data-kill-page]');if(page&&this.kills.contains(page)){const value=Number(page.dataset.killPage);if(!Number.isSafeInteger(value)||value<1||value>10000||(value>1&&!this.cursors.has(value)))return;this.page=value;void this.loadKills();return}
  const retry=target.closest<HTMLElement>('[data-fw-retry]');if(retry){const part=retry.dataset.fwRetry;if(part==='sidebar')void this.loadSidebar();else if(part==='kills')void this.loadKills();else if(part==='members')void this.loadMembers();else if(part==='intel')void this.loadIntel();else if(part==='systems')void this.loadSystems()}
 }
 popstate(){this.activate(location.hash.slice(1),false)}
 hide(){this.interrupted||=(['sidebar','kills','members','intel','systems'] as const).some(part=>this.sectionPending(part));this.systems.querySelector('[data-fw-map]')?.dispatchEvent(new Event('fw-map-hide'));super.hide()}
 show(event:PageTransitionEvent){if(event.persisted&&this.interrupted){this.interrupted=false;this.activate(this.tab,false)}}
}
registerController('faction-war',FactionWarController)
