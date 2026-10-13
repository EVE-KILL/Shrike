import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {docsEndpointCard} from './controllers/docs-endpoint'
import {docsElement as element} from './shared/docs-dom'
import {eventElement} from './shared/dom'
import {isAbortError,errorMessage} from './shared/json'
import {buildDocsModel,type DocsModel} from '../models/openapiDocs'

type Card = {card:HTMLDetailsElement;searchText:string;endpointKey:string}
type Tag = {section:HTMLElement;button:HTMLButtonElement;count:HTMLElement;cards:HTMLDetailsElement[]}
type Group = {name:string;section:HTMLElement;navGroup:HTMLDetailsElement;groupCount:HTMLElement;tags:Tag[]}
export class DocsController extends RequestComponent {
  private alive = false
  private model:DocsModel|undefined
  private cards:Card[] = []
  private sections:Group[] = []
  private tags = new Map<string,Tag>()
  private openGroups = new Set<string>()
  private debounce:ReturnType<typeof setTimeout>|undefined
  private read:AbortController|undefined
  private dirty = false
  connect() {this.alive = true;if(!this.model) void this.load()}
  async load() {
    if(!this.alive || this.read) return
    const request = new AbortController();this.read = request;this.dirty = true
    this.q<HTMLElement>('#docs-loading').hidden = false;this.q<HTMLElement>('#docs-error').hidden = true
    if(!this.model) this.q<HTMLElement>('#docs-layout').hidden = true
    try {
      const value = await this.request<unknown>('/api/openapi.json',{signal:request.signal},'Failed to load the API document')
      if(!this.alive || this.read !== request) return
      this.render(buildDocsModel(value));this.dirty = false
    } catch(error) {
      if(this.alive && this.read === request && !isAbortError(error)) {this.q('#docs-error-message').textContent = errorMessage(error,'Failed to load the API document');this.q<HTMLElement>('#docs-error').hidden = false;this.dirty = false}
    } finally {if(this.read === request) {this.read = undefined;this.q<HTMLElement>('#docs-loading').hidden = true}}
  }
  private render(model:DocsModel) {
    this.model = model;this.cards = [];this.sections = [];this.tags.clear()
    const nav = this.q('#docs-nav'),content = this.q('#docs-content');nav.replaceChildren();content.replaceChildren()
    this.q('#docs-spec-title').textContent = model.title;this.q('#docs-spec-version').textContent = model.version;this.q<HTMLElement>('#docs-spec-meta').hidden = false
    for(const group of model.groups) {
      const section = element('section','docs-group');section.append(element('h2','',group.name))
      const navGroup = element('details','docs-nav-group'),summary = element('summary','',group.name),groupCount = element('span'),navTags = element('div','docs-nav-tags')
      navGroup.dataset.docsGroup = group.name;summary.append(groupCount);navGroup.append(summary,navTags);nav.append(navGroup)
      const tags:Tag[] = []
      for(const tag of group.tags) {
        const tagSection = element('section','docs-tag');tagSection.id = 'tag-'+tag.name.replace(/\s+/g,'-');tagSection.append(element('h3','',tag.name))
        if(tag.description) tagSection.append(element('p','docs-tag-description',tag.description))
        const list = element('div','docs-endpoint-list'),cards:HTMLDetailsElement[] = []
        for(const endpoint of tag.endpoints) {
          const card = docsEndpointCard(endpoint);cards.push(card);list.append(card)
          this.cards.push({card,searchText:`${endpoint.path} ${endpoint.summary} ${endpoint.method} ${endpoint.description ?? ''}`.toLowerCase(),endpointKey:endpoint.method+' '+endpoint.path})
        }
        tagSection.append(list);section.append(tagSection)
        const button = element('button','',tag.name),count = element('span','',String(cards.length));button.type = 'button';button.dataset.docsTag = String(this.tags.size);button.append(count);navTags.append(button)
        const entry:Tag = {section:tagSection,button,count,cards};this.tags.set(button.dataset.docsTag,entry);tags.push(entry)
      }
      content.append(section);this.sections.push({name:group.name,section,navGroup,groupCount,tags})
    }
    this.filter();this.q<HTMLElement>('#docs-layout').hidden = false
  }
  search(event:InputEvent) {if(event.target !== this.q('#docs-search')) return;this.stopTimer(this.debounce);this.debounce = this.after(80,()=>{this.debounce = undefined;this.filter()})}
  private filter() {
    if(!this.model) return
    const query = this.q<HTMLInputElement>('#docs-search').value.trim().toLowerCase(),visible = new Set<string>()
    for(const entry of this.cards) {entry.card.hidden = Boolean(query && !entry.searchText.includes(query));if(!entry.card.hidden) visible.add(entry.endpointKey)}
    for(const group of this.sections) {
      let groupVisible = 0
      for(const tag of group.tags) {const count = tag.cards.filter(card=>!card.hidden).length;tag.section.hidden = tag.button.hidden = count === 0;tag.count.textContent = String(count);groupVisible += count}
      group.section.hidden = group.navGroup.hidden = groupVisible === 0;group.groupCount.textContent = String(groupVisible);group.navGroup.open = Boolean(query) || this.openGroups.has(group.name)
    }
    this.q('#docs-count').textContent = `${visible.size} of ${this.model.total} endpoints`
    const content = this.q('#docs-content')
    if(!visible.size) content.replaceChildren(element('p','docs-empty','No endpoints match'))
    else if(content.querySelector('.docs-empty')) content.replaceChildren(...this.sections.map(group=>group.section))
  }
  toggle(event:Event) {
    const group = eventElement(event.target)
    if(!(group instanceof HTMLDetailsElement) || !group.matches('.docs-nav-group') || this.q<HTMLInputElement>('#docs-search').value.trim()) return
    const name = group.dataset.docsGroup;if(name) {if(group.open) this.openGroups.add(name);else this.openGroups.delete(name)}
  }
  click(event:MouseEvent) {
    const target = eventElement(event.target)
    if(target?.closest('#docs-retry')) {void this.load();return}
    const button = target?.closest<HTMLButtonElement>('[data-docs-tag]'),tag = button ? this.tags.get(button.dataset.docsTag ?? '') : undefined
    if(tag) {tag.section.scrollIntoView({behavior:'smooth',block:'start'});for(const entry of this.tags.values()) entry.button.classList.toggle('active',entry === tag)}
  }
  hide() {this.alive = false;this.read?.abort();this.read = undefined;this.stopTimer(this.debounce);this.debounce = undefined;this.q<HTMLElement>('#docs-loading').hidden = true;super.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;if(this.dirty) void this.load();else this.filter()}}
}
registerController('docs',DocsController)
