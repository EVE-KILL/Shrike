package mcpserver

func registerTools(registry *Registry) error {
	if err := registerSearchTool(registry); err != nil {
		return err
	}
	if err := registerEntityOverviewTool(registry); err != nil {
		return err
	}
	if err := registerEntityKillsTool(registry); err != nil {
		return err
	}
	if err := registerKillmailTool(registry); err != nil {
		return err
	}
	if err := registerKillmailFittingTool(registry); err != nil {
		return err
	}
	if err := registerSDEInfoTools(registry); err != nil {
		return err
	}
	if err := registerGlobalPulseTool(registry); err != nil {
		return err
	}
	if err := registerIntelTools(registry); err != nil {
		return err
	}
	if err := registerIntelAggregationTools(registry); err != nil {
		return err
	}
	if err := registerFunTools(registry); err != nil {
		return err
	}
	if err := registerEntityExtraTools(registry); err != nil {
		return err
	}
	if err := registerRouteTool(registry); err != nil {
		return err
	}
	if err := registerWarTool(registry); err != nil {
		return err
	}
	if err := registerBattleTools(registry); err != nil {
		return err
	}
	if err := registerKillsWithTool(registry); err != nil {
		return err
	}
	if err := registerShipsUsedTool(registry); err != nil {
		return err
	}
	if err := registerDoctrineTools(registry); err != nil {
		return err
	}
	if err := registerDogmaTools(registry); err != nil {
		return err
	}
	if err := registerForensicsTool(registry); err != nil {
		return err
	}
	if err := registerMeTools(registry); err != nil {
		return err
	}
	return nil
}
