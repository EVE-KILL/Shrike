import type {SdeData} from '../../dogma/src/metadata'
import type {JSONRequestOptions} from '../browser/browser-request'
export type EditorEvent<K extends keyof GlobalEventHandlersEventMap,T extends EventTarget> = GlobalEventHandlersEventMap[K] & {currentTarget:T}
// Stimulus owner supplies transport/timers and registers actions for the existing
// editor render/runtime functions. The runtime owns no global event listeners.
export interface EditorScope {
 on<T extends HTMLElement|Document|Window,K extends keyof GlobalEventHandlersEventMap>(target:T,event:K,handler:(event:EditorEvent<K,T>)=>unknown,capture?:boolean):void
 off<T extends HTMLElement|Document|Window,K extends keyof GlobalEventHandlersEventMap>(target:T,event:K,handler:(event:EditorEvent<K,T>)=>unknown):void
 later(callback:()=>void,delay:number):ReturnType<typeof setTimeout>
 clear(timer:ReturnType<typeof setTimeout>|undefined):void
 observe(target:HTMLElement,callback:()=>void):void
 active():boolean
 epoch():number
 json(path:string,options?:JSONRequestOptions):Promise<unknown>
 text(path:string,options?:RequestInit):Promise<string>
 sde:SdeData
}
