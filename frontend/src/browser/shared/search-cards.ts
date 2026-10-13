import {iconHref} from '../icon-hrefs'
import {searchTypes,searchEntityPath,searchEntityImage,type SearchEntity} from './search-model'
const largeImages = new Set(['character','corporation','alliance','faction','ship','system','region','constellation'])
export function searchNode<K extends keyof HTMLElementTagNameMap>(tag:K,className = '',text?:string|number):HTMLElementTagNameMap[K] {const node = document.createElement(tag);if(className) node.className = className;if(text !== undefined) node.textContent = String(text);return node}
export function searchIcon(name:string):SVGSVGElement {const svg = document.createElementNS('http://www.w3.org/2000/svg','svg');svg.setAttribute('class','icon');svg.setAttribute('aria-hidden','true');const use = document.createElementNS('http://www.w3.org/2000/svg','use');use.setAttribute('href',iconHref(name));svg.append(use);return svg}
function logo(kind:'corporation'|'alliance',id:number):HTMLAnchorElement {const link = searchNode('a','search-page-logo');link.href = `/${kind}/${id}`;link.setAttribute('aria-label','View '+kind);const image = searchNode('img');image.src = `https://images.evetech.net/${kind === 'alliance' ? 'alliances' : 'corporations'}/${id}/logo?size=64`;image.alt = '';image.loading = 'lazy';link.append(image);return link}
// Preserve the original search card DOM/classes; filtering reuses these nodes.
export function searchCard(hit:SearchEntity):HTMLElement {
  const article = searchNode('article',`search-page-card search-type-${hit.type}`);article.dataset.type = hit.type
  const link = searchNode('a','search-page-card-main');link.href = searchEntityPath(hit)
  const media = searchNode('div','search-page-media '+(largeImages.has(hit.type) ? 'large' : 'small')),url = searchEntityImage(hit)
  if(url) {const image = searchNode('img');image.src = url;image.alt = hit.name || '';image.loading = 'lazy';media.append(image)}else media.append(searchIcon(searchTypes[hit.type].icon))
  const shade = searchNode('div','search-page-shade'),badge = searchNode('span','search-page-badge',hit.type),info = searchNode('div','search-page-info');badge.prepend(searchIcon(searchTypes[hit.type].icon));info.append(searchNode('strong','',hit.name || hit.id))
  if(hit.ticker) info.append(searchNode('span','',`[${hit.ticker}]`))
  if(hit.type === 'character' && hit.corporation_name) info.append(searchNode('small','',hit.corporation_name+(hit.corporation_ticker ? ` [${hit.corporation_ticker}]` : '')))
  if((hit.type === 'character' || hit.type === 'corporation') && hit.alliance_name) info.append(searchNode('small','',hit.alliance_name+(hit.alliance_ticker ? ` [${hit.alliance_ticker}]` : '')))
  media.append(shade,badge,info);link.append(media);article.append(link)
  if(hit.type === 'character' || hit.type === 'corporation') {const logos = searchNode('div','search-page-logos');if(hit.type === 'character' && hit.corporation_id) logos.append(logo('corporation',hit.corporation_id));if(hit.alliance_id) logos.append(logo('alliance',hit.alliance_id));if(logos.childElementCount) article.append(logos)}
  return article
}
