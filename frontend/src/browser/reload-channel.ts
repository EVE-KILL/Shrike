export interface ReloadPort {
  onmessage:((event:MessageEvent<string>)=>void)|null
  postMessage(data:string):void
  start():void
  close():void
}
export interface ReloadStream {
  close():void
  addEventListener(name:'build',callback:(event:{data:string})=>void):void
}
// One EventSource per origin/build. Last document departure releases it.
export function createReloadChannel(createStream:()=>ReloadStream){
  const ports=new Set<ReloadPort>()
  let stream:ReloadStream|null=null
  function disconnect(port:ReloadPort){
    ports.delete(port);port.close()
    if(!ports.size){stream?.close();stream=null}
  }
  return {
    connect(port:ReloadPort){
      ports.add(port)
      port.onmessage=event=>{if(event.data==='disconnect')disconnect(port)}
      port.start()
      if(stream)return
      stream=createStream()
      stream.addEventListener('build',event=>{
        for(const client of ports)try{client.postMessage(event.data)}catch{disconnect(client)}
      })
      // New documents probe independently rather than receiving a stale replay.
    },
  }
}
