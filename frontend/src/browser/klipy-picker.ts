import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {KLIPY_FAVORITES_KEY,parseKlipyFavorites,addKlipyFavorite,type KlipyFavorite} from './comment-input-data'
import {mountCommentDialog,commentIcon,type CommentModal} from './comment-controls'
import {eventElement} from './shared/dom'
import {isRecord,errorMessage,isAbortError} from './shared/json'
import type {CommentsKlipySearchResponses} from '../../shared/api/types.gen'

type GifTab='trending'|'search'|'favorites'
export interface KlipyState {tab?:GifTab;query?:string;search?:string}
interface MediaTile {id:number;title:string;preview:string;insert:string;width:number;height:number}
interface Configuration {onInsert:(markdown:string)=>void;state:KlipyState;modal:CommentModal}
const configurations=new WeakMap<HTMLElement,Configuration>()
const readFavorites=()=>{try{return parseKlipyFavorites(localStorage.getItem(KLIPY_FAVORITES_KEY))}catch{return []}}
const safeMediaURL=(raw:unknown)=>{if(typeof raw!=='string')return null;try{const url=new URL(raw);return url.protocol==='http:'||url.protocol==='https:'?url.href:null}catch{return null}}
const dimension=(value:unknown)=>typeof value==='number'&&Number.isFinite(value)&&value>=0?value:0
export function readKlipyTiles(rows:unknown):MediaTile[]{
  if(!Array.isArray(rows))return []
  return rows.flatMap(raw=>{
    if(!isRecord(raw)||typeof raw.id!=='number'||!Number.isSafeInteger(raw.id))return []
    const file=isRecord(raw.file)?raw.file:{},sm=isRecord(file.sm)?file.sm:{},md=isRecord(file.md)?file.md:{},webp=isRecord(sm.webp)?sm.webp:{},mp4=isRecord(md.mp4)?md.mp4:{}
    const preview=safeMediaURL(webp.url),insert=safeMediaURL(mp4.url)
    return preview&&insert?[{id:raw.id,title:typeof raw.title==='string'?raw.title:'',preview,insert,width:dimension(mp4.width)||dimension(raw.width),height:dimension(mp4.height)||dimension(raw.height)}]:[]
  })
}
export class KlipyPickerController extends RequestComponent {
  private config:Configuration|undefined
  private tab:GifTab='trending'
  private query=''
  private favorites:KlipyFavorite[]=[]
  private items:MediaTile[]=[]
  private page=1
  private hasMore=false
  private loading=false
  private errorText=''
  private pending:AbortController|undefined
  private debounce:ReturnType<typeof setTimeout>|undefined
  private get searchInput(){return this.q<HTMLInputElement>('[data-gif-search]')}
  private get grid(){return this.q<HTMLElement>('[data-gif-grid]')}
  connect(){this.config=configurations.get(this.element);if(!this.config)return;this.tab=this.config.state.tab??'trending';this.query=this.config.state.query??'';this.searchInput.value=this.config.state.search??'';this.favorites=readFavorites();void this.load();this.searchInput.focus()}
  private entries():MediaTile[]{return this.tab==='favorites'?this.favorites.flatMap(favorite=>{const preview=safeMediaURL(favorite.preview_url),insert=safeMediaURL(favorite.insert_url);return preview&&insert?[{id:favorite.id,title:favorite.title,preview,insert,width:favorite.width,height:favorite.height}]:[]}):this.items}
  private render(){
    this.q<HTMLElement>('[data-gif-clear]').hidden=!this.searchInput.value;this.q<HTMLElement>('[data-gif-favorites-count]').textContent=this.favorites.length?`(${this.favorites.length})`:''
    for(const button of this.all<HTMLButtonElement>('[data-gif-tab]')){const active=button.dataset.gifTab===this.tab;button.classList.toggle('klipy-tab-active',active);button.setAttribute('aria-selected',String(active));if(button.dataset.gifTab==='search')button.hidden=this.query.length<2}
    this.grid.replaceChildren()
    for(const item of this.entries()){
      const tile=document.createElement('div');tile.className='klipy-tile';tile.tabIndex=0;tile.setAttribute('role','button');tile.dataset.gifItem=String(item.id);tile.title=item.title;tile.setAttribute('aria-label',item.title||'Insert GIF')
      const image=new Image();image.src=item.preview;image.alt=item.title;image.loading='lazy';tile.append(image)
      const saved=this.favorites.some(favorite=>favorite.id===item.id),star=document.createElement('button');star.type='button';star.dataset.gifFavorite=String(item.id);star.className=`favorite-btn${saved?' favorited':''}`;star.setAttribute('aria-label',saved?'Remove favorite':'Add to favorites');star.title=saved?'Remove favorite':'Add to favorites';star.innerHTML=commentIcon('star');tile.append(star);this.grid.append(tile)
    }
    const notice=(message:string,red=false)=>{const node=document.createElement('div');node.className=`col-span-full text-center py-8 text-sm ${red?'text-red-400':'text-gray-500 italic'}`;node.textContent=message;this.grid.append(node)}
    if(this.tab==='favorites'&&!this.favorites.length)notice('No favorites yet — click the star on any GIF to save it.')
    if(this.loading&&!this.items.length&&this.tab!=='favorites')notice('Loading…')
    if(this.errorText)notice(this.errorText,true)
    this.q<HTMLElement>('[data-gif-more-wrap]').hidden=this.tab==='favorites'||!this.hasMore||!this.items.length
    const more=this.q<HTMLButtonElement>('[data-gif-more]');more.disabled=this.loading;more.textContent=this.loading?'Loading…':'Load more'
    if(this.config)Object.assign(this.config.state,{tab:this.tab,query:this.query,search:this.searchInput.value})
  }
  private async load(reset=true){
    this.pending?.abort();this.pending=undefined
    if(this.tab==='favorites'){this.loading=false;this.errorText='';this.render();return}
    if(this.tab==='search'&&this.query.length<2)return
    if(reset){this.items=[];this.page=1;this.hasMore=false}
    this.loading=true;this.errorText='';this.render();const controller=new AbortController();this.pending=controller;const page=reset?1:this.page+1,params=new URLSearchParams({page:String(page),per_page:'24'});if(this.tab==='search')params.set('q',this.query)
    try{const data=await this.request<CommentsKlipySearchResponses[200]>(`/api/comments/klipy/${this.tab==='search'?'search':'trending'}?${params}`,{signal:controller.signal},'Failed to load GIFs');if(this.pending!==controller)return;const rows=readKlipyTiles(data.items);this.items=reset?rows:[...this.items,...rows];this.page=page;this.hasMore=data.has_next}
    catch(error){if(this.pending===controller&&!isAbortError(error))this.errorText=errorMessage(error,'Failed to load GIFs')}
    finally{if(this.pending===controller){this.pending=undefined;this.loading=false;this.render()}}
  }
  search(event:Event){if(event.target!==this.searchInput)return;this.q<HTMLElement>('[data-gif-clear]').hidden=!this.searchInput.value;this.stopTimer(this.debounce);this.pending?.abort();this.debounce=this.after(300,()=>{this.debounce=undefined;this.query=this.searchInput.value.trim();if(this.query.length>=2)this.tab='search';else if(this.tab==='search')this.tab='trending';void this.load()})}
  private pick(id:number){const item=this.entries().find(item=>item.id===id);if(!item||!this.config)return;this.config.modal.dispose();this.config.onInsert(`[GIF](${item.insert})`)}
  async click(event:MouseEvent){
    const target=eventElement(event.target),favorite=target?.closest<HTMLElement>('[data-gif-favorite]')
    if(favorite){event.stopPropagation();const id=Number(favorite.dataset.gifFavorite),item=this.entries().find(item=>item.id===id);if(!item)return;this.favorites=this.favorites.some(favorite=>favorite.id===id)?this.favorites.filter(favorite=>favorite.id!==id):addKlipyFavorite(this.favorites,{id,title:item.title,preview_url:item.preview,insert_url:item.insert,width:item.width,height:item.height});try{localStorage.setItem(KLIPY_FAVORITES_KEY,JSON.stringify(this.favorites))}catch{/* Storage may be unavailable. */}this.render();return}
    const tile=target?.closest<HTMLElement>('[data-gif-item]');if(tile){this.pick(Number(tile.dataset.gifItem));return}
    const tab=target?.closest<HTMLElement>('[data-gif-tab]')?.dataset.gifTab;if(tab==='trending'||tab==='search'||tab==='favorites'){this.stopTimer(this.debounce);this.tab=tab;this.favorites=readFavorites();await this.load();return}
    if(target?.closest('[data-gif-clear]')){this.searchInput.value='';this.query='';if(this.tab==='search')this.tab='trending';this.stopTimer(this.debounce);await this.load();this.searchInput.focus();return}
    if(target?.closest('[data-gif-more]')&&this.hasMore&&!this.loading)await this.load(false)
  }
  keyboard(event:KeyboardEvent){const target=eventElement(event.target);if(target?.hasAttribute('data-gif-item')&&(event.key==='Enter'||event.key===' ')){event.preventDefault();this.pick(Number(target.getAttribute('data-gif-item')))}}
  storage(event:StorageEvent){if(event.key===KLIPY_FAVORITES_KEY){this.favorites=readFavorites();this.render()}}
  hide(){this.stopTimer(this.debounce);this.pending?.abort();this.pending=undefined;super.hide()}
}
registerController('klipy-picker',KlipyPickerController)
export function openKlipyPicker(onInsert:(markdown:string)=>void,state:KlipyState={}){
  const modal=mountCommentDialog('comment-gif-template');configurations.set(modal.dialog,{onInsert,state,modal})
  modal.dialog.dataset.controller=[modal.dialog.dataset.controller,'klipy-picker'].filter(Boolean).join(' ')
  modal.dialog.dataset.action=[modal.dialog.dataset.action,'click->klipy-picker#click input->klipy-picker#search keydown->klipy-picker#keyboard storage@window->klipy-picker#storage close->klipy-picker#hide'].filter(Boolean).join(' ')
  return modal
}
