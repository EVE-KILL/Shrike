<script setup lang="ts">
const props = defineProps<{
  /** Show loading skeleton instead of content */
  loading?: boolean;
  /** Entity identity accent (opaque CSS color) — adds a top strip + background wash */
  accent?: string | null;
  /** Optional large image rendered behind the entity identity content */
  backgroundImage?: string | null;
}>();

const accentStyle = computed(() =>
  props.accent
    ? {
        backgroundImage: `linear-gradient(to bottom, color-mix(in srgb, ${props.accent} 8%, transparent), transparent 60%)`,
        boxShadow: `inset 0 2px 0 0 ${props.accent}`,
      }
    : undefined,
);
</script>

<template>
  <div
    v-if="loading"
    class="h-64 rounded-lg bg-white/[0.04] animate-pulse mb-6"
  />

  <header
    v-else
    class="entity-header hero-surface glass-panel relative overflow-hidden mb-6"
    :style="accentStyle"
  >
    <img
      v-if="backgroundImage"
      :src="backgroundImage"
      alt=""
      class="entity-header__background pointer-events-none absolute inset-0 h-full w-full object-cover"
    />
    <div
      v-if="backgroundImage"
      class="entity-header__shade pointer-events-none absolute inset-0"
    />
    <div class="relative p-6">
      <div class="flex flex-col md:flex-row gap-6">
        <!-- Image slot -->
        <div
          v-if="$slots.image"
          class="flex-shrink-0 flex justify-center md:justify-start"
        >
          <slot name="image" />
        </div>

        <!-- Middle: default slot -->
        <div class="flex-1 min-w-0">
          <slot />
        </div>

        <!-- Right slot -->
        <div v-if="$slots.right" class="flex-shrink-0">
          <slot name="right" />
        </div>
      </div>
    </div>

    <!-- Stats section -->
    <div
      v-if="$slots.stats"
      class="relative px-6 pb-6 pt-3 border-t border-white/[0.06] bg-black/[0.08]"
    >
      <slot name="stats" />
    </div>
  </header>
</template>

<style scoped>

.entity-header__background {
  z-index: -3;
  filter: blur(14px) saturate(0.9);
  opacity: 0.5;
  transform: scale(1.08);
}

.entity-header__shade {
  z-index: -2;
  background:
    linear-gradient(
      90deg,
      rgba(8, 10, 15, 0.94) 0%,
      rgba(8, 10, 15, 0.7) 52%,
      rgba(8, 10, 15, 0.34) 100%
    ),
    linear-gradient(0deg, rgba(8, 10, 15, 0.72), transparent 70%);
}
</style>
