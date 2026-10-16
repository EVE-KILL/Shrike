import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {readLocalSearches,savedSearchDocument,type LocalSavedSearch,type SavedSearchDocument} from './saved-search-data'
import {readBrowserSession,type BrowserUser} from './browser-request'
import {eventElement} from './shared/dom'
import {errorMessage,isAbortError} from './shared/json'
import type {AccountSavedSearchesGetResponses,AccountSavedSearchesPostData,AuthMeLegacyResponses} from '../../shared/api/types.gen'

type AccountSearch=AccountSavedSearchesGetResponses[200]['searches'][number]
type EditorKind='create'|'rename'|'replace'|'import'
type Editor={kind:EditorKind;name:string;document:SavedSearchDocument;row?:AccountSearch;localIndex?:number}
export interface SavedSearchCallbacks {current:()=>SavedSearchDocument;load:(document:SavedSearchDocument)=>void}
export interface SavedSearchCurrent {document?:SavedSearchDocument;error?:unknown}
export interface SavedSearchLoad {document:SavedSearchDocument;error?:unknown}
const storageKey='evekill-saved-searches'
function node<K extends keyof HTMLElementTagNameMap>(tag:K,cls:string,text?:string){const element=document.createElement(tag);element.className=cls;if(text!==undefined)element.textContent=text;return element}

export class SavedSearchesController extends RequestComponent {
  private user:BrowserUser|null=null
  private account:AccountSearch[]=[]
  private local:LocalSavedSearch[]=[]
  private opened=false
  private busy:object|undefined
  private pending:AbortController|undefined
  private loading=false
  private ready=false
  private identifying=true
  private identityGeneration=0
  private initialized=false
  private editor:Editor|null=null
  private deleting:AccountSearch|number|null=null
  private revision=0
  private errorText=''
  private message=''
  private get trigger(){return this.q<HTMLElement>('[data-saved-searches]')}
  private get panel(){return this.q<HTMLDivElement>('[data-saved-panel]')}
  private get form(){return this.q<HTMLFormElement>('[data-saved-editor]')}
  private get nameInput(){return this.q<HTMLInputElement>('[data-saved-editor] [name="name"]')}
  private get toggle(){return this.q<HTMLButtonElement>('[data-saved-toggle]')}
  connect(){
    if(!this.initialized){this.initialized=true;try{this.local=readLocalSearches(localStorage.getItem(storageKey))}catch(error){this.errorText=errorMessage(error)}}
    this.render();void this.identity()
  }
  layout(){
    if(!this.opened)return
    const box=this.trigger.getBoundingClientRect(),width=Math.min(384,innerWidth-32),top=Math.max(16,Math.min(box.bottom+8,innerHeight-220))
    Object.assign(this.panel.style,{width:width+'px',left:Math.max(16,Math.min(box.right-width,innerWidth-width-16))+'px',top:top+'px',maxHeight:innerHeight-top-16+'px'})
  }
  private setText(selector:string,text:string){const element=this.q<HTMLElement>(selector);element.hidden=!text;element.textContent=text}
  close(){this.opened=false;this.render()}
  private current(){const detail:SavedSearchCurrent={};this.dispatch('current',{detail});if(detail.error)throw detail.error;if(!detail.document)throw new Error('The current search is not available.');return detail.document}
  private load(row:AccountSearch|LocalSavedSearch){
    try{const document=row.document??('filters' in row?savedSearchDocument(row.filters):undefined);if(!document)throw new Error('This search could not be read.');const detail:SavedSearchLoad={document};this.dispatch('load',{detail});if(detail.error)throw detail.error;this.close()}catch(error){this.errorText=errorMessage(error);this.render()}
  }
  private action(label:string,action:string,key:string|number,disabled=false){const button=node('button','',label);button.type='button';button.disabled=disabled;button.dataset.savedRowAction=action;button.dataset.savedRowKey=String(key);return button}
  private renderRows(){
    const accounts=this.q<HTMLElement>('[data-saved-account-rows]'),browsers=this.q<HTMLElement>('[data-saved-browser-rows]');accounts.replaceChildren();browsers.replaceChildren()
    if(this.user){
      if(this.ready&&!this.account.length)accounts.append(node('p','text-xs text-gray-500 py-2','No account searches yet.'))
      for(const row of this.account){const card=node('div','py-3 border-b border-white/[0.06]'),name=this.action(row.name,'load-account',row.id),actions=node('div','flex flex-wrap gap-3 mt-2 text-xs text-gray-400');name.className='text-sm text-gray-200 text-left break-words';actions.append(this.action('Rename','rename',row.id,Boolean(this.busy)||!this.ready),this.action('Use current filters','replace',row.id,Boolean(this.busy)||!this.ready),this.action('Delete','delete-account',row.id,Boolean(this.busy)||!this.ready));card.append(name,actions);accounts.append(card)}
    }else if(!this.local.length)browsers.append(node('p','text-xs text-gray-500 py-2','No browser searches yet.'))
    this.q<HTMLElement>('[data-saved-import-note]').hidden=!(this.user&&this.local.length)
    for(const [index,row]of this.local.entries()){const card=node('div','py-3 border-b border-white/[0.06]'),name=this.action(row.name,'load-local',index),actions=node('div','flex gap-3 mt-2 text-xs text-gray-400');name.className='text-sm text-gray-200 text-left break-words';if(this.user)actions.append(this.action('Import to account','import',index,Boolean(this.busy)||!this.ready));actions.append(this.action('Delete browser copy','delete-local',index,Boolean(this.busy)));card.append(name,actions);browsers.append(card)}
  }
  private render(){
    this.panel.hidden=!this.opened
    if(this.panel.showPopover){if(this.opened&&!this.panel.matches(':popover-open'))this.panel.showPopover();else if(!this.opened&&this.panel.matches(':popover-open'))this.panel.hidePopover()}
    this.toggle.setAttribute('aria-expanded',String(this.opened));this.q<HTMLButtonElement>('[data-saved-create]').disabled=Boolean(this.busy)||this.identifying||Boolean(this.user&&!this.ready)
    this.q<HTMLElement>('[data-saved-count]').textContent=`Saved (${this.user?this.account.length:this.local.length})`;this.q<HTMLElement>('[data-saved-title]').textContent=this.user?'Account searches':'Browser searches'
    this.setText('[data-saved-error]',this.errorText);this.setText('[data-saved-message]',this.message);this.q<HTMLElement>('[data-saved-loading]').hidden=!this.loading
    const refresh=this.q<HTMLButtonElement>('[data-saved-refresh]');refresh.hidden=!this.user;refresh.disabled=Boolean(this.busy)||this.loading;this.form.hidden=!this.editor;this.q<HTMLElement>('[data-saved-deleting]').hidden=this.deleting===null
    for(const button of this.all<HTMLButtonElement>('[data-saved-cancel],[data-saved-delete-confirm],[data-saved-delete-cancel]'))button.disabled=Boolean(this.busy)
    const submit=this.q<HTMLButtonElement>('[data-saved-submit]');submit.disabled=Boolean(this.busy)||!this.nameInput.value.trim();submit.textContent=this.busy?'Saving…':'Save search'
    this.q<HTMLElement>('[data-saved-owner-note]').textContent=this.user?'Private to this account, available across devices.':'Stored in this browser. Sign in to save across devices.'
    if(this.editor)this.q<HTMLElement>('[data-saved-editor-title]').textContent={create:'Save current search',rename:'Rename search',replace:'Replace with current filters',import:'Import browser search'}[this.editor.kind]
    this.renderRows();this.layout()
  }
  private async refresh(){
    this.pending?.abort();const controller=new AbortController();this.pending=controller;this.ready=false;this.errorText='';this.message=''
    if(!this.user){this.pending=undefined;this.loading=false;this.render();return}
    const owner=this.user.characterId;this.loading=true;this.render()
    try{const result=await this.request<AccountSavedSearchesGetResponses[200]>('/api/me/saved-searches',{signal:controller.signal});if(this.pending===controller&&this.user?.characterId===owner){this.account=result.searches;this.ready=true}}
    catch(error){if(this.pending===controller&&!isAbortError(error))this.errorText=errorMessage(error)}finally{if(this.pending===controller){this.pending=undefined;this.loading=false;this.render()}}
  }
  async identity(event?:Event){
    const epoch=++this.identityGeneration
    try{
      const next=event?(await this.request<AuthMeLegacyResponses[200]>('/auth/me')).user??null:await readBrowserSession()
      if(epoch!==this.identityGeneration||!this.element.isConnected)return
      if(this.user?.characterId!==next?.characterId){const opened=this.opened;this.hide();this.opened=opened;this.user=next;this.account=[];this.editor=this.deleting=null;this.errorText=this.message='';await this.refresh()}
      else if(next&&!this.ready)await this.refresh()
    }catch{/* Anonymous/browser searches remain available if identity is unavailable. */}
    finally{if(epoch===this.identityGeneration&&this.element.isConnected){this.identifying=false;this.render()}}
  }
  private begin(kind:EditorKind,row?:AccountSearch,localIndex?:number){
    if(this.busy)return
    this.errorText=this.message='';this.deleting=null
    try{
      const browser=localIndex===undefined?undefined:this.local[localIndex],document=kind==='rename'?row?.document:kind==='import'?browser?.document??(browser?savedSearchDocument(browser.filters):undefined):this.current()
      if(!document)throw new Error('This search could not be read.')
      this.editor={kind,name:row?.name??browser?.name??'',document,...(row?{row}:{}),...(localIndex!==undefined?{localIndex}:{})};++this.revision;this.nameInput.value=this.editor.name
    }catch(error){this.errorText=errorMessage(error)}
    this.opened=true;this.render();if(this.editor)this.nameInput.focus()
  }
  private persistLocal(next:LocalSavedSearch[]){localStorage.setItem(storageKey,JSON.stringify(next));this.local=next}
  async save(event:SubmitEvent){
    if(event.target!==this.form)return;event.preventDefault();if(!this.editor||this.busy)return
    const draft=this.editor,name=this.nameInput.value.trim(),revision=this.revision
    if(!name||[...name].length>100){this.errorText='Use a name of 1–100 characters';this.render();return}
    const token={};this.busy=token;this.errorText='';this.render();const owner=this.user?.characterId
    try{
      if(owner){
        if(!this.ready)throw new Error('Refresh account searches before saving')
        const body:AccountSavedSearchesPostData['body']={name,document:draft.document,revision:draft.row?.revision??0}
        const result=await this.request<AccountSearch>(`/api/me/saved-searches${draft.row?'/'+draft.row.id:''}`,{method:draft.row?'PUT':'POST',body})
        if(this.busy!==token||owner!==this.user?.characterId)return
        if(this.editor===draft)draft.row=result
        await this.refresh();if(this.busy!==token)return;this.message=draft.kind==='import'?'Imported. The browser copy has been kept.':'Saved to your account.'
      }else{
        if(this.local.length>=50)throw new Error('Browser saved-search limit reached (50)')
        if(this.local.some(row=>row.name.trim().toLowerCase()===name.toLowerCase()))throw new Error('A browser search with this name already exists. Choose another name.')
        this.persistLocal([...this.local,{name,filters:JSON.stringify(draft.document.filters),document:draft.document,date:new Date().toISOString()}]);this.message='Saved in this browser.'
      }
      if(this.editor===draft&&this.revision===revision)this.editor=null
    }catch(error){if(this.busy===token&&owner===this.user?.characterId&&!isAbortError(error))this.errorText=errorMessage(error)}finally{if(this.busy===token){this.busy=undefined;this.render()}}
  }
  private async remove(){
    if(this.deleting===null||this.busy)return
    const deleting=this.deleting,token={},owner=this.user?.characterId;this.busy=token;this.errorText='';this.render()
    try{if(typeof deleting==='number')this.persistLocal(this.local.filter((_,index)=>index!==deleting));else{await this.request(`/api/me/saved-searches/${deleting.id}?revision=${deleting.revision}`,{method:'DELETE'});if(this.busy!==token||owner!==this.user?.characterId)return;await this.refresh()}if(this.busy===token)this.deleting=null}
    catch(error){if(this.busy===token&&owner===this.user?.characterId&&!isAbortError(error))this.errorText=errorMessage(error)}finally{if(this.busy===token){this.busy=undefined;this.render()}}
  }
  input(event:Event){if(event.target===this.nameInput){if(this.editor)this.editor.name=this.nameInput.value;++this.revision;this.q<HTMLButtonElement>('[data-saved-submit]').disabled=Boolean(this.busy)||!this.nameInput.value.trim()}}
  async click(event:MouseEvent){
    const target=eventElement(event.target)
    if(target?.closest('[data-saved-create]')){this.begin('create');return}
    if(target?.closest('[data-saved-toggle]')){this.opened=!this.opened;this.render();if(this.opened)await this.identity(event);return}
    if(target?.closest('[data-saved-close]')){this.close();return}if(target?.closest('[data-saved-refresh]')){await this.refresh();return}
    if(target?.closest('[data-saved-cancel]')){this.editor=null;this.render();return}if(target?.closest('[data-saved-delete-cancel]')){this.deleting=null;this.render();return}if(target?.closest('[data-saved-delete-confirm]')){await this.remove();return}
    const button=target?.closest<HTMLButtonElement>('[data-saved-row-action]');if(!button||button.disabled)return
    const key=Number(button.dataset.savedRowKey),action=button.dataset.savedRowAction,account=this.account.find(row=>row.id===key),local=this.local[key]
    if(action==='load-account'&&account)this.load(account)
    else if(action==='load-local'&&local)this.load(local)
    else if((action==='rename'||action==='replace')&&account)this.begin(action,account)
    else if(action==='import'&&local)this.begin('import',undefined,key)
    else if((action==='delete-account'&&account)||(action==='delete-local'&&local)){this.deleting=action==='delete-account'&&account?account:key;this.editor=null;this.render()}
  }
  outside(event:PointerEvent){const target=eventElement(event.target);if(this.opened&&!this.panel.contains(target)&&!this.trigger.contains(target))this.close()}
  keyboard(event:KeyboardEvent){if(event.key==='Escape'&&this.panel.contains(eventElement(event.target))){this.close();this.toggle.focus()}}
  show(event:PageTransitionEvent){if(event.persisted)void this.identity(event)}
  hide(){this.pending?.abort();this.pending=undefined;this.ready=false;this.loading=false;this.busy=undefined;this.deleting=null;this.opened=false;super.hide();this.render()}
  disconnect(){++this.identityGeneration;super.disconnect()}
}
registerController('saved-searches',SavedSearchesController)

