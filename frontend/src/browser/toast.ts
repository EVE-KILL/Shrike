// Reusable feedback with a single document-level live region.
let root:HTMLDivElement|undefined
export function toast(message:string,type:'info'|'success'|'error'='info',duration=4000):HTMLButtonElement {
  if(!root?.isConnected){
    const scope=document.createElement('div');scope.className='go-utility-page'
    root=document.createElement('div');root.className='fixed bottom-4 left-1/2 -translate-x-1/2 z-[10000] flex flex-col gap-2 pointer-events-none'
    root.setAttribute('aria-live','polite');scope.append(root);document.body.append(scope)
  }
  const item=document.createElement('button');item.type='button'
  item.className='pointer-events-auto flex items-center gap-2 px-4 py-2.5 rounded-lg border backdrop-blur-md text-sm shadow-xl '+(type==='success'?'bg-green-500/10 border-green-400/30 text-green-300':type==='error'?'bg-red-500/10 border-red-400/30 text-red-300':'bg-blue-500/10 border-blue-400/30 text-blue-300')
  const icon=document.createElementNS('http://www.w3.org/2000/svg','svg')
  icon.setAttribute('class','icon text-base flex-shrink-0');icon.setAttribute('width','16');icon.setAttribute('height','16');icon.setAttribute('aria-hidden','true')
  const use=document.createElementNS('http://www.w3.org/2000/svg','use')
  use.setAttribute('href','/assets/icons.svg#'+(type==='success'?'circle-check':type==='error'?'circle-alert':'info'));icon.append(use)
  const text=document.createElement('span');text.textContent=message;item.append(icon,text)
  item.addEventListener('click',()=>item.remove());root.append(item)
  if(duration>0)setTimeout(()=>item.remove(),duration)
  return item
}
