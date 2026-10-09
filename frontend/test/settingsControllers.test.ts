// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {SettingsController} from '../src/browser/settings'
import {registerController} from '../src/browser/controllers/application'
import {readCookieJSON} from '../src/browser/shared/json'
import {CSS_VAR_MAP} from '../src/utils/themeData'
import type {UserSessionsLegacyResponses} from '../shared/api/types.gen'

const sections=['overview','preferences','appearance','sessions','description','wallet','comments'] as const
type Section=typeof sections[number]
// Go-rendered disposable-account fixtures: selectors, nested SVGs and form markup
// must work without a separate hand-written test version of each component.
const markup=Object.fromEntries(sections.map(section=>[section,readFileSync(resolve(import.meta.dirname,'fixtures/settings',section+'.html'),'utf8')]))
const settle=()=>new Promise<void>(resolve=>setTimeout(resolve,0))
const errors=vi.fn()
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function q<T extends Element=HTMLElement>(root:ParentNode,selector:string):T{
  const node=root.querySelector<T>(selector)
  if(!node)throw new Error(`Missing fixture element ${selector}`)
  return node
}
function mount(section:Section){
  document.body.innerHTML=markup.overview
  const root=q(document,'[data-settings-root]')
  if(section!=='overview')q(root,'[data-settings-panels]').innerHTML=markup[section]
  root.dataset.active=section
  history.replaceState({},'',section==='overview'?'/settings':`/settings/${section}`)
  return root
}
function fragment(section:Section){return new Response(markup[section])}
function deferredFetch(){
  const pending:Array<{path:string;signal:AbortSignal;complete:(response:Response)=>void}>=[]
  fetchMock.mockImplementation((path,options)=>new Promise(resolve=>pending.push({path:String(path),signal:options!.signal!,complete:resolve})))
  return pending
}
beforeEach(()=>{
  errors.mockClear();registerController('settings',SettingsController).handleError=errors
  for(const name of ['ek_theme','ek_prefs','ek_tz'])document.cookie=`${name}=; Max-Age=0; Path=/`
  document.documentElement.removeAttribute('style')
  fetchMock=vi.fn<typeof fetch>(async()=>Response.json({}))
  vi.stubGlobal('fetch',fetchMock)
})
afterEach(async()=>{
  vi.useRealTimers();document.body.replaceChildren();await settle()
  expect(errors.mock.calls).toEqual([])
  vi.unstubAllGlobals();vi.restoreAllMocks()
})

test('settings cache retains bio drafts and Back reuses the existing tab without fetching it again',async()=>{
  fetchMock.mockImplementation(async path=>fragment(String(path).endsWith('/preferences')?'preferences':'description'))
  const root=mount('description');await settle()
  const text=q<HTMLTextAreaElement>(root,'textarea');text.value='Unsaved bio';text.dispatchEvent(new Event('input',{bubbles:true}))
  q<HTMLAnchorElement>(root,'[data-settings-tab="preferences"]').click();await settle();await settle()
  expect(location.pathname).toBe('/settings/preferences')
  expect(q(root,'[data-settings-section="description"]').hidden).toBe(true)
  history.replaceState({},'','/settings/description');window.dispatchEvent(new PopStateEvent('popstate'));await settle()
  expect(q<HTMLTextAreaElement>(root,'textarea')).toBe(text);expect(text.value).toBe('Unsaved bio')
  expect(q(root,'[data-settings-section="description"]').hidden).toBe(false);expect(fetchMock).toHaveBeenCalledOnce()
  // Modified clicks keep normal browser navigation semantics.
  q(root,'[data-settings-tab="preferences"]').dispatchEvent(new MouseEvent('click',{bubbles:true,metaKey:true,cancelable:true}))
  expect(fetchMock).toHaveBeenCalledOnce()
})
test('superseded settings fragments abort and a late response cannot replace the selected panel',async()=>{
  const pending=deferredFetch(),root=mount('overview');await settle()
  q<HTMLAnchorElement>(root,'[data-settings-tab="description"]').click()
  q<HTMLAnchorElement>(root,'[data-settings-tab="preferences"]').click()
  expect(pending[0].signal.aborted).toBe(true)
  pending[1].complete(fragment('preferences'));await settle();await settle()
  pending[0].complete(fragment('description'));await settle()
  expect(root.dataset.active).toBe('preferences');expect(root.querySelector('[data-settings-section="description"]')).toBeNull()
  expect(root.hasAttribute('aria-busy')).toBe(false)
})
test('failed settings navigation stays retryable and rejects the wrong fragment section',async()=>{
  fetchMock.mockResolvedValueOnce(fragment('description')).mockResolvedValueOnce(fragment('preferences'))
  const root=mount('overview');await settle();q<HTMLAnchorElement>(root,'[data-settings-tab="preferences"]').click();await settle()
  expect(q(root,'[data-settings-section="preferences"] [role="alert"]').textContent).toContain('Could not load')
  q<HTMLButtonElement>(root,'[data-settings-section-retry]').click();await settle();await settle()
  expect(root.querySelector('[data-settings-preferences]')).not.toBeNull();expect(root.querySelector('[data-settings-section-retry]')).toBeNull()
})
test('preferences retain keyboard focus and save the actual defaultTabs contract without losing other cookie fields',async()=>{
  const root=mount('preferences');await settle()
  document.cookie=`ek_prefs=${encodeURIComponent(JSON.stringify({anotherPreference:true}))}; Path=/`
  const menu=q(root,'[data-settings-select="character"]'),trigger=q<HTMLButtonElement>(menu,'button')
  trigger.dispatchEvent(new KeyboardEvent('keydown',{key:'ArrowDown',bubbles:true,cancelable:true}))
  expect(q(menu,'[data-settings-select-options]').hidden).toBe(false)
  const option=q<HTMLButtonElement>(menu,'[data-settings-select-option="losses"]');option.click()
  expect(document.activeElement).toBe(trigger);expect(menu.dataset.value).toBe('losses')
  fetchMock.mockImplementation(async(_path,options)=>Response.json({defaultTabs:JSON.parse(String(options!.body)).defaultTabs}))
  q<HTMLButtonElement>(root,'[data-settings-save-prefs]').click();await settle()
  expect(readCookieJSON('ek_prefs')).toMatchObject({anotherPreference:true,defaultTabs:{character:'losses'}})
  expect(new Headers(fetchMock.mock.calls[0][1]!.headers).get('Content-Type')).toBe('application/json')
  expect(q(root,'[data-settings-prefs-saved]').hidden).toBe(false)
})
test('preference menus close on Escape/outside and timezone updates notify the site',async()=>{
  const root=mount('preferences');await settle();const menu=q(root,'[data-settings-select="character"]'),trigger=q<HTMLButtonElement>(menu,'button')
  trigger.click();q(menu,'[data-settings-select-options]').dispatchEvent(new KeyboardEvent('keydown',{key:'Escape',bubbles:true}))
  expect(document.activeElement).toBe(trigger);expect(trigger.getAttribute('aria-expanded')).toBe('false')
  trigger.click();document.body.dispatchEvent(new PointerEvent('pointerdown',{bubbles:true}));expect(trigger.getAttribute('aria-expanded')).toBe('false')
  const change=vi.fn();document.addEventListener('eve-time-zone-change',change,{once:true})
  q<HTMLButtonElement>(root,'[data-settings-zone="local"]').click()
  expect(change).toHaveBeenCalledOnce();expect(document.cookie).toContain('ek_tz=local')
})
test('theme saves preserve edits made while saving and keep them dirty when the component reconnects',async()=>{
  const pending=deferredFetch(),root=mount('appearance');await settle()
  const color=q<HTMLInputElement>(root,'[data-settings-color="brandPrimary"]'),save=q<HTMLButtonElement>(root,'[data-settings-save-theme]')
  color.value='#abcdef';color.dispatchEvent(new Event('input',{bubbles:true}));save.click()
  color.value='#fedcba';color.dispatchEvent(new Event('input',{bubbles:true}));pending[0].complete(Response.json({ok:true}));await settle()
  expect(document.documentElement.style.getPropertyValue(CSS_VAR_MAP.brandPrimary)).toBe('#fedcba')
  expect(q(root,'[data-settings-theme-status]').textContent).toContain('Unsaved changes');expect(save.disabled).toBe(false)
  root.remove();await settle();document.body.append(root);await settle()
  expect(q(root,'[data-settings-theme-status]').textContent).toContain('Unsaved changes');expect(save.disabled).toBe(false)
})
test('detaching an editor aborts saves and an ignored abort cannot show a success indicator',async()=>{
  const pending=deferredFetch(),root=mount('appearance');await settle()
  const color=q<HTMLInputElement>(root,'[data-settings-color="brandPrimary"]');color.value='#abcdef';color.dispatchEvent(new Event('input',{bubbles:true}))
  q<HTMLButtonElement>(root,'[data-settings-save-theme]').click();root.remove();await settle()
  expect(pending[0].signal.aborted).toBe(true);pending[0].complete(Response.json({ok:true}));await settle()
  expect(q(root,'[data-settings-theme-saved]').hidden).toBe(true);expect(root.querySelector('[data-settings-theme-error]')!.textContent).toBe('')
})
test('bio submission captures entity and format while preserving newer drafts and sibling editors',async()=>{
  const pending=deferredFetch(),root=mount('description');await settle()
  const character=q(root,'[data-settings-bio="character"]'),corporation=q(root,'[data-settings-bio="corporation"]'),text=q<HTMLTextAreaElement>(character,'textarea')
  const other=q<HTMLTextAreaElement>(corporation,'textarea');other.value='Corporation draft'
  text.value='Submitted draft';q<HTMLButtonElement>(character,'[data-settings-bio-format="eve_html"]').click()
  q<HTMLButtonElement>(character,'[data-settings-bio-save]').click();text.value='Newer draft';text.dispatchEvent(new Event('input',{bubbles:true}))
  expect(JSON.parse(String(fetchMock.mock.calls[0][1]!.body))).toEqual({entity:'character',format:'eve_html',description:'Submitted draft'})
  pending[0].complete(Response.json({ok:true,status:'submitted',entity:'character',entity_id:42}));await settle()
  expect(text.value).toBe('Newer draft');expect(other.value).toBe('Corporation draft');expect(q(character,'[data-settings-bio-pending]').hidden).toBe(false)
})
const sessions:UserSessionsLegacyResponses[200]['sessions']=[true,false].map((current,index)=>({id:index+1,current,browser:'Fixture <Browser>',operatingSystem:'Test OS',device:'Desktop',countryCode:'DK',ipAddress:'127.0.0.1',createdAt:'2026-10-07T12:00:00Z',lastSeenAt:'2026-10-07T12:00:00Z',expiresAt:'2026-10-08T12:00:00Z'}))
test('session bulk revoke requires confirmation and escapes device metadata',async()=>{
  fetchMock.mockImplementation(async path=>Response.json(String(path).endsWith('/sessions')?{sessions}:{}))
  const root=mount('sessions');await settle();await settle()
  expect(root.querySelector('browser')).toBeNull();expect(root.textContent).toContain('Fixture <Browser>')
  const bulk=q<HTMLButtonElement>(root,'[data-settings-revoke-others]');bulk.click()
  expect(fetchMock).toHaveBeenCalledOnce();expect(bulk.textContent).toContain('Confirm logout of 1')
  bulk.click();await settle();await settle()
  expect(fetchMock.mock.calls.filter(([path])=>path==='/api/user/sessions/revoke-others')).toHaveLength(1)
})
test('expired session confirmation disarms without posting a revoke',async()=>{
  fetchMock.mockResolvedValue(Response.json({sessions}));const root=mount('sessions');await settle();await settle();vi.useFakeTimers()
  q<HTMLButtonElement>(root,'[data-settings-revoke-others]').click();await vi.advanceTimersByTimeAsync(5001)
  expect(q(root,'[data-settings-revoke-label]').textContent).toBe('Log out everywhere else');expect(fetchMock).toHaveBeenCalledOnce()
})
test('session revoke failures surface the API error and re-enable individual controls',async()=>{
  fetchMock.mockImplementation(async path=>String(path)==='/api/user/sessions'?Response.json({sessions}):Response.json({message:'Fixture revoke unavailable'},{status:503}))
  const root=mount('sessions');await settle();await settle()
  q<HTMLButtonElement>(root,'[data-settings-revoke="2"]').click();await settle()
  expect(q(root,'[data-settings-sessions-error]').textContent).toBe('Fixture revoke unavailable')
  expect(q<HTMLButtonElement>(root,'[data-settings-revoke="2"]').disabled).toBe(false)
})
test('private wallet copy gives temporary confirmation and restores the original SVG and label',async()=>{
  const writeText=vi.spyOn(navigator.clipboard,'writeText').mockResolvedValue()
  const root=mount('wallet');await settle();vi.useFakeTimers();const button=q<HTMLButtonElement>(root,'[data-settings-wallet-copy]')
  button.click();await vi.advanceTimersByTimeAsync(0)
  expect(writeText).toHaveBeenCalledWith(button.dataset.settingsWalletCopy)
  expect(q(button,'[data-settings-wallet-copy-label]').textContent).toBe('Copied')
  await vi.advanceTimersByTimeAsync(2001);expect(q(button,'[data-settings-wallet-copy-label]').textContent).toBe('Copy name');expect(q(button,'use').getAttribute('href')).toBe('/assets/icons.svg#copy')
})
test('private wallet pagination replaces the component and a failed retry restores the original controls',async()=>{
  const root=mount('wallet');await settle();const old=q(root,'[data-settings-wallet]'),next=q<HTMLButtonElement>(old,'[data-settings-wallet-page="2"]')
  fetchMock.mockResolvedValueOnce(new Response('',{status:503})).mockResolvedValueOnce(new Response(old.outerHTML.replace('data-page="1"','data-page="2"')))
  next.click();await settle();expect(next.disabled).toBe(false);expect(q(old,'[data-settings-wallet-error]').hidden).toBe(false)
  next.click();await settle();await settle();expect(q(root,'[data-settings-wallet]')).not.toBe(old)
  expect(fetchMock.mock.calls[1][0]).toBe('/_fragments/settings/wallet?page=2');expect(q(root,'[data-settings-wallet-error]').hidden).toBe(true)
})
test('comment deletion requires a second click and an API failure preserves the card for retry',async()=>{
  const root=mount('comments');await settle();const card=q(root,'[data-settings-comment]'),button=q<HTMLButtonElement>(card,'[data-settings-comment-delete]')
  fetchMock.mockResolvedValueOnce(Response.json({message:'Fixture delete unavailable'},{status:503})).mockResolvedValueOnce(Response.json({ok:true}))
  button.click();expect(fetchMock).not.toHaveBeenCalled();expect(q(button,'[data-settings-comment-delete-label]').textContent).toBe('Confirm')
  button.click();await settle();expect(card.isConnected).toBe(true);expect(button.disabled).toBe(false);expect(q(root,'[data-settings-comments-error-text]').textContent).toBe('Fixture delete unavailable')
  button.click();button.click();await settle();expect(card.isConnected).toBe(false);expect(fetchMock.mock.calls[1][0]).toBe('/api/user/comments/'+card.dataset.settingsComment)
})
test('comments pagination deduplicates cards, preserves cursor headers and resets on refresh',async()=>{
  const root=mount('comments');await settle();const list=q(root,'[data-settings-comments-list]'),first=q(list,'[data-settings-comment]'),newCard=first.cloneNode(true)
  if(!(newCard instanceof HTMLElement))throw Error('Missing comment card');newCard.dataset.settingsComment='999'
  fetchMock.mockResolvedValueOnce(new Response(first.outerHTML+newCard.outerHTML,{headers:{'X-Next-Cursor':'next:999'}})).mockResolvedValueOnce(new Response(''))
  const count=list.children.length;q<HTMLButtonElement>(root,'[data-settings-comments-more]').click();await settle()
  expect(list.children.length).toBe(count+1);expect(q(root,'[data-settings-comments]').dataset.cursor).toBe('next:999')
  q<HTMLButtonElement>(root,'[data-settings-comments-refresh]').click();await settle()
  expect(list.children).toHaveLength(0);expect(q(root,'[data-settings-comments-empty]').hidden).toBe(false)
})
test('comments error keeps its icon and interrupted loading releases controls without accepting stale cards',async()=>{
  const root=mount('comments');await settle();const more=q<HTMLButtonElement>(root,'[data-settings-comments-more]')
  fetchMock.mockResolvedValueOnce(new Response('',{status:503}));more.click();await settle()
  expect(q(root,'[data-settings-comments-error] svg')).not.toBeNull();expect(q(root,'[data-settings-comments-error-text]').textContent).toContain('Could not load')
  const pending=deferredFetch();more.click();window.dispatchEvent(new PageTransitionEvent('pagehide'));expect(pending[0].signal.aborted).toBe(true);expect(more.disabled).toBe(false)
  pending[0].complete(new Response('<li data-settings-comment="999">Late comment</li>'));await settle();expect(root.textContent).not.toContain('Late comment')
})
