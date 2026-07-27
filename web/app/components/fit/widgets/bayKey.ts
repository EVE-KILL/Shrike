/**
 * Shared literal-union type for which bay drawer is currently expanded
 * below the ResourceBar. Lives in its own file because Vue's
 * `<script setup>` doesn't support exporting types — two callers
 * (ResourceBar.vue and the /fits/create + /fit/:id pages) import it
 * from here instead.
 */
export type BayKey = "drones" | "cargo" | null;
