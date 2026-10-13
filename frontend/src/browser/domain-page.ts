import {FragmentSections} from './controllers/fragment-sections'
import {registerController} from './controllers/application'
import {initializePrivateKillFeed} from './private-killfeed'
import {isRecord} from './shared/json'
import {eventElement} from './shared/dom'

export class DomainPageController extends FragmentSections<string>{
 protected retryAttribute='data-domain-retry'
 private feeds:Array<ReturnType<typeof initializePrivateKillFeed>>=[]
 private active=false
 connect(){
  this.active=true;let entities:unknown=[];try{entities=JSON.parse(this.element.dataset.domainEntities||'[]')}catch{/* Invalid scope must not break widgets. */}
  const topics=Array.isArray(entities)?[...new Set(entities.flatMap(entity=>isRecord(entity)&&['character','corporation','alliance'].includes(String(entity.type))&&typeof entity.id==='number'&&Number.isSafeInteger(entity.id)&&entity.id>0?[`victim.${entity.id}`,`attacker.${entity.id}`]:[]))]:[]
  for(const host of this.element.querySelectorAll<HTMLElement>('[data-private-killfeed]'))this.feeds.push(initializePrivateKillFeed(host,{endpoint:'/_fragments/domain/killlist',topics,params:{type:host.dataset.domainKilltype||'latest'}}))
  this.syncTab();this.show()
 }
 disconnect(){this.active=false;this.hide();for(const feed of this.feeds)feed.destroy();this.feeds=[];super.disconnect()}
 private load(host:HTMLElement,kind?:string){
  const widget=host.dataset.domainWidget,key=host.dataset.domainKey;if(!widget||!key)return
  const selected=kind||host.dataset.domainKind||'kills';if(widget==='valuable'&&!['kills','ships','structures'].includes(selected))return
  const query=new URLSearchParams();if(widget==='valuable')query.set('kind',selected);if(host.dataset.domainKilltype)query.set('type',host.dataset.domainKilltype)
  const path=`/_fragments/domain/widgets/${encodeURIComponent(widget)}?${query}`
  host.dataset.domainLoaded='false';host.dataset.domainKind=selected
  void this.section(key,{container:host,path,selector:widget==='valuable'?(host.dataset.domainKilltype?'.kills-valuable':'.valuable'):widget==='campaigns'?'section':'.top-box',message:'Could not load this widget.',loaded:()=>{host.dataset.domainLoaded='true'}})
 }
 click(event:MouseEvent){
  const target=eventElement(event.target);if(!target)return
  const tab=target.closest<HTMLElement>('[data-kills-tab]')?.dataset.killsTab
  if(tab&&['kills','top','valuable'].includes(tab)){history.pushState(history.state,'',tab==='kills'?location.pathname:'#'+tab);this.syncTab();return}
  const control=target.closest<HTMLElement>('[data-domain-retry],[data-valuable-tab]');if(!control)return
  const host=control.closest<HTMLElement>('[data-domain-widget]');if(host)this.load(host,control.dataset.valuableTab)
 }
 syncTab(){
  if(!this.element.classList.contains('kills-page'))return
  const selected=location.hash.slice(1),tab=['kills','top','valuable'].includes(selected)?selected:'kills';this.element.dataset.tab=tab
  for(const button of this.element.querySelectorAll<HTMLElement>('[data-kills-tab]'))button.setAttribute('aria-selected',String(button.dataset.killsTab===tab))
  const label=tab==='top'?'Top Lists':tab==='valuable'?'Valuable':''
  document.title=`${this.element.dataset.title}${label?' ('+label+')':''} - ${this.element.dataset.domainName}`
 }
 show(){if(this.active)for(const host of this.element.querySelectorAll<HTMLElement>('[data-domain-widget]'))if(host.dataset.domainLoaded!=='true'&&!this.sectionPending(host.dataset.domainKey||''))this.load(host)}
}
registerController('domain-page',DomainPageController)
