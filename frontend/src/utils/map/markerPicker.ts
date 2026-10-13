import {positionViewportPanel} from '../../browser/shared/floating-panel'

export type MapMarkerChoice={node:HTMLButtonElement;name:string;count:string;select:()=>void}

// The map's labels keep their geographic position. When several labels cover
// a pointer hit, native popover dismissal and focus handling avoid picking the
// last-painted neighbour silently. Keyboard activation has an explicit target.
export function createMapMarkerPicker(host:HTMLElement){
  const panel=document.createElement('dialog')
  panel.className='go-map-marker-picker'
  panel.setAttribute('popover','auto')
  panel.setAttribute('aria-label','Choose a map location')
  host.append(panel)
  const native=typeof panel.showPopover==='function'
  const open=()=>native?panel.matches(':popover-open'):panel.open
  const hide=()=>{if(open()){if(native)panel.hidePopover();else panel.close()}panel.replaceChildren()}
  return {
    hide,
    activate(event:MouseEvent,chosen:MapMarkerChoice,choices:readonly MapMarkerChoice[]){
      hide()
      if(!host.isConnected||!chosen.node.isConnected)return
      const hits=event.detail===0?[chosen]:choices.filter(choice=>{
        if(choice.node.hidden||!choice.node.isConnected)return false
        const rect=choice.node.getBoundingClientRect()
        return rect.width>0&&rect.height>0&&event.clientX>=rect.left&&event.clientX<=rect.right&&event.clientY>=rect.top&&event.clientY<=rect.bottom
      })
      if(hits.length<2){chosen.select();return}
      const heading=document.createElement('strong');heading.textContent='Choose a location'
      const close=document.createElement('button');close.type='button';close.className='go-map-marker-picker-close';close.textContent='×';close.setAttribute('aria-label','Close location picker');close.addEventListener('click',hide)
      const header=document.createElement('header');header.append(heading,close);panel.append(header)
      for(const choice of hits){
        const button=document.createElement('button');button.type='button'
        const name=document.createElement('span');name.textContent=choice.name
        const count=document.createElement('small');count.textContent=choice.count
        button.append(name,count)
        button.addEventListener('click',()=>{if(!button.isConnected||!open()||!panel.isConnected||!choice.node.isConnected)return;hide();choice.select()})
        panel.append(button)
      }
      chosen.node.focus({preventScroll:true})
      if(native)panel.showPopover();else panel.showModal()
      positionViewportPanel(panel,new DOMRect(event.clientX,event.clientY,0,0))
      panel.querySelector<HTMLButtonElement>('header + button')?.focus({preventScroll:true})
    },
  }
}
