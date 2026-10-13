import { initializeDateTimePickers } from "./date-time-picker.js";
import {
  initializeSearchPickers,
  updateSearchPicker,
  closeSearchPickers,
} from "./search-picker.js";
import {RequestComponent} from './controllers/request-component';
import {registerController} from './controllers/application';
import {iconHref} from './icon-hrefs';
import {requiredElement,cloneComponent,eventElement} from './shared/dom';
import {errorMessage,isAbortError,isRecord} from './shared/json';
import {isCampaignEntity,isCampaignLocation,type CampaignEntity} from './shared/campaign-model';
import type {AuthMeLegacyResponses,AuthLoginLegacyResponses,WalletAccountBalanceLegacyResponses,CampaignCreateData,CampaignCreateResponses} from '../../shared/api/types.gen';
type CreatorIdentity={characterId:number;characterName:string};
function readIdentity(raw:unknown):CreatorIdentity|null{return isRecord(raw)&&typeof raw.characterId==='number'&&typeof raw.characterName==='string'?{characterId:raw.characterId,characterName:raw.characterName}:null}
import {
  CAMPAIGN_PRIZE_DEFAULT_SPLITS,
  CAMPAIGN_SIDE_FILL,
  CAMPAIGN_NEUTRAL_FILL,
} from "./campaign-data.js";
import {
  newCampaignDraft,
  campaignVisibleSides,
  campaignPicked,
  campaignLocationIds,
  campaignHasName,
  campaignHasParticipants,
  campaignTimeError,
  campaignPrizeError,
  campaignAvailableBalance,
  campaignChecklist,
  campaignCanSubmit,
  campaignCreateDocument,
  CAMPAIGN_LOCATION_CAPS,
} from "./campaign-creator-state.js";

function createCampaignCreator(root:HTMLElement,io:CampaignCreatorController) {
 const q=<T extends Element=HTMLElement>(selector:string)=>requiredElement<T>(root,selector);
 const get=<T extends Element=HTMLElement>(scope:ParentNode,selector:string)=>requiredElement<T>(scope,selector);
  const state = newCampaignDraft(root.dataset.start??""),
    form = q<HTMLElement>("[data-campaign-form]"),
    gate = q<HTMLElement>("[data-campaign-login]");
  const grid = q<HTMLElement>("[data-campaign-side-grid]"),
    from = q<HTMLInputElement>('[name="campaign_start"]'),
    to = q<HTMLInputElement>('[name="campaign_end"]');
  let user:CreatorIdentity|null = readIdentity(JSON.parse(
      q<HTMLElement>("#campaign-creator-user").textContent??"null",
    )),
    wallet:WalletAccountBalanceLegacyResponses[200]|null = null,
    walletPending = false,
    submitting = false,
    authGeneration = 0,
    alive = true;
  let identityRequest:AbortController|undefined,submissionToken:object|undefined,fundingSignature="",uncertain=false;
  const clone = (id:string) => cloneComponent(root,"#"+id);
  const node = <K extends keyof HTMLElementTagNameMap>(tag:K, cls:string="", text?:string) => {
    const el = document.createElement(tag);
    if (cls) el.className = cls;
    if (text != null) el.textContent = text;
    return el;
  };
  function icon(name:string) {
    const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg"),
      use = document.createElementNS("http://www.w3.org/2000/svg", "use");
    svg.setAttribute("class", "icon");
    svg.setAttribute("viewBox", "0 0 24 24");
    svg.setAttribute("aria-hidden", "true");
    use.setAttribute("href", iconHref(name));
    svg.append(use);
    return svg;
  }
  const entityImage = (entity:CampaignEntity) =>
    `https://images.evetech.net/${entity.type === "character" ? "characters" : entity.type === "corporation" ? "corporations" : "alliances"}/${entity.id}/${entity.type === "character" ? "portrait" : "logo"}?size=64`;
  const section = (key:string) =>
    q<HTMLElement>('[data-campaign-section="' + key + '"]');
  function mark(key:string, step:number, done:boolean) {
    const badge = get<HTMLElement>(section(key),"[data-campaign-step]");
    badge.className =
      "w-6 h-6 rounded-md flex items-center justify-center flex-shrink-0 text-fine font-bold tabular-nums transition-colors " +
      (done ? "bg-green-500/15 text-isk" : "bg-blue-500/15 text-blue-400");
    badge.replaceChildren(done ? icon("check") : document.createTextNode(String(step)));
  }
  function error(selector:string, text:string|null) {
    const box = q(selector);
    box.hidden = !text;
    get<HTMLElement>(box,"[data-error-text]").textContent = text ?? "";
  }
  function updateDateDisabled() {
    for (const input of to
      .closest<HTMLElement>("[data-date-time-picker]")!
      .querySelectorAll<HTMLButtonElement|HTMLInputElement>("button,input"))
      input.disabled = state.ongoing;
  }
  function renderPreview() {
    const card = q<HTMLElement>("[data-campaign-preview] .entity-campaign-card",
    );
    get<HTMLElement>(card,"h3").textContent =
      state.name.trim() || "Untitled campaign";
    let desc = card.querySelector<HTMLElement>(".entity-campaign-description");
    if (!desc) {
      desc = node("p", "entity-campaign-description");
      get<HTMLElement>(card,"header").after(desc);
    }
    desc.textContent = state.description.trim();
    desc.hidden = !desc.textContent;
    const sides = campaignVisibleSides(state),
      area = state.mode === "area",
      colors = sides.map(
        (_, i) => CAMPAIGN_SIDE_FILL[i] ?? CAMPAIGN_NEUTRAL_FILL,
      );
    const rail = get<HTMLElement>(card,".entity-campaign-rail");
    rail.style.backgroundImage = "none";
    rail.style.backgroundColor = area
      ? "#f59e0b"
      : (colors[0] ?? CAMPAIGN_NEUTRAL_FILL);
    if (!area && colors.length > 1)
      rail.style.backgroundImage =
        "linear-gradient(to bottom," +
        colors
          .flatMap((color, i) => [
            `${color} ${(i / colors.length) * 100}%`,
            `${color} ${((i + 1) / colors.length) * 100}%`,
          ])
          .join(",") +
        ")";
    const date = get<HTMLElement>(card,".entity-campaign-date");
    date.replaceChildren(
      icon(area ? "map-pinned" : sides.length === 1 ? "crosshair" : "swords"),
      node(
        "span",
        "",
        `${formatDate(state.startTime)} — ${state.ongoing || !state.endTime ? "ongoing" : formatDate(state.endTime)}`,
      ),
    );
    let visibility = card.querySelector<HTMLElement>(".visibility");
    if (visibility) visibility.remove();
    if (state.visibility) {
      visibility = node(
        "span",
        "visibility " + (state.visibility === 2 ? "domain" : "private"),
      );
      visibility.title =
        state.visibility === 2
          ? "Killboard only — not listed on EVE-KILL"
          : "Private — owner and selected-domain members only";
      visibility.append(icon(state.visibility === 2 ? "shield" : "lock"));
      get<HTMLElement>(card,".entity-campaign-badges").prepend(visibility);
    }
    const body = get<HTMLElement>(card,".entity-campaign-body");
    body.replaceChildren();
    if (area) {
      const block = node("div", "entity-campaign-area"),
        content = node("div"),
        label = (["system", "constellation", "region"] as const)
          .map((type) => {
            const count = campaignLocationIds(state, type).length;
            return count ? `${count} ${type}${count === 1 ? "" : "s"}` : "";
          })
          .filter(Boolean)
          .join(" · ");
      content.append(
        node("strong", "", "All battlefield activity"),
        node("small", "", label || "Scope pending"),
      );
      block.append(icon("map-pinned"), content);
      body.append(block);
    } else {
      const rows = node("div", "entity-campaign-sides");
      for (const [i, side] of sides.entries()) {
        const row = node("div", "entity-campaign-side"),
          portraits = node("div", "entity-campaign-portraits");
        for (const entity of side.entities.slice(0, 4)) {
          const image = node("img");
          image.src = entityImage(entity);
          image.alt = "";
          image.width = image.height = 20;
          image.loading = "lazy";
          portraits.append(image);
        }
        if (side.entities.length > 4)
          portraits.append(node("span", "", `+${side.entities.length - 4}`));
        const name = node("strong", "", side.name.trim() || `Side ${i + 1}`);
        name.style.color = colors[i];
        const stats = node("small");
        stats.append(
          node("span", "kills", "0"),
          document.createTextNode(" kills · "),
          node("span", "isk", "0"),
        );
        row.append(portraits, name, stats);
        rows.append(row);
      }
      body.append(rows);
    }
    let author = card.querySelector<HTMLElement>("footer>small");
    if (!author) {
      author = node("small");
      get<HTMLElement>(card,"footer").append(author);
    }
    author.textContent = user?.characterName ? "by " + user.characterName : "";
    author.hidden = !author.textContent;
  }
  function formatDate(value:string) {
    if (!value) return "";
    const date = new Date(value + "Z");
    return Number.isNaN(date.getTime())
      ? ""
      : date.toLocaleDateString("en-US", {
          month: "short",
          day: "numeric",
          year: "numeric",
        });
  }
  function renderLists() {
    closeSearchPickers(grid);
    grid.replaceChildren(
      ...campaignVisibleSides(state).map((side, index) => {
        const block = clone("campaign-side-template");
        block.dataset.campaignSide = String(index);
        block.className =
          "relative rounded-lg bg-white/[0.02] border p-3 pl-4 overflow-hidden " +
          [
            "border-red-500/20",
            "border-blue-500/20",
            "border-emerald-500/20",
            "border-orange-500/20",
          ][index];
        get<HTMLElement>(block,"[data-side-rail]").style.backgroundColor =
          CAMPAIGN_SIDE_FILL[index];
        const name = get<HTMLInputElement>(block,"[data-campaign-side-name]");
        name.value = side.name;
        name.setAttribute("aria-label", `Side ${index + 1} name`);
        name.className =
          "flex-1 min-w-0 px-2.5 py-1.5 rounded bg-white/[0.04] border border-white/[0.06] text-sm font-medium focus:border-blue-500/40 focus:outline-none " +
          [
            "text-red-400",
            "text-blue-400",
            "text-emerald-400",
            "text-orange-400",
          ][index];
        const remove = get<HTMLElement>(block,"[data-campaign-remove-side]");
        remove.hidden = state.mode !== "conflict" || state.sides.length <= 1;
        remove.setAttribute("aria-label", `Remove side ${index + 1}`);
        const entities = get<HTMLElement>(block,"[data-side-entities]");
        entities.replaceChildren(
          ...side.entities.map((entity, entityIndex) => {
            const row = clone("campaign-participant-template"),
              img = get<HTMLImageElement>(row,"img");
            img.src = entityImage(entity);
            img.alt = entity.name;
            get<HTMLElement>(row,"[data-participant-name]").textContent =
              entity.name;
            get<HTMLElement>(row,"[data-participant-type]").textContent =
              entity.type;
            const button = get<HTMLButtonElement>(row,"button");
            button.dataset.campaignRemoveEntity = String(entityIndex);
            button.setAttribute("aria-label", "Remove " + entity.name);
            return row;
          }),
        );
        if (!side.entities.length)
          entities.append(
            node(
              "div",
              "text-center py-4 text-fine text-gray-600",
              "Add up to 15 alliances, corporations or characters",
            ),
          );
        get<HTMLElement>(block,"[data-side-count]").textContent =
          `${side.entities.length}/15`;
        return block;
      }),
    );
    grid.className =
      "grid grid-cols-1 gap-3 " +
      (campaignVisibleSides(state).length > 1 ? "md:grid-cols-2" : "");
    initializeSearchPickers(root);
    const locations = q<HTMLElement>("[data-campaign-locations]");
    locations.replaceChildren(
      ...state.locations.map((location, index) => {
        const tag = clone("campaign-location-template");
        get<SVGUseElement>(tag,"use")
          .setAttribute(
            "href",
            iconHref(location.type === "region"
                ? "map"
                : location.type === "constellation"
                  ? "compass"
                  : "map-pin"),
          );
        get<HTMLElement>(tag,"[data-tag-name]").textContent = location.name;
        const remove = get<HTMLButtonElement>(tag,"button");
        remove.dataset.campaignRemoveLocation = String(index);
        remove.setAttribute("aria-label", "Remove " + location.name);
        return tag;
      }),
    );
    locations.hidden = !state.locations.length;
    const allowed = q<HTMLElement>("[data-campaign-allowed]");
    allowed.replaceChildren(
      ...state.allowed.map((entity, index) => {
        const tag = clone("campaign-allowed-template");
        get<HTMLImageElement>(tag,"img").src = entityImage(entity);
        get<HTMLElement>(tag,"[data-tag-name]").textContent = entity.name;
        get<HTMLButtonElement>(tag,"button").dataset.campaignRemoveAllowed = String(index);
        get<HTMLButtonElement>(tag,"button")
          .setAttribute("aria-label", "Remove allowed " + entity.name);
        return tag;
      }),
    );
    allowed.hidden = !state.allowed.length;
  }
  function updatePickers() {
    for (const picker of root.querySelectorAll<HTMLElement>("[data-search-picker]")) {
      const kind = picker.dataset.pickerName,
        side =
          state.sides[
            Number(picker.closest<HTMLElement>("[data-campaign-side]")?.dataset.campaignSide)
          ];
      const count =
        kind === "allowed"
          ? state.allowed.length
          : kind === "side"
            ? side.entities.length
            : 0;
      updateSearchPicker(picker, {
        disabled: count >= (kind === "allowed" ? 10 : 15),
        placeholder:
          count >= (kind === "allowed" ? 10 : 15)
            ? `Max ${kind === "allowed" ? 10 : 15} entities`
            : undefined,
        isPicked: (type, id) =>
          kind === "side"
            ? campaignPicked(state, type, id)
            : (kind === "allowed" ? state.allowed : state.locations).some(
                (entity) => entity.type === type && entity.id === id,
              ),
      });
    }
  }
  function renderSplits() {
    const grid = q<HTMLElement>("[data-campaign-splits]");
    grid.replaceChildren(
      ...state.prizePercentages.map((value, index) => {
        const label = node("label", "block"),
          wrap = node("div", "relative"),
          input = node(
            "input",
            "w-full px-2 py-2 pr-5 rounded-lg bg-white/[0.04] border border-white/[0.08] text-sm text-white tabular-nums focus:border-amber-500/40 focus:outline-none",
          );
        input.type = "number";
        input.min = "1";
        input.max = "100";
        input.step = "1";
        input.value = String(value);
        input.dataset.campaignSplit = String(index);
        input.setAttribute("aria-label", `Payout place ${index + 1}`);
        label.append(
          node("span", "text-fine text-gray-600 mb-1 block", "#" + (index + 1)),
        );
        wrap.append(
          input,
          node("span", "absolute right-1.5 top-2 text-xs text-gray-600", "%"),
        );
        label.append(wrap);
        return label;
      }),
    );
  }
  function render() {
    form.hidden = !user;
    gate.hidden = !!user;
    const area = state.mode === "area",
      timeError = campaignTimeError(state),
      prizeError = campaignPrizeError(state, wallet, walletPending),
      hasParticipants = campaignHasParticipants(state);
    for (const choice of root.querySelectorAll<HTMLElement>("[data-campaign-choice]")) {
      const active =
        choice.dataset.campaignChoice === "mode"
          ? state.mode === choice.dataset.value
          : state.visibility === Number(choice.dataset.value);
      choice.className =
        "flex items-start gap-3 rounded-lg border p-3 text-left transition-colors cursor-pointer " +
        (active
          ? "border-blue-500/40 bg-blue-500/[0.08]"
          : "border-white/[0.08] bg-white/[0.02] hover:bg-white/[0.05]");
      choice.setAttribute("aria-pressed", String(active));
      get<HTMLElement>(choice,"[data-choice-icon]").className =
        "text-lg mt-0.5 flex-shrink-0 " +
        (active ? "text-blue-400" : "text-gray-500");
      get<HTMLElement>(choice,"[data-choice-label]").className =
        "text-sm font-medium " + (active ? "text-white" : "text-gray-300");
    }
    section("sides").hidden = area;
    get<HTMLElement>(section("sides"),"[data-campaign-section-title]",
    ).textContent = state.mode === "single" ? "Who to track" : "The sides";
    get<HTMLElement>(section("sides"),"[data-campaign-section-hint]").textContent =
      state.mode === "single"
        ? "Every kill and loss for these alliances, corporations or characters."
        : "Name each side and add the alliances, corporations or characters that fight for it.";
    const aside = get<HTMLElement>(section("sides"),"[data-campaign-section-aside]",
    );
    aside.hidden = state.mode !== "conflict" || state.sides.length >= 4;
    get<HTMLElement>(aside,"span:last-child").textContent =
      `Add side (${state.sides.length}/4)`;
    get<HTMLElement>(section("locations"),"[data-campaign-section-optional]",
    ).hidden = area;
    get<HTMLElement>(section("locations"),"[data-campaign-section-hint]").textContent = area
      ? "The regions, constellations or systems whose activity is tracked."
      : "Limit the campaign to specific space. Leave empty to count kills everywhere.";
    q<HTMLElement>("[data-campaign-location-hint]").hidden =
      !area || !!state.locations.length;
    mark("mode", 1, false);
    mark("identity", 2, campaignHasName(state));
    mark("sides", 3, hasParticipants);
    mark("locations", area ? 3 : 4, area && !!state.locations.length);
    mark("time", area ? 4 : 5, !timeError);
    mark("visibility", area ? 5 : 6, false);
    mark("prize", area ? 6 : 7, state.prizeEnabled && !prizeError);
    q<HTMLElement>("[data-campaign-name-error]").hidden =
      !state.name.trim() || campaignHasName(state);
    error("[data-campaign-time-error]", timeError);
    error("[data-campaign-prize-error]", prizeError);
    q<HTMLElement>("[data-campaign-ongoing-note]").hidden =
      !!timeError || !state.ongoing;
    q<HTMLElement>("[data-campaign-ongoing-description]").textContent = area
      ? "Ongoing area campaigns update hourly and remain active until manually archived."
      : "Ongoing campaigns update hourly and auto-archive after 30 days without a matching kill.";
    q<HTMLElement>("[data-campaign-public-limit]").hidden =
      state.visibility !== 0;
    q<HTMLElement>("[data-campaign-public-limit]").textContent =
      " Each character may have 3 active public ongoing campaigns.";
    q<HTMLInputElement>("[data-campaign-ongoing]").checked = state.ongoing;
    updateDateDisabled();
    q<HTMLElement>("[data-campaign-grants]").hidden = !(
      area && state.visibility === 2
    );
    q<HTMLElement>("[data-campaign-inferred-grants]").hidden =
      area || state.visibility !== 2;
    q<HTMLElement>("[data-campaign-no-prize]").hidden = state.prizeEnabled;
    q<HTMLElement>("[data-campaign-prize-fields]").hidden =
      !state.prizeEnabled;
    q<HTMLElement>("[data-campaign-metric-description]").textContent = [
      "Each qualifying killmail counts once per pilot.",
      "Rewards the pilots with the most ship losses.",
      "Ranks pilots by the value of their qualifying kills.",
      "Rewards the pilots who lose the most ship value.",
    ][state.prizeMetric];
    q<HTMLElement>("[data-campaign-loss-warning]").hidden = ![
      1, 3,
    ].includes(state.prizeMetric);
    const total = state.prizePercentages.reduce<number>(
        (sum, value) => sum + (Number(value) || 0),
        0,
      ),
      totalLabel = q<HTMLElement>("[data-campaign-split-total]");
    totalLabel.textContent = `Total: ${total}%`;
    totalLabel.className = total === 100 ? "text-green-400" : "text-red-400";
    const balance = wallet?.availableBalance ?? wallet?.balance,
      amount = Number(balance);
    q<HTMLElement>("[data-campaign-wallet-balance]").textContent =
      balance != null && Number.isFinite(amount)
        ? Math.round(amount).toLocaleString("en-US") + " ISK"
        : "— ISK";
    q<HTMLButtonElement>("[data-campaign-use-max]").disabled =
      campaignAvailableBalance(wallet) === null ||
      (campaignAvailableBalance(wallet)??0) <= 0;
    const checklist = q<HTMLElement>("[data-campaign-checklist]");
    checklist.replaceChildren(
      ...campaignChecklist(state, wallet, walletPending).map((item) => {
        const row = node("li", "flex items-start gap-2.5"),
          symbol = node(
            "span",
            "text-sm mt-px flex-shrink-0 " +
              (item.ok ? "text-isk" : "text-gray-600"),
          ),
          content = node("div", "min-w-0");
        symbol.append(icon(item.ok ? "circle-check" : "circle"));
        content.append(
          node(
            "div",
            "text-xs " + (item.ok ? "text-gray-400" : "text-gray-300"),
            item.label,
          ),
        );
        if (item.detail)
          content.append(
            node(
              "div",
              "text-fine mt-0.5 " +
                (item.ok ? "text-gray-600" : "text-amber-400/80"),
              item.detail,
            ),
          );
        row.append(symbol, content);
        return row;
      }),
    );
    const ready = campaignCanSubmit(state, wallet, walletPending),
      submit = q<HTMLButtonElement>("[data-campaign-create]");
    submit.disabled = !ready || submitting || uncertain;
    submit.className =
      "w-full mt-4 py-2.5 rounded-lg text-sm font-medium transition-colors " +
      (ready && !submitting && !uncertain
        ? "bg-blue-500 text-white hover:bg-blue-600 cursor-pointer"
        : "bg-white/[0.04] text-gray-600 cursor-not-allowed");
    q<HTMLElement>("[data-campaign-submit-label]").textContent = submitting
      ? "Creating campaign…"
      : "Create Campaign";
    q<HTMLElement>("[data-campaign-creating-icon]").hidden = !submitting;
    updatePickers();
    renderPreview();
  }
  function pick(event:CustomEvent<unknown>) {
    const picker=eventElement(event.target)?.closest<HTMLElement>('[data-search-picker]');if(!picker)return;
    const picked=event.detail,kind=picker.dataset.pickerName;
    if(kind==='side'||kind==='allowed'){if(!isCampaignEntity(picked))return;}else if(!isCampaignLocation(picked))return;
    if (kind === "side") {
      const index = Number(
          picker.closest<HTMLElement>("[data-campaign-side]")!.dataset.campaignSide,
        ),
        side = state.sides[index];
      if (
        side.entities.length >= 15 ||
        campaignPicked(state, picked.type, picked.id)
      )
        return;
      if(isCampaignEntity(picked))side.entities.push(picked);
    } else if (kind === "allowed") {
      if (
        state.allowed.length >= 10 ||
        state.allowed.some((e) => e.type === picked.type && e.id === picked.id)
      )
        return;
      if(isCampaignEntity(picked))state.allowed.push(picked);
    } else {
      if(!isCampaignLocation(picked))return;
      if (
        state.locations.some(
          (e) => e.type === picked.type && e.id === picked.id,
        ) ||
        state.locations.filter((e) => e.type === picked.type).length >=
          CAMPAIGN_LOCATION_CAPS[picked.type]
      )
        return;
      if(isCampaignLocation(picked))state.locations.push(picked);
    }
    renderLists();
    render();
  }
  function input(event:Event) {
    const target=event.target;if(!(target instanceof HTMLInputElement||target instanceof HTMLTextAreaElement))return;
    if (target.matches("[data-campaign-name]")) state.name = target.value;
    else if (target.matches("[data-campaign-description]"))
      state.description = target.value;
    else if (target.matches("[data-campaign-side-name]"))
      state.sides[
        Number(target.closest<HTMLElement>("[data-campaign-side]")!.dataset.campaignSide)
      ].name = target.value;
    else if (target.matches("[data-campaign-prize-funding]"))
      state.prizeInitialContribution = target.value;
    else if (target.matches("[data-campaign-split]"))
      state.prizePercentages[Number(target.dataset.campaignSplit)] =
        target.value;
    else return;
    render();
  }
  function change(event:Event) {
    const target=event.target;if(!(target instanceof HTMLInputElement||target instanceof HTMLSelectElement))return;
    if (target === from) state.startTime = target.value;
    else if (target === to) state.endTime = target.value;
    else if (target.matches("[data-campaign-ongoing]"))
      state.ongoing = target instanceof HTMLInputElement&&target.checked;
    else if (target.matches("[data-campaign-prize-enabled]")) {
      state.prizeEnabled = target instanceof HTMLInputElement&&target.checked;
      if (state.prizeEnabled) state.ongoing = false;
      else {
        state.prizeInitialContribution = "";
        q<HTMLInputElement>("[data-campaign-prize-funding]").value = "";
      }
    } else if (target.matches("[data-campaign-prize-metric]"))
      state.prizeMetric = Number(target.value);
    else if (target.matches("[data-campaign-prize-winners]")) {
      state.prizeWinnerCount = Number(target.value);
      state.prizePercentages = [
        ...CAMPAIGN_PRIZE_DEFAULT_SPLITS[state.prizeWinnerCount],
      ];
      renderSplits();
    } else return;
    render();
  }
  async function click(event:MouseEvent) {
    const button = eventElement(event.target)?.closest<HTMLButtonElement>("button");
    if (!button || !root.contains(button) || button.disabled) return;
    if (button.hasAttribute("data-campaign-choice")) {
      if (button.dataset.campaignChoice === "mode") {
        const mode=button.dataset.value;if(mode!=="single"&&mode!=="area"&&mode!=="conflict")return;state.mode=mode;
        renderLists();
      } else state.visibility = Number(button.dataset.value);
      render();
    } else if (button.hasAttribute("data-campaign-add-side")) {
      if (state.sides.length < 4)
        state.sides.push({
          name: `Side ${state.sides.length + 1}`,
          entities: [],
        });
      renderLists();
      render();
    } else if (button.hasAttribute("data-campaign-remove-side")) {
      if (state.sides.length > 1)
        state.sides.splice(
          Number(button.closest<HTMLElement>("[data-campaign-side]")!.dataset.campaignSide),
          1,
        );
      renderLists();
      render();
    } else if (button.hasAttribute("data-campaign-remove-entity")) {
      state.sides[
        Number(button.closest<HTMLElement>("[data-campaign-side]")!.dataset.campaignSide)
      ].entities.splice(Number(button.dataset.campaignRemoveEntity), 1);
      renderLists();
      render();
    } else if (button.hasAttribute("data-campaign-remove-location")) {
      state.locations.splice(Number(button.dataset.campaignRemoveLocation), 1);
      renderLists();
      render();
    } else if (button.hasAttribute("data-campaign-remove-allowed")) {
      state.allowed.splice(Number(button.dataset.campaignRemoveAllowed), 1);
      renderLists();
      render();
    } else if (button.hasAttribute("data-campaign-reset-splits")) {
      state.prizePercentages = [
        ...CAMPAIGN_PRIZE_DEFAULT_SPLITS[state.prizeWinnerCount],
      ];
      renderSplits();
      render();
    } else if (button.hasAttribute("data-campaign-use-max")) {
      const available = wallet?.availableBalance ?? wallet?.balance;
      if (available) {
        state.prizeInitialContribution = available;
        q<HTMLInputElement>("[data-campaign-prize-funding]").value = available;
        render();
      }
    } else if (button.hasAttribute("data-campaign-create")) {
      if (
        !user ||
        !campaignCanSubmit(state, wallet, walletPending) ||
        submitting
      )
        return;
      if(uncertain){error('[data-campaign-submit-error]','Creation was interrupted. Check your campaigns before retrying.');return;}
      const signature=JSON.stringify({...campaignCreateDocument(state),prizePool:{...campaignCreateDocument(state).prizePool,fundingRequestId:null}});
      if(Number(state.prizeInitialContribution.trim()||0)>0){if(signature!==fundingSignature)state.fundingRequestId=crypto.randomUUID();fundingSignature=signature;}
      const token={};submissionToken=token;
      submitting = true;
      error("[data-campaign-submit-error]", null);
      render();
      try {
        const result=await io.create(campaignCreateDocument(state));
        if(submissionToken!==token)return;
        if (!/^[a-zA-Z0-9]{14}$/.test(result?.campaign_id ?? "")){uncertain=true;throw new Error("The created campaign is unavailable. Check your campaigns before retrying.");}
        location.assign("/campaign/" + result.campaign_id);
      } catch (cause) {
        if(submissionToken!==token||isAbortError(cause))return;
        error(
          "[data-campaign-submit-error]",
          errorMessage(cause,"Failed to create campaign"),
        );
      } finally {
        if(submissionToken!==token)return;submissionToken=undefined;
        submitting = false;
        render();
      }
    } else if (button.hasAttribute("data-utility-login")) {
      try {
        const result = await io.login(
          "/auth/login?" +
            new URLSearchParams({
              redirect: location.pathname + location.search,
            }),
        );
        if (result?.url) location.assign(result.url);
      } catch (cause) {
        const box = q<HTMLElement>("[data-utility-login-error]");
        if(isAbortError(cause))return;box.textContent = errorMessage(cause);
        box.hidden = false;
      }
    }
  }
  async function identity() {
    if(!alive)return;identityRequest?.abort();const request=new AbortController();identityRequest=request;
    const epoch = ++authGeneration;
    try {
      const result = await io.identity(request.signal);
      if (epoch !== authGeneration || !alive) return;
      const next = readIdentity(result?.user),
        changed = user?.characterId !== next?.characterId;
      user = next;
      if (changed || !wallet) {
        wallet = null;
        if (user) {
          walletPending = true;
          render();
          try {
            const result = await io.wallet(request.signal);
            if (epoch === authGeneration && alive) wallet = result;
          } catch {
          } finally {
            if (epoch === authGeneration) walletPending = false;
          }
        }
      }
      if (alive) render();
    } catch {
      /* SSR session still renders; mutations revalidate it. */
    }
  }
  get<HTMLElement>(section("sides"),"[data-campaign-section-aside]")
    .append(clone("campaign-side-add"));
  const prizeAside = get<HTMLElement>(section("prize"),"[data-campaign-section-aside]",
  );
  prizeAside.append(clone("campaign-prize-toggle"));
  prizeAside.hidden = false;
  initializeDateTimePickers(root);
  renderLists();
  renderSplits();
  render();
  identity();
  function hide(){alive=false;authGeneration++;identityRequest?.abort();identityRequest=undefined;walletPending=false;uncertain||=submitting;submissionToken=undefined;submitting=false;closeSearchPickers(root)}
  function resume(){alive=true;void identity();if(uncertain)error('[data-campaign-submit-error]','Creation was interrupted. Check your campaigns before retrying.')}
  return {click,input,change,pick,identity,hide,resume};
}
export class CampaignCreatorController extends RequestComponent {
 private ui:ReturnType<typeof createCampaignCreator>|undefined
 connect(){if(!this.ui)this.ui=createCampaignCreator(this.element,this);else this.ui.resume()}
 identity(signal:AbortSignal){return this.request<AuthMeLegacyResponses[200]>('/auth/me',{signal})}
 wallet(signal:AbortSignal){return this.request<WalletAccountBalanceLegacyResponses[200]>('/api/user/wallet/balance',{signal})}
 create(body:CampaignCreateData['body']){return this.request<CampaignCreateResponses[200]>('/api/campaign/create',{method:'POST',body})}
 login(path:string){return this.request<AuthLoginLegacyResponses[200]>(path)}
 click(event:MouseEvent){return this.ui?.click(event)}
 input(event:Event){this.ui?.input(event)}
 change(event:Event){this.ui?.change(event)}
 pick(event:CustomEvent<unknown>){this.ui?.pick(event)}
 focus(){void this.ui?.identity()}
 show(event:PageTransitionEvent){if(event.persisted)this.ui?.resume()}
 hide(){this.ui?.hide();super.hide()}
}
registerController('campaign-creator',CampaignCreatorController)
