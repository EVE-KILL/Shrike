/**
 * Installs the seven callbacks that the dogma-engine's wasm-bindgen glue
 * expects on `window.*`. Since we aliased `window` to `globalThis` in the
 * loader, these go on globalThis directly.
 *
 * Each callback returns an object shaped to match what serde expects on the
 * Rust side (see data_types.rs). Field names are significant and the two
 * conventions are mixed:
 *   - DogmaType       → camelCase (groupID, categoryID, ...)
 *   - DogmaAttribute  → camelCase (defaultValue, highIsGood, stackable)
 *   - TypeDogmaEntry  → camelCase (attributeID, value / effectID, isDefault)
 *   - DogmaEffect     → camelCase (effectCategory, modifierInfo, ...)
 *   - ModifierInfo    → domain/func are INTEGERS (serde_repr)
 */

import type { SdeData } from "./core";

interface GlobalWithCallbacks {
  get_type: (typeId: number) => unknown;
  get_dogma_attributes: (typeId: number) => unknown;
  get_dogma_attribute: (attributeId: number) => unknown;
  get_dogma_effects: (typeId: number) => unknown;
  get_dogma_effect: (effectId: number) => unknown;
  attribute_name_to_id: (name: string) => number;
  type_name_to_id: (name: string) => number;
}

let installed = false;

export function installCallbacks(sde: SdeData): void {
  if (installed) return;

  const g = globalThis as unknown as GlobalWithCallbacks;

  g.get_type = (typeId: number) => {
    const t = sde.types.get(typeId);
    if (!t) {
      throw new Error(`get_type: unknown type_id ${typeId}`);
    }
    // Return only the fields the Rust `Type` struct reads. serde ignores
    // extras but we keep the payload small.
    return {
      groupID: t.groupID,
      categoryID: t.categoryID,
      capacity: t.capacity,
      mass: t.mass,
      radius: t.radius,
      volume: t.volume,
    };
  };

  g.get_dogma_attributes = (typeId: number) => {
    const td = sde.typeDogma.get(typeId);
    // An unknown type_id means the type simply has no dogma entries (e.g.
    // skillbook without fitted attributes) — return an empty list rather
    // than throwing, matching Rust's unwrap_or_default behavior.
    return td?.dogmaAttributes ?? [];
  };

  g.get_dogma_attribute = (attributeId: number) => {
    const a = sde.dogmaAttributes.get(attributeId);
    if (!a) {
      throw new Error(`get_dogma_attribute: unknown attribute_id ${attributeId}`);
    }
    return {
      defaultValue: a.defaultValue,
      highIsGood: a.highIsGood,
      stackable: a.stackable,
    };
  };

  g.get_dogma_effects = (typeId: number) => {
    const td = sde.typeDogma.get(typeId);
    return td?.dogmaEffects ?? [];
  };

  g.get_dogma_effect = (effectId: number) => {
    const e = sde.dogmaEffects.get(effectId);
    if (!e) {
      throw new Error(`get_dogma_effect: unknown effect_id ${effectId}`);
    }
    // Full DogmaEffect passthrough — all fields already camelCase from the
    // protobuf decode, and modifierInfo[].domain/func are already integers.
    return e;
  };

  g.attribute_name_to_id = (name: string) => {
    const id = sde.attributeNameToId.get(name);
    if (id === undefined) {
      throw new Error(`attribute_name_to_id: unknown name ${name}`);
    }
    return id;
  };

  g.type_name_to_id = (name: string) => {
    const id = sde.typeNameToId.get(name);
    if (id === undefined) {
      throw new Error(`type_name_to_id: unknown name ${name}`);
    }
    return id;
  };

  installed = true;
}
