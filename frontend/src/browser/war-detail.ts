import {FragmentSections} from './controllers/fragment-sections'
import {registerController} from './controllers/application'
import {eventElement} from './shared/dom'
import {isMemberSort,type MemberSort} from './shared/war-model'
type CombatSide='combined'|'aggressor'|'defender'
type WarTab=CombatSide|'members'|'intel'
type Part='stats'|'kills'|'members'|'intel'
const isSide=(value:unknown):value is CombatSide=>value==='combined'||value==='aggressor'||value==='defender'
const isTab=(value:unknown):value is WarTab=>isSide(value)||value==='members'||value==='intel'

export class WarDetailController extends FragmentSections<Part>{
 protected retryAttribute='data-war-retry'
 private tab:WarTab='combined'
 private combatSide:CombatSide='combined'
 private killPage=1
 private killLimit=50
 private killsDirty=false
 private memberSide:CombatSide='combined'
 private memberSort:MemberSort='activity'
 private memberEntity=''
 private statsCache=new Map<CombatSide,string>()
 private intelHTML:string|undefined
 private interrupted=false
 private get id(){return this.element.dataset.warId??''}
 private get sidebar(){return this.q<HTMLElement>('[data-war-sidebar]')}
 private get kills(){return this.q<HTMLElement>('[data-war-kills]')}
 private get memberResults(){return this.q<HTMLElement>('[data-war-member-results]')}
 private get intel(){return this.q<HTMLElement>('[data-war-intel]')}
 private side():CombatSide{return isSide(this.tab)?this.tab:'combined'}
 connect(){this.activate(location.hash.slice(1),false)}
 private async stats(){const side=this.side();this.cancelSection('stats');const cached=this.statsCache.get(side);if(cached){this.sidebar.innerHTML=cached;return}await this.section('stats',{container:this.sidebar,path:`/_fragments/war/${this.id}/stats?side=${side}`,selector:'.war-sidebar-content',message:'Could not load stats.',loading:'Loading stats...',loaded:content=>this.statsCache.set(side,content.outerHTML)})}
 private async loadKills(){this.killsDirty=true;const query=new URLSearchParams({side:this.side(),page:String(this.killPage),limit:String(this.killLimit)});await this.section('kills',{container:this.kills,path:`/_fragments/war/${this.id}/kills?${query}`,selector:'.kill-list',message:'Could not load kills.',loaded:()=>{this.killsDirty=false}})}
 private syncMembers(){
  for(const button of this.all<HTMLElement>('[data-member-side]')){const active=button.dataset.memberSide===this.memberSide;button.classList.toggle('active',active);button.setAttribute('aria-pressed',String(active))}
  for(const button of this.all<HTMLElement>('[data-member-sort]')){const active=button.dataset.memberSort===this.memberSort;button.classList.toggle('active',active);button.setAttribute('aria-pressed',String(active))}
  for(const button of this.all<HTMLElement>('[data-member-entity]')){const active=button.dataset.memberEntity===this.memberEntity;button.classList.toggle('active',active);button.setAttribute('aria-pressed',String(active));button.hidden=Boolean(button.dataset.memberEntity)&&this.memberSide!=='combined'&&button.dataset.memberTeam!==this.memberSide}
 }
 private async loadMembers(){const query=new URLSearchParams({side:this.memberSide,sort:this.memberSort}),entity=/^(alliance|corporation):([1-9]\d*)$/.exec(this.memberEntity);if(entity)query.set(entity[1]==='alliance'?'allianceId':'corporationId',entity[2]);await this.section('members',{container:this.memberResults,path:`/_fragments/war/${this.id}/members?${query}`,selector:'.war-members-table',message:'Could not load members.',loading:'Loading members...'})}
 private async loadIntel(){if(this.intelHTML){this.intel.innerHTML=this.intelHTML;return}if(this.sectionPending('intel'))return;await this.section('intel',{container:this.intel,path:`/_fragments/war/${this.id}/intel`,selector:'.war-intel-content',message:'Could not load intel.',loading:'Loading intel...',loaded:content=>{this.intelHTML=content.outerHTML}})}
 private activate(value:unknown,push=true){
  const tab=isTab(value)?value:'combined',previous=this.combatSide;this.tab=tab
  for(const button of this.all<HTMLElement>('[data-war-tab]')){const active=button.dataset.warTab===tab;button.classList.toggle('active',active);button.setAttribute('aria-pressed',String(active))}
  this.q<HTMLElement>('[data-war-overview]').hidden=tab==='members'||tab==='intel';this.q<HTMLElement>('[data-war-members]').hidden=tab!=='members';this.intel.hidden=tab!=='intel'
  if(push){const url=new URL(location.href);url.hash=tab==='combined'?'':tab;history.pushState(history.state,'',url)}
  if(tab!=='members')this.cancelSection('members');if(tab!=='intel')this.cancelSection('intel')
  if(tab==='members'||tab==='intel'){this.cancelSection('stats');this.cancelSection('kills');if(tab==='members'){this.syncMembers();void this.loadMembers()}else void this.loadIntel();return}
  this.combatSide=this.side();const changed=this.combatSide!==previous;if(changed){this.killPage=1;this.killLimit=50}if(changed||this.killsDirty)void this.loadKills();void this.stats()
 }
 click(event:MouseEvent){
  const target=eventElement(event.target),button=target?.closest<HTMLButtonElement>('button');if(!target||button?.disabled)return
  const tab=target.closest<HTMLElement>('[data-war-tab]');if(tab){if(tab.dataset.warTab!==this.tab)this.activate(tab.dataset.warTab);return}
  const side=target.closest<HTMLElement>('[data-member-side]');if(side&&isSide(side.dataset.memberSide)){this.memberSide=side.dataset.memberSide;const selected=this.element.querySelector<HTMLElement>(`[data-member-entity="${CSS.escape(this.memberEntity)}"]`);if(this.memberSide!=='combined'&&selected?.dataset.memberTeam&&selected.dataset.memberTeam!==this.memberSide)this.memberEntity='';this.syncMembers();void this.loadMembers();return}
  const sort=target.closest<HTMLElement>('[data-member-sort]');if(sort&&isMemberSort(sort.dataset.memberSort)){this.memberSort=sort.dataset.memberSort;this.syncMembers();void this.loadMembers();return}
  const entity=target.closest<HTMLElement>('[data-member-entity]');if(entity){const value=entity.dataset.memberEntity??'';if(value!==''&&!/^(alliance|corporation):[1-9]\d*$/.test(value))return;this.memberEntity=this.memberEntity===value?'':value;this.syncMembers();void this.loadMembers();return}
  const limit=target.closest<HTMLElement>('[data-kill-limit]');if(limit&&this.kills.contains(limit)){const value=Number(limit.dataset.killLimit);if(![10,25,50,100].includes(value))return;this.killLimit=value;this.killPage=1;void this.loadKills();return}
  const page=target.closest<HTMLElement>('[data-kill-page]');if(page&&this.kills.contains(page)){const value=Number(page.dataset.killPage);if(!Number.isSafeInteger(value)||value<1||value>10000)return;this.killPage=value;void this.loadKills();return}
  const retry=target.closest<HTMLElement>('[data-war-retry]');if(retry){const part=retry.dataset.warRetry;if(part==='stats')void this.stats();else if(part==='kills')void this.loadKills();else if(part==='members')void this.loadMembers();else if(part==='intel')void this.loadIntel()}
 }
 popstate(){this.activate(location.hash.slice(1),false)}
 hide(){this.interrupted||=(['stats','kills','members','intel'] as const).some(part=>this.sectionPending(part));super.hide()}
 show(event:PageTransitionEvent){if(event.persisted&&this.interrupted){this.interrupted=false;this.activate(this.tab,false);if(isSide(this.tab)&&!this.sectionPending('kills'))void this.loadKills()}}
}
registerController('war-detail',WarDetailController)
