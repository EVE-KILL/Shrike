/**
 * Inject key for propagating the current tree node's render height
 * down to nested TreeHeader / TreeHeaderAction / TreeLeaf components.
 *
 * The React reference uses React.Context for this — we mirror with
 * a Vue provide/inject Symbol so every descendant of a TreeListing
 * can read its parent's configured icon size without prop-drilling.
 *
 * Consumers inject a *function* (`() => number`), not a raw value,
 * because `TreeListing`'s `height` prop is reactive and we want the
 * latest value on each read rather than capturing the size at the
 * moment provide() ran.
 */

import type { InjectionKey } from "vue";

export const treeSizeInjectKey: InjectionKey<() => number> = Symbol("treeSize");
