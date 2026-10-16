import { initializeDateTimePickers } from "./date-time-picker.js";
import "./saved-searches";
import type {SavedSearchCallbacks,SavedSearchCurrent,SavedSearchLoad} from "./saved-searches";
import {RequestComponent} from "./controllers/request-component";
import {registerController} from "./controllers/application";
import {iconHref} from './icon-hrefs';
import {requiredElement,eventElement} from "./shared/dom";
import {errorMessage,isAbortError} from "./shared/json";
import {advancedRoles,advancedLocations,isAdvancedRole,isAdvancedScalar,readAdvancedState,readAdvancedOptions,readAdvancedHits,type AdvancedHit,type AdvancedRole} from "./advanced-search-state";
import { savedSearchDocument, savedSearchQuery } from "./saved-search-data.js";
import {
  defaultAdvancedFilters,
  restoreAdvancedFilters,
  hasAdvancedFilters,
  advancedFiltersForURL,
  advancedURL,
  advancedRoleColor,
  advancedEntityLabel,
  advancedEntityImage,
  advancedLocationLabel,
  advancedActiveChips,
  clearAdvancedChip,
  advancedButtonClass,
} from "./advanced-search-state.js";

function createAdvancedSearch(root:HTMLElement,io:AdvancedSearchController) {
  const q=<T extends Element=HTMLElement>(selector:string)=>requiredElement<T>(root,selector);
  const get=<T extends Element=HTMLElement>(parent:ParentNode,selector:string)=>requiredElement<T>(parent,selector);
  const state = readAdvancedState(JSON.parse(
    q("#advanced-search-state").textContent??"null",
  ));
  state.filters = restoreAdvancedFilters(state.filters);
  const options = readAdvancedOptions(JSON.parse(
      q("#advanced-search-options").textContent??"null",
    )),
    results = q<HTMLElement>("[data-advanced-results]");
  const input = q<HTMLInputElement>("[data-advanced-query]"),
    suggestions = q<HTMLElement>("[data-advanced-suggestions]"),
    spinner = q<HTMLElement>("[data-advanced-search-spinner]");
  const from = q<HTMLInputElement>('[name="advanced_from"]'),
    to = q<HTMLInputElement>('[name="advanced_to"]');
  let collapsed = false,
    searchTimer:ReturnType<typeof setTimeout>|undefined = undefined,
    searchRequest:AbortController|undefined = undefined,
    resultTimer:ReturnType<typeof setTimeout>|undefined = undefined,
    resultRequest:AbortController|undefined = undefined,
    hits:AdvancedHit[] = [],
    settingDates = false;
  let interrupted=false;
  let page = 1,
    limit = 50,
    cursors:Array<string|null> = [null],
    drilldownShipName = "";
  const clone = (id:string) => {const template=q<HTMLTemplateElement>("#"+id),child=template.content.firstElementChild?.cloneNode(true);if(!(child instanceof HTMLElement))throw new Error("Missing template "+id);return child;};
  const node = <K extends keyof HTMLElementTagNameMap>(tag:K, cls:string, text?:string) => {
    const el = document.createElement(tag);
    if (cls) el.className = cls;
    if (text != null) el.textContent = text;
    return el;
  };
  const icon = (name:string) => {
    const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg"),
      use = document.createElementNS("http://www.w3.org/2000/svg", "use");
    svg.setAttribute("class", "icon");
    svg.setAttribute("viewBox", "0 0 24 24");
    svg.setAttribute("aria-hidden", "true");
    use.setAttribute("href", iconHref(name));
    svg.append(use);
    return svg;
  };
  const switchIcon = (container:Element, name:string) =>
    container.querySelector("use")?.setAttribute("href", iconHref(name));
  const notice = (text:string) => {
    const el = q<HTMLElement>("[data-advanced-notice]");
    el.textContent = text;
    el.hidden = !text;
  };
  const syncURL = (push = false, explicit:URLSearchParams|null = null) => {
    const query = explicit ?? advancedURL(state),
      path = "/advancedsearch" + (query.size ? "?" + query : "");
    history[push ? "pushState" : "replaceState"](null, "", path);
  };
  function setDates(a = "", b = "") {
    settingDates = true;
    try {
      from.value = a;
      to.value = b;
      from.dispatchEvent(new Event("change", { bubbles: true }));
      to.dispatchEvent(new Event("change", { bubbles: true }));
    } finally {
      settingDates = false;
    }
  }
  function renderTags() {
    const rows = [];
    for (const role of advancedRoles)
      for (const [index, e] of state.filters.entities[role].entries()) {
        const tag = clone("advanced-entity-template");
        tag.className =
          "inline-flex items-center gap-1.5 px-2 py-1 rounded border text-xs font-medium " +
          advancedRoleColor(role, e.exclude);
        const image = advancedEntityImage(e.type, e.id),
          img = get<HTMLImageElement>(tag,"[data-tag-image]");
        img.hidden = !image;
        if (image) img.src = image;
        get<HTMLElement>(tag,"[data-tag-icon]").hidden = !!image;
        get<HTMLElement>(tag,"[data-tag-name]").textContent = e.name;
        get<HTMLElement>(tag,"[data-tag-label]").textContent = advancedEntityLabel(
          role,
          e,
        );
        const button = get<HTMLElement>(tag,"[data-advanced-remove-entity]");
        button.dataset.advancedRemoveEntity = role;
        button.dataset.index = String(index);
        button.setAttribute(
          "aria-label",
          "Remove " + e.name + " " + advancedEntityLabel(role, e),
        );
        rows.push(tag);
      }
    for (const [index, item] of state.filters.items.entries()) {
      const tag = clone("advanced-item-template");
      tag.className =
        "inline-flex items-center gap-1 px-2 py-1 rounded border text-xs font-medium " +
        (item.groupId != null
          ? "bg-cyan-500/20 text-cyan-400 border-cyan-500/30"
          : "bg-purple-500/20 text-purple-400 border-purple-500/30");
      const img = get<HTMLImageElement>(tag,"[data-tag-image]");
      img.hidden = item.typeId == null;
      if (item.typeId != null)
        img.src = advancedEntityImage("item", item.typeId);
      get<HTMLElement>(tag,"[data-tag-icon]").hidden = item.typeId != null;
      get<HTMLElement>(tag,"[data-tag-name]").textContent = item.name;
      for (const field of ["slot", "side"] as const) {
        const button = get<HTMLButtonElement>(tag,"[data-advanced-item-" + field + "]");
        button.dataset[
          "advancedItem" + field[0].toUpperCase() + field.slice(1)
        ] = String(index);
        button.textContent = item[field];
        button.title =
          (field === "slot" ? "Slot scope: " : "Side: ") + item[field];
      }
      const button = get<HTMLElement>(tag,"[data-advanced-remove-item]");
      button.dataset.advancedRemoveItem = String(index);
      button.setAttribute("aria-label", "Remove " + item.name);
      rows.push(tag);
    }
    const label = advancedLocationLabel(state.filters.location);
    if (label) {
      const tag = clone("advanced-location-template");
      get<HTMLElement>(tag,"[data-tag-name]").textContent = label;
      rows.push(tag);
    }
    const tags = q<HTMLElement>("[data-advanced-tags]");
    tags.replaceChildren(...rows);
    tags.hidden = !rows.length;
  }
  function render() {
    const f = state.filters;
    renderTags();
    const chips = advancedActiveChips(f, options),
      applied = q<HTMLElement>("[data-advanced-applied]");
    const label = node(
      "span",
      "text-fine text-secondary font-medium uppercase tracking-wider mr-0.5",
      "Applied",
    );
    applied.replaceChildren(
      label,
      ...chips.map((chip) => {
        const button = clone("advanced-chip-template");
        button.dataset.advancedRemoveChip = chip.key;
        button.title = `Remove ${chip.group}: ${chip.label}`;
        get<HTMLElement>(button,"[data-chip-group]").textContent = chip.group;
        get<HTMLElement>(button,"[data-chip-label]").textContent = chip.label;
        return button;
      }),
    );
    applied.hidden = !chips.length;
    for (const button of root.querySelectorAll<HTMLButtonElement>("[data-advanced-option]")) {
      const key = button.dataset.advancedOption,
        value = button.dataset.value??"";
      const active =
        key === "security"
          ? f.location.securityTypes.includes(value)
          : key === "timePreset"
            ? f.timeRange.preset === value
            : key === "sortField"
              ? f.sort.field === value
              : key === "sortDirection"
                ? f.sort.direction === value
                : isAdvancedScalar(key)&&f[key] === value;
      button.className =
        "px-2 py-1 text-xs font-medium rounded border transition-colors " +
        advancedButtonClass(active);
      button.setAttribute("aria-pressed", String(active));
    }
    q<HTMLSelectElement>("[data-advanced-label]").value = f.label ?? "";
    for (const field of ["iskMin", "iskMax"] as const)
      q<HTMLInputElement>('[data-advanced-isk="' + field + '"]').value =
        f[field];
    for (const button of root.querySelectorAll<HTMLButtonElement>("[data-advanced-view]")) {
      const active = state.view === button.dataset.advancedView;
      button.setAttribute("aria-pressed", String(active));
      button.className =
        "px-3 py-1 text-xs font-medium border transition-colors " +
        (button.dataset.advancedView === "kills"
          ? "rounded-l-lg"
          : "rounded-r-lg border-l-0") +
        " " +
        (active
          ? "bg-blue-500/20 text-link border-blue-400/40"
          : "bg-white/[0.04] text-gray-400 border-white/[0.08] hover:text-link");
    }
    q<HTMLElement>("[data-advanced-copy]").hidden = q("[data-advanced-clear]").hidden = !hasAdvancedFilters(f);
    q<HTMLElement>("[data-advanced-panels]").hidden = q("[data-advanced-sort]").hidden = collapsed;
    const count = q<HTMLElement>("[data-advanced-collapse-count]");
    count.hidden = !(collapsed && chips.length);
    count.textContent = `(${chips.length} applied)`;
    const collapse = q<HTMLElement>("[data-advanced-collapse]");
    collapse.setAttribute("aria-expanded", String(!collapsed));
    switchIcon(collapse, collapsed ? "chevron-right" : "chevron-down");
  }
  function readPager() {
    const list = results.querySelector<HTMLElement>("#kill-list");
    if (list) {
      page = Number(list.dataset.page);
      limit = Number(list.dataset.limit);
      if (list.dataset.nextCursor) cursors[page] = list.dataset.nextCursor;
    }
    const name = results.querySelector("[data-advanced-drilldown-name]");
    if (name) name.textContent = drilldownShipName;
  }
  async function loadResults({
    nextPage = 1,
    nextLimit = state.view === "kills" ? limit : 50,
    append = false,
  } = {}) {
    resultRequest?.abort();
    const controller = new AbortController();
    resultRequest = controller;
    page = nextPage;
    limit = nextLimit;
    const params = advancedURL(state);
    if (state.showResults && !params.has("q")) params.set("q", "{}");
    params.set("page", String(page));
    params.set("limit", String(limit));
    if (!state.showResults) params.set("empty", "1");
    if (nextPage > 1 && cursors[nextPage - 1])
      params.set("after", cursors[nextPage - 1]??"");
    if (append) {
      const cursor = results.querySelector<HTMLButtonElement>("[data-advanced-more-fits]")?.dataset
        .cursor;
      if (cursor) params.set("after", cursor);
    }
    results.setAttribute("aria-busy", "true");
    results.classList.add("advanced-results-loading");
    const more = results.querySelector<HTMLButtonElement>("[data-advanced-more-fits]");
    if (more) {
      more.disabled = true;
      more.textContent = "Loading fits…";
    }
    try {
      const text=await io.readFragment("/_fragments/advancedsearch?"+params,controller.signal),document=new DOMParser().parseFromString(text,"text/html"),
        next = document.querySelector<HTMLElement>("[data-advanced-result-content]");
      if (!next) throw new Error("Search results are unavailable");
      if (resultRequest !== controller) return;
      if (
        append &&
        next.querySelector("[data-advanced-fit-grid]") &&
        results.querySelector("[data-advanced-fit-grid]")
      ) {
        get(results,"[data-advanced-fit-grid]").append(
            ...Array.from(
              get(next,"[data-advanced-fit-grid]").children,
            ).map((el) => window.document.importNode(el, true)),
          );
        const button = next.querySelector<HTMLButtonElement>("[data-advanced-more-fits]");
        if (button) more?.replaceWith(window.document.importNode(button, true));
        else more?.parentElement?.remove();
      } else results.replaceChildren(window.document.importNode(next, true));
      readPager();
    } catch (cause) {
      if (!isAbortError(cause) && resultRequest === controller) {
        const error = node(
          "div",
          "glass-panel p-8 text-center text-sm text-red-400",
          errorMessage(cause),
        );
        error.setAttribute("role", "alert");
        results.replaceChildren(error);
      }
    } finally {
      if (resultRequest === controller) {
        resultRequest = undefined;
        results.removeAttribute("aria-busy");
        results.classList.remove("advanced-results-loading");
        if (more?.isConnected) {
          more.disabled = false;
          more.textContent = "Load more fits";
        }
      }
    }
  }
  function changed(typed = false) {
    io.cancelTimer(resultTimer);
    resultRequest?.abort();
    notice("");
    cursors = [null];
    render();
    syncURL();
    if (!hasAdvancedFilters(state.filters) && !state.hash) {
      state.showResults = false;
      limit = 50;
      loadResults();
      return;
    }
    resultTimer = io.schedule(
      () => {
        resultTimer=undefined;
        state.showResults = true;
        loadResults();
      },
      typed ? 500 : 0,
    );
  }
  function clearLocation() {
    for (const kind of advancedLocations)
      state.filters.location[`${kind}Id`] = state.filters.location[
        `${kind}Name`
      ] = null;
  }
  function hideSuggestions() {
    suggestions.hidden = true;
  }
  const suggestionActions=new Map<number,()=>void>();let actionID=0;
  function action(label:string,cls:string,run:()=>void,aria?:string){const button=node("button",cls,label);button.type="button";if(aria)button.setAttribute("aria-label",aria);const key=actionID++;suggestionActions.set(key,run);button.dataset.advancedSuggestionAction=String(key);return button;}
  function addEntity(hit:AdvancedHit, role:AdvancedRole, exclude = false) {
    const id = Number(hit.id.split("_").slice(1).join("_")),
      type = hit.type === "item" && role === "attacker" ? "weapon" : hit.type,
      list = state.filters.entities[role];
    if (
      list.some(
        (e) => e.id === id && e.type === type && !!e.exclude === exclude,
      )
    ) {
      hideSuggestions();
      return;
    }
    if (list.length >= 15) {
      notice(`Max 15 ${role} entities per filter`);
      hideSuggestions();
      return;
    }
    list.push({ id, type, name: hit.name, role, exclude });
    hideSuggestions();
    changed();
  }
  function addItem(hit:AdvancedHit, side = "victim") {
    const id = Number(hit.id.split("_").slice(1).join("_")),
      group = hit.type === "group",
      list = state.filters.items;
    if (
      list.some((i) =>
        group ? i.groupId === id && i.side === side : i.typeId === id,
      )
    ) {
      hideSuggestions();
      return;
    }
    if (list.length >= 15) {
      notice("Max 15 items per filter");
      hideSuggestions();
      return;
    }
    list.push({
      ...(group ? { groupId: id } : { typeId: id }),
      name: hit.name,
      slot: "any",
      side,
    });
    hideSuggestions();
    changed();
  }
  function addLocation(hit:AdvancedHit) {
    const id = Number(hit.id.split("_").slice(1).join("_"));
    clearLocation();
    state.filters.location.securityTypes = [];
    if(hit.type!=="system"&&hit.type!=="region"&&hit.type!=="constellation")return;
    state.filters.location[`${hit.type}Id`] = id;
    state.filters.location[`${hit.type}Name`] = hit.name;
    hideSuggestions();
    changed();
  }
  function renderSuggestions() {
    suggestionActions.clear();
    suggestions.replaceChildren(
      ...hits.map((hit) => {
        const row = clone("advanced-suggestion-template"),
          id = Number(hit.id.split("_").slice(1).join("_")),
          image = advancedEntityImage(hit.type, id),
          img = get<HTMLImageElement>(row,"[data-hit-image]");
        img.hidden = !image;
        if (image) img.src = image;
        get<HTMLElement>(row,"[data-hit-icon]").hidden = !!image;
        switchIcon(
          get<HTMLElement>(row,"[data-hit-symbol]"),
          hit.type === "group" ? "layers" : "map-pin",
        );
        get<HTMLElement>(row,"[data-hit-name]").textContent = hit.name;
        get<HTMLElement>(row,"[data-hit-type]").textContent = hit.type;
        const ticker = get<HTMLElement>(row,"[data-hit-ticker]");
        ticker.hidden = !hit.ticker;
        ticker.textContent = hit.ticker ? "[" + hit.ticker + "]" : "";
        const actions = get<HTMLElement>(row,"[data-hit-actions]");
        if (
          [
            "character",
            "corporation",
            "alliance",
            "ship",
            "shipgroup",
            "faction",
          ].includes(hit.type)
        ) {
          for (const [role, label, cls] of [
            [
              "victim",
              "Victim",
              "bg-red-500/20 text-red-400 hover:bg-red-500/30",
            ],
            [
              "both",
              "Both",
              "bg-green-500/20 text-green-400 hover:bg-green-500/30",
            ],
            [
              "attacker",
              "Attacker",
              "bg-blue-500/20 text-link hover:bg-blue-500/30",
            ],
          ] as const)
            actions.append(
              action(
                label,
                "px-2 py-1 text-fine font-medium rounded transition-colors " +
                  cls,
                () => addEntity(hit, role),
              ),
            );
          const exclude = action(
            "",
            "px-1.5 py-1 text-fine font-medium rounded bg-red-900/30 text-red-300 hover:bg-red-900/50 transition-colors",
            () => addEntity(hit, "victim", true),
            "Exclude " + hit.name,
          );
          exclude.title = "Exclude";
          exclude.append(icon("ban"));
          actions.append(exclude);
        } else if (hit.type === "item" || hit.type === "group") {
          actions.append(
            action(
              "Item",
              "px-2 py-1 text-fine font-medium rounded bg-purple-500/20 text-purple-400 hover:bg-purple-500/30 transition-colors",
              () => addItem(hit),
            ),
          );
          actions.append(
            action(
              "Weapon",
              "px-2 py-1 text-fine font-medium rounded bg-amber-500/20 text-amber-400 hover:bg-amber-500/30 transition-colors",
              () =>
                hit.type === "group"
                  ? addItem(hit, "attacker")
                  : addEntity(hit, "attacker"),
            ),
          );
        } else if (["system", "region", "constellation"].includes(hit.type))
          actions.append(
            action(
              "Set Location",
              "px-2 py-1 text-fine font-medium rounded bg-amber-500/20 text-amber-400 hover:bg-amber-500/30 transition-colors",
              () => addLocation(hit),
            ),
          );
        return row;
      }),
    );
    suggestions.hidden = !hits.length;
  }
  function search() {
    io.cancelTimer(searchTimer);
    searchRequest?.abort();
    const q = input.value.trim();
    hideSuggestions();
    spinner.hidden = true;
    if (q.length < 3) {
      hits = [];
      return;
    }
    const controller = new AbortController();
    searchRequest = controller;
    searchTimer = io.schedule(async () => {
      searchTimer=undefined;
      spinner.hidden = false;
      try {
        const data=await Promise.all([io.readSuggestions("/api/search?q="+encodeURIComponent(q),controller.signal),io.readSuggestions("/api/search?q="+encodeURIComponent(q)+"&type=group",controller.signal)]);
        if(searchRequest!==controller)return;hits=Array.from(new Map(data.flatMap(readAdvancedHits).map(hit=>[hit.id,hit])).values());
        renderSuggestions();
      } catch (cause) {
        if (!isAbortError(cause) && searchRequest === controller) {
          hits = [];
          notice("Search is unavailable. Please try again.");
        }
      } finally {
        if (searchRequest === controller) spinner.hidden = true;
      }
    }, 300);
  }
  function keyboard(event:KeyboardEvent){
    if(event.target!==input)return;
    if (event.key === "Escape") hideSuggestions();
    if (event.key === "Enter") event.preventDefault();
    if (event.key === "ArrowDown") {
      event.preventDefault();
      suggestions.querySelector("button")?.focus();
    }
  }

  function outside(event:PointerEvent){
    if (
      !q<HTMLElement>("[data-advanced-search-box]").contains(eventElement(event.target))
    )
      hideSuggestions();
  }

  function inputChanged(event:Event){
    if(event.target===input){search();return;}
    const target=event.target;if(!(target instanceof HTMLInputElement))return;
    if (target.matches("[data-advanced-isk]")) {
      const key=target.dataset.advancedIsk;if(key!=="iskMin"&&key!=="iskMax")return;
      state.filters[key] = target.value;
      state.filters.iskValue = null;
      changed(true);
    }
  }

  function change(event:Event){
    const target=event.target;if(!(target instanceof HTMLInputElement||target instanceof HTMLSelectElement))return;
    if (target.matches("[data-advanced-label]")) {
      state.filters.label = target.value || null;
      changed();
    }
    if (!settingDates && (event.target === from||event.target === to) && from.value && to.value) {
      state.filters.timeRange = { from: from.value, to: to.value };
      changed();
    }
  }

  async function click(event:MouseEvent){
    const targetElement=eventElement(event.target);if(!targetElement)return;
    const button = targetElement.closest<HTMLButtonElement>("button");
    const actionKey=button?.dataset.advancedSuggestionAction;if(actionKey!==undefined){event.stopPropagation();suggestionActions.get(Number(actionKey))?.();return;}
    if (button && root.contains(button) && !button.disabled) {
      if (button.hasAttribute("data-advanced-option")) {
        const key = button.dataset.advancedOption,
          value = button.dataset.value,
          f = state.filters;
        if(value===undefined)return;
        if (key === "security") {
          const list = f.location.securityTypes;
          f.location.securityTypes =
            list.includes(value) && list.length === 1 ? [] : [value];
        } else if (key === "timePreset") {
          f.timeRange = { preset: f.timeRange.preset === value ? null : value };
          setDates();
        } else if (key === "sortField") f.sort.field = value;
        else if (key === "sortDirection") f.sort.direction = value;
        else if(isAdvancedScalar(key)) {
          f[key] = f[key] === value ? null : value;
          if (key === "iskValue") f.iskMin = f.iskMax = "";
        }
        changed();
        return;
      }
      if (button.hasAttribute("data-advanced-remove-chip")) {
        const key = button.dataset.advancedRemoveChip??"";
        clearAdvancedChip(state.filters, key);
        if (key === "timePreset" || key === "timeRange") setDates();
        changed();
        return;
      }
      if (button.hasAttribute("data-advanced-remove-entity")) {
        const role=button.dataset.advancedRemoveEntity;if(!isAdvancedRole(role))return;
        state.filters.entities[role].splice(
          Number(button.dataset.index),
          1,
        );
        changed();
        return;
      }
      if (button.hasAttribute("data-advanced-remove-item")) {
        state.filters.items.splice(
          Number(button.dataset.advancedRemoveItem),
          1,
        );
        changed();
        return;
      }
      for (const field of ["slot", "side"] as const)
        if (button.hasAttribute("data-advanced-item-" + field)) {
          const item =
              state.filters.items[
                Number(
                  button.dataset[
                    "advancedItem" + field[0].toUpperCase() + field.slice(1)
                  ],
                )
              ],
            order =
              field === "slot"
                ? ["any", "fitted", "cargo"]
                : ["victim", "attacker", "either"];
          if (item) {
            item[field] =
              order[(order.indexOf(item[field]) + 1) % order.length];
            changed();
          }
          return;
        }
      if (button.hasAttribute("data-advanced-clear-location")) {
        clearLocation();
        changed();
        return;
      }
      if (button.hasAttribute("data-advanced-clear")) {
        state.filters = defaultAdvancedFilters();
        state.showResults = false;
        setDates();
        changed();
        return;
      }
      if (button.hasAttribute("data-advanced-collapse")) {
        collapsed = !collapsed;
        render();
        return;
      }
      if (button.hasAttribute("data-advanced-copy")) {
        try {
          await navigator.clipboard.writeText(location.href);
          switchIcon(button, "check");
          get<HTMLElement>(button,"span").textContent = "Copied!";
          io.schedule(() => {
            switchIcon(button, "link");
            get<HTMLElement>(button,"span").textContent = "Copy Link";
          }, 2000);
        } catch {}
        return;
      }
      if (button.hasAttribute("data-advanced-view")) {
        const view=button.dataset.advancedView;if(view!=="kills"&&view!=="fits")return;
        if (state.view !== view) limit = 50;
        state.view = view;
        render();
        syncURL();
        if (
          (hasAdvancedFilters(state.filters) || state.hash) &&
          state.showResults
        ) {
          cursors = [null];
          loadResults();
        }
        return;
      }
      if (button.hasAttribute("data-advanced-dedup")) {
        const dedup=button.dataset.advancedDedup;if(dedup!=="none"&&dedup!=="exact"&&dedup!=="family")return;
        state.dedup = dedup;
        syncURL();
        loadResults();
        return;
      }
      if (button.hasAttribute("data-fit-show-kills")) {
        if (button.dataset.fitShowKills && (button.dataset.fitDedup==="exact"||button.dataset.fitDedup==="family")) {
          state.hash = button.dataset.fitShowKills;
          state.hashMode = button.dataset.fitDedup;
          drilldownShipName = button.dataset.fitShipName??"";
          state.view = "kills";
          state.showResults = true;
          limit = 50;
          render();
          syncURL();
          cursors = [null];
          loadResults();
        }
        return;
      }
      if (button.hasAttribute("data-advanced-clear-drilldown")) {
        state.hash = state.hashMode = "";
        drilldownShipName = "";
        state.view = "fits";
        render();
        syncURL();
        loadResults();
        return;
      }
      if (button.hasAttribute("data-kill-limit")) {
        cursors = [null];
        loadResults({ nextLimit: Number(button.dataset.killLimit) });
        return;
      }
      if (button.hasAttribute("data-kill-page")) {
        const nextPage = Number(button.dataset.killPage);
        if (nextPage > 1 && !cursors[nextPage - 1]) return;
        loadResults({ nextPage, nextLimit: limit });
        return;
      }
      if (button.hasAttribute("data-advanced-more-fits")) {
        loadResults({ append: true });
        return;
      }
    }
    const card = targetElement.closest<HTMLElement>("[data-fit-preview]");
    if (card && !targetElement.closest("a,button")) {
      const target = card.dataset.fitEditorUrl;if(!target)return;
      if (event.ctrlKey || event.metaKey)
        window.open(target, "_blank", "noopener");
      else location.assign(target);
    }
  }

  initializeDateTimePickers(root);
  const savedCallbacks:SavedSearchCallbacks={
    current: () =>
      savedSearchDocument(advancedFiltersForURL(state.filters) || "{}", {
        view: state.view,
        dedup: state.dedup,
        ...(state.hash && state.hashMode === "exact"
          ? { fitHash: state.hash }
          : state.hash
            ? { familyHash: state.hash }
            : {}),
      }),
    load: (document) => {
      const query = savedSearchQuery(document);
      state.filters = restoreAdvancedFilters(document.filters);
      state.view = document.view;
      state.dedup = document.dedup;
      state.hash = document.fitHash ?? document.familyHash ?? "";
      state.hashMode = document.fitHash
        ? "exact"
        : document.familyHash
          ? "family"
          : "";
      state.showResults = true;
      limit = 50;
      drilldownShipName = "";
      setDates(
        document.filters.timeRange?.from ?? "",
        document.filters.timeRange?.to ?? "",
      );
      render();
      syncURL(true, new URLSearchParams(query));
      cursors = [null];
      loadResults();
    },
  };
  function popstate(){
    const query = new URL(location.href).searchParams;
    try {
      state.filters = restoreAdvancedFilters(
        JSON.parse(query.get("q") || "{}"),
      );
      state.view = query.get("view") === "fits" ? "fits" : "kills";
      const dedup=query.get("dedup"),mode=query.get("dm");state.dedup=dedup==="none"||dedup==="exact"?dedup:"family";
      state.hash=query.get("dh")??"";state.hashMode=mode==="exact"||mode==="family"?mode:"";
      state.showResults = !!(
        query.get("q") ||
        state.hash ||
        state.view === "fits"
      );
      drilldownShipName = "";
      setDates(
        state.filters.timeRange?.from ?? "",
        state.filters.timeRange?.to ?? "",
      );
      render();
      cursors = [null];
      loadResults();
    } catch {
      notice("The search link contains invalid filters");
    }
  }

  function hide(){
    interrupted||=Boolean(resultTimer)||Boolean(resultRequest);
    io.cancelTimer(searchTimer);
    io.cancelTimer(resultTimer);
    searchRequest?.abort();
    resultRequest?.abort();
    searchTimer=resultTimer=undefined;searchRequest=resultRequest=undefined;
    spinner.hidden=true;hideSuggestions();results.removeAttribute('aria-busy');results.classList.remove('advanced-results-loading');
    const more=results.querySelector<HTMLButtonElement>('[data-advanced-more-fits]');if(more){more.disabled=false;more.textContent='Load more fits'}
  }
  function resume(){if(interrupted){interrupted=false;state.showResults=Boolean(hasAdvancedFilters(state.filters)||state.hash||state.view==='fits');void loadResults()}}

  render();
  readPager();
  return {click,input:inputChanged,change,keyboard,outside,popstate,hide,resume,current:savedCallbacks.current,load:savedCallbacks.load};
}

export class AdvancedSearchController extends RequestComponent {
  private ui:ReturnType<typeof createAdvancedSearch>|undefined
  connect(){if(!this.ui)this.ui=createAdvancedSearch(this.element,this);else this.ui.resume()}
  readFragment(path:string,signal:AbortSignal){return this.text(path,"Could not load search results. Please try again.",{signal})}
  readSuggestions(path:string,signal:AbortSignal){return this.request<unknown>(path,{signal},"Search unavailable")}
  schedule(run:()=>void,delay:number){return this.after(delay,run)}
  cancelTimer(timer:ReturnType<typeof setTimeout>|undefined){this.stopTimer(timer)}
  click(event:MouseEvent){return this.ui?.click(event)}
  input(event:Event){this.ui?.input(event)}
  change(event:Event){this.ui?.change(event)}
  keyboard(event:KeyboardEvent){this.ui?.keyboard(event)}
  outside(event:PointerEvent){this.ui?.outside(event)}
  popstate(){this.ui?.popstate()}
  show(event:PageTransitionEvent){if(event.persisted)this.ui?.resume()}
  savedCurrent(event:CustomEvent<SavedSearchCurrent>){try{event.detail.document=this.ui?.current()}catch(error){event.detail.error=error}}
  savedLoad(event:CustomEvent<SavedSearchLoad>){try{this.ui?.load(event.detail.document)}catch(error){event.detail.error=error}}
  hide(){this.ui?.hide();super.hide()}
}
registerController("advanced-search",AdvancedSearchController)
