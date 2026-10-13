// Compile utility styles from the Go pages and their browser controllers.
import {compile,optimize} from '@tailwindcss/node'
import {Scanner} from '@tailwindcss/oxide'
import {readFile} from 'node:fs/promises'
import {resolve} from 'node:path'
import {writeChanged} from './io'
const base=resolve(import.meta.dirname,'..')
const theme=await readFile(resolve(base,'styles/theme.css'),'utf8')
const components=await readFile(resolve(base,'styles/components.css'),'utf8')
const input=`@import "tailwindcss/theme.css" layer(theme);
${theme}
@scope (.go-utility-page) {
  @import "tailwindcss/preflight.css";
  @import "tailwindcss/utilities.css" source(none);
  .icon { display: inline-block; }
  .grid > * { min-width: 0; }
  ${components}
}
@source "../internal/frontend/*.templ";
@source "../internal/frontend/*.go";
@source not "../internal/frontend/*_templ.go";
@source "../internal/frontend/static/*.js";
@source "./src/**/*.ts";
@source "./src/**/*.js";
`
const compiler=await compile(input,{base,onDependency(){}})
const scanner=new Scanner({sources:compiler.sources})
const result=optimize(compiler.build(scanner.scan()),{minify:true}).code+'\n'
await writeChanged(resolve(base,'../internal/frontend/static/utilities.css'),result)
console.log(`Go utility CSS: ${Buffer.byteLength(result).toLocaleString()} bytes`)
