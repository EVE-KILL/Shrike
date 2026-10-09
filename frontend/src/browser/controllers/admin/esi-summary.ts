import {AdminComponent} from './base'
import {registerController} from '../application'
import type {AdminController} from '../../admin'
import {renderEsiVolume} from '../../esi'
import {fmt} from '../../esi-data'
import {eventElement} from '../../shared/dom'
import {errorMessage,isAbortError} from '../../shared/json'
import {escapeHTML} from '../../shared/html'

export class AdminEsiSummaryController extends AdminComponent {
  static outlets=['admin']
  private parent:AdminController|undefined
  private epoch=0
  private connected=false
  connect(){this.connected=true;if(this.parent)void this.load()}
  adminOutletConnected(parent:AdminController){this.parent=parent;if(this.connected)void this.load()}
  adminOutletDisconnected(parent:AdminController){if(this.parent===parent)this.parent=undefined;this.hide()}
  async click(event:MouseEvent){if(eventElement(event.target)?.closest('[data-admin-esi-retry]'))await this.load(true)}
  private async load(retry=false){
    const epoch=++this.epoch;this.error('[data-admin-esi-error]')
    try{
      const parent=this.parent;if(!parent)return
      const data=await parent.loadSummary(retry);if(epoch!==this.epoch||!this.element.isConnected)return
      const chart=this.q<HTMLElement>('[data-esi-volume]');chart.dataset.hours=JSON.stringify(data.volumeByHour);renderEsiVolume(chart)
      const count=this.element.querySelector<HTMLElement>('[data-admin-esi-count]');if(count)count.textContent=fmt(data.rateLimit.request_count)
      for(const [key,value]of [['average',data.responseTime.avg_ms],['p95',data.responseTime.p95_ms]] as const){const node=this.element.querySelector<HTMLElement>(`[data-admin-esi-${key}]`);if(node)node.textContent=value===null?'—':String(value)}
      const tone=this.element.querySelector<HTMLElement>('[data-admin-esi-p95-tone]');if(tone)tone.className='text-2xl font-bold tabular-nums '+((data.responseTime.p95_ms??0)>3000?'text-yellow-400':'text-white')
    }catch(error){if(epoch===this.epoch&&this.element.isConnected&&!isAbortError(error)){const box=this.q<HTMLElement>('[data-admin-esi-error]');box.innerHTML=`${escapeHTML(errorMessage(error))} <button type="button" class="text-blue-400" data-admin-esi-retry>Retry</button>`;box.hidden=false}}
  }
  hide(){++this.epoch;super.hide()}
  show(event:PageTransitionEvent){if(event.persisted)void this.load()}
  disconnect(){this.connected=false;this.parent=undefined;super.disconnect()}
}
registerController('admin-esi-summary',AdminEsiSummaryController)
