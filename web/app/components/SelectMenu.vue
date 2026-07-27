<script setup lang="ts">
import { onClickOutside } from '@vueuse/core'

const props = defineProps<{
    modelValue: string
    options: { value: string; label: string }[]
    placeholder?: string
}>()

const emit = defineEmits(['update:modelValue'])

const isOpen = ref(false)
const containerRef = ref<HTMLElement | null>(null)

onClickOutside(containerRef, () => { isOpen.value = false })

const selectedLabel = computed(() => {
    if (!props.modelValue) return props.placeholder ?? 'Select...'
    return props.options.find(o => o.value === props.modelValue)?.label ?? props.modelValue
})

const select = (value: string) => {
    emit('update:modelValue', value)
    isOpen.value = false
}

const onKeydown = (e: KeyboardEvent) => {
    if (e.key === 'Escape' && isOpen.value) isOpen.value = false
}

onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => document.removeEventListener('keydown', onKeydown))
</script>

<template>
    <div ref="containerRef" class="relative inline-flex">
        <button
            class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs transition-colors cursor-pointer border"
            :class="isOpen
                ? 'bg-blue-500/10 text-blue-400 border-white/[0.12]'
                : modelValue
                    ? 'bg-white/[0.06] text-gray-300 border-white/[0.08] hover:border-white/[0.12]'
                    : 'bg-white/[0.04] text-gray-500 border-white/[0.06] hover:border-white/[0.10] hover:text-blue-400'"
            @click="isOpen = !isOpen"
        >
            <span class="truncate max-w-[120px]">{{ selectedLabel }}</span>
            <Icon name="lucide:chevron-down" class="text-[10px] opacity-60 transition-transform" :class="isOpen && 'rotate-180'" />
        </button>

        <Transition
            enter-active-class="transition duration-150 ease-out"
            enter-from-class="opacity-0 translate-y-1 scale-[0.98]"
            enter-to-class="opacity-100 translate-y-0 scale-100"
            leave-active-class="transition duration-100 ease-in"
            leave-from-class="opacity-100 translate-y-0 scale-100"
            leave-to-class="opacity-0 translate-y-1 scale-[0.98]"
        >
            <div
                v-if="isOpen"
                class="absolute top-full left-0 mt-1.5 z-50 rounded-xl border border-white/[0.10] bg-[#141414]/95 backdrop-blur-2xl shadow-2xl shadow-black/60 overflow-y-auto max-h-[50vh] p-1.5 min-w-[10rem]"
            >
                <button
                    v-for="opt in options"
                    :key="opt.value"
                    class="flex items-center w-full px-2.5 py-1.5 rounded-md text-xs transition-colors text-left cursor-pointer"
                    :class="modelValue === opt.value
                        ? 'text-white bg-blue-500/[0.12]'
                        : 'text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06]'"
                    @click="select(opt.value)"
                >
                    {{ opt.label }}
                </button>
            </div>
        </Transition>
    </div>
</template>
