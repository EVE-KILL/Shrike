// Small DOM primitives shared by the composer, media picker, and report dialog.
import {escapeHTML as escapeCommentText} from './shared/html'
import {Controller} from '@hotwired/stimulus'
import {registerController} from './controllers/application'
import {eventElement} from './shared/dom'
import {isBackdropClick} from './shared/dialog'
export {escapeCommentText}
export const commentIcon=(name:string):string=>`<svg class="icon" aria-hidden="true" viewBox="0 0 24 24"><use href="/assets/icons.svg#${escapeCommentText(name)}"></use></svg>`
export function cloneCommentTemplate<T extends HTMLElement=HTMLElement>(id:string):T {
  const template=document.getElementById(id)
  if(!(template instanceof HTMLTemplateElement)||!(template.content.firstElementChild instanceof HTMLElement))throw new Error('Comment controls are unavailable')
  return template.content.firstElementChild.cloneNode(true) as T
}
let scrollOwners=0,previousOverflow=''
export class CommentDialogController extends Controller<HTMLDialogElement>{
  private locked=false
  connect(){if(!this.locked){if(scrollOwners++===0){previousOverflow=document.documentElement.style.overflow;document.documentElement.style.overflow='hidden'}this.locked=true}}
  close(){if(this.element.open)this.element.close();this.element.remove()}
  click(event:MouseEvent){if(isBackdropClick(this.element,event)||eventElement(event.target)?.closest('[data-dialog-close]'))this.close()}
  disconnect(){if(this.locked){this.locked=false;if(--scrollOwners===0)document.documentElement.style.overflow=previousOverflow}}
}
registerController('comment-dialog',CommentDialogController)
export function mountCommentDialog(id:string){
  const dialog=cloneCommentTemplate(id)
  if(!(dialog instanceof HTMLDialogElement))throw new Error('Comment dialog is unavailable')
  document.body.append(dialog)
  let disposed=false
  const dispose=()=>{
    if(disposed)return
    disposed=true
    if(dialog.open)dialog.close()
    dialog.remove()
  }
  dialog.dataset.controller=[dialog.dataset.controller,'comment-dialog'].filter(Boolean).join(' ')
  dialog.dataset.action=[dialog.dataset.action,'click->comment-dialog#click close->comment-dialog#close cancel->comment-dialog#close pagehide@window->comment-dialog#close'].filter(Boolean).join(' ')
  dialog.showModal()
  return {dialog,dispose}
}
export type CommentModal=ReturnType<typeof mountCommentDialog>
