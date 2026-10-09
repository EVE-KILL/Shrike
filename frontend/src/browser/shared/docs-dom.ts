export function docsElement<K extends keyof HTMLElementTagNameMap>(tag:K,className = '',value?:unknown):HTMLElementTagNameMap[K] {
  const node = document.createElement(tag)
  if(className) node.className = className
  if(value != null) node.textContent = String(value)
  return node
}
