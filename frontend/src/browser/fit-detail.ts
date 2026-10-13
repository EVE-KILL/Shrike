import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {eventElement,requiredElement} from './shared/dom'
import {isRecord,isAbortError} from './shared/json'
import {readFitRatingSeed,readRatingResult,isRating,type FitRatingSeed} from './shared/fit-detail'
import {readBrowserSession} from './browser-request'

type CommentModule = Pick<typeof import('./comment-thread'),'initializeCommentThreads'>
export class FitRatingController extends RequestComponent {
  private alive = false
  private seed!:FitRatingSeed
  private hoverValue:number|null = null
  private saving = false
  private generation = 0
  private reconcile = false
  connect() {const script = requiredElement(this.element.closest('[data-fit-detail="true"]')!,'#saved-fit-state');const value:unknown = JSON.parse(script.textContent ?? 'null');this.seed = readFitRatingSeed(value);this.alive = true;this.render()}
  private render() {
    if(!this.seed) return
    const selected = this.hoverValue ?? this.seed.viewer ?? (this.seed.average == null ? 0 : Math.round(this.seed.average))
    for(const star of this.all<HTMLButtonElement>('[data-fit-rate]')) {const value = Number(star.dataset.fitRate),on = value <= selected;star.classList.toggle('on',on);star.classList.toggle('own',this.seed.viewer !== null);star.classList.toggle('preview',this.hoverValue !== null && on);star.setAttribute('aria-pressed',String(this.seed.viewer === value));star.disabled = this.saving}
    const summary = this.q('[data-fit-rating-summary]');summary.replaceChildren();if(this.seed.count) {const number = document.createElement('strong');number.textContent = this.seed.average?.toFixed(1) ?? '–';summary.append(number,` (${this.seed.count})`)}else summary.textContent = 'unrated'
    const clear = this.q<HTMLButtonElement>('[data-fit-clear-rating]');clear.hidden = this.seed.viewer === null;clear.disabled = this.saving
  }
  hover(event:MouseEvent) {const star = eventElement(event.target)?.closest<HTMLElement>('[data-fit-rate]');if(!this.alive || this.saving || !star) return;const value = Number(star.dataset.fitRate);if(isRating(value)) {this.hoverValue = value;this.render()}}
  leave(event:MouseEvent) {const rating = eventElement(event.target)?.closest('[data-fit-rating]');if(!rating || event.relatedTarget instanceof Node && rating.contains(event.relatedTarget)) return;this.hoverValue = null;this.render()}
  focus(event:FocusEvent) {const star = eventElement(event.target)?.closest<HTMLElement>('[data-fit-rate]');if(!star || !this.alive || this.saving) return;this.hoverValue = event.type === 'focusin' && isRating(Number(star.dataset.fitRate)) ? Number(star.dataset.fitRate) : null;this.render()}
  click(event:MouseEvent) {const target = eventElement(event.target);if(!this.alive || !target) return;const star = target.closest<HTMLElement>('[data-fit-rate]');if(star && isRating(Number(star.dataset.fitRate))) void this.save(Number(star.dataset.fitRate));else if(target.closest('[data-fit-clear-rating]')) void this.save(null)}
  navigate(url:string) {location.assign(url)}
  private async reload(generation:number) {
    const next = readFitRatingSeed(await this.request<unknown>('/api/fit/'+encodeURIComponent(this.seed.id),{},'Could not refresh your rating.'))
    if(!this.alive || this.generation !== generation) return false
    if(next.id !== this.seed.id) throw new Error('Invalid saved fitting')
    this.seed = next;this.reconcile = false;return true
  }
  private async save(value:number|null) {
    if(!this.alive || this.saving) return
    const generation = ++this.generation;this.saving = true;this.error('[data-fit-rating-error]');this.render()
    try {
      if(this.reconcile && !await this.reload(generation)) return
      const user = await readBrowserSession();if(!this.alive || this.generation !== generation) return
      if(!user) {const login = await this.request<unknown>('/auth/login?'+new URLSearchParams({redirect:location.pathname+location.search}),{},'Login is unavailable.');if(!this.alive || this.generation !== generation) return;if(!isRecord(login) || typeof login.url !== 'string') throw new Error('Invalid login destination');const destination = new URL(login.url);if(destination.protocol !== 'https:') throw new Error('Invalid login destination');this.navigate(destination.href);return}
      this.reconcile = true
      const result = readRatingResult(await this.request<unknown>('/api/fit/'+encodeURIComponent(this.seed.id)+'/rating',value === null ? {method:'DELETE'} : {method:'PUT',body:{rating:value}},'Could not save your rating. Please try again.'),value === null)
      if(!this.alive || this.generation !== generation) return
      this.seed = {...this.seed,viewer:'rating' in result ? result.rating : null,average:result.aggregate.rating_avg,count:result.aggregate.rating_count};this.reconcile = false
    }catch(error) {if(this.alive && this.generation === generation && !isAbortError(error)) this.error('[data-fit-rating-error]',error)}
    finally {if(this.generation === generation) {this.saving = false;this.hoverValue = null;this.render()}}
  }
  hide() {this.alive = false;this.generation++;this.saving = false;this.hoverValue = null;super.hide();this.render()}
  show(event:PageTransitionEvent) {if(!event.persisted) return;this.alive = true;if(this.reconcile) {const generation = ++this.generation;this.saving = true;this.render();void this.reload(generation).catch(error=>{if(this.alive && this.generation === generation && !isAbortError(error)) this.error('[data-fit-rating-error]',error)}).finally(()=>{if(this.generation === generation) {this.saving = false;this.render()}})}}
  disconnect() {this.hide();super.disconnect()}
}
export class FitCommentsController extends RequestComponent {
  private alive = false
  private generation = 0
  private comments:Promise<CommentModule>|undefined
  private interrupted = false
  connect() {this.alive = true}
  importComments():Promise<CommentModule> {return import('./comment-thread')}
  private async display() {
    const container = this.q<HTMLElement>('#fit-comments'),generation = ++this.generation;this.interrupted = true
    try {this.comments ??= this.importComments();const module = await this.comments;if(!this.alive || this.generation !== generation || container.hidden) return;const template = this.q<HTMLTemplateElement>('[data-fit-comments-template]');container.replaceChildren(template.content.cloneNode(true));module.initializeCommentThreads(container);this.interrupted = false}
    catch {if(this.alive && this.generation === generation && !container.hidden) {this.comments = undefined;this.interrupted = false;container.textContent = 'Could not load comments. Close and reopen to try again.'}}
  }
  click(event:MouseEvent) {if(!this.alive || !eventElement(event.target)?.closest('[data-fit-comments-toggle]')) return;const container = this.q<HTMLElement>('#fit-comments');container.hidden = !container.hidden;this.q('[data-fit-comments-toggle]').setAttribute('aria-expanded',String(!container.hidden));if(container.hidden) {this.generation++;this.interrupted = false;container.replaceChildren()}else void this.display()}
  hide() {this.alive = false;this.generation++;super.hide()}
  show(event:PageTransitionEvent) {if(!event.persisted) return;this.alive = true;if(this.interrupted && !this.q<HTMLElement>('#fit-comments').hidden) void this.display()}
  disconnect() {this.hide();super.disconnect()}
}
registerController('fit-rating',FitRatingController)
registerController('fit-comments',FitCommentsController)
