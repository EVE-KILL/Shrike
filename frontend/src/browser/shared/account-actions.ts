import {BrowserRequestError,requestJSON} from '../browser-request'
import {errorMessage} from './json'

export function accountError(error:unknown,fallback='Could not complete this request.'){
  return error instanceof BrowserRequestError&&error.status===401?'Your session expired. Log in again to continue.':errorMessage(error,fallback)
}
export async function accountLoginDestination(redirect=location.pathname){
  const result=await requestJSON<{url:string}>('/auth/login?'+new URLSearchParams({redirect,charKm:'1',corpKm:'1',delay:'0'}))
  const destination=new URL(result.url)
  if(destination.protocol!=='https:')throw new Error('Invalid login response.')
  return destination.href
}
