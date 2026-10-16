// Manual activation is appropriate for tabs that fetch a new panel. Enter and
// Space retain native button activation; moving focus does not start a request.
export function moveTabFocus(event:KeyboardEvent, activate = false):void {
  if(event.altKey || event.ctrlKey || event.metaKey || !(event.target instanceof Element)) return
  const current = event.target.closest<HTMLElement>('[role="tab"]')
  const list = current?.closest<HTMLElement>('[role="tablist"]')
  if(!current || !list) return
  const tabs = [...list.querySelectorAll<HTMLButtonElement>('[role="tab"]')].filter(tab=>!tab.disabled && !tab.hidden && tab.closest('[role="tablist"]') === list)
  const index = tabs.indexOf(current as HTMLButtonElement)
  if(index < 0) return
  const vertical = list.getAttribute('aria-orientation') === 'vertical'
  let next:number
  if(event.key === 'Home') next = 0
  else if(event.key === 'End') next = tabs.length-1
  else if(event.key === (vertical ? 'ArrowDown' : 'ArrowRight')) next = (index+1)%tabs.length
  else if(event.key === (vertical ? 'ArrowUp' : 'ArrowLeft')) next = (index+tabs.length-1)%tabs.length
  else return
  event.preventDefault()
  tabs.forEach((tab,i)=>tab.tabIndex = i === next ? 0 : -1)
  tabs[next]!.focus()
  if(activate) tabs[next]!.click()
}
