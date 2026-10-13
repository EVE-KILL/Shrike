import {AdminComponent} from './controllers/admin/base'
import {registerController} from './controllers/application'
import {readQuerySeed} from './shared/query-seed'
import {eventElement} from './shared/dom'
import {errorMessage,isAbortError,isRecord} from './shared/json'
import {toast} from './toast'

type ReviewAction='approve'|'reject'|'hide'|'restore'|'delete'
const isAction=(value:unknown):value is ReviewAction=>value==='approve'||value==='reject'||value==='hide'||value==='restore'||value==='delete'
export class AdminReviewController extends AdminComponent {
  private query:Record<string,string>|undefined
  private pending:AbortController|undefined
  private busy=new Map<string,object>()
  private confirmId:string|null=null
  private confirmTimer:ReturnType<typeof setTimeout>|undefined
  private interrupted=false
  private parent:HTMLElement|null=null
  private visibility=(event:Event)=>{if(event instanceof CustomEvent){if(!event.detail)this.hide();else if(this.interrupted||!this.q<HTMLElement>('[data-admin-review-error]').hidden)void this.load()}}
  private get section(){return this.element.dataset.adminReview==='comments'?'comments':'moderation'}
  private get host(){return this.q<HTMLElement>('[data-admin-review-results]')}
  connect(){
    this.query??=readQuerySeed(this.element.dataset.query);this.parent=this.element.closest('[data-admin-section]');this.parent?.addEventListener('admin-visibility',this.visibility);this.render()
    if(this.interrupted&&!this.parent?.hidden)void this.load()
  }
  private render(){
    if(!this.element.isConnected)return
    for(const button of this.all<HTMLElement>('[data-admin-review-filter]')){
      const selected=this.query?.[button.dataset.adminReviewFilter??'']===button.dataset.value;button.setAttribute('aria-pressed',String(selected))
      button.className=this.section==='comments'?'px-3 py-1.5 rounded-md text-xs font-medium transition-colors cursor-pointer '+(selected?'bg-blue-500/[0.15] text-blue-300 border border-blue-500/30':'bg-white/[0.04] text-gray-400 hover:bg-white/[0.08] border border-transparent'):'px-3 py-1 rounded text-xs font-medium transition-colors cursor-pointer '+(selected?'bg-blue-500/[0.15] text-blue-300':'text-gray-400 hover:text-white')
    }
    for(const row of this.host.querySelectorAll<HTMLElement>('[data-admin-review-id]'))for(const button of row.querySelectorAll<HTMLButtonElement>('[data-admin-review-action]')){
      button.disabled=this.busy.has(row.dataset.adminReviewId??'')
      if(button.dataset.adminReviewAction==='delete'){const armed=this.confirmId===row.dataset.adminReviewId;button.textContent=armed?'Confirm delete':'Delete';button.className='px-3 py-1 rounded-md text-xs bg-red-500/[0.15] hover:bg-red-500/[0.25] cursor-pointer transition-colors disabled:opacity-40 disabled:cursor-not-allowed '+(armed?'text-red-200 ring-1 ring-red-500/40':'text-red-300')}
    }
    const next=this.host.querySelector<HTMLElement>('[data-admin-review-result]')
    if(next?.dataset.counts&&next.dataset.failed!=='true'){
      let counts:unknown;try{counts=JSON.parse(next.dataset.counts)}catch{counts={}}
      this.q<HTMLElement>('[data-admin-review-counts]').hidden=false
      for(const count of this.all<HTMLElement>('[data-admin-review-count]')){const value=isRecord(counts)?counts[count.dataset.adminReviewCount??'']:undefined;count.textContent=typeof value==='number'&&Number.isFinite(value)?String(value):'0'}
    }
  }
  private fail(message:string){
    const error=this.q<HTMLElement>('[data-admin-review-error]');error.hidden=!message;if(!message)return
    error.replaceChildren(document.createTextNode(message+' '));const retry=document.createElement('button');retry.type='button';retry.dataset.adminReviewRefresh='';retry.className='text-blue-400';retry.textContent='Retry';error.append(retry)
  }
  private async load(){
    this.pending?.abort();const controller=new AbortController();this.pending=controller;this.interrupted=false;this.host.setAttribute('aria-busy','true');this.fail('')
    const fallback=this.section==='comments'?'Could not load comments.':'Could not load the moderation queue.'
    try{
      const text=await this.html(`/_fragments/admin/${this.section}?`+new URLSearchParams(this.query),fallback,{signal:controller.signal})
      const parsed=new DOMParser().parseFromString(text,'text/html'),source=parsed.querySelector<HTMLElement>('[data-admin-review-result]')
      if(!source||source.dataset.failed==='true')throw new Error(fallback)
      if(this.pending!==controller)return
      const next=document.importNode(source,true),key=(node:Element)=>(node.closest<HTMLElement>('[data-admin-review-id]')?.dataset.adminReviewId||'')+':'+node.outerHTML
      const media=new Map([...this.host.querySelectorAll('img,video,iframe')].map(node=>[key(node),node]))
      for(const node of next.querySelectorAll('img,video,iframe')){const identity=key(node),existing=media.get(identity);if(existing){node.replaceWith(existing);media.delete(identity)}}
      this.host.replaceChildren(next);this.render()
    }catch(error){if(!isAbortError(error)&&this.pending===controller&&this.element.isConnected)this.fail(errorMessage(error))}
    finally{if(this.pending===controller){this.pending=undefined;this.host.removeAttribute('aria-busy')}}
  }
  private async act(id:string,action:ReviewAction){
    if(this.busy.has(id)||!/^\d+$/.test(id))return
    if(this.section==='moderation'&&(action!=='approve'&&action!=='reject'))return
    if(this.section==='comments'&&(action==='approve'||action==='reject'))return
    if(action==='delete'&&this.confirmId!==id){this.stopTimer(this.confirmTimer);this.confirmId=id;this.render();this.confirmTimer=this.after(5000,()=>{this.confirmId=null;this.render()});return}
    if(action==='delete'){this.stopTimer(this.confirmTimer);this.confirmId=null}
    const token={};this.busy.set(id,token);this.render()
    try{await this.request(action==='delete'?`/api/comments/${id}`:`/api/admin/${this.section}/${id}/${action}`,{method:action==='delete'?'DELETE':'POST'});await this.load()}
    catch(error){if(this.element.isConnected&&!isAbortError(error))toast(errorMessage(error),'error')}
    finally{if(this.busy.get(id)===token)this.busy.delete(id);this.render()}
  }
  async click(event:MouseEvent){
    const target=eventElement(event.target),filter=target?.closest<HTMLElement>('[data-admin-review-filter]')
    if(filter&&this.query){const key=filter.dataset.adminReviewFilter,value=filter.dataset.value;if(!key||value===undefined||this.query[key]===value)return;this.query[key]=value;this.render();await this.load();return}
    const button=target?.closest<HTMLElement>('[data-admin-review-action]'),action=button?.dataset.adminReviewAction,id=button?.closest<HTMLElement>('[data-admin-review-id]')?.dataset.adminReviewId
    if(id&&isAction(action)){await this.act(id,action);return}
    if(target?.closest('[data-admin-review-refresh]'))await this.load()
  }
  hide(){this.interrupted||=Boolean(this.pending)||this.busy.size>0;this.pending?.abort();this.pending=undefined;this.busy.clear();this.host.removeAttribute('aria-busy');this.stopTimer(this.confirmTimer);this.confirmId=null;super.hide();this.render()}
  show(event:PageTransitionEvent){if(event.persisted&&this.interrupted&&!this.parent?.hidden)void this.load()}
  disconnect(){this.parent?.removeEventListener('admin-visibility',this.visibility);this.parent=null;super.disconnect()}
}
registerController('admin-review',AdminReviewController)
