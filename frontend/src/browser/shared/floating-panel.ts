// Absolute panels must fit the viewport and any scrolling/clipping ancestor.
export function positionViewportPanel(panel:HTMLElement,anchor:DOMRectReadOnly){
  const root=panel.ownerDocument.documentElement,width=root.clientWidth||innerWidth,height=root.clientHeight||innerHeight
  panel.style.maxWidth=`${Math.max(0,width-16)}px`
  const bounds=panel.getBoundingClientRect()
  panel.style.left=`${Math.max(8,Math.min(anchor.left,width-bounds.width-8))}px`
  const below=height-anchor.bottom-8,above=anchor.top-8
  const desired=below>=bounds.height||below>=above?anchor.bottom+4:anchor.top-bounds.height-4
  panel.style.top=`${Math.max(8,Math.min(desired,height-bounds.height-8))}px`
}

export function placeFloatingPanel(anchor:HTMLElement,panel:HTMLElement){
  panel.style.maxHeight='';panel.style.overflowY=''
  let top=8,bottom=innerHeight-8
  // Native dialogs sit above the site header. Ordinary page popups must leave
  // room for a fixed/sticky navbar rather than hide their first controls behind it.
  if(!anchor.closest('dialog[open]')){
    const nav=anchor.ownerDocument.querySelector<HTMLElement>('nav[aria-label="Main navigation"]')
    if(nav&&/^(fixed|sticky)$/.test(getComputedStyle(nav).position)){
      const rect=nav.getBoundingClientRect()
      if(rect.top<=8&&rect.bottom>0)top=Math.max(top,rect.bottom+8)
    }
  }
  for(let parent=anchor.parentElement;parent;parent=parent.parentElement){
    if(/^(auto|scroll|hidden|clip)$/.test(getComputedStyle(parent).overflowY)){
      const rect=parent.getBoundingClientRect();top=Math.max(top,rect.top+4);bottom=Math.min(bottom,rect.bottom-4)
    }
  }
  const rect=anchor.getBoundingClientRect(),height=panel.getBoundingClientRect().height,above=rect.top-top,below=bottom-rect.bottom
  const upwards=below<height&&above>below,available=Math.max(0,(upwards?above:below)-4)
  panel.classList.toggle('bottom-full',upwards);panel.classList.toggle('mb-1',upwards);panel.classList.toggle('mt-1',!upwards)
  if(available<height){panel.style.maxHeight=available+'px';panel.style.overflowY='auto'}
}
