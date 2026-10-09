import {selectRoamCompositionRows,readRoamComposition,type RoamCompositionOptions,type RoamPilotRow,type RoamAggregateRow} from './shared/roam-composition';
import {Controller} from '@hotwired/stimulus';
import {registerController} from './controllers/application';
import {requiredElement,cloneComponent,eventElement} from './shared/dom';
import {formatIsk} from '../utils/formatters';
import {iconHref} from './icon-hrefs';
const views=['pilots','ships','groups'] as const;
const sorts=['default','ship','pilot','pilots','observations','losses','iskLost','damage'] as const;
const isView=(value:unknown):value is NonNullable<RoamCompositionOptions['view']>=>views.some(view=>view===value);
const isSort=(value:unknown):value is NonNullable<RoamCompositionOptions['sort']>=>sorts.some(sort=>sort===value);
const isPilot=(row:RoamPilotRow|RoamAggregateRow):row is RoamPilotRow=>'pilot'in row;
export {selectRoamCompositionRows};

function createComposition(section:HTMLElement) {
  const q=<T extends Element=HTMLElement>(selector:string)=>requiredElement<T>(section,selector);
  const field=<T extends Element=HTMLElement>(node:ParentNode,selector:string)=>requiredElement<T>(node,selector);
  const data = readRoamComposition(JSON.parse(q('#roam-composition-data').textContent??'null'));
  const body = q<HTMLElement>('[data-roam-composition-rows]');
  const empty = q<HTMLElement>('[data-roam-composition-empty]');
  const count = new Intl.NumberFormat('en-US');
  const compact=formatIsk;
  const options:Required<RoamCompositionOptions> = { view:'pilots',search:'',group:null,losses:false,sort:'default',descending:true };
  const render = () => {
    const rows = selectRoamCompositionRows(data.rows,options);
    const pilotView = options.view === 'pilots';
    q<HTMLElement>('[data-roam-composition-reset]').hidden = !options.search && !options.group && !options.losses;
    q<HTMLElement>('[data-roam-composition-count]').textContent = pilotView ? `${data.pilots} pilots · ${rows.length} rows` : `${rows.length} ${options.view === 'ships' ? 'ship types' : 'ship groups'}`;
    empty.hidden = rows.length !== 0; empty.textContent = pilotView ? 'No pilots match these filters.' : 'No observed ships match these filters.';
    for (const button of section.querySelectorAll<HTMLElement>('[data-roam-composition-view]')) {
      const active = button.dataset.roamCompositionView === options.view;
      button.setAttribute('aria-pressed',String(active));
      button.className = `inline-flex items-center gap-2 rounded-lg border px-3 py-2 text-sm transition-colors ${active ? 'border-blue-500/30 bg-blue-500/10 text-blue-300' : 'border-white/10 text-gray-400 hover:text-gray-200'}`;
    }
    for (const button of section.querySelectorAll<HTMLElement>('[data-roam-composition-sort]')) {
      if (['pilot','pilots'].includes(button.dataset.roamCompositionSort??'')) { button.dataset.roamCompositionSort = pilotView ? 'pilot' : 'pilots'; field<HTMLElement>(button,'span').textContent = pilotView ? 'Pilot' : 'Pilots'; }
      if (button.dataset.roamCompositionSort === 'ship') field<HTMLElement>(button,'span').textContent = options.view === 'groups' ? 'Ship group' : 'Ship';
      const icon = field<SVGUseElement>(button,'use');
      const selected = button.dataset.roamCompositionSort === options.sort;
      icon.setAttribute('href',iconHref(selected ? options.descending ? 'chevron-down' : 'chevron-up' : 'chevrons-up-down'));
    }
    const nodes = rows.map(row => {
      const node = cloneComponent(section,'[data-roam-composition-row-template]');
      const ship = isPilot(row) ? row.ship : { ship_type_id:row.id,ship_name:row.name,ship_group_name:row.groupName };
      field<HTMLElement>(node,'[data-roam-unknown-ship]').hidden = !!ship;
      field<HTMLElement>(node,'[data-roam-ship-cell]').hidden = !ship;
      if (ship) {
        const image = field<HTMLImageElement>(node,'[data-roam-ship-image]'); image.hidden = options.view === 'groups' || !ship.ship_type_id;
        if (!image.hidden) { image.src = `/images/types/${ship.ship_type_id}/render?size=32`; image.alt = ship.ship_name; }
        const link = field<HTMLAnchorElement>(node,'[data-roam-ship-link]'); link.href = `/${options.view === 'groups' ? 'group' : 'item'}/${ship.ship_type_id}`; link.textContent = ship.ship_name; link.hidden = !ship.ship_type_id;
        const name = field<HTMLElement>(node,'[data-roam-ship-name]'); name.hidden = !!ship.ship_type_id; name.textContent = ship.ship_name;
        const group = field<HTMLElement>(node,'[data-roam-ship-group]'); group.hidden = !pilotView; group.textContent = ship.ship_group_name;
      }
      field<HTMLElement>(node,'[data-roam-pilot-cell]').hidden = !pilotView;
      const aggregate = field<HTMLElement>(node,'[data-roam-aggregate-pilots]'); aggregate.hidden = pilotView;
      if (isPilot(row)) {
        const image = field<HTMLImageElement>(node,'[data-roam-pilot-image]'); image.src = `https://images.evetech.net/characters/${row.pilot.character_id}/portrait?size=32`; image.alt = row.pilot.name;
        const link = field<HTMLAnchorElement>(node,'[data-roam-pilot-link]'); link.href = `/character/${row.pilot.character_id}`; link.textContent = row.pilot.name;
        field<HTMLElement>(node,'[data-roam-pilot-corp]').textContent = row.pilot.corporation_name;
      } else aggregate.textContent = `${count.format(row.pilotIds.size)} ${row.pilotIds.size === 1 ? 'pilot' : 'pilots'}`;
      field<HTMLElement>(node,'[data-roam-row-observations]').textContent = isPilot(row) ? row.ship ? count.format(row.ship.killmails) : '—' : count.format(row.observations);
      field<HTMLElement>(node,'[data-roam-row-damage]').textContent = isPilot(row) ? row.ship ? count.format(row.ship.damage_done) : '—' : count.format(row.damage);
      const losses = isPilot(row) ? row.ship?.losses ?? 0 : row.losses;
      for (const [key,value,text] of [['losses',losses,losses ? count.format(losses) : '—'],['isk-lost',row.iskLost,row.iskLost ? compact(row.iskLost) : '—']] as const) {
        const cell = field<HTMLElement>(node,`[data-roam-row-${key}]`); cell.textContent = text;
        cell.classList.toggle('text-red-400',!!value); cell.classList.toggle('text-gray-600',!value);
      }
      return node;
    });
    body.replaceChildren(...nodes);
  };
  function input(event:Event){const target=event.target;if(target instanceof HTMLInputElement&&target.matches('[data-roam-composition-search]')){options.search=target.value;render();}}
  function change(event:Event){const target=event.target;if(target instanceof HTMLSelectElement&&target.matches('[data-roam-composition-group]'))options.group=Number(target.value)||null;else if(target instanceof HTMLInputElement&&target.matches('[data-roam-composition-loss]'))options.losses=target.checked;else return;render();}
  function click(event:MouseEvent){const button=eventElement(event.target)?.closest<HTMLButtonElement>('button');if(!button||button.disabled)return;
    if(button.hasAttribute('data-roam-composition-view')){const view=button.dataset.roamCompositionView;if(!isView(view))return;options.view=view;options.sort='default';options.descending=true;}
    else if(button.hasAttribute('data-roam-composition-sort')){const key=button.dataset.roamCompositionSort;if(!isSort(key))return;options.descending=options.sort===key?!options.descending:true;options.sort=key;}
    else if(button.hasAttribute('data-roam-composition-reset')){options.search='';options.group=null;options.losses=false;q<HTMLInputElement>('[data-roam-composition-search]').value='';q<HTMLSelectElement>('[data-roam-composition-group]').value='';q<HTMLInputElement>('[data-roam-composition-loss]').checked=false;}
    else return;render();
  }
  // Keep the initial Go-rendered rows/images until a control changes.
  return {click,input,change};
}
export class RoamCompositionController extends Controller<HTMLElement>{
 private ui:ReturnType<typeof createComposition>|undefined;
 connect(){this.ui??=createComposition(this.element)}
 click(event:MouseEvent){this.ui?.click(event)}
 input(event:Event){this.ui?.input(event)}
 change(event:Event){this.ui?.change(event)}
}
registerController('roam-composition',RoamCompositionController);
