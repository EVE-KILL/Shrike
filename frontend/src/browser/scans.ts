import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {BrowserRequestError} from './browser-request'
import {eventElement} from './shared/dom'
import {isAbortError,isRecord} from './shared/json'
import {isScanKind,localScanNames,readScanAnalysis,readScanHash,scanSaveBody,type ScanAnalysis,type ScanKind} from './shared/scans'

export class ScansController extends RequestComponent {
  private alive = false
  private authenticated = false
  private result:ScanAnalysis|undefined
  private analysis:AbortController|undefined
  private identityRead:AbortController|undefined
  private write:AbortController|undefined
  private loginRead:AbortController|undefined
  private clipboardGeneration = 0
  private get kind():ScanKind {
    const kind = this.element.dataset.scanKind
    if(!isScanKind(kind)) throw new Error('Invalid scan component kind')
    return kind
  }
  connect() {this.alive = true;this.update();void this.identity()}
  private current(request:AbortController,owner:AbortController|undefined) {
    return this.alive && this.element.isConnected && owner === request && !request.signal.aborted
  }
  private showError(error:unknown = '') {this.error('[data-scan-error]',error)}
  private update() {
    const loading = Boolean(this.analysis),saving = Boolean(this.write)
    this.q<HTMLElement>('[data-scan-actions]').hidden = !this.result
    this.q<HTMLButtonElement>('[data-scan-save]').hidden = !this.authenticated || !this.result
    this.q<HTMLElement>('[data-scan-login-hint]').hidden = this.authenticated || !this.result
    this.q<HTMLButtonElement>('[data-scan-paste]').disabled = loading || saving
    this.q<HTMLButtonElement>('[data-scan-save]').disabled = saving || loading || !this.result
    this.q<HTMLButtonElement>('[data-scan-login]').disabled = Boolean(this.loginRead)
    for(const part of ['paste','save'] as const) {
      const pending = part === 'paste' ? loading : saving
      this.q<HTMLElement>(`[data-scan-${part}-idle]`).hidden = pending
      this.q<HTMLElement>(`[data-scan-${part}-spinner]`).hidden = !pending
      this.q(`[data-scan-${part}-label]`).textContent = pending ? (part === 'paste' ? 'Processing...' : 'Saving...') : (part === 'paste' ? 'Paste from Clipboard' : 'Save & Share')
    }
  }
  async identity() {
    if(!this.alive) return
    this.identityRead?.abort()
    const request = new AbortController();this.identityRead = request
    try {
      const data = await this.request<unknown>('/auth/me',{signal:request.signal})
      if(this.current(request,this.identityRead)) {this.authenticated = isRecord(data) && isRecord(data.user);this.update()}
    } catch {/* Public analysis remains usable; save independently checks the session. */}
    finally {if(this.identityRead === request) this.identityRead = undefined}
  }
  async paste() {
    if(!this.alive || this.analysis || this.write) return
    const generation = ++this.clipboardGeneration
    try {
      const text = await navigator.clipboard.readText()
      if(this.alive && this.element.isConnected && generation === this.clipboardGeneration) await this.process(text)
    } catch {if(this.alive && generation === this.clipboardGeneration) this.showError('Could not read clipboard. Try pressing Ctrl+V instead.')}
  }
  pasteEvent(event:ClipboardEvent) {
    const target = eventElement(event.target)
    if(!this.alive || this.write || target?.closest('input, textarea, select, [contenteditable]') || (target instanceof HTMLElement && target.isContentEditable)) return
    const text = event.clipboardData?.getData('text/plain') || event.clipboardData?.getData('text') || ''
    if(text) {event.preventDefault();this.clipboardGeneration++;void this.process(text)}
  }
  private async process(text:string) {
    if(!this.alive || this.write) return
    if(!text.trim()) {this.showError(this.kind === 'dscan' ? 'Clipboard is empty — copy your D-Scan from EVE first.' : 'Clipboard is empty — copy your local chat from EVE first.');return}
    const names = localScanNames(text)
    if(this.kind === 'localscan' && !names.length) {this.showError('No valid character names found.');return}
    this.analysis?.abort()
    const request = new AbortController();this.analysis = request;this.showError();this.update()
    try {
      const value = await this.request<unknown>(`/api/tools/${this.kind}`,{method:'POST',body:this.kind === 'dscan' ? {dscan:text} : names,signal:request.signal},'Failed to process scan.')
      if(!this.current(request,this.analysis)) return
      const result = readScanAnalysis(this.kind,text,value)
      const html = await this.text(`/_fragments/scans/${this.kind}`,'Could not display scan results.',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(result.result),signal:request.signal})
      if(!this.current(request,this.analysis)) return
      const doc = new DOMParser().parseFromString(html,'text/html'),content = doc.querySelector('[data-scan-result-content]')
      if(!content) throw new Error('Could not display scan results.')
      this.q('[data-scan-results]').replaceChildren(document.importNode(content,true));this.result = result
    } catch(error) {if(this.current(request,this.analysis) && !isAbortError(error)) this.showError(error)}
    finally {if(this.analysis === request) {this.analysis = undefined;this.update()}}
  }
  async save() {
    if(!this.alive || this.write || this.analysis || !this.result) return
    this.clipboardGeneration++
    const scan = this.result,request = new AbortController();this.write = request;this.showError();this.update()
    try {
      const data = await this.request<unknown>(`/api/tools/${scan.kind}/save`,{method:'POST',body:scanSaveBody(scan),signal:request.signal},'Failed to save scan.')
      if(this.current(request,this.write)) location.assign(`/tools/${scan.kind}/${readScanHash(data)}`)
    } catch(error) {
      if(!this.current(request,this.write) || isAbortError(error)) return
      if(error instanceof BrowserRequestError && error.status === 401) {
        this.identityRead?.abort();this.identityRead = undefined;this.authenticated = false
        this.showError('Your session expired. Log in again to save this scan.')
      } else this.showError(error)
    } finally {if(this.write === request) {this.write = undefined;this.update()}}
  }
  async login() {
    if(!this.alive || this.loginRead) return
    const request = new AbortController();this.loginRead = request;this.showError();this.update()
    try {
      const data = await this.request<unknown>('/auth/login?'+new URLSearchParams({redirect:location.pathname+location.search}),{signal:request.signal})
      if(!this.current(request,this.loginRead)) return
      if(!isRecord(data) || typeof data.url !== 'string') throw new Error('Invalid login response.')
      const destination = new URL(data.url)
      if(destination.protocol !== 'https:') throw new Error('Invalid login response.')
      location.assign(destination.href)
    } catch(error) {if(this.current(request,this.loginRead) && !isAbortError(error)) this.showError(error)}
    finally {if(this.loginRead === request) {this.loginRead = undefined;this.update()}}
  }
  resultClick(event:MouseEvent) {if(eventElement(event.target)?.closest('summary a')) event.stopPropagation()}
  hide() {
    if(this.write) this.showError('Saving was interrupted and may have completed. You can save the same scan again after returning.')
    this.alive = false;this.clipboardGeneration++
    for(const request of [this.analysis,this.identityRead,this.write,this.loginRead]) request?.abort()
    this.analysis = this.identityRead = this.write = this.loginRead = undefined
    this.update();super.hide()
  }
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;this.update();void this.identity()}}
}
registerController('scans',ScansController)
