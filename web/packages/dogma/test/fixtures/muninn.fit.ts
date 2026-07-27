/**
 * Real Muninn fit captured from the live evekill dev server, top result at
 * /api/item/12015/fittings on 2026-04-08.
 *
 * Fit details:
 *   fit_hash:   d8c764f2fd24eee2839aa1ac3a1cfd3b4e24a89e75a5dc1f1c5586686b2dca1e
 *   family:     dce038bd5149e1b8653840f38ae4e73dd3e0b4de5b784917eaeaadcf795a167c
 *   uses:       86 exact / 98 family (top family, in-window)
 *   flown by:   The Initiative. (68.5% of their Muninn losses)
 *
 * Shield HML Muninn — 5x Heavy Missile Launcher II, multispec hardener +
 * 2x LSE II + MWD, Co-Proc + 2x BCS II + 2x MGE II + Assault DCU, 2 shield
 * rigs. No charges or drones in the fitting_items shape by design.
 */

import type { FittingItemRow } from "../../src/converter";

export const MUNINN_TYPE_ID = 12015;

export const MUNINN_TOP_FIT_HASH =
  "d8c764f2fd24eee2839aa1ac3a1cfd3b4e24a89e75a5dc1f1c5586686b2dca1e";

export const MUNINN_TOP_FIT_ITEMS: ReadonlyArray<FittingItemRow> = [
  // High slots — 5x Heavy Missile Launcher II
  { slot_group: 1, ordinal: 0, type_id: 2410 },
  { slot_group: 1, ordinal: 1, type_id: 2410 },
  { slot_group: 1, ordinal: 2, type_id: 2410 },
  { slot_group: 1, ordinal: 3, type_id: 2410 },
  { slot_group: 1, ordinal: 4, type_id: 2410 },

  // Med slots — Multispec Hardener II, 2x LSE II, 50MN Quad LiF MWD
  { slot_group: 2, ordinal: 0, type_id: 2281 }, // Multispectrum Shield Hardener II
  { slot_group: 2, ordinal: 1, type_id: 3841 }, // Large Shield Extender II
  { slot_group: 2, ordinal: 2, type_id: 3841 }, // Large Shield Extender II
  { slot_group: 2, ordinal: 3, type_id: 35660 }, // 50MN Quad LiF Restrained Microwarpdrive

  // Low slots — Co-Proc II, 2x BCS II, 2x MGE II, Assault DCU II
  { slot_group: 3, ordinal: 0, type_id: 3888 }, // Co-Processor II
  { slot_group: 3, ordinal: 1, type_id: 22291 }, // Ballistic Control System II
  { slot_group: 3, ordinal: 2, type_id: 22291 }, // Ballistic Control System II
  { slot_group: 3, ordinal: 3, type_id: 35771 }, // Missile Guidance Enhancer II
  { slot_group: 3, ordinal: 4, type_id: 35771 }, // Missile Guidance Enhancer II
  { slot_group: 3, ordinal: 5, type_id: 47257 }, // Assault Damage Control II

  // Rigs — Med Kinetic Shield Reinforcer I + Med Core Defense Field Extender I
  { slot_group: 4, ordinal: 0, type_id: 31742 },
  { slot_group: 4, ordinal: 1, type_id: 31790 },
];
