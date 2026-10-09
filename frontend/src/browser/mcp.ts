import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {eventElement} from './shared/dom'
import {isAbortError} from './shared/json'

const copyTargets:Readonly<Record<string,string>> = {http:'mcp-http-config',desktop:'mcp-desktop-config',curl:'mcp-curl-example'}
export class MCPController extends RequestComponent {
  private alive = false
  private outputs = new Map<HTMLElement,AbortController>()
  private copies = new Map<HTMLButtonElement,{generation:number;timer?:ReturnType<typeof setTimeout>}>()
  connect() {this.alive = true;this.filter()}
  filter() {
    const input = this.element.querySelector<HTMLInputElement>('#mcp-filter');if(!input) return
    const query = input.value.trim().toLowerCase();let total = 0
    for(const group of this.all<HTMLElement>('.mcp-group')) {
      let count = 0
      for(const tool of group.querySelectorAll<HTMLDetailsElement>('.mcp-tool')) {tool.hidden = Boolean(query && !(tool.dataset.search ?? '').toLowerCase().includes(query));if(!tool.hidden) {count++;if(tool.open) void this.output(tool)}}
      group.hidden = count === 0
      const link = [...this.all<HTMLElement>('[data-mcp-nav]')].find(link=>link.dataset.mcpNav === group.id)
      if(link) {link.hidden = count === 0;const countNode = link.querySelector('span:last-child');if(countNode) countNode.textContent = String(count)}
      total += count
    }
    this.q('#mcp-tool-count').textContent = String(total);this.q<HTMLElement>('#mcp-no-nav-results').hidden = total !== 0
  }
  toggle(event:Event) {
    const tool = eventElement(event.target)
    if(!(tool instanceof HTMLDetailsElement) || !tool.matches('.mcp-tool')) return
    if(tool.open) {void this.output(tool);return}
    const output = tool.querySelector<HTMLElement>('[data-mcp-output]');if(output) this.cancel(output)
  }
  private cancel(output:HTMLElement) {const request = this.outputs.get(output);if(!request) return;request.abort();this.outputs.delete(output);delete output.dataset.loaded;output.removeAttribute('aria-busy')}
  private async output(tool:HTMLDetailsElement) {
    const output = tool.querySelector<HTMLElement>('[data-mcp-output]'),name = output?.dataset.mcpOutput
    if(!this.alive || !output || !name || output.dataset.loaded === 'true' || this.outputs.has(output)) return
    const request = new AbortController();this.outputs.set(output,request);output.dataset.loaded = 'loading';output.setAttribute('aria-busy','true');output.textContent = 'Loading output schema…'
    try {
      const html = await this.text('/_fragments/mcp/output/'+encodeURIComponent(name),'Could not load output schema.',{signal:request.signal})
      if(!this.alive || this.outputs.get(output) !== request || !tool.open) return
      output.innerHTML = html;output.dataset.loaded = 'true'
    } catch(error) {
      if(this.alive && this.outputs.get(output) === request && !isAbortError(error)) {delete output.dataset.loaded;const retry = document.createElement('button');retry.type = 'button';retry.dataset.mcpRetry = '';retry.textContent = 'Retry';output.replaceChildren('Could not load output schema. ',retry)}
    } finally {if(this.outputs.get(output) === request) {this.outputs.delete(output);output.removeAttribute('aria-busy')}}
  }
  async click(event:MouseEvent) {
    if(!this.alive) return
    const target = eventElement(event.target),retry = target?.closest('[data-mcp-retry]')
    if(retry) {const tool = retry.closest<HTMLDetailsElement>('.mcp-tool');if(tool) await this.output(tool);return}
    const button = target?.closest<HTMLButtonElement>('[data-copy]'),kind = button?.dataset.copy
    if(!button || !kind || !Object.hasOwn(copyTargets,kind)) return
    const value = this.element.querySelector<HTMLElement>('#'+copyTargets[kind])?.textContent;if(!value) return
    const previous = this.copies.get(button);this.stopTimer(previous?.timer);const generation = (previous?.generation ?? 0)+1;this.copies.set(button,{generation})
    let label = 'Copied!'
    try {await navigator.clipboard.writeText(value)} catch {label = 'Copy failed'}
    if(!this.alive || !button.isConnected || this.copies.get(button)?.generation !== generation) return
    button.textContent = label;const timer = this.after(1500,()=>button.textContent = 'Copy');this.copies.set(button,{generation,timer})
  }
  hide() {this.alive = false;for(const output of [...this.outputs.keys()]) this.cancel(output);for(const [button,copy] of this.copies) {this.stopTimer(copy.timer);button.textContent = 'Copy';this.copies.set(button,{generation:copy.generation+1})}super.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;this.filter()}}
}
registerController('mcp',MCPController)
