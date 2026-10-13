import { expect,test } from 'vitest'
import {adminCommentTarget,adminCommentStatus,moderationKindLabel,moderationStatusLabel,moderationTargetHref,moderationTargetIdentity} from '../src/utils/adminModeration'

test('admin target links retain numeric routes and use canonical slug routes for newer comment kinds',()=>{
    const c={id:42,target_type:1,target_id:99,target_slug:null as string|null}
    expect(adminCommentTarget(c)).toEqual({label:'Kill #99',href:'/kill/99#comment-42'})
    expect(adminCommentTarget({...c,target_type:7,target_slug:'battle one'}).href).toBe('/battle/battle%20one/comments#comment-42')
    expect(adminCommentTarget({...c,target_type:8,target_slug:'fit-one'}).href).toBe('/fit/fit-one#comment-42')
    expect(adminCommentTarget({...c,target_type:9,target_slug:'news'}).href).toBe('/blog/news#comment-42')
    expect(adminCommentTarget({...c,target_type:10,target_slug:'campaign'}).href).toBe('/campaign/campaign/comments#comment-42')
    expect(adminCommentTarget({...c,target_type:6,target_slug:'/faq'}).href).toBe('/faq#comment-42')
    expect(adminCommentTarget({...c,target_type:8}).href).toBe('#')
    const item={target_kind:0,target_id:42,comment_context:{target_type:9,target_id:99,target_slug:'news'}}
    expect(moderationTargetHref(item)).toBe('/blog/news#comment-42')
    expect(moderationTargetIdentity(item)).toBe('news')
    expect(moderationTargetHref({...item,comment_context:null})).toBe('/comments')
})
test('the two review queues keep independent status vocabularies and safe unknown fallbacks',()=>{
    expect([0,1,2,3,4].map(s=>moderationStatusLabel(s).label)).toEqual(['Pending','Approved','Rejected','Auto-approved','Auto-rejected'])
    expect([0,1,2,3].map(s=>adminCommentStatus(s).label)).toEqual(['OK','Pending','AI flagged','Hidden'])
    expect(moderationKindLabel(3)).toBe('Alliance bio')
    expect(moderationKindLabel(99)).toBe('Unknown')
    expect(moderationStatusLabel(99).label).toBe('99')
    expect(adminCommentStatus(99).label).toBe('?')
})
