import { initializeDateTimePickers } from "./date-time-picker.js";
import {
  initializeSearchPickers,
  updateSearchPicker,
  closeSearchPickers,
} from "./search-picker.js";
import { readBrowserSession,BrowserRequestError } from "./browser-request";
import {RequestComponent} from './controllers/request-component';
import {registerController} from './controllers/application';
import {requiredElement,cloneComponent,eventElement} from './shared/dom';
import {isAbortError,errorMessage} from './shared/json';
import {isBackdropClick} from './shared/dialog';
import {readCampaignEdit,isCampaignEntity,type CampaignEditView,type CampaignSide,type CampaignEntity} from './shared/campaign-model';
import type {CampaignUpdateData,AuthLoginLegacyResponses} from '../../shared/api/types.gen';
type Editor={name:string;description:string;endTime:string;ongoing:boolean;visibility:number;sides:CampaignSide[];allowed:CampaignEntity[];originalSides:string};

function createCampaignOwner(campaign:HTMLElement,io:CampaignOwnerController) {
 const q=<T extends Element=HTMLElement>(selector:string)=>requiredElement<T>(campaign,selector);
 const get=<T extends Element=HTMLElement>(scope:ParentNode,selector:string)=>requiredElement<T>(scope,selector);
  if (!campaign.querySelector("[data-campaign-owner-actions]")) return;
  const controls = q<HTMLElement>("[data-campaign-owner-actions]");
  const id = campaign.dataset.campaignId,
    slot = q<HTMLElement>("[data-campaign-edit-slot]"),
    deleteDialog = q<HTMLDialogElement>("[data-campaign-delete-dialog]");
  const actionError = q("[data-campaign-action-error]"),
    actionMessage = q("[data-campaign-action-message]");
  let editor:Editor|null = null,
    view:CampaignEditView|null = null,
    saving = false,
    opening:AbortController|null = null;
  let alive=true,interrupted=false,mutationToken:object|undefined,identityEpoch=0;
  const node = <K extends keyof HTMLElementTagNameMap>(tag:K, cls:string, text:string) => {
    const el = document.createElement(tag);
    if (cls) el.className = cls;
    if (text != null) el.textContent = text;
    return el;
  };
  const fail = (text:string, target = actionError) => {
    target.textContent = text;
    target.hidden = false;
    if (target === actionError) actionMessage.hidden = true;
  };
  let writeRequest:AbortController|undefined,needsReview=false;
  const clearStatus = () => {
    actionError.hidden = actionMessage.hidden = true;
  };
  const normalizedSides = () =>
    editor!.sides.map((side) => ({
      name: side.name.trim(),
      entities: side.entities.map((entity) => ({
        type: entity.type,
        id: entity.id,
      })),
    }));
  const sidesValid = () =>
    !!editor && !!view && (view.Mode === "area" ||
    (editor.sides.length > 0 &&
      editor.sides.length <= 4 &&
      editor.sides.every(
        (side) => side.entities.length > 0 && side.entities.length <= 15,
      )));
  const clone = (selector:string) => cloneComponent(slot,selector);
  const collect = () => {
    if (!editor) return;
    editor.name = get<HTMLInputElement>(slot,"[data-edit-name]").value;
    editor.description = get<HTMLTextAreaElement>(slot,"[data-edit-description]").value;
    editor.endTime = get<HTMLInputElement>(slot,'[name="campaign_edit_end"]').value;
    editor.ongoing = get<HTMLInputElement>(slot,"[data-edit-ongoing]").checked;
    editor.visibility = Number(
      get<HTMLSelectElement>(slot,"[data-edit-visibility]").value,
    );
    const draft=editor;draft.sides.forEach((side, index) => {
      side.name = get<HTMLInputElement>(slot,'[data-edit-side-name="' + index + '"]',
      ).value;
    });
  };
  const editError = (message:string|null, reauth = false) => {
    const box = get<HTMLElement>(slot,"[data-edit-error]");
    box.hidden = !message;
    get<HTMLElement>(box,"[data-error-text]").textContent = message ?? "";
    get<HTMLElement>(box,"[data-edit-reauth]").hidden = !reauth;
  };
  const cancel = () => {
    opening?.abort();writeRequest?.abort();writeRequest=undefined;mutationToken=undefined;saving=false;
    closeSearchPickers(slot);
    editor = null;
    view = null;
    slot.replaceChildren();
  };
  function entityTag(entity:CampaignEntity, index:number, kind:"allowed"|"side") {
    const tag = clone("[data-edit-entity-template]");
    tag.className =
      "inline-flex items-center gap-1.5 px-2 py-1 rounded border text-xs font-medium " +
      (kind === "allowed"
        ? "border-indigo-500/30 bg-indigo-500/10 text-indigo-300"
        : "border-white/[0.08] bg-white/[0.04] text-gray-300");
    get<HTMLElement>(tag,"[data-edit-entity-name]").textContent =
      entity.name || "#" + entity.id;
    const button = get<HTMLButtonElement>(tag,"button");
    button.dataset.editRemove = kind;
    button.dataset.editIndex = String(index);
    button.setAttribute(
      "aria-label",
      "Remove " + (entity.name || "#" + entity.id),
    );
    return tag;
  }
  function update() {
    if (!editor||!view) return;
    const picker = get<HTMLInputElement>(slot,'[name="campaign_edit_end"]')
       .closest<HTMLElement>("[data-date-time-picker]")!;
    for (const input of picker.querySelectorAll<HTMLButtonElement|HTMLInputElement>("button,input"))
      input.disabled = editor.ongoing || view!.Locked;
    const grants = slot.querySelector<HTMLElement>("[data-edit-grants]");
    if (grants) grants.hidden = editor.visibility !== 2;
    const add = slot.querySelector<HTMLElement>("[data-edit-add-side]");
    if (add) add.hidden = editor.sides.length >= 4 || view!.Locked;
    for (const picker of slot.querySelectorAll<HTMLElement>("[data-search-picker]")) {
      const allowed = picker.dataset.pickerName === "edit-allowed",
        side =
          editor.sides[
            Number(picker.closest<HTMLElement>("[data-edit-side]")?.dataset.editSide)
          ],
        entities = allowed ? editor.allowed : side.entities;
      updateSearchPicker(picker, {
        disabled: entities.length >= (allowed ? 10 : 15),
        placeholder:
          entities.length >= (allowed ? 10 : 15)
            ? allowed
              ? "Max 10 entities"
              : "Maximum 15 participants"
            : allowed
              ? undefined
              : "Add a participant to this side",
        isPicked: (type, id) =>
          entities.some((entity) => entity.type === type && entity.id === id),
      });
    }
    get<HTMLButtonElement>(slot,"[data-edit-save]").disabled = saving || needsReview || !sidesValid();
  }
  function renderLists() {
    if(!editor||!view)return;const draft=editor;
    const list = slot.querySelector<HTMLElement>("[data-edit-side-list]");
    if (list && !view!.Locked) {
      closeSearchPickers(list);
      list.replaceChildren(
        ...draft.sides.map((side, index) => {
          const block = clone("[data-edit-side-template]");
          block.dataset.editSide = String(index);
          const name = get<HTMLInputElement>(block,"[data-edit-side-name]");
          name.dataset.editSideName = String(index);
          name.value = side.name;
          name.setAttribute("aria-label", `Side ${index + 1} name`);
          get<HTMLElement>(block,"[data-edit-side-count]").textContent =
            `${side.entities.length}/15`;
          const remove = get<HTMLElement>(block,"[data-edit-remove-side]");
          remove.dataset.editRemoveSide = String(index);
          remove.hidden = draft.sides.length <= 1;
          remove.setAttribute("aria-label", `Remove side ${index + 1}`);
          const rows = get<HTMLElement>(block,"[data-edit-side-entities]");
          rows.replaceChildren(
            ...side.entities.map((entity, entityIndex) =>
              entityTag(entity, entityIndex, "side"),
            ),
          );
          if (!side.entities.length)
            rows.append(
              node(
                "span",
                "text-fine text-red-400",
                "This side needs at least one participant.",
              ),
            );
          return block;
        }),
      );
    }
    const allowed = slot.querySelector<HTMLElement>("[data-edit-allowed-list]");
    if (allowed) {
      allowed.replaceChildren(
        ...editor.allowed.map((entity, index) =>
          entityTag(entity, index, "allowed"),
        ),
      );
      get<HTMLElement>(slot,"[data-edit-no-grants]").hidden =
        !!editor.allowed.length;
    }
    initializeSearchPickers(slot);
    update();
  }
  async function open() {
    clearStatus();
    opening?.abort();
    const controller = new AbortController();
    opening = controller;
    try {
      const text = await io.readEditor(`/_fragments/campaign/${id}/edit`,controller.signal);
      const doc = new DOMParser().parseFromString(text,"text/html"),form=doc.querySelector<HTMLElement>("[data-campaign-edit-form]");
      if (!form) throw new Error("The campaign editor is unavailable");
      if (opening !== controller) return;
      const nextView=readCampaignEdit(JSON.parse(get<HTMLInputElement>(form,"[data-edit-state]").value));
      closeSearchPickers(slot);
      slot.replaceChildren(
        document.importNode(
          form.closest("[data-campaign-edit-boundary]") ?? form,
          true,
        ),
      );
      needsReview=false;view = nextView;
      editor = {
        name: view.Name,
        description: view.Description,
        endTime: view.EndTime,
        ongoing: view.Ongoing,
        visibility: view.Visibility,
        sides: view.Sides,
        allowed: view.Allowed,
        originalSides:"",
      };
      editor.originalSides = JSON.stringify(normalizedSides());
      initializeDateTimePickers(slot);
      initializeSearchPickers(slot);
      update();
      get<HTMLInputElement>(slot,"[data-edit-name]").focus();
    } catch (cause) {
      if (!isAbortError(cause)) {
        fail(errorMessage(cause));
        if (cause instanceof BrowserRequestError && cause.status === 401) {
          const button = node("button", "", "Sign in again");
          button.type = "button";
          button.dataset.editReauth = "";
          actionError.append(" ", button);
        }
      }
    } finally {
      if (opening === controller) opening = null;
    }
  }
  async function save() {
    if (!editor || saving || needsReview) return;
    collect();
    if (!sidesValid()) {
      editError("Every side must contain 1–15 participants.");
      return;
    }
    if(!editor.ongoing&&editor.endTime&&!Number.isFinite(new Date(editor.endTime+"Z").getTime())){editError("Enter a valid end time.");return}
    const submitted=JSON.stringify(editor);
    const body:CampaignUpdateData["body"] = {
      name: editor.name,
      description: editor.description || null,
      endTime:
        editor.ongoing || !editor.endTime
          ? null
          : new Date(editor.endTime + "Z").toISOString(),
      visibility: editor.visibility,
      allowedEntities: editor.allowed,
    };
    const sides = normalizedSides();
    if (JSON.stringify(sides) !== editor.originalSides) body.sides = sides;
    const token={};mutationToken=token;writeRequest=new AbortController();saving = true;
    editError(null);
    const button = get<HTMLButtonElement>(slot,"[data-edit-save]");
    button.textContent = "Saving…";
    update();
    try {
      await io.write(`/api/campaign/${id}/update`, {
        method: "POST",
        body: body,signal:writeRequest.signal,
      });
      if(mutationToken!==token)return;
      collect();if(editor&&JSON.stringify(editor)!==submitted){editor.originalSides=JSON.stringify(sides);actionMessage.textContent='Saved. You have newer unsaved changes.';actionMessage.hidden=false}else location.reload();
    } catch (cause) {
      if(mutationToken!==token||isAbortError(cause))return;
      editError(
        cause instanceof BrowserRequestError && cause.status === 401
          ? "This browser is no longer signed in. Sign in again to save your changes."
          : errorMessage(cause,"Failed to save"),
        cause instanceof BrowserRequestError && cause.status === 401,
      );
    } finally {
      if(mutationToken!==token)return;mutationToken=undefined;saving = false;
      if (button.isConnected) {
        button.textContent = "Save changes";
        update();
      }
    }
  }
  async function mutation(body:CampaignUpdateData["body"], errorText:string) {
    if (saving||needsReview) return;
    const token={};mutationToken=token;writeRequest=new AbortController();saving = true;
    clearStatus();
    try {
      await io.write(`/api/campaign/${id}/update`, {
        method: "POST",
        body: body,signal:writeRequest.signal,
      });
      if(mutationToken===token)location.reload();
    } catch (cause) {
      if(mutationToken!==token||isAbortError(cause))return;
      fail(errorMessage(cause,errorText));
    } finally {
      if(mutationToken!==token)return;mutationToken=undefined;saving = false;
    }
  }
  async function remove() {
    if(saving||needsReview)return;const token={};mutationToken=token;writeRequest=new AbortController();saving=true;
    const button = get<HTMLButtonElement>(deleteDialog,"[data-campaign-delete-confirm]");
    button.disabled = true;
    button.textContent = "Deleting…";
    try {
      await io.write(`/api/campaign/${id}`, { method: "DELETE",signal:writeRequest.signal });
      if(mutationToken===token)location.assign("/campaigns");
    } catch (cause) {
      if(mutationToken!==token||isAbortError(cause))return;
      const box = get<HTMLElement>(deleteDialog,"[data-campaign-delete-error]");
      box.textContent = errorMessage(cause,"Failed to delete campaign");
      box.hidden = false;
      button.disabled = false;
      button.textContent = "Delete campaign";
    }finally{if(mutationToken===token){mutationToken=undefined;saving=false}}
  }
  async function signIn() {
    const data = await io.login(
      "/auth/login?" +
        new URLSearchParams({ redirect: location.pathname + location.search }),
    );
    if (data?.url) location.assign(data.url);
  }
  function click(event:MouseEvent) {
    const target = eventElement(event.target);if(!target)return;
    if(isBackdropClick(deleteDialog,event)){deleteDialog.close();return}
    if (target.closest("[data-campaign-edit]")) {
      open();
      return;
    }
    if (target.closest("[data-campaign-archive]")) {
      mutation(
        { archived: Number(controls.dataset.campaignStatus) !== 2 },
        "Failed to update campaign archive status",
      );
      return;
    }
    if (target.closest("[data-campaign-resume]")) {
      mutation(
        { resumeProcessing: true },
        "Failed to resume campaign processing",
      );
      return;
    }
    if (target.closest("[data-campaign-delete]")) {
      deleteDialog.showModal();
      return;
    }
    if (target.closest("[data-campaign-delete-cancel]")) {
      deleteDialog.close();
      return;
    }
    if (target.closest("[data-campaign-delete-confirm]")) {
      remove();
      return;
    }
    if (target.closest("[data-edit-cancel]")) {
      cancel();
      return;
    }
    if (target.closest("[data-edit-reauth]")) {
      void signIn().catch((cause:unknown) => {if(!isAbortError(cause))fail(errorMessage(cause))});
      return;
    }
    if (!editor) return;
    if (target.closest("[data-edit-add-side]")) {
      collect();
      if (editor.sides.length < 4 && !view!.Locked)
        editor.sides.push({
          name: `Side ${editor.sides.length + 1}`,
          entities: [],
        });
      renderLists();
      return;
    }
    const removeSide = target.closest<HTMLElement>("[data-edit-remove-side]");
    if (removeSide) {
      collect();
      if (editor.sides.length > 1 && !view!.Locked)
        editor.sides.splice(Number(removeSide.dataset.editRemoveSide), 1);
      renderLists();
      return;
    }
    const removeEntity = target.closest<HTMLElement>("[data-edit-remove]");
    if (removeEntity) {
      collect();
      if (removeEntity.dataset.editRemove === "allowed")
        editor.allowed.splice(Number(removeEntity.dataset.editIndex), 1);
      else if (!view!.Locked)
        editor.sides[
          Number(removeEntity.closest<HTMLElement>("[data-edit-side]")!.dataset.editSide)
        ].entities.splice(Number(removeEntity.dataset.editIndex), 1);
      renderLists();
    }
  }
  function change(event:Event) {
    const target=eventElement(event.target);if(!target)return;
    if (
      editor &&
      target.matches("[data-edit-ongoing],[data-edit-visibility]")
    ) {
      collect();
      update();
    }
  }
  function pick(event:CustomEvent<unknown>) {
    const target=eventElement(event.target);if(!target||!isCampaignEntity(event.detail))return;
    if (!editor || !slot.contains(target)) return;
    collect();
    const picker = target.closest<HTMLElement>("[data-search-picker]");if(!picker)return;const
      picked = event.detail;
    if (picker.dataset.pickerName === "edit-allowed") {
      if (
        editor.allowed.length < 10 &&
        !editor.allowed.some(
          (e) => e.type === picked.type && e.id === picked.id,
        )
      )
        editor.allowed.push(picked);
    } else if (!view!.Locked) {
      const index = Number(picker.closest<HTMLElement>("[data-edit-side]")!.dataset.editSide);
      if (editor.sides[index].entities.length >= 15) return;
      for (const side of editor.sides)
        side.entities = side.entities.filter(
          (e) => e.type !== picked.type || e.id !== picked.id,
        );
      editor.sides[index].entities.push(picked);
    }
    renderLists();
  }
  function submit(event:SubmitEvent) {
    const target=eventElement(event.target);if(!target)return;
    if (target.matches("[data-campaign-edit-form]")) {
      event.preventDefault();
      save();
    }
  }
 function hide(){alive=false;++identityEpoch;interrupted||=saving;writeRequest?.abort();writeRequest=undefined;mutationToken=undefined;saving=false;opening?.abort();opening=null;deleteDialog.close();closeSearchPickers(slot);if(editor){const button=get<HTMLButtonElement>(slot,'[data-edit-save]');button.textContent='Save changes';update()}const button=get<HTMLButtonElement>(deleteDialog,'[data-campaign-delete-confirm]');button.disabled=false;button.textContent='Delete campaign'}
 function resume(){alive=true;void identity();if(interrupted){interrupted=false;needsReview=true;fail('The campaign may have changed while this page was away. Reload it before submitting further changes.');if(editor)get<HTMLButtonElement>(slot,'[data-edit-save]').disabled=true}}
  function identity(){const epoch=++identityEpoch;return readBrowserSession()
    .then((user) => {
      if (
        alive && epoch===identityEpoch && campaign.isConnected && user &&
        (user.isAdmin ||
          Number(user.characterId) === Number(controls.dataset.campaignOwnerId))
      ) {
        controls.hidden = false;
        const paused = campaign.querySelector<HTMLElement>("[data-campaign-paused-actions]");
        if (paused) paused.hidden = false;
      }
    })
    .catch(() => {});}
  void identity();
  return {click,change,pick,submit,hide,resume};
}

export class CampaignOwnerController extends RequestComponent {
 private ui:ReturnType<typeof createCampaignOwner>|undefined
 connect(){if(!this.ui)this.ui=createCampaignOwner(this.element,this);else this.ui.resume()}
 readEditor(path:string,signal:AbortSignal){return this.text(path,'Could not load campaign editor',{signal})}
 write(path:string,options:{method:string;body?:CampaignUpdateData['body'];signal?:AbortSignal}){return this.request<unknown>(path,options)}
 login(path:string){return this.request<AuthLoginLegacyResponses[200]>(path)}
 click(event:MouseEvent){this.ui?.click(event)}
 change(event:Event){this.ui?.change(event)}
 pick(event:CustomEvent<unknown>){this.ui?.pick(event)}
 submit(event:SubmitEvent){this.ui?.submit(event)}
 show(event:PageTransitionEvent){if(event.persisted)this.ui?.resume()}
 hide(){this.ui?.hide();super.hide()}
}
registerController('campaign-owner',CampaignOwnerController)
