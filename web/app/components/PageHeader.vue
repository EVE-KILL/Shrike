<script setup lang="ts">
defineProps<{
  title: string;
  description?: string;
  icon: string;
  eyebrow?: string;
  imageSrc?: string;
  imageMode?: "accent" | "background";
}>();
</script>

<template>
  <header class="page-header hero-surface glass-panel relative overflow-hidden p-5">
    <img
      v-if="imageSrc && imageMode === 'background'"
      :src="imageSrc"
      alt=""
      class="page-header__background pointer-events-none absolute inset-0 h-full w-full object-cover"
    />
    <div
      v-if="imageSrc && imageMode === 'background'"
      class="page-header__background-shade pointer-events-none absolute inset-0"
    />
    <img
      v-if="imageSrc && imageMode !== 'background'"
      :src="imageSrc"
      alt=""
      class="pointer-events-none absolute -right-6 -top-10 hidden h-44 w-44 object-contain opacity-[0.12] sm:block"
    />
    <div class="relative flex flex-wrap items-start justify-between gap-4">
      <div class="flex min-w-0 items-start gap-3">
        <span
          class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-blue-400/15 bg-blue-500/10"
        >
          <Icon :name="icon" class="text-lg text-blue-400" />
        </span>
        <div class="min-w-0">
          <p
            v-if="eyebrow"
            class="text-fine font-bold uppercase tracking-[0.16em] text-blue-400/70"
          >
            {{ eyebrow }}
          </p>
          <h1 class="text-xl font-bold text-white">{{ title }}</h1>
          <div
            v-if="$slots.description || description"
            class="mt-1.5 max-w-3xl text-sm leading-relaxed text-gray-500"
          >
            <slot name="description">{{ description }}</slot>
          </div>
        </div>
      </div>
      <div v-if="$slots.actions" class="relative shrink-0">
        <slot name="actions" />
      </div>
    </div>
    <div
      v-if="$slots.meta"
      class="relative mt-4 border-t border-white/[0.06] pt-3"
    >
      <slot name="meta" />
    </div>
  </header>
</template>

<style scoped>

.page-header__background {
  z-index: -3;
  filter: blur(12px) saturate(0.9);
  opacity: 0.46;
  transform: scale(1.06);
}

.page-header__background-shade {
  z-index: -2;
  background:
    linear-gradient(
      90deg,
      rgba(8, 10, 15, 0.92) 0%,
      rgba(8, 10, 15, 0.68) 52%,
      rgba(8, 10, 15, 0.3) 100%
    ),
    linear-gradient(0deg, rgba(8, 10, 15, 0.66), transparent 65%);
}
</style>
