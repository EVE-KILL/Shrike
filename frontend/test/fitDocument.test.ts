import { expect, test } from 'vitest';
import { savedFitToUI, fitToSaveBody, enforcePropmodExclusivity } from '../src/browser/fit-document';

test('saved fit slots, states, charges and grouped drone/cargo quantities round-trip to the editor document', () => {
  const items = Array.from({ length: 5 }, (_, group) => ({ slot_group: group + 1, ordinal: group + 1, type_id: 100 + group, state: group % 4, charge_type_id: group === 0 ? 178 : null, quantity: 1 }));
  items.push(...[
    { slot_group: 6, ordinal: 0, type_id: 2454, state: 2, charge_type_id: null, quantity: 3 },
    { slot_group: 6, ordinal: 1, type_id: 2454, state: 0, charge_type_id: null, quantity: 2 },
    { slot_group: 6, ordinal: 2, type_id: 2454, state: 2, charge_type_id: null, quantity: 1 },
    { slot_group: 7, ordinal: 0, type_id: 178, state: 0, charge_type_id: null, quantity: 500 },
    { slot_group: 7, ordinal: 1, type_id: 178, state: 0, charge_type_id: null, quantity: 20 },
  ]);
  const fit = savedFitToUI({ fit: { ship_type_id: 626, name: 'Fixture', description: null }, items });
  expect(fit.modules.map(m => m.slot)).toEqual(['High','Medium','Low','Rig','SubSystem'].map((type,i)=>({type,index:i+2})));
  expect(fit.modules.map(m=>m.state)).toEqual(['Passive','Online','Active','Overload','Passive']);
  expect(fit.modules[0].charge).toEqual({typeId:178});
  expect(fit.drones).toEqual([{typeId:2454,states:{Active:4,Passive:2}}]);
  expect(fit.cargo).toEqual([{typeId:178,quantity:520}]);
  const expectedItems = [...items.slice(0,5),
    {slot_group:6,ordinal:0,type_id:2454,state:2,charge_type_id:null,quantity:4},
    {slot_group:6,ordinal:1,type_id:2454,state:0,charge_type_id:null,quantity:2},
    {slot_group:7,ordinal:0,type_id:178,state:0,charge_type_id:null,quantity:520},
  ];
  for (const visibility of [0,1,2,3]) expect(fitToSaveBody(fit,visibility)).toEqual({
    ship_type_id:626,name:'Fixture',description:null,visibility,items:expectedItems,
  });
  expect(savedFitToUI({fit:{ship_type_id:626,name:fit.name,description:null},items:fitToSaveBody(fit,3).items})).toEqual(fit);
});

test('imported propmods keep only the first active or overloaded module',()=>{
  const fit = savedFitToUI({fit:{ship_type_id:626,name:'Fixture',description:' '},items:[
    {slot_group:2,ordinal:0,type_id:439,state:3},{slot_group:2,ordinal:1,type_id:440,state:2},{slot_group:2,ordinal:2,type_id:441,state:1},
  ]});
  const sde={types:new Map([439,440,441].map(id=>[id,{groupID:46}]))};
  enforcePropmodExclusivity(fit,sde);
  expect(fit.modules.map(m=>m.state)).toEqual(['Overload','Online','Online']);
  fit.modules[0].state='Preview';fit.cargo.push({typeId:178,quantity:0});
  expect(fitToSaveBody(fit,0)).toEqual({
    ship_type_id:626,name:'Fixture',description:null,visibility:0,items:[
      {slot_group:2,ordinal:0,type_id:439,state:2,charge_type_id:null,quantity:1},
      {slot_group:2,ordinal:1,type_id:440,state:1,charge_type_id:null,quantity:1},
      {slot_group:2,ordinal:2,type_id:441,state:1,charge_type_id:null,quantity:1},
    ],
  });
});
