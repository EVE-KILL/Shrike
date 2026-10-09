/* @ts-self-types="./esf_dogma_engine.d.ts" */
import * as wasm from "./esf_dogma_engine_bg.wasm";
import { __wbg_set_wasm } from "./esf_dogma_engine_bg.js";

__wbg_set_wasm(wasm);
wasm.__wbindgen_start();
export {
    calculate, engine_version, init, load_sde
} from "./esf_dogma_engine_bg.js";
