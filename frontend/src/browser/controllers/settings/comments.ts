import {SettingsComponent} from './base'
import {registerController} from '../application'
import {eventElement,requiredElement} from '../../shared/dom'
import {errorMessage,isAbortError} from '../../shared/json'

export class SettingsCommentsController extends SettingsComponent{
  private cursor:string|null=null
  private loaded=false
  private loading=false
  private confirmId:string|null=null
  private confirmTimer:ReturnType<typeof setTimeout>|undefined
  private epoch=0
  private get list(){return this.q<HTMLElement>('[data-settings-comments-list]')}
  connect(){this.cursor=this.element.dataset.cursor||null;this.loaded=this.element.dataset.loaded==='true';this.loading=false;this.confirmId=null;this.render()}
  hide(){++this.epoch;this.loading=false;super.hide();this.render()}
  private showError(error:unknown=''){
    if(!this.element.isConnected||isAbortError(error))return
    const message=typeof error==='string'?error:errorMessage(error),box=this.q<HTMLElement>('[data-settings-comments-error]')
    box.hidden=!message;this.q<HTMLElement>('[data-settings-comments-error-text]').textContent=message
  }
  private render(){
    if(!this.element.isConnected)return
    const count=this.list.children.length,refresh=this.q<HTMLButtonElement>('[data-settings-comments-refresh]'),more=this.q<HTMLButtonElement>('[data-settings-comments-more]')
    this.q<HTMLElement>('[data-settings-comments-empty]').hidden=!this.loaded||count>0
    this.q<HTMLElement>('[data-settings-comments-more-wrap]').hidden=!count||!this.cursor
    this.q<HTMLElement>('[data-settings-comments-end]').hidden=!count||Boolean(this.cursor)
    refresh.disabled=this.loading;more.disabled=this.loading;requiredElement<SVGElement>(refresh,'svg').classList.toggle('animate-spin',this.loading)
    requiredElement<SVGUseElement>(more,'use').setAttribute('href','/assets/icons.svg#'+(this.loading?'loader-2':'chevron-down'));requiredElement<SVGElement>(more,'svg').classList.toggle('animate-spin',this.loading)
    this.q<HTMLElement>('[data-settings-comments-more-label]').textContent=this.loading?'Loading…':'Load more'
    for(const button of this.list.querySelectorAll<HTMLButtonElement>('[data-settings-comment-delete]')){
      const armed=button.dataset.settingsCommentDelete===this.confirmId
      button.title=armed?'Click again to delete permanently':'Delete comment'
      button.className='flex items-center gap-1 px-2 py-1 rounded text-xs text-gray-500 hover:text-red-400 hover:bg-red-500/[0.08] cursor-pointer transition-colors disabled:opacity-50 disabled:cursor-not-allowed'+(armed?' text-red-300 bg-red-500/[0.12] ring-1 ring-red-500/40':'')
      requiredElement<HTMLElement>(button,'[data-settings-comment-delete-label]').textContent=armed?'Confirm':'Delete'
    }
  }
  private async load(reset:boolean){
    if(this.loading)return
    this.loading=true;const epoch=this.epoch;this.showError();this.render()
    try{
      const response=await this.readTextWithHeaders('/_fragments/settings-comments'+(!reset&&this.cursor?'?cursor='+encodeURIComponent(this.cursor):''),'Could not load your comments')
      if(epoch!==this.epoch)return
      const doc=new DOMParser().parseFromString(response.text,'text/html')
      const existing=new Set(reset?[]:[...this.list.children].map(node=>node instanceof HTMLElement?node.dataset.settingsComment:undefined))
      if(reset){this.list.replaceChildren();this.confirmId=null;this.stopTimer(this.confirmTimer)}
      for(const card of doc.querySelectorAll<HTMLElement>('[data-settings-comment]'))if(card.dataset.settingsComment&&!existing.has(card.dataset.settingsComment)){this.list.append(document.importNode(card,true));existing.add(card.dataset.settingsComment)}
      this.cursor=response.headers.get('X-Next-Cursor')||null;this.loaded=true;this.element.dataset.cursor=this.cursor??'';this.element.dataset.loaded='true'
    }catch(error){if(epoch===this.epoch)this.showError(error)}finally{if(epoch===this.epoch){this.loading=false;this.render()}}
  }
  async click(event:MouseEvent){
    const target=eventElement(event.target)
    if(target?.closest('[data-settings-comments-refresh]')){await this.load(true);return}
    if(target?.closest('[data-settings-comments-more]')){await this.load(false);return}
    const button=target?.closest<HTMLButtonElement>('[data-settings-comment-delete]'),id=button?.dataset.settingsCommentDelete
    if(!button||button.disabled||!id||!/^\d+$/.test(id))return
    if(this.confirmId!==id){this.confirmId=id;this.stopTimer(this.confirmTimer);this.confirmTimer=this.after(5000,()=>{if(this.confirmId===id){this.confirmId=null;this.render()}});this.render();return}
    this.confirmId=null;this.stopTimer(this.confirmTimer);this.showError();button.disabled=true;this.render()
    requiredElement<SVGUseElement>(button,'use').setAttribute('href','/assets/icons.svg#loader-2');requiredElement<SVGElement>(button,'svg').classList.add('animate-spin')
    try{await this.request('/api/user/comments/'+id,{method:'DELETE'},'Failed to delete comment');button.closest('[data-settings-comment]')?.remove()}
    catch(error){this.showError(error)}finally{button.disabled=false;requiredElement<SVGUseElement>(button,'use').setAttribute('href','/assets/icons.svg#trash-2');requiredElement<SVGElement>(button,'svg').classList.remove('animate-spin');this.render()}
  }
}
registerController('settings-comments',SettingsCommentsController)
