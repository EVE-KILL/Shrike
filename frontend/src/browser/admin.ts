import {FragmentTabsController} from './controllers/fragment-tabs'
import {registerController} from './controllers/application'
import type {AdminSummary} from './shared/admin-summary'
import {isRecord} from './shared/json'
import type {AdminOverviewResponses} from '../../shared/api/types.gen'
import './admin-announcements'
import './admin-domains'
import './admin-campaigns'
import './admin-river'
import './admin-wallet'
import './admin-blog'
import './admin-review'
import './controllers/admin/esi-summary'
import './controllers/admin/users'
import './controllers/admin/user-detail'

export class AdminController extends FragmentTabsController {
  protected readonly kind='admin'
  private summaries:Promise<AdminSummary>|undefined
  loadSummary(retry=false):Promise<AdminSummary>{
    if(retry)this.summaries=undefined
    if(!this.summaries){
      const request=this.request<AdminSummary>('/api/admin/esi')
      let cached:Promise<AdminSummary>
      cached=request.catch(error=>{if(this.summaries===cached)this.summaries=undefined;throw error})
      this.summaries=cached
    }
    return this.summaries
  }
  protected fragmentQuery(id:string){return id==='esi'?location.search:''}
  protected preparePanel(panel:HTMLElement){
    const overview=panel.querySelector<HTMLElement>('[data-admin-overview]')
    if(overview){
      let raw:unknown;try{raw=JSON.parse(overview.dataset.overview??'null')}catch{raw=null}
      if(isRecord(raw)&&isRecord(raw.attention)){
        const attention:Partial<AdminOverviewResponses[200]['attention']>={}
        for(const key of ['moderation','domainAssets','campaignsPaused','campaignsFailed'] as const){const value=raw.attention[key];if(typeof value==='number'&&Number.isFinite(value))attention[key]=value}
        const counts:Record<string,number>={moderation:attention.moderation??0,domains:attention.domainAssets??0,campaigns:(attention.campaignsPaused??0)+(attention.campaignsFailed??0)}
        for(const badge of this.all<HTMLElement>('[data-admin-badge]')){const count=counts[badge.dataset.adminBadge??'']??0;badge.hidden=!count;badge.textContent=String(count)}
      }
    }
  }
  hide(){this.summaries=undefined;super.hide()}
}
registerController('admin',AdminController)
