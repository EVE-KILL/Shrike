import {AdminSectionComponent} from './controllers/admin/base'
import {registerController} from './controllers/application'
import {riverStamp,riverJSON,riverStateClass,type RiverJob} from './admin-data'
import {readQuerySeed} from './shared/query-seed'
import {readRiverQueues,readRiverJobs} from './shared/admin-runtime-seeds'
import {eventElement} from './shared/dom'
import {errorMessage,isAbortError} from './shared/json'
import {isBackdropClick} from './shared/dialog'
import {toast} from './toast'
import type {AdminRiverJobActionData,AdminRiverQueueActionData,AdminRiverQueueClearData,AdminRiverQueueClearResponses} from '../../shared/api/types.gen'

type Part='queues'|'jobs'
type JobAction=AdminRiverJobActionData['body']['action']
type QueueAction=AdminRiverQueueActionData['body']['action']|'clear'
const isPart=(value:unknown):value is Part=>value==='queues'||value==='jobs'
const isQueueAction=(value:unknown):value is QueueAction=>value==='pause'||value==='resume'||value==='clear'
const isJobAction=(value:unknown):value is JobAction=>value==='cancel'||value==='retry'||value==='delete'
export class AdminRiverController extends AdminSectionComponent {
  private query:Record<string,string>|undefined
  private pending=new Map<Part,AbortController>()
  private interrupted=new Set<Part>()
  private queues:Array<{name:string}>=[]
  private jobs:RiverJob[]=[]
  private busy:object|undefined
  private clearID:string|null=null
  private deleteID:string|null=null
  private clearTimer:ReturnType<typeof setTimeout>|undefined
  private deleteTimer:ReturnType<typeof setTimeout>|undefined
  private get dialog(){return this.q<HTMLDialogElement>('[data-river-dialog]')}
  private host(part:Part){return this.q<HTMLElement>(`[data-admin-river-${part}]`)}
  connect(){super.connect();this.query??=readQuerySeed(this.element.dataset.query);this.rows('queues');this.rows('jobs');this.render()}
  private rows(part:Part){const source=this.host(part).querySelector<HTMLElement>('[data-river-result]')?.dataset.initial;if(part==='queues')this.queues=readRiverQueues(source);else this.jobs=readRiverJobs(source)}
  private render(){
    for(const stamp of this.all<HTMLElement>('[data-river-stamp]'))stamp.textContent=riverStamp(stamp.dataset.riverStamp)
    for(const control of this.all<HTMLButtonElement>('[data-river-queue-action],[data-river-job-action]'))control.disabled=Boolean(this.busy)
    for(const button of this.all<HTMLButtonElement>('[data-river-queue-action="clear"]'))button.textContent=button.closest<HTMLElement>('[data-admin-river-queue]')?.dataset.adminRiverQueue===this.clearID?'Confirm clear':'Clear history'
    for(const button of this.all<HTMLButtonElement>('[data-river-job-action="delete"]'))button.textContent=button.closest<HTMLElement>('[data-admin-river-job]')?.dataset.adminRiverJob===this.deleteID?'Confirm':'Delete'
    this.q<SVGElement>('[data-river-spinner] svg').classList.toggle('animate-spin',this.pending.size>0)
  }
  private failure(part:Part,error:unknown=''){
    if(isAbortError(error)||!this.element.isConnected)return
    const message=typeof error==='string'?error:errorMessage(error),node=this.q<HTMLElement>(`[data-river-error="${part}"]`);node.hidden=!message
    if(message){node.replaceChildren(document.createTextNode(message+' '));const retry=document.createElement('button');retry.type='button';retry.dataset.riverRetry=part;retry.className='text-blue-400';retry.textContent='Retry';node.append(retry)}
  }
  private choices(){const menu=this.q<HTMLSelectElement>('[data-river-filter="queue"]');menu.replaceChildren(new Option('All queues',''),...this.queues.map(queue=>new Option(queue.name,queue.name)));menu.value=this.query?.queue??''}
  private async load(part:Part){
    this.pending.get(part)?.abort();const controller=new AbortController();this.pending.set(part,controller);this.interrupted.delete(part);const host=this.host(part);host.setAttribute('aria-busy','true');this.failure(part);this.render()
    try{const text=await this.html(`/_fragments/admin/river/${part}`+(part==='jobs'?'?'+new URLSearchParams(this.query):''),`Could not load River ${part}.`,{signal:controller.signal}),next=new DOMParser().parseFromString(text,'text/html').querySelector<HTMLElement>(`[data-river-result="${part}"]`);if(!next||next.dataset.failed==='true')throw new Error(`Could not load River ${part}.`);if(this.pending.get(part)!==controller)return;host.replaceChildren(document.importNode(next,true));this.rows(part);if(part==='queues')this.choices();this.render()}
    catch(error){if(this.pending.get(part)===controller)this.failure(part,error)}finally{if(this.pending.get(part)===controller){this.pending.delete(part);host.removeAttribute('aria-busy');this.render()}}
  }
  private reload(){return Promise.all([this.load('queues'),this.load('jobs')])}
  private inspect(id:string){
    const job=this.jobs.find(row=>String(row.id)===id);if(!job)return
    const values:Record<string,string|number>={title:`Job ${job.id}`,state:job.state,queue:`${job.queue} / ${job.kind}`,attempt:`${job.attempt}/${job.max_attempts}`,priority:job.priority,scheduled:riverStamp(job.scheduled_at),finalized:riverStamp(job.finalized_at),args:riverJSON(job.args),output:job.output==null?'No durable output recorded':riverJSON(job.output),errors:riverJSON(job.errors),workers:job.attempted_by.join(', ')||'Not attempted yet'}
    for(const field of this.dialog.querySelectorAll<HTMLElement>('[data-river-field]'))field.textContent=String(values[field.dataset.riverField??'']??'')
    this.q<HTMLElement>('[data-river-field="state"]').className='rounded px-2 py-0.5 text-fine font-semibold uppercase bg-white/[0.05] '+riverStateClass(job.state);this.q<HTMLElement>('[data-river-errors]').hidden=!job.errors.length;this.dialog.setAttribute('aria-label',`River job ${job.id}`);this.dialog.showModal()
  }
  private async action(id:string,operation:QueueAction|JobAction,queue:boolean){
    if(this.busy)return
    if(operation==='clear'&&this.clearID!==id){this.stopTimer(this.clearTimer);this.clearID=id;this.clearTimer=this.after(5000,()=>{this.clearID=null;this.render()});this.render();return}
    if(operation==='delete'&&this.deleteID!==id){this.stopTimer(this.deleteTimer);this.deleteID=id;this.deleteTimer=this.after(5000,()=>{this.deleteID=null;this.render()});this.render();return}
    if(operation==='clear'){this.stopTimer(this.clearTimer);this.clearID=null}if(operation==='delete'){this.stopTimer(this.deleteTimer);this.deleteID=null}
    const token={};this.busy=token;this.render()
    try{
      if(queue&&operation==='clear'){const body:AdminRiverQueueClearData['body']={states:['completed','cancelled','discarded'],limit:10000};const result=await this.request<AdminRiverQueueClearResponses[200]>(`/api/admin/river/queues/${encodeURIComponent(id)}/clear`,{method:'POST',body});toast(`Deleted ${result.deleted} finalized jobs from ${id}`,'success')}
      else if(queue&&(operation==='pause'||operation==='resume')){const body:AdminRiverQueueActionData['body']={action:operation};await this.request(`/api/admin/river/queues/${encodeURIComponent(id)}/action`,{method:'POST',body})}
      else if(!queue&&isJobAction(operation)){const body:AdminRiverJobActionData['body']={action:operation};await this.request(`/api/admin/river/jobs/${encodeURIComponent(id)}/action`,{method:'POST',body});this.close()}
      if(this.busy===token)await this.reload()
    }catch(error){if(!isAbortError(error)&&this.element.isConnected)toast(errorMessage(error),'error')}finally{if(this.busy===token){this.busy=undefined;this.render()}}
  }
  change(event:Event){const select=event.target,key=select instanceof HTMLSelectElement?select.dataset.riverFilter:undefined;if(select instanceof HTMLSelectElement&&(key==='queue'||key==='state')&&this.query){this.query[key]=select.value;delete this.query.before_id;void this.load('jobs')}}
  close(){this.dialog.close()}
  async click(event:MouseEvent){
    const target=eventElement(event.target),retry=target?.closest<HTMLElement>('[data-river-retry]')?.dataset.riverRetry
    if(isPart(retry)){await this.load(retry);return}if(target?.closest('[data-river-refresh]')){await this.reload();return}
    if(isBackdropClick(this.dialog,event)||target?.closest('[data-river-close]')){this.close();return}
    const older=target?.closest<HTMLElement>('[data-river-older]')?.dataset.riverOlder;if(older&&this.query){this.query.before_id=older;await this.load('jobs');return}
    const inspect=target?.closest('[data-river-inspect]');if(inspect){this.inspect(inspect.closest<HTMLElement>('[data-admin-river-job]')?.dataset.adminRiverJob??'');return}
    const queue=target?.closest<HTMLElement>('[data-river-queue-action]'),queueID=queue?.closest<HTMLElement>('[data-admin-river-queue]')?.dataset.adminRiverQueue,operation=queue?.dataset.riverQueueAction
    if(queueID&&isQueueAction(operation)){await this.action(queueID,operation,true);return}
    const job=target?.closest<HTMLElement>('[data-river-job-action]'),id=job?.closest<HTMLElement>('[data-admin-river-job]')?.dataset.adminRiverJob,action=job?.dataset.riverJobAction
    if(id&&isJobAction(action))await this.action(id,action,false)
  }
  protected resumed(){for(const part of ['queues','jobs'] as const)if(this.interrupted.has(part)||!this.q<HTMLElement>(`[data-river-error="${part}"]`).hidden)void this.load(part)}
  hide(){for(const [part,request] of this.pending){this.interrupted.add(part);request.abort();this.host(part).removeAttribute('aria-busy')}if(this.busy){this.interrupted.add('queues');this.interrupted.add('jobs')}this.pending.clear();this.busy=undefined;this.close();this.stopTimer(this.clearTimer);this.stopTimer(this.deleteTimer);this.clearID=null;this.deleteID=null;super.hide();this.render()}
}
registerController('admin-river',AdminRiverController)
