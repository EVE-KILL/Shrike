import {afterEach,expect,test,vi} from 'vitest'
afterEach(()=>{vi.unstubAllGlobals();vi.resetModules()})

test('page widgets share one in-flight session request',async()=>{
  vi.resetModules()
  const fetch=vi.fn(async()=>new Response(JSON.stringify({user:{characterId:123,settings:{defaultTabs:{character:'losses'}}}})))
  vi.stubGlobal('fetch',fetch)
  const {readBrowserSession}=await import('../src/browser/browser-request')
  const first=readBrowserSession(),second=readBrowserSession()
  expect(first).toBe(second)
  expect(await first).toMatchObject({characterId:123})
  await readBrowserSession()
  expect(fetch).toHaveBeenCalledTimes(1)
})

test('failed sessions preserve status and retry instead of caching failure',async()=>{
  vi.resetModules()
  const fetch=vi.fn().mockResolvedValueOnce(new Response(JSON.stringify({message:'Temporarily unavailable'}),{status:503}))
    .mockResolvedValueOnce(new Response(JSON.stringify({user:null})))
  vi.stubGlobal('fetch',fetch)
  const {readBrowserSession}=await import('../src/browser/browser-request')
  await expect(readBrowserSession()).rejects.toMatchObject({status:503,message:'Temporarily unavailable'})
  expect(await readBrowserSession()).toBeNull()
  expect(fetch).toHaveBeenCalledTimes(2)
})

test('JSON transport preserves request options and same-origin credentials',async()=>{
  const fetch=vi.fn(async()=>new Response(JSON.stringify({saved:true})))
  vi.stubGlobal('fetch',fetch)
  const {requestJSON}=await import('../src/browser/browser-request')
  expect(await requestJSON('/api/fixture',{method:'POST',body:'{}'})).toEqual({saved:true})
  expect(fetch).toHaveBeenCalledWith('/api/fixture',{credentials:'same-origin',method:'POST',body:'{}'})
})
