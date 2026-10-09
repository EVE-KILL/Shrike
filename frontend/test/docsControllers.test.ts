// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {DocsController} from '../src/browser/docs'
import {DocsEndpointController} from '../src/browser/controllers/docs-endpoint'
import {registerController} from '../src/browser/controllers/application'
import {buildDocsModel} from '../src/models/openapiDocs'
import {buildUrl,buildCurl} from '../src/models/apiDocs'
import fullSpec from '../shared/api.openapi.json'

// Vitest imports the real controllers; happy-dom disables external module
// loading. Keep component markup while removing only its browser entry loader.
const markup = readFileSync(resolve(import.meta.dirname,'fixtures/docs/page.html'),'utf8').replace(/<script[^>]*type="module"[^>]*>[\s\S]*?<\/script>/g,'')
const spec = {info:{title:'Fixture API',version:'1'},tags:[{name:'one'},{name:'shared'},{name:'streams'}],'x-tagGroups':[{name:'Review',tags:['one','shared','streams']}],paths:{
  '/fixture/{slug}':{get:{operationId:'fixture-read',summary:'Read fixture',tags:['one','shared'],parameters:[{in:'path',name:'slug',required:true,schema:{type:'string'}},{in:'query',name:'mutual',schema:{type:'boolean'}},{in:'query',name:'q',schema:{type:'string'}}],responses:{200:{description:'OK'}}}},
  '/fixture':{post:{operationId:'fixture-write',summary:'Write fixture',tags:['one'],requestBody:{content:{'application/json':{schema:{type:'object',properties:{names:{type:'array',items:{type:'string'},maxItems:2}},required:['names']},example:{names:['Fixture Pilot']}}}},responses:{200:{description:'Created fixture'}}}},
  '/characters/{id}':{get:{operationId:'fixture-character',summary:'Read character',tags:['one'],parameters:[{in:'path',name:'id',required:true,schema:{type:'integer'}},{in:'query',name:'attacker','x-entity':'ship',schema:{type:'integer'}}],responses:{200:{description:'OK'}}}},
  '/feed/stream':{get:{operationId:'fixture-stream',summary:'Read stream',tags:['streams'],responses:{200:{description:'Feed stream'}}}},
}}
const errors = vi.fn(),settle = () => new Promise<void>(resolve=>setTimeout(resolve,0))
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function q<T extends Element = HTMLElement>(root:ParentNode,selector:string):T {const node = root.querySelector<T>(selector);if(!node) throw new Error('Missing docs fixture '+selector);return node}
async function mount() {document.body.innerHTML = markup;await settle();await settle();await settle();return q<HTMLElement>(document.body,'.docs-page')}
async function open(root:HTMLElement,id:string) {const card = q<HTMLDetailsElement>(root,'#operation-'+id);card.open = true;card.dispatchEvent(new Event('toggle'));await settle();return card}
function input(root:ParentNode,selector:string,value:string) {const field = q<HTMLInputElement|HTMLSelectElement|HTMLTextAreaElement>(root,selector);field.value = value;field.dispatchEvent(new InputEvent('input',{bubbles:true}));return field}
function pendingRequests() {const pending:Array<{path:string;options:RequestInit;signal:AbortSignal;complete:(response:Response)=>void}> = [];fetchMock.mockImplementation((path,options)=>new Promise(resolve=>pending.push({path:String(path),options:options!,signal:options!.signal!,complete:resolve})));return pending}
function hide() {window.dispatchEvent(new PageTransitionEvent('pagehide'))}
function show() {const event = new PageTransitionEvent('pageshow');Object.defineProperty(event,'persisted',{value:true});window.dispatchEvent(event)}
beforeEach(()=>{
  for(const [name,controller] of [['docs',DocsController],['docs-endpoint',DocsEndpointController]] as const) registerController(name,controller).handleError = errors
  errors.mockClear();fetchMock = vi.fn<typeof fetch>(async path=>String(path) === '/api/openapi.json' ? Response.json(spec) : Response.json({fixture:true}));vi.stubGlobal('fetch',fetchMock)
})
afterEach(async()=>{vi.useRealTimers();document.body.replaceChildren();await settle();expect(errors.mock.calls.map(call=>String(call[0]))).toEqual([]);vi.unstubAllGlobals();vi.restoreAllMocks()})

test('docs uses one OpenAPI request, counts unique operations and builds card bodies lazily',async()=>{
  const root = await mount();expect(fetchMock).toHaveBeenCalledOnce();expect(q(root,'#docs-count').textContent).toBe('4 of 4 endpoints');expect(q(root,'#docs-spec-title').textContent).toBe('Fixture API')
  expect(root.querySelectorAll('.docs-endpoint')).toHaveLength(5);expect(root.querySelectorAll('.docs-endpoint-body')).toHaveLength(0)
  const card = await open(root,'fixture-write');expect(q<HTMLTextAreaElement>(card,'.docs-body-input').value).toBe('{\n  "names": [\n    "Fixture Pilot"\n  ]\n}')
  expect(card.textContent).toContain('(max 2)');card.dispatchEvent(new Event('toggle'));expect(card.querySelectorAll('.docs-endpoint-body')).toHaveLength(1)
})
test('the entire generated API document renders every operation without parser drift',async()=>{
  fetchMock.mockResolvedValue(Response.json(fullSpec));const root = await mount(),model = buildDocsModel(fullSpec)
  expect(q(root,'#docs-count').textContent).toBe(`${model.total} of ${model.total} endpoints`);expect(root.querySelectorAll('.docs-endpoint')).toHaveLength(model.groups.flatMap(group=>group.tags.flatMap(tag=>tag.endpoints)).length)
  const card = await open(root,'dscan-analyze');expect(card.textContent).toContain('dscan');expect(q<HTMLTextAreaElement>(card,'.docs-body-input')).toBeTruthy()
})
test('doc load errors retry once and interrupted reads reconcile on page return',async()=>{
  fetchMock.mockResolvedValue(new Response('Failure',{status:503}));const root = await mount();expect(q<HTMLElement>(root,'#docs-error').hidden).toBe(false)
  const pending = pendingRequests();q<HTMLButtonElement>(root,'#docs-retry').click();q<HTMLButtonElement>(root,'#docs-retry').click();expect(pending).toHaveLength(1);hide();show();expect(pending[0].signal.aborted).toBe(true);expect(pending).toHaveLength(2)
  pending[1].complete(Response.json(spec));pending[0].complete(Response.json({info:{title:'Stale'},paths:{}}));await settle();await settle();expect(q(root,'#docs-spec-title').textContent).toBe('Fixture API');expect(root.hasAttribute('aria-busy')).toBe(false)
})
test('late OpenAPI JSON bodies cannot paint a detached page',async()=>{
  let complete:(value:unknown)=>void = ()=>{};const response = Response.json({});vi.spyOn(response,'json').mockImplementation(()=>new Promise(resolve=>complete = resolve));fetchMock.mockResolvedValue(response)
  const root = await mount();root.remove();await settle();complete(spec);await settle();expect(root.querySelector('.docs-endpoint')).toBeNull();expect(root.hasAttribute('aria-busy')).toBe(false)
})
test('filters preserve expanded field drafts across no-match detach/reconnect',async()=>{
  const root = await mount(),card = await open(root,'fixture-write');input(card,'.docs-body-input','{"names":["Newer draft"]}');vi.useFakeTimers()
  input(root,'#docs-search','POST');await vi.advanceTimersByTimeAsync(81);expect(q(root,'#docs-count').textContent).toBe('1 of 4 endpoints')
  input(root,'#docs-search','No matches');await vi.advanceTimersByTimeAsync(81);expect(q(root,'.docs-empty').textContent).toBe('No endpoints match')
  input(root,'#docs-search','');await vi.advanceTimersByTimeAsync(81);expect(q<HTMLTextAreaElement>(root,'#operation-fixture-write .docs-body-input').value).toContain('Newer draft');expect(card.open).toBe(true)
})
test('path/query preview validates missing IDs and encodes values with blank enums left out',async()=>{
  const root = await mount(),card = await open(root,'fixture-read');q<HTMLButtonElement>(card,'.docs-run').click();await settle();expect(q(card,'.docs-request-error').textContent).toContain('required path')
  expect(q(card,'.docs-param-field')).toBeTruthy();expect(q(card,'.docs-param-label code').textContent).toBe(':slug');expect(q(card,'select option').textContent).toBe('(not set)')
  input(card,'[aria-label="Path parameters: slug"]','A/B C');input(card,'[aria-label="Query parameters: q"]','x&y');expect(q(card,'.docs-request-url').textContent).toBe('GET /api/fixture/A%2FB%20C?q=x%26y')
  input(card,'[aria-label="Query parameters: mutual"]','true');q<HTMLButtonElement>(card,'.docs-run').click();await settle();expect(fetchMock.mock.calls.at(-1)![0]).toBe('/api/fixture/A%2FB%20C?mutual=true&q=x%26y');expect(fetchMock.mock.calls.at(-1)![1]?.cache).toBe('no-store')
})
test('requests are single flight and render HTTP errors as text while retaining controls',async()=>{
  const root = await mount(),card = await open(root,'fixture-read');input(card,'[data-docs-param="slug"]','Fixture');const pending = pendingRequests();q<HTMLButtonElement>(card,'.docs-run').click();q<HTMLButtonElement>(card,'.docs-run').click();expect(pending).toHaveLength(1)
  pending[0].complete(new Response('<script>Not executable</script>',{status:503}));await settle();expect(q(card,'.docs-response-heading .failure').textContent).toBe('503');expect(q(card,'.docs-response-text').textContent).toContain('<script>');expect(card.querySelector('script')).toBeNull();expect(q<HTMLInputElement>(card,'[data-docs-param="slug"]').value).toBe('Fixture');expect(q<HTMLButtonElement>(card,'.docs-run').disabled).toBe(false)
})
test('body requests send exact parsed JSON and retain edits made during the request',async()=>{
  const root = await mount(),card = await open(root,'fixture-write');const pending = pendingRequests();q<HTMLButtonElement>(card,'.docs-run').click();expect(pending[0].options.method).toBe('POST');expect(JSON.parse(String(pending[0].options.body))).toEqual({names:['Fixture Pilot']})
  input(card,'.docs-body-input','{"names":["Newer draft"]}');pending[0].complete(Response.json({saved:true}));await settle();expect(q<HTMLTextAreaElement>(card,'.docs-body-input').value).toContain('Newer draft');expect(q(card,'.docs-response-text').textContent).toContain('"saved": true')
  input(card,'.docs-body-input','bad JSON');q<HTMLButtonElement>(card,'.docs-run').click();expect(pending).toHaveLength(1);expect(q(card,'.docs-request-error').textContent).toContain('valid JSON')
})
test('entity suggestions supersede old reads and use picked IDs for path and query values',async()=>{
  const root = await mount(),card = await open(root,'fixture-character'),pending = pendingRequests();vi.useFakeTimers();const path = input(card,'[data-docs-param="id"]','First');path.focus();await vi.advanceTimersByTimeAsync(201)
  input(card,'[data-docs-param="id"]','Latest');await vi.advanceTimersByTimeAsync(201);expect(pending[0].signal.aborted).toBe(true)
  pending[1].complete(Response.json({hits:[{id:'character_42',name:'Fixture Pilot'}]}));await vi.advanceTimersByTimeAsync(0);pending[0].complete(Response.json({hits:[{id:'character_99',name:'Stale Pilot'}]}));await vi.advanceTimersByTimeAsync(0)
  q<HTMLButtonElement>(card,'[data-docs-hit-id="42"]').click();expect(q(card,'.docs-request-url').textContent).toBe('GET /api/characters/42');expect(card.textContent).not.toContain('Stale Pilot')
  input(card,'[data-docs-param="attacker"]','Rifter');await vi.advanceTimersByTimeAsync(201);pending[2].complete(Response.json({hits:[{id:'ship_587',name:'Rifter'}]}));await vi.advanceTimersByTimeAsync(0);q<HTMLButtonElement>(card,'[data-docs-hit-id="587"]').click();expect(q(card,'.docs-request-url').textContent).toBe('GET /api/characters/42?attacker=587')
})
test('late request bodies and page return cannot repeat a write or commit stale responses',async()=>{
  const root = await mount(),card = await open(root,'fixture-write');const pending = pendingRequests();q<HTMLButtonElement>(card,'.docs-run').click();hide();show();expect(pending[0].signal.aborted).toBe(true);pending[0].complete(Response.json({saved:true}));await settle()
  expect(pending).toHaveLength(1);expect(q<HTMLButtonElement>(card,'.docs-run').disabled).toBe(true);expect(q(card,'.docs-request-error').textContent).toContain('may have completed');expect(q<HTMLElement>(card,'.docs-response').hidden).toBe(true)
})
test('read requests cancel on detach without leaving a busy button when their card returns',async()=>{
  const root = await mount(),card = await open(root,'fixture-read');input(card,'[data-docs-param="slug"]','Fixture');let complete:(text:string)=>void = ()=>{};const response = new Response('');vi.spyOn(response,'text').mockImplementation(()=>new Promise(resolve=>complete = resolve));fetchMock.mockResolvedValue(response)
  q<HTMLButtonElement>(card,'.docs-run').click();await settle();const parent = card.parentElement!;card.remove();await settle();complete('Late response');await settle();parent.append(card);await settle()
  expect(q<HTMLElement>(card,'.docs-response').hidden).toBe(true);expect(q<HTMLButtonElement>(card,'.docs-run').disabled).toBe(false);expect(card.hasAttribute('aria-busy')).toBe(false)
})
test('request timeout aborts transport and leaves read requests retryable',async()=>{
  const root = await mount(),card = await open(root,'fixture-read');input(card,'[data-docs-param="slug"]','Fixture');vi.useFakeTimers();fetchMock.mockImplementation((_path,options)=>new Promise((_resolve,reject)=>options!.signal!.addEventListener('abort',()=>reject(new DOMException('Aborted','AbortError')))))
  q<HTMLButtonElement>(card,'.docs-run').click();await vi.advanceTimersByTimeAsync(30000);expect(q(card,'.docs-request-error').textContent).toBe('Request timed out.');expect(q<HTMLButtonElement>(card,'.docs-run').disabled).toBe(false)
})
test('stream previews cancel after ten seconds and show the collected text',async()=>{
  const root = await mount(),card = await open(root,'fixture-stream'),cancel = vi.fn();vi.useFakeTimers()
  fetchMock.mockResolvedValue(new Response(new ReadableStream<Uint8Array>({start(controller){controller.enqueue(new TextEncoder().encode('data: Fixture stream\n\n'))},cancel})))
  q<HTMLButtonElement>(card,'.docs-run').click();await vi.advanceTimersByTimeAsync(10000);expect(cancel).toHaveBeenCalledOnce();expect(q(card,'.docs-response-text').textContent).toContain('Fixture stream');expect(q<HTMLButtonElement>(card,'.docs-run').disabled).toBe(false);expect(card.hasAttribute('aria-busy')).toBe(false)
})
test('large stream previews stop at the byte limit and release the reader',async()=>{
  const root = await mount(),card = await open(root,'fixture-stream'),cancel = vi.fn();fetchMock.mockResolvedValue(new Response(new ReadableStream<Uint8Array>({start(controller){controller.enqueue(new Uint8Array(250000).fill(65))},cancel})))
  q<HTMLButtonElement>(card,'.docs-run').click();await settle();await settle();expect(q(card,'.docs-response-text').textContent).toHaveLength(200000);expect(cancel).toHaveBeenCalledOnce()
})
test('copy URL/curl uses encoded paths and safely quotes the JSON body',async()=>{
  const root = await mount(),read = await open(root,'fixture-read'),copy = vi.spyOn(navigator.clipboard,'writeText').mockResolvedValue();input(read,'[data-docs-param="slug"]','A/B');q<HTMLButtonElement>(read,'[data-docs-action="url"]').click();await settle();expect(copy).toHaveBeenLastCalledWith('/api/fixture/A%2FB')
  const write = await open(root,'fixture-write');input(write,'.docs-body-input','{"names":["Pilot\u0027s name"]}');q<HTMLButtonElement>(write,'[data-docs-action="curl"]').click();await settle();expect(copy.mock.calls.at(-1)![0]).toContain("Pilot'\\''s name")
  input(write,'.docs-body-input','Incomplete draft');q<HTMLButtonElement>(write,'[data-docs-action="url"]').click();await settle();expect(copy).toHaveBeenLastCalledWith('/api/fixture');q<HTMLButtonElement>(write,'[data-docs-action="curl"]').click();await settle();expect(copy.mock.calls.at(-1)![0]).toContain("-d 'Incomplete draft'")
})
test('OpenAPI references resolve parameters/bodies/responses and cyclic refs stay bounded',()=>{
  const doc = {components:{parameters:{ID:{in:'path',name:'id',schema:{type:'integer'}}},schemas:{'A/B':{type:'object',properties:{names:{type:'array',items:{type:'string'}}},example:{names:['Fixture']}},Cycle:{$ref:'#/components/schemas/Cycle'}},responses:{Created:{description:'A created record'}}},paths:{'/characters/{id}':{post:{parameters:[{$ref:'#/components/parameters/ID'}],requestBody:{content:{'application/json':{schema:{$ref:'#/components/schemas/A~1B'}}}},responses:{201:{$ref:'#/components/responses/Created'}}}},'/cycle':{get:{parameters:[{in:'query',name:'value',schema:{$ref:'#/components/schemas/Cycle'}}]}}}}
  const model = buildDocsModel(doc),endpoint = model.groups[0].tags[0].endpoints.find(endpoint=>endpoint.method === 'POST')!
  expect(endpoint.pathParams?.[0].entityType).toBe('character');expect(endpoint.body?.example).toEqual({names:['Fixture']});expect(endpoint.returns).toBe('201 — A created record');expect(model.total).toBe(2)
  expect(()=>buildDocsModel({paths:[]})).toThrow('API document');expect(buildDocsModel({paths:{bad:null,'/valid':{get:{parameters:[null,{}],tags:[42,'valid']}}},'x-tagGroups':[null,{name:'Broken',tags:42}]}).total).toBe(1)
})
test('compatibility URL placeholders and shared path parameters remain usable request controls',()=>{
  const model=buildDocsModel({paths:{
    '/character/{id}':{get:{summary:'Compatibility profile'}},
    '/characters/{id}':{parameters:[{in:'path',name:'id',required:true,schema:{type:'integer'}},{in:'query',name:'limit',schema:{type:'integer',default:10}}],get:{parameters:[{in:'query',name:'limit',schema:{type:'integer',default:25}}]}}
  }})
  const endpoints=model.groups.flatMap(group=>group.tags.flatMap(tag=>tag.endpoints))
  expect(endpoints.find(endpoint=>endpoint.path==='/character/:id')?.pathParams).toEqual([{name:'id',type:'entity',required:true,entityType:'character'}])
  expect(endpoints.find(endpoint=>endpoint.path==='/characters/:id')).toMatchObject({pathParams:[{name:'id',type:'entity',required:true,entityType:'character'}],query:[{name:'limit',default:25}]})
  const full=buildDocsModel(fullSpec)
  for(const endpoint of full.groups.flatMap(group=>group.tags.flatMap(tag=>tag.endpoints))) for(const match of endpoint.path.matchAll(/:([A-Za-z_][A-Za-z0-9_]*)/g)) {
    expect(endpoint.pathParams?.some(parameter=>parameter.name===match[1])).toBe(true)
  }
})
test('URL/curl helpers encode slashes and quotes without losing optional query values',()=>{
  const endpoint = {method:'GET',path:'/fixture/:slug',summary:'',query:[{name:'q',type:'string'}]} as const
  expect(buildUrl('/api',{...endpoint,query:[...endpoint.query]},{slug:'A/B'},{q:'a&b'})).toBe('/api/fixture/A%2FB?q=a%26b')
  expect(buildCurl('/api',{...endpoint,query:[...endpoint.query]},{slug:"Pilot's"},{},undefined)).toBe("curl '/api/fixture/Pilot'\\''s'")
})
