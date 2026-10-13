type LoadingOptions<Source,Timer>={
  delay?:number
  setTimer?:(callback:()=>void,delay:number)=>Timer
  clearTimer?:(timer:Timer)=>void
  render:(visible:boolean,sources:(Source|null)[])=>void
}

// A delayed, reference-counted indicator. Overlapping work shares one bar.
export function createLoadingState<Source=unknown,Timer=ReturnType<typeof setTimeout>>({
  delay=150,setTimer=setTimeout as unknown as (callback:()=>void,delay:number)=>Timer,
  clearTimer=clearTimeout as unknown as (timer:Timer)=>void,render,
}:LoadingOptions<Source,Timer>){
  const pending=new Set<{source:Source|null}>()
  let timer:Timer|null=null,visible=false
  const update=()=>{
    if(!pending.size){
      if(timer!==null)clearTimer(timer)
      timer=null
      if(visible){visible=false;render(false,[])}
    }else if(visible)render(true,[...pending].map(task=>task.source))
    else if(timer===null)timer=setTimer(()=>{
      timer=null
      if(pending.size){visible=true;render(true,[...pending].map(task=>task.source))}
    },delay)
  }
  return {
    start(source:Source|null=null,{immediate=false}:{immediate?:boolean}={}){
      const task={source};pending.add(task)
      if(immediate&&!visible){
        if(timer!==null)clearTimer(timer)
        timer=null;visible=true;render(true,[...pending].map(task=>task.source))
      }else update()
      return ()=>{if(pending.delete(task))update()}
    },
    clear(){pending.clear();update()},
  }
}

export function isPageDestination(destination:string,current:string):boolean{
  let url:URL
  try{url=new URL(destination,current)}catch{return false}
  if(!['http:','https:'].includes(url.protocol))return false
  const previous=new URL(current)
  return !(url.origin===previous.origin&&url.pathname===previous.pathname&&
    url.search===previous.search&&(url.hash||previous.hash))
}

type NavigateEvent=Event&{
  destination:{sameDocument:boolean;url:string};hashChange:boolean
  downloadRequest:string|null;signal:AbortSignal;sourceElement:unknown
}
export function observePageNavigations(navigation:Pick<EventTarget,'addEventListener'|'removeEventListener'>,
  begin:(source:unknown)=>()=>void,end:()=>void){
  const navigate=(original:Event)=>{
    const event=original as NavigateEvent
    if(event.destination.sameDocument||event.hashChange||event.downloadRequest!=null||
      !/^https?:/.test(event.destination.url)||event.defaultPrevented||event.signal.aborted)return
    const finish=begin(event.sourceElement)
    event.signal.addEventListener('abort',finish,{once:true})
    queueMicrotask(()=>{if(event.defaultPrevented)finish()})
  }
  navigation.addEventListener('navigate',navigate)
  navigation.addEventListener('navigatesuccess',end)
  navigation.addEventListener('navigateerror',end)
  return ()=>{
    navigation.removeEventListener('navigate',navigate)
    navigation.removeEventListener('navigatesuccess',end)
    navigation.removeEventListener('navigateerror',end)
  }
}
