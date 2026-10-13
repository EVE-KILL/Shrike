export function isBackdropClick(dialog:HTMLDialogElement,event:MouseEvent):boolean {
  if(event.target!==dialog)return false
  const box=dialog.getBoundingClientRect()
  return event.clientX<box.left||event.clientX>box.right||event.clientY<box.top||event.clientY>box.bottom
}
