import {AdminSectionComponent} from './controllers/admin/base'
import {registerController} from './controllers/application'
import {createSelectMenu,type SelectMenu} from './select-menu'
import {initializeDateTimePickers,closeDateTimePickers} from './date-time-picker'
import {announcementForm,announcementPayload,tickerDurations} from './admin-data'
import type {AdminAnnouncement} from '../utils/adminAnnouncement'
import {readAnnouncementSeed} from './shared/admin-editor-seeds'
import {eventElement,requiredElement} from './shared/dom'
import {errorMessage,isAbortError} from './shared/json'
import {isBackdropClick} from './shared/dialog'
import {toast} from './toast'
import type {AnnouncementAdminCreateResponses} from '../../shared/api/types.gen'

const stringFields=['title','body_md','icon','link_url','link_label','starts_at','expires_at'] as const
const stringField=(value:string):value is typeof stringFields[number]=>stringFields.some(field=>field===value)
const selected='bg-blue-500/20 border-blue-500/40 text-blue-300',unselected='bg-white/[0.02] border-white/[0.08] text-gray-500 hover:text-gray-300'
const colors:Record<string,string>={info:selected,warning:'bg-yellow-500/20 border-yellow-500/40 text-yellow-400',danger:'bg-red-500/20 border-red-500/40 text-red-300',success:'bg-green-500/20 border-green-500/40 text-green-300'}
export class AdminAnnouncementsController extends AdminSectionComponent {
  private status='all'
  private rows:AdminAnnouncement[]=[]
  private model=announcementForm()
  private editId:number|null=null
  private revision=0
  private saving:object|undefined
  private pending:AbortController|undefined
  private archiveId:number|null=null
  private archiveTimer:ReturnType<typeof setTimeout>|undefined
  private archiving:object|undefined
  private menu:SelectMenu|undefined
  private interrupted=false
  private initialized=false
  private get host(){return this.q<HTMLElement>('[data-admin-announcement-results]')}
  private get dialog(){return this.q<HTMLDialogElement>('dialog')}
  private get form(){return this.q<HTMLElement>('[data-admin-announcement-form]')}
  connect(){
    super.connect();if(!this.initialized){this.status=this.element.dataset.status??'all';this.initialized=true}
    this.menu=createSelectMenu('Announcement status',[{value:'all',label:'All announcements'},{value:'active',label:'Active'},{value:'scheduled',label:'Scheduled'},{value:'expired',label:'Expired'},{value:'archived',label:'Archived'}],this.status,value=>{this.status=value;void this.load()});this.q<HTMLElement>('[data-admin-announcement-filter]').append(this.menu.root)
    initializeDateTimePickers(this.form);this.readRows()
  }
  private readRows(){this.rows=readAnnouncementSeed(this.host.querySelector<HTMLElement>('[data-admin-announcement-list]')?.dataset.initial)}
  private listError(error:unknown=''){
    if(!this.element.isConnected||isAbortError(error))return
    const message=typeof error==='string'?error:errorMessage(error),box=this.q<HTMLElement>('[data-admin-announcement-error]');box.hidden=!message
    if(message){box.replaceChildren(document.createTextNode(message+' '));const retry=document.createElement('button');retry.type='button';retry.dataset.adminAnnouncementRetry='';retry.className='text-blue-400';retry.textContent='Retry';box.append(retry)}
  }
  private async load(){
    this.pending?.abort();const controller=new AbortController();this.pending=controller;this.interrupted=false;this.listError();this.host.setAttribute('aria-busy','true')
    try{
      const text=await this.html('/_fragments/admin/announcements?'+new URLSearchParams({status:this.status}),'Could not load announcements.',{signal:controller.signal})
      const doc=new DOMParser().parseFromString(text,'text/html'),next=doc.querySelector<HTMLElement>('[data-admin-announcement-list]')
      if(!next||doc.querySelector('[data-admin-announcement-error]:not([hidden])'))throw new Error('Could not load announcements.')
      if(this.pending!==controller)return
      this.host.replaceChildren(document.importNode(next,true));this.readRows();this.resetArchive()
    }catch(error){if(this.pending===controller)this.listError(error)}finally{if(this.pending===controller){this.pending=undefined;this.host.removeAttribute('aria-busy')}}
  }
  private renderChoices(){
    for(const [attr,key]of [['tier','tier'],['color','color'],['duration-value','duration']] as const)for(const button of this.form.querySelectorAll<HTMLButtonElement>(`[data-announcement-${attr}]`)){
      const value=button.getAttribute('data-announcement-'+attr)??'',active=String(this.model[key])===value
      button.className='flex-1 px-3 py-2 rounded-lg text-xs font-medium border transition-colors cursor-pointer '+(attr==='color'?'capitalize ':'')+(active?(attr==='color'?colors[value]??selected:selected):unselected);button.setAttribute('aria-pressed',String(active))
    }
    this.q<HTMLElement>('[data-announcement-duration]').hidden=this.model.tier!==1;this.q<HTMLElement>('[data-announcement-dates]').hidden=this.model.tier===1
  }
  private open(row?:AdminAnnouncement){
    if(this.saving)return
    this.editId=row?.id??null;this.model=announcementForm(row);++this.revision;this.q<HTMLElement>('[data-admin-announcement-form-error]').hidden=true
    this.q<HTMLElement>('[data-admin-modal-title]').textContent=row?'Edit Announcement':'New Announcement';this.dialog.setAttribute('aria-label',row?'Edit Announcement':'New Announcement');this.q<HTMLElement>('[data-admin-announcement-save-label]').textContent=row?'Update':'Create'
    for(const input of this.form.querySelectorAll<HTMLInputElement|HTMLTextAreaElement>('[name]'))if(stringField(input.name)){input.value=this.model[input.name];if(input.hasAttribute('data-date-time-value'))input.dispatchEvent(new Event('change',{bubbles:true}))}
    this.renderChoices();this.dialog.showModal()
  }
  close(){closeDateTimePickers(this.form);this.dialog.close()}
  private fail(error:unknown){if(!this.element.isConnected||isAbortError(error))return;this.q<HTMLElement>('[data-admin-announcement-form-error]').hidden=false;this.q<HTMLElement>('[data-admin-announcement-form-message]').textContent=typeof error==='string'?error:errorMessage(error)}
  private async submit(){
    if(this.saving)return;if(!this.model.title.trim()){this.fail('Title is required');return}
    let payload:ReturnType<typeof announcementPayload>;try{payload=announcementPayload(this.model)}catch{this.fail('Enter valid start and expiry dates');return}
    const token={},id=this.editId,revision=this.revision;this.saving=token;const save=this.q<HTMLButtonElement>('[data-admin-announcement-save]');save.disabled=true;this.q<HTMLElement>('[data-admin-announcement-spinner]').hidden=false;this.q<HTMLElement>('[data-admin-announcement-form-error]').hidden=true
    try{
      const response=await this.request<AnnouncementAdminCreateResponses[200]>('/api/admin/announcements'+(id?'/'+id:''),{method:id?'PATCH':'POST',body:payload})
      if(this.saving!==token)return
      if(!id&&typeof response.announcement.id==='number'){this.editId=response.announcement.id;this.q<HTMLElement>('[data-admin-announcement-save-label]').textContent='Update'}
      if(this.revision===revision)this.close();await this.load()
    }catch(error){if(this.saving===token)this.fail(error)}finally{if(this.saving===token){this.saving=undefined;save.disabled=false;this.q<HTMLElement>('[data-admin-announcement-spinner]').hidden=true}}
  }
  private renderArchive(){for(const button of this.host.querySelectorAll<HTMLButtonElement>('[data-admin-announcement-archive]')){const armed=Number(button.closest<HTMLElement>('[data-admin-announcement-id]')?.dataset.adminAnnouncementId)===this.archiveId;button.disabled=Boolean(this.archiving);button.className='flex items-center justify-center h-7 rounded-md transition-colors cursor-pointer '+(armed?'px-2 gap-1 text-red-300 bg-red-500/15 ring-1 ring-red-500/40':'w-7 text-gray-500 hover:text-red-400 hover:bg-red-500/[0.08]');button.title=armed?'Click again to archive — it stops showing to users':'Archive';button.setAttribute('aria-pressed',String(armed));requiredElement<HTMLElement>(button,'[data-admin-announcement-confirm]').hidden=!armed}}
  private resetArchive(){this.stopTimer(this.archiveTimer);this.archiveId=null;this.renderArchive()}
  private async archive(id:number){
    if(this.archiving)return
    if(this.archiveId!==id){this.stopTimer(this.archiveTimer);this.archiveId=id;this.renderArchive();this.archiveTimer=this.after(5000,()=>this.resetArchive());return}
    this.resetArchive();const token={};this.archiving=token;this.renderArchive()
    try{await this.request(`/api/admin/announcements/${id}/archive`,{method:'POST'});await this.load()}catch(error){if(this.element.isConnected&&!isAbortError(error))toast(errorMessage(error),'error')}finally{if(this.archiving===token){this.archiving=undefined;this.renderArchive()}}
  }
  async click(event:MouseEvent){
    const target=eventElement(event.target)
    if(isBackdropClick(this.dialog,event)||target?.closest('[data-admin-modal-close]')){this.close();return}
    if(target?.closest('[data-admin-announcement-new]')){this.open();return}
    if(target?.closest('[data-admin-announcement-save]')){await this.submit();return}
    if(target?.closest('[data-admin-announcement-retry]')){await this.load();return}
    const edit=target?.closest('[data-admin-announcement-edit]');if(edit){const row=this.rows.find(row=>row.id===Number(edit.closest<HTMLElement>('[data-admin-announcement-id]')?.dataset.adminAnnouncementId));if(row)this.open(row);return}
    const archive=target?.closest('[data-admin-announcement-archive]');if(archive){await this.archive(Number(archive.closest<HTMLElement>('[data-admin-announcement-id]')?.dataset.adminAnnouncementId));return}
    for(const [attr,key]of [['tier','tier'],['color','color'],['duration-value','duration']] as const){const choice=target?.closest(`[data-announcement-${attr}]`);if(choice){const value=choice.getAttribute('data-announcement-'+attr)??'';if(key==='color'){if(Object.hasOwn(colors,value))this.model.color=value}else if(key==='duration'){const duration=tickerDurations.find(item=>item.value===Number(value));if(duration)this.model.duration=duration.value}else this.model.tier=Number(value);++this.revision;this.renderChoices()}}
  }
  input(event:Event){const input=event.target;if((input instanceof HTMLInputElement||input instanceof HTMLTextAreaElement)&&stringField(input.name)){this.model[input.name]=input.value;++this.revision}}
  protected resumed(){if(this.interrupted||!this.q<HTMLElement>('[data-admin-announcement-error]').hidden)void this.load()}
  hide(){this.interrupted||=Boolean(this.pending)||Boolean(this.saving)||Boolean(this.archiving);this.pending?.abort();this.pending=undefined;this.saving=undefined;this.archiving=undefined;this.q<HTMLButtonElement>('[data-admin-announcement-save]').disabled=false;this.q<HTMLElement>('[data-admin-announcement-spinner]').hidden=true;this.host.removeAttribute('aria-busy');this.resetArchive();this.close();super.hide()}
  disconnect(){this.menu?.dispose();this.menu?.root.remove();this.menu=undefined;super.disconnect()}
}
registerController('admin-announcements',AdminAnnouncementsController)
