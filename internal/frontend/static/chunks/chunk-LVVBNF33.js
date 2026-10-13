// Generated browser asset. Edit frontend/src/browser and run npm --prefix frontend run build.
function l(s){let a=new Set,t=null;function o(e){a.delete(e),e.close(),a.size||(t?.close(),t=null)}return{connect(e){a.add(e),e.onmessage=n=>{n.data==="disconnect"&&o(e)},e.start(),!t&&(t=s(),t.addEventListener("build",n=>{for(let d of a)try{d.postMessage(n.data)}catch{o(d)}}))}}}export{l as a};
