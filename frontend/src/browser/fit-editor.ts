import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {loadSde} from './dogma'
import {createEditorRuntime} from '../fit/editor-runtime'
import type {EditorScope,EditorEvent} from '../fit/editor-scope'
import type {SdeData} from '../../dogma/src/metadata'
import {eventElement} from './shared/dom'
import {isAbortError} from './shared/json'

type ActionBinding = {target:HTMLElement|Document|Window;node:HTMLElement;event:string;capture:boolean;token:string;source:unknown;run:(event:Event)=>unknown}
export class FitEditorController extends RequestComponent {
 private alive = false
 private generation = 0
 private runtime:ReturnType<typeof createEditorRuntime>|undefined
 private loading:Promise<void>|undefined
 private bindings:ActionBinding[] = []
 private ownedTimers = new Set<ReturnType<typeof setTimeout>>()
 private observers:Array<{node:HTMLElement;callback:()=>void;observer:ResizeObserver}> = []
 connect() {this.alive = true;if(!this.runtime) void this.load()}
 loadSde():Promise<SdeData> {return loadSde()}
 private async load() {
  if(this.loading) return this.loading
  const generation = this.generation
  this.loading = (async()=>{
   try {
    const sde = await this.loadSde();if(!this.alive || generation !== this.generation) return
    const scope:EditorScope = {
     sde,on:(target,event,handler,capture)=>this.bind(target,event,handler,capture),off:(target,event,handler)=>this.unbind(target,event,handler),
     later:(callback,delay)=>{const timer=this.after(delay,()=>{this.ownedTimers.delete(timer);if(this.alive) callback()});this.ownedTimers.add(timer);return timer},clear:timer=>{this.stopTimer(timer);if(timer!==undefined)this.ownedTimers.delete(timer)},
     observe:(node,callback)=>{const observer = new ResizeObserver(()=>{if(this.alive) callback()});observer.observe(node);this.observers.push({node,callback,observer})},
     active:()=>this.alive && this.element.isConnected,epoch:()=>this.generation,
     json:(path,options)=>this.request<unknown>(path,options),text:(path,options)=>this.text(path,'Could not load fitting data',options),
    }
    this.runtime = createEditorRuntime(this.element,scope);await this.runtime.ready
   }catch(error) {if(this.alive && generation === this.generation && !isAbortError(error)) {console.error('Could not initialize fitting editor',error);this.q('[data-fit-status]').textContent = 'Fitting data unavailable';this.q('[data-fit-catalog]').textContent = 'The fitting data could not be loaded. Please refresh the page.'}}
   finally {if(generation === this.generation) this.loading = undefined}
  })()
  return this.loading
 }
 // Preserve the existing callback bodies, but let Stimulus own every listener.
 // Dispatch is narrowed by the registered DOM event type and current target.
 private bind<T extends HTMLElement|Document|Window,K extends keyof GlobalEventHandlersEventMap>(target:T,event:K,handler:(event:EditorEvent<K,T>)=>unknown,capture = false) {
  const identity:EventTarget = target
  const node = target instanceof HTMLElement ? target : this.element,suffix = identity === document ? '@document' : identity === window ? '@window' : '',method = capture ? 'handleCapture' : 'handle',token = `${event}${suffix}->fit-editor#${method}${capture ? ':capture' : ''}`
  this.bindings.push({target,node,event,capture,token,source:handler,run:value=>handler(value as EditorEvent<K,T>)})
  const tokens = new Set((node.dataset.action ?? '').split(/\s+/).filter(Boolean));tokens.add(token);node.dataset.action = [...tokens].join(' ')
 }
 private unbind(target:EventTarget,event:string,handler:unknown) {
  const removed = this.bindings.filter(binding=>binding.target === target && binding.event === event && binding.source === handler)
  this.bindings = this.bindings.filter(binding=>!removed.includes(binding))
  for(const binding of removed) if(!this.bindings.some(other=>other.node === binding.node && other.token === binding.token)) binding.node.dataset.action = (binding.node.dataset.action ?? '').split(/\s+/).filter(token=>token !== binding.token).join(' ')
 }
 private dispatchOwned(event:Event,capture:boolean) {
  if(!this.alive) return
  const generation = this.generation
  for(const binding of [...this.bindings]) if(binding.capture === capture && binding.event === event.type && binding.target === event.currentTarget) {
   try {const result = binding.run(event);if(result instanceof Promise) void result.catch(error=>{if(this.alive && generation === this.generation && !isAbortError(error)) this.q('[data-fit-status]').textContent = 'Action failed'})}
   catch(error) {if(!isAbortError(error)) this.q('[data-fit-status]').textContent = 'Action failed'}
  }
 }
 handle(event:Event) {this.dispatchOwned(event,false)}
 handleCapture(event:Event) {this.dispatchOwned(event,true)}
 private getRuntime() {return this.runtime}
 async bootAction(event:MouseEvent) {
  if(this.runtime || !this.alive) return
  const action = eventElement(event.target)?.closest<HTMLElement>('[data-fit-action]')?.dataset.fitAction
  if(!action || !['paste','new','expand'].includes(action)) return
  const generation = this.generation,input = action === 'paste' ? navigator.clipboard.readText().catch(()=> '') : undefined
  await this.load();const text = input ? await input : undefined
  const runtime = this.getRuntime();if(!this.alive || generation !== this.generation || !runtime) return
  if(action === 'paste') await runtime.importText(text ?? '');else await runtime.action(action)
 }
 hide() {this.alive = false;this.generation++;this.loading = undefined;this.runtime?.hide();for(const item of this.observers) item.observer.disconnect();for(const timer of this.ownedTimers)this.stopTimer(timer);this.ownedTimers.clear();super.hide()}
 show(event:PageTransitionEvent) {if(!event.persisted) return;this.alive = true;for(const item of this.observers) item.observer.observe(item.node);if(this.runtime) this.runtime.show();else void this.load()}
 disconnect() {
  // Fullscreen moves the same editor within this document. Stimulus reconnects
  // it, so retain the workspace rather than starting a second editor.
  if(this.element.isConnected) return
  this.hide();this.runtime?.destroy();this.runtime = undefined
  for(const binding of [...this.bindings]) this.unbind(binding.target,binding.event,binding.source)
  this.observers = []
 }
}
registerController('fit-editor',FitEditorController)
