/** Shared thread and source CommentInput composer for every Go-rendered target. */
import { createCommentInput,type CommentInputHandle } from './comment-input.js';
import { readBrowserSession,type BrowserUser } from './browser-request.js';
import { escapeCommentText as escape, commentIcon, mountCommentDialog,type CommentModal } from './comment-controls.js';
import { subscribeCommentStream,mergeCommentEvent,type Comment } from './comment-stream.js';

import {RequestComponent} from './controllers/request-component';
import {registerController} from './controllers/application';
import {requiredElement,eventElement,placeElement} from './shared/dom';
import {errorMessage,isAbortError} from './shared/json';
import type {CommentsCreateData,CommentsCreateResponses,CommentDetailResponses,CommentsThreadResponses,CommentReportData,AuthLoginLegacyResponses} from '../../shared/api/types.gen';

const numeric = (value:unknown) => Number.isSafeInteger(Number(value)) && Number(value) > 0 ? Number(value) : 0;
const dateText = (value:string) => {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
  return `${date.getUTCDate()} ${months[date.getUTCMonth()]} ${date.getUTCFullYear()}, ${String(date.getUTCHours()).padStart(2, '0')}:${String(date.getUTCMinutes()).padStart(2, '0')}`;
};

function createThread(root:HTMLElement,io:CommentThreadController) {
  const q=<T extends Element=HTMLElement>(selector:string)=>requiredElement<T>(root,selector);
  const targetType = Number(root.dataset.targetType), targetId = Number(root.dataset.targetId), targetSlug = root.dataset.targetSlug || '';
  const status = q('[data-comments-status]'), list = q('[data-comments-list]'), composer = q('[data-comments-composer]');
  const items = new Map<number,Comment>(), editors = new Map<string,CommentInputHandle>(), collapsed = new Set<number>(), autoCollapsed = new Set<number>();
  let user:BrowserUser|null=null,nextCursor:string|null=null,replyingTo:number|null=null,editingId:number|null=null,loading=false,disposed=true,rootEditor:CommentInputHandle|null=null,reportModal:CommentModal|null=null,loadRequest:AbortController|undefined,hashScrolled=false,editRequest=0,generation=0,unsubscribe:(()=>void)|undefined;
  let loaded=false,everConnected=false;
  if (root.dataset.showHeader === 'true') root.insertAdjacentHTML('afterbegin', `<header class="comment-list-header"><h2>${commentIcon('message-square')} Comments <span data-comments-count hidden></span></h2><span data-comments-connection>Reconnecting…</span></header>`);
  const setStatus = (message:string|null, state = 'error') => { status.hidden = !message; status.textContent = message || ''; status.dataset.state = state; };
  const actionButton = (action:string, id:number, label:string, icon:string) => `<button type="button" data-comment-${action}="${id}" aria-label="${label}" title="${label}">${commentIcon(icon)}${action === 'reply' ? ' Reply' : ''}</button>`;

  async function save(body:string,parentId:number|null=null,editId:number|null=null,signal?:AbortSignal) {
    if(disposed)throw new DOMException("Aborted","AbortError");
    const payload:CommentsCreateData['body']|{body_md:string} = editId ? { body_md: body } : { target_type: targetType, target_id: targetId, ...(targetSlug ? { target_slug: targetSlug } : {}), parent_id: parentId, body_md: body };
    const data = await io.write<CommentsCreateResponses[200]>(editId ? `/api/comments/${editId}` : '/api/comments',editId ? 'PATCH' : 'POST',payload,signal);
    if (disposed) return;
    if (data.comment) items.set(numeric(data.comment.id), data.comment);
    if(editId&&editors.get(`edit-${editId}`)?.value.trim()===body)editingId=null;
    if(parentId&&editors.get(`reply-${parentId}`)?.value.trim()===body)replyingTo=null;
    renderThreads();
  }
  function renderComposer() {
    rootEditor?.destroy(); rootEditor = null;
    if (user) {
      const id = numeric(user.characterId), name = escape(user.characterName), corpID = numeric(user.corporationId);
      composer.innerHTML = `<div class="comment-identity"><a href="/character/${id}"><img src="https://images.evetech.net/characters/${id}/portrait?size=64" width="24" height="24" alt="${name}"></a><span>Posting as</span><a href="/character/${id}">${name}</a>${user.corporationName && corpID ? `<span class="comment-identity-corp">· <a href="/corporation/${corpID}">${escape(user.corporationName)}</a></span>` : ''}</div>`;
      rootEditor = createCommentInput({ placeholder: 'Share a thought… Markdown supported.', submitLabel: 'Post comment', onSubmit: (body,signal) => save(body,null,null,signal) });
      composer.append(rootEditor.element);
    } else composer.innerHTML = `<div class="comment-signin">${commentIcon('message-square-plus')}<p>Join the conversation</p><small>Sign in with EVE Online to leave a comment.</small><button type="button" data-comment-login>${commentIcon('log-in')} Sign in with EVE Online</button></div>`;
  }
  function renderThreads() {
    const children = new Map<number|null,Comment[]>();
    for (const item of items.values()) {
      const parent = item.parent_id == null ? null : numeric(item.parent_id);
      if (parent === null && item.deleted_at) continue; // Nuxt hides deleted roots; removed replies retain their descendants.
      if (!children.has(parent)) children.set(parent, []);
      children.get(parent)?.push(item);
      if (item.depth >= 4 && !autoCollapsed.has(item.id)) { autoCollapsed.add(item.id); collapsed.add(item.id); }
    }
    children.get(null)?.sort((a, b) => Number(b.id) - Number(a.id));
    for (const [key, siblings] of children) if (key !== null) siblings.sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime());
    const descendants = (id:number,visited=new Set<number>()):number => {
      if (visited.has(id)) return 0; visited.add(id);
      return (children.get(id) || []).reduce((count, child) => count + 1 + descendants(child.id, visited), 0);
    };
    const renderNode = (item:Comment,depth=0):string => {
      if (depth > 40) return '';
      const id = numeric(item.id), characterID = numeric(item.character_id), corporationID = numeric(item.corporation_id), allianceID = numeric(item.alliance_id);
      const replies = children.get(id) || [], isCollapsed = collapsed.has(id), hidden = item.moderation_status >= 2 || item.deleted_at != null;
      const isAuthor = user && numeric(user.characterId) === characterID;
      const canEdit = isAuthor && Date.now() - new Date(item.created_at).getTime() < 86400000;
      const canDelete = user && (isAuthor || user.isAdmin) && !hidden;
      const canReport = user && !isAuthor && !hidden;
      const author = escape(item.character_name || `Character ${characterID}`), corp = escape(item.corporation_name), alliance = escape(item.alliance_name), timestamp = dateText(item.created_at);
      const header = `<header><a href="/character/${characterID}"><img src="https://images.evetech.net/characters/${characterID}/portrait?size=64" alt="${author}" width="40" height="40" loading="lazy"></a><div><a class="comment-author" href="/character/${characterID}">${author}</a><small><a href="/corporation/${corporationID}">${corp}</a>${alliance && allianceID ? ` / <a href="/alliance/${allianceID}">${alliance}</a>` : ''}</small></div><time datetime="${escape(item.created_at)}" title="${escape(timestamp)} EVE">${escape(timestamp)}</time>${item.edited_at ? '<span class="comment-edited" title="Edited">(edited)</span>' : ''}${canReport ? actionButton('report', id, 'Report comment', 'flag') : ''}${canEdit ? actionButton('edit', id, 'Edit comment', 'pencil') : ''}${canDelete ? actionButton('delete', id, 'Delete comment', 'x') : ''}</header>`;
      const card = editingId === id ? `<div class="comment-inline-editor" data-comment-editor="edit-${id}"></div>` : `<section id="comment-${id}" class="comment-card${hidden ? ' comment-hidden' : ''}">${header}${hidden ? '<div class="comment-removed">[comment removed]</div>' : `<div class="comment-body">${item.body_html || escape(item.body_md)}</div>`}${user && !hidden ? `<footer>${actionButton('reply', id, 'Reply', 'reply')}</footer>` : ''}</section>`;
      const reply = replyingTo === id ? `<div class="comment-inline-editor" data-comment-editor="reply-${id}"></div>` : '';
      const count = descendants(id);
      const line = replies.length && !isCollapsed ? `<button type="button" class="thread-line" data-comment-collapse="${id}" aria-label="Collapse ${count} replies" title="Collapse ${count} ${count === 1 ? 'reply' : 'replies'}"></button>` : '';
      const nested = !replies.length ? '' : isCollapsed ? `<button type="button" class="thread-expand-btn" data-comment-collapse="${id}">${commentIcon('square-plus')} Show ${count} ${count === 1 ? 'reply' : 'replies'}</button>` : `<div class="comment-children">${replies.map(child => renderNode(child, depth + 1)).join('')}</div>`;
      return `<div class="comment-node${line ? ' has-line' : ''}">${line}${card}${reply}${nested}</div>`;
    };
    // Retain mounted inline editors and selection when live events rerender cards.
    const focused = document.activeElement, selected = focused instanceof HTMLTextAreaElement ? [focused.selectionStart, focused.selectionEnd] : null;
    const roots = children.get(null) || [];
    list.innerHTML = roots.map(item => renderNode(item)).join('') + (nextCursor ? `<button type="button" class="comment-more" data-comment-more${loading ? ' disabled' : ''}>${loading ? 'Loading…' : 'Load more comments'}</button>` : '');
    const mounted = new Set<string>();
    for (const slot of list.querySelectorAll<HTMLElement>('[data-comment-editor]')) {
      const key = slot.dataset.commentEditor??'', [kind, rawID] = key.split('-'), id = numeric(rawID); mounted.add(key);
      if (!editors.has(key)) editors.set(key, createCommentInput({
        initial: kind === 'edit' ? items.get(id)?.body_md || '' : '',
        placeholder: kind === 'reply' ? 'Write a reply…' : undefined,
        submitLabel: kind === 'edit' ? 'Save' : 'Reply',
        onSubmit: (body,signal) => save(body, kind === 'reply' ? id : null, kind === 'edit' ? id : null,signal),
        onCancel: () => { if (kind === 'edit') editingId = null; else replyingTo = null; renderThreads(); },
      }));
      const editor=editors.get(key);if(editor)placeElement(slot,editor.element,null);
    }
    for (const [key, editor] of editors) if (!mounted.has(key)) { editor.destroy(); editors.delete(key); }
    if (selected && focused instanceof HTMLTextAreaElement && focused.isConnected) { focused.focus({ preventScroll: true }); focused.setSelectionRange(selected[0]!,selected[1]!); }
    setStatus(roots.length ? null : 'No comments yet. Be the first.', 'empty');
    const countBadge = root.querySelector<HTMLElement>('[data-comments-count]'); if (countBadge) { countBadge.textContent = String(items.size); countBadge.hidden = !items.size; }
    if (!hashScrolled && location.hash.startsWith('#comment-')) {
      const anchor = document.getElementById(location.hash.slice(1));
      if (anchor) { hashScrolled = true; anchor.scrollIntoView({ behavior: 'smooth', block: 'center' }); }
    }
  }
  async function load(refresh=false) {
    if (loading || disposed) return;
    loading = true;
    const params = new URLSearchParams({ target_type: String(targetType), target_id: String(targetId), limit: '50' });
    if (targetSlug) params.set('target_slug', targetSlug);
    if (!refresh&&nextCursor) params.set('cursor', nextCursor);
    const request=loadRequest = new AbortController();
    try {
      const data = await io.read<CommentsThreadResponses[200]>(`/api/comments/thread?${params}`,request.signal);
      if (disposed||loadRequest!==request) return;
      if(refresh){
        const roots=data.roots??[],visible=new Set(roots.map(item=>item.id)),floor=roots.length?Math.min(...roots.map(item=>item.id)):0;
        // A full first page proves visibility for its ID range. A shorter page
        // proves the entire root set, including deletions missed while hidden.
        for(const item of items.values())if(item.parent_id===null&&!visible.has(item.id)&&(roots.length<50||item.id>=floor))item.deleted_at=new Date().toISOString();
      }
      for (const item of [...(data.roots || []), ...(data.replies || [])]) items.set(numeric(item.id), item);
      if(!refresh||!loaded)nextCursor=data.nextCursor?String(data.nextCursor):null;loaded=true;renderThreads();
    } catch (error) { if (!disposed && !isAbortError(error)) setStatus(errorMessage(error,'Failed to load comments')); }
    finally { if(loadRequest!==request)return;loadRequest=undefined;loading = false; const button = root.querySelector<HTMLButtonElement>('[data-comment-more]'); if (button) { button.disabled = false; button.textContent = 'Load more comments'; } }
  }
  function report(id:number){reportModal?.dispose();reportModal=mountCommentDialog('comment-report-template');reportModal.dialog.dataset.commentReportId=String(id);reportModal.dialog.dataset.controller=[reportModal.dialog.dataset.controller,'comment-report'].join(' ');reportModal.dialog.dataset.action=[reportModal.dialog.dataset.action,'submit->comment-report#submit'].join(' ');}
  async function click(event:MouseEvent){
    const button = eventElement(event.target)?.closest<HTMLButtonElement>('button'); if (disposed||!button||button.disabled||!root.contains(button)) return;
    if (button.hasAttribute('data-comment-more')) { button.disabled = true; button.textContent = 'Loading…'; load(); }
    if (button.dataset.commentCollapse) { const id = numeric(button.dataset.commentCollapse); if (collapsed.has(id)) collapsed.delete(id); else collapsed.add(id); renderThreads(); }
    if (button.dataset.commentReply) { replyingTo = numeric(button.dataset.commentReply); renderThreads(); editors.get(`reply-${replyingTo}`)?.textarea.focus(); }
    if (button.dataset.commentEdit) {
      const id = numeric(button.dataset.commentEdit), request = ++editRequest;
      button.disabled = true;
      try {
        // Production WS frames omit body_md. Load the current Markdown even
        // if this card arrived or was edited through another browser tab.
        const data = await io.read<CommentDetailResponses[200]>(`/api/comments/${id}`);
        if (disposed || request !== editRequest) return;
        if (data.comment) items.set(id, { ...items.get(id), ...data.comment });
        editingId = id; renderThreads(); editors.get(`edit-${id}`)?.textarea.focus();
      } catch (error) { if (!disposed&&!isAbortError(error)) setStatus(errorMessage(error)); }
      finally { button.disabled = false; }
    }
    if (button.dataset.commentReport) report(numeric(button.dataset.commentReport));
    if (button.dataset.commentDelete) {
      if (!confirm('Delete this comment?')) return;
      button.disabled = true;
      try { const id = numeric(button.dataset.commentDelete); await io.write(`/api/comments/${id}`,'DELETE'); const item = items.get(id); if (item) item.deleted_at = new Date().toISOString(); renderThreads(); }
      catch(error){button.disabled=false;if(!disposed&&!isAbortError(error))setStatus(errorMessage(error));}
    }
    if (button.hasAttribute('data-comment-login')) {
      button.disabled = true;
      try { const data = await io.read<AuthLoginLegacyResponses[200]>(`/auth/login?${new URLSearchParams({ redirect: location.pathname + location.search + location.hash })}`); const destination = new URL(data.url); if (destination.protocol !== 'https:') throw new Error('Invalid login response'); location.assign(destination.href); }
      catch(error){button.disabled=false;if(!disposed&&!isAbortError(error))setStatus(errorMessage(error,'Login is unavailable'));}
    }
  }
  function resume(){
    if(!disposed)return;disposed=false;const epoch=++generation;
    unsubscribe = subscribeCommentStream(event => {
    if (disposed || Number(event.target_type) !== targetType || Number(event.target_id) !== targetId || (event.domain_id ?? null) !== (root.dataset.domainId ? Number(root.dataset.domainId) : null) || (targetSlug && (event.target_slug ?? '') !== targetSlug)) return;
    const id = numeric(event.comment_id);
    if (event.event_type === 'deleted') { const item = items.get(id); if (item) item.deleted_at = new Date().toISOString(); }
    else if(event.comment){const merged=mergeCommentEvent(items.get(id),event);if(merged)items.set(id,merged);}
    renderThreads();
  }, connected => {
    const indicator = root.querySelector<HTMLElement>('[data-comments-connection]');
    if (indicator) { indicator.textContent = connected ? 'Live' : 'Reconnecting…'; indicator.classList.toggle('connected', connected); }
    if(connected){if(everConnected&&!disposed&&!loading)void load(true);everConnected=true;}
  });
  readBrowserSession().then(account => { if (disposed||epoch!==generation) return;const changed=user?.characterId!==account?.characterId;user=account;if(changed||!composer.childElementCount)renderComposer(); renderThreads(); }).catch(() => { if (!disposed&&epoch===generation&&!composer.childElementCount) renderComposer(); });
  load(true);
  }
  function hide(){disposed=true;++generation;++editRequest;loadRequest?.abort();loadRequest=undefined;loading=false;unsubscribe?.();unsubscribe=undefined;rootEditor?.pause();reportModal?.dispose();for(const editor of editors.values())editor.pause();}
  return {click,resume,hide};
}

export class CommentThreadController extends RequestComponent {
  private ui:ReturnType<typeof createThread>|undefined
  connect(){this.ui??=createThread(this.element,this);this.ui.resume()}
  read<T>(path:string,signal?:AbortSignal){return this.request<T>(path,{signal})}
  write<T=unknown>(path:string,method:string,body?:unknown,signal?:AbortSignal){return this.request<T>(path,{method,body,signal})}
  click(event:MouseEvent){return this.ui?.click(event)}
  show(event:PageTransitionEvent){if(event.persisted)this.ui?.resume()}
  hide(){this.ui?.hide();super.hide()}
}
registerController('comment-thread',CommentThreadController)
export class CommentReportController extends RequestComponent {
  private busy:object|undefined
  async submit(event:SubmitEvent){
    event.preventDefault();if(this.busy)return
    const reason=this.q<HTMLSelectElement>('[name="reason"]').value;if(reason!=='spam'&&reason!=='harassment'&&reason!=='nsfw'&&reason!=='offtopic'&&reason!=='other')return
    const token={};this.busy=token;const button=this.q<HTMLButtonElement>('[type="submit"]');button.disabled=true;button.textContent='Reporting…';this.error('[data-report-error]')
    try{const body:CommentReportData['body']={reason,message:this.q<HTMLTextAreaElement>('[name="message"]').value.trim()||null};await this.request(`/api/comments/${Number(this.element.dataset.commentReportId)}/report`,{method:'POST',body});if(this.busy===token&&this.element instanceof HTMLDialogElement)this.element.close()}
    catch(error){if(this.busy===token)this.error('[data-report-error]',error)}finally{if(this.busy===token){this.busy=undefined;button.disabled=false;button.textContent='Report'}}
  }
  hide(){this.busy=undefined;super.hide()}
}
registerController('comment-report',CommentReportController)
const threadActions='click->comment-thread#click pagehide@window->comment-thread#hide pageshow@window->comment-thread#show'
export function initializeCommentThreads(scope:ParentNode=document){for(const root of scope.querySelectorAll<HTMLElement>('[data-comment-thread]')){const names=root.dataset.controller?.split(/\s+/)??[];if(!names.includes('comment-thread'))root.dataset.controller=[...names,'comment-thread'].join(' ');if(!root.dataset.action?.includes('click->comment-thread#click'))root.dataset.action=[root.dataset.action,threadActions].filter(Boolean).join(' ')}}
