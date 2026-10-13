(() => {
  const initial=document.querySelector<HTMLMetaElement>('meta[name="shrike-dev-instance"]')?.content
  if(!initial)return
  let dispose:(()=>void)|null=null,reloading=false
  function receive(version:unknown){
    if(typeof version!=='string'||!version.trim()||version.trim()===initial||reloading)return
    reloading=true;dispose?.();dispose=null;window.location.reload()
  }
  function connect(){
    if(dispose||reloading)return
    let worker:SharedWorker|null=null,timer:ReturnType<typeof setTimeout>|undefined,pending:AbortController|null=null,closed=false,polling=false
    async function probe(){
      if(closed||pending)return
      const controller=new AbortController();pending=controller
      const timeout=setTimeout(()=>controller.abort(),5000)
      try{
        const response=await fetch('/__dev/version',{cache:'no-store',signal:controller.signal})
        if(response.ok&&!closed)receive(await response.text())
      }catch{/* Air may be replacing the process. Retry on the next tick. */}
      finally{clearTimeout(timeout);if(pending===controller)pending=null}
    }
    function closePort(){if(worker){worker.port.postMessage('disconnect');worker.port.close();worker=null}}
    function poll(){
      if(closed||polling)return
      polling=true;closePort()
      const tick=async()=>{await probe();if(!closed)timer=setTimeout(tick,1500)}
      void tick()
    }
    dispose=()=>{closed=true;clearTimeout(timer);pending?.abort();closePort()}
    if(window.SharedWorker){
      try{
        worker=new SharedWorker('/assets/reload-worker.js',{type:'module',name:'shrike-development-reload'})
        worker.port.onmessage=event=>receive(event.data)
        worker.onerror=poll;worker.port.start();void probe()
      }catch{poll()}
    }else poll()
  }
  window.addEventListener('pagehide',()=>{dispose?.();dispose=null})
  window.addEventListener('pageshow',event=>{if(event.persisted)connect()})
  connect()
})()
