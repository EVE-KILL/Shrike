import {expect,test} from 'vitest'
import {gunzipSync} from 'node:zlib'
import {encodeFitText} from '../src/fit/codec'

test('v3 fit encoding is gzip UTF-8/base64 and round-trips the original document',async()=>{const document = 'ship,587,Rifter — Ω,From killmail #42\nmodule,High,1,10,Active,11\n';const encoded = await encodeFitText(document);expect(encoded.startsWith('v3:')).toBe(true);expect(gunzipSync(Buffer.from(encoded.slice(3),'base64')).toString('utf8')).toBe(document)})
test('large documents preserve bytes across chunked base64 conversion',async()=>{const document = Array.from({length:2000},(_,index)=>`module,High,${index+1},${100+index},Active,\n`).join('');const encoded = await encodeFitText(document);expect(gunzipSync(Buffer.from(encoded.slice(3),'base64')).toString('utf8')).toBe(document)})
