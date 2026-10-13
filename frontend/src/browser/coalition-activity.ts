// Keep the already-rendered coalition links and filter state while totals load.
export async function loadCoalitionActivity(root: HTMLElement, request: typeof fetch = fetch, signal?:AbortSignal): Promise<void> {
  const status = root.querySelector<HTMLElement>('[data-coalition-activity-status]');
  const current = [...root.querySelectorAll<HTMLElement>('[data-coalition-activity]')];
  if (!status || !current.length || status.getAttribute('aria-busy') === 'true' || signal?.aborted) return;
  const message = status.querySelector<HTMLElement>('span');
  const retry = status.querySelector<HTMLButtonElement>('[data-coalition-activity-retry]');
  if (!message || !retry) return;
  status.hidden = false;
  status.setAttribute('aria-busy', 'true');
  message.textContent = 'Loading combat statistics…';
  retry.hidden = true;
  try {
    const response = await request('/_fragments/coalitions/activity', {headers: {Accept: 'text/html'},...(signal?{signal}:{})});
    if (!response.ok) throw new Error('Could not load combat statistics.');
    const html = await response.text();
    if (!root.isConnected || !status.isConnected || signal?.aborted) return;
    const template = root.ownerDocument.createElement('template');
    template.innerHTML = html;
    const received = new Map([...template.content.querySelectorAll<HTMLElement>('[data-coalition-activity]')]
      .map(node => [node.dataset.coalitionActivity, node]));
    if (current.some(node => !node.dataset.coalitionActivity || !received.has(node.dataset.coalitionActivity))) {
      throw new Error('Could not read combat statistics.');
    }
    for (const node of current) node.replaceWith(received.get(node.dataset.coalitionActivity)!);
    status.hidden = true;
  } catch (error) {
    if (!root.isConnected || !status.isConnected || signal?.aborted) return;
    message.textContent = error instanceof Error ? error.message : 'Could not load combat statistics.';
    retry.hidden = false;
  } finally {
    if (status.isConnected && !signal?.aborted) status.setAttribute('aria-busy', 'false');
  }
}
