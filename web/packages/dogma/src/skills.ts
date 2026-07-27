/**
 * Skill map helpers.
 *
 * The dogma engine takes skills as BTreeMap<String, i32> on the Rust side —
 * type IDs as string keys, levels as integers. The most common preset for
 * ship ranking and theorycraft is "all V" (every published skill at level 5),
 * matching what eveship.fit and osmium default to.
 */

import type { SdeData } from "./core";
import type { SkillMap } from "./types";

/** EVE SDE category ID for skills. */
const SKILL_CATEGORY_ID = 16;

/**
 * Build a SkillMap with every published skill at level 5. Used as the default
 * skill set when calculateFit is called without an explicit map.
 */
export function allFive(sde: SdeData): SkillMap {
  const skills: SkillMap = {};
  for (const [typeId, t] of sde.types) {
    if (t.categoryID === SKILL_CATEGORY_ID && t.published) {
      skills[String(typeId)] = 5;
    }
  }
  return skills;
}
