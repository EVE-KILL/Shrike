import {RequestComponent} from './request-component'
import {registerController} from './application'
import {docsElement as element} from '../shared/docs-dom'
import {eventElement} from '../shared/dom'
import {errorMessage,isAbortError,isRecord} from '../shared/json'
import {buildUrl,buildCurl,type ApiEndpoint,type ApiParam} from '../../models/apiDocs'

const endpoints = new WeakMap<HTMLElement,ApiEndpoint>()
type Control = HTMLInputElement|HTMLSelectElement
export function docsEndpointCard(endpoint:ApiEndpoint):HTMLDetailsElement {
  const card = element('details','docs-endpoint'),summary = element('summary','docs-endpoint-summary')
  if(endpoint.operationId) card.id = 'operation-'+endpoint.operationId
  card.dataset.controller = 'docs-endpoint'
  card.dataset.action = 'toggle->docs-endpoint#toggle input->docs-endpoint#input change->docs-endpoint#input click->docs-endpoint#click focusout->docs-endpoint#blur pagehide@window->docs-endpoint#hide pageshow@window->docs-endpoint#show'
  summary.append(element('span','docs-method method-'+endpoint.method.toLowerCase(),endpoint.method))
  const title = element('span','docs-endpoint-title'),path = element('code','docs-endpoint-path')
  for(const part of endpoint.path.split(/(:[A-Za-z0-9_]+)/)) path.append(part.startsWith(':') ? element('span','docs-path-param',part) : document.createTextNode(part))
  title.append(path,element('span','docs-endpoint-summary-text',endpoint.summary));summary.append(title,element('span','docs-chevron','⌄'));card.append(summary)
  endpoints.set(card,endpoint);return card
}
export class DocsEndpointController extends RequestComponent {
  private alive = false
  private built = false
  private paths = new Map<string,Control>()
  private query = new Map<string,Control>()
  private runRequest:AbortController|undefined
  private timeout:ReturnType<typeof setTimeout>|undefined
  private writeUncertain = false
  private reader:ReadableStreamDefaultReader<Uint8Array>|undefined
  private suggestion:AbortController|undefined
  private suggestionTimer:ReturnType<typeof setTimeout>|undefined
  private suggestionInput:Control|undefined
  private suggestionGeneration = 0
  private clipboardGeneration = 0
  private get endpoint() {const endpoint = endpoints.get(this.element);if(!endpoint) throw new Error('Missing API endpoint model');return endpoint}
  connect() {this.alive = true;this.toggle()}
  toggle() {if(this.element instanceof HTMLDetailsElement && this.element.open && !this.built) this.build()}
  private controls(title:string,params:ApiParam[],target:Map<string,Control>,isPath:boolean,body:HTMLElement) {
    if(!params.length) return
    const section = element('section','docs-params');section.append(element('h4','',title))
    for(const param of params) {
      const row = element('div','docs-param-row'),label = element('label','docs-param-label'),field = element('div','docs-param-field')
      label.append(element('code','',(isPath ? ':' : '')+param.name));if(param.required) label.append(element('span','docs-required','*'))
      label.append(element('small','',param.entityType ? `${param.type} (${param.entityType})` : param.type))
      let control:Control
      if(param.type === 'enum' && param.values?.length) {control = element('select','docs-input');const empty = element('option','','(not set)');empty.value = '';control.append(empty);for(const value of param.values) {const option = element('option','',value);option.value = value;control.append(option)}}
      else {control = element('input','docs-input');control.type = param.type === 'int' ? 'number' : param.type === 'date' ? 'date' : 'text';control.placeholder = String(param.example ?? param.default ?? param.name)}
      control.setAttribute('aria-label',title+': '+param.name);control.dataset.docsParam = param.name
      if(param.entityType) control.dataset.docsEntity = param.entityType
      if(param.example != null && isPath) control.value = String(param.example);else if(param.default != null && !isPath) control.value = String(param.default)
      field.append(control)
      if(param.entityType) {const results = element('div','docs-entity-results');results.hidden = true;field.append(results)}
      if(param.description) field.append(element('p','docs-param-description',param.description))
      row.append(label,field);section.append(row);target.set(param.name,control)
    }
    body.append(section)
  }
  private build() {
    this.built = true;const endpoint = this.endpoint,body = element('div','docs-endpoint-body')
    if(endpoint.description) body.append(element('p','docs-description',endpoint.description))
    if(endpoint.returns) {const row = element('div','docs-returns');row.append(element('span','','Returns'),element('code','',endpoint.returns));body.append(row)}
    this.controls('Path parameters',endpoint.pathParams ?? [],this.paths,true,body);this.controls('Query parameters',endpoint.query ?? [],this.query,false,body)
    if(endpoint.body) {
      const section = element('section','docs-body-section');section.append(element('h4','','Request body'))
      for(const [name,field] of Object.entries(endpoint.body.fields)) {const row = element('p','docs-body-field');row.append(element('code','',name));if(field.required) row.append(element('span','docs-required','*'));row.append(element('span','',field.type));if(field.maxItems) row.append(element('span','',`(max ${field.maxItems})`));if(field.description) row.append(element('span','','— '+field.description));section.append(row)}
      const input = element('textarea','docs-body-input');input.rows = 8;input.spellcheck = false
      const template = Object.fromEntries(Object.entries(endpoint.body.fields).map(([name,field])=>[name,field.default ?? (field.type.endsWith('[]') ? [] : field.type === 'int' ? 0 : '')]))
      input.value = JSON.stringify(endpoint.body.example ?? template,null,2);section.append(input);body.append(section)
    }
    const request = element('section','docs-request'),preview = element('code','docs-request-url'),actions = element('div','docs-actions')
    request.append(element('h4','','Request'),preview)
    for(const [action,label,className] of [['run','Try it','docs-run'],['url','Copy URL',''],['curl','Copy curl','']]) {const button = element('button',className,label);button.type = 'button';button.dataset.docsAction = action;actions.append(button)}
    const issue = element('div','docs-request-error'),response = element('section','docs-response');issue.hidden = response.hidden = true;issue.setAttribute('role','alert')
    response.append(element('div','docs-response-heading'),element('pre','docs-response-text'));request.append(actions,issue,response);body.append(request);this.element.append(body);this.preview()
  }
  private values(controls:Map<string,Control>) {return Object.fromEntries([...controls].map(([name,control])=>[name,control.dataset.resolvedId || control.value.trim()]))}
  private url() {
    const paths = this.values(this.paths),query = this.values(this.query)
    const missing = [...this.paths].some(([name,control])=>!paths[name] || Boolean(control.dataset.docsEntity && !/^\d+$/.test(paths[name])))
    return {url:buildUrl('/api',this.endpoint,paths,query),paths,query,missing}
  }
  private preview() {this.q('.docs-request-url').textContent = this.endpoint.method+' '+this.url().url}
  private issue(message = '') {if(this.built) this.error('.docs-request-error',message)}
  input(event:Event) {
    const control = eventElement(event.target)
    if(!(control instanceof HTMLInputElement || control instanceof HTMLSelectElement) || !control.hasAttribute('data-docs-param')) return
    delete control.dataset.resolvedId;this.preview()
    if(control.dataset.docsEntity) this.search(control)
  }
  private cancelSuggestions() {this.suggestionGeneration++;this.stopTimer(this.suggestionTimer);this.suggestionTimer = undefined;this.suggestion?.abort();this.suggestion = undefined;this.suggestionInput = undefined;for(const results of this.all<HTMLElement>('.docs-entity-results')) results.hidden = true}
  private search(control:Control) {
    this.cancelSuggestions();const query = control.value.trim(),generation = this.suggestionGeneration
    const results = control.parentElement?.querySelector<HTMLElement>('.docs-entity-results');if(!results) return
    results.replaceChildren();if(/^\d+$/.test(query) || query.length < 2) return
    this.suggestionInput = control
    this.suggestionTimer = this.after(200,()=>{this.suggestionTimer = undefined;void this.suggestions(control,results,query,generation)})
  }
  private async suggestions(control:Control,results:HTMLElement,query:string,generation:number) {
    if(!this.alive || generation !== this.suggestionGeneration) return
    const request = new AbortController();this.suggestion = request
    try {
      const value = await this.request<unknown>('/api/search?'+new URLSearchParams({q:query,type:control.dataset.docsEntity ?? '',limit:'6'}),{signal:request.signal})
      if(!this.alive || this.suggestion !== request || generation !== this.suggestionGeneration || control.value.trim() !== query) return
      if(!isRecord(value) || !Array.isArray(value.hits)) return
      for(const hit of value.hits.slice(0,6)) {
        if(!isRecord(hit) || typeof hit.name !== 'string' || typeof hit.id !== 'string' && typeof hit.id !== 'number') continue
        const id = String(hit.id).replace(/^[a-z]+_/i,'');if(!/^\d+$/.test(id)) continue
        const button = element('button','',hit.name+' · '+id);button.type = 'button';button.dataset.docsHitId = id;button.dataset.docsHitName = hit.name;results.append(button)
      }
      results.hidden = !results.childElementCount
    } catch {/* Numeric IDs and ordinary request controls remain usable. */}
    finally {if(this.suggestion === request) this.suggestion = undefined}
  }
  blur(event:FocusEvent) {const control = eventElement(event.target);if(control instanceof HTMLInputElement || control instanceof HTMLSelectElement) {if(event.relatedTarget instanceof Node && control.parentElement?.contains(event.relatedTarget)) return;const generation = this.suggestionGeneration;this.after(150,()=>{if(this.suggestionInput === control && generation === this.suggestionGeneration) this.cancelSuggestions()})}}
  async click(event:MouseEvent) {
    if(!this.alive) return
    const target = eventElement(event.target),hit = target?.closest<HTMLButtonElement>('[data-docs-hit-id]')
    if(hit) {const control = hit.closest('.docs-param-field')?.querySelector<Control>('[data-docs-param]');if(control) {control.value = hit.dataset.docsHitName ?? '';control.dataset.resolvedId = hit.dataset.docsHitId;this.cancelSuggestions();this.preview()}return}
    const action = target?.closest<HTMLElement>('[data-docs-action]')?.dataset.docsAction
    if(action === 'run') {await this.run();return}
    if(action === 'url' || action === 'curl') {
      const generation = ++this.clipboardGeneration
      try {const current = this.url(),input = this.element.querySelector<HTMLTextAreaElement>('.docs-body-input')
        await navigator.clipboard.writeText(action === 'url' ? current.url : buildCurl('/api',this.endpoint,current.paths,current.query,undefined,input?.value))
      } catch {if(this.alive && generation === this.clipboardGeneration) this.issue('Could not copy. Select the request URL and copy it manually.')}
    }
  }
  private async run() {
    if(!this.alive || this.runRequest || this.writeUncertain) return
    this.issue();this.q<HTMLElement>('.docs-response').hidden = true;const current = this.url()
    if(current.missing) {this.issue('Fill in the required path parameters first.');return}
    const input = this.element.querySelector<HTMLTextAreaElement>('.docs-body-input');let body:unknown
    if(input) {try {body = JSON.parse(input.value || '{}')} catch {this.issue('Request body must be valid JSON.');return}}
    const request = new AbortController(),streaming = this.endpoint.path === '/feed/stream',start = performance.now();this.runRequest = request
    const run = this.q<HTMLButtonElement>('.docs-run');run.disabled = true;run.textContent = 'Loading…'
    this.timeout = this.after(streaming ? 10000 : 30000,()=>request.abort())
    try {
      const result = await this.response(current.url,{method:this.endpoint.method,signal:request.signal,cache:'no-store',...(input ? {headers:{'Content-Type':'application/json'},body:JSON.stringify(body)} : {})},async response=>{
        let text:string
        if(streaming && response.body) {
          this.stopTimer(this.timeout);const reader = response.body.getReader();this.reader = reader
          this.timeout = this.after(10000,()=>{void reader.cancel().catch(()=>{})})
          const chunks:Uint8Array[] = [];let size = 0
          try {while(size < 200000) {const next = await reader.read();if(next.done) break;const chunk = next.value.slice(0,200000-size);chunks.push(chunk);size += chunk.length}} catch {/* Keep the collected stream preview. */}
          finally {await reader.cancel().catch(()=>{});reader.releaseLock();if(this.reader === reader) this.reader = undefined}
          const bytes = new Uint8Array(size);let offset = 0;for(const chunk of chunks) {bytes.set(chunk,offset);offset += chunk.length}text = new TextDecoder().decode(bytes)
        } else text = await response.text()
        return {status:response.status,text}
      })
      if(!this.alive || this.runRequest !== request) return
      let data:unknown = result.text;try {data = JSON.parse(result.text)} catch {/* Plain text remains readable. */}
      let pretty = typeof data === 'string' ? data : JSON.stringify(data,null,2)
      if(pretty.length > 200000) pretty = pretty.slice(0,200000)+'\n\n… response truncated …'
      this.q('.docs-response-heading').replaceChildren(element('span','','Response'),element('strong',result.status < 300 ? 'success' : 'failure',result.status),element('span','',Math.round(performance.now()-start)+'ms'))
      this.q('.docs-response-text').textContent = pretty;this.q<HTMLElement>('.docs-response').hidden = false
    } catch(error) {
      if(this.alive && this.runRequest === request) {
        if(isAbortError(error) && this.endpoint.method !== 'GET') this.writeUncertain = true
        this.issue(isAbortError(error) ? 'Request timed out.'+(this.writeUncertain ? ' It may have completed. Verify before sending again.' : '') : errorMessage(error,'Request failed'))
      }
    } finally {if(this.runRequest === request) {this.runRequest = undefined;this.stopTimer(this.timeout);this.timeout = undefined;run.disabled = this.writeUncertain;run.textContent = 'Try it'}}
  }
  hide() {
    if(this.runRequest && this.endpoint.method !== 'GET') {this.writeUncertain = true;this.issue('Request was interrupted and may have completed. Verify before sending again.')}
    this.alive = false;this.clipboardGeneration++;this.runRequest?.abort();this.runRequest = undefined;this.stopTimer(this.timeout);this.timeout = undefined;void this.reader?.cancel().catch(()=>{});this.reader = undefined;this.cancelSuggestions()
    if(this.built) {const run = this.q<HTMLButtonElement>('.docs-run');run.disabled = this.writeUncertain;run.textContent = 'Try it'}
    super.hide()
  }
  show(event:PageTransitionEvent) {if(event.persisted) this.alive = true}
}
registerController('docs-endpoint',DocsEndpointController)
