import {Controller} from '@hotwired/stimulus'
import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {eventElement} from './shared/dom'
import {errorMessage,isAbortError} from './shared/json'

export class BlogDatesController extends Controller<HTMLElement> {
  connect() {
    for(const element of this.element.querySelectorAll<HTMLTimeElement>('time[data-blog-date]:not([data-localized])')) {
      const date = new Date(element.dateTime)
      if(!Number.isNaN(date.getTime())) element.textContent = date.toLocaleDateString(undefined,{year:'numeric',month:element.dataset.blogDate === 'long' ? 'long' : 'short',day:'numeric'})
      element.dataset.localized = 'true'
    }
  }
}
export class BlogController extends RequestComponent {
  private alive = false
  private read:AbortController|undefined
  private interrupted = false
  connect() {this.alive = true}
  private get button() {return this.element.querySelector<HTMLButtonElement>('[data-blog-more]')}
  private busy(value:boolean) {const button = this.button;if(button) {button.disabled = value;button.textContent = value ? 'Loading...' : 'Load more'}const list = this.q('[data-blog-list]');if(value) list.setAttribute('aria-busy','true');else list.removeAttribute('aria-busy')}
  private async load() {
    const button = this.button,cursor = button?.dataset.cursor;if(!this.alive || !button || this.read || !cursor) return
    const request = new AbortController();this.read = request;this.interrupted = true;this.busy(true);this.error('[data-blog-error]')
    const query = new URLSearchParams({cursor}),tag = this.element.dataset.blogTag;if(tag) query.set('tag',tag)
    try {
      const result = await this.readTextWithHeaders('/_fragments/blog?'+query,'Could not load more posts.',{signal:request.signal})
      if(!this.alive || this.read !== request) return
      const parsed = new DOMParser().parseFromString(result.text,'text/html'),cards = [...parsed.body.children]
      if(cards.some(card=>!(card instanceof HTMLAnchorElement) || !card.matches('.blog-card') || !new URL(card.href,location.href).pathname.startsWith('/blog/')) || !cards.length && result.text.trim()) throw new Error('Could not load more posts.')
      const list = this.q('[data-blog-list]'),seen = new Set([...list.querySelectorAll<HTMLAnchorElement>('.blog-card')].map(card=>card.getAttribute('href')))
      for(const card of cards) {const href = card.getAttribute('href');if(!seen.has(href)) {seen.add(href);list.append(document.importNode(card,true))}}
      button.dataset.cursor = result.headers.get('X-Next-Cursor') ?? '';const more = button.closest<HTMLElement>('.blog-more');if(more) more.hidden = !button.dataset.cursor;this.interrupted = false
    } catch(error) {if(this.alive && this.read === request && !isAbortError(error)) {this.interrupted = false;this.error('[data-blog-error]',errorMessage(error,'Could not load more posts.'))}}
    finally {if(this.read === request) {this.read = undefined;this.busy(false)}}
  }
  click(event:MouseEvent) {if(eventElement(event.target)?.closest('[data-blog-more]')) void this.load()}
  hide() {this.alive = false;this.read?.abort();this.read = undefined;this.busy(false);super.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;if(this.interrupted) void this.load()}}
}
registerController('blog',BlogController)
registerController('blog-dates',BlogDatesController)
