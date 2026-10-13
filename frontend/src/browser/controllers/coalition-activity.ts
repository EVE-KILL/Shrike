import {Controller} from '@hotwired/stimulus'
import {registerController} from './application'
import {loadCoalitionActivity} from '../coalition-activity'

export class CoalitionActivityController extends Controller<HTMLElement> {
  private request:AbortController|undefined
  private loaded=false
  connect(){if(!this.loaded)void this.load()}
  disconnect(){
    this.request?.abort()
    this.element.querySelector('[data-coalition-activity-status]')?.setAttribute('aria-busy','false')
  }
  async load(){
    if(this.request&&!this.request.signal.aborted)return
    const request=new AbortController();this.request=request
    try{
      await loadCoalitionActivity(this.element,fetch,request.signal)
      if(this.request===request&&!request.signal.aborted)this.loaded=Boolean(this.element.querySelector<HTMLElement>('[data-coalition-activity-status]')?.hidden)
    }finally{if(this.request===request)this.request=undefined}
  }
}
registerController('coalition-activity',CoalitionActivityController)
