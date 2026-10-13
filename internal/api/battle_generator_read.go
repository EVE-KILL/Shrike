package api

import (
	"context"
	"encoding/json"
	"fmt"
)

// BattleGeneratorPreview retains the API's validation and authoritative
// calculation while letting HTML render without a loopback HTTP request.
func (r *HomepageReader) BattleGeneratorPreview(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("battle generator reader unavailable")
	}
	var draft conflictBattleGeneratorPreview
	if err := json.Unmarshal(input, &draft); err != nil {
		return nil, apiError(400, "Invalid preview request")
	}
	payload, err := loadBattleGeneratorPreview(ctx, Options{DB: r.db}, &draft)
	if err != nil {
		return nil, err
	}
	return json.Marshal(normalizeJSON(payload.Body))
}
