// Required elements belong to the Go component's markup contract.
export function requiredElement<T extends Element=HTMLElement>(scope:ParentNode,selector:string):T {
  const element=scope.querySelector<T>(selector)
  if(!element)throw new Error(`Missing component element: ${selector}`)
  return element
}
export function eventElement(target:unknown):Element|null {
  return target instanceof Element?target:null
}
export function formInput(form:HTMLFormElement,name:string):HTMLInputElement {
  const element=form.elements.namedItem(name)
  if(!(element instanceof HTMLInputElement))throw new Error(`Missing form input: ${name}`)
  return element
}
export function cloneComponent(scope:ParentNode,selector:string):HTMLElement {
  const template=requiredElement<HTMLTemplateElement>(scope,selector)
  const element=template.content.firstElementChild?.cloneNode(true)
  if(!(element instanceof HTMLElement))throw new Error(`Missing component template: ${selector}`)
  return element
}
export function selectOption(text:string,value:string):HTMLOptionElement {
  const option=document.createElement('option');option.textContent=text;option.value=value;return option
}

type StatefulElement=HTMLElement&{moveBefore?:(node:Node,before:Node|null)=>void}
// Use the browser's state-preserving move only for a retained sibling. New
// template nodes still require normal adoption/insertion into this document.
export function placeElement(parent:HTMLElement,node:HTMLElement,before:Node|null):void {
  const destination=parent as StatefulElement
  if(node.parentNode===parent&&typeof destination.moveBefore==='function')destination.moveBefore(node,before)
  else parent.insertBefore(node,before)
}
