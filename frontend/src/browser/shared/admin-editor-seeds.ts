import {isRecord} from './json'
import type {AdminAnnouncement} from '../../utils/adminAnnouncement'
import type {AdminBlogPost} from '../../utils/adminBlog'

const text=(value:unknown)=>typeof value==='string'?value:''
const nullable=(value:unknown)=>typeof value==='string'?value:null
const number=(value:unknown)=>typeof value==='number'&&Number.isFinite(value)?value:0
function seed(text:string|undefined,key:string):unknown[]{try{const value:unknown=JSON.parse(text??'null');return isRecord(value)&&Array.isArray(value[key])?value[key]:[]}catch{return []}}
export function readAnnouncementSeed(source:string|undefined):AdminAnnouncement[]{
  return seed(source,'announcements').flatMap(value=>{
    if(!isRecord(value)||typeof value.id!=='number'||typeof value.title!=='string')return []
    return [{id:value.id,title:value.title,tier:number(value.tier),body_md:text(value.body_md),body_html:text(value.body_html),color:text(value.color),icon:nullable(value.icon),link_url:nullable(value.link_url),link_label:nullable(value.link_label),starts_at:text(value.starts_at),expires_at:text(value.expires_at),created_by:number(value.created_by),created_at:text(value.created_at),updated_at:text(value.updated_at),archived_at:nullable(value.archived_at)}]
  })
}
export function readBlogSeed(source:string|undefined):AdminBlogPost[]{
  return seed(source,'posts').flatMap(value=>{
    if(!isRecord(value)||typeof value.id!=='number'||typeof value.title!=='string')return []
    return [{id:value.id,title:value.title,slug:text(value.slug),body_md:text(value.body_md),body_html:text(value.body_html),status:number(value.status),excerpt:nullable(value.excerpt),cover_image_url:nullable(value.cover_image_url),tags:Array.isArray(value.tags)?value.tags.filter((tag):tag is string=>typeof tag==='string'):[],published_at:nullable(value.published_at),created_at:text(value.created_at),updated_at:text(value.updated_at),author_id:number(value.author_id),author_name:text(value.author_name),author_corporation_id:typeof value.author_corporation_id==='number'?value.author_corporation_id:null,author_corporation_name:nullable(value.author_corporation_name),author_alliance_id:typeof value.author_alliance_id==='number'?value.author_alliance_id:null,author_alliance_name:nullable(value.author_alliance_name)}]
  })
}
