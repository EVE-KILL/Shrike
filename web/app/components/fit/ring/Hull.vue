<script setup lang="ts">
/**
 * Ship render in the center of the ring. Pulls from CCP's image
 * server (images.evetech.net/types/{id}/render) which covers all
 * current ships including recent additions.
 */
import styles from "./ShipFit.module.css";

const { currentFit } = useCurrentFit();

const shipTypeId = computed(() => currentFit.value?.shipTypeId ?? null);
const shipName = computed(() => currentFit.value?.name ?? "");
</script>

<template>
    <div v-if="shipTypeId !== null" :class="styles.hull">
        <img :src="`https://images.evetech.net/types/${shipTypeId}/render?size=1024`" :alt="shipName" />
    </div>
</template>
