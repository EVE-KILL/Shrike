// Kill lists use opaque decimal killmail cursors and retain previous cursors
// within a tab. Limit/tab changes deliberately reset that history.
export class KillPages {
  private cursors = new Map<number,string>()
  clear() {this.cursors.clear()}
  params(current:HTMLElement,page:number,limit:number):URLSearchParams|null {
    if(!Number.isSafeInteger(page) || page < 1 || page > 1_000_000 || ![10,25,50,100].includes(limit)) return null
    if(page > 1 && page === Number(current.dataset.page)+1 && current.dataset.nextCursor) this.cursors.set(page-1,current.dataset.nextCursor)
    const after = this.cursors.get(page-1);if(page > 1 && !after) return null
    const params = new URLSearchParams({page:String(page),limit:String(limit)});if(page > 1 && after) params.set('after',after);return params
  }
}
export function matchingKillPage(next:HTMLElement,kind:string,page:number,limit:number):boolean {return next.dataset.kind === kind && Number(next.dataset.page) === page && Number(next.dataset.limit) === limit}
