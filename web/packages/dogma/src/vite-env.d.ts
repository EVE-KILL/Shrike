/**
 * Type declaration for Vite's `?url` asset imports.
 *
 * Vite transforms `import foo from './bar.wasm?url'` into the emitted URL of
 * the bundled asset at build time. Without this declaration, TypeScript
 * flags every such import in src/index.browser.ts as an unresolvable module.
 *
 * Kept minimal and vite-free so Bun's type checker doesn't choke on it when
 * running the node-side tests that never touch the browser entry.
 */
declare module "*?url" {
    const url: string;
    export default url;
}
