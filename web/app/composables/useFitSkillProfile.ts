import type { SkillMap } from "@evekill/dogma";

export type FitSkillLevel = 3 | 4 | 5;

const STORAGE_KEY = "ek-fit-skill-level-v1";
let persistenceInstalled = false;

export function useFitSkillProfile() {
    const level = useState<FitSkillLevel>("fit:skill-level", () => 5);
    const { sde } = useEveData();

    if (import.meta.client && !persistenceInstalled) {
        persistenceInstalled = true;
        const saved = Number.parseInt(localStorage.getItem(STORAGE_KEY) ?? "", 10);
        if (saved === 3 || saved === 4 || saved === 5) {
            level.value = saved;
        }
        watch(level, (next) => localStorage.setItem(STORAGE_KEY, String(next)));
    }

    const skills = computed<SkillMap | undefined>(() => {
        if (!sde.value) return undefined;
        const result: SkillMap = {};
        for (const [typeId, type] of sde.value.types) {
            if (type.categoryID === 16 && type.published) {
                result[String(typeId)] = level.value;
            }
        }
        return result;
    });

    const label = computed(() => `All Skills L${level.value}`);

    return { level, skills, label };
}
