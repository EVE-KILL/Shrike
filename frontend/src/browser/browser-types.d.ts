export interface PickedEntity {
  id:number
  type:string
  name:string
  role?:string
  [key:string]:unknown
}
