export interface ClassifiableFittingModule {
  name: string | null;
}

export interface ClassifiableFittingDrone {
  name: string | null;
  quantity?: number;
}

export interface FittingClassifierContext {
  hullGroupName?: string | null;
}

/**
 * Give a killmail-derived fitting family a stable, human-readable role name.
 * This intentionally uses only fitting evidence, so the same family receives
 * the same label everywhere without an external or probabilistic classifier.
 */
export function classifyFitFamily(
  modules: ClassifiableFittingModule[],
  drones: ClassifiableFittingDrone[] = [],
  context: FittingClassifierContext = {},
): string {
  const names = modules.map((module) => (module.name ?? "").toLowerCase());
  const has = (...terms: string[]) =>
    names.some((name) => terms.some((term) => name.includes(term)));
  const combatNames = names.filter((name) => !name.includes("civilian"));
  const hasCombat = (...terms: string[]) =>
    combatNames.some((name) => terms.some((term) => name.includes(term)));
  const droneNames = drones.map((drone) => (drone.name ?? "").toLowerCase());
  const hasDrone = (...terms: string[]) =>
    droneNames.some((name) => terms.some((term) => name.includes(term)));
  const hasWeb = has("stasis webifier", "stasis grappler");
  const hasPainter = has("target painter");
  const hullGroup = (context.hullGroupName ?? "").toLowerCase();
  const isHaulingHull = ["industrial", "transport ship", "freighter"].some(
    (group) => hullGroup.includes(group),
  );
  const hasLocalShieldBoost = names.some(
    (name) =>
      name.includes("shield booster") &&
      !name.includes("remote shield booster"),
  );
  const hasLocalArmorRepair = names.some(
    (name) =>
      name.includes("armor repairer") &&
      !name.includes("remote armor repairer"),
  );
  const weapon = hasCombat("heavy assault missile launcher")
    ? "Heavy Assault Missile"
    : hasCombat("rapid light missile launcher")
      ? "Rapid Light Missile"
      : hasCombat("rapid heavy missile launcher")
        ? "Rapid Heavy Missile"
        : hasCombat("heavy missile launcher")
          ? "Heavy Missile"
          : hasCombat("rocket launcher")
            ? "Rocket"
            : hasCombat("light missile launcher")
              ? "Light Missile"
              : hasCombat("torpedo launcher")
                ? "Torpedo"
                : hasCombat("cruise missile launcher")
                  ? "Cruise Missile"
                  : hasCombat("pulse laser")
                    ? "Pulse Laser"
                    : hasCombat("beam laser", "tachyon beam")
                      ? "Beam Laser"
                      : hasCombat("autocannon")
                        ? "Autocannon"
                        : hasCombat("artillery")
                          ? "Artillery"
                          : hasCombat("blaster")
                            ? "Blaster"
                            : hasCombat("railgun")
                              ? "Railgun"
                              : hasCombat("entropic disintegrator")
                                ? "Disintegrator"
                                : hasCombat("vorton projector")
                                  ? "Vorton Projector"
                                  : hasCombat("smartbomb")
                                    ? "Smartbomb"
                                    : null;
  const hasDroneDamage = has(
    "drone damage amplifier",
    "drone link augmentor",
    "omnidirectional tracking link",
  );
  const hasCombatDrones = droneNames.some(
    (name) =>
      name &&
      ![
        "maintenance bot",
        "salvage drone",
        "mining drone",
        "ec-",
        "ev-",
        "sw-",
        "sd-",
        "tp-",
      ].some((term) => name.includes(term)),
  );
  const droneWeapon = has("fighter support unit")
    ? "Fighter"
    : hasDrone("bouncer", "curator", "garde", "warden") && hasDroneDamage
      ? "Sentry Drone"
      : hasDroneDamage || hasCombatDrones
        ? "Drone"
        : null;
  const offense =
    droneWeapon && weapon ? `${droneWeapon}/${weapon}` : droneWeapon || weapon;
  const hasNeut = has("energy neutralizer");
  const mode = has("industrial cynosural field generator")
    ? "Industrial Cyno"
    : has("cynosural field generator")
      ? "Cyno"
      : has("higgs anchor")
        ? "Rolling"
        : has("interdiction sphere launcher")
          ? "Interdiction"
          : has("bomb launcher")
            ? "Bomber"
            : has("siege module")
              ? "Siege"
              : has("bastion module")
                ? "Bastion"
                : has("triage module")
                  ? "Triage"
                  : has("micro jump field generator")
                    ? "Jump Field"
                    : has("industrial core")
                      ? "Industrial Command"
                      : "";
  const ewar = has("ecm", "burst jammer")
    ? "ECM"
    : has("remote sensor dampener")
      ? "Sensor Damping"
      : has("weapon disruptor", "tracking disruptor", "guidance disruptor")
        ? "Weapon Disruption"
        : hasWeb && hasPainter
          ? "Web/Painter"
          : hasPainter
            ? "Target Painting"
            : hasWeb && !has("warp scrambler")
              ? "Web Support"
              : "";
  const modeDefinesRole = has(
    "cynosural field generator",
    "industrial cynosural field generator",
  );
  const role = modeDefinesRole
    ? null
    : has(
          "bastion module",
          "siege module",
          "interdiction sphere launcher",
          "bomb launcher",
        ) && offense
      ? offense
      : has("remote shield booster")
        ? "Remote Shield Boost"
        : has("remote armor repairer")
          ? "Remote Armor Repair"
          : has("gas cloud scoop", "gas cloud harvester")
            ? "Gas Harvesting"
            : has("mining laser", "strip miner")
              ? "Mining"
              : has("command burst")
                ? "Command"
                : has("data analyzer", "relic analyzer", "integrated analyzer")
                  ? "Exploration"
                  : has("salvager")
                    ? "Salvaging"
                    : has("probe launcher") &&
                        has(
                          "scan acquisition",
                          "scan pinpointing",
                          "scan rangefinding",
                        )
                      ? "Scanning"
                      : offense ||
                        (hasNeut ? "Neut" : ewar || null) ||
                        (has("expanded cargohold", "cargohold optimization") ||
                        isHaulingHull
                          ? "Hauling"
                          : null) ||
                        (has("interdiction nullifier") ? "Travel" : null);
  const utility = weapon && hasNeut ? "Neut" : "";
  const modifier = has("covert ops cloaking") ? "Covert" : "";
  const tank = has(
    "shield booster",
    "shield extender",
    "shield hardener",
    "shield recharger",
  )
    ? "Shield"
    : has(
          "armor repairer",
          "armor plate",
          "steel plates",
          "restrained plates",
          "energized adaptive",
        )
      ? "Armor"
      : has("reinforced bulkhead", "transverse bulkhead")
        ? "Hull"
        : "";
  const style =
    has("warp scrambler", "stasis webifier") &&
    has("heavy assault missile", "autocannon", "blaster", "pulse laser")
      ? "Brawler"
      : has("warp disruptor") &&
          has(
            "artillery",
            "beam laser",
            "railgun",
            "heavy missile",
            "cruise missile",
          )
        ? "Kite"
        : hasLocalShieldBoost || hasLocalArmorRepair
          ? "Active"
          : has(
                "shield power relay",
                "shield recharger",
                "core defense field purger",
              )
            ? "Passive"
            : has(
                  "shield extender",
                  "armor plate",
                  "steel plates",
                  "restrained plates",
                  "reinforced bulkhead",
                  "transverse bulkhead",
                )
              ? "Buffer"
              : "";
  const ewarQualifier = ewar && role !== ewar ? ewar : "";
  const droneQualifier =
    role?.startsWith("Remote ") && droneWeapon ? droneWeapon : "";
  const identity = mode || role ? [mode, role] : ["General Purpose"];
  return [
    modifier,
    ...identity,
    utility,
    droneQualifier,
    ewarQualifier,
    tank,
    style,
  ]
    .filter(Boolean)
    .join(" ");
}
