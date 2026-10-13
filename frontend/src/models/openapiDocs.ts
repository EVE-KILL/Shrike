// One checked OpenAPI adapter for the Go API explorer and its shared helpers.
import type {ApiBody,ApiBodyField,ApiEndpoint,ApiParam,EntityKind,HttpMethod,ParamType} from './apiDocs'
import {isRecord} from '../browser/shared/json'

export interface DocsTag {name:string;description?:string;endpoints:ApiEndpoint[]}
export interface DocsGroup {name:string;tags:DocsTag[]}
export interface DocsModel {title:string;version:string;description?:string;groups:DocsGroup[];total:number}
const METHODS:HttpMethod[] = ['GET','POST','PUT','PATCH','DELETE']
const text = (value:unknown):string|undefined => typeof value === 'string' ? value : undefined
const strings = (value:unknown):string[] => Array.isArray(value) ? value.filter((item:unknown):item is string=>typeof item === 'string') : []
function record(value:unknown):Record<string,unknown> {return isRecord(value) ? value : {}}

// Local JSON pointers also cover parameter/request/response references. Ref
// chains may be cyclic; external references are not fetched by the explorer.
function resolveSchema(value:unknown,doc:Record<string,unknown>,seen = new Set<string>()):Record<string,unknown> {
  const node = record(value),ref = node.$ref
  if(typeof ref !== 'string') return node
  if(!ref.startsWith('#/') || seen.has(ref)) return {}
  seen.add(ref)
  let found:unknown = doc
  for(const key of ref.slice(2).split('/').map(part=>part.replace(/~1/g,'/').replace(/~0/g,'~'))) {
    if(isRecord(found) && Object.hasOwn(found,key)) found = found[key]
    else if(Array.isArray(found) && /^\d+$/.test(key)) found = found[Number(key)]
    else return {}
  }
  return resolveSchema(found,doc,seen)
}
function paramType(schema:Record<string,unknown>):Exclude<ParamType,'entity'> {
  if(Array.isArray(schema.enum) && schema.enum.length || schema.type === 'boolean') return 'enum'
  if(schema.format === 'date' || schema.format === 'date-time') return 'date'
  return schema.type === 'integer' || schema.type === 'number' ? 'int' : 'string'
}
function paramValues(schema:Record<string,unknown>):string[]|undefined {
  if(Array.isArray(schema.enum) && schema.enum.length) return schema.enum.map((value:unknown)=>String(value))
  return schema.type === 'boolean' ? ['true','false'] : undefined
}
const ENTITY_BY_SEGMENT:Readonly<Record<string,EntityKind>> = {
  characters:'character',character:'character',corporations:'corporation',corporation:'corporation',
  alliances:'alliance',alliance:'alliance',ships:'ship',ship:'ship',systems:'system',system:'system',
  regions:'region',region:'region',constellations:'constellation',constellation:'constellation',
  factions:'faction',faction:'faction',types:'item',type:'item',items:'item',item:'item',
  oldcharacters:'character',sovereignty:'system',prices:'item',
}
function entityKind(path:string,name:string):EntityKind|undefined {
  if(!/^(id|.*Id)$/.test(name)) return
  const parts = path.split('/').filter(Boolean),index = parts.indexOf('{'+name+'}')
  return index > 0 && Object.hasOwn(ENTITY_BY_SEGMENT,parts[index-1]) ? ENTITY_BY_SEGMENT[parts[index-1]] : undefined
}
export function isDocsEntityKind(value:unknown):value is EntityKind {
  return ['character','corporation','alliance','ship','item','system','region','constellation','faction'].some(kind=>kind === value)
}
function toParam(raw:Record<string,unknown>,doc:Record<string,unknown>,path:string):ApiParam|undefined {
  if(typeof raw.name !== 'string' || !raw.name) return
  const schema = resolveSchema(raw.schema,doc),entity = raw.in === 'path' ? entityKind(path,raw.name) : isDocsEntityKind(raw['x-entity']) ? raw['x-entity'] : undefined
  return {name:raw.name,type:entity ? 'entity' : paramType(schema),required:raw.required === true,
    description:text(raw.description) ?? text(schema.description),default:schema.default,values:paramValues(schema),
    example:raw.example ?? (Array.isArray(schema.examples) ? schema.examples[0] : undefined) ?? schema.example,entityType:entity}
}
function bodyFieldType(schema:Record<string,unknown>):ApiBodyField['type'] {
  if(schema.type === 'array') {const item = record(schema.items).type;return item === 'integer' || item === 'number' ? 'int[]' : 'string[]'}
  return paramType(schema)
}
function toBody(operation:Record<string,unknown>,doc:Record<string,unknown>):ApiBody|undefined {
  const request = resolveSchema(operation.requestBody,doc),media = record(record(request.content)['application/json'])
  const schema = resolveSchema(media.schema,doc)
  if(!isRecord(schema.properties)) return
  const required = strings(schema.required),fields:Array<[string,ApiBodyField]> = []
  for(const [name,raw] of Object.entries(schema.properties)) {
    const field = resolveSchema(raw,doc)
    fields.push([name,{type:bodyFieldType(field),description:text(field.description),required:required.includes(name),
      values:paramValues(field),default:field.default,maxItems:typeof field.maxItems === 'number' && Number.isSafeInteger(field.maxItems) && field.maxItems >= 0 ? field.maxItems : undefined}])
  }
  return {contentType:'application/json',fields:Object.fromEntries(fields),example:media.example ?? schema.example}
}
function describeReturns(operation:Record<string,unknown>,doc:Record<string,unknown>):string|undefined {
  const responses = record(operation.responses),key = Object.keys(responses).find(code=>code.startsWith('2'))
  if(!key) return
  const response = resolveSchema(responses[key],doc),schema = resolveSchema(record(record(response.content)['application/json']).schema,doc)
  if(schema.type === 'array') return key+' — array'
  const description = text(response.description)?.trim()
  return description && !/^(ok|created|no content)$/i.test(description) ? key+' — '+description : key
}
function toEndpoint(method:HttpMethod,path:string,operation:Record<string,unknown>,doc:Record<string,unknown>,sharedParameters:unknown):ApiEndpoint {
  const declared = new Map<string,Record<string,unknown>>()
  for(const values of [sharedParameters,operation.parameters]) if(Array.isArray(values)) for(const value of values) {
    const parameter = resolveSchema(value,doc)
    if(typeof parameter.in === 'string' && typeof parameter.name === 'string') declared.set(parameter.in+':'+parameter.name,parameter)
  }
  const parameters = [...declared.values()]
  const params = (kind:string)=>parameters.filter(param=>param.in === kind).flatMap(param=>{const converted = toParam(param,doc,path);return converted ? [converted] : []})
  const pathParams = params('path'),query = params('query')
  // Some retained compatibility routes omit parameter metadata. Their URL
  // placeholders still require inputs before the console can issue a request.
  for(const match of path.matchAll(/\{([^{}]+)\}/g)) if(!pathParams.some(parameter=>parameter.name === match[1])) {
    const name=match[1],entityType=entityKind(path,name)
    pathParams.push({name,type:entityType?'entity':'string',required:true,entityType})
  }
  return {method,path:path.replace(/\{([^}]+)\}/g,':$1'),summary:text(operation.summary) || method+' '+path,
    description:text(operation.description),pathParams:pathParams.length ? pathParams : undefined,query:query.length ? query : undefined,
    body:toBody(operation,doc),returns:describeReturns(operation,doc),operationId:text(operation.operationId),tags:strings(operation.tags),deprecated:operation.deprecated === true}
}
export function buildDocsModel(value:unknown):DocsModel {
  if(!isRecord(value) || !isRecord(value.paths)) throw new Error('Could not read the API document.')
  const doc = value,paths = value.paths,info = record(doc.info),tagDescriptions = new Map<string,string>(),byTag = new Map<string,ApiEndpoint[]>()
  if(Array.isArray(doc.tags)) for(const raw of doc.tags) if(isRecord(raw) && typeof raw.name === 'string') tagDescriptions.set(raw.name,text(raw.description) ?? '')
  let total = 0
  for(const [path,raw] of Object.entries(paths)) {
    const item = resolveSchema(raw,doc)
    for(const method of METHODS) {
      const operation = item[method.toLowerCase()]
      if(!isRecord(operation)) continue
      const endpoint = toEndpoint(method,path,operation,doc,item.parameters);total++
      for(const tag of endpoint.tags?.length ? endpoint.tags : ['other']) {let endpoints = byTag.get(tag);if(!endpoints) {endpoints = [];byTag.set(tag,endpoints)}endpoints.push(endpoint)}
    }
  }
  for(const endpoints of byTag.values()) endpoints.sort((a,b)=>a.path.localeCompare(b.path) || a.method.localeCompare(b.method))
  const groups:DocsGroup[] = [],placed = new Set<string>()
  if(Array.isArray(doc['x-tagGroups'])) for(const raw of doc['x-tagGroups']) {
    if(!isRecord(raw) || typeof raw.name !== 'string') continue
    const tags:DocsTag[] = []
    for(const name of strings(raw.tags)) {const endpoints = byTag.get(name);if(endpoints?.length) {placed.add(name);tags.push({name,description:tagDescriptions.get(name),endpoints})}}
    if(tags.length) groups.push({name:raw.name,tags})
  }
  const extra:DocsTag[] = []
  for(const name of [...byTag.keys()].filter(name=>!placed.has(name)).sort()) {const endpoints = byTag.get(name);if(endpoints) extra.push({name,description:tagDescriptions.get(name),endpoints})}
  if(extra.length) groups.push({name:'Other',tags:extra})
  return {title:text(info.title) ?? 'API',version:text(info.version) ?? '',description:text(info.description),groups,total}
}
