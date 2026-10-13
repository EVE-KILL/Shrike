import {expect,test} from 'vitest'
import {MY_COMMENT_TARGETS,myCommentTarget} from '../src/utils/myCommentTarget'

test('comment chips use comment IDs for anchors and support newer targets',()=>{
 expect(myCommentTarget({id:23,target_type:1,target_id:100,target_slug:null}).href).toBe('/kill/100#comment-23')
 expect(myCommentTarget({id:23,target_type:7,target_id:0,target_slug:'battle-one'}).href).toBe('/battle/battle-one/comments#comment-23')
 expect(myCommentTarget({id:23,target_type:9,target_id:100,target_slug:'blog-one'}).href).toBe('/blog/blog-one#comment-23')
 expect(myCommentTarget({id:23,target_type:10,target_id:0,target_slug:'campaign-one'}).href).toBe('/campaign/campaign-one/comments#comment-23')
 expect(myCommentTarget({id:23,target_type:6,target_id:0,target_slug:'//example.com'}).href).toBe('/example.com')
 expect(myCommentTarget({id:23,target_type:8,target_id:0,target_slug:null}).href).toBe('#')
 const definitions=JSON.parse(require('node:fs').readFileSync(new URL('../../internal/frontend/data/settings.json',import.meta.url),'utf8'))
 expect(definitions.commentTargets).toEqual(MY_COMMENT_TARGETS)
})
