// @vitest-environment happy-dom
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {liveRow,parseLiveKill} from '../src/browser/kill-row'
import {initializePrivateKillFeed,PrivateKillFeedController} from '../src/browser/private-killfeed'
import {createRelayConnection} from '../src/browser/account-data'
import type {RelayCoreOptions} from '../src/utils/relaySocketCore'
import {registerController} from '../src/browser/controllers/application'
vi.mock('../src/browser/account-data',async importOriginal=>({...await importOriginal<typeof import('../src/browser/account-data')>(),createRelayConnection:vi.fn()}))
const settle=()=>new Promise<void>(resolve=>setTimeout(resolve,0))
const errors=vi.fn()
let relays:Array<{options:RelayCoreOptions;connect:ReturnType<typeof vi.fn>;disconnect:ReturnType<typeof vi.fn>;dispose:ReturnType<typeof vi.fn>}>=[]
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
const kill=(id=101)=>({killmail_id:id,killmail_time:'2026-10-07T12:00:00Z',total_value:150000000,ship_type_id:587,ship_name:'Rifter',victim_character_id:42,victim_character_name:'Pilot',solar_system_id:30000142,solar_system_name:'Jita',solar_system_security:.9,attacker_count:3})
function fragment(id=100,cursor='next-cursor'){
  return `<div class="kill-list" data-next-cursor="${cursor}"><span class="kill-live-dot"></span><button class="new-kills" hidden><span></span></button><span class="kill-day-heading"></span><nav class="kill-pager"><button data-kill-page="1">Previous</button><button data-kill-page="2">Next</button><button data-kill-limit="10">10</button></nav><div class="kill-rows"><article class="kill-row" data-kill-id="${id}" data-victim-character-id="42"><time datetime="2026-10-07T11:00:00Z"></time></article></div></div>`
}
function mount(){const host=document.createElement('div');document.body.append(host);return {host,feed:initializePrivateKillFeed(host,{endpoint:'/_fragments/test',topics:['victim.42'],lossType:'character',lossId:42})}}
beforeEach(()=>{
  errors.mockClear();relays=[];registerController('private-killfeed',PrivateKillFeedController).handleError=errors
  vi.mocked(createRelayConnection).mockImplementation(options=>{const relay={options,connect:vi.fn(),disconnect:vi.fn(),dispose:vi.fn()};relays.push(relay);return {...relay,connected:false}})
  fetchMock=vi.fn<typeof fetch>(async()=>new Response(fragment()));vi.stubGlobal('fetch',fetchMock)
})
afterEach(async()=>{document.body.replaceChildren();await settle();expect(errors.mock.calls).toEqual([]);vi.unstubAllGlobals()})
test('typed live rows preserve item final blows, people count, identity links and fixed portrait geometry',()=>{
  expect(parseLiveKill({killmail_id:1,killmail_time:'invalid'})).toBeNull()
  const data=parseLiveKill({...kill(),final_blow_ship_type_id:45000,final_blow_ship_name:'Standup projector',victim_character_name:'<unsafe>'})!
  const row=liveRow(data)!
  expect(row.querySelector('[aria-label="3 involved"] use')!.getAttribute('href')).toBe('/assets/icons.svg#users')
  expect(row.querySelectorAll('.kill-identity')[1].querySelector('a')!.getAttribute('href')).toBe('/item/45000')
  expect(row.querySelectorAll('.kill-identity')[1].querySelector('img')!.getAttribute('src')).toBe('/images/types/45000/icon?size=64')
  expect(row.querySelector('.identity-primary')!.textContent).toBe('<unsafe>');expect(row.querySelector('unsafe')).toBeNull()
  expect(row.querySelector<HTMLImageElement>('.kill-portrait img')!.width).toBe(40)
})
test('cursor pages send the recorded cursor, queue live arrivals off page one and color losses',async()=>{
  const {host}=mount();await settle();await settle()
  expect(host.querySelector('.kill-row')!.classList.contains('loss-row')).toBe(true)
  host.querySelector<HTMLButtonElement>('[data-kill-page="2"]')!.click();await settle();await settle()
  const url=new URL(String(fetchMock.mock.calls.at(-1)![0]),location.href);expect(url.searchParams.get('after')).toBe('next-cursor');expect(url.searchParams.get('page')).toBe('2')
  relays[0].options.onMessage('killlist',{killmail:kill()},[])
  expect(host.querySelector<HTMLButtonElement>('.new-kills')!.hidden).toBe(false);expect(host.querySelector('.new-kills span')!.textContent).toBe('1')
  expect(host.querySelector('[data-kill-id="101"]')).toBeNull()
  host.querySelector<HTMLButtonElement>('.new-kills')!.click();await settle();await settle()
  expect(new URL(String(fetchMock.mock.calls.at(-1)![0]),location.href).searchParams.get('page')).toBe('1')
})
test('page-one relay inserts newer rows once and ignores malformed and out-of-order messages',async()=>{
  const {host}=mount();await settle();await settle()
  relays[0].options.onMessage('killlist',{killmail:kill()},[]);relays[0].options.onMessage('killlist',{killmail:kill()},[])
  relays[0].options.onMessage('killlist',{killmail:kill(99)},[]);relays[0].options.onMessage('killlist',{killmail:{killmail_id:102}},[])
  expect(host.querySelectorAll('.kill-row')).toHaveLength(2);expect(host.querySelector('.kill-row')!.getAttribute('data-kill-id')).toBe('101')
  expect(host.querySelector('.kill-row')!.classList.contains('loss-row')).toBe(true);expect(host.querySelector('.kill-bottom-pager')).not.toBeNull()
})
test('superseded reads and detach cannot overwrite newer content, and failed reads remain retryable',async()=>{
  const pending:Array<{signal:AbortSignal;complete:(response:Response)=>void}>=[]
  fetchMock.mockImplementation((_path,options)=>new Promise(resolve=>pending.push({signal:options!.signal!,complete:resolve})))
  const {host,feed}=mount();await settle();feed.refresh();expect(pending[0].signal.aborted).toBe(true)
  pending[1].complete(new Response(fragment(200)));await settle();await settle();pending[0].complete(new Response(fragment(100)));await settle()
  expect(host.querySelector('.kill-row')!.getAttribute('data-kill-id')).toBe('200')
  feed.refresh();pending[2].complete(new Response('',{status:401}));await settle();expect(host.querySelector('[role=alert]')!.textContent).toContain('session expired')
  host.querySelector<HTMLButtonElement>('[data-private-kill-retry]')!.click();await settle();expect(pending).toHaveLength(4)
  host.remove();await settle();expect(pending[3].signal.aborted).toBe(true);expect(relays[0].dispose).toHaveBeenCalledOnce()
  pending[3].complete(new Response(fragment(999)));await settle();expect(host.textContent).not.toContain('999')
})
test('bfcache restoration retries interrupted work and explicit parent destroy stops future reconnection',async()=>{
  const pending:Array<{signal:AbortSignal;complete:(response:Response)=>void}>=[]
  fetchMock.mockImplementation((_path,options)=>new Promise(resolve=>pending.push({signal:options!.signal!,complete:resolve})))
  const {host,feed}=mount();await settle();window.dispatchEvent(new Event('pagehide'));expect(pending[0].signal.aborted).toBe(true)
  window.dispatchEvent(new Event('pageshow'));expect(pending).toHaveLength(2);expect(relays[0].connect).toHaveBeenCalledTimes(2)
  pending[1].complete(new Response(fragment()));await settle();feed.destroy();host.remove();await settle();document.body.append(host);await settle()
  expect(relays).toHaveLength(1);expect(relays[0].dispose).toHaveBeenCalledOnce()
  pending[0].complete(new Response(fragment()));await settle()
})
