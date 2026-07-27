<script setup lang="ts">
import type { BioEditorState } from '~/components/settings/BioEditor.vue'

// Settings → Description section (custom bios). Extracted verbatim from
// pages/settings/[[tab]].vue. This component only mounts when the tab is
// first opened (the parent keeps it alive via <KeepAlive>), so the
// `immediate: true` fetch below fires lazily on first visit — replacing the
// old `immediate: activeSection === 'description'` + activeSection watch.

// ── Description (custom bios) ─────────────────────────────────────────────
// Reuses the same "save state + flash toast" pattern as the preferences block.
// Data comes from /api/user/manageable-entities, which returns the current
// user's character (always), their corporation (if any), and their alliance
// (if any) — each with a `canEdit` flag and the current stored bio. We prefill
// the editor with `custom_description ?? esi_description` so first-time edits
// start from the user's existing ESI bio instead of an empty box.

type DescriptionFormat = 'markdown' | 'eve_html'
type DescriptionEntity = 'character' | 'corporation' | 'alliance'

interface PendingSubmission {
    body: string
    body_format: string
    submitted_at: string
}

interface ManageableResponse {
    character: {
        id: number
        name: string
        esi_description: string | null
        custom_description: string | null
        custom_description_format: string
        canEdit: true
        pending_submission: PendingSubmission | null
    }
    corporation: null | {
        id: number
        name: string
        ticker: string
        ceo_id: number | null
        ceo_name: string | null
        esi_description: string | null
        custom_description: string | null
        custom_description_format: string
        canEdit: boolean
        pending_submission: PendingSubmission | null
    }
    alliance: null | {
        id: number
        name: string
        ticker: string
        executor_corporation_id: number | null
        executor_ceo_id: number | null
        executor_ceo_name: string | null
        custom_description: string | null
        custom_description_format: string
        canEdit: boolean
        pending_submission: PendingSubmission | null
    }
}

const { data: manageable, refresh: refreshManageable } = useApiFetch<ManageableResponse>('/api/user/manageable-entities', {
    immediate: true,
    lazy: true,
})

const descChar = reactive<BioEditorState>({ text: '', format: 'markdown', saving: false, submitted: false, error: '' })
const descCorp = reactive<BioEditorState>({ text: '', format: 'markdown', saving: false, submitted: false, error: '' })
const descAlly = reactive<BioEditorState>({ text: '', format: 'markdown', saving: false, submitted: false, error: '' })

// Seed the editors from the manageable payload. Preference order:
//   1. pending_submission.body (the last thing the user submitted, awaiting
//      admin review — so they can continue iterating on the draft rather than
//      starting over from the published version)
//   2. custom_description (published, admin-approved bio)
//   3. esi_description (ESI fallback for character/corporation)
//   4. empty string (first-ever edit)
const seedDescriptions = () => {
    const m = manageable.value
    if (!m) return

    descChar.text = m.character.pending_submission?.body
        ?? m.character.custom_description
        ?? m.character.esi_description
        ?? ''
    descChar.format = (m.character.pending_submission?.body_format
        ?? m.character.custom_description_format) as DescriptionFormat
    descChar.error = ''

    if (m.corporation?.canEdit) {
        descCorp.text = m.corporation.pending_submission?.body
            ?? m.corporation.custom_description
            ?? m.corporation.esi_description
            ?? ''
        descCorp.format = (m.corporation.pending_submission?.body_format
            ?? m.corporation.custom_description_format) as DescriptionFormat
        descCorp.error = ''
    }
    if (m.alliance?.canEdit) {
        descAlly.text = m.alliance.pending_submission?.body
            ?? m.alliance.custom_description
            ?? ''
        descAlly.format = (m.alliance.pending_submission?.body_format
            ?? m.alliance.custom_description_format) as DescriptionFormat
        descAlly.error = ''
    }
}

watch(manageable, seedDescriptions, { immediate: true })

const saveDescription = async (entity: DescriptionEntity) => {
    const state = entity === 'character' ? descChar : entity === 'corporation' ? descCorp : descAlly
    state.saving = true
    state.error = ''
    state.submitted = false
    try {
        await apiFetch('/api/user/descriptions', {
            method: 'PUT',
            body: {
                entity,
                description: state.text,
                format: state.format,
            },
        })
        state.submitted = true
        setTimeout(() => { state.submitted = false }, 3000)
        // Refresh so pending_submission updates and the "pending review"
        // banner renders immediately.
        await refreshManageable()
    } catch (err: unknown) {
        const e = err as { data?: { statusMessage?: string }; statusMessage?: string; message?: string }
        state.error = extractFetchError(e, 'Failed to save')
    } finally {
        state.saving = false
    }
}
</script>

<template>
    <div class="space-y-4">
        <div class="glass-panel p-5">
            <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-3">Custom Bios</h2>
            <p class="text-xs text-gray-400 leading-relaxed">
                Set a custom bio for your character, and — if you're a CEO or alliance executor CEO —
                for your corporation or alliance. These override the ESI-sourced description on entity
                pages. Markdown is recommended for new bios; EVE HTML is supported for pasting existing
                in-game descriptions verbatim.
            </p>
            <p class="text-xs text-gray-500 mt-2">
                <Icon name="lucide:info" class="inline w-3 h-3 -mt-0.5" />
                Every submission is reviewed manually by an admin before it's published. Once approved,
                changes take effect within a few minutes as page caches refresh. Submitting an empty
                bio clears your published one immediately.
            </p>
        </div>

        <SettingsBioEditor
            v-if="manageable?.character"
            title="Character Bio"
            :subtitle="manageable.character.name"
            :state="descChar"
            :pending="!!manageable.character.pending_submission"
            :rows="8"
            placeholder="Write something about yourself..."
            @save="saveDescription('character')"
        />

        <SettingsBioEditor
            v-if="manageable?.corporation?.canEdit"
            title="Corporation Bio"
            :subtitle="`[${manageable.corporation.ticker}] ${manageable.corporation.name}`"
            :state="descCorp"
            :pending="!!manageable.corporation.pending_submission"
            placeholder="Describe your corporation..."
            @save="saveDescription('corporation')"
        />
        <div v-else-if="manageable?.corporation" class="rounded-lg bg-white/[0.02] border border-white/[0.06] p-5 opacity-70">
            <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-2">Corporation Bio</h2>
            <p class="text-xs text-gray-400">
                You're a member of <span class="text-gray-300">[{{ manageable.corporation.ticker }}] {{ manageable.corporation.name }}</span>,
                but only the CEO<span v-if="manageable.corporation.ceo_name"> ({{ manageable.corporation.ceo_name }})</span> can edit the corporation bio.
            </p>
        </div>

        <SettingsBioEditor
            v-if="manageable?.alliance?.canEdit"
            title="Alliance Bio"
            :subtitle="`[${manageable.alliance.ticker}] ${manageable.alliance.name}`"
            :state="descAlly"
            :pending="!!manageable.alliance.pending_submission"
            placeholder="Describe your alliance..."
            @save="saveDescription('alliance')"
        />
        <div v-else-if="manageable?.alliance" class="rounded-lg bg-white/[0.02] border border-white/[0.06] p-5 opacity-70">
            <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-2">Alliance Bio</h2>
            <p class="text-xs text-gray-400">
                You're a member of <span class="text-gray-300">[{{ manageable.alliance.ticker }}] {{ manageable.alliance.name }}</span>,
                but only the executor corporation's CEO<span v-if="manageable.alliance.executor_ceo_name"> ({{ manageable.alliance.executor_ceo_name }})</span> can edit the alliance bio.
            </p>
        </div>
    </div>
</template>
