/**
 * Type definitions for @evekill/dogma — mirrors of the Rust types in
 * EVEShipFit/dogma-engine's data_types.rs, matching exactly what serde
 * expects on the wasm-bindgen boundary.
 *
 * Two naming conventions coexist:
 *   - `Esf*` input types use snake_case (ship_type_id, type_id)
 *   - Dogma/Type callback types use camelCase (groupID, attributeID)
 * This matches the Rust side; do not "fix" it.
 */

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

// ========== Callback return types (JS → Rust during calculate()) ==========

/** Returned from `get_type(type_id)` — only the fields the engine actually reads. */
export interface DogmaType {
  groupID: number;
  categoryID: number;
  capacity?: number;
  mass?: number;
  radius?: number;
  volume?: number;
}

/** One entry in `get_dogma_attributes(type_id)`. */
export interface TypeDogmaAttribute {
  attributeID: number;
  value: number;
}

/** One entry in `get_dogma_effects(type_id)`. */
export interface TypeDogmaEffect {
  effectID: number;
  isDefault: boolean;
}

/** Returned from `get_dogma_attribute(attribute_id)`. */
export interface DogmaAttribute {
  defaultValue: number;
  highIsGood: boolean;
  stackable: boolean;
}

/**
 * ModifierInfo inside a DogmaEffect. `domain` and `func` are integer enum
 * values (serde_repr on the Rust side).
 */
export interface DogmaEffectModifierInfo {
  /** 0=ItemID 1=ShipID 2=CharID 3=OtherID 4=StructureID 5=Target 6=TargetID */
  domain: number;
  /** 0=ItemModifier 1=LocationGroupModifier 2=LocationModifier 3=LocationRequiredSkillModifier 4=OwnerRequiredSkillModifier 5=EffectStopper */
  func: number;
  modifiedAttributeID?: number | null;
  modifyingAttributeID?: number | null;
  operation?: number | null;
  groupID?: number | null;
  skillTypeID?: number | null;
}

/** Returned from `get_dogma_effect(effect_id)`. */
export interface DogmaEffect {
  effectCategory: number;
  electronicChance: boolean;
  isAssistance: boolean;
  isOffensive: boolean;
  isWarpSafe: boolean;
  propulsionChance: boolean;
  rangeChance: boolean;
  dischargeAttributeID?: number | null;
  durationAttributeID?: number | null;
  rangeAttributeID?: number | null;
  falloffAttributeID?: number | null;
  trackingSpeedAttributeID?: number | null;
  fittingUsageChanceAttributeID?: number | null;
  resistanceAttributeID?: number | null;
  modifierInfo: DogmaEffectModifierInfo[];
}

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
 * custom pass_4 computation — null if no effects modified the base value.
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
 * The top-level result of `calculateFit()` — mirrors the Rust `Ship` struct.
 * `hull` holds the ship's computed state including all the custom EVEShipFit
 * aggregate stats (ehp, alignTime, capacitorDepletesIn, etc.) under their
 * negative attribute IDs. `items` has one entry per fitted module in input
 * order. `skills` has one entry per skill in the SkillMap.
 */
export interface FitStats {
  hull: ComputedItem;
  items: ComputedItem[];
  skills: ComputedItem[];
  char: ComputedItem;
  structure: ComputedItem;
  target: ComputedItem;
  damage_profile: DamageProfile;
}
