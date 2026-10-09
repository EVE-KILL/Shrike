import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {readBrowserSession} from './browser-request'
import {eventElement,placeElement} from './shared/dom'
import {isAbortError,errorMessage} from './shared/json'

export class CommentsFeedController extends RequestComponent {
  private rows=new Map<number,HTMLElement>()
  private cursor:string|null=null
  private filter=0
  private query=''
  private loading=false
  private pending:AbortController|undefined
  private debounce:ReturnType<typeof setTimeout>|undefined
  private currentUser=0
  private generation=0
  private initialized=false
  private interrupted=false
  private deleting=new Map<number,object>()
  private get grid(){return this.q<HTMLElement>('[data-comments-grid]')}
  private get search(){return this.q<HTMLInputElement>('[data-comments-search]')}
  private get sort(){return this.q<HTMLSelectElement>('[data-comments-sort]')}
  connect(){
    if(!this.initialized){this.initialized=true;const initial=this.q<HTMLElement>('[data-comments-initial-next]');this.cursor=initial.dataset.commentsInitialNext??null;for(const row of this.grid.querySelectorAll<HTMLElement>('.comment-feed-card')){const id=Number(row.dataset.commentId);if(Number.isSafeInteger(id)&&id>0)this.rows.set(id,row)}if(initial.dataset.commentsInitialError)this.error('[data-comments-error]',initial.dataset.commentsInitialError)}
    const generation=++this.generation;void readBrowserSession().then(user=>{if(generation===this.generation&&this.element.isConnected){this.currentUser=user?.characterId??0;this.sync()}}).catch(()=>{});this.sync();if(this.interrupted)void this.load(true)
  }
  private hasFilters(){return this.filter!==0||this.query.length>=2||this.sort.value==='oldest'}
  private sync(){
    const nodes=Array.from(this.rows.values());if(this.sort.value==='oldest')nodes.reverse();const keep=new Set(nodes)
    for(const [index,node]of nodes.entries())placeElement(this.grid,node,this.grid.children[index]??null)
    for(const node of Array.from(this.grid.children))if(!(node instanceof HTMLElement)||!keep.has(node))node.remove()
    this.q<HTMLElement>('[data-comments-shown]').innerHTML=nodes.length?`<b>${nodes.length}</b> shown`:'';this.q<HTMLElement>('[data-comments-empty]').hidden=nodes.length!==0||this.loading;this.q<HTMLElement>('[data-comments-empty-text]').textContent=this.hasFilters()?'Nothing matches the current filters. Try clearing them.':'Be the first to leave one on a killmail or battle.'
    this.q<HTMLElement>('[data-comments-clear]').hidden=this.q<HTMLElement>('[data-comments-empty-clear]').hidden=!this.hasFilters();this.q<HTMLElement>('[data-comments-search-clear]').hidden=!this.search.value;this.q<HTMLElement>('[data-comments-load-wrap]').hidden=!nodes.length||!this.cursor
    const more=this.q<HTMLButtonElement>('[data-comments-more]');more.disabled=this.loading;more.textContent=this.loading?'Loading…':'⌄ Load more comments';this.q<HTMLElement>('[data-comments-end]').hidden=!nodes.length||Boolean(this.cursor)
    for(const button of this.all<HTMLButtonElement>('[data-comment-filter]')){const active=Number(button.dataset.commentFilter)===this.filter;button.classList.toggle('active',active);button.setAttribute('aria-pressed',String(active))}
    for(const button of this.all<HTMLButtonElement>('[data-comment-delete]')){const card=button.closest<HTMLElement>('[data-author-id]');button.hidden=!this.currentUser||Number(card?.dataset.authorId)!==this.currentUser||card?.classList.contains('deleted')===true;button.disabled=this.deleting.has(Number(button.dataset.commentDelete))}
  }
  private async load(reset:boolean){
    this.pending?.abort();const controller=new AbortController();this.pending=controller;this.interrupted=false;this.loading=true;this.error('[data-comments-error]');this.grid.classList.add('loading');this.sync()
    const params=new URLSearchParams();if(this.filter)params.set('target_type',String(this.filter));if(this.query.length>=2)params.set('q',this.query);if(!reset&&this.cursor)params.set('cursor',this.cursor)
    try{const result=await this.readTextWithHeaders(`/_fragments/comments?${params}`,'Failed to load comments',{signal:controller.signal});if(this.pending!==controller)return;const documentFragment=new DOMParser().parseFromString(result.text,'text/html'),cards=documentFragment.querySelectorAll<HTMLElement>('.comment-feed-card');if(!documentFragment.querySelector('[data-comments-fragment]'))throw new Error('Failed to load comments');if(reset)this.rows.clear();for(const card of cards){const id=Number(card.dataset.commentId);if(Number.isSafeInteger(id)&&id>0&&!this.rows.has(id))this.rows.set(id,document.importNode(card,true))}this.cursor=result.headers.get('X-Next-Cursor')}
    catch(error){if(this.pending===controller)this.error('[data-comments-error]',error)}finally{if(this.pending===controller){this.loading=false;this.pending=undefined;this.grid.classList.remove('loading');this.sync()}}
  }
  private clear(){this.stopTimer(this.debounce);this.filter=0;this.query='';this.search.value='';this.sort.value='newest';void this.load(true)}
  searchChanged(event:Event){if(event.target!==this.search)return;this.stopTimer(this.debounce);this.pending?.abort();this.interrupted=true;this.q<HTMLElement>('[data-comments-search-clear]').hidden=!this.search.value;this.debounce=this.after(300,()=>{this.debounce=undefined;this.query=this.search.value.trim();void this.load(true)})}
  change(event:Event){if(event.target===this.sort)this.sync()}
  async click(event:MouseEvent){
    const target=eventElement(event.target),filter=target?.closest<HTMLElement>('[data-comment-filter]')
    if(filter){this.stopTimer(this.debounce);this.filter=Number(filter.dataset.commentFilter)||0;await this.load(true);return}
    if(target?.closest('[data-comments-search-clear]')){this.stopTimer(this.debounce);this.search.value=this.query='';await this.load(true);return}
    if(target?.closest('[data-comments-clear],[data-comments-empty-clear]')){this.clear();return}
    if(target?.closest('[data-comments-refresh]')){await this.load(true);return}
    if(target?.closest('[data-comments-more]')){if(this.cursor&&!this.loading)await this.load(false);return}
    const button=target?.closest<HTMLButtonElement>('[data-comment-delete]'),id=Number(button?.dataset.commentDelete)
    if(!button||button.hidden||button.disabled||!Number.isSafeInteger(id)||id<=0||!confirm('Delete this comment?'))return
    const token={};this.deleting.set(id,token);this.sync()
    try{await this.request(`/api/comments/${id}`,{method:'DELETE'},'Failed to delete');if(this.deleting.get(id)!==token)return;this.rows.delete(id);this.sync()}
    catch(error){if(!isAbortError(error)&&this.deleting.get(id)===token)alert(errorMessage(error,'Failed to delete'))}finally{if(this.deleting.get(id)===token){this.deleting.delete(id);this.sync()}}
  }
  show(event:PageTransitionEvent){if(event.persisted&&this.interrupted){this.query=this.search.value.trim();void this.load(true)}}
  hide(){this.interrupted||=Boolean(this.pending)||Boolean(this.debounce)||this.deleting.size>0;this.stopTimer(this.debounce);this.debounce=undefined;this.pending?.abort();this.pending=undefined;this.loading=false;this.deleting.clear();this.grid.classList.remove('loading');super.hide();this.sync()}
  disconnect(){++this.generation;super.disconnect()}
}
registerController('comments-feed',CommentsFeedController)
