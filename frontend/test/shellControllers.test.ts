// @vitest-environment happy-dom
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {ShellController} from '../src/browser/shell'
import {AccountController} from '../src/browser/account'
import {registerController} from '../src/browser/controllers/application'
import {trackerState,setTrackerUser} from '../src/browser/tracker-state'
import {reconcileNotificationRows} from '../src/browser/notifications'
import type {BrowserUser} from '../src/browser/browser-request'
import type {AccountNotificationRepliesResponses} from '../shared/api/types.gen'

const owner:BrowserUser={characterId:4242,characterName:'Fixture Owner',isAdmin:true,corporationId:10,corporationName:'Fixture Corp',allianceId:20,allianceName:'Fixture Alliance',lastSeenNotificationId:0}
const reply:AccountNotificationRepliesResponses[200]['replies'][number]={id:9,created_at:'2026-10-07T10:00:00Z',target_type:9,target_id:100,target_slug:'test-post',parent_id:1,root_id:1,body_html:'A safe <strong>reply</strong>',character_id:4343,character_name:'Reply Pilot',corporation_id:10,corporation_name:'Fixture Corp',alliance_id:null,alliance_name:null,parent_comment_id:1,parent_snippet:'Parent text'}
const settle=()=>new Promise<void>(resolve=>setTimeout(resolve,0))
const errors=vi.fn()
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function loginForm(){return '<form class="login-form" data-account-target="loginForm" data-action="input->account#preferences change->account#preferences"><input type="checkbox" name="charKm"><input type="checkbox" name="corpKm"><input type="hidden" name="delay"><input type="range" data-login-delay><span data-login-delay-label></span><button></button><p class="login-error"></p></form>'}
function mount(){
  document.body.setAttribute('data-controller','shell')
  document.body.setAttribute('data-action','keydown@document->shell#keyboard click@document->shell#outside click@document->shell#rememberKillNavigation:capture')
  document.body.innerHTML=`<nav class="navbar" data-shell-target="navbar" data-controller="account" data-action="click->account#closeMenus">
    <a class="home-link" href="/">Home</a><button class="search-trigger" data-action="click->shell#openSearch">Search</button>
    <details class="nav-dropdown"><summary>Menu</summary></details>
    <details data-shell-target="mobile" data-action="toggle->shell#toggleMobile toggle->account#refreshBalance"><summary>Mobile</summary><button data-close-mobile data-action="click->shell#closeMobile">Close</button>${loginForm()}</details>
    ${loginForm()}<span data-account-authenticated hidden></span><span data-account-guest></span><span data-account-login-icon></span>
    <button data-account-trigger></button><img data-account-avatar hidden><span data-account-name></span><a data-account-character-link></a>
    <img data-account-picture><span data-account-admin hidden></span><span data-account-balance></span>
    <a data-account-entity="corporation"><span></span><img></a><a data-account-entity="alliance"><span></span><img></a>
    <details data-notifications data-controller="notifications" data-action="toggle->notifications#open click->notifications#follow"><summary>Notifications</summary>
      <span data-notifications-target="badge"></span><span data-notifications-target="count"></span><button data-notifications-target="read" data-action="click->notifications#markRead">Mark read</button>
      <div data-notifications-target="list"><p>Empty</p></div><div data-notifications-target="footer"></div></details>
    <button data-background="bg1" data-action="click->shell#selectBackground"></button></nav>
    <main class="content" data-shell-target="content"><div class="kill-list"><div class="kill-row" data-kill-id="101"><a href="/kill/101"><span>Pilot</span></a></div><div class="kill-row" data-kill-id="100"></div></div></main>
    <button data-shell-target="view" data-action="click->shell#toggleBackgroundView"><svg><use></use></svg></button>
    <dialog data-shell-target="dialog" data-action="close->shell#cancelSearch"><input data-shell-target="searchInput" data-action="input->shell#search keydown->shell#searchKeys"><div data-shell-target="searchResults" data-action="keydown->shell#resultKeys"></div></dialog>`
}
beforeEach(async()=>{
  await setTrackerUser(null);localStorage.clear();sessionStorage.clear();errors.mockClear()
  registerController('shell',ShellController).handleError=errors
  registerController('account',AccountController)
  fetchMock=vi.fn<typeof fetch>(async input=>{
    const path=String(input)
    if(path==='/auth/me')return Response.json({user:owner})
    if(path.startsWith('/api/me/trackers'))return Response.json({trackers:[],limit:50})
    if(path.startsWith('/api/me/tracker-notifications'))return Response.json({notifications:[],unreadCount:0})
    if(path.startsWith('/api/notifications/replies'))return Response.json({replies:[reply],highestId:9})
    if(path==='/api/user/wallet/balance')return Response.json({availableBalance:25000})
    return Response.json({})
  })
  vi.stubGlobal('fetch',fetchMock)
})
afterEach(async()=>{
  document.body.removeAttribute('data-controller');document.body.removeAttribute('data-action');document.body.replaceChildren()
  await settle();await setTrackerUser(null)
  document.documentElement.classList.remove('background-view','mobile-menu-open')
  expect(errors.mock.calls).toEqual([])
  vi.unstubAllGlobals()
})

test('shared account/notification controllers use the typed session and retain settings in both login forms',async()=>{
  mount();await settle();await settle()
  expect(fetchMock.mock.calls.filter(([path])=>path==='/auth/me')).toHaveLength(1)
  expect(document.querySelector('[data-account-name]')!.textContent).toBe('Fixture Owner')
  expect(document.querySelector<HTMLImageElement>('[data-account-avatar]')!.src).toContain('/characters/4242/portrait')
  expect(document.querySelector('[data-account-balance]')!.textContent).toBe('25k ISK')
  expect(document.querySelector<HTMLAnchorElement>('[data-notification-row="reply:9"]')!.getAttribute('href')).toBe('/blog/test-post#comment-9')
  const slider=document.querySelector<HTMLInputElement>('[data-login-delay]')!;slider.value='3';slider.dispatchEvent(new Event('input',{bubbles:true}))
  expect([...document.querySelectorAll<HTMLInputElement>('[name=delay]')].map(input=>input.value)).toEqual(['6','6'])
  expect([...document.querySelectorAll('[data-login-delay-label]')].map(label=>label.textContent)).toEqual(['6 hours','6 hours'])
})

test('shell shortcuts, inert background/mobile state and reconnection keep one event binding',async()=>{
  mount();await settle()
  const view=document.querySelector<HTMLButtonElement>('[data-shell-target=view]')!,content=document.querySelector<HTMLElement>('.content')!,navbar=document.querySelector<HTMLElement>('.navbar')!
  view.click();expect(content.inert).toBe(true);expect(navbar.inert).toBe(true)
  document.dispatchEvent(new KeyboardEvent('keydown',{key:'k',metaKey:true,bubbles:true,cancelable:true}));await settle()
  const dialog=document.querySelector<HTMLDialogElement>('dialog')!,input=document.querySelector<HTMLInputElement>('[data-shell-target=searchInput]')!
  expect(dialog.open).toBe(true);expect(document.activeElement).toBe(input);expect(content.inert).toBe(false)
  document.dispatchEvent(new KeyboardEvent('keydown',{key:'Escape',bubbles:true}));await settle();expect(dialog.open).toBe(false)
  const mobile=document.querySelector<HTMLDetailsElement>('[data-shell-target=mobile]')!;mobile.open=true;mobile.dispatchEvent(new Event('toggle'));await settle()
  expect(content.inert).toBe(true);expect(document.activeElement).toBe(mobile.querySelector('button'))
  mobile.querySelector('button')!.click();await settle();expect(content.inert).toBe(false);expect(document.activeElement).toBe(mobile.querySelector('summary'))
  mobile.open=true;mobile.dispatchEvent(new Event('toggle'));await settle()
  document.dispatchEvent(new KeyboardEvent('keydown',{key:'Escape',bubbles:true}));await settle()
  expect(mobile.open).toBe(false);expect(content.inert).toBe(false);expect(document.activeElement).toBe(mobile.querySelector('summary'))
  document.body.removeAttribute('data-controller');await settle();document.body.setAttribute('data-controller','shell');await settle()
  view.click();expect(view.getAttribute('aria-pressed')).toBe('true')
})

test('closing the shared search cancels pending work and ignores its late response',async()=>{
  mount();await settle()
  let complete:(response:Response)=>void=()=>{}
  fetchMock.mockImplementationOnce(()=>new Promise(resolve=>{complete=resolve}))
  const shell=registerController('shell',ShellController).getControllerForElementAndIdentifier(document.body,'shell') as ShellController
  shell.openSearch();const input=document.querySelector<HTMLInputElement>('[data-shell-target=searchInput]')!;input.value='rifter';input.dispatchEvent(new Event('input',{bubbles:true}))
  await new Promise(resolve=>setTimeout(resolve,220))
  const call=fetchMock.mock.calls.find(([path])=>String(path).startsWith('/api/search'))!
  document.querySelector<HTMLDialogElement>('dialog')!.close();await settle()
  expect(call[1]!.signal!.aborted).toBe(true)
  complete(Response.json({hits:[{id:'ship_587',type:'ship',name:'Rifter'}]}));await settle()
  expect(document.querySelector('[data-shell-target=searchResults]')!.querySelector('a')).toBeNull()
})

test('killmail navigation captures a bounded, ordered list before the clicked link is followed',async()=>{
  mount();await settle()
  const shell=registerController('shell',ShellController).getControllerForElementAndIdentifier(document.body,'shell') as ShellController
  const anchor=document.querySelector<HTMLAnchorElement>('a[href="/kill/101"]')!
  const event=new MouseEvent('click');Object.defineProperty(event,'target',{value:anchor.querySelector('span')})
  shell.rememberKillNavigation(event)
  const token=new URL(anchor.href).searchParams.get('nav')!
  expect(JSON.parse(sessionStorage.getItem('evekill-kill-navigation:'+token)!)).toMatchObject({version:1,ids:[101,100]})
})

test('notification reconciliation retains link identity and keyboard focus through read-state changes',()=>{
  const list=document.createElement('div');document.body.append(list)
  reconcileNotificationRows(list,'<a href="/kill/1" data-notification-row="tracker:1">One</a><a href="/kill/2" data-notification-row="tracker:2">Two</a>')
  const focused=list.querySelector<HTMLAnchorElement>('[data-notification-row="tracker:2"]')!;focused.focus()
  reconcileNotificationRows(list,'<a href="/kill/2" class="read" data-notification-row="tracker:2">Two updated</a>')
  expect(list.firstElementChild).toBe(focused);expect(document.activeElement).toBe(focused);expect(list.children).toHaveLength(1)
})

test('detaching the account and notification scopes cancels requests and prevents stale DOM updates',async()=>{
  mount();await settle()
  let complete:(response:Response)=>void=()=>{}
  fetchMock.mockImplementationOnce(()=>new Promise(resolve=>{complete=resolve}))
  const nav=document.querySelector<HTMLElement>('nav')!,account=registerController('account',AccountController).getControllerForElementAndIdentifier(nav,'account') as AccountController
  const pending=account.refreshBalance();const call=fetchMock.mock.calls.at(-1)!
  nav.remove();await settle();expect(call[1]!.signal!.aborted).toBe(true)
  complete(Response.json({availableBalance:999999}));await pending
  expect(nav.querySelector('[data-account-balance]')!.textContent).not.toContain('1000k')
  expect(trackerState.user).toBeNull()
})
