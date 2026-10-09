package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type remoteGenerator struct{ origin string }
type generatorHTTPError struct {
	status  int
	message string
}

func (e generatorHTTPError) Error() string  { return e.message }
func (e generatorHTTPError) GetStatus() int { return e.status }
func (r remoteGenerator) BattleGeneratorPreview(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", r.origin+"/api/battle/generator/preview", bytes.NewReader(input))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != 200 {
		var body struct {
			Message string `json:"message"`
			Detail  string `json:"detail"`
			Error   string `json:"error"`
		}
		_ = json.Unmarshal(data, &body)
		message := body.Message
		if message == "" {
			message = body.Detail
		}
		if message == "" {
			message = body.Error
		}
		if message == "" {
			message = fmt.Sprintf("Fixture preview: HTTP %d", response.StatusCode)
		}
		return nil, generatorHTTPError{response.StatusCode, message}
	}
	return data, nil
}
