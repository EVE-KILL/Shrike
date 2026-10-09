import {RequestComponent} from './request-component'
import {registerController} from './application'
import {formInput} from '../shared/dom'
import {isAbortError} from '../shared/json'
import {uniqueRoamPilots,roamTimeError,roamBody,sameRoamDraft,readRoamDraft,readRoamSavedID,isRoamID,type RoamDraft,type RoamSaved} from '../shared/roam-report'
import '../date-time-picker'
import {iconHref} from '../icon-hrefs'

export function roamFormSnapshot(form:HTMLFormElement):RoamDraft{const names=form.querySelector<HTMLTextAreaElement>('[data-roam-names]');if(!names)throw new Error('Missing roam pilot input');return {names:names.value,start:formInput(form,'start_time').value,end:formInput(form,'end_time').value}}
export class RoamFormController extends RequestComponent{
 private original:RoamDraft|undefined
 private working=false
 private write:AbortController|undefined
 private uncertain=false
 private alive=false
 private clipboardGeneration=0
 private revision=0
 private get form(){return this.q<HTMLFormElement>('[data-roam-form]')}
 private get edit(){return this.form.dataset.edit==='true'}
 private get id(){return this.element.closest<HTMLElement>('[data-roam-report]')?.dataset.roamId}
 connect(){
  this.alive=true
  if(!this.original){
   this.original=roamFormSnapshot(this.form)
   if(!this.edit){try{const saved=readRoamDraft(JSON.parse(sessionStorage.getItem('ek_roam_form')??'null'));if(saved)this.restore(saved)}catch{/* Unavailable storage does not prevent building a report. */}}
  }
  this.validate()
 }
 private restore(draft:RoamDraft){this.q<HTMLTextAreaElement>('[data-roam-names]').value=draft.names;for(const [name,value]of [['start_time',draft.start],['end_time',draft.end]]){const input=formInput(this.form,name);input.value=value;input.dispatchEvent(new Event('change',{bubbles:true}))}}
 private icon(name:string){const use=this.element.querySelector<SVGUseElement>('[data-roam-submit-icon] use');if(use)use.setAttribute('href',iconHref(name))}
 private validate(){const draft=roamFormSnapshot(this.form),count=uniqueRoamPilots(draft.names).length,message=roamTimeError(draft);this.q('[data-roam-pilot-count]').textContent=`${count} unique ${count===1?'pilot':'pilots'} · up to 256`;this.q<HTMLElement>('[data-roam-pilot-error]').hidden=count<=256;const warning=this.q<HTMLElement>('[data-roam-time-error]');warning.hidden=!message;warning.textContent=message;this.q<HTMLButtonElement>('[data-roam-submit]').disabled=!count||count>256||Boolean(message)||this.working||this.uncertain;if(!this.edit){try{sessionStorage.setItem('ek_roam_form',JSON.stringify(draft))}catch{/* Keep the visible draft. */}}}
 change(){this.revision++;this.validate()}
 reset(){this.revision++;this.clipboardGeneration++;queueMicrotask(()=>{if(!this.alive||!this.element.isConnected||!this.original)return;this.restore(this.original);this.error('[data-roam-form-error]','');this.validate()})}
 async paste(){const generation=++this.clipboardGeneration,revision=this.revision;try{const text=await navigator.clipboard.readText();if(!this.alive||!this.element.isConnected||generation!==this.clipboardGeneration||revision!==this.revision)return;this.q<HTMLTextAreaElement>('[data-roam-names]').value=text;this.error('[data-roam-form-error]','');this.change()}catch{if(this.alive&&generation===this.clipboardGeneration)this.error('[data-roam-form-error]','Clipboard access failed. Paste the fleet list into the text box instead.')}}
 private savedDraftNotice(id:string){const box=this.q<HTMLElement>('[data-roam-form-status]'),link=document.createElement('a');link.href=`/tools/roam-report/${id}`;link.className='underline';link.textContent='Open saved report';box.replaceChildren('Report saved. You have newer unsaved changes. ',link);box.hidden=false}
 async submit(event:SubmitEvent){
  event.preventDefault();this.validate();if(this.q<HTMLButtonElement>('[data-roam-submit]').disabled)return;const id=this.id;if(this.edit&&!isRoamID(id)){this.error('[data-roam-form-error]','Report link is unavailable. Reload this page.');return}
  const submitted=roamFormSnapshot(this.form),write=new AbortController();this.write=write;this.working=true;this.error('[data-roam-form-error]','');this.q<HTMLElement>('[data-roam-form-status]').hidden=true;this.validate();this.form.classList.add('roam-form-working');this.icon('loader-2');this.q('[data-roam-submit-label]').textContent=this.edit?'Rebuilding report…':'Building report…'
  try{
   const result=await this.request<unknown>(`/api/tools/roam-report${this.edit?'/'+id:''}`,{method:this.edit?'PUT':'POST',body:roamBody(submitted),signal:write.signal},this.edit?'Could not update the report. Please try again.':'Could not create the report. Please try again.');if(this.write!==write||!this.alive)return
   this.write=undefined;let savedID:string;try{savedID=readRoamSavedID(result);if(this.edit&&savedID!==id)throw new Error('The updated report link did not match. Reload before submitting again.')}catch(error){this.uncertain=true;throw error}
   this.original=submitted;const hasNewerDraft=!sameRoamDraft(submitted,roamFormSnapshot(this.form));if(hasNewerDraft)this.savedDraftNotice(savedID)
   if(this.edit){const detail:RoamSaved={id:savedID,hasNewerDraft};this.dispatch('saved',{detail});await detail.refresh}else if(!hasNewerDraft)location.assign(`/tools/roam-report/${savedID}`)
  }catch(error){if(!isAbortError(error)&&this.alive)this.error('[data-roam-form-error]',error)}
  finally{if(this.alive){this.write=undefined;this.working=false;this.form.classList.remove('roam-form-working');this.icon(this.edit?'save':'chart-no-axes-combined');this.q('[data-roam-submit-label]').textContent=this.edit?'Save changes':'Create Roam Report';this.validate()}}
 }
 hide(){this.alive=false;this.clipboardGeneration++;if(this.write){this.uncertain=true;this.error('[data-roam-form-error]','Saving was interrupted and may have completed. Reload before submitting again.')}this.write?.abort();this.write=undefined;this.working=false;this.form.classList.remove('roam-form-working');this.icon(this.edit?'save':'chart-no-axes-combined');this.q('[data-roam-submit-label]').textContent=this.edit?'Save changes':'Create Roam Report';this.validate();super.hide()}
 show(event:PageTransitionEvent){if(event.persisted){this.alive=true;this.validate()}}
}
registerController('roam-form',RoamFormController)
