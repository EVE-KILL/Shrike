<script setup lang="ts">
import type {
    EveImageFormat,
    EveImageSize,
} from '#shared/utils/eveImage'
import {
    eveImageSizeFromURL,
    eveImageSrcset,
    eveImageURL,
} from '#shared/utils/eveImage'

defineOptions({ inheritAttrs: false })

const props = withDefaults(defineProps<{
    src: string
    alt: string
    size?: EveImageSize
    width?: string | number
    height?: string | number
    format?: EveImageFormat
    responsive?: boolean
    fallbackSrc?: string
    loading?: 'eager' | 'lazy'
    decoding?: 'async' | 'sync' | 'auto'
    fetchpriority?: 'high' | 'low' | 'auto'
}>(), {
    format: 'auto',
    responsive: true,
    loading: 'lazy',
    decoding: 'async',
    fetchpriority: 'auto',
})

const emit = defineEmits<{
    error: [event: Event]
    load: [event: Event]
}>()

const showingFallback = ref(false)

watch(
    () => [props.src, props.fallbackSrc],
    () => {
        showingFallback.value = false
    },
)

const activeSource = computed(() =>
    showingFallback.value && props.fallbackSrc
        ? props.fallbackSrc
        : props.src,
)
const requestedSize = computed(
    () => props.size ?? eveImageSizeFromURL(activeSource.value),
)
const resolvedSource = computed(() => eveImageURL(activeSource.value, {
    size: requestedSize.value,
    format: props.format,
}))
const resolvedSrcset = computed(() => {
    if (!props.responsive) return undefined
    return eveImageSrcset(
        activeSource.value,
        requestedSize.value,
        props.format,
    )
})
const intrinsicWidth = computed(() => props.width ?? requestedSize.value)
const intrinsicHeight = computed(() => props.height ?? requestedSize.value)

function handleError(event: Event) {
    if (!showingFallback.value && props.fallbackSrc) {
        showingFallback.value = true
        return
    }
    emit('error', event)
}
</script>

<template>
    <img
        v-bind="$attrs"
        :src="resolvedSource"
        :srcset="resolvedSrcset"
        :alt="alt"
        :width="intrinsicWidth"
        :height="intrinsicHeight"
        :loading="loading"
        :decoding="decoding"
        :fetchpriority="fetchpriority"
        @error="handleError"
        @load="emit('load', $event)"
    >
</template>
