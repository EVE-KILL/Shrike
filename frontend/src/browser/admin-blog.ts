import {AdminSectionComponent} from './controllers/admin/base'
import {registerController} from './controllers/application'
import {createSelectMenu,type SelectMenu} from './select-menu'
import {initializeDateTimePickers,closeDateTimePickers} from './date-time-picker'
import {blogForm,blogPayload,parseTagInput} from './admin-data'
import type {AdminBlogPost} from '../utils/adminBlog'
import {readBlogSeed} from './shared/admin-editor-seeds'
import {eventElement,requiredElement} from './shared/dom'
import {errorMessage,isAbortError} from './shared/json'
import {isBackdropClick} from './shared/dialog'
import {toast} from './toast'
import type {BlogAdminCreateResponses} from '../../shared/api/types.gen'

const fields=['title','slug','excerpt','body_md','cover_image_url','tags','published_at'] as const
const stringField=(value:string):value is typeof fields[number]=>fields.some(field=>field===value)
export class AdminBlogController extends AdminSectionComponent {
  private status='all'
  private rows:AdminBlogPost[]=[]
  private model=blogForm()
  private editId:number|null=null
  private revision=0
  private saving:object|undefined
  private deleting:object|undefined
  private pending:AbortController|undefined
  private confirmId:number|null=null
  private confirmTimer:ReturnType<typeof setTimeout>|undefined
  private hasLoaded=false
  private initialized=false
  private interrupted=false
  private menu:SelectMenu|undefined
  private get host(){return this.q<HTMLElement>('[data-admin-blog-results]')}
  private get dialog(){return this.q<HTMLDialogElement>('dialog')}
  private get form(){return this.q<HTMLElement>('[data-admin-blog-form]')}
  connect(){
    super.connect();if(!this.initialized){this.status=this.element.dataset.status??'all';this.hasLoaded=this.element.dataset.failed!=='true';this.initialized=true}
    this.menu=createSelectMenu('Blog status',[{value:'all',label:'All posts'},{value:'draft',label:'Draft'},{value:'published',label:'Published'},{value:'archived',label:'Archived'}],this.status,value=>{this.status=value;void this.load()});this.q<HTMLElement>('[data-admin-blog-filter]').append(this.menu.root)
    initializeDateTimePickers(this.form);this.readRows()
  }
  private readRows(){this.rows=readBlogSeed(this.host.querySelector<HTMLElement>('[data-admin-blog-list]')?.dataset.initial)}
  private listError(error:unknown=''){
    if(!this.element.isConnected||isAbortError(error))return
    const message=typeof error==='string'?error:errorMessage(error),box=this.q<HTMLElement>('[data-admin-blog-error]');box.hidden=!message
    if(message){box.replaceChildren(document.createTextNode(message+' '));const retry=document.createElement('button');retry.type='button';retry.dataset.adminBlogRetry='';retry.className='text-blue-400';retry.textContent='Retry';box.append(retry)}
  }
  private async load(){
    this.pending?.abort();const controller=new AbortController();this.pending=controller;this.interrupted=false;this.listError();this.host.setAttribute('aria-busy','true')
    try{
      const text=await this.html('/_fragments/admin/blog?'+new URLSearchParams({status:this.status}),'Could not load blog posts.',{signal:controller.signal})
      const doc=new DOMParser().parseFromString(text,'text/html'),next=doc.querySelector<HTMLElement>('[data-admin-blog-list]')
      if(!next||doc.querySelector('[data-admin-blog-error]:not([hidden])'))throw new Error('Could not load blog posts.')
      if(this.pending!==controller)return
      const focus=this.host.contains(document.activeElement)?document.activeElement:null,row=focus?.closest<HTMLElement>('[data-admin-blog-id]'),id=row?.dataset.adminBlogId,action=focus?.hasAttribute('data-admin-blog-edit')?'edit':focus?.hasAttribute('data-admin-blog-delete')?'delete':undefined
      this.host.replaceChildren(document.importNode(next,true));this.readRows();this.hasLoaded=true;this.resetConfirm();this.renderDeleting()
      if(action&&id)this.host.querySelector<HTMLButtonElement>(`[data-admin-blog-id="${id}"] [data-admin-blog-${action}]`)?.focus()
    }catch(error){if(this.pending===controller)this.listError(error)}finally{if(this.pending===controller){this.pending=undefined;this.host.removeAttribute('aria-busy')}}
  }
  private renderForm(){
    for(const button of this.form.querySelectorAll<HTMLButtonElement>('[data-admin-blog-status]')){const selected=Number(button.dataset.adminBlogStatus)===this.model.status;button.className='flex-1 px-3 py-2 rounded-lg text-xs font-medium border transition-colors cursor-pointer '+(selected?'bg-blue-500/20 border-blue-500/40 text-blue-300':'bg-white/[0.02] border-white/[0.08] text-gray-500 hover:text-gray-300');button.setAttribute('aria-pressed',String(selected))}
    const preview=this.q<HTMLElement>('[data-admin-blog-tag-preview]');preview.hidden=!this.model.tags.trim();preview.replaceChildren(...parseTagInput(this.model.tags).map(tag=>{const span=document.createElement('span');span.className='inline-flex items-center px-1.5 py-0.5 rounded-full text-[10px] font-medium bg-blue-500/10 text-blue-300 border border-blue-500/20 max-w-full break-all';span.textContent='#'+tag;return span}))
  }
  private open(row?:AdminBlogPost){
    if(this.saving)return
    this.editId=row?.id??null;this.model=blogForm(row);++this.revision;this.q<HTMLElement>('[data-admin-blog-form-error]').hidden=true
    const title=row?'Edit Post':'New Post';this.q<HTMLElement>('[data-admin-modal-title]').textContent=title;this.dialog.setAttribute('aria-label',title);this.q<HTMLElement>('[data-admin-blog-save-label]').textContent=row?'Update':'Create'
    for(const input of this.form.querySelectorAll<HTMLInputElement|HTMLTextAreaElement>('[name]'))if(stringField(input.name)){input.value=this.model[input.name];if(input.hasAttribute('data-date-time-value'))input.dispatchEvent(new Event('change',{bubbles:true}))}
    this.renderForm();this.dialog.showModal()
  }
  close(){closeDateTimePickers(this.form);this.dialog.close()}
  private fail(error:unknown){if(!this.element.isConnected||isAbortError(error))return;this.q<HTMLElement>('[data-admin-blog-form-error]').hidden=false;this.q<HTMLElement>('[data-admin-blog-form-message]').textContent=typeof error==='string'?error:errorMessage(error)}
  private async submit(){
    if(this.saving)return;if(!this.model.title.trim()){this.fail('Title is required');return}
    let payload:ReturnType<typeof blogPayload>;try{payload=blogPayload(this.model,this.editId!==null)}catch{this.fail('Enter a valid publish date');return}
    const token={},id=this.editId,revision=this.revision;this.saving=token;const save=this.q<HTMLButtonElement>('[data-admin-blog-save]');save.disabled=true;this.q<HTMLElement>('[data-admin-blog-spinner]').hidden=false;this.q<HTMLElement>('[data-admin-blog-form-error]').hidden=true
    try{
      const response=await this.request<BlogAdminCreateResponses[200]>('/api/admin/blog'+(id?'/'+id:''),{method:id?'PATCH':'POST',body:payload})
      if(this.saving!==token)return
      if(!id&&typeof response.post.id==='number'){this.editId=response.post.id;this.q<HTMLElement>('[data-admin-blog-save-label]').textContent='Update'}
      if(this.revision===revision)this.close();await this.load()
    }catch(error){if(this.saving===token)this.fail(error)}finally{if(this.saving===token){this.saving=undefined;save.disabled=false;this.q<HTMLElement>('[data-admin-blog-spinner]').hidden=true}}
  }
  private renderConfirm(){for(const button of this.host.querySelectorAll<HTMLButtonElement>('[data-admin-blog-delete]')){const armed=Number(button.closest<HTMLElement>('[data-admin-blog-id]')?.dataset.adminBlogId)===this.confirmId;button.className='flex items-center justify-center h-7 rounded-md transition-colors cursor-pointer '+(armed?'px-2 gap-1 text-red-300 bg-red-500/15 ring-1 ring-red-500/40':'w-7 text-gray-500 hover:text-red-400 hover:bg-red-500/[0.08]');button.title=armed?'Click again to delete permanently':'Delete';button.setAttribute('aria-pressed',String(armed));requiredElement<HTMLElement>(button,'[data-admin-blog-confirm]').hidden=!armed}}
  private resetConfirm(){this.stopTimer(this.confirmTimer);this.confirmId=null;this.renderConfirm()}
  private renderDeleting(){for(const button of this.host.querySelectorAll<HTMLButtonElement>('[data-admin-blog-delete]'))button.disabled=Boolean(this.deleting)}
  private async remove(id:number){
    if(this.deleting)return
    if(this.confirmId!==id){this.stopTimer(this.confirmTimer);this.confirmId=id;this.renderConfirm();this.confirmTimer=this.after(5000,()=>this.resetConfirm());return}
    this.resetConfirm();const token={};this.deleting=token;this.renderDeleting()
    try{await this.request('/api/admin/blog/'+id,{method:'DELETE'});await this.load()}catch(error){if(this.element.isConnected&&!isAbortError(error))toast(errorMessage(error),'error')}finally{if(this.deleting===token){this.deleting=undefined;this.renderDeleting()}}
  }
  async click(event:MouseEvent){
    const target=eventElement(event.target)
    if(isBackdropClick(this.dialog,event)||target?.closest('[data-admin-modal-close]')){this.close();return}
    if(target?.closest('[data-admin-blog-new]')){this.open();return}
    if(target?.closest('[data-admin-blog-save]')){await this.submit();return}
    if(target?.closest('[data-admin-blog-retry]')){await this.load();return}
    const edit=target?.closest('[data-admin-blog-edit]');if(edit){const row=this.rows.find(row=>row.id===Number(edit.closest<HTMLElement>('[data-admin-blog-id]')?.dataset.adminBlogId));if(row)this.open(row);return}
    const del=target?.closest('[data-admin-blog-delete]');if(del){await this.remove(Number(del.closest<HTMLElement>('[data-admin-blog-id]')?.dataset.adminBlogId));return}
    const choice=target?.closest<HTMLElement>('[data-admin-blog-status]');if(choice){this.model.status=Number(choice.dataset.adminBlogStatus);++this.revision;this.renderForm()}
  }
  input(event:Event){const input=event.target;if((input instanceof HTMLInputElement||input instanceof HTMLTextAreaElement)&&stringField(input.name)){this.model[input.name]=input.value;++this.revision;if(input.name==='tags')this.renderForm()}}
  protected resumed(){if(this.interrupted||(!this.hasLoaded&&!this.pending))void this.load()}
  hide(){this.interrupted||=Boolean(this.pending)||Boolean(this.saving)||Boolean(this.deleting);this.pending?.abort();this.pending=undefined;this.saving=undefined;this.deleting=undefined;this.q<HTMLButtonElement>('[data-admin-blog-save]').disabled=false;this.q<HTMLElement>('[data-admin-blog-spinner]').hidden=true;this.host.removeAttribute('aria-busy');this.resetConfirm();this.renderDeleting();this.close();super.hide()}
  disconnect(){this.menu?.dispose();this.menu?.root.remove();this.menu=undefined;super.disconnect()}
}
registerController('admin-blog',AdminBlogController)
