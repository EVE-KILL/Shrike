package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestStatusReferenceKeepsMarkupAndReplacesOnlyPageController(t *testing.T) {
	renderer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Fatal("source received credentials")
		}
		_, _ = w.Write([]byte(`<div data-controller="status" data-status-panels hidden></div><script src="/assets/current/status.js"></script>`))
	})
	handler := statusReferenceRenderer(referenceRendererMustNotMutate{}, renderer, map[string][]byte{"status.js": []byte("// archive")})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/status", nil)
	r.Header.Set("Cookie", "private")
	r.Header.Set("Authorization", "private")
	handler.ServeHTTP(w, r)
	if !strings.Contains(w.Body.String(), `data-status-panels hidden`) || strings.Contains(w.Body.String(), `data-controller="status"`) || !strings.Contains(w.Body.String(), `/__fixture/status-source/status.js`) {
		t.Fatal(w.Body.String())
	}
}

func TestStatusRelayDeliversSameSnapshotAndBlocksOnlyLocalReconnect(t *testing.T) {
	relay := newFixtureStatusRelay()
	server := httptest.NewServer(relay.wrap(http.NotFoundHandler()))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	post := func(body string, expected int) {
		t.Helper()
		r, err := http.Post(server.URL+"/__fixture/status-relay", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()
		if r.StatusCode != expected {
			t.Fatal(r.StatusCode)
		}
	}
	post(`{"snapshot":[]}`, http.StatusBadRequest)
	post(`{"snapshot":{"queues":{},"system":{"platform":"fixture"}}}`, http.StatusNoContent)
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/status"
	c, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	_, data, err := c.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Channel string `json:"channel"`
		Data    struct {
			System struct{ Platform string } `json:"system"`
		} `json:"data"`
	}
	if json.Unmarshal(data, &envelope) != nil || envelope.Channel != "status" || envelope.Data.System.Platform != "fixture" {
		t.Fatal(string(data))
	}
	// A reader must consume the close frame for Close's handshake to finish.
	read := make(chan error, 1)
	go func() { _, _, err := c.Read(ctx); read <- err }()
	post(`{"blocked":true,"disconnect":true}`, http.StatusNoContent)
	if err := <-read; websocket.CloseStatus(err) != websocket.StatusGoingAway {
		t.Fatal(err)
	}
	_, response, err := websocket.Dial(ctx, url, nil)
	if err == nil || response == nil || response.StatusCode != http.StatusServiceUnavailable {
		t.Fatal("local reconnect was not blocked", err)
	}
	post(`{"blocked":false}`, http.StatusNoContent)
	c2, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c2.CloseNow()
	if _, replay, err := c2.Read(ctx); err != nil || string(replay) != string(data) {
		t.Fatal("reconnect lost snapshot", string(replay), err)
	}
}
