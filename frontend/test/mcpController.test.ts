// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {MCPController} from '../src/browser/mcp'
import {registerController} from '../src/browser/controllers/application'

const directory = resolve(import.meta.dirname,'fixtures/mcp')
function mountMarkup(name:string) {return readFileSync(resolve(directory,name+'.html'),'utf8').replace(/<script[^>]*type="module"[^>]*>[\s\S]*?<\/script>/g,'').replace(/<link[^>]*rel="stylesheet"[^>]*>/g,'')}
const page = mountMarkup('page')
const failure = mountMarkup('failure')
const output = readFileSync(resolve(directory,'output.html'),'utf8')
const errors = vi.fn(),settle = () => new Promise<void>(resolve=>setTimeout(resolve,0))
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function q<T extends Element = HTMLElement>(root:ParentNode,selector:string):T {const node = root.querySelector<T>(selector);if(!node) throw new Error('Missing MCP fixture '+selector);return node}
async function mount(markup = page) {document.body.innerHTML = markup;await settle();await settle();return q<HTMLElement>(document.body,'.mcp-page')}
function tool(root:HTMLElement,name = 'search') {return q(root,`[data-mcp-output="${name}"]`).closest<HTMLDetailsElement>('.mcp-tool')!}
function toggle(details:HTMLDetailsElement,open = true) {details.open = open;details.dispatchEvent(new Event('toggle'))}
function pendingRequests() {const pending:Array<{path:string;signal:AbortSignal;complete:(response:Response)=>void}> = [];fetchMock.mockImplementation((path,options)=>new Promise(resolve=>pending.push({path:String(path),signal:options!.signal!,complete:resolve})));return pending}
function hide() {window.dispatchEvent(new PageTransitionEvent('pagehide'))}
function show() {const event = new PageTransitionEvent('pageshow');Object.defineProperty(event,'persisted',{value:true});window.dispatchEvent(event)}
beforeEach(()=>{registerController('mcp',MCPController).handleError = errors;errors.mockClear();fetchMock = vi.fn<typeof fetch>(async()=>new Response(output));vi.stubGlobal('fetch',fetchMock)})
afterEach(async()=>{vi.useRealTimers();document.body.replaceChildren();await settle();expect(errors.mock.calls.map(call=>String(call[0]))).toEqual([]);vi.unstubAllGlobals();vi.restoreAllMocks()})

test('MCP keeps registered tool markup and filters groups/nav/counts without network reads',async()=>{
  const root = await mount(),total = root.querySelectorAll('.mcp-tool').length;expect(total).toBeGreaterThan(20);expect(q(root,'#mcp-tool-count').textContent).toBe(String(total));expect(fetchMock).not.toHaveBeenCalled()
  const input = q<HTMLInputElement>(root,'#mcp-filter');input.value = 'No fixture tool matches this';input.dispatchEvent(new Event('input',{bubbles:true}));expect(q(root,'#mcp-tool-count').textContent).toBe('0');expect(q<HTMLElement>(root,'#mcp-no-nav-results').hidden).toBe(false)
  expect([...root.querySelectorAll<HTMLElement>('.mcp-group,[data-mcp-nav]')].every(node=>node.hidden)).toBe(true)
  input.value = '';input.dispatchEvent(new Event('input',{bubbles:true}));expect(q(root,'#mcp-tool-count').textContent).toBe(String(total))
})
test('output schemas load once on opening and cached HTML survives closing/reopening',async()=>{
  const root = await mount(),details = tool(root);toggle(details);toggle(details);await settle();expect(fetchMock).toHaveBeenCalledOnce();expect(fetchMock.mock.calls[0][0]).toBe('/_fragments/mcp/output/search')
  const schema = q(details,'[data-mcp-output] pre');expect(schema.textContent).toContain('fixture');toggle(details,false);toggle(details);await settle();expect(q(details,'pre')).toBe(schema);expect(fetchMock).toHaveBeenCalledOnce();expect(root.hasAttribute('aria-busy')).toBe(false)
})
test('closing a pending schema cancels its read and reopening accepts only the new body',async()=>{
  const root = await mount(),details = tool(root),pending = pendingRequests();toggle(details);toggle(details,false);expect(pending[0].signal.aborted).toBe(true);toggle(details);expect(pending).toHaveLength(2)
  pending[1].complete(new Response(output));pending[0].complete(new Response('<pre>Stale schema</pre>'));await settle();expect(details.textContent).not.toContain('Stale schema');expect(q(details,'[data-mcp-output]').dataset.loaded).toBe('true')
})
test('schema failures provide a delegated retry without duplicating requests',async()=>{
  const root = await mount(),details = tool(root),pending = pendingRequests();toggle(details);pending[0].complete(new Response('Failed',{status:503}));await settle();expect(details.textContent).toContain('Could not load output schema')
  const retry = q<HTMLButtonElement>(details,'[data-mcp-retry]');retry.click();retry.click();expect(pending).toHaveLength(2);pending[1].complete(new Response(output));await settle();expect(details.querySelector('[data-mcp-retry]')).toBeNull()
})
test('late schema text is ignored after detach and does not leave busy state',async()=>{
  const root = await mount(),details = tool(root);let complete:(text:string)=>void = ()=>{};const response = new Response('');vi.spyOn(response,'text').mockImplementation(()=>new Promise(resolve=>complete = resolve));fetchMock.mockResolvedValue(response)
  toggle(details);await settle();root.remove();await settle();complete(output);await settle();expect(details.querySelector('pre')).toBeNull();expect(root.hasAttribute('aria-busy')).toBe(false);expect(q(details,'[data-mcp-output]').hasAttribute('aria-busy')).toBe(false)
})
test('page return resumes an open schema read once and retains its filter',async()=>{
  const root = await mount(),details = tool(root),pending = pendingRequests(),input = q<HTMLInputElement>(root,'#mcp-filter');input.value = 'search';input.dispatchEvent(new Event('input',{bubbles:true}));toggle(details);hide();show();show()
  expect(pending[0].signal.aborted).toBe(true);expect(pending).toHaveLength(2);expect(input.value).toBe('search');pending[1].complete(new Response(output));await settle();expect(q(details,'[data-mcp-output]').dataset.loaded).toBe('true')
})
test('each connection configuration copies its real Go-rendered text and resets feedback',async()=>{
  const root = await mount(),copy = vi.spyOn(navigator.clipboard,'writeText').mockResolvedValue();vi.useFakeTimers()
  for(const [kind,id] of [['http','mcp-http-config'],['desktop','mcp-desktop-config'],['curl','mcp-curl-example']]) {const button = q<HTMLButtonElement>(root,`[data-copy="${kind}"]`);button.click();await vi.advanceTimersByTimeAsync(0);expect(copy).toHaveBeenLastCalledWith(q(root,'#'+id).textContent);expect(button.textContent).toBe('Copied!');await vi.advanceTimersByTimeAsync(1501);expect(button.textContent).toBe('Copy')}
})
test('late clipboard completion cannot update a returned page or overwrite a newer click',async()=>{
  const root = await mount(),button = q<HTMLButtonElement>(root,'[data-copy="http"]'),completions:Array<()=>void> = [];vi.spyOn(navigator.clipboard,'writeText').mockImplementation(()=>new Promise(resolve=>completions.push(resolve)))
  button.click();hide();show();completions[0]();await settle();expect(button.textContent).toBe('Copy')
  button.click();button.click();completions[2]();await settle();expect(button.textContent).toBe('Copied!');completions[1]();await settle();expect(button.textContent).toBe('Copied!');hide();expect(button.textContent).toBe('Copy')
})
test('copy failures are visible and reset normally, including the catalog error view',async()=>{
  const root = await mount(failure),copy = vi.spyOn(navigator.clipboard,'writeText').mockRejectedValue(new Error('Denied'));vi.useFakeTimers();const button = q<HTMLButtonElement>(root,'[data-copy="http"]');button.click();await vi.advanceTimersByTimeAsync(0);expect(copy).toHaveBeenCalledOnce();expect(button.textContent).toBe('Copy failed');await vi.advanceTimersByTimeAsync(1501);expect(button.textContent).toBe('Copy');expect(root.textContent).toContain('Fixture catalog unavailable')
})
