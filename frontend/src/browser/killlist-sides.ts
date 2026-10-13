export interface KilllistSide {characterIds?:number[];corporationIds?:number[];allianceIds?:number[]}
export interface SideVictim {characterId?:number|string;corporationId?:number|string;allianceId?:number|string}
export type VictimSideLookup=(victim:SideVictim)=>number|undefined

// Pilots take precedence over corporations and alliances; last assignment wins.
export function createVictimSideLookup(sides:KilllistSide[]=[]):VictimSideLookup {
  const characters=new Map<number,number>(),corporations=new Map<number,number>(),alliances=new Map<number,number>()
  sides.forEach((side,index)=>{
    for(const id of side.characterIds??[])characters.set(Number(id),index)
    for(const id of side.corporationIds??[])corporations.set(Number(id),index)
    for(const id of side.allianceIds??[])alliances.set(Number(id),index)
  })
  return victim=>characters.get(Number(victim.characterId))??corporations.get(Number(victim.corporationId))??alliances.get(Number(victim.allianceId))
}
export function colorKilllistSides(root:HTMLElement,lookup:VictimSideLookup,colors:string[]=[]):void {
  const fixed=['var(--color-error)','var(--color-brand-primary)','var(--color-brand-secondary)','#f97316']
  for(const row of root.querySelectorAll<HTMLElement>('.kill-row')){
    const index=lookup({characterId:row.dataset.victimCharacterId,corporationId:row.dataset.victimCorporationId,allianceId:row.dataset.victimAllianceId})
    if(index===undefined)continue
    const accent=colors[index],color=/^#[0-9a-f]{6}$/i.test(accent??'')?accent:fixed[index]
    if(!color)continue
    row.style.borderLeft=`2px solid color-mix(in srgb, ${color} 40%, transparent)`
    row.style.backgroundColor=`color-mix(in srgb, ${color} ${accent?4:3}%, transparent)`
  }
}
