// Generated browser asset. Edit frontend/src/browser and run npm --prefix frontend run build.
async function s(n){let o=await new Response(new Blob([n]).stream().pipeThrough(new CompressionStream("gzip"))).arrayBuffer(),r=new Uint8Array(o),t="";for(let e=0;e<r.length;e+=32768)t+=String.fromCharCode(...r.subarray(e,e+32768));return"v3:"+btoa(t)}export{s as a};
