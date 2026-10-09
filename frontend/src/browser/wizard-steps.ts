// The connected stepper is shared by multi-step Go tools.
export function updateWizardSteps(root:HTMLElement,current:number):void {
  for(const button of root.querySelectorAll<HTMLButtonElement>('[data-wizard-step]')){
    const step=Number(button.dataset.wizardStep),done=step<current
    button.disabled=step>current
    button.classList.toggle('cursor-pointer',step<=current);button.classList.toggle('hover:opacity-80',step<=current);button.classList.toggle('cursor-default',step>current)
    if(step===current)button.setAttribute('aria-current','step');else button.removeAttribute('aria-current')
    const circle=button.querySelector<HTMLElement>('[data-wizard-circle]')!,number=button.querySelector<HTMLElement>('[data-wizard-number]')!
    const check=button.querySelector<HTMLElement>('[data-wizard-check]')!,label=button.querySelector<HTMLElement>('[data-wizard-label]')!
    circle.className='w-7 h-7 rounded-full flex items-center justify-center text-fine font-bold tabular-nums flex-shrink-0 border transition-colors '+(done?'bg-green-500/15 text-isk border-green-500/30':step===current?'bg-blue-500/20 text-blue-400 border-blue-500/40':'bg-white/[0.03] text-gray-600 border-white/[0.08]')
    number.hidden=done;check.hidden=!done
    label.className='block text-xs font-medium truncate '+(step>current?'text-gray-600':'text-gray-200')
    const connector=button.parentElement?.querySelector('[data-wizard-connector]')
    if(connector){connector.classList.toggle('bg-blue-500/40',step+1<=current);connector.classList.toggle('bg-white/[0.08]',step+1>current)}
  }
}
