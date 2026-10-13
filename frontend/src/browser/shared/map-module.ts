// Map pages share one lazy renderer import. Failed downloads remain retryable.
let canvas:Promise<typeof import('../map-canvas')>|undefined
export function loadMapCanvas(){
 if(!canvas){const request=import('../map-canvas');canvas=request;void request.catch(()=>{if(canvas===request)canvas=undefined})}
 return canvas
}
