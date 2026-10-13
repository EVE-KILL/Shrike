import {Controller} from '@hotwired/stimulus'
import {registerController} from './controllers/application'

export class LegacyKillController extends Controller {
  static targets=['more','row']
  declare readonly moreTarget:HTMLButtonElement
  declare readonly rowTargets:HTMLElement[]
  private visible=10
  more(){
    this.visible+=25
    for(const row of this.rowTargets)row.hidden=Number(row.dataset.attackerIndex)>=this.visible
    const remaining=Math.max(0,this.rowTargets.length-this.visible)
    if(remaining)this.moreTarget.textContent=`Show more (${remaining} remaining)`
    else this.moreTarget.hidden=true
  }
  highlight(event:MouseEvent){
    const cell=event.target instanceof Element?event.target.closest<HTMLElement>('[data-cell]'):null
    const row=cell?.closest<HTMLElement>('[data-attacker-index]')
    if(row&&this.element.contains(row))row.dataset.hovered=cell!.dataset.cell
  }
  unhighlight(event:MouseEvent){
    const cell=event.target instanceof Element?event.target.closest<HTMLElement>('[data-cell]'):null
    if(!cell||(event.relatedTarget instanceof Node&&cell.contains(event.relatedTarget)))return
    const row=cell.closest<HTMLElement>('[data-attacker-index]')
    if(row&&this.element.contains(row))delete row.dataset.hovered
  }
}
registerController('legacy-kill',LegacyKillController)
