// Shared with CommentInput.vue; keep provider validation and thumbnails identical.
import {isRecord} from '../../src/browser/shared/json'
export type EmbedKind = 'image' | 'youtube' | 'imgur' | 'giphy' | 'tenor'

export const EMBED_CONFIG: Record<EmbedKind, {
    title: string
    placeholder: string
    icon: string
    accentColor: string
    /** Returns null if URL is valid for this kind, otherwise an error message. */
    validate: (url: string) => string | null
    /** Returns the markdown to insert. */
    toMarkdown: (url: string) => string
}> = {
    image: {
        title: 'Insert image',
        placeholder: 'https://example.com/image.png',
        icon: 'lucide:image',
        accentColor: 'text-blue-400',
        validate: (u) => /^https?:\/\/.+\.(jpg|jpeg|png|gif|webp|svg)(\?.*)?$/i.test(u)
            ? null
            : 'Must be a direct image URL ending in .jpg, .png, .gif, .webp, or .svg',
        toMarkdown: (u) => `[image](${u})`,
    },
    youtube: {
        title: 'Insert YouTube video',
        placeholder: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ',
        icon: 'lucide:youtube',
        accentColor: 'text-red-400',
        validate: (u) => /^(https?:\/\/)?(www\.)?(youtube\.com\/(watch\?v=|embed\/|shorts\/)|youtu\.be\/)[\w-]{11}/.test(u)
            ? null
            : 'Must be a YouTube video URL (youtube.com/watch, youtu.be, or youtube.com/shorts)',
        toMarkdown: (u) => `[YouTube](${u})`,
    },
    imgur: {
        title: 'Insert Imgur image',
        placeholder: 'https://imgur.com/abc123 or https://i.imgur.com/abc123.jpg',
        icon: 'simple-icons:imgur',
        accentColor: 'text-green-400',
        validate: (u) => /^https?:\/\/(i\.|m\.)?imgur\.com\/(?:gallery\/|a\/)?[a-zA-Z0-9]{5,8}(\.(jpg|jpeg|png|gif|webp))?$/.test(u)
            ? null
            : 'Must be an Imgur URL (imgur.com/abc123 or i.imgur.com/abc123.jpg)',
        toMarkdown: (u) => `[Imgur](${u})`,
    },
    giphy: {
        title: 'Insert Giphy',
        placeholder: 'https://giphy.com/gifs/example-abc123',
        icon: 'lucide:image-play',
        accentColor: 'text-purple-400',
        validate: (u) => /^https?:\/\/(media\d?\.)?giphy\.com\/(media\/|gifs\/)/.test(u)
            ? null
            : 'Must be a Giphy URL (giphy.com/gifs/...)',
        toMarkdown: (u) => `[GIF](${u})`,
    },
    tenor: {
        title: 'Insert Tenor GIF',
        placeholder: 'https://tenor.com/view/example-1234567',
        icon: 'lucide:image-play',
        accentColor: 'text-cyan-400',
        validate: (u) => /^https?:\/\/(www\.)?tenor\.com\/view\/[\w-]+-\d+/.test(u)
            || /^https?:\/\/media\d?\.tenor\.com\/[\w-]+\/[\w-]+\.gif$/i.test(u)
            ? null
            : 'Must be a Tenor URL (tenor.com/view/... or media.tenor.com/...)',
        toMarkdown: (u) => `[GIF](${u})`,
    },
}


export function commentEmbedPreview(kind: EmbedKind, raw: string): string | null {
    const url = raw.trim()
    if (!url || EMBED_CONFIG[kind].validate(url)) return null
    if (kind === 'youtube') {
        const m = url.match(/(?:youtube\.com\/(?:watch\?v=|embed\/|shorts\/)|youtu\.be\/)([\w-]{11})/)
        return m ? `https://img.youtube.com/vi/${m[1]}/hqdefault.jpg` : null
    }
    if (kind === 'imgur') {
        const m = url.match(/imgur\.com\/(?:gallery\/|a\/)?([a-zA-Z0-9]{5,8})(?:\.(jpg|jpeg|png|gif|webp))?$/)
        if (m) {
            const ext = m[2] || 'jpg'
            return `https://i.imgur.com/${m[1]}.${ext}`
        }
        return null
    }
    if (kind === 'image') return url
    return null
}

export function insertCommentText(body: string, start: number, end: number, before: string, after = '', wrap = true) {
    const selection = wrap ? body.slice(start, end) : ''
    return {
        body: body.slice(0, start) + before + selection + after + body.slice(end),
        start: start + before.length,
        end: start + before.length + selection.length,
    }
}

export interface KlipyFavorite {
    id: number
    title: string
    preview_url: string
    insert_url: string
    width: number
    height: number
    added_at: number
}
export const KLIPY_FAVORITES_KEY = 'ek_klipy_favs'
export const KLIPY_MAX_FAVORITES = 100
export function parseKlipyFavorites(raw: string | null): KlipyFavorite[] {
    try {
        const parsed:unknown = JSON.parse(raw || '[]')
        return Array.isArray(parsed) ? parsed.flatMap(value=>{
            if(!isRecord(value)||typeof value.id!=='number'||!Number.isSafeInteger(value.id)||typeof value.preview_url!=='string'||typeof value.insert_url!=='string')return []
            const number=(raw:unknown)=>typeof raw==='number'&&Number.isFinite(raw)&&raw>=0?raw:0
            return [{id:value.id,title:typeof value.title==='string'?value.title:'',preview_url:value.preview_url,insert_url:value.insert_url,width:number(value.width),height:number(value.height),added_at:number(value.added_at)}]
        }) : []
    } catch { return [] }
}
export function addKlipyFavorite(list: readonly KlipyFavorite[], favorite: Omit<KlipyFavorite, 'added_at'>, now = Date.now()): KlipyFavorite[] {
    if (list.some(item => item.id === favorite.id)) return [...list]
    return [{ ...favorite, added_at: now }, ...list].slice(0, KLIPY_MAX_FAVORITES)
}
