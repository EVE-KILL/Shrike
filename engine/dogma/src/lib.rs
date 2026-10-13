//! Compatibility adapter shared by Shrike's native worker and browser WASM.
//! Calculation rules are supplied by the unmodified upstream engine.
use esf_data::{InfoSde, Sde};
use esf_dogma_engine::{self as engine, FitItem, Slot, Source, State};
use serde::{Deserialize, Serialize};
use std::{collections::BTreeMap, sync::OnceLock};

pub const ENGINE_VERSION: &str = "13.6.0";
pub const COMPAT: &str = include_str!("../compat.json");
#[derive(Deserialize)]
struct Compatibility {
    aliases: BTreeMap<String, String>,
    #[serde(rename = "customAttributes")]
    custom: Vec<CustomAttribute>,
}
#[derive(Deserialize)]
struct CustomAttribute {
    id: i32,
    name: String,
}

#[derive(Deserialize, Clone)]
pub struct LegacySlot {
    #[serde(rename = "type")]
    pub kind: String,
    pub index: u16,
}
#[derive(Deserialize, Clone)]
pub struct LegacyCharge {
    pub type_id: i32,
}
#[derive(Deserialize, Clone)]
pub struct LegacyModule {
    pub type_id: i32,
    pub slot: LegacySlot,
    pub state: String,
    pub charge: Option<LegacyCharge>,
}
#[derive(Deserialize, Clone)]
pub struct LegacyDrone {
    pub type_id: i32,
    pub state: String,
}
#[derive(Deserialize, Clone)]
pub struct LegacyFit {
    pub ship_type_id: i32,
    #[serde(default)]
    pub modules: Vec<LegacyModule>,
    #[serde(default)]
    pub drones: Vec<LegacyDrone>,
}
#[derive(Serialize)]
pub struct OutputSlot {
    #[serde(rename = "type")]
    pub kind: String,
    pub index: Option<u16>,
}
#[derive(Serialize)]
pub struct Attribute {
    pub base_value: f64,
    pub value: Option<f64>,
    pub effects: Vec<Source>,
}
#[derive(Serialize)]
pub struct Item {
    pub type_id: i32,
    pub slot: OutputSlot,
    pub state: String,
    pub max_state: String,
    pub attributes: BTreeMap<i32, Attribute>,
    pub charge: Option<Box<Item>>,
    pub effects: Vec<i32>,
}
#[derive(Serialize)]
pub struct Stats {
    pub hull: Item,
    pub items: Vec<Item>,
    pub char: Item,
    pub damage_profile: BTreeMap<&'static str, f64>,
}

pub fn attribute_id(sde: &Sde<'_>, name: &str) -> Option<i32> {
    static SPEC: OnceLock<Compatibility> = OnceLock::new();
    let spec =
        SPEC.get_or_init(|| serde_json::from_str(COMPAT).expect("valid compatibility metadata"));
    if let Some(c) = spec.custom.iter().find(|c| c.name == name) {
        return Some(c.id);
    }
    sde.attribute_name_to_id(spec.aliases.get(name).map(String::as_str).unwrap_or(name))
}

fn state(value: &str, rig: bool) -> Result<State, String> {
    Ok(match value {
        "Passive" if rig => State::Online,
        "Passive" => State::Offline,
        "Online" => State::Online,
        "Active" => State::Active,
        "Overload" => State::Overload,
        _ => return Err(format!("unknown module state {value}")),
    })
}
fn legacy_state(value: State, rig: bool) -> String {
    if rig {
        return "Passive".into();
    }
    match value {
        State::Offline => "Passive",
        State::Online => "Online",
        State::Active => "Active",
        State::Overload => "Overload",
    }
    .into()
}
fn slot(value: &LegacySlot, canonical: bool) -> Result<Slot, String> {
    let index = if canonical {
        value.index
    } else {
        value.index.saturating_sub(1)
    };
    let index = u8::try_from(index).map_err(|_| "slot index out of range")?;
    Ok(match value.kind.as_str() {
        "High" => Slot::High(index),
        "Medium" => Slot::Medium(index),
        "Low" => Slot::Low(index),
        "Rig" => Slot::Rig(index),
        "SubSystem" => Slot::Subsystem(index),
        "Service" => Slot::Service(index),
        _ => return Err(format!("unknown slot type {}", value.kind)),
    })
}
fn item(type_id: i32, slot: Slot, state: State, charge: Option<i32>) -> FitItem {
    FitItem {
        type_id,
        slot,
        state,
        quantity: 1,
        charge: charge.map(|type_id| engine::Charge { type_id }),
        mutation: None,
        fighter_abilities: None,
        booster_side_effects: Default::default(),
        spool: None,
    }
}
fn canonicalize(sde: &Sde<'_>, fit: &mut LegacyFit) {
    let mut propulsion_active = false;
    for module in &mut fit.modules {
        if module.slot.kind == "Rig" {
            module.state = "Passive".into();
            continue;
        }
        if module.slot.kind == "SubSystem" {
            module.state = "Online".into();
            continue;
        }
        let effects: Vec<_> = sde
            .get_type(module.type_id)
            .and_then(|t| t.dogma_effects())
            .into_iter()
            .flatten()
            .filter_map(|e| sde.get_dogma_effect(e.effect_id()))
            .collect();
        let active = effects.iter().any(|e| e.effect_category().0 == 1);
        let forbidden = effects.iter().any(|e| {
            let n = e.name().to_lowercase();
            n.contains("cloak") || n.contains("cyno")
        });
        let propulsion = effects.iter().any(|e| {
            let n = e.name().to_lowercase();
            n.contains("afterburner") || n.contains("microwarpdrive")
        });
        module.state = if active && !forbidden {
            "Active"
        } else {
            "Online"
        }
        .into();
        if propulsion && module.state == "Active" {
            if propulsion_active {
                module.state = "Online".into()
            } else {
                propulsion_active = true
            }
        }
    }
}
fn convert(
    result: engine::ItemResult,
    type_id: i32,
    kind: &str,
    index: Option<u16>,
    charge_id: Option<i32>,
) -> Item {
    let rig = kind == "Rig";
    Item {
        type_id,
        slot: OutputSlot {
            kind: kind.into(),
            index,
        },
        state: legacy_state(result.state, rig),
        max_state: legacy_state(result.max_state, rig),
        attributes: result
            .attributes
            .into_iter()
            .map(|(id, a)| {
                (
                    id,
                    Attribute {
                        base_value: a.base,
                        value: Some(a.value),
                        effects: a.sources,
                    },
                )
            })
            .collect(),
        charge: result
            .charge
            .map(|c| Box::new(convert(*c, charge_id.unwrap_or(0), "Charge", None, None))),
        effects: Vec::new(),
    }
}
fn has_effect(sde: &Sde<'_>, type_id: i32, name: &str) -> bool {
    sde.get_type(type_id)
        .and_then(|t| t.dogma_effects())
        .into_iter()
        .flatten()
        .any(|e| {
            sde.get_dogma_effect(e.effect_id())
                .is_some_and(|effect| effect.name() == name)
        })
}
fn outgoing_totals(sde: &Sde<'_>, stats: &mut Stats) {
    // Upstream now calculates the per-module rates, including fuel, paste,
    // skill/ship bonuses and spool. Shrike only sums the displayed outgoing rates.
    let specs = [
        (
            "remoteShieldBoostRate",
            "shieldBoostRate",
            Some("remoteShieldBoostRate"),
        ),
        (
            "remoteArmorRepairRate",
            "armorRepairRate",
            Some("remoteArmorRepairRate"),
        ),
        (
            "remoteHullRepairRate",
            "hullRepairRate",
            Some("remoteHullRepairRate"),
        ),
        (
            "energyNeutralizerRate",
            "capacitorTransferRate",
            Some("capacitorTransferRateNeutralizer"),
        ),
        (
            "energyNosferatuRate",
            "capacitorTransferRate",
            Some("energyNosferatuFalloff"),
        ),
        (
            "remoteCapTransferRate",
            "capacitorTransferRate",
            Some("shipModuleRemoteCapacitorTransmitter"),
        ),
    ];
    for (output, input, effect) in specs {
        let (Some(output_id), Some(input_id)) =
            (attribute_id(sde, output), sde.attribute_name_to_id(input))
        else {
            continue;
        };
        let mut total = None;
        for item in &stats.items {
            if item.state != "Active" && item.state != "Overload" {
                continue;
            }
            if effect.is_some_and(|effect| !has_effect(sde, item.type_id, effect)) {
                continue;
            }
            if let Some(value) = item.attributes.get(&input_id).and_then(|a| a.value) {
                total = Some(total.unwrap_or(0.0) + value);
            }
        }
        stats.hull.attributes.remove(&output_id);
        if let Some(value) = total {
            stats.hull.attributes.insert(
                output_id,
                Attribute {
                    base_value: 0.0,
                    value: Some(value),
                    effects: Vec::new(),
                },
            );
        }
    }
}

pub fn calculate<'a>(
    sde: &'a Sde<'a>,
    mut fit: LegacyFit,
    skills: Option<BTreeMap<i32, u8>>,
    canonical: bool,
) -> Result<Stats, String> {
    if sde.get_type(fit.ship_type_id).is_none() {
        return Err(format!("unknown hull {}", fit.ship_type_id));
    }
    if canonical {
        canonicalize(sde, &mut fit)
    }
    let skills = skills.unwrap_or_else(|| {
        sde.types()
            .filter(|t| t.category_id() == 16 && t.published())
            .map(|t| (t.id(), 5))
            .collect()
    });
    let mut items = Vec::with_capacity(fit.modules.len() + fit.drones.len());
    for m in &fit.modules {
        items.push(item(
            m.type_id,
            slot(&m.slot, canonical)?,
            state(&m.state, m.slot.kind == "Rig")?,
            m.charge.as_ref().map(|c| c.type_id),
        ));
    }
    for d in &fit.drones {
        items.push(item(
            d.type_id,
            Slot::DroneBay,
            state(&d.state, false)?,
            None,
        ));
    }
    let native = engine::Fit {
        name: None,
        ship: engine::Ship {
            type_id: fit.ship_type_id,
            mode: None,
        },
        items,
        character: engine::Character {
            skills,
            security_status: 0.0,
        },
        environment: Default::default(),
        incoming: Default::default(),
    };
    let calculation = engine::calculate(&InfoSde::new(sde), &native, &engine::Options::default());
    let mut results = Vec::with_capacity(calculation.items.len());
    for (index, result) in calculation.items.into_iter().enumerate() {
        if let Some(m) = fit.modules.get(index) {
            results.push(convert(
                result,
                m.type_id,
                &m.slot.kind,
                Some(m.slot.index),
                m.charge.as_ref().map(|c| c.type_id),
            ));
        } else {
            let d = &fit.drones[index - fit.modules.len()];
            results.push(convert(result, d.type_id, "DroneBay", None, None));
        }
    }
    let mut stats = Stats {
        hull: convert(calculation.ship, fit.ship_type_id, "None", None, None),
        items: results,
        char: convert(calculation.character, 1373, "None", None, None),
        damage_profile: [
            ("em", 0.25),
            ("explosive", 0.25),
            ("kinetic", 0.25),
            ("thermal", 0.25),
        ]
        .into(),
    };
    outgoing_totals(sde, &mut stats);
    Ok(stats)
}

#[cfg(all(feature = "browser", target_arch = "wasm32"))]
mod browser {
    use super::*;
    use std::sync::OnceLock;
    use wasm_bindgen::prelude::*;
    static BYTES: OnceLock<Vec<u8>> = OnceLock::new();
    static DATA: OnceLock<Sde<'static>> = OnceLock::new();
    #[wasm_bindgen]
    pub fn init() {
        console_error_panic_hook::set_once()
    }
    #[wasm_bindgen]
    pub fn load_sde(bytes: Vec<u8>) -> Result<i32, JsError> {
        if DATA.get().is_some() {
            return Err(JsError::new("SDE already loaded"));
        }
        let bytes = BYTES.get_or_init(|| bytes);
        let data = Sde::new(bytes)?;
        let build = data.build_number();
        let _ = DATA.set(data);
        Ok(build)
    }
    #[wasm_bindgen]
    pub fn engine_version() -> String {
        ENGINE_VERSION.into()
    }
    #[wasm_bindgen(js_name=calculate)]
    pub fn calculate_fit(
        fit: JsValue,
        skills: Option<JsValue>,
        canonical: bool,
    ) -> Result<JsValue, JsError> {
        let fit: LegacyFit = serde_wasm_bindgen::from_value(fit)?;
        let skills = match skills {
            Some(value) => {
                let raw: BTreeMap<String, u8> = serde_wasm_bindgen::from_value(value)?;
                Some(
                    raw.into_iter()
                        .map(|(id, level)| id.parse::<i32>().map(|id| (id, level)))
                        .collect::<Result<_, _>>()?,
                )
            }
            None => None,
        };
        let sde = DATA.get().ok_or_else(|| JsError::new("load SDE first"))?;
        let stats =
            super::calculate(sde, fit, skills, canonical).map_err(|error| JsError::new(&error))?;
        Ok(serde_wasm_bindgen::to_value(&stats)?)
    }
}
