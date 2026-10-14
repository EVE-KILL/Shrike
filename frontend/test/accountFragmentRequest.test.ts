// @vitest-environment happy-dom
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {readAccountFragment,readBrowserSession,resetBrowserSession,type BrowserUser} from '../src/browser/browser-request'

const user:BrowserUser={characterId:42,characterName:'Pilot',isAdmin:false,corporationId:null,corporationName:null,allianceId:null,allianceName:null,lastSeenNotificationId:7,settings:{defaultTabs:{character:'losses'}}}
function fragment(identity:unknown=user){
  const root=document.createElement('div');root.setAttribute('data-account-fragment','');root.dataset.accountSession=JSON.stringify(identity)
  root.innerHTML='<template data-account-content="desktop"><button>Account</button></template><template data-account-content="mobile"><a href="/settings">Settings</a></template>'
  return new Response(root.outerHTML)
}
beforeEach(()=>{resetBrowserSession();document.body.innerHTML='<nav><div data-account-slot="desktop"></div></nav>'})
afterEach(()=>{resetBrowserSession();document.body.replaceChildren();vi.unstubAllGlobals()})

test('page session and responsive account slots share one credentialed, uncached HTML request',async()=>{
  const fetchMock=vi.fn<typeof fetch>(async()=>fragment());vi.stubGlobal('fetch',fetchMock)
  const account=readAccountFragment(),session=readBrowserSession()
  const data=await account
  expect(await session).toEqual(user);expect(data.desktop.querySelector('button')?.textContent).toBe('Account')
  expect(data.mobile.querySelector('a')?.getAttribute('href')).toBe('/settings')
  expect(fetchMock).toHaveBeenCalledOnce()
  expect(fetchMock).toHaveBeenCalledWith('/_fragments/account',expect.objectContaining({credentials:'same-origin',cache:'no-store',headers:{Accept:'text/html'}}))
  expect(fetchMock.mock.calls[0]?.[1]?.signal).toBeDefined()
  expect(decodeURIComponent(document.cookie)).toContain('"character":"losses"')
})

test('malformed or unrelated HTML cannot silently become a logged-out session',async()=>{
  const fetchMock=vi.fn().mockResolvedValueOnce(new Response('<html><body>Maintenance page</body></html>'))
    .mockResolvedValueOnce(fragment({characterId:'42',characterName:'Pilot',isAdmin:false}))
    .mockResolvedValueOnce(fragment(null))
  vi.stubGlobal('fetch',fetchMock)
  await expect(readBrowserSession()).rejects.toThrow('Invalid account response')
  await expect(readBrowserSession()).rejects.toThrow('Invalid account session')
  expect(await readBrowserSession()).toBeNull();expect(fetchMock).toHaveBeenCalledTimes(3)
})

test('reset cancels an in-flight private read and a late old response cannot overwrite the new session',async()=>{
  let complete:(response:Response)=>void=()=>{}
  const fetchMock=vi.fn<typeof fetch>().mockImplementationOnce(()=>new Promise(resolve=>{complete=resolve}))
    .mockResolvedValueOnce(fragment(null))
  vi.stubGlobal('fetch',fetchMock)
  const pending=readBrowserSession();const rejected=expect(pending).rejects.toMatchObject({name:'AbortError'})
  const signal=fetchMock.mock.calls[0]![1]!.signal!
  resetBrowserSession();expect(signal.aborted).toBe(true)
  const current=readBrowserSession();expect(await current).toBeNull()
  complete(fragment());await rejected
  expect(readBrowserSession()).toBe(current);expect(await readBrowserSession()).toBeNull()
  expect(fetchMock).toHaveBeenCalledTimes(2)
})
