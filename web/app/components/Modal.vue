<script setup lang="ts">
const props = defineProps<{
    modelValue: boolean
    title?: string
    maxWidth?: string
    /** Remove default body padding */
    noPadding?: boolean
    /** Position near top instead of centered */
    topAlign?: boolean
    /** Hide on screens below md breakpoint */
    hideBelowMd?: boolean
}>()

const emit = defineEmits(['update:modelValue'])

const close = () => emit('update:modelValue', false)

const onKeydown = (e: KeyboardEvent) => {
    if (e.key === 'Escape' && props.modelValue) close()
}

onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => document.removeEventListener('keydown', onKeydown))
</script>

<template>
    <Teleport to="body">
        <Transition
            enter-active-class="transition duration-200 ease-out"
            enter-from-class="opacity-0"
            enter-to-class="opacity-100"
            leave-active-class="transition duration-150 ease-in"
            leave-from-class="opacity-100"
            leave-to-class="opacity-0"
        >
            <div
                v-if="modelValue"
                class="fixed inset-0 z-[9999] flex justify-center p-4 bg-black/60 backdrop-blur-sm"
                :class="[
                    topAlign ? 'items-start pt-[15vh]' : 'items-center',
                    hideBelowMd ? 'hidden md:flex' : '',
                ]"
                @click.self="close"
            >
                <div
                    class="w-full rounded-xl border border-white/[0.10] bg-[#141414] shadow-2xl shadow-black/60 overflow-hidden flex flex-col"
                    :class="maxWidth ?? 'max-w-lg'"
                >
                    <!-- Header -->
                    <div v-if="title || $slots.header" class="flex items-center justify-between px-5 py-4 border-b border-white/[0.06]">
                        <slot name="header">
                            <h2 class="text-sm font-semibold text-white">{{ title }}</h2>
                        </slot>
                        <button
                            class="flex items-center justify-center w-7 h-7 rounded-md text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.08] transition-colors cursor-pointer"
                            @click="close"
                        >
                            <Icon name="lucide:x" class="text-sm" />
                        </button>
                    </div>

                    <!-- Body -->
                    <div
                        class="max-h-[70vh] overflow-y-auto"
                        :class="noPadding ? '' : 'px-5 py-4'"
                    >
                        <slot />
                    </div>

                    <!-- Footer -->
                    <div v-if="$slots.footer" class="px-5 py-3 border-t border-white/[0.06]">
                        <slot name="footer" />
                    </div>
                </div>
            </div>
        </Transition>
    </Teleport>
</template>
