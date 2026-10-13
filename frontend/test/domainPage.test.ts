// @vitest-environment happy-dom
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {DomainPageController} from '../src/browser/domain-page'
import {registerController} from '../src/browser/controllers/application'
import {initializePrivateKillFeed} from '../src/browser/private-killfeed'
vi.mock('../src/browser/private-killfeed',()=>({initializePrivateKillFeed:vi.fn()}))
const settle=()=>new Promise<void>(resolve=>setTimeout(resolve,0))
const errors=vi.fn(),destroy=vi.fn()
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function mount(filtered=false){
 const host=document.createElement('div');host.dataset.controller='domain-page';host.dataset.action='click->domain-page#click pagehide@window->domain-page#hide pageshow@window->domain-page#show';host.dataset.domainEntities=JSON.stringify([{type:'corporation',id:42},{type:'character',id:43},{type:'corporation',id:42},{type:'system',id:44}]);
 host.innerHTML=`<div data-private-killfeed data-domain-killtype="solo"></div><div data-domain-widget="topCharacters" data-domain-key="top" ${filtered?'data-domain-killtype="solo"':''}><section class="top-box">Loading</section></div>`
 document.body.append(host);return host
}
beforeEach(()=>{
 errors.mockClear();destroy.mockClear();registerController('domain-page',DomainPageController).handleError=errors
 vi.mocked(initializePrivateKillFeed).mockClear();vi.mocked(initializePrivateKillFeed).mockReturnValue({destroy,refresh:vi.fn()})
 fetchMock=vi.fn<typeof fetch>(async()=>new Response('<section class="top-box">Scoped pilot</section>'));vi.stubGlobal('fetch',fetchMock)
})
afterEach(async()=>{document.body.replaceChildren();await settle();expect(errors.mock.calls).toEqual([]);vi.unstubAllGlobals()})
test('widgets defer independently and live topics contain only configured entities',async()=>{
 const host=mount();await settle();await settle()
 expect(host.querySelector('.top-box')!.textContent).toBe('Scoped pilot')
 expect(String(fetchMock.mock.calls[0][0])).toContain('/_fragments/domain/widgets/topCharacters')
 expect(vi.mocked(initializePrivateKillFeed).mock.calls[0][1]).toMatchObject({endpoint:'/_fragments/domain/killlist',topics:['victim.42','attacker.42','victim.43','attacker.43'],params:{type:'solo'}})
 host.remove();await settle();expect(destroy).toHaveBeenCalledOnce()
})
test('filtered pages preserve their kill type for top and valuable widgets',async()=>{
 const host=mount(true);await settle();await settle()
 expect(String(fetchMock.mock.calls[0][0])).toContain('type=solo')
 const valuable=document.createElement('div');valuable.dataset.domainWidget='valuable';valuable.dataset.domainKey='valuable';valuable.dataset.domainKilltype='solo';valuable.innerHTML='<button data-domain-retry>Retry</button>';host.append(valuable)
 fetchMock.mockResolvedValueOnce(new Response('<section class="kills-valuable">Scoped valuable kill</section>'))
 valuable.querySelector<HTMLButtonElement>('button')!.click();await settle();await settle()
 expect(String(fetchMock.mock.calls.at(-1)![0])).toContain('type=solo');expect(valuable.textContent).toBe('Scoped valuable kill')
})
test('widget failures retry locally and hiding aborts unfinished reads before resuming',async()=>{
 fetchMock.mockResolvedValueOnce(new Response('failed',{status:503}))
 const host=mount();await settle();await settle()
 expect(host.querySelector('[role=alert]')!.textContent).toContain('Could not load this widget')
 host.querySelector<HTMLButtonElement>('[data-domain-retry]')!.click();await settle();await settle();expect(host.querySelector('.top-box')!.textContent).toBe('Scoped pilot')
 const pending:Array<{signal:AbortSignal;complete:(response:Response)=>void}>=[]
 fetchMock.mockImplementation((_path,options)=>new Promise(resolve=>pending.push({signal:options!.signal!,complete:resolve})))
 const next=mount();await settle();window.dispatchEvent(new Event('pagehide'));expect(pending[0].signal.aborted).toBe(true)
 window.dispatchEvent(new Event('pageshow'));await settle();expect(pending).toHaveLength(2)
 pending[1].complete(new Response('<section class="top-box">Current</section>'));await settle();await settle();pending[0].complete(new Response('<section class="top-box">Stale</section>'));await settle();expect(next.querySelector('.top-box')!.textContent).toBe('Current')
})
