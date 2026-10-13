package testfixtures

import (
	"context"
	"fmt"
)

// Disposable accounts 4242/4343 are seeded by the browser fixture. They are
// in different corporations but share an alliance, exercising both scopes.
func SeedFittings(ctx context.Context, db SQLWriter) error {
	statements := []string{
		`UPDATE characters SET alliance_id=70121 WHERE character_id IN (4242,4343)`,
		`UPDATE corporations SET alliance_id=70121 WHERE corporation_id IN (10,11)`,
		`INSERT INTO inv_groups (group_id,name,category_id) VALUES (26,'Cruiser',6) ON CONFLICT DO NOTHING`,
		`INSERT INTO inv_types (type_id,name,group_id,category_id) VALUES (626,'Vexor',26,6) ON CONFLICT DO NOTHING`,
		`INSERT INTO user_fittings (fit_id,owner_character_id,ship_type_id,name,description,visibility) VALUES
		 ('PublicFit00001',4242,626,'Fixture Vexor','Disposable saved fitting with modules, charges, drones and cargo.',3),
		 ('PrivateFit0001',4242,626,'Private fixture','Private description',0),
		 ('CorpFit0000001',4242,626,'Corporation fixture','Corporation description',1),
		 ('AlliedFit00001',4242,626,'Alliance fixture','Alliance description',2)`,
		`INSERT INTO user_fitting_items (fit_id,slot_group,ordinal,type_id,state,charge_type_id,quantity)
		 SELECT fit_id,items.slot_group,items.ordinal,items.type_id,items.state,items.charge_type_id,items.quantity FROM user_fittings CROSS JOIN
		 (VALUES (1::smallint,0::smallint,484,2::smallint,178,1::smallint),(2,0,439,3,NULL,1),(3,1,2048,1,NULL,1),(6,0,2454,2,NULL,3),(6,1,2454,0,NULL,2),(7,0,178,0,NULL,500)) items(slot_group,ordinal,type_id,state,charge_type_id,quantity)`,
	}
	for i, statement := range statements {
		if _, err := db.Exec(ctx, statement); err != nil {
			return fmt.Errorf("seed fitting statement %d: %w", i+1, err)
		}
	}
	return nil
}
