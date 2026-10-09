// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {SettingsDomainsController} from '../src/browser/settings-domains'
import {registerController} from '../src/browser/controllers/application'
import {domainEntity,domainDraft,readDomainSeed} from '../src/browser/shared/domain-seeds'

const markup=readFileSync(resolve(import.meta.dirname,'fixtures/settings/domains.html'),'utf8')
const settle=()=>new Promise<void>(resolve=>setTimeout(resolve,0)),errors=vi.fn()
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function q<T extends Element=HTMLElement>(root:ParentNode,selector:string):T{const node=root.querySelector<T>(selector);if(!node)throw Error(`Missing domain fixture element ${selector}`);return node}
function mount(edit=false){
  history.replaceState({},'','/settings/domains'+(edit?'?edit=9104':''));document.body.innerHTML=markup
  const root=q(document,'[data-settings-domains]'),domains=readDomainSeed(root.dataset.domains),first=domains.find(domain=>domain.id===9104)
  if(!first)throw Error('Missing fixture domain')
  return {root,domains,first}
}
function field(root:ParentNode,selector:string,value:string){const input=q<HTMLInputElement|HTMLTextAreaElement>(root,selector);input.value=value;input.dispatchEvent(new Event('input',{bubbles:true}));return input}
beforeEach(()=>{
  errors.mockClear();registerController('settings-domains',SettingsDomainsController).handleError=errors
  fetchMock=vi.fn<typeof fetch>(async path=>String(path).includes('/campaigns/search')?Response.json({campaigns:[]}):Response.json({}))
  vi.stubGlobal('fetch',fetchMock)
})
afterEach(async()=>{vi.useRealTimers();document.body.replaceChildren();await settle();expect(errors.mock.calls).toEqual([]);vi.unstubAllGlobals();vi.restoreAllMocks()})

test('domain seeds accept numeric and legacy prefixed search IDs and reject malformed entries',()=>{
  expect(domainEntity({id:42,type:'character',name:'Pilot'})).toEqual({id:42,type:'character',name:'Pilot'})
  expect(domainEntity({id:'corporation_99',type:'corporation',name:'Corp'})).toEqual({id:99,type:'corporation',name:'Corp'})
  for(const id of [NaN,0,-1,'character_NaN'])expect(domainEntity({id,type:'character',name:'Bad'})).toBeNull()
  expect(readDomainSeed('broken')).toEqual([]);expect(readDomainSeed('[{"id":1}]')).toEqual([])
  const parsed=domainDraft({id:1,subdomain:'typed-fixture',theme:{bannerUrl:42,showLogoInBanner:'yes',defaultThemeOverrides:{brandPrimary:'#123456',bad:42}},widgets:{top:[],left:[],right:[{type:'campaigns',enabled:true,campaignId:'Fixture01',title:'Original title'}]}})!
  expect(parsed.theme.bannerUrl).toBeUndefined();expect(parsed.theme.showLogoInBanner).toBeUndefined();expect(parsed.theme.defaultThemeOverrides).toEqual({brandPrimary:'#123456'})
  expect(parsed.widgets.right[0]).toMatchObject({campaignId:'Fixture01',title:'Original title'})
})
test('new domains debounce availability and submit validated numeric entity selections',async()=>{
  fetchMock.mockImplementation(async path=>String(path).includes('/check-subdomain')?Response.json({available:true}):String(path).startsWith('/api/search?')?Response.json({hits:[{id:42,type:'character',name:'<Pilot>'},{id:'bad',type:'alliance',name:'Invalid'}]}):String(path)==='/api/user/domains'?Response.json({domains:[]}):Response.json({}))
  const {root}=mount();await settle();q<HTMLButtonElement>(root,'[data-domain-create]').click();vi.useFakeTimers()
  field(root,'[data-domain-field="subdomain"]','typed-fixture');field(root,'[data-domain-entity-search]','Pilot')
  await vi.advanceTimersByTimeAsync(401);expect(root.querySelectorAll('[data-domain-entity-add]')).toHaveLength(1);expect(root.querySelector('pilot')).toBeNull()
  q<HTMLButtonElement>(root,'[data-domain-entity-add="0"]').click();expect(q<HTMLButtonElement>(root,'[data-domain-save]').disabled).toBe(false)
  q<HTMLButtonElement>(root,'[data-domain-save]').click();await vi.advanceTimersByTimeAsync(0)
  const create=fetchMock.mock.calls.find(([,options])=>options?.method==='POST')!
  expect(create[0]).toBe('/api/user/domains');expect(JSON.parse(String(create[1]!.body))).toMatchObject({subdomain:'typed-fixture',entities:[{id:42,type:'character',name:'<Pilot>'}]})
})
test('domain save rebases submitted values and retains a newer site-name draft for the next patch',async()=>{
  const {root,domains,first}=mount(true);await settle();await settle()
  let complete:((response:Response)=>void)|undefined
  fetchMock.mockImplementation(async(path,options)=>{
    if(options?.method==='PATCH')return new Promise(resolve=>complete=resolve)
    return Response.json(String(path)==='/api/user/domains'?{domains}:{campaigns:[]})
  })
  field(root,'[data-domain-field="site_name"]','Submitted name');q<HTMLButtonElement>(root,'[data-domain-save]').click()
  field(root,'[data-domain-field="site_name"]','Newer draft');complete!(Response.json({domain:{...first,site_name:'Submitted name'}}));await settle();await settle()
  expect(q<HTMLInputElement>(root,'[data-domain-field="site_name"]').value).toBe('Newer draft')
  q<HTMLButtonElement>(root,'[data-domain-save]').click()
  const patches=fetchMock.mock.calls.filter(([,options])=>options?.method==='PATCH')
  expect(JSON.parse(String(patches[0][1]!.body))).toEqual({site_name:'Submitted name'})
  expect(JSON.parse(String(patches[1][1]!.body))).toEqual({site_name:'Newer draft'})
  complete!(Response.json({domain:{...first,site_name:'Newer draft'}}));await settle()
})
test('nested navigation and widget edits keep their structure and managed image fields out of patches',async()=>{
  const {root,domains,first}=mount(true);await settle();await settle()
  fetchMock.mockImplementation(async(path,options)=>Response.json(options?.method==='PATCH'?{domain:first}:String(path)==='/api/user/domains'?{domains}:{campaigns:[]}))
  q<HTMLButtonElement>(root,'[data-domain-tab="navigation"]').click();q<HTMLButtonElement>(root,'[data-domain-nav-clear]').click();q<HTMLButtonElement>(root,'[data-domain-nav-add]').click()
  field(root,'[data-domain-nav-field="0:label"]','Custom');q<HTMLButtonElement>(root,'[data-domain-nav-items] [data-domain-nav-expand="0"]').click();q<HTMLButtonElement>(root,'[data-domain-nav-group-add="0"]').click();q<HTMLButtonElement>(root,'[data-domain-nav-child-add="0:0"]').click();field(root,'[data-domain-nav-field="0:0:0:label"]','My map');field(root,'[data-domain-nav-field="0:0:0:href"]','/map')
  q<HTMLButtonElement>(root,'[data-domain-tab="widgets"]').click();q<HTMLButtonElement>(root,'[data-domain-widget-menu="right"]').click();q<HTMLButtonElement>(root,'[data-domain-widget-add="right:textBlock"]').click()
  field(root,'textarea[data-domain-widget-field]','Typed text');q<HTMLButtonElement>(root,'[data-domain-save]').click();await settle()
  const patch=JSON.parse(String(fetchMock.mock.calls.find(([,options])=>options?.method==='PATCH')![1]!.body))
  expect(patch.navbar_links[0]).toMatchObject({label:'Custom',children:[{items:[{label:'My map',href:'/map'}]}]})
  expect(patch.widgets.right).toContainEqual({type:'textBlock',enabled:true,content:'Typed text'});expect(patch.theme?.bannerUrl).toBeUndefined()
})
test('campaign audience changes submit campaign IDs and their public selection together',async()=>{
  const {root,domains,first}=mount(true);first.campaigns=[{campaign_id:'Private01',name:'Private fixture',visibility:1,public_on_domain:false}];root.dataset.domains=JSON.stringify(domains)
  await settle();await settle();fetchMock.mockImplementation(async(path,options)=>Response.json(options?.method==='PATCH'?{domain:first}:String(path)==='/api/user/domains'?{domains}:{campaigns:[]}))
  q<HTMLButtonElement>(root,'[data-domain-tab="campaigns"]').click();q<HTMLButtonElement>(root,'[data-domain-campaign-audience="Private01"]').click();q<HTMLButtonElement>(root,'[data-domain-save]').click();await settle()
  const patch=JSON.parse(String(fetchMock.mock.calls.find(([,options])=>options?.method==='PATCH')![1]!.body))
  expect(patch).toEqual({campaign_ids:['Private01'],campaign_public_ids:['Private01']})
})
test('disconnect aborts pending searches and stale responses do not alter a retained domain draft',async()=>{
  const {root}=mount();await settle();q<HTMLButtonElement>(root,'[data-domain-create]').click();field(root,'[data-domain-field="site_name"]','Retained draft')
  const pending:Array<{signal:AbortSignal;complete:(response:Response)=>void}>=[];fetchMock.mockImplementation((_path,options)=>new Promise(resolve=>pending.push({signal:options!.signal!,complete:resolve})))
  vi.useFakeTimers();field(root,'[data-domain-entity-search]','Pilot');await vi.advanceTimersByTimeAsync(301)
  root.remove();await vi.advanceTimersByTimeAsync(0);expect(pending[0].signal.aborted).toBe(true)
  pending[0].complete(Response.json({hits:[{id:42,type:'character',name:'Late pilot'}]}));await vi.advanceTimersByTimeAsync(0)
  document.body.append(root);await vi.advanceTimersByTimeAsync(0);expect(q<HTMLInputElement>(root,'[data-domain-field="site_name"]').value).toBe('Retained draft');expect(root.textContent).not.toContain('Late pilot');expect(q(root,'[data-domain-error]').hidden).toBe(true)
})
test('page restoration restarts an interrupted subdomain check and rejects its canceled response',async()=>{
  const {root}=mount();await settle();q<HTMLButtonElement>(root,'[data-domain-create]').click()
  const pending:Array<{signal:AbortSignal;complete:(response:Response)=>void}>=[];fetchMock.mockImplementation((_path,options)=>new Promise(resolve=>pending.push({signal:options!.signal!,complete:resolve})))
  vi.useFakeTimers();field(root,'[data-domain-field="subdomain"]','typed-restored');await vi.advanceTimersByTimeAsync(401)
  window.dispatchEvent(new PageTransitionEvent('pagehide'));expect(pending[0].signal.aborted).toBe(true)
  // happy-dom aliases PageTransitionEvent to Event, so supply its browser field.
  const shown=new PageTransitionEvent('pageshow');Object.defineProperty(shown,'persisted',{value:true});window.dispatchEvent(shown);await vi.advanceTimersByTimeAsync(401);expect(pending).toHaveLength(2)
  pending[0].complete(Response.json({available:true}));pending[1].complete(Response.json({available:false,reason:'Already taken'}));await vi.advanceTimersByTimeAsync(0)
  expect(q(root,'[data-domain-subdomain-status]').textContent).toBe('Already taken');expect(q<HTMLButtonElement>(root,'[data-domain-save]').disabled).toBe(true)
})
test('image drops send multipart data, block duplicate uploads, abort on detach and remain retryable',async()=>{
  const {root}=mount(true);await settle();await settle();q<HTMLButtonElement>(root,'[data-domain-tab="appearance"]').click()
  let signal:AbortSignal|undefined,complete:((response:Response)=>void)|undefined
  fetchMock.mockImplementation((_path,options)=>new Promise(resolve=>{signal=options?.signal??undefined;complete=resolve}))
  const zone=q(root,'[data-domain-upload="background"]'),file=new File(['fixture'],'fixture.png',{type:'image/png'}),transfer=new DataTransfer();transfer.items.add(file)
  const drop=new DragEvent('drop',{bubbles:true,cancelable:true});Object.defineProperty(drop,'dataTransfer',{value:transfer});zone.dispatchEvent(drop);zone.dispatchEvent(drop)
  const uploads=fetchMock.mock.calls.filter(([,options])=>options?.method==='POST');expect(uploads).toHaveLength(1)
  const body=uploads[0][1]!.body;expect(body).toBeInstanceOf(FormData);if(!(body instanceof FormData))throw Error('Expected multipart upload');expect(body.get('type')).toBe('background');expect(body.get('file')).toBeInstanceOf(File)
  root.remove();await settle();expect(signal!.aborted).toBe(true);complete!(Response.json({assetId:999,status:'pending',message:'Accepted'}));await settle()
  document.body.append(root);await settle();expect(root.querySelector('img[src="/api/domains/preview/999"]')).toBeNull();expect(q(root,'[data-domain-error]').hidden).toBe(true)
  expect(q<HTMLButtonElement>(root,'[data-domain-upload="background"]').disabled).toBe(false)
})
