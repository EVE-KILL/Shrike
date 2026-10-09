import {RequestComponent} from '../request-component'
import {BrowserRequestError} from '../../browser-request'

export abstract class AdminComponent extends RequestComponent {
  protected async html(path:string,fallback:string,options:RequestInit={}){
    try{return await this.text(path,fallback,options)}catch(error){
      if(error instanceof BrowserRequestError&&error.status===403)throw new BrowserRequestError('Admin access required.',403)
      throw error
    }
  }
}

export abstract class AdminSectionComponent extends AdminComponent {
  private parentSection:HTMLElement|null=null
  private sectionListener=(event:Event)=>{if(event instanceof CustomEvent){if(event.detail)this.resumed();else this.hide()}}
  connect(){this.parentSection=this.element.closest('[data-admin-section]');this.parentSection?.addEventListener('admin-visibility',this.sectionListener)}
  protected resumed(){}
  show(event:PageTransitionEvent){if(event.persisted&&!this.parentSection?.hidden)this.resumed()}
  disconnect(){this.parentSection?.removeEventListener('admin-visibility',this.sectionListener);this.parentSection=null;super.disconnect()}
}
