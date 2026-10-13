import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {readBrowserSession} from './browser-request'
import {eventElement} from './shared/dom'

export class CampaignsController extends RequestComponent {
  private pending:AbortController|undefined
  private searchTimer:ReturnType<typeof setTimeout>|undefined
  private generation=0
  private interrupted=false
  private get search(){return this.q<HTMLInputElement>('[data-campaign-search]')}
  private get results(){return this.q<HTMLElement>('[data-campaign-results]')}
  connect(){const generation=++this.generation;void readBrowserSession().then(user=>{if(generation===this.generation&&this.element.isConnected)this.q<HTMLElement>('[data-campaign-private]').hidden=!user?.characterId}).catch(()=>{});this.sync();if(this.interrupted)void this.load()}
  private params(){return new URLSearchParams(location.search)}
  private sync(){const query=this.params(),status=query.get('status')||'active',mode=query.get('mode')||'all';for(const button of this.all<HTMLButtonElement>('[data-campaign-status]')){const active=button.dataset.campaignStatus===status;button.classList.toggle('active',active);button.setAttribute('aria-pressed',String(active))}for(const button of this.all<HTMLButtonElement>('[data-campaign-mode]')){const active=button.dataset.campaignMode===mode;button.classList.toggle('active',active);button.setAttribute('aria-pressed',String(active))}this.q<HTMLElement>('[data-campaign-clear-search]').hidden=!this.search.value}
  private async load(){
    this.pending?.abort();const controller=new AbortController();this.pending=controller;this.interrupted=false;this.error('[data-campaign-error]');const current=this.results;current.classList.add('loading');current.setAttribute('aria-busy','true')
    try{const text=await this.text(`/_fragments/campaigns?${this.params()}`,'Unable to load campaigns',{signal:controller.signal}),next=new DOMParser().parseFromString(text,'text/html').querySelector<HTMLElement>('[data-campaign-results]');if(!next)throw new Error('Campaign results are unavailable');if(this.pending!==controller)return;current.replaceWith(document.importNode(next,true))}
    catch(error){if(this.pending===controller)this.error('[data-campaign-error]',error)}finally{if(this.pending===controller){this.pending=undefined;this.results.classList.remove('loading');current.removeAttribute('aria-busy')}}
  }
  private change(patch:Record<string,string|null|undefined>){this.stopTimer(this.searchTimer);this.searchTimer=undefined;const query=this.params();for(const [key,value]of Object.entries(patch))if(value===null||value===undefined||value==='')query.delete(key);else query.set(key,value);query.delete('page');history.pushState(null,'',`${location.pathname}${query.size?'?'+query:''}`);this.search.value=query.get('q')??'';this.sync();void this.load()}
  async click(event:MouseEvent){
    const target=eventElement(event.target),status=target?.closest<HTMLElement>('[data-campaign-status]'),mode=target?.closest<HTMLElement>('[data-campaign-mode]')
    if(status){this.change({status:status.dataset.campaignStatus==='active'?null:status.dataset.campaignStatus});return}if(mode){this.change({mode:mode.dataset.campaignMode==='all'?null:mode.dataset.campaignMode});return}
    const page=target?.closest<HTMLButtonElement>('[data-campaign-page]');if(page&&!page.disabled){const next=Number(page.dataset.campaignPage);if(!Number.isSafeInteger(next)||next<=0)return;this.stopTimer(this.searchTimer);const query=this.params();if(next>1)query.set('page',String(next));else query.delete('page');history.pushState(null,'',`${location.pathname}${query.size?'?'+query:''}`);await this.load();this.element.scrollIntoView({behavior:'smooth',block:'start'});return}
    if(target?.closest('[data-campaign-clear-filters]')){this.change({mode:null,q:null});return}
    if(target?.closest('[data-campaign-clear-search]')){this.change({q:null});this.search.focus()}
  }
  searchChanged(event:Event){if(event.target!==this.search)return;this.q<HTMLElement>('[data-campaign-clear-search]').hidden=!this.search.value;this.stopTimer(this.searchTimer);this.pending?.abort();this.interrupted=true;this.searchTimer=this.after(300,()=>{this.searchTimer=undefined;this.change({q:this.search.value.trim()||null})})}
  popstate(){this.stopTimer(this.searchTimer);this.search.value=this.params().get('q')??'';this.sync();void this.load()}
  show(event:PageTransitionEvent){if(event.persisted&&this.interrupted){this.change({q:this.search.value.trim()||null})}}
  hide(){this.interrupted||=Boolean(this.pending)||Boolean(this.searchTimer);this.stopTimer(this.searchTimer);this.searchTimer=undefined;this.pending?.abort();this.pending=undefined;this.results.classList.remove('loading');this.results.removeAttribute('aria-busy');super.hide()}
  disconnect(){++this.generation;super.disconnect()}
}
registerController('campaigns',CampaignsController)
