// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {ScansController} from '../src/browser/scans'
import {registerController} from '../src/browser/controllers/application'
import {localScanNames,readScanAnalysis,readScanHash,scanSaveBody,type ScanKind} from '../src/browser/shared/scans'

const directory = resolve(import.meta.dirname,'fixtures/scans')
const fixtures = {
  dscan:readFileSync(resolve(directory,'dscan.html'),'utf8'),
  localscan:readFileSync(resolve(directory,'localscan.html'),'utf8'),
}
const results = {
  dscan:JSON.parse(readFileSync(resolve(directory,'dscan.json'),'utf8')) as unknown,
  localscan:JSON.parse(readFileSync(resolve(directory,'localscan.json'),'utf8')) as unknown,
}
const fragments = {
  dscan:readFileSync(resolve(directory,'dscan-results.html'),'utf8'),
  localscan:readFileSync(resolve(directory,'localscan-results.html'),'utf8'),
}
const hash = '0123456789abcdef'.repeat(4),errors = vi.fn()
const settle = () => new Promise<void>(resolve=>setTimeout(resolve,0))
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function q<T extends Element = HTMLElement>(root:ParentNode,selector:string):T {
  const element = root.querySelector<T>(selector)
  if(!element) throw new Error('Missing scan fixture '+selector)
  return element
}
async function mount(kind:ScanKind = 'dscan') {
  document.body.innerHTML = fixtures[kind];await settle();await settle()
  return q<HTMLElement>(document.body,'[data-scan-kind]')
}
function paste(text:string,target:EventTarget = document) {
  const clipboardData = new DataTransfer();clipboardData.setData('text/plain',text)
  const event = new ClipboardEvent('paste',{clipboardData,bubbles:true,cancelable:true})
  target.dispatchEvent(event);return event
}
async function analyze(root:HTMLElement,text = '1\tObject\tAstero\t10 km') {paste(text,root);await settle();await settle()}
function pendingRequests() {
  const pending:Array<{path:string;options:RequestInit;signal:AbortSignal;complete:(response:Response)=>void}> = []
  fetchMock.mockImplementation((path,options)=>new Promise(resolve=>pending.push({path:String(path),options:options!,signal:options!.signal!,complete:resolve})))
  return pending
}
function hide() {window.dispatchEvent(new PageTransitionEvent('pagehide'))}
function show() {const event = new PageTransitionEvent('pageshow');Object.defineProperty(event,'persisted',{value:true});window.dispatchEvent(event)}
beforeEach(()=>{
  registerController('scans',ScansController).handleError = errors;errors.mockClear()
  history.replaceState(null,'','/tools/dscan')
  fetchMock = vi.fn<typeof fetch>(async path=>{
    const url = String(path),kind = url.includes('localscan') ? 'localscan' : 'dscan'
    if(url === '/auth/me') return Response.json({user:{characterId:4242}})
    if(url.startsWith('/auth/login?')) return Response.json({url:'https://login.eveonline.com/v2/oauth/authorize?fixture=1'})
    if(url.endsWith('/save')) return Response.json({hash})
    return url.startsWith('/_fragments/') ? new Response(fragments[kind]) : Response.json(results[kind])
  })
  vi.stubGlobal('fetch',fetchMock)
})
afterEach(async()=>{
  document.body.replaceChildren();await settle()
  expect(errors.mock.calls.map(call=>String(call[0]))).toEqual([])
  vi.unstubAllGlobals();vi.restoreAllMocks()
})

test('D-Scan forwards typed analysis to Go and saves its original input/result',async()=>{
  const root = await mount(),assign = vi.spyOn(location,'assign').mockImplementation(()=>{}),text = '1\tObject\tAstero\t10 km'
  await analyze(root,text)
  expect(q(root,'[data-scan-results]').textContent).toContain('Astero')
  const analyzeCall = fetchMock.mock.calls.find(([path])=>path === '/api/tools/dscan')!
  expect(JSON.parse(String(analyzeCall[1]!.body))).toEqual({dscan:text})
  const renderCall = fetchMock.mock.calls.find(([path])=>path === '/_fragments/scans/dscan')!
  expect(JSON.parse(String(renderCall[1]!.body))).toEqual(results.dscan)
  q<HTMLButtonElement>(root,'[data-scan-save]').click();await settle()
  const saveCall = fetchMock.mock.calls.find(([path])=>path === '/api/tools/dscan/save')!
  expect(JSON.parse(String(saveCall[1]!.body))).toEqual({dscan:text,result:results.dscan})
  expect(assign).toHaveBeenCalledWith('/tools/dscan/'+hash)
  expect(root.hasAttribute('aria-busy')).toBe(false)
})
test('Local Scan trims valid names, preserves order and keeps Go details/entity links',async()=>{
  const root = await mount('localscan'),assign = vi.spyOn(location,'assign').mockImplementation(()=>{})
  await analyze(root,' Fixture Red Pilot\r\n\nFixture Blue Pilot\n'+'x'.repeat(65))
  const analyzeCall = fetchMock.mock.calls.find(([path])=>path === '/api/tools/localscan')!
  expect(JSON.parse(String(analyzeCall[1]!.body))).toEqual(['Fixture Red Pilot','Fixture Blue Pilot'])
  const details = q<HTMLDetailsElement>(root,'[data-scan-alliance]'),link = q<HTMLAnchorElement>(details,'summary a'),seen = vi.fn()
  document.addEventListener('click',seen,{once:true});link.addEventListener('click',event=>event.preventDefault(),{once:true});link.click()
  expect(seen).not.toHaveBeenCalled();document.removeEventListener('click',seen)
  expect(details.open).toBe(false);expect(link.getAttribute('href')).toBe('/alliance/99003581')
  q<HTMLButtonElement>(root,'[data-scan-save]').click();await settle()
  const saveCall = fetchMock.mock.calls.find(([path])=>path === '/api/tools/localscan/save')!
  expect(JSON.parse(String(saveCall[1]!.body))).toEqual({names:['Fixture Red Pilot','Fixture Blue Pilot'],result:results.localscan})
  expect(assign).toHaveBeenCalledWith('/tools/localscan/'+hash)
})
test('public analysis works without a session and exposes login only after results',async()=>{
  fetchMock.mockImplementation(async path=>String(path) === '/auth/me' ? Response.json({user:null}) : String(path).startsWith('/_fragments/') ? new Response(fragments.dscan) : Response.json(results.dscan))
  const root = await mount();expect(q<HTMLElement>(root,'[data-scan-actions]').hidden).toBe(true)
  await analyze(root);expect(q<HTMLElement>(root,'[data-scan-actions]').hidden).toBe(false)
  expect(q<HTMLButtonElement>(root,'[data-scan-save]').hidden).toBe(true);expect(q<HTMLElement>(root,'[data-scan-login-hint]').hidden).toBe(false)
})
test('editable and empty pastes preserve native behavior and do not request analysis',async()=>{
  const root = await mount(),input = document.createElement('textarea');root.append(input);fetchMock.mockClear()
  expect(paste('Native text',input).defaultPrevented).toBe(false)
  const editable = document.createElement('div');editable.setAttribute('contenteditable','');root.append(editable)
  expect(paste('Native text',editable).defaultPrevented).toBe(false);expect(paste('',root).defaultPrevented).toBe(false)
  expect(fetchMock).not.toHaveBeenCalled()
})
test('clipboard failures and empty scans show actionable errors without disabling the page',async()=>{
  const root = await mount();vi.spyOn(navigator.clipboard,'readText').mockRejectedValue(new Error('Denied'))
  q<HTMLButtonElement>(root,'[data-scan-paste]').click();await settle()
  expect(q(root,'[data-scan-error]').textContent).toContain('Ctrl+V')
  vi.spyOn(navigator.clipboard,'readText').mockResolvedValue(' ');q<HTMLButtonElement>(root,'[data-scan-paste]').click();await settle()
  expect(q(root,'[data-scan-error]').textContent).toContain('Clipboard is empty');expect(q<HTMLButtonElement>(root,'[data-scan-paste]').disabled).toBe(false)
  const local = await mount('localscan');vi.spyOn(navigator.clipboard,'readText').mockResolvedValue('x'.repeat(65))
  q<HTMLButtonElement>(local,'[data-scan-paste]').click();await settle();expect(q(local,'[data-scan-error]').textContent).toContain('No valid character names')
})
test('a delayed clipboard read cannot replace a newer keyboard paste or a returned page',async()=>{
  const root = await mount();let complete:(text:string)=>void = ()=>{}
  vi.spyOn(navigator.clipboard,'readText').mockImplementation(()=>new Promise(resolve=>complete = resolve))
  q<HTMLButtonElement>(root,'[data-scan-paste]').click();await analyze(root,'Keyboard scan');complete('Stale clipboard');await settle()
  expect(fetchMock.mock.calls.filter(([path])=>path === '/api/tools/dscan')).toHaveLength(1)
  q<HTMLButtonElement>(root,'[data-scan-paste]').click();hide();show();complete('Old document clipboard');await settle()
  expect(fetchMock.mock.calls.filter(([path])=>path === '/api/tools/dscan')).toHaveLength(1)
})
test('new analysis supersedes old responses and cannot save the previous result while pending',async()=>{
  const root = await mount();await analyze(root);const pending = pendingRequests()
  paste('First scan',root);paste('Latest scan',root)
  expect(pending[0].signal.aborted).toBe(true);expect(q<HTMLButtonElement>(root,'[data-scan-save]').disabled).toBe(true)
  pending[1].complete(Response.json(results.dscan));await settle();expect(pending[2].path).toBe('/_fragments/scans/dscan')
  pending[2].complete(new Response(fragments.dscan.replaceAll('Astero','Latest ship')));await settle()
  pending[0].complete(Response.json(results.dscan));await settle()
  expect(q(root,'[data-scan-results]').textContent).toContain('Latest ship');expect(pending).toHaveLength(3)
  expect(q<HTMLButtonElement>(root,'[data-scan-save]').disabled).toBe(false);expect(root.hasAttribute('aria-busy')).toBe(false)
})
test('analysis and malformed fragment failures retain the previous usable result and allow retry',async()=>{
  const root = await mount();await analyze(root);const content = q(root,'[data-scan-result-content]')
  fetchMock.mockResolvedValue(Response.json({error:'Fixture analysis failure'},{status:503}));paste('New scan',root);await settle()
  expect(q(root,'[data-scan-result-content]')).toBe(content);expect(q(root,'[data-scan-error]').textContent).toContain('Fixture analysis failure')
  fetchMock.mockImplementation(async path=>String(path).startsWith('/_fragments/') ? new Response('<div>Invalid fragment</div>') : Response.json(results.dscan))
  paste('New scan',root);await settle();await settle();expect(q(root,'[data-scan-result-content]')).toBe(content)
  expect(q(root,'[data-scan-error]').textContent).toContain('display scan results')
  fetchMock.mockImplementation(async path=>String(path).startsWith('/_fragments/') ? new Response(fragments.dscan) : Response.json(results.dscan))
  await analyze(root);expect(q(root,'[data-scan-result-content]')).not.toBe(content);expect(q<HTMLElement>(root,'[data-scan-error]').hidden).toBe(true)
})
test('malformed nested analysis is rejected before Go rendering',async()=>{
  const root = await mount();fetchMock.mockClear();fetchMock.mockResolvedValue(Response.json({grouped:{Ship:{categoryId:6,groups:{Bad:{groupId:25,types:[{count:'bad'}]}}}},totalCount:1,uniqueTypes:1}))
  paste('New scan',root);await settle();expect(fetchMock).toHaveBeenCalledTimes(1)
  expect(q(root,'[data-scan-error]').textContent).toContain('read scan results');expect(q<HTMLElement>(root,'[data-scan-actions]').hidden).toBe(true)
})
test('late fragment bodies cannot paint after replacement or leave loading indicators',async()=>{
  const root = await mount();let complete:(text:string)=>void = ()=>{},response = new Response('')
  vi.spyOn(response,'text').mockImplementation(()=>new Promise(resolve=>complete = resolve))
  fetchMock.mockImplementation(async path=>String(path).startsWith('/_fragments/') ? response : Response.json(results.dscan))
  paste('Scan',root);await settle();root.remove();await settle();complete(fragments.dscan);await settle()
  expect(root.querySelector('[data-scan-result-content]')).toBeNull();expect(root.hasAttribute('aria-busy')).toBe(false)
  expect(q<HTMLButtonElement>(root,'[data-scan-paste]').disabled).toBe(false)
})
test('save is single flight, rejects new analysis and validates the returned hash',async()=>{
  const root = await mount();await analyze(root);const assign = vi.spyOn(location,'assign').mockImplementation(()=>{}),pending = pendingRequests()
  q<HTMLButtonElement>(root,'[data-scan-save]').click();q<HTMLButtonElement>(root,'[data-scan-save]').click();paste('Other scan',root)
  expect(pending).toHaveLength(1);expect(q(root,'[data-scan-save-label]').textContent).toBe('Saving...')
  pending[0].complete(Response.json({hash:'../unsafe'}));await settle();expect(assign).not.toHaveBeenCalled()
  expect(q(root,'[data-scan-error]').textContent).toContain('Failed to save scan');expect(q<HTMLButtonElement>(root,'[data-scan-save]').disabled).toBe(false)
})
test('expired save aborts an older identity read and retains results for login',async()=>{
  const root = await mount();await analyze(root);const content = q(root,'[data-scan-result-content]'),pending = pendingRequests()
  window.dispatchEvent(new Event('focus'));q<HTMLButtonElement>(root,'[data-scan-save]').click()
  pending[1].complete(Response.json({error:'Unauthorized'},{status:401}));await settle();expect(pending[0].signal.aborted).toBe(true)
  pending[0].complete(Response.json({user:{characterId:4242}}));await settle()
  expect(q(root,'[data-scan-result-content]')).toBe(content);expect(q<HTMLButtonElement>(root,'[data-scan-save]').hidden).toBe(true)
  expect(q(root,'[data-scan-error]').textContent).toContain('session expired')
})
test('page return never repeats a save or analysis and reconciles identity with a read',async()=>{
  const root = await mount();await analyze(root);const assign = vi.spyOn(location,'assign').mockImplementation(()=>{}),pending = pendingRequests()
  q<HTMLButtonElement>(root,'[data-scan-save]').click();hide();show()
  expect(pending[0].signal.aborted).toBe(true);expect(pending[1].path).toBe('/auth/me');expect(pending[1].options.method??'GET').toBe('GET')
  pending[0].complete(Response.json({hash}));pending[1].complete(Response.json({user:{characterId:4242}}));await settle()
  expect(assign).not.toHaveBeenCalled();expect(pending).toHaveLength(2);expect(q(root,'[data-scan-error]').textContent).toContain('may have completed')
  expect(root.hasAttribute('aria-busy')).toBe(false);expect(q<HTMLButtonElement>(root,'[data-scan-save]').disabled).toBe(false)
})
test('login is owned, checks HTTPS and preserves the current redirect',async()=>{
  const root = await mount(),assign = vi.spyOn(location,'assign').mockImplementation(()=>{}),pending = pendingRequests()
  q<HTMLButtonElement>(root,'[data-scan-login]').click();q<HTMLButtonElement>(root,'[data-scan-login]').click()
  expect(pending).toHaveLength(1);expect(pending[0].path).toBe('/auth/login?redirect=%2Ftools%2Fdscan')
  pending[0].complete(Response.json({url:'javascript:alert(1)'}));await settle();expect(assign).not.toHaveBeenCalled()
  expect(q(root,'[data-scan-error]').textContent).toContain('Invalid login')
  q<HTMLButtonElement>(root,'[data-scan-login]').click();hide();pending[1].complete(Response.json({url:'https://login.eveonline.com/'}));await settle()
  expect(assign).not.toHaveBeenCalled();expect(q<HTMLButtonElement>(root,'[data-scan-login]').disabled).toBe(false)
})
test('typed boundaries preserve counters/null type IDs and reject unsafe result/hash shapes',()=>{
  const unknownType = {grouped:{Unknown:{categoryId:null,groups:{Unknown:{groupId:null,types:[{typeName:'Unknown object',typeId:null,count:2}]}}}},totalCount:2,uniqueTypes:1}
  expect(scanSaveBody(readScanAnalysis('dscan','Raw scan',unknownType))).toEqual({dscan:'Raw scan',result:unknownType})
  expect(localScanNames(' Alice\r\nBob\nAlice\n'+'x'.repeat(65))).toEqual(['Alice','Bob','Alice'])
  expect(()=>readScanAnalysis('localscan','Alice',{alliances:[],corporations:{},unresolved:[],totalCharacters:0,totalDangerous:0})).toThrow('read scan results')
  expect(()=>readScanHash({hash:'a'.repeat(63)})).toThrow('Failed to save');expect(readScanHash({hash})).toBe(hash)
})
