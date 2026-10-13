// Application-facing contract of Shrike's pinned Rust compatibility adapter.
// wasm-bindgen emits no declaration for its JS background module; the public
// generated entry uses any for serde values. These signatures match the stable
// adapter structs and are exercised by the real-engine regression suite.
import type {EsfFit,FitStats,SkillMap} from '../../../src/types'

export function calculate(fit:EsfFit,skills:SkillMap|null|undefined,canonical:boolean):FitStats
export function engine_version():string
export function init():void
export function load_sde(bytes:Uint8Array):number
export function __wbg_set_wasm(exports:WebAssembly.Exports):void
