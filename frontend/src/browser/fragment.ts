// Retain focus without changing scroll when replacing server-rendered HTML.
function keepFragmentFocus(current:Element,update:()=>Element){
  const focused=current.ownerDocument.activeElement
  let selector:string|undefined,index=-1
  if(focused&&current.contains(focused)){
    const attribute=['data-focus-key','id','aria-label','name'].find(name=>focused.getAttribute(name))
    if(attribute){
      selector=`${focused.localName}[${attribute}="${CSS.escape(focused.getAttribute(attribute)!)}"]`
      index=Array.from(current.querySelectorAll(selector)).indexOf(focused)
    }
  }
  const next=update()
  if(selector&&index>=0){
    const control=next.querySelectorAll<HTMLElement&{disabled?:boolean}>(selector)[index]
    if(control?.focus&&!control.disabled&&control.getClientRects().length)control.focus({preventScroll:true})
  }
}
export function replaceFragment(current:Element,next:Element,prepare?:()=>void):void{
  keepFragmentFocus(current,()=>{prepare?.();current.replaceWith(next);return next})
}
export function replaceFragmentChildren(current:Element,...nodes:(Node|string)[]):void{
  keepFragmentFocus(current,()=>{current.replaceChildren(...nodes);return current})
}
