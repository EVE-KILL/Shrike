import {Controller} from '@hotwired/stimulus'
import {registerController} from './controllers/application'
import {requiredElement} from './shared/dom'
import {submitKillmails,killSubmissionToast} from './kill-submit'
import type {SubmissionResult} from './shared/kill-submissions'

export class PostController extends Controller<HTMLElement> {
  private alive = false
  private pending:AbortController|undefined
  private get textarea() {return requiredElement<HTMLTextAreaElement>(this.element,'textarea')}
  private get queue() {return requiredElement<HTMLButtonElement>(this.element,'[data-post-submit]')}
  private get clipboard() {return requiredElement<HTMLButtonElement>(this.element,'[data-post-clipboard]')}
  private get result() {return requiredElement<HTMLElement>(this.element,'[data-post-result]')}
  connect() {this.alive = true;this.busy(false)}
  private busy(value:boolean) {this.textarea.disabled = value;this.clipboard.disabled = value;this.queue.disabled = value || !this.textarea.value.trim();this.queue.textContent = value ? 'Queue ...' : 'Queue killmails';if(value) this.element.setAttribute('aria-busy','true');else this.element.removeAttribute('aria-busy')}
  private showResult(result:SubmissionResult) {
    if(!result.total) {this.result.hidden = true;return}
    this.result.hidden = false;requiredElement(this.element,'[data-post-total]').textContent = String(result.total);requiredElement(this.element,'[data-post-accepted]').textContent = String(result.accepted);requiredElement<HTMLElement>(this.element,'[data-post-rejected-row]').hidden = !result.rejected;requiredElement(this.element,'[data-post-rejected]').textContent = String(result.rejected)
    const links = requiredElement(this.element,'[data-post-links]');links.replaceChildren();for(const id of result.killmails.slice(0,20)) {const link = document.createElement('a');link.href = '/kill/'+id;link.textContent = '#'+id;links.append(link)}if(result.killmails.length > 20) {const more = document.createElement('span');more.textContent = '+'+(result.killmails.length-20)+' more';links.append(more)}
  }
  private async submitText(text:string,request:AbortController) {
    const draft = this.textarea.value;this.result.hidden = true
    try {const result = await submitKillmails(text,result=>{if(this.alive && this.pending === request && !request.signal.aborted) this.showResult(result)},request.signal);if(this.alive && this.pending === request && result && result.accepted+result.existing === result.total && this.textarea.value === draft) this.textarea.value = ''}
    finally {if(this.pending === request) {this.pending = undefined;this.busy(false)}}
  }
  input() {if(this.alive) this.queue.disabled = !this.textarea.value.trim() || Boolean(this.pending)}
  submit(event:SubmitEvent) {event.preventDefault();if(!this.alive || this.pending) return;const text = this.textarea.value;if(!text.trim()) {killSubmissionToast('info','Paste one or more killmail links first');return}const request = new AbortController();this.pending = request;this.busy(true);void this.submitText(text,request)}
  async fromClipboard() {
    if(!this.alive || this.pending) return
    const request = new AbortController();this.pending = request;this.busy(true)
    try {const text = await navigator.clipboard.readText();if(!this.alive || this.pending !== request || request.signal.aborted) return;if(!text.trim()) {killSubmissionToast('info','Clipboard is empty');return}await this.submitText(text,request)}
    catch {if(this.alive && this.pending === request) killSubmissionToast('error','Could not read clipboard — paste the link into the text box instead')}
    finally {if(this.pending === request) {this.pending = undefined;this.busy(false)}}
  }
  hide() {this.alive = false;this.pending?.abort();this.pending = undefined;this.busy(false)}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;this.busy(false)}}
  disconnect() {this.hide()}
}
registerController('post',PostController)
