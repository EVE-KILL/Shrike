import {EMBED_CONFIG,commentEmbedPreview,insertCommentText,type EmbedKind} from './comment-input-data'
import {cloneCommentTemplate,mountCommentDialog,commentIcon,type CommentModal} from './comment-controls'
import {openKlipyPicker,type KlipyState} from './klipy-picker'
import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {eventElement,requiredElement} from './shared/dom'
import {errorMessage,isAbortError} from './shared/json'
import type {CommentsPreviewData,CommentsPreviewResponses} from '../../shared/api/types.gen'

const wrappers={bold:['**','**'],italic:['_','_'],strike:['~~','~~'],code:['`','`'],quote:['\n> ',''],link:['[','](https://)']} as const
const isWrapper=(value:string):value is keyof typeof wrappers=>Object.hasOwn(wrappers,value)
const isEmbed=(value:string):value is EmbedKind=>value==='image'||value==='youtube'||value==='imgur'||value==='giphy'||value==='tenor'
export interface CommentInputOptions {initial?:string;placeholder?:string;submitLabel?:string;onSubmit?:(body:string,signal:AbortSignal)=>void|Promise<void>;onCancel?:()=>void}
type Configuration={options:CommentInputOptions;active:boolean}
const configurations=new WeakMap<HTMLElement,Configuration>(),controllers=new WeakMap<HTMLElement,CommentInputController>()
interface EmbedConfiguration {kind:EmbedKind;insert:(markdown:string)=>void;modal:CommentModal}
const embedConfigurations=new WeakMap<HTMLElement,EmbedConfiguration>()

export class CommentEmbedController extends RequestComponent {
  private config:EmbedConfiguration|undefined
  private previous:string|null=null
  private get input(){return this.q<HTMLInputElement>('[data-embed-url]')}
  connect(){
    this.config=embedConfigurations.get(this.element);if(!this.config)return
    const config=EMBED_CONFIG[this.config.kind];this.element.setAttribute('aria-label',config.title);this.q<HTMLElement>('[data-embed-title]').textContent=config.title
    const icon=this.q<HTMLElement>('[data-embed-icon]');icon.innerHTML=commentIcon(config.icon.split(':')[1]??'image');icon.className=config.accentColor;this.input.placeholder=config.placeholder;this.validate();this.input.focus()
  }
  validate(){
    if(!this.config)return
    const url=this.input.value.trim(),message=url?EMBED_CONFIG[this.config.kind].validate(url):null,error=this.q<HTMLElement>('[data-embed-error]');error.textContent=message??'';error.hidden=!message;this.q<HTMLButtonElement>('[data-embed-insert]').disabled=!url||Boolean(message)
    const source=commentEmbedPreview(this.config.kind,url),thumbnail=this.q<HTMLElement>('[data-embed-preview]'),image=this.q<HTMLImageElement>('[data-embed-preview] img');thumbnail.hidden=!source;this.q<HTMLElement>('[data-embed-hint]').hidden=!url||Boolean(message)||Boolean(source)
    if(source&&source!==this.previous){image.style.display='';image.src=source}if(!source)image.removeAttribute('src');this.previous=source
  }
  imageError(event:Event){if(event.target instanceof HTMLImageElement)event.target.style.display='none'}
  submit(event:SubmitEvent){
    event.preventDefault();if(!this.config)return
    const url=this.input.value.trim(),config=EMBED_CONFIG[this.config.kind],message=url?config.validate(url):'URL required'
    if(message){this.error('[data-embed-error]',message);return}
    this.config.modal.dispose();this.config.insert(config.toMarkdown(url))
  }
}
registerController('comment-embed',CommentEmbedController)

export class CommentInputController extends RequestComponent {
  private config:Configuration|undefined
  private selection={start:0,end:0}
  private tab:'write'|'preview'='write'
  private previewRequest:AbortController|undefined
  private posting:AbortController|undefined
  private revision=0
  private modal:CommentModal|undefined
  private gifState:KlipyState={}
  private get form(){return this.q<HTMLFormElement>('[data-comment-input]')}
  private get textarea(){return this.q<HTMLTextAreaElement>('[data-comment-body]')}
  private get preview(){return this.q<HTMLElement>('[data-comment-preview]')}
  private get toolbar(){return this.q<HTMLElement>('[data-comment-toolbar]')}
  private get submitButton(){return this.q<HTMLButtonElement>('[data-comment-submit]')}
  connect(){this.config=configurations.get(this.element);controllers.set(this.element,this);this.update()}
  private fail(error:unknown=''){if(isAbortError(error)||!this.config?.active)return;const message=typeof error==='string'?error:errorMessage(error,'Could not save comment');this.q<HTMLElement>('[data-comment-error]').hidden=!message;this.q<HTMLElement>('[data-comment-error-text]').textContent=message}
  private update(){
    const tooLong=this.textarea.value.length>2000,count=this.q<HTMLElement>('[data-comment-count]');count.textContent=`${this.textarea.value.length} / 2000`;count.className=tooLong?'text-red-400':'text-gray-600';this.textarea.classList.toggle('text-red-400',tooLong)
    this.submitButton.disabled=Boolean(this.posting)||tooLong||this.textarea.value.trim().length<2;this.submitButton.textContent=this.posting?'Posting…':this.config?.options.submitLabel??'Post'
  }
  remember(event?:Event){if(!this.config?.active||(event&&event.target!==this.textarea))return;this.selection={start:this.textarea.selectionStart,end:this.textarea.selectionEnd}}
  input(event:Event){if(event.target===this.textarea&&this.config?.active){++this.revision;this.update()}}
  private insert(before:string,after='',wrap=true){
    if(!this.config?.active)return
    const selection=document.activeElement===this.textarea&&wrap?{start:this.textarea.selectionStart,end:this.textarea.selectionEnd}:this.selection,result=insertCommentText(this.textarea.value,selection.start,selection.end,before,after,wrap)
    this.textarea.value=result.body;this.textarea.focus({preventScroll:true});this.textarea.setSelectionRange(result.start,result.end);++this.revision;this.remember();this.update()
  }
  private async selectTab(next:'write'|'preview'){
    if(!this.config?.active)return
    this.tab=next;this.textarea.hidden=this.toolbar.hidden=next!=='write';this.preview.hidden=next!=='preview'
    for(const button of this.all<HTMLButtonElement>('[data-comment-tab]')){const active=button.dataset.commentTab===next;button.setAttribute('aria-selected',String(active));button.tabIndex=active?0:-1}
    this.previewRequest?.abort();this.previewRequest=undefined;if(next!=='preview')return
    if(!this.textarea.value.trim()){this.preview.innerHTML='<em class="text-gray-600">Nothing to preview</em>';return}
    const controller=new AbortController();this.previewRequest=controller;this.preview.innerHTML=`<div class="text-xs text-gray-500 italic flex items-center gap-2">${commentIcon('loader-circle')} Rendering preview…</div>`
    try{const body:CommentsPreviewData['body']={body_md:this.textarea.value},data=await this.request<CommentsPreviewResponses[200]>('/api/comments/preview',{method:'POST',signal:controller.signal,body});if(this.previewRequest!==controller||!this.config?.active)return;if(data.error){this.preview.textContent=data.error;this.preview.classList.add('preview-error')}else{this.preview.classList.remove('preview-error');this.preview.innerHTML=data.html||'<em class="text-gray-600">Nothing to preview</em>'}}
    catch(error){if(this.previewRequest===controller&&!isAbortError(error)&&this.config?.active)this.preview.textContent='Preview failed: '+errorMessage(error,'unknown error')}
    finally{if(this.previewRequest===controller)this.previewRequest=undefined}
  }
  private openEmbed(kind:EmbedKind){
    this.modal?.dispose();const modal=mountCommentDialog('comment-embed-template');this.modal=modal;embedConfigurations.set(modal.dialog,{kind,insert:markdown=>this.insert(markdown,'',false),modal})
    modal.dialog.dataset.controller=[modal.dialog.dataset.controller,'comment-embed'].filter(Boolean).join(' ');modal.dialog.dataset.action=[modal.dialog.dataset.action,'input->comment-embed#validate submit->comment-embed#submit error->comment-embed#imageError:capture'].filter(Boolean).join(' ')
  }
  private openGIF(){this.modal?.dispose();this.modal=openKlipyPicker(markdown=>this.insert(markdown,'',false),this.gifState)}
  async post(event?:SubmitEvent){
    event?.preventDefault();if(event&&event.target!==this.form)return;if(!this.config?.active||this.submitButton.disabled)return
    const controller=new AbortController(),revision=this.revision;this.posting=controller;this.fail();this.update()
    try{if(!this.config.options.onSubmit)throw new Error('Comment controls are unavailable');await this.config.options.onSubmit(this.textarea.value.trim(),controller.signal);if(this.posting!==controller||!this.config?.active)return;if(this.revision===revision){this.textarea.value='';this.selection={start:0,end:0};await this.selectTab('write')}}
    catch(error){if(this.posting===controller)this.fail(error)}finally{if(this.posting===controller){this.posting=undefined;if(this.config?.active)this.update()}}
  }
  pointer(event:MouseEvent){if(this.config?.active&&eventElement(event.target)?.closest('[data-comment-toolbar] button'))event.preventDefault()}
  async click(event:MouseEvent){
    if(!this.config?.active)return
    const target=eventElement(event.target),action=target?.closest<HTMLElement>('[data-comment-action]')?.dataset.commentAction
    if(action){if(isWrapper(action)){const [before,after]=wrappers[action];this.insert(before,after)}else if(action==='gif')this.openGIF();else if(isEmbed(action))this.openEmbed(action);return}
    const tab=target?.closest<HTMLElement>('[data-comment-tab]')?.dataset.commentTab;if(tab==='write'||tab==='preview'){await this.selectTab(tab);return}
    if(target?.closest('[data-comment-dismiss]'))this.fail()
    if(target?.closest('[data-comment-input-cancel]')){this.textarea.value=this.config.options.initial??'';++this.revision;this.fail();this.update();this.config.options.onCancel?.()}
  }
  keyboard(event:KeyboardEvent){
    if(!this.config?.active)return
    if(event.target===this.textarea&&(event.ctrlKey||event.metaKey)){if(event.key==='Enter'){event.preventDefault();void this.post()}else if(event.key.toLowerCase()==='g'){event.preventDefault();this.remember();this.openGIF()}return}
    if(eventElement(event.target)?.hasAttribute('data-comment-tab')&&['ArrowLeft','ArrowRight','Home','End'].includes(event.key)){event.preventDefault();const next=event.key==='Home'?'write':event.key==='End'?'preview':this.tab==='write'?'preview':'write';void this.selectTab(next);this.q<HTMLButtonElement>(`[data-comment-tab="${next}"]`).focus()}
  }
  hide(){this.previewRequest?.abort();this.posting?.abort();this.previewRequest=this.posting=undefined;this.modal?.dispose();this.modal=undefined;super.hide();if(this.config?.active)this.update()}
  disconnect(){controllers.delete(this.element);super.disconnect()}
}
registerController('comment-input',CommentInputController)

export function createCommentInput(options:CommentInputOptions={}){
  const element=cloneCommentTemplate('comment-input-template'),textarea=requiredElement<HTMLTextAreaElement>(element,'[data-comment-body]'),config:Configuration={options,active:true};configurations.set(element,config)
  textarea.value=options.initial??'';textarea.placeholder=options.placeholder??'Write a comment… Markdown supported. Drop YouTube/Imgur/Giphy links to embed.';requiredElement<HTMLElement>(element,'[data-comment-input-cancel]').hidden=!options.onCancel&&!options.initial;requiredElement<HTMLElement>(element,'[data-comment-tab="preview"]').tabIndex=-1
  element.dataset.controller='comment-input';element.dataset.action='submit->comment-input#post input->comment-input#input select->comment-input#remember click->comment-input#remember keyup->comment-input#remember blur->comment-input#remember:capture keydown->comment-input#keyboard mousedown->comment-input#pointer click->comment-input#click pagehide@window->comment-input#hide'
  return {element,textarea,get value(){return textarea.value},pause(){controllers.get(element)?.hide()},destroy(){controllers.get(element)?.hide();config.active=false;configurations.delete(element)}}
}
export type CommentInputHandle=ReturnType<typeof createCommentInput>
