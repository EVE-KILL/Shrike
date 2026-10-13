/** Stable application fitting shapes; the Rust adapter translates upstream types. */

// ========== Input types (JS → Rust via calculate()) ==========

/** State of a module on a fit. serde deserializes these as the variant name. */
export type EsfStateName = "Passive" | "Online" | "Active" | "Overload";

/** Slot type for a module. serde deserializes these as the variant name. */
export type EsfSlotTypeName =
  | "High"
  | "Medium"
  | "Low"
  | "Rig"
  | "SubSystem"
  | "Service";

export interface EsfSlot {
  type: EsfSlotTypeName;
  index: number;
}

export interface EsfCharge {
  type_id: number;
}

export interface EsfModule {
  type_id: number;
  slot: EsfSlot;
  state: EsfStateName;
  charge?: EsfCharge | null;
}

export interface EsfDrone {
  type_id: number;
  state: EsfStateName;
}

export interface EsfFit {
  ship_type_id: number;
  modules: EsfModule[];
  drones: EsfDrone[];
}

/**
 * Skill levels, keyed by skill type ID as a string. The Rust side receives
 * this as BTreeMap<String, i32> and parses the keys back to i32 internally.
 */
export type SkillMap = Record<string, number>;

// ========== Output (Rust → JS from calculate()) ==========

/**
 * State of a computed item in the engine output. This is the full Rust
 * `EffectCategory` enum, which is a superset of `EsfStateName` — the engine
 * may flag items with states like "Target" or "Area" for special effects.
 */
export type ItemStateName =
  | "Passive"
  | "Online"
  | "Active"
  | "Overload"
  | "Target"
  | "Area"
  | "Dungeon"
  | "System";

/**
 * Slot type in the engine output. Superset of `EsfSlotTypeName` — includes
 * pseudo-slots for drones, charges, and "None" (used for the hull, char,
 * structure, target, and skills which don't occupy a real fitted slot).
 */
export type ItemSlotTypeName =
  | "High"
  | "Medium"
  | "Low"
  | "Rig"
  | "SubSystem"
  | "Service"
  | "DroneBay"
  | "Charge"
  | "None";

export interface ItemSlot {
  type: ItemSlotTypeName;
  /** Present for modules in a real slot, null for hull/char/skills/charge. */
  index?: number | null;
}

/**
 * One computed attribute on an item. `base_value` is the raw pre-modifier
 * value (from `get_dogma_attributes` or type-level fields like mass). `value`
 * is the final post-passes value after effects, stacking penalties, and
 * custom pass_4 computation — the adapter retains the final value even when it equals the base.
 */
export interface ComputedAttribute {
  base_value: number;
  value: number | null;
  /** Effects that modified this attribute. Opaque for consumers. */
  effects: unknown[];
}

/**
 * A computed item in the engine output — the hull, a fitted module, a drone,
 * a skill, or one of the special pseudo-items (char/structure/target).
 *
 * `attributes` is a JS `Map<number, ComputedAttribute>` because serde
 * serializes Rust's `BTreeMap<i32, _>` as a Map (integer keys can't live in
 * a plain JS object). `Object.keys(item.attributes)` returns `[]` — use
 * `.size`, `.get(id)`, or iteration via `for (const [id, a] of ...)`.
 *
 * Common negative attribute IDs (custom EVEShipFit stats, resolved via
 * attribute_name_to_id):
 *   -1  alignTime                      -25 ehp
 *   -2  capacitorPeakRecharge          -31 armorEhp
 *   -4  capacitorPeakLoad              -37 hullEhp
 *   -5  capacitorPeakDelta             -43 shieldEhp
 *   -7  capacitorDepletesIn (sentinel: -1 = stable)
 *   -11 damageAlpha
 *   -12 damagePerSecondWithoutReload
 *   -13 damagePerSecondWithReload
 */
export interface ComputedItem {
  type_id: number;
  slot: ItemSlot;
  charge?: ComputedItem | null;
  state: ItemStateName;
  max_state: ItemStateName;
  attributes: Map<number, ComputedAttribute>;
  effects: number[];
}

export interface DamageProfile {
  em: number;
  explosive: number;
  kinetic: number;
  thermal: number;
}

/**
 * The application result returned by the shared Rust compatibility adapter.
 * `hull` holds the ship's computed state including all the custom EVEShipFit
 * aggregate stats (ehp, alignTime, capacitorDepletesIn, etc.) under their
 * negative attribute IDs. `items` has one entry per fitted module in input
 * order, followed by individual drones. `char` contains calculated pilot attributes.
 */
export interface FitStats {
  hull: ComputedItem;
  items: ComputedItem[];
  char: ComputedItem;
  damage_profile: DamageProfile;
}
