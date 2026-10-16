import { describe, expect, test } from "vitest";
import {
  defaultAdvancedFilters,
  restoreAdvancedFilters,
  serializeAdvancedFilters,
  advancedFiltersForURL,
  advancedURL,
  hasAdvancedFilters,
  advancedActiveChips,clearAdvancedChip,readAdvancedHits,readAdvancedState,readAdvancedOptions,
} from "../src/browser/advanced-search-state";
import {
  savedSearchDocument,
  savedSearchQuery,
} from "../src/utils/savedSearches";

describe("Go advanced search interoperability", () => {
  test('end-only dates survive shared links and saved documents with a visible applied chip',()=>{
    const filters=restoreAdvancedFilters({timeRange:{to:'2026-10-16T12:00'}}),range={from:'',to:'2026-10-16T12:00'}
    expect(filters.timeRange).toEqual(range)
    expect(JSON.parse(serializeAdvancedFilters(filters)).timeRange).toEqual(range)
    const document=savedSearchDocument(advancedFiltersForURL(filters)!)
    expect(document.filters.timeRange).toEqual(range)
    expect(restoreAdvancedFilters(document.filters).timeRange).toEqual(range)
    expect(advancedActiveChips(filters,readAdvancedOptions({}))).toEqual([{key:'timeRange',group:'Time',label:'Until 2026-10-16 12:00 EVE'}])
  })
  test('unknown filter input is narrowed without retaining prototype keys or malformed rows',()=>{
    const filters=restoreAdvancedFilters(JSON.parse('{"__proto__":{"polluted":true},"entities":{"victim":[null,{"id":-1},{"id":42,"type":"character","name":"Pilot","exclude":true}]},"items":[null,{"typeId":34}],"location":{"systemId":"bad","regionId":10000042,"securityTypes":["highsec",{}]},"timeRange":{"from":"2026-01-01"}}'))
    expect(filters.entities.victim).toEqual([{id:42,type:'character',name:'Pilot',exclude:true}]);expect(filters.items[0]).toMatchObject({typeId:34,slot:'any',side:'victim'});expect(filters.location.systemId).toBeNull();expect(filters.location.securityTypes).toEqual(['highsec']);expect(Object.hasOwn(filters,'__proto__')).toBe(false)
  })
  test('active chips keep their source ordering and clear only supported fields',()=>{
    const filters=restoreAdvancedFilters({label:'solo',location:{securityTypes:['highsec']},attackerCount:'5+',attackerType:'ganked',iskValue:'1b',iskMin:'500m',shipCategory:'frigates',techLevel:'t2'}),options=readAdvancedOptions({labels:[{id:'solo',name:'Solo'}]})
    expect(advancedActiveChips(filters,options).map(chip=>chip.key)).toEqual(['label','sec-highsec','attackerCount','attackerType','iskValue','iskRange','shipCategory','techLevel','timePreset']);clearAdvancedChip(filters,'__proto__');expect(Object.hasOwn(filters,'__proto__')).toBe(false);clearAdvancedChip(filters,'sec-highsec');clearAdvancedChip(filters,'iskRange');clearAdvancedChip(filters,'timePreset');expect(filters.location.securityTypes).toEqual([]);expect(filters.iskMin).toBe('');expect(filters.timeRange).toEqual({preset:null})
  })
  test('SSR state, options and mixed lookup responses validate their boundaries',()=>{
    expect(readAdvancedState({view:'other',dedup:'wrong',hashMode:'unknown',filters:null})).toMatchObject({view:'kills',dedup:'family',hashMode:'',filters:defaultAdvancedFilters()});expect(readAdvancedOptions({groups:[null,{key:'security',options:[{value:'highsec',label:'High Sec'},{value:3}]}]}).groups[0].options).toHaveLength(1);expect(readAdvancedHits({hits:[null,{id:'bad',name:'Bad',type:'character'},{id:'character_42',name:'Pilot',type:'character'}]})).toEqual([{id:'character_42',name:'Pilot',type:'character'}])
  })
  test("translates every filter family without leaking display-only state", () => {
    const f = restoreAdvancedFilters({
      label: "solo",
      entities: {
        victim: [
          {
            id: 587,
            type: "ship",
            name: "Rifter",
            role: "victim",
            exclude: true,
          },
        ],
        attacker: [
          { id: 123, type: "character", name: "Pilot", role: "attacker" },
        ],
        both: [{ id: 456, type: "corporation", name: "Fleet" }],
      },
      items: [
        {
          groupId: 55,
          name: "Projectile weapons",
          slot: "cargo",
          side: "either",
        },
      ],
      location: {
        systemId: 30000142,
        systemName: "Jita",
        securityTypes: ["highsec"],
      },
      timeRange: { from: "2026-08-18T12:00", to: "2026-08-18T16:00" },
      attackerCount: "5+",
      attackerType: "ganked",
      iskValue: "1b",
      iskMin: "500m",
      iskMax: "2b",
      shipCategory: "frigates",
      techLevel: "t2",
      sort: { field: "total_value", direction: "asc" },
    });
    const wire = JSON.parse(serializeAdvancedFilters(f));
    expect(wire.iskMin).toBe(500e6);
    expect(wire.iskMax).toBe(2e9);
    expect(wire.iskValue).toBeUndefined();
    expect(wire.location).toEqual({
      systemId: 30000142,
      securityTypes: ["highsec"],
    });
    expect(wire.entities.victim[0]).toEqual({
      id: 587,
      type: "ship",
      name: "Rifter",
      exclude: true,
    });
    expect(wire.entities.attacker[0].role).toBeUndefined();
    expect(wire.items[0].side).toBe("either");
    expect(hasAdvancedFilters(f)).toBe(true);
    const shared = JSON.parse(advancedFiltersForURL(f)!);
    expect(shared.location.systemName).toBe("Jita");
    expect(shared.entities.attacker[0].role).toBe("attacker");
  });
  test("restores legacy item searches and numeric account amounts", () => {
    const f = restoreAdvancedFilters({
      items: [{ typeId: 425, name: "Autocannon" }],
      iskMin: 500e6,
    });
    expect(f.items[0]).toEqual({
      typeId: 425,
      name: "Autocannon",
      slot: "any",
      side: "victim",
    });
    expect(f.iskMin).toBe("500000000");
    expect(JSON.parse(serializeAdvancedFilters(f)).timeRange).toEqual({
      preset: "30d",
    });
    expect(advancedFiltersForURL(defaultAdvancedFilters())).toBeNull();
    expect(hasAdvancedFilters(defaultAdvancedFilters())).toBe(false);
  });
  test("preserves fitting drilldown, grouping, and the canonical saved document", () => {
    const filters = restoreAdvancedFilters({
        entities: {
          victim: [{ id: 587, type: "ship", name: "Rifter", role: "victim" }],
        },
        iskMin: "500m",
      }),
      hash = "a".repeat(64);
    const document = savedSearchDocument(advancedFiltersForURL(filters)!, {
        view: "fits",
        dedup: "exact",
        familyHash: hash,
      }),
      restored = restoreAdvancedFilters(document.filters);
    expect(restored.entities.victim[0].role).toBeUndefined();
    expect(document.filters.iskMin).toBe(500e6);
    expect(document.filters.timeRange).toEqual({ preset: "30d" });
    const query = savedSearchQuery(document);
    expect(query.view).toBe("fits");
    expect(query.dedup).toBe("exact");
    expect(query.dm).toBe("family");
    expect(query.dh).toBe(hash);
    const url = advancedURL({
      filters: restored,
      view: "fits",
      dedup: "exact",
      hash,
      hashMode: "family",
    });
    expect(url.get("dh")).toBe(hash);
    expect(url.get("dedup")).toBe("exact");
  });
});
