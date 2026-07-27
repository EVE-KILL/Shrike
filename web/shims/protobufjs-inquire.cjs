/**
 * Stub replacement for @protobufjs/inquire — wired in via `vite.resolve.alias`
 * in nuxt.config.ts.
 *
 * The original is a tiny helper that does:
 *
 *     eval("quire".replace(/^/,"re"))(moduleName)
 *
 * to obfuscate `require()` from bundlers, so protobufjs's minimal build can
 * optionally pull in Node-only modules (`buffer`, `long`, `fs`) at runtime
 * without static imports tripping a webpack/Vite warning. Even though the
 * eval failure is caught in a try/catch, the eval CALL itself fires a CSP
 * `script-src` violation under `'strict-dynamic'` (no `'unsafe-eval'`),
 * which clutters the console with two warnings on every dogma test load.
 *
 * Returning null is exactly what protobufjs already expects in a browser
 * environment without Node Buffer or Long.js — the surrounding code falls
 * through to its native `Uint8Array` and `Number` paths, which is what we
 * already use (we pass `longs: Number` to every `toObject(...)` call and
 * load the binary SDE blobs as raw ArrayBuffers). No functional change,
 * just a clean console.
 *
 * Why .cjs and `module.exports = fn`: protobufjs/minimal is pre-bundled
 * as CJS by Vite's optimizeDeps, and its source does
 *     util.inquire = require("@protobufjs/inquire");
 *     util.inquire(name);
 * — it expects `require()` to return the function itself, not an object
 * with a `.default` property. An ESM `export default function` shim would
 * be wrapped into `{ default: fn }` under CJS→ESM interop and break
 * `util.inquire(name)` with "util.inquire is not a function".
 */
module.exports = function inquire() {
    return null;
};
