import {Controller} from '@hotwired/stimulus'
import {createLoadingState,isPageDestination,observePageNavigations} from './loading-state'
import {registerController} from './controllers/application'
import {eventElement,requiredElement} from './shared/dom'

type BusyTask={source:Element|null;finish:(()=>void)|null}
// Navigation API is optional on browsers without this progressive enhancement.
type NavigationWindow={navigation?:Pick<EventTarget,'addEventListener'|'removeEventListener'>}

export class NavigationProgressController extends Controller<HTMLElement> {
  private highlighted=new Set<Element>()
  private busy=new Map<Element,BusyTask>()
  private lastSource:Element|null=null
  private lastAction=0
  private finishNavigation:(()=>void)|null=null
  private state:ReturnType<typeof createLoadingState<Element>>|undefined
  private observer:MutationObserver|undefined
  private stopNavigations:(()=>void)|undefined
  private nativeNavigation=false

  connect(){
    const bar=requiredElement<HTMLElement>(this.element,'#navigation-progress')
    const status=requiredElement<HTMLElement>(this.element,'#navigation-loading-status')
    this.state=createLoadingState<Element>({render:(visible,sources)=>{
      const next=new Set(sources.filter((source):source is Element=>visible&&Boolean(source?.isConnected)))
      for(const source of this.highlighted)if(!next.has(source))source.classList.remove('navigation-pending')
      for(const source of next)if(!this.highlighted.has(source))source.classList.add('navigation-pending')
      this.highlighted=next;bar.hidden=!visible
      const message=visible?'Loading…':''
      if(status.textContent!==message)status.textContent=message
    }})
    const navigation=(window as NavigationWindow).navigation
    this.nativeNavigation=Boolean(navigation?.addEventListener)
    if(navigation?.addEventListener)this.stopNavigations=observePageNavigations(navigation,
      source=>this.beginNavigation(this.sourceFor(source)||this.recentSource()),()=>this.endNavigation())
    this.observer=new MutationObserver(records=>{
      for(const record of records){
        if(record.type!=='attributes'||record.attributeName!=='aria-busy'||!(record.target instanceof Element))continue
        const node=record.target
        if(node.getAttribute('aria-busy')!=='true')continue
        const source=this.recentSource(),task=this.busy.get(node)
        if(!task)this.busy.set(node,{source,finish:null})
        else if(source&&task.source!==source){
          const finish=task.finish;task.source=source
          task.finish=node.closest('[hidden], [data-loading-silent]')?null:this.state?.start(source)??null
          finish?.()
        }
      }
      this.reconcileBusy()
    })
    this.observer.observe(requiredElement(this.element,'#main'),{
      subtree:true,childList:true,attributes:true,attributeFilter:['aria-busy','hidden'],
    })
  }
  disconnect(){
    this.observer?.disconnect();this.stopNavigations?.();this.stopNavigations=undefined
    this.endNavigation();this.state?.clear();this.state=undefined;this.busy.clear()
    this.lastSource=null;this.lastAction=0
  }
  private recentSource(){return performance.now()-this.lastAction<1500?this.lastSource:null}
  private sourceFor(target:unknown){
    const source=eventElement(target)?.closest('a, button, input, select, summary, [role="link"], [data-admin-user-row]')??null
    return source?.matches('a[href^="/kill/"]')?source.closest('.kill-row')||source:source
  }
  remember(event:Event){this.lastSource=this.sourceFor(event.target);this.lastAction=performance.now()}
  click(event:MouseEvent){
    if(event.button!==0||event.metaKey||event.ctrlKey||event.shiftKey||event.altKey){this.lastSource=null;this.lastAction=0;return}
    this.remember(event)
    if(this.nativeNavigation)return
    const element=eventElement(event.target),link=element?.closest<HTMLAnchorElement>('a[href]'),row=element?.closest<HTMLElement>('[data-admin-user-row]')
    const destination=link?.href||row?.dataset.adminUserRow
    const target=link?.target||document.querySelector<HTMLBaseElement>('base[target]')?.target||'_self'
    if(!destination||target!=='_self'||link?.hasAttribute('download')||event.defaultPrevented||!isPageDestination(destination,location.href))return
    const finish=this.beginNavigation()
    queueMicrotask(()=>{if(event.defaultPrevented)finish()})
  }
  keyboard(event:KeyboardEvent){
    if(event.key==='Enter'||event.key===' ')this.remember(event)
    if(event.key==='Escape')this.endNavigation()
    if(this.nativeNavigation||event.key!=='Enter'||event.defaultPrevented)return
    const row=eventElement(event.target)?.closest<HTMLElement>('[data-admin-user-row]'),destination=row?.dataset.adminUserRow
    if(row&&destination&&isPageDestination(destination,location.href)){
      const finish=this.beginNavigation(row)
      queueMicrotask(()=>{if(event.defaultPrevented)finish()})
    }
  }
  submit(event:SubmitEvent){
    const form=event.target
    if(this.nativeNavigation||!(form instanceof HTMLFormElement)||(form.target&&form.target!=='_self')||event.defaultPrevented)return
    const finish=this.beginNavigation(event.submitter||this.recentSource())
    queueMicrotask(()=>{if(event.defaultPrevented)finish()})
  }
  private beginNavigation(source=this.recentSource()){
    const previous=this.finishNavigation
    // A departing document can suspend timers before the response arrives.
    const finish=this.state?.start(source,{immediate:true})??(()=>{})
    this.finishNavigation=finish;previous?.();return finish
  }
  private endNavigation(){this.finishNavigation?.();this.finishNavigation=null}
  private reconcileBusy(){
    for(const [node,task]of this.busy){
      if(!node.isConnected||node.getAttribute('aria-busy')!=='true'){task.finish?.();this.busy.delete(node)}
      else if(node.closest('[hidden], [data-loading-silent]')){task.finish?.();task.finish=null}
      else if(!task.finish)task.finish=this.state?.start(task.source)??null
    }
  }
  pageHide(){this.endNavigation()}
  pageShow(){this.endNavigation();this.reconcileBusy()}
}
registerController('navigation-progress',NavigationProgressController)
