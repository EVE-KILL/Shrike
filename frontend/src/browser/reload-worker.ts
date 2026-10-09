import {createReloadChannel} from './reload-channel'
// This entry runs in a SharedWorker; the application project uses DOM globals.
declare const self:{onconnect:((event:MessageEvent<unknown>)=>void)|null}
const channel=createReloadChannel(()=>{
  const source=new EventSource('/__dev/reload')
  return {
    close:()=>source.close(),
    addEventListener(name,callback){
      source.addEventListener(name,event=>{
        if(event instanceof MessageEvent&&typeof event.data==='string')callback({data:event.data})
      })
    },
  }
})
self.onconnect=event=>{const port=event.ports[0];if(port)channel.connect(port)}
