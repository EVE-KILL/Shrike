import {searchIcon,searchNode} from './search-cards'
import {readSearchResults,searchEntityPath,searchTypes,type SearchEntity,type SearchResults,type SearchType} from './search-model'

const recentKey='evekill-recent-searches'
const categories={...searchTypes,
  ship:{label:'Ships',icon:'rocket'},item:{label:'Items',icon:'package'},
  character:{label:'Characters',icon:'user'},corporation:{label:'Corporations',icon:'building'},
  alliance:{label:'Alliances',icon:'users'},system:{label:'Systems',icon:'map-pin'},
  constellation:{label:'Constellations',icon:'hexagon'},region:{label:'Regions',icon:'globe'},
}

// Ported from Nuxt SpotlightSearch/SpotlightContent. The shell owns its lifecycle;
// full-page search shares the validated response and entity-route model.
export class SpotlightSearch {
  private timer:ReturnType<typeof setTimeout>|undefined
  private request:AbortController|undefined
  private results:SearchResults|undefined
  private filter:SearchType|'all'='all'
  private selected=-1
  private recent:string[]=[]
  private visible:SearchEntity[]=[]
  private readonly input:HTMLInputElement
  private readonly content:HTMLElement

  constructor(private readonly dialog:HTMLDialogElement) {
    this.input=dialog.querySelector<HTMLInputElement>('[data-shell-target=searchInput]')!
    this.content=dialog.querySelector<HTMLElement>('[data-shell-target=searchResults]')!
  }
  open() {
    this.close()
    try {
      const value:unknown=JSON.parse(localStorage.getItem(recentKey)||'[]')
      this.recent=Array.isArray(value)?value.filter((query):query is string=>typeof query==='string'&&query.trim().length>=2&&query.length<=256).slice(0,5):[]
    } catch {this.recent=[]}
    this.input.placeholder=matchMedia('(max-width:767px)').matches?'Search...':'Search characters, corporations, alliances, ships...'
    document.documentElement.classList.add('spotlight-open')
    this.renderDefault();this.input.focus()
  }
  close() {
    this.cancel();this.input.value=''
    this.results=undefined;this.visible=[];this.filter='all';this.selected=-1
    this.input.setAttribute('aria-expanded','false');this.input.removeAttribute('aria-activedescendant')
    this.content.replaceChildren();this.content.removeAttribute('aria-busy');this.toggleClear()
    document.documentElement.classList.remove('spotlight-open')
  }
  private cancel() {clearTimeout(this.timer);this.request?.abort();this.request=undefined}
  private toggleClear() {const button=this.dialog.querySelector<HTMLElement>('[data-spotlight-clear]');if(button)button.hidden=!this.input.value}
  private saveRecent() {try{localStorage.setItem(recentKey,JSON.stringify(this.recent))}catch{/* Storage can be disabled. */}}
  private renderDefault() {
    const template=this.dialog.querySelector<HTMLTemplateElement>('[data-spotlight-default]')
    this.content.replaceChildren(...(template?[template.content.cloneNode(true)]:[]))
    const section=this.content.querySelector<HTMLElement>('[data-spotlight-recents]'),list=this.content.querySelector<HTMLElement>('[data-spotlight-recent-list]')
    if(!section||!list)return
    section.hidden=!this.recent.length
    this.recent.forEach((query,index)=>{
      const row=searchNode('div','spotlight-recent'),select=searchNode('button','spotlight-recent-select')
      select.type='button';select.dataset.spotlightRecent=String(index);select.append(searchIcon('clock'),searchNode('span','',query))
      const remove=searchNode('button','spotlight-recent-remove');remove.type='button';remove.dataset.spotlightRemove=String(index)
      remove.setAttribute('aria-label','Remove recent search: '+query);remove.append(searchIcon('x'));row.append(select,remove);list.append(row)
    })
  }
  private message(icon:string,title:string,detail?:string) {
    const message=searchNode('div','spotlight-message');message.append(searchIcon(icon),searchNode('p','',title))
    if(detail)message.append(searchNode('small','',detail))
    this.content.replaceChildren(message)
  }
  search() {
    this.cancel();this.toggleClear();this.filter='all';this.selected=-1;this.visible=[];this.results=undefined
    this.input.removeAttribute('aria-activedescendant');this.input.setAttribute('aria-expanded','false');this.content.removeAttribute('aria-busy')
    const query=this.input.value.trim()
    if(!query){this.renderDefault();return}
    if(query.length<3){this.message('type','Type at least 3 characters to search');return}
    this.message('loader','Searching...')
    const request=new AbortController();this.request=request
    this.timer=setTimeout(async()=>{
      this.content.setAttribute('aria-busy','true')
      try {
        const response=await fetch('/api/search?q='+encodeURIComponent(query),{signal:request.signal})
        if(!response.ok)throw new Error('Search unavailable')
        const data=readSearchResults(await response.json())
        if(request!==this.request||!this.dialog.open||!this.dialog.isConnected)return
        this.results=data
        if(!this.recent.includes(query)){this.recent.unshift(query);this.recent=this.recent.slice(0,5)}
        this.saveRecent()
        this.renderResults(query)
      } catch {
        if(request===this.request&&!request.signal.aborted&&this.dialog.open&&this.dialog.isConnected)this.message('circle-alert','Search is unavailable. Please try again.')
      } finally {
        if(request===this.request){this.request=undefined;this.content.removeAttribute('aria-busy')}
      }
    },300)
  }
  private renderResults(query:string) {
    const results=this.results
    if(!results?.hits.length){this.message('search-x','No results found','No results for "'+query+'"');return}
    const wrapper=searchNode('div','spotlight-results'),counts=Object.entries(results.counts) as [SearchType,number][]
    if(counts.length>1){
      const filters=searchNode('div','spotlight-filters')
      const pill=(type:SearchType|'all',label:string,count:number,icon:string)=>{
        const button=searchNode('button','',label+' '+count);button.type='button';button.dataset.spotlightFilter=type
        button.setAttribute('aria-pressed',String(this.filter===type));button.prepend(searchIcon(icon));filters.append(button)
      }
      pill('all','All',results.hits.length,'layers');counts.forEach(([type,count])=>pill(type,categories[type].label,count,categories[type].icon));wrapper.append(filters)
    }
    const groups=new Map<SearchType,SearchEntity[]>()
    for(const hit of results.hits)if(this.filter==='all'||hit.type===this.filter){const group=groups.get(hit.type)||[];group.push(hit);groups.set(hit.type,group)}
    this.visible=[...groups.values()].flat()
    const list=searchNode('div','spotlight-groups');list.id='spotlight-result-list';list.setAttribute('role','listbox');list.setAttribute('aria-label','Search results')
    let index=0
    for(const [type,hits] of groups){
      const section=searchNode('section','spotlight-group');section.setAttribute('role','group');section.setAttribute('aria-label',categories[type].label)
      const heading=searchNode('h3','',categories[type].label);heading.prepend(searchIcon(categories[type].icon));section.append(heading)
      for(const hit of hits){
        const link=searchNode('a','spotlight-result');link.href=searchEntityPath(hit);link.dataset.spotlightResult=String(index)
        link.id='spotlight-result-'+index++;link.setAttribute('role','option');link.setAttribute('aria-selected','false')
        const media=searchNode('span','spotlight-image'),url=this.image(hit)
        if(url){const image=searchNode('img');image.src=url;image.alt='';image.width=32;image.height=32;image.loading='lazy';media.append(image)}else media.append(searchIcon(categories[type].icon))
        const name=searchNode('span','spotlight-name');name.append(searchNode('strong','',hit.name||hit.id))
        if(hit.ticker)name.append(searchNode('small','', '['+hit.ticker+']'))
        link.append(media,name,searchNode('span','spotlight-type',hit.type),searchIcon('arrow-up-right'));section.append(link)
      }
      list.append(section)
    }
    const all=searchNode('a','spotlight-view-all','View all results');all.href='/search?q='+encodeURIComponent(query);all.prepend(searchIcon('arrow-right'))
    wrapper.append(list,all);this.content.replaceChildren(wrapper);this.input.setAttribute('aria-expanded',String(this.visible.length>0))
  }
  private image(hit:SearchEntity):string {
    switch(hit.type){
      case 'character':return 'https://images.evetech.net/characters/'+hit.id+'/portrait?size=64'
      case 'corporation':case 'faction':return 'https://images.evetech.net/corporations/'+hit.id+'/logo?size=64'
      case 'alliance':return 'https://images.evetech.net/alliances/'+hit.id+'/logo?size=64'
      case 'ship':case 'item':return '/images/types/'+hit.id+'/icon?size=64'
      case 'system':case 'region':case 'constellation':return '/images/'+hit.type+'s/'+hit.id+'?size=64'
      default:return ''
    }
  }
  click(event:MouseEvent) {
    const target=event.target instanceof Element?event.target.closest<HTMLElement>('button,a'):null
    if(!target)return
    if(target.hasAttribute('data-spotlight-close')){this.dialog.close();return}
    if(target.hasAttribute('data-spotlight-clear')){this.input.value='';this.search();this.input.focus();return}
    const recent=target.dataset.spotlightRecent,remove=target.dataset.spotlightRemove
    if(recent!==undefined){const query=this.recent[Number(recent)];if(query){this.input.value=query;this.search();this.input.focus()}return}
    if(remove!==undefined){this.recent.splice(Number(remove),1);this.saveRecent();this.renderDefault();this.input.focus();return}
    const filter=target.dataset.spotlightFilter
    if(filter==='all'||(filter&&Object.hasOwn(categories,filter))){
      this.filter=this.filter===filter?'all':filter as SearchType;this.selected=-1;this.input.removeAttribute('aria-activedescendant')
      this.renderResults(this.input.value.trim());this.content.querySelector<HTMLElement>('[data-spotlight-filter="'+filter+'"]')?.focus();return
    }
    if(target instanceof HTMLAnchorElement)this.dialog.close()
  }
  key(event:KeyboardEvent) {
    const focused=event.target instanceof Element?event.target.closest<HTMLElement>('[data-spotlight-result]'):null
    if(event.target!==this.input&&!focused)return
    if(focused){
      this.selected=Number(focused.dataset.spotlightResult)
      // A tab-focused link keeps native Enter, modifier and new-tab behavior.
      if(event.key==='Enter')return
    }
    if(event.key==='Enter'){
      event.preventDefault();const index=this.visible.length===1?0:this.selected
      if(index>=0)this.content.querySelector<HTMLAnchorElement>('[data-spotlight-result="'+index+'"]')?.click()
      return
    }
    if(event.key!=='ArrowDown'&&event.key!=='ArrowUp')return
    event.preventDefault();if(!this.visible.length)return
    this.selected=event.key==='ArrowDown'?(this.selected+1)%this.visible.length:(this.selected<=0?this.visible.length:this.selected)-1
    const links=this.content.querySelectorAll<HTMLAnchorElement>('[data-spotlight-result]')
    links.forEach((link,index)=>link.setAttribute('aria-selected',String(index===this.selected)))
    const selected=links[this.selected]
    if(selected){this.input.setAttribute('aria-activedescendant',selected.id);selected.scrollIntoView?.({block:'nearest'});if(event.target!==this.input)selected.focus()}
  }
}
