import {Application, type ControllerConstructor} from '@hotwired/stimulus'

// All page bundles share this module. Register only the controllers they use.
let application:Application|undefined
const registered=new Map<string,ControllerConstructor>()
export function registerController(name:string,controller:ControllerConstructor):Application {
  application??=Application.start()
  if(registered.get(name)!==controller){
    application.register(name,controller)
    registered.set(name,controller)
  }
  return application
}
