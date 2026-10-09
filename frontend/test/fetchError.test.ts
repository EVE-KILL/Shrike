import {expect,test} from 'vitest'
import {extractFetchError} from '../src/utils/fetchError'

test('API problem details survive generic HTTP status text and malformed errors',()=>{
    expect(extractFetchError({statusMessage:'Bad Request',data:{detail:'This session already expired'}},'Could not revoke session')).toBe('This session already expired')
    expect(extractFetchError({data:{error:'Injected comparison API failure'}},'Request failed')).toBe('Injected comparison API failure')
    expect(extractFetchError({data:{message:{unexpected:true},detail:''}},'Could not save')).toBe('Could not save')
    expect(extractFetchError(null,'Could not save')).toBe('Could not save')
})
