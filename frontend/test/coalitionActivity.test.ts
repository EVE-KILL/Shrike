import {expect,test} from 'vitest'
import {loadCoalitionActivity} from '../src/browser/coalition-activity'

function directory() {
  const message={textContent:''},retry={hidden:true}
  const status={hidden:false,isConnected:true,busy:'false',
    getAttribute:()=>status.busy,setAttribute:(_name:string,value:string)=>{status.busy=value},
    querySelector:(selector:string)=>selector==='span'?message:retry}
  const links=[{href:'/coalitions/alpha',hidden:false},{href:'/coalitions/beta',hidden:true}]
  const nodes:any[]=[]
  const node=(id:string,text:string)=>({dataset:{coalitionActivity:id},text,
    replaceWith(replacement:any){nodes[nodes.indexOf(this)]=replacement}})
  nodes.push(node('1','—'),node('2','—'))
  const root={isConnected:true,links,nodes,
    querySelector:()=>status,querySelectorAll:()=>nodes,
    ownerDocument:{createElement:()=>{
      let received:any[]=[]
      return {set innerHTML(value:string){received=JSON.parse(value).map(([id,text]:string[])=>node(id,text))},
        content:{querySelectorAll:()=>received}}
    }}}
  return {root:root as unknown as HTMLElement,rawRoot:root,status,message,retry,nodes,links}
}

test('delayed activity preserves usable filtered links and fills only matching totals',async()=>{
  const d=directory()
  let resolve:(value:Response)=>void
  const response=new Promise<Response>(r=>{resolve=r})
  const pending=loadCoalitionActivity(d.root,()=>response)
  expect(d.status.busy).toBe('true')
  expect(d.nodes.map(n=>n.text)).toEqual(['—','—'])
  expect(d.links).toEqual([{href:'/coalitions/alpha',hidden:false},{href:'/coalitions/beta',hidden:true}])
  resolve!(new Response(JSON.stringify([['2','200'],['1','100']])))
  await pending
  expect(d.nodes.map(n=>n.text)).toEqual(['100','200'])
  expect(d.links[1].hidden).toBe(true)
  expect(d.status.hidden).toBe(true)
  expect(d.status.busy).toBe('false')
})

test('activity failure keeps the directory and supports retry without duplicate requests',async()=>{
  const d=directory()
  let requests=0,finish:()=>void
  const wait=new Promise<void>(resolve=>{finish=resolve})
  const request=async()=>{requests++;await wait;return new Response(null,{status:503})}
  const pending=loadCoalitionActivity(d.root,request)
  await loadCoalitionActivity(d.root,request)
  expect(requests).toBe(1)
  finish!();await pending
  expect(d.nodes.map(n=>n.text)).toEqual(['—','—'])
  expect(d.retry.hidden).toBe(false)
  expect(d.status.busy).toBe('false')
  await loadCoalitionActivity(d.root,async()=>new Response(JSON.stringify([['1','100'],['2','200']])))
  expect(d.status.hidden).toBe(true)
  expect(d.retry.hidden).toBe(true)
})

test('incomplete activity responses never partially replace visible totals',async()=>{
  const d=directory()
  await loadCoalitionActivity(d.root,async()=>new Response(JSON.stringify([['1','100']])))
  expect(d.nodes.map(n=>n.text)).toEqual(['—','—'])
  expect(d.retry.hidden).toBe(false)
})

test('activity arriving after leaving the page cannot replace detached content',async()=>{
  const d=directory()
  await loadCoalitionActivity(d.root,async()=>{
    d.rawRoot.isConnected=false;d.status.isConnected=false
    return new Response(JSON.stringify([['1','100'],['2','200']]))
  })
  expect(d.nodes.map(n=>n.text)).toEqual(['—','—'])
})
