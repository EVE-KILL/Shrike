<script setup lang="ts">
const props = defineProps<{
    /** Initial body for editing; empty string for new comments. */
    initial?: string
    placeholder?: string
    submitLabel?: string
    /** Called with the trimmed body. Should return a promise; component handles the loading state. */
    onSubmit: (body: string) => Promise<void>
    onCancel?: () => void
}>()

const body = ref(props.initial ?? '')
const submitting = ref(false)
const error = ref<string | null>(null)
const tab = ref<'write' | 'preview'>('write')
const previewHtml = ref('')
const textareaRef = ref<HTMLTextAreaElement | null>(null)
/** Cached selection — populated on every textarea selection change so the
    toolbar can still operate on it after the user clicks a toolbar button. */
const cachedSelection = ref<{ start: number; end: number }>({ start: 0, end: 0 })

function rememberSelection() {
    const ta = textareaRef.value
    if (!ta) return
    cachedSelection.value = { start: ta.selectionStart, end: ta.selectionEnd }
}

const charCount = computed(() => body.value.length)
const tooLong = computed(() => charCount.value > 2000)
const tooShort = computed(() => body.value.trim().length < 2)
const canSubmit = computed(() => !submitting.value && !tooLong.value && !tooShort.value)

/**
 * Load a server-rendered preview. We hit /api/comments/preview rather than
 * rendering locally so the preview tab shows EXACTLY what the server would
 * store on submit — markdown pipeline, sanitization, and embed transforms
 * (YouTube/Imgur/Giphy/Tenor) all match. No client-side drift.
 */
const previewLoading = ref(false)

async function loadPreview() {
    if (!body.value.trim()) {
        previewHtml.value = ''
        return
    }
    previewLoading.value = true
    try {
        const res = await apiFetch<{ html: string; error?: string }>('/api/comments/preview', {
            method: 'POST',
            body: { body_md: body.value },
        })
        previewHtml.value = res.error
            ? `<em class="text-amber-400">${res.error}</em>`
            : res.html
    } catch (err: any) {
        previewHtml.value = `<em class="text-red-400">Preview failed: ${err?.statusMessage || err?.message || 'unknown error'}</em>`
    } finally {
        previewLoading.value = false
    }
}

watch(tab, (t) => {
    if (t === 'preview') loadPreview()
})

/**
 * Insert text around the current selection (or at the cursor) and put the
 * cursor / selection back inside the wrapped text. Uses the cached selection
 * because toolbar buttons fire AFTER the textarea has lost focus on click —
 * we offset that by also using @mousedown.prevent on the buttons so the
 * textarea actually keeps focus, but the cache is still useful as a fallback.
 */
function insert(before: string, after: string = '') {
    const ta = textareaRef.value
    if (!ta) return

    const live = document.activeElement === ta
    const start = live ? ta.selectionStart : cachedSelection.value.start
    const end = live ? ta.selectionEnd : cachedSelection.value.end

    const sel = body.value.slice(start, end)
    body.value = body.value.slice(0, start) + before + sel + after + body.value.slice(end)

    nextTick(() => {
        ta.focus()
        ta.selectionStart = start + before.length
        ta.selectionEnd = start + before.length + sel.length
        rememberSelection()
    })
}

/** Insert a literal string at the cursor (no wrapping). Used by the URL modal. */
function insertAtCursor(text: string) {
    const ta = textareaRef.value
    if (!ta) return
    const start = cachedSelection.value.start
    const end = cachedSelection.value.end
    body.value = body.value.slice(0, start) + text + body.value.slice(end)
    nextTick(() => {
        ta.focus()
        const pos = start + text.length
        ta.selectionStart = ta.selectionEnd = pos
        rememberSelection()
    })
}

// ── Klipy GIF picker ─────────────────────────────────────────────────────────

const klipyOpen = ref(false)

function onKlipyInsert(markdown: string) {
    insertAtCursor(markdown)
}

// ── URL embed modal ──────────────────────────────────────────────────────────

type EmbedKind = 'image' | 'youtube' | 'imgur' | 'giphy' | 'tenor'

const embedModalOpen = ref(false)
const embedKind = ref<EmbedKind>('image')
const embedUrl = ref('')
const embedError = ref<string | null>(null)

function openEmbedModal(kind: EmbedKind) {
    embedKind.value = kind
    embedUrl.value = ''
    embedError.value = null
    embedModalOpen.value = true
}

const EMBED_CONFIG: Record<EmbedKind, {
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

const currentConfig = computed(() => EMBED_CONFIG[embedKind.value])

const liveValidationError = computed(() => {
    if (!embedUrl.value.trim()) return null
    return currentConfig.value.validate(embedUrl.value.trim())
})

/** Try to derive a thumbnail/preview URL for the input. */
const previewSrc = computed<string | null>(() => {
    const url = embedUrl.value.trim()
    if (!url || liveValidationError.value) return null

    if (embedKind.value === 'youtube') {
        const m = url.match(/(?:youtube\.com\/(?:watch\?v=|embed\/|shorts\/)|youtu\.be\/)([\w-]{11})/)
        return m ? `https://img.youtube.com/vi/${m[1]}/hqdefault.jpg` : null
    }
    if (embedKind.value === 'imgur') {
        const m = url.match(/imgur\.com\/(?:gallery\/|a\/)?([a-zA-Z0-9]{5,8})(?:\.(jpg|jpeg|png|gif|webp))?$/)
        if (m) {
            const ext = m[2] || 'jpg'
            return `https://i.imgur.com/${m[1]}.${ext}`
        }
        return null
    }
    if (embedKind.value === 'image') return url
    return null
})

function confirmEmbed() {
    const url = embedUrl.value.trim()
    if (!url) {
        embedError.value = 'URL required'
        return
    }
    const err = currentConfig.value.validate(url)
    if (err) {
        embedError.value = err
        return
    }
    insertAtCursor(currentConfig.value.toMarkdown(url))
    embedModalOpen.value = false
}

// ── Submit / cancel ──────────────────────────────────────────────────────────

async function submit() {
    if (!canSubmit.value) return
    submitting.value = true
    error.value = null
    try {
        await props.onSubmit(body.value.trim())
        useAnalytics().track('comment.post', {
            length: body.value.trim().length,
            edit: !!props.initial,
        })
        body.value = ''
        tab.value = 'write'
    } catch (err: any) {
        error.value = extractFetchError(err, 'Failed to submit')
    } finally {
        submitting.value = false
    }
}

function cancel() {
    body.value = props.initial ?? ''
    error.value = null
    props.onCancel?.()
}
</script>

<template>
    <div class="comment-input rounded-lg border border-white/[0.08] bg-black/30">
        <!-- Tabs -->
        <div class="flex items-center gap-1 px-2 pt-2 border-b border-white/[0.06] flex-wrap">
            <button
                class="px-3 py-1 text-xs rounded-md transition-colors cursor-pointer"
                :class="tab === 'write' ? 'bg-white/[0.08] text-white' : 'text-gray-500 hover:text-gray-300'"
                @click="tab = 'write'"
            >
                Write
            </button>
            <button
                class="px-3 py-1 text-xs rounded-md transition-colors cursor-pointer"
                :class="tab === 'preview' ? 'bg-white/[0.08] text-white' : 'text-gray-500 hover:text-gray-300'"
                @click="tab = 'preview'"
            >
                Preview
            </button>
            <div class="flex-1" />
            <!-- Toolbar (write mode only).
                 @mousedown.prevent stops the textarea from losing focus when
                 the button is clicked, so the live selection is preserved. -->
            <div v-if="tab === 'write'" class="flex items-center gap-0.5 flex-wrap">
                <button type="button" class="toolbar-btn" v-tooltip="'Bold'" @mousedown.prevent @click="insert('**', '**')"><Icon name="lucide:bold" class="text-xs" /></button>
                <button type="button" class="toolbar-btn" v-tooltip="'Italic'" @mousedown.prevent @click="insert('_', '_')"><Icon name="lucide:italic" class="text-xs" /></button>
                <button type="button" class="toolbar-btn" v-tooltip="'Strikethrough'" @mousedown.prevent @click="insert('~~', '~~')"><Icon name="lucide:strikethrough" class="text-xs" /></button>
                <button type="button" class="toolbar-btn" v-tooltip="'Code'" @mousedown.prevent @click="insert('`', '`')"><Icon name="lucide:code" class="text-xs" /></button>
                <button type="button" class="toolbar-btn" v-tooltip="'Quote'" @mousedown.prevent @click="insert('\n> ', '')"><Icon name="lucide:quote" class="text-xs" /></button>
                <button type="button" class="toolbar-btn" v-tooltip="'Link'" @mousedown.prevent @click="insert('[', '](https://)')"><Icon name="lucide:link" class="text-xs" /></button>
                <span class="toolbar-divider" />
                <button type="button" class="toolbar-btn" v-tooltip="'Insert image'" @mousedown.prevent @click="openEmbedModal('image')"><Icon name="lucide:image" class="text-xs" /></button>
                <button type="button" class="toolbar-btn hover:!text-red-400" v-tooltip="'Insert YouTube video'" @mousedown.prevent @click="openEmbedModal('youtube')"><Icon name="lucide:youtube" class="text-xs" /></button>
                <button type="button" class="toolbar-btn hover:!text-green-400" v-tooltip="'Insert Imgur'" @mousedown.prevent @click="openEmbedModal('imgur')"><Icon name="simple-icons:imgur" class="text-xs" /></button>
                <button type="button" class="toolbar-btn hover:!text-purple-400" v-tooltip="'Pick a GIF'" @mousedown.prevent @click="klipyOpen = true"><Icon name="lucide:image-play" class="text-xs" /></button>
            </div>
        </div>

        <!-- Body -->
        <div class="p-2">
            <textarea
                v-if="tab === 'write'"
                ref="textareaRef"
                v-model="body"
                :placeholder="placeholder ?? 'Write a comment… Markdown supported. Drop YouTube/Imgur/Giphy links to embed.'"
                class="w-full min-h-[80px] max-h-[300px] bg-transparent text-sm text-white placeholder-gray-600 focus:outline-none resize-y"
                :class="{ 'text-red-400': tooLong }"
                @keydown.ctrl.enter.prevent="submit"
                @keydown.meta.enter.prevent="submit"
                @keydown.ctrl.g.prevent="klipyOpen = true"
                @keydown.meta.g.prevent="klipyOpen = true"
                @select="rememberSelection"
                @click="rememberSelection"
                @keyup="rememberSelection"
                @blur="rememberSelection"
            />
            <div
                v-else
                class="comment-body min-h-[80px] text-sm text-gray-200 prose prose-invert max-w-none prose-sm"
            >
                <div v-if="previewLoading" class="text-xs text-gray-500 italic flex items-center gap-2">
                    <Icon name="lucide:loader-2" class="animate-spin" />
                    Rendering preview…
                </div>
                <div
                    v-else-if="previewHtml"
                    v-html="previewHtml"
                />
                <em v-else class="text-gray-600">Nothing to preview</em>
            </div>
        </div>

        <!-- Error banner — own row so a long moderation rejection isn't truncated -->
        <div
            v-if="error"
            class="mx-2 mb-2 px-2.5 py-1.5 rounded-md bg-red-500/[0.08] border border-red-500/20 text-xs text-red-300 flex items-start gap-2"
        >
            <Icon name="lucide:alert-triangle" class="text-sm flex-shrink-0 mt-px" />
            <span class="flex-1">{{ error }}</span>
            <button
                class="text-red-400/70 hover:text-red-300 cursor-pointer flex-shrink-0"
                v-tooltip="'Dismiss'"
                @click="error = null"
            >
                <Icon name="lucide:x" class="text-xs" />
            </button>
        </div>

        <!-- Footer -->
        <div class="flex items-center gap-2 px-2 pb-2 text-xs">
            <span :class="tooLong ? 'text-red-400' : 'text-gray-600'">
                {{ charCount }} / 2000
            </span>
            <div class="flex-1" />
            <button
                v-if="onCancel || initial"
                class="px-3 py-1 rounded-md text-gray-400 hover:text-white hover:bg-white/[0.06] transition-colors cursor-pointer"
                @click="cancel"
            >
                Cancel
            </button>
            <button
                class="px-3 py-1 rounded-md bg-blue-500/[0.15] text-blue-300 hover:bg-blue-500/[0.25] disabled:opacity-40 disabled:cursor-not-allowed transition-colors cursor-pointer"
                :disabled="!canSubmit"
                @click="submit"
            >
                {{ submitting ? 'Posting…' : (submitLabel ?? 'Post') }}
            </button>
        </div>

        <!-- ═══════════════ KLIPY GIF PICKER ═══════════════ -->
        <CommentsKlipyPicker v-model="klipyOpen" :on-insert="onKlipyInsert" />

        <!-- ═══════════════ EMBED URL MODAL ═══════════════ -->
        <Modal v-model="embedModalOpen" :title="currentConfig.title">
            <template #header>
                <h2 class="text-sm font-semibold text-white flex items-center gap-2">
                    <Icon :name="currentConfig.icon" :class="currentConfig.accentColor" />
                    {{ currentConfig.title }}
                </h2>
            </template>

            <div class="space-y-3">
                <div>
                    <label class="block text-xs font-semibold text-gray-400 mb-1.5 uppercase tracking-wide">URL</label>
                    <input
                        v-model="embedUrl"
                        type="url"
                        :placeholder="currentConfig.placeholder"
                        class="w-full px-3 py-2 bg-black/40 border border-white/[0.08] rounded-md text-sm text-white placeholder-gray-600 focus:outline-none focus:border-blue-500/50"
                        autofocus
                        @keydown.enter.prevent="confirmEmbed"
                    >
                    <p v-if="liveValidationError" class="mt-1.5 text-xs text-amber-400">
                        {{ liveValidationError }}
                    </p>
                    <p v-else-if="embedError" class="mt-1.5 text-xs text-red-400">
                        {{ embedError }}
                    </p>
                </div>

                <!-- Live preview -->
                <div v-if="previewSrc" class="rounded-md border border-white/[0.08] bg-black/40 p-3">
                    <div class="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">Preview</div>
                    <img
                        :src="previewSrc"
                        alt="preview"
                        class="max-w-full max-h-60 mx-auto rounded object-contain"
                        @error="(e: Event) => ((e.target as HTMLImageElement).style.display = 'none')"
                    >
                </div>
                <div v-else-if="embedUrl && !liveValidationError" class="rounded-md border border-white/[0.08] bg-black/40 p-3">
                    <div class="text-xs text-gray-500 flex items-center gap-2">
                        <Icon :name="currentConfig.icon" :class="currentConfig.accentColor" />
                        Embed will render in the comment.
                    </div>
                </div>
            </div>

            <template #footer>
                <div class="flex items-center justify-end gap-2">
                    <button
                        class="px-3 py-1 rounded-md text-sm text-gray-400 hover:text-white hover:bg-white/[0.06] cursor-pointer transition-colors"
                        @click="embedModalOpen = false"
                    >
                        Cancel
                    </button>
                    <button
                        class="px-3 py-1 rounded-md text-sm bg-blue-500/[0.15] text-blue-300 hover:bg-blue-500/[0.25] disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer transition-colors"
                        :disabled="!embedUrl.trim() || !!liveValidationError"
                        @click="confirmEmbed"
                    >
                        Insert
                    </button>
                </div>
            </template>
        </Modal>
    </div>
</template>

<style scoped>
/* Preview-tab embed styling — matches Comment.vue so the preview looks
   identical to the rendered comment. */
.comment-body :deep(.embed) {
    margin: 0.5rem 0;
    border-radius: 0.5rem;
    overflow: hidden;
}
.comment-body :deep(.embed-youtube) {
    position: relative;
    padding-bottom: 56.25%;
    height: 0;
}
.comment-body :deep(.embed-youtube iframe) {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    border: 0;
}
.comment-body :deep(.embed-image img),
.comment-body :deep(.embed-gif img) {
    max-width: 100%;
    max-height: 400px;
    border-radius: 0.5rem;
}
.comment-body :deep(.embed-video video) {
    max-width: 100%;
    max-height: 400px;
    border-radius: 0.5rem;
}
.comment-body :deep(.embed-media) {
    margin: 0.5rem 0;
}
.comment-body :deep(.embed-media video),
.comment-body :deep(.embed-media img) {
    max-width: 100%;
    max-height: 600px;
    border-radius: 0.5rem;
    display: block;
    background: #000;
}

.toolbar-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 1.5rem;
    height: 1.5rem;
    border-radius: 0.25rem;
    color: rgb(107, 114, 128);
    cursor: pointer;
    transition: all 0.15s;
}
.toolbar-btn:hover {
    color: rgb(96, 165, 250);
    background: rgba(255, 255, 255, 0.06);
}
.toolbar-divider {
    width: 1px;
    height: 1rem;
    background: rgba(255, 255, 255, 0.08);
    margin: 0 0.25rem;
}
</style>
