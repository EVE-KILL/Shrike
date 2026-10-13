import {readdir,readFile} from 'node:fs/promises'
import {expect,test} from 'vitest'
import ts from 'typescript'
import config from '../tsconfig.json'

test('all authored browser sources are strict TypeScript without legacy or type-check escapes',async()=>{
  const base=new URL('../src/',import.meta.url),files=await readdir(base,{recursive:true})
  expect(files.filter(name=>name.endsWith('.js'))).toEqual([])
  expect(config.compilerOptions.strict).toBe(true)
  expect(config.compilerOptions.noImplicitAny).toBe(true)
  expect(config.compilerOptions).not.toHaveProperty('allowJs')
  expect(config.compilerOptions).not.toHaveProperty('checkJs')
  expect(config.include.some(pattern=>pattern.endsWith('.js'))).toBe(false)
  for(const name of files.filter(name=>name.endsWith('.ts'))){
    const source=await readFile(new URL(name,base),'utf8')
    expect(source,name).not.toMatch(/@ts-(?:nocheck|ignore|expect-error)/)
    const parsed=ts.createSourceFile(name,source,ts.ScriptTarget.Latest,true)
    let anyType=false;const visit=(node:ts.Node)=>{if(node.kind===ts.SyntaxKind.AnyKeyword)anyType=true;ts.forEachChild(node,visit)};visit(parsed)
    expect(anyType,name).toBe(false)
  }
})
