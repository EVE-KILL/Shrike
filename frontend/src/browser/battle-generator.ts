import { initializeDateTimePickers } from './date-time-picker.js';
import { initializeSearchPickers, updateSearchPicker, closeSearchPickers } from './search-picker.js';
import {BrowserRequestError} from './browser-request';
import {RequestComponent} from './controllers/request-component';
import {registerController} from './controllers/application';
import {iconHref} from './icon-hrefs';
import {requiredElement,cloneComponent,eventElement,placeElement} from './shared/dom';
import {errorMessage,isAbortError,isRecord} from './shared/json';
import {readGeneratorInitial,readGeneratorEntities,readGeneratorSeedTeams,readGeneratorPreview,isGeneratorSystem,type GeneratorPreview,type GeneratorSystem} from './shared/battle-generator-model';
import type {GeneratorEntity} from '../utils/battleGenerator';
import type {AuthMeLegacyResponses,AuthLoginLegacyResponses,BattleGeneratorEntitiesData,BattleGeneratorPreviewData,BattleGeneratorSaveResponses} from '../../shared/api/types.gen';
type Lane='a'|'b'|'u';
type Part='entities'|'preview'|'seed'|'save'|'identity';
type DraftBody=BattleGeneratorEntitiesData['body']&BattleGeneratorPreviewData['body'];
type GeneratorState={step:number;systems:GeneratorSystem[];entities:GeneratorEntity[];corps:GeneratorEntity[];sideA:GeneratorEntity[];sideB:GeneratorEntity[];filter:string;preview:GeneratorPreview|null;dropped:number;authenticated:boolean;loadingEntities:boolean;loadingPreview:boolean;saving:boolean;auto:boolean;alive:boolean;requests:Map<Part,AbortController>;authGeneration:number;revision:number;uncertainSave:boolean};
import { updateWizardSteps } from './wizard-steps.js';
import { generatorKey, generatorWindowHours, generatorTimeError, generatorCanLoad, generatorEntityList, generatorKeepAssignments, generatorUndecided, generatorMove, generatorSeedTeams, eveInputToIso } from './battle-generator-data.js';

function createGenerator(root:HTMLElement,io:BattleGeneratorController) {
  const q=<T extends Element=HTMLElement>(selector:string)=>requiredElement<T>(root,selector);
  const get=<T extends Element=HTMLElement>(scope:ParentNode,selector:string)=>requiredElement<T>(scope,selector);
  const initial=readGeneratorInitial(JSON.parse(q('#battle-generator-state').textContent??'null')),prefill=initial.prefill;
  const start=q<HTMLInputElement>('[name="generator_start"]'),end=q<HTMLInputElement>('[name="generator_end"]'),picker=q<HTMLElement>('[data-search-picker]');
  const editor=q<HTMLElement>('[data-generator-editor]'),gate=q<HTMLElement>('[data-generator-login]');
  const state:GeneratorState={step:1,systems:prefill.systems||[],entities:[],corps:[],sideA:[],sideB:[],filter:'',preview:null,dropped:0,authenticated:initial.authenticated,loadingEntities:false,loadingPreview:false,saving:false,auto:prefill.autoAdvance,alive:true,requests:new Map(),authGeneration:0,revision:0,uncertainSave:false};
  const key=generatorKey,sideName=(side:'a'|'b')=>q<HTMLInputElement>(`[data-generator-side-name="${side}"]`).value;
  const clone=(selector:string)=>cloneComponent(root,selector);
  function icon(name:string){const svg=document.createElementNS('http://www.w3.org/2000/svg','svg'),use=document.createElementNS('http://www.w3.org/2000/svg','use');svg.setAttribute('class','icon');svg.setAttribute('viewBox','0 0 24 24');svg.setAttribute('aria-hidden','true');use.setAttribute('href',iconHref(name));svg.append(use);return svg;}
  const windowBody=()=>({systemIds:state.systems.map(s=>s.id),startTime:eveInputToIso(start.value),endTime:eveInputToIso(end.value)});
  const draftBody=()=>({...windowBody(),sides:[{name:sideName('a'),entities:state.sideA.map(e=>({id:e.id,type:e.type}))},{name:sideName('b'),entities:state.sideB.map(e=>({id:e.id,type:e.type}))}]});
  function showError(selector:string,message:string|null){const box=q(selector);box.hidden=!message;const label=box.querySelector('[data-generator-error-text]')||box;label.textContent=message||'';}
  function cancel(part:Part){state.requests.get(part)?.abort();state.requests.delete(part);if(part==='entities')state.loadingEntities=false;if(part==='preview')state.loadingPreview=false;}
  function invalidate(){++state.revision;cancel('preview');state.preview=null;showError('[data-generator-preview-error]','');showError('[data-generator-save-error]','');}
  function button(selector:string,enabled:boolean,pending:boolean,label:string,normal:string) {
    const node=q<HTMLButtonElement>(selector);node.disabled=!enabled||pending;
    for(const cls of ['bg-blue-500','text-white','hover:bg-blue-600','cursor-pointer'])node.classList.toggle(cls,enabled&&!pending);
    for(const cls of ['bg-white/[0.04]','text-gray-600','cursor-not-allowed'])node.classList.toggle(cls,!enabled||pending);
    get(node,'[data-generator-'+normal+'-spinner]').hidden=!pending;
    get(node,'[data-generator-'+normal+'-label]').textContent=label;
  }
  function render() {
    editor.hidden=!state.authenticated;gate.hidden=state.authenticated;
    for(const section of root.querySelectorAll<HTMLElement>('[data-generator-stage]'))section.hidden=Number(section.dataset.generatorStage)!==state.step;
    updateWizardSteps(root,state.step);
    const ready=generatorCanLoad(state.systems,start.value,end.value),error=generatorTimeError(start.value,end.value),hours=generatorWindowHours(start.value,end.value),hasWindow=!!start.value&&!!end.value;
    showError('[data-generator-time-error]',error);
    const span=q<HTMLElement>('[data-generator-time-span]');span.hidden=!!error||!(hours>0);span.textContent=`${hours.toFixed(1)}h window`;
    q<HTMLElement>('[data-generator-system-count]').textContent=`${state.systems.length}/5`;
    q<HTMLElement>('[data-generator-system-empty]').hidden=state.systems.length>0;q<HTMLElement>('[data-generator-systems]').hidden=!state.systems.length;
    updateSearchPicker(picker,{disabled:state.systems.length>=5,placeholder:state.systems.length>=5?'Maximum 5 systems reached':'Search for a solar system...',isPicked:(_,id)=>state.systems.some(s=>s.id===id)});
    const checks=[{ok:state.systems.length>0,detail:state.systems.length?`${state.systems.length} selected`:'Search for the system the fight happened in'},{ok:hasWindow,detail:hasWindow?null:'Both ends of the window are required'},{ok:hasWindow&&!error,detail:error||(hours>0?`${hours.toFixed(1)}h window`:null)}];
    checks.forEach((check,i)=>{const row=q<HTMLElement>(`[data-generator-check="${i}"]`),symbol=get<HTMLElement>(row,'[data-generator-check-icon]'),label=get<HTMLElement>(row,'[data-generator-check-label]'),detail=get<HTMLElement>(row,'[data-generator-check-detail]');symbol.classList.toggle('text-isk',check.ok);symbol.classList.toggle('text-gray-600',!check.ok);get<HTMLElement>(row,'[data-generator-check-empty]').hidden=check.ok;get<HTMLElement>(row,'[data-generator-check-done]').hidden=!check.ok;label.classList.toggle('text-gray-400',check.ok);label.classList.toggle('text-gray-300',!check.ok);detail.hidden=!check.detail;detail.textContent=check.detail||'';detail.classList.toggle('text-gray-600',check.ok);detail.classList.toggle('text-amber-400/80',!check.ok);});
    button('[data-generator-load]',ready,state.loadingEntities,state.loadingEntities?'Loading entities…':'Load Entities','load');
    const assigned=state.sideA.length+state.sideB.length,canPreview=state.sideA.length>0&&state.sideB.length>0;
    const status=q<HTMLElement>('[data-generator-assigned-status]');status.classList.toggle('text-gray-400',canPreview);status.classList.toggle('text-amber-400/80',!canPreview);
    if(canPreview){const a=document.createElement('span'),b=document.createElement('span');a.className='text-red-400';a.textContent=String(state.sideA.length);b.className='text-blue-400';b.textContent=String(state.sideB.length);status.replaceChildren(a,document.createTextNode(' vs '),b,document.createTextNode(` · ${assigned} of ${state.entities.length} assigned`));}else status.textContent='Both sides need at least one entity.';
    button('[data-generator-preview]',canPreview,state.loadingPreview,state.loadingPreview?'Generating…':'Generate Preview','preview');
    const save=q<HTMLButtonElement>('[data-generator-save]');save.disabled=state.saving||state.uncertainSave||!state.preview;get<HTMLElement>(save,'[data-generator-save-spinner]').hidden=!state.saving;get<HTMLElement>(save,'[data-generator-save-label]').textContent=state.saving?'Saving…':prefill.battleId>0?`Update Battle #${prefill.battleId}`:'Save Battle Report';
    const extra=root.querySelector<HTMLElement>('[data-generator-extra-teams]');if(extra){extra.hidden=!state.dropped;extra.textContent=`This battle had ${state.dropped+2} sides and the generator holds two. The extra ${state.dropped===1?'side is':'sides are'} waiting in Undecided — place them before saving or they are dropped.`;}
  }
  const systemNodes=new Map([...root.querySelectorAll<HTMLElement>('[data-generator-systems] > [data-generator-system-id]')].map(node=>[Number(node.dataset.generatorSystemId),node]));
  function syncNodes(holder:HTMLElement,nodes:HTMLElement[]){for(let i=0;i<nodes.length;i++)if(holder.children[i]!==nodes[i])placeElement(holder,nodes[i],holder.children[i]??null);const keep=new Set(nodes);for(const child of Array.from(holder.children))if(child instanceof HTMLElement&&!keep.has(child))child.remove();}
  function renderSystems() {
    const holder=q<HTMLElement>('[data-generator-systems]');
    const nodes=state.systems.map(system=>{
      let node=systemNodes.get(system.id);if(!node){node=clone('[data-generator-system-template]');node.dataset.generatorSystemId=String(system.id);const image=get<HTMLImageElement>(node,'img');image.src=`/images/systems/${system.id}?size=32`;image.alt=system.name;get<HTMLElement>(node,'[data-generator-system-name]').textContent=system.name;get<HTMLButtonElement>(node,'button').setAttribute('aria-label',`Remove ${system.name}`);systemNodes.set(system.id,node);}return node;
    });syncNodes(holder,nodes);const keep=new Set(state.systems.map(system=>system.id));for(const id of systemNodes.keys())if(!keep.has(id))systemNodes.delete(id);
  }
  const entityNodes=new Map<string,HTMLElement>();
  function renderEntities() {
    const lists:Record<Lane,GeneratorEntity[]>={a:state.sideA,b:state.sideB,u:generatorUndecided(state.entities,state.sideA,state.sideB,state.filter)};
    for(const lane of ['a','b','u'] as const) {const rows=lists[lane];
      const holder=q<HTMLElement>(`[data-generator-lane="${lane}"]`);
      const nodes=rows.map(entity=>{
        const nodeKey=(lane==='u'?'u:':'picked:')+key(entity);
        let node=entityNodes.get(nodeKey);
        if(!node){node=clone(lane==='u'?'[data-generator-unassigned-template]':'[data-generator-assigned-template]');node.dataset.generatorEntityKey=key(entity);const image=get<HTMLImageElement>(node,'img');image.alt=entity.name;image.title=entity.name;
          image.src=entity.type==='faction'?`/images/factions/${entity.id}/logo?size=64`:`https://images.evetech.net/${entity.type==='alliance'?'alliances':'corporations'}/${entity.id}/logo?size=64`;
          for(const control of node.querySelectorAll<HTMLElement>('[data-generator-move]')){const target=control.dataset.generatorMove;control.setAttribute('aria-label',target==='u'?`Remove ${entity.name}`:`Move ${entity.name} to Side ${target==='a'?1:2}`);control.title=target==='u'?'Remove':`Move to Side ${target==='a'?1:2}`;}
          entityNodes.set(nodeKey,node);
        }
        get<HTMLElement>(node,'[data-generator-entity-name]').textContent=entity.name;
        get<HTMLElement>(node,'[data-generator-entity-detail]').textContent=entity.type+(lane==='u'&&entity.type==='corporation'&&entity.alliance_name?` · ${entity.alliance_name}`:lane==='u'&&entity.type==='alliance'?` · ${state.corps.filter(c=>c.alliance_id===entity.id).length} corps`:'');
        return node;
      });syncNodes(holder,nodes);
      if(lane!=='u'){
        q<HTMLElement>(`[data-generator-side-heading="${lane}"]`).textContent=`${sideName(lane)} (${rows.length})`;q<HTMLElement>(`[data-generator-side-count="${lane}"]`).textContent=String(rows.length);q<HTMLElement>(`[data-generator-side-empty="${lane}"]`).hidden=!!rows.length;
      }
    }
    const keep=new Set(state.entities.flatMap(entity=>['u:'+key(entity),'picked:'+key(entity)]));for(const id of entityNodes.keys())if(!keep.has(id))entityNodes.delete(id);
    q<HTMLElement>('[data-generator-unassigned-heading]').textContent=`Unassigned (${lists.u.length})`;q<HTMLElement>('[data-generator-filter-count]').textContent=`${lists.u.length} unassigned`;
    const empty=q<HTMLElement>('[data-generator-unassigned-empty]');empty.hidden=!!lists.u.length;empty.textContent=state.entities.length?'All entities assigned':'No entities found for this window';empty.classList.toggle('text-isk/60',!!state.entities.length);empty.classList.toggle('text-gray-600',!state.entities.length);
  }
  async function loadEntities() {
    if(state.loadingEntities||!state.authenticated||!generatorCanLoad(state.systems,start.value,end.value))return false;
    cancel('entities');const controller=new AbortController();state.requests.set('entities',controller);state.loadingEntities=true;showError('[data-generator-load-error]','');render();
    try {
      const result=readGeneratorEntities(await io.entities(windowBody(),controller.signal));
      if(state.requests.get('entities')!==controller||!state.alive)return false;
      const listed=generatorEntityList(result);state.entities=listed;state.corps=result.corporations;state.sideA=generatorKeepAssignments(state.sideA,listed);state.sideB=generatorKeepAssignments(state.sideB,listed);invalidate();state.step=2;renderEntities();render();return true;
    } catch(error){if(!isAbortError(error)&&state.requests.get('entities')===controller){showError('[data-generator-load-error]',errorMessage(error));if(error instanceof BrowserRequestError&&error.status===401)state.authenticated=false;}return false;}
    finally{if(state.requests.get('entities')===controller){state.requests.delete('entities');state.loadingEntities=false;render();}}
  }
  async function seedExisting() {
    if(!prefill.battleId)return;
    cancel('seed');const controller=new AbortController();state.requests.set('seed',controller);
    try{const existing=await io.seed(prefill.battleId,controller.signal);if(!state.alive||state.requests.get('seed')!==controller)return;const seeded=generatorSeedTeams(state.entities,readGeneratorSeedTeams(existing));Object.assign(state,seeded,{dropped:seeded.droppedTeamCount});renderEntities();render();}catch{/* The window is still usable when the old battle is unavailable. */}
    finally{if(state.requests.get('seed')===controller)state.requests.delete('seed');}
  }
  async function preview() {
    if(state.loadingPreview||!state.sideA.length||!state.sideB.length)return;
    cancel('preview');const controller=new AbortController();state.requests.set('preview',controller);state.loadingPreview=true;showError('[data-generator-preview-error]','');render();
    try{
      const text=await io.preview(draftBody(),controller.signal);
      const doc=new DOMParser().parseFromString(text,'text/html'),content=doc.querySelector<HTMLElement>('[data-generator-preview-loaded]');if(!content)throw new Error('Could not render preview');
      if(state.requests.get('preview')!==controller||!state.alive)return;
      state.preview=readGeneratorPreview(JSON.parse(get(content,'#battle-generator-preview-data').textContent??'null'));q<HTMLElement>('[data-generator-preview-content]').replaceChildren(document.importNode(content,true));state.step=3;render();
    }catch(error){if(!isAbortError(error)&&state.requests.get('preview')===controller){showError('[data-generator-preview-error]',errorMessage(error));if(error instanceof BrowserRequestError&&error.status===401)state.authenticated=false;}}
    finally{if(state.requests.get('preview')===controller){state.requests.delete('preview');state.loadingPreview=false;render();}}
  }
  async function save() {
    if(state.saving||state.uncertainSave||!state.preview)return;
    const controller=new AbortController(),revision=state.revision;state.requests.set('save',controller);state.saving=true;showError('[data-generator-save-error]','');render();
    try{const body=prefill.battleId>0?{...state.preview,battle_id:prefill.battleId}:state.preview,result=await io.save(body,controller.signal);if(!state.alive||state.requests.get('save')!==controller)return;if(!Number.isSafeInteger(result?.battle_id)||result.battle_id<=0){state.uncertainSave=true;throw new Error('The saved battle could not be opened. Check your battles before retrying.')}if(state.revision===revision)location.assign(`/battle/${result.battle_id}`);else{prefill.battleId=result.battle_id;const box=q('[data-generator-save-message]'),link=document.createElement('a');link.href=`/battle/${result.battle_id}`;link.className='underline';link.textContent=`Battle #${result.battle_id}`;box.replaceChildren('Saved ',link,'. You have newer unsaved changes.');box.hidden=false;}}
    catch(error){if(state.requests.get('save')!==controller||isAbortError(error))return;showError('[data-generator-save-error]',errorMessage(error));if(error instanceof BrowserRequestError&&error.status===401){state.authenticated=false;q('[data-utility-login-error]').textContent='Your session expired. Log in again to continue.';q('[data-utility-login-error]').hidden=false;}}
    finally{if(state.requests.get('save')===controller){state.requests.delete('save');state.saving=false;render();}}
  }
  async function login() {
    try{const result=await io.login('/auth/login?'+new URLSearchParams({redirect:location.pathname+location.search}));if(result?.url)location.assign(result.url);}catch(error){if(!state.alive||isAbortError(error))return;const box=q<HTMLElement>('[data-utility-login-error]');box.textContent=errorMessage(error);box.hidden=false;}
  }
  async function identity() {
    if(!state.alive)return;cancel('identity');const controller=new AbortController();state.requests.set('identity',controller);const epoch=++state.authGeneration;
    try{const result=await io.identity(controller.signal);if(epoch!==state.authGeneration||!state.alive)return;state.authenticated=!!result?.user;render();if(state.authenticated&&state.auto){state.auto=false;if(await loadEntities())await seedExisting();}}
    catch{/* Keep the SSR session; preview/save independently check authorization. */}
    finally{if(state.requests.get('identity')===controller)state.requests.delete('identity');}
  }
  function windowChanged(){cancel('entities');cancel('seed');invalidate();showError('[data-generator-load-error]','');renderSystems();render();}
  function pick(event:CustomEvent<unknown>){const hit=event.detail;if(!isRecord(hit)||hit.type!=='system'||!isGeneratorSystem(hit))return;if(state.systems.length<5&&!state.systems.some(s=>s.id===hit.id)){state.systems.push({id:hit.id,name:hit.name});windowChanged();}}
  function change(event:Event){if(event.target===start||event.target===end)windowChanged();}
  function input(event:Event){const target=event.target;if(!(target instanceof HTMLInputElement))return;if(target.matches('[data-generator-entity-filter]')){state.filter=target.value;renderEntities();}if(target.matches('[data-generator-side-name]')){invalidate();renderEntities();render();}}
  function keyboard(event:KeyboardEvent){const target=eventElement(event.target);if(event.key==='Enter'&&target instanceof HTMLElement&&target.matches('[data-generator-move][role="button"]')){event.preventDefault();target.click();}}
  function click(event:MouseEvent){const target=eventElement(event.target);if(!target)return;
    const move=target.closest<HTMLElement>('[data-generator-move]');if(move){const entity=state.entities.find(e=>key(e)===move.closest<HTMLElement>('[data-generator-entity-key]')?.dataset.generatorEntityKey);const targetLane=move.dataset.generatorMove;if(entity&&(targetLane==='a'||targetLane==='b'||targetLane==='u')){cancel('seed');Object.assign(state,generatorMove(state.sideA,state.sideB,entity,targetLane));invalidate();renderEntities();render();}return;}
    const remove=target.closest<HTMLElement>('[data-generator-remove-system]');if(remove){const id=Number(remove.closest<HTMLElement>('[data-generator-system-id]')?.dataset.generatorSystemId);state.systems=state.systems.filter(s=>s.id!==id);windowChanged();return;}
    const step=target.closest<HTMLButtonElement>('[data-wizard-step]');if(step&&!step.disabled){cancel('preview');cancel('seed');state.step=Number(step.dataset.wizardStep);render();return;}
    const back=target.closest<HTMLElement>('[data-generator-back]');if(back){cancel('preview');cancel('seed');state.step=Number(back.dataset.generatorBack);render();return;}
    if(target.closest<HTMLElement>('[data-generator-load]'))loadEntities();else if(target.closest<HTMLElement>('[data-generator-preview]'))preview();else if(target.closest<HTMLElement>('[data-generator-save]'))save();else if(target.closest<HTMLElement>('[data-utility-login]'))login();
  }
  function imageError(event:Event){const image=event.target;if(!(image instanceof HTMLImageElement)||!image.matches('[data-generator-entity-image]'))return;const fallback=document.createElement('span');fallback.className=image.className+' flex items-center justify-center text-gray-500';fallback.title=image.title;for(const name of ['data-generator-move','role','aria-label','tabindex']){const value=image.getAttribute(name);if(value!==null)fallback.setAttribute(name,value)}fallback.append(icon('flag'));image.replaceWith(fallback);}
  initializeDateTimePickers(root);initializeSearchPickers(root);renderSystems();renderEntities();render();
  if(state.authenticated&&state.auto){state.auto=false;loadEntities().then(loaded=>{if(loaded)seedExisting();});}else identity();
  function hide(){state.alive=false;state.authGeneration++;state.uncertainSave||=state.saving;for(const part of [...state.requests.keys()])cancel(part);state.saving=false;closeSearchPickers(root);if(state.uncertainSave)showError('[data-generator-save-error]','Saving was interrupted. Check your battles before retrying.');render();}
  function resume(){state.alive=true;render();void identity();}
  return {click,input,change,pick,keyboard,imageError,identity,hide,resume};
}
export class BattleGeneratorController extends RequestComponent {
 private ui:ReturnType<typeof createGenerator>|undefined
 connect(){if(!this.ui)this.ui=createGenerator(this.element,this);else this.ui.resume()}
 entities(body:BattleGeneratorEntitiesData['body'],signal:AbortSignal){return this.request<unknown>('/api/battle/generator/entities',{method:'POST',body,signal})}
 seed(id:number,signal:AbortSignal){return this.request<unknown>(`/api/battle/${id}`,{signal})}
 preview(body:DraftBody,signal:AbortSignal){return this.text('/_fragments/battlegenerator/preview','Failed to generate preview',{method:'POST',headers:{'Content-Type':'application/json',Accept:'text/html'},body:JSON.stringify(body),signal})}
 save(body:GeneratorPreview,signal:AbortSignal){return this.request<BattleGeneratorSaveResponses[200]>('/api/battle/generator/save',{method:'POST',body,signal})}
 identity(signal:AbortSignal){return this.request<AuthMeLegacyResponses[200]>('/auth/me',{signal})}
 login(path:string){return this.request<AuthLoginLegacyResponses[200]>(path)}
 click(event:MouseEvent){this.ui?.click(event)}
 input(event:Event){this.ui?.input(event)}
 change(event:Event){this.ui?.change(event)}
 pick(event:CustomEvent<unknown>){this.ui?.pick(event)}
 keyboard(event:KeyboardEvent){this.ui?.keyboard(event)}
 imageError(event:Event){this.ui?.imageError(event)}
 focus(){void this.ui?.identity()}
 show(event:PageTransitionEvent){if(event.persisted)this.ui?.resume()}
 hide(){this.ui?.hide();super.hide()}
}
registerController('battle-generator',BattleGeneratorController)
