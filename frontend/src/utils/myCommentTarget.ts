export interface MyCommentTargetSpec {
    label: string
    href: string
    icon: string
    chipBg: string
    chipText: string
    needsSlug?: boolean
    emptyLabel?: string
}

// The settings comment chips use the same definitions in Vue and Go. Keeping
// the routes here also covers newer slug-based battles, fits, posts and campaigns.
export const MY_COMMENT_TARGETS: Record<number, MyCommentTargetSpec> = {
    0: {label:'Unknown',href:'#',icon:'circle-help',chipBg:'bg-gray-500/[0.08]',chipText:'text-gray-300'},
    1: {label:'Killmail #{id}',href:'/kill/{id}#comment-{comment}',icon:'swords',chipBg:'bg-red-500/[0.08]',chipText:'text-red-300'},
    2: {label:'Character',href:'/character/{id}',icon:'user',chipBg:'bg-blue-500/[0.08]',chipText:'text-blue-300'},
    3: {label:'Corporation',href:'/corporation/{id}',icon:'building',chipBg:'bg-yellow-500/[0.08]',chipText:'text-yellow-400'},
    4: {label:'Alliance',href:'/alliance/{id}',icon:'flag',chipBg:'bg-blue-500/[0.14]',chipText:'text-blue-200'},
    5: {label:'Solar system',href:'/system/{id}',icon:'globe',chipBg:'bg-emerald-500/[0.08]',chipText:'text-emerald-300'},
    6: {label:'Page: {slug}',emptyLabel:'Page',href:'/{slug}',needsSlug:true,icon:'file-text',chipBg:'bg-gray-500/[0.08]',chipText:'text-gray-300'},
    7: {label:'Battle #{battle}',href:'/battle/{battle}/comments#comment-{comment}',icon:'shield',chipBg:'bg-yellow-500/[0.14]',chipText:'text-yellow-400'},
    8: {label:'Fit {slug}',emptyLabel:'Fit',href:'/fit/{slug}',needsSlug:true,icon:'wrench',chipBg:'bg-emerald-500/[0.14]',chipText:'text-emerald-200'},
    9: {label:'Blog post',href:'/blog/{slug}#comment-{comment}',needsSlug:true,icon:'file-text',chipBg:'bg-gray-500/[0.08]',chipText:'text-gray-300'},
    10: {label:'Campaign',href:'/campaign/{slug}/comments#comment-{comment}',needsSlug:true,icon:'flag',chipBg:'bg-blue-500/[0.14]',chipText:'text-blue-200'},
}

export function myCommentTarget(c: {id?: number; target_type:number; target_id:number; target_slug:string|null}) {
    const spec=MY_COMMENT_TARGETS[c.target_type]??MY_COMMENT_TARGETS[0]!
    const slug=c.target_slug??''
    const pathSlug=slug.replace(/^\/+/, '').split('/').map(encodeURIComponent).join('/')
    const labelValues={id:String(c.target_id),comment:String(c.id??c.target_id),slug,battle:slug||String(c.target_id)}
    const pathValues={...labelValues,slug:pathSlug,battle:slug?encodeURIComponent(slug):String(c.target_id)}
    const substitute=(text:string,values:Record<string,string>)=>text.replace(/\{(\w+)\}/g,(_,key:string)=>values[key]??'')
    return {label:spec.needsSlug&&!slug?(spec.emptyLabel??spec.label):substitute(spec.label,labelValues),href:spec.needsSlug&&!slug?'#':substitute(spec.href,pathValues),icon:'lucide:'+spec.icon,chipBg:spec.chipBg,chipText:spec.chipText}
}
