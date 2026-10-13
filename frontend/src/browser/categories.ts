import {Controller} from '@hotwired/stimulus'
import {registerController} from './controllers/application'

export class CategoriesController extends Controller {
  static targets=['input','group','count','empty','query']
  declare readonly inputTarget:HTMLInputElement
  declare readonly groupTargets:HTMLElement[]
  declare readonly countTarget:HTMLElement
  declare readonly emptyTarget:HTMLElement
  declare readonly queryTarget:HTMLElement

  connect(){this.filter()}
  filter(){
    const needle=this.inputTarget.value.trim().toLowerCase()
    let visibleGroups=0
    for(const group of this.groupTargets){
      let visibleLabels=0
      for(const card of group.querySelectorAll<HTMLElement>('.category-card')){
        const match=(card.dataset.categorySearch||'').includes(needle)
        card.hidden=!match
        if(match)visibleLabels++
      }
      group.hidden=visibleLabels===0
      const count=group.querySelector<HTMLElement>('.category-group-count')
      if(count)count.textContent=`${visibleLabels} categories`
      if(visibleLabels)visibleGroups++
    }
    this.countTarget.textContent=String(visibleGroups)
    this.emptyTarget.hidden=visibleGroups!==0
    this.queryTarget.textContent=this.inputTarget.value
  }
}
registerController('categories',CategoriesController)
